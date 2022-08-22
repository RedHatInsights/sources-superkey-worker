package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
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
)

func main() {
	l.InitLogger(conf)

	initMetrics()
	initHealthCheck()

	l.Log.Infof("Listening to Kafka at: %s:%d, topic: %v", conf.KafkaBrokerConfig.Hostname, conf.KafkaBrokerConfig.Port, superkeyTopic)
	l.Log.Infof("Talking to Sources API at: [%v] using PSK [%v]", fmt.Sprintf("%v://%v:%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort), conf.SourcesPSK)

	reader, err := kafka.GetReader(&kafka.Options{
		BrokerConfig: &conf.KafkaBrokerConfig,
		Topic:        superkeyTopic,
		GroupID:      &conf.KafkaGroupID,
		Logger:       l.Log.WithField("kafka", ""),
	})
	if err != nil {
		l.Log.Fatalf(`could not get Kafka reader: %s`, err)
	}

	go func() {
		l.Log.Info("SuperKey Worker started.")

		kafka.Consume(
			reader,
			func(msg kafka.Message) {
				l.Log.Infof("Started processing message %s", string(msg.Value))
				processSuperkeyRequest(msg)
				l.Log.Infof("Finished processing message %s", string(msg.Value))
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
		l.Log.WithFields(logrus.Fields{"kafka_message": msg}).Warn(`no "x-rh-identity" or "x-rh-sources-org-id" headers found, skipping...`)
		return
	}

	switch eventType {
	case "create_application":
		if DisableCreation == "true" {
			l.Log.Infof("Skipping create_application request: %v", msg.Value)
			return
		}

		l.Log.Info("Processing `create_application` request")
		req := &superkey.CreateRequest{}
		err := msg.ParseTo(req)
		if err != nil {
			l.Log.Warnf("Error parsing request: %v", err)
			return
		}
		req.IdentityHeader = identityHeader
		req.OrgIdHeader = orgIdHeader

		createResources(req)
		l.Log.Infof("Finished processing `create_application` request for tenant %v type %v", req.TenantID, req.ApplicationType)

	case "destroy_application":
		if DisableDeletion == "true" {
			l.Log.Infof("Skipping destroy_application request: %v", msg.Value)
			return
		}

		l.Log.Info("Processing `destroy_application` request")
		req := &superkey.DestroyRequest{}
		err := msg.ParseTo(req)
		if err != nil {
			l.Log.Warnf("Error parsing request: %v", err)
			return
		}

		destroyResources(req)
		l.Log.Infof("Finished processing `destroy_application` request for GUID %v", req.GUID)

	default:
		l.Log.Warn("Unknown event_type")
	}
}

func createResources(req *superkey.CreateRequest) {
	l.Log.Infof("Forging request: %v", req)
	newApp, err := provider.Forge(req)
	if err != nil {
		l.Log.Errorf("Error forging request %v, error: %v", req, err)
		l.Log.Errorf("Tearing down superkey request %v", req)

		errors := provider.TearDown(newApp)
		if len(errors) != 0 {
			for _, err := range errors {
				l.Log.Errorf("Error during teardown: %v", err)
			}
		}

		err := req.MarkSourceUnavailable(err, newApp)
		if err != nil {
			l.Log.Errorf("Error during PATCH unavailable to application/source: %v", err)
		}

		return
	}
	l.Log.Infof("Finished Forging request: %v", req)

	err = newApp.CreateInSourcesAPI()
	if err != nil {
		l.Log.Errorf("Failed to POST req to sources-api: %v, tearing down.", req)
		provider.TearDown(newApp)
	}
}

func destroyResources(req *superkey.DestroyRequest) {
	l.Log.Infof("Un-Forging request: %v", req)
	errors := provider.TearDown(superkey.ReconstructForgedApplication(req))
	if len(errors) != 0 {
		for _, err := range errors {
			l.Log.Errorf("Error during teardown: %v", err)
		}
	}
	l.Log.Infof("Finished Un-Forging request: %v", req)
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

func initHealthCheck() {
	go func() {
		healthFile := "/tmp/healthy"
		// which endpoints can we hit and get a 200 from without any auth, these are the main 2
		// we need anyway.
		svcs := []string{"https://iam.amazonaws.com", "https://s3.amazonaws.com"}

		// custom http client with timeout
		client := http.Client{Timeout: 5 * time.Second}

		for {
			for _, svc := range svcs {
				resp, err := client.Get(svc)

				if err == nil && resp.StatusCode == 200 {
					// Copying to the bitbucket in order to gc the memory.
					_, err := io.Copy(io.Discard, resp.Body)
					if err != nil {
						l.Log.Errorf("Error discarding response body: %v", err)
					}

					err = resp.Body.Close()
					if err != nil {
						l.Log.Errorf("Error closing response body: %v", err)
					}

					// check if file exists, creating it if not (first run, or recovery)
					_, err = os.Stat(healthFile)
					if err != nil {
						_, err = os.Create(healthFile)
						if err != nil {
							l.Log.Errorf("Failed to touch healthcheck file")
						}
					}

				} else {
					l.Log.Warnf("Error hitting %s, err %v, removing %s", svc, err, healthFile)

					_, err = os.Stat(healthFile)
					// this is a bit complicated - err will be nil _if the file is there_, so we want
					// to remove only if the err is nil.
					if err == nil {
						_ = os.Remove(healthFile)
					}

					break
				}
			}

			time.Sleep(10 * time.Second)
		}
	}()
}
