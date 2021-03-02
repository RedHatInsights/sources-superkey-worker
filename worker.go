package main

import (
	"encoding/json"
	"os"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/provider"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
	"github.com/segmentio/kafka-go"
)

var (
	// DisableCreation disabled processing `create_application` sk requests
	DisableCreation = os.Getenv("DISABLE_RESOURCE_CREATION")
	// DisableDeletion disabled processing `destroy_application` sk requests
	DisableDeletion = os.Getenv("DISABLE_RESOURCE_DELETION")
)

// ProcessSuperkeyRequest - processes messages.
func ProcessSuperkeyRequest(msg kafka.Message) {
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

		l.Log.Infof("Un-Forging request: %v", req)
		errors := provider.TearDown(superkey.ReconstructForgedApplication(req))
		if len(errors) != 0 {
			for _, err := range errors {
				l.Log.Errorf("Error during teardown: %v", err)
			}
		}

		l.Log.Infof("Finished processing `destroy_application` request for GUID %v")

	default:
		l.Log.Warn("Unknown event_type")
	}
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
