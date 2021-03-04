package superkey

import (
	"context"
	"fmt"

	sourcesapi "github.com/lindgrenj6/sources-api-client-go"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// MarkSourceUnavailable marks the application and source as unavailable, while
// also marking the application's availability_status_error to what AWS updated
// us with.
func (req *CreateRequest) MarkSourceUnavailable(err error, newApplication *ForgedApplication) error {
	client := sources.NewAPIClient(req.TenantID)
	availabilityStatus := "unavailable"
	availabilityStatusError := fmt.Sprintf("Resource Creation erorr: failed to create resources in amazon, error: %v", err)
	extra := make(map[string]interface{})

	// creating the aws resources was at least partially successful, need to store
	// how far progress was made.
	if newApplication != nil {
		extra = newApplication.applicationExtraPayload()
	}

	appRequest := client.DefaultApi.UpdateApplication(context.Background(), req.ApplicationID)
	appRequest = appRequest.Application(
		sourcesapi.Application{
			AvailabilityStatus:      &availabilityStatus,
			AvailabilityStatusError: &availabilityStatusError,
			Extra:                   &extra,
		},
	)

	r, err := appRequest.Execute()
	if r == nil || r.StatusCode != 204 {
		l.Log.Errorf("Failed to update application with error message %v", err)
		return err
	}

	srcRequest := client.DefaultApi.UpdateSource(context.Background(), req.SourceID)
	srcRequest = srcRequest.Source(
		sourcesapi.Source{AvailabilityStatus: &availabilityStatus},
	)

	r, err = srcRequest.Execute()
	if r == nil || r.StatusCode != 204 {
		l.Log.Errorf("Failed to update source with error message %v", err)
		return err
	}

	return nil
}
