package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/provider"
	"github.com/redhatinsights/sources-superkey-worker/sources"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
	"github.com/sirupsen/logrus"
)

const (
	// probesFilePath defines the path for Kubernetes' probes.
	probesFilePath         = "/tmp/healthy"
	superkeyRequestedTopic = "platform.sources.superkey-requests"
	// healthCheckInterval defines how often to check consumer health
	healthCheckInterval = 30 * time.Second
	// consumerStaleTimeout defines max time since last message before marking unhealthy
	consumerStaleTimeout = 5 * time.Minute
)

var (
	// DisableCreation disabled processing `create_application` sk requests
	DisableCreation = os.Getenv("DISABLE_RESOURCE_CREATION")
	// DisableDeletion disabled processing `destroy_application` sk requests
	DisableDeletion = os.Getenv("DISABLE_RESOURCE_DELETION")

	conf          = config.Get()
	superkeyTopic = conf.KafkaTopic(superkeyRequestedTopic)

	// Metrics
	successfulResourcesCreationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_superkey_successful_creation_requests",
		Help: "The number of successful resources creation requests",
	})
	unsuccessfulResourcesCreationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_superkey_unsuccessful_creation_requests",
		Help: "The number of unsuccessful resources creation requests",
	})
	successfulResourcesDeletionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_superkey_successful_deletion_requests",
		Help: "The number of successful resources deletion requests",
	})
	unsuccessfulResourcesDeletionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_superkey_unsuccessful_deletion_requests",
		Help: "The number of unsuccessful resources deletion requests",
	})
	consumerHealthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_consumer_healthy",
		Help: "Indicates if the Kafka consumer is healthy (1 = healthy, 0 = unhealthy)",
	})
	lastMessageTimestampGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_last_message_timestamp_seconds",
		Help: "Unix timestamp of the last processed message",
	})
	partitionsAssignedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_partitions_assigned",
		Help: "Number of partitions currently assigned to this consumer",
	})
	sourcesAPIHealthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_sources_api_healthy",
		Help: "Indicates if the Sources API is reachable and healthy (1 = healthy, 0 = unhealthy)",
	})
	lastSourcesAPICheckGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sources_superkey_last_sources_api_check_timestamp_seconds",
		Help: "Unix timestamp of the last Sources API health check",
	})
)

// consumerHealthState tracks the health state of the Kafka consumer and Sources API
type consumerHealthState struct {
	mu                  sync.RWMutex
	lastMessageTime     time.Time
	partitionOffsets    map[int32]int64
	messagesProcessed   uint64
	isHealthy           bool
	consumerStarted     bool
	sourcesAPIHealthy   bool
	lastSourcesAPICheck time.Time
}

var healthState = &consumerHealthState{
	partitionOffsets:  make(map[int32]int64),
	isHealthy:         false,
	consumerStarted:   false,
	sourcesAPIHealthy: false,
}

