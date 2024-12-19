package superkey

import (
	"fmt"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
	"github.com/sirupsen/logrus"
)

// MarkSourceUnavailable marks the application and source as unavailable, while
// also marking the application's availability_status_error to what AWS updated
// us with.
func (req *CreateRequest) MarkSourceUnavailable(incomingErr error, newApplication *ForgedApplication) error {
	availabilityStatus := "unavailable"
	availabilityStatusError := fmt.Sprintf("Resource Creation error: failed to create resources in Amazon. Error: %s", incomingErr)
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

	err := newApplication.SourcesClient.PatchApplication(req.TenantID, req.SourceID, req.ApplicationID, map[string]interface{}{
		"availability_status":       availabilityStatus,
		"availability_status_error": availabilityStatusError,
		"extra":                     extra,
	})
	if err != nil {
		return fmt.Errorf("error while updating the application: %w", err)
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": req.TenantID, "source_id": req.SourceID, "application_id": req.ApplicationID, "application_type": req.ApplicationType}).Info(`Application marked as "unavailable"`)

	err = newApplication.SourcesClient.PatchSource(req.TenantID, req.SourceID, map[string]interface{}{
		"availability_status": availabilityStatus,
	})
	if err != nil {
		return fmt.Errorf("error while updating the source: %w", err)
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": req.TenantID, "source_id": req.SourceID, "application_id": req.ApplicationID, "application_type": req.ApplicationType}).Info(`Source marked as "unavailable"`)

	return nil
}
