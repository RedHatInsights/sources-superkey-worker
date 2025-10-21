package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
	kafkago "github.com/segmentio/kafka-go"
)

// Health check configuration
const (
	healthCheckInterval = 30 * time.Second
	stuckConsumerWindow = 2 * time.Minute
	probesFilePath      = "/tmp/healthy"
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

// newHealthTracker creates a new health tracker instance
func newHealthTracker() *healthTracker {
	return &healthTracker{
		partitionOffsets: make(map[int32]int64),
	}
}

// ===== Message Tracking =====

func (h *healthTracker) start(reader *kafka.Reader, brokerAddr string, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reader = reader
	h.brokerAddr = brokerAddr
	h.topic = topic
	h.started = true
	h.lastMessageTime = time.Now()
	l.Log.Info("Consumer started")
}

func (h *healthTracker) recordMessage(partition int32, offset int64) {
	h.mu.Lock()
	_, exists := h.partitionOffsets[partition]
	h.lastMessageTime = time.Now()
	h.partitionOffsets[partition] = offset
	h.messagesProcessed++
	partitionCount := len(h.partitionOffsets)
	h.mu.Unlock()

	// Log when we discover a new partition
	if !exists {
		l.Log.Infof("Discovered new partition %d (total partitions assigned: %d)", partition, partitionCount)
	}
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

func (h *healthTracker) isConsumerHealthy(lag int64, lagErr error) bool {
	h.mu.RLock()
	started := h.started
	lastMsg := h.lastMessageTime
	h.mu.RUnlock()

	if !started {
		return false
	}

	// If we couldn't calculate lag, don't mark unhealthy on transient Kafka connectivity issues
	if lagErr != nil {
		l.Log.Debugf("Failed to calculate lag: %v", lagErr)
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

	// Log transitions
	if nowHealthy {
		if !wasHealthy {
			l.Log.Info("Sources API healthy")
		}
	} else {
		if wasHealthy {
			l.Log.Warn("Sources API unhealthy")
		}
	}
}

func (h *healthTracker) updateOverallHealth(lag int64, lagErr error) {
	consumerOK := h.isConsumerHealthy(lag, lagErr)

	h.mu.RLock()
	apiOK := h.apiHealthy
	h.mu.RUnlock()

	nowHealthy := consumerOK && apiOK

	h.mu.Lock()
	wasHealthy := h.healthy
	h.healthy = nowHealthy
	h.mu.Unlock()

	// Manage health file on transitions
	if nowHealthy != wasHealthy {
		if nowHealthy {
			if _, err := os.Create(probesFilePath); err != nil {
				l.Log.Errorf("Failed to create health file: %v", err)
			} else {
				l.Log.Info("Health file created")
			}
		} else {
			if err := os.Remove(probesFilePath); err != nil && !os.IsNotExist(err) {
				l.Log.Errorf("Failed to remove health file: %v", err)
			} else {
				l.Log.Warn("Health file removed")
			}
		}
	}
}

// ===== Health Monitor =====

func monitorConsumerHealth(h *healthTracker, stop chan struct{}) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	ctx := context.Background()

	l.Log.Infof("Health monitor started (interval=%v)", healthCheckInterval)
	h.checkAPI(ctx) // Initial API check

	checkCount := 0
	for {
		select {
		case <-ticker.C:
			checkCount++
			// Calculate real lag from Kafka using ReadOffsets() (once per iteration)
			lag, lagErr := h.calculateLag(ctx)

			// Check API and update overall health
			h.checkAPI(ctx)
			h.updateOverallHealth(lag, lagErr)

			h.mu.RLock()
			partitionCount := len(h.partitionOffsets)
			l.Log.Debugf("Health: overall=%v api=%v partitions=%d msgs=%d lag=%d idle=%v",
				h.healthy,
				h.apiHealthy,
				partitionCount,
				h.messagesProcessed,
				lag,
				time.Since(h.lastMessageTime).Round(time.Second))

			// Log partition info at INFO level every 10 health checks (5 minutes by default)
			if checkCount%10 == 0 {
				l.Log.Infof("Consumer health summary: partitions=%d msgs_processed=%d lag=%d",
					partitionCount,
					h.messagesProcessed,
					lag)
			}
			h.mu.RUnlock()

		case <-stop:
			l.Log.Info("Health monitor stopped")
			return
		}
	}
}