func main() {
	l.InitLogger(conf)

	initMetrics()

	// Start the health monitoring goroutine that tracks consumer health
	// and manages the Kubernetes probe health file based on consumer activity.
	stopHealthMonitor := make(chan struct{})
	defer close(stopHealthMonitor)
	go monitorConsumerHealth(stopHealthMonitor)

	var brokers strings.Builder
	for i, broker := range conf.KafkaBrokerConfig {
		brokers.WriteString(broker.Hostname + ":" + strconv.Itoa(*broker.Port))
		if i < len(conf.KafkaBrokerConfig)-1 {
			brokers.WriteString(",")
		}
	}

	l.Log.Infof("Listening to Kafka at: %s, topic: %v", brokers.String(), superkeyTopic)
	l.Log.Infof("Talking to Sources API at: [%v]", fmt.Sprintf("%v://%v:%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort))

	reader, err := kafka.GetReader(&kafka.Options{
		BrokerConfig: conf.KafkaBrokerConfig,
		Topic:        superkeyTopic,
		GroupID:      &conf.KafkaGroupID,
		Logger:       l.Log.WithField("kafka", ""),
	})
	if err != nil {
		l.Log.Fatalf(`could not get Kafka reader: %s`, err)
	}

	go func() {
		l.Log.Info("SuperKey Worker started.")

		// Mark consumer as started
		healthState.markConsumerStarted()

		kafka.Consume(
			reader,
			func(msg kafka.Message) {
				// Track message processing for health monitoring
				healthState.recordMessageProcessed(int32(msg.Partition), msg.Offset)
				
				processSuperkeyRequest(msg)
			},
		)
	}()

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	// wait for a signal from the OS, gracefully terminating the consumer
	// if/when that comes in
	s := <-interrupts

	l.Log.Infof("Received %v, exiting", s)
	kafka.CloseReader(reader, "superkey reader")
	os.Exit(0)
}

// processSuperkeyRequest - processes messages.
func processSuperkeyRequest(msg kafka.Message) {
	eventType := msg.GetHeader("event_type")
	identityHeader := msg.GetHeader("x-rh-identity")
	orgIdHeader := msg.GetHeader("x-rh-sources-org-id")

	if identityHeader == "" && orgIdHeader == "" {
		l.Log.WithFields(logrus.Fields{"kafka_message": string(msg.Value)}).Error(`Skipping Superkey request because no "x-rh-identity" or "x-rh-sources-org-id" headers were found`)

		return
	}

	l.Log.WithFields(logrus.Fields{"org_id": orgIdHeader}).Debugf(`Processing Kafka message: %s`, string(msg.Value))

	switch eventType {
	case "create_application":
		req := &superkey.CreateRequest{}
		err := msg.ParseTo(req)
		if err != nil {
			l.Log.WithFields(logrus.Fields{"org_id": orgIdHeader}).Errorf(`Error parsing "create_application" request "%s": %s`, string(msg.Value), err)
			return
		}
		req.IdentityHeader = identityHeader
		req.OrgIdHeader = orgIdHeader

		// Define the log context with the fields we want to log.
		ctx := l.WithTenantId(context.Background(), req.TenantID)
		ctx = l.WithSourceId(ctx, req.SourceID)
		ctx = l.WithApplicationId(ctx, req.ApplicationID)
		ctx = l.WithApplicationType(ctx, req.ApplicationType)

		if DisableCreation == "true" {
			l.LogWithContext(ctx).Info(`Skipping "create_application" request because the the resource creation was disabled by the env var`)
			l.LogWithContext(ctx).Debugf(`Skipped "create_application" Kafka message: %s`, string(msg.Value))
			return
		}

		l.LogWithContext(ctx).Info(`Processing "create_application" request`)

		createResources(ctx, req)

		l.LogWithContext(ctx).Info(`Finished processing "create_application"`)

	case "destroy_application":
		req := &superkey.DestroyRequest{}
		err := msg.ParseTo(req)
		if err != nil {
			l.Log.WithFields(logrus.Fields{"org_id": orgIdHeader}).Errorf(`Error parsing "destroy_application" request "%s": %s`, string(msg.Value), err)
			return
		}

		// Define the log context with the fields we want to log.
		ctx := l.WithTenantId(context.Background(), req.TenantID)

		if DisableDeletion == "true" {
			l.LogWithContext(ctx).Info(`Skipping "create_application"" request because the the resource creation was disabled by the env var`)
			l.LogWithContext(ctx).Debugf(`Skipping destroy_application request: %s`, string(msg.Value))
			return
		}

		l.LogWithContext(ctx).Info(`Processing "destroy_application" request`)

		destroyResources(ctx, req)

		l.LogWithContext(ctx).Info(`Finished processing "destroy_application" request`)

	default:
		l.Log.WithFields(logrus.Fields{"org_id": orgIdHeader}).Errorf(`Unknown event type "%s" received in the header, skipping request...`, eventType)
	}
}

func createResources(ctx context.Context, req *superkey.CreateRequest) {
	l.LogWithContext(ctx).Debugf("Forging request: %v", req)

	newApp, err := provider.Forge(ctx, req)
	if err != nil {
		l.LogWithContext(ctx).Errorf(`Tearing down Superkey request due to an error while forging the request \"%v\": %s`, req, err)

		errors := provider.TearDown(ctx, newApp)
		if len(errors) != 0 {
			for _, err := range errors {
				l.LogWithContext(ctx).Errorf(`Unable to tear down application: %s`, err)
			}
		}

		err := req.MarkSourceUnavailable(ctx, err, newApp)
		if err != nil {
			l.LogWithContext(ctx).Errorf(`Error while marking the source and application as "unavailable" in Sources: %s`, err)
		}

		unsuccessfulResourcesCreationCounter.Inc()
		return
	}

	l.LogWithContext(ctx).Debug("Finished forging request")

	err = newApp.CreateInSourcesAPI(ctx)
	if err != nil {
		l.LogWithContext(ctx).Errorf(`Error while creating or updating the resources in Sources: %s`, err)
		provider.TearDown(ctx, newApp)
		unsuccessfulResourcesCreationCounter.Inc()
		return
	}

	successfulResourcesCreationCounter.Inc()
}

func destroyResources(ctx context.Context, req *superkey.DestroyRequest) {
	l.LogWithContext(ctx).Debugf(`Unforging request "%v"`, req)

	errors := provider.TearDown(ctx, superkey.ReconstructForgedApplication(req))
	if len(errors) != 0 {
		for _, err := range errors {
			l.LogWithContext(ctx).Errorf(`Error during teardown: %s"`, err)
		}

		unsuccessfulResourcesDeletionCounter.Inc()
	} else {
		successfulResourcesDeletionCounter.Inc()
	}

	l.LogWithContext(ctx).Info("Finished destroying resources")
}

func initMetrics() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(fmt.Sprintf(":%d", conf.MetricsPort), nil)
		if err != nil {
			l.Log.Errorf("Metrics init error: %s", err)
		}
	}()
}

