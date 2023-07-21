package superkey

import (
	"fmt"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// MarkSourceUnavailable marks the application and source as unavailable, while
// also marking the application's availability_status_error to what AWS updated
// us with.
func (req *CreateRequest) MarkSourceUnavailable(incomingErr error, newApplication *ForgedApplication) error {
	availabilityStatus := "unavailable"
	availabilityStatusError := fmt.Sprintf("Resource Creation erorr: failed to create resources in amazon, error: %v", incomingErr)
	extra := make(map[string]interface{})

	// creating the aws resources was at least partially successful, need to store
	// how far progress was made.
	if newApplication != nil {
		extra = newApplication.applicationExtraPayload()
	} else {
		// this can happen if the request fails _very early_ in the request
		// process, e.g. if the superkey auth is unavailable.
		newApplication = &ForgedApplication{}
	}

	if newApplication.SourcesClient == nil {
		newApplication.SourcesClient = &sources.SourcesClient{IdentityHeader: req.IdentityHeader, OrgId: req.OrgIdHeader, AccountNumber: req.TenantID}
	}

	l.Log.Infof("Marking Application %v Unavailable with message: %v", req.ApplicationID, availabilityStatusError)

	err := newApplication.SourcesClient.PatchApplication(req.TenantID, req.ApplicationID, map[string]interface{}{
		"availability_status":       availabilityStatus,
		"availability_status_error": availabilityStatusError,
		"extra":                     extra,
	})
	if err != nil {
		l.Log.Errorf("Failed to update application with error message %v", err)
		return err
	}

	l.Log.Infof("Marking Source %v Unavailable", req.SourceID)
	err = newApplication.SourcesClient.PatchSource(req.TenantID, req.SourceID, map[string]interface{}{
		"availability_status": availabilityStatus,
	})
	if err != nil {
		l.Log.Errorf("Failed to update source with error message %v", err)
		return err
	}

	l.Log.Infof("Finished Marking Source %v + Application %v Unavailable", req.SourceID, req.ApplicationID)
	return nil
}
