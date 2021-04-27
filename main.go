package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/messaging"
	"github.com/redhatinsights/sources-superkey-worker/provider"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
	"github.com/segmentio/kafka-go"
)

var (
	// SuperKeyRequestQueue - the queue to listen on for superkey requests
	SuperKeyRequestQueue = "platform.sources.superkey-requests"

	// DisableCreation disabled processing `create_application` sk requests
	DisableCreation = os.Getenv("DISABLE_RESOURCE_CREATION")
	// DisableDeletion disabled processing `destroy_application` sk requests
	DisableDeletion = os.Getenv("DISABLE_RESOURCE_DELETION")

	conf           = config.Get()
	identityHeader string
)

func main() {
	l.InitLogger(conf)

	l.Log.Infof("Listening to Kafka at: %v", conf.KafkaBrokers)
	l.Log.Infof("Talking to Sources API at: %v", fmt.Sprintf("%v://%v:%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort))

	l.Log.Info("SuperKey Worker started.")

	initMetrics()

	// returns real topic name from config (identical in local and app-interface mode)
	requestQueue, found := conf.KafkaTopics[SuperKeyRequestQueue]
	if !found {
		requestQueue = SuperKeyRequestQueue
	}

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction(requestQueue, func(msg kafka.Message) {
		l.Log.Infof("Started processing message %s", string(msg.Value))
		processSuperkeyRequest(msg)
		l.Log.Infof("Finished processing message %s", string(msg.Value))
	})
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

// processSuperkeyRequest - processes messages.
func processSuperkeyRequest(msg kafka.Message) {
	eventType := getHeader("event_type", msg.Headers)
	identityHeader = getHeader("x-rh-identity", msg.Headers)
	if identityHeader == "" {
		l.Log.Warnf("No x-rh-identity header found for message, skipping...")
	}

	switch eventType {
	case "create_application":
		if DisableCreation == "true" {
			l.Log.Infof("Skipping create_application request: %v", msg.Value)
			return
		}

		l.Log.Info("Processing `create_application` request")
		req, err := parseSuperKeyCreateRequest(msg.Value)
		if err != nil {
			l.Log.Warnf("Error parsing request: %v", err)
			return
		}

		createResources(req)
		l.Log.Infof("Finished processing `create_application` request for tenant %v type %v", req.TenantID, req.ApplicationType)

	case "destroy_application":
		if DisableDeletion == "true" {
			l.Log.Infof("Skipping destroy_application request: %v", msg.Value)
			return
		}

		l.Log.Info("Processing `destroy_application` request")
		req, err := parseSuperKeyDestroyRequest(msg.Value)
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

		err := req.MarkSourceUnavailable(err, newApp, identityHeader)
		if err != nil {
			l.Log.Errorf("Error during PATCH unavailable to application/source: %v", err)
		}

		return
	}
	l.Log.Infof("Finished Forging request: %v", req)

	err = newApp.CreateInSourcesAPI(identityHeader)
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

func parseSuperKeyCreateRequest(value []byte) (*superkey.CreateRequest, error) {
	request := superkey.CreateRequest{}
	err := json.Unmarshal(value, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func parseSuperKeyDestroyRequest(value []byte) (*superkey.DestroyRequest, error) {
	request := superkey.DestroyRequest{}
	err := json.Unmarshal(value, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func getHeader(name string, headers []kafka.Header) string {
	for _, header := range headers {
		if header.Key == name {
			return string(header.Value)
		}
	}

	return ""
}