// markConsumerStarted marks that the consumer has started processing messages
func (h *consumerHealthState) markConsumerStarted() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.consumerStarted = true
	h.lastMessageTime = time.Now()
	l.Log.Info("Consumer started")
}

// recordMessageProcessed records that a message was processed from a specific partition
func (h *consumerHealthState) recordMessageProcessed(partition int32, offset int64) {
	now := time.Now()
	h.mu.Lock()
	h.lastMessageTime = now
	h.partitionOffsets[partition] = offset
	h.messagesProcessed++
	partitionCount := len(h.partitionOffsets)
	h.mu.Unlock()
	
	lastMessageTimestampGauge.Set(float64(now.Unix()))
	partitionsAssignedGauge.Set(float64(partitionCount))
}

// checkHealth evaluates the overall health based on Kafka consumer activity and Sources API availability
func (h *consumerHealthState) checkHealth() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if !h.consumerStarted {
		return false
	}
	
	if timeSince := time.Since(h.lastMessageTime); timeSince > consumerStaleTimeout {
		l.Log.Warnf("Consumer stale: %v since last message", timeSince)
		return false
	}
	
	if !h.sourcesAPIHealthy {
		l.Log.Warn("Sources API unhealthy")
		return false
	}
	
	return true
}

// checkSourcesAPIHealth performs a health check against the Sources API
func (h *consumerHealthState) checkSourcesAPIHealth(ctx context.Context) {
	err := sources.HealthCheck(ctx)
	now := time.Now()
	
	h.mu.Lock()
	h.lastSourcesAPICheck = now
	prevHealth := h.sourcesAPIHealthy
	h.sourcesAPIHealthy = (err == nil)
	h.mu.Unlock()
	
	lastSourcesAPICheckGauge.Set(float64(now.Unix()))
	if err == nil {
		sourcesAPIHealthGauge.Set(1)
		if !prevHealth {
			l.Log.Info("Sources API healthy")
		}
	} else {
		sourcesAPIHealthGauge.Set(0)
		if prevHealth {
			l.Log.Warnf("Sources API unhealthy: %v", err)
		}
	}
}

// updateHealthStatus updates the overall health status and manages the health file
func (h *consumerHealthState) updateHealthStatus() {
	isHealthy := h.checkHealth()
	
	h.mu.Lock()
	prevHealth := h.isHealthy
	h.isHealthy = isHealthy
	h.mu.Unlock()
	
	if isHealthy {
		consumerHealthGauge.Set(1)
	} else {
		consumerHealthGauge.Set(0)
	}
	
	// Manage health file on state transitions
	if isHealthy != prevHealth {
		if isHealthy {
			if _, err := os.Create(probesFilePath); err != nil {
				l.Log.Errorf("Failed to create health file: %s", err)
			} else {
				l.Log.Info("Health file created")
			}
		} else {
			if err := os.Remove(probesFilePath); err != nil && !os.IsNotExist(err) {
				l.Log.Errorf("Failed to remove health file: %s", err)
			} else {
				l.Log.Warn("Health file removed")
			}
		}
	}
}

// monitorConsumerHealth periodically checks consumer health and manages the health file
func monitorConsumerHealth(stop chan struct{}) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	ctx := context.Background()
	
	l.Log.Infof("Health monitor started (interval: %v, stale timeout: %v)", 
		healthCheckInterval, consumerStaleTimeout)
	
	healthState.checkSourcesAPIHealth(ctx) // Initial check
	
	for {
		select {
		case <-ticker.C:
			healthState.checkSourcesAPIHealth(ctx)
			healthState.updateHealthStatus()
			
			healthState.mu.RLock()
			l.Log.Debugf("Health: overall=%v, kafka=%v, api=%v, partitions=%d, msgs=%d, last=%v",
				healthState.isHealthy, 
				time.Since(healthState.lastMessageTime) < consumerStaleTimeout,
				healthState.sourcesAPIHealthy,
				len(healthState.partitionOffsets),
				healthState.messagesProcessed,
				time.Since(healthState.lastMessageTime).Round(time.Second))
			healthState.mu.RUnlock()
				
		case <-stop:
			l.Log.Info("Health monitor stopped")
			return
		}
	}
}
