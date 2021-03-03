package main

import (
	"encoding/json"
	"fmt"
	"os"

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

	conf = config.Get()
)

func main() {
	l.InitLogger(conf)

	l.Log.Infof("Listening to Kafka at: %v", conf.KafkaBrokers)
	l.Log.Infof("Talking to Sources API at: %v", fmt.Sprintf("%v://%v:%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort))

	l.Log.Info("SuperKey Worker started.")

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction(SuperKeyRequestQueue, func(msg kafka.Message) {
		l.Log.Infof("Started processing message %s", string(msg.Value))
		processSuperkeyRequest(msg)
		l.Log.Infof("Finished processing message %s", string(msg.Value))
	})
}

// processSuperkeyRequest - processes messages.
func processSuperkeyRequest(msg kafka.Message) {
	eventType := getEventType(msg.Headers)

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

// getEventType - iterates through headers to find the `event_type` header
// from miq-messaging.
// returns: event_type value
func getEventType(headers []kafka.Header) string {
	for _, header := range headers {
		if header.Key == "event_type" {
			return string(header.Value)
		}
	}

	return ""
}
