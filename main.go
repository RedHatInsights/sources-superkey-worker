package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/provider"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
	"github.com/sirupsen/logrus"
)

const (
	superkeyRequestedTopic = "platform.sources.superkey-requests"
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
)

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

	// Store reader reference and config for health checks
	health.mu.Lock()
	health.reader = reader
	if len(conf.KafkaBrokerConfig) > 0 {
		health.brokerAddr = fmt.Sprintf("%s:%d", conf.KafkaBrokerConfig[0].Hostname, *conf.KafkaBrokerConfig[0].Port)
	}
	health.topic = superkeyTopic
	health.mu.Unlock()

	go func() {
		l.Log.Info("SuperKey Worker started.")

		health.start()

		kafka.Consume(
			reader,
			func(msg kafka.Message) {
				health.recordMessage(int32(msg.Partition), msg.Offset)

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
