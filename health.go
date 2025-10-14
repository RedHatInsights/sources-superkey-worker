package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	kafkago "github.com/segmentio/kafka-go"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// Health check configuration
const (
	healthCheckInterval = 30 * time.Second
	stuckConsumerWindow = 2 * time.Minute
	probesFilePath      = "/tmp/healthy"
)

// Prometheus metrics for monitoring health
var (
	metricConsumerHealth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_consumer_healthy",
		Help: "Overall health: 1=healthy, 0=unhealthy",
	})
	metricAPIHealth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_sources_api_healthy",
		Help: "Sources API health: 1=healthy, 0=unhealthy",
	})
	metricConsumerLag = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_consumer_lag",
		Help: "Number of messages behind",
	})
)

// Health state tracking
type healthTracker struct {
	mu                sync.RWMutex
	reader            *kafka.Reader
	brokerAddr        string // Store broker address for offset queries
	topic             string // Store topic name
	lastMessageTime   time.Time
	partitionOffsets  map[int32]int64
	messagesProcessed uint64
	healthy           bool
	started           bool
	apiHealthy        bool
}

var health = &healthTracker{
	partitionOffsets: make(map[int32]int64),
}

// ===== Message Tracking =====

func (h *healthTracker) start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.started = true
	h.lastMessageTime = time.Now()
	l.Log.Info("Consumer started")
}

func (h *healthTracker) recordMessage(partition int32, offset int64) {
	h.mu.Lock()
	h.lastMessageTime = time.Now()
	h.partitionOffsets[partition] = offset
	h.messagesProcessed++
	h.mu.Unlock()
}

// ===== Health Checks =====

// calculateLag queries Kafka for actual lag: (latest offset - committed offset)
func (h *healthTracker) calculateLag(ctx context.Context) (int64, error) {
	h.mu.RLock()
	broker := h.brokerAddr
	topic := h.topic
	partitions := make(map[int32]int64, len(h.partitionOffsets))
	for p, offset := range h.partitionOffsets {
		partitions[p] = offset
	}
	h.mu.RUnlock()

	if broker == "" || topic == "" || len(partitions) == 0 {
		return 0, nil
	}

	var totalLag int64
	for partition, committed := range partitions {
		lag, err := h.getPartitionLag(ctx, broker, topic, partition, committed)
		if err != nil {
			return 0, err
		}
		totalLag += lag
	}
	
	return totalLag, nil
}

func (h *healthTracker) getPartitionLag(ctx context.Context, broker, topic string, partition int32, committed int64) (int64, error) {
	conn, err := kafkago.DialLeader(ctx, "tcp", broker, topic, int(partition))
	if err != nil {
		return 0, fmt.Errorf("dial partition %d: %w", partition, err)
	}
	defer conn.Close()
	
	_, latest, err := conn.ReadOffsets()
	if err != nil {
		return 0, fmt.Errorf("read offsets partition %d: %w", partition, err)
	}
	
	// Lag = how many messages behind we are
	lag := latest - committed - 1
	if lag < 0 {
		lag = 0
	}
	
	return lag, nil
}

func (h *healthTracker) isConsumerHealthy() bool {
	h.mu.RLock()
	started := h.started
	lastMsg := h.lastMessageTime
	h.mu.RUnlock()

	if !started {
		return false
	}

	// Check if stuck: query Kafka for real lag using ReadOffsets()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	lag, err := h.calculateLag(ctx)
	if err != nil {
		l.Log.Debugf("Failed to calculate lag: %v", err)
		// Don't mark unhealthy on transient Kafka connectivity issues
		return true
	}
	
	idleTime := time.Since(lastMsg)
	
	// Only unhealthy if messages are available but we're not processing them
	if lag > 0 && idleTime > stuckConsumerWindow {
		l.Log.Warnf("Consumer stuck: lag=%d (messages waiting), idle=%v", lag, idleTime.Round(time.Second))
		return false
	}

	return true
}

func (h *healthTracker) checkAPI(ctx context.Context) {
	wasHealthy := h.apiHealthy
	nowHealthy := (sources.HealthCheck(ctx) == nil)

	h.mu.Lock()
	h.apiHealthy = nowHealthy
	h.mu.Unlock()

	// Update metric and log transitions
	if nowHealthy {
		metricAPIHealth.Set(1)
		if !wasHealthy {
			l.Log.Info("Sources API healthy")
		}
	} else {
		metricAPIHealth.Set(0)
		if wasHealthy {
			l.Log.Warn("Sources API unhealthy")
		}
	}
}

func (h *healthTracker) updateOverallHealth() {
	consumerOK := h.isConsumerHealthy()
	
	h.mu.RLock()
	apiOK := h.apiHealthy
	h.mu.RUnlock()
	
	nowHealthy := consumerOK && apiOK
	
	h.mu.Lock()
	wasHealthy := h.healthy
	h.healthy = nowHealthy
	h.mu.Unlock()

	// Update metric
	if nowHealthy {
		metricConsumerHealth.Set(1)
	} else {
		metricConsumerHealth.Set(0)
	}

	// Manage health file on transitions
	if nowHealthy != wasHealthy {
		if nowHealthy {
			os.Create(probesFilePath)
			l.Log.Info("Health file created")
		} else {
			os.Remove(probesFilePath)
			l.Log.Warn("Health file removed")
		}
	}
}

// ===== Health Monitor =====

func monitorConsumerHealth(stop chan struct{}) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	ctx := context.Background()

	l.Log.Infof("Health monitor started (interval=%v)", healthCheckInterval)
	health.checkAPI(ctx) // Initial API check

	for {
		select {
		case <-ticker.C:
			// Check API and update overall health
			health.checkAPI(ctx)
			health.updateOverallHealth()

			// Calculate real lag from Kafka using ReadOffsets()
			lag, err := health.calculateLag(ctx)
			if err == nil {
				metricConsumerLag.Set(float64(lag))
			}
			
			health.mu.RLock()
			l.Log.Debugf("Health: overall=%v api=%v partitions=%d msgs=%d lag=%d idle=%v",
				health.healthy,
				health.apiHealthy,
				len(health.partitionOffsets),
				health.messagesProcessed,
				lag,
				time.Since(health.lastMessageTime).Round(time.Second))
			health.mu.RUnlock()

		case <-stop:
			l.Log.Info("Health monitor stopped")
			return
		}
	}
}

