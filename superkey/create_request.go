package superkey

import (
	"context"
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// MarkSourceUnavailable marks the application and source as unavailable, while
// also marking the application's availability_status_error to what AWS updated
// us with.
func (req *CreateRequest) MarkSourceUnavailable(ctx context.Context, incomingErr error, newApplication *ForgedApplication) error {
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

	sourcesClient := sources.NewSourcesClient(config.Get())

	authData := &sources.AuthenticationData{
		IdentityHeader: req.IdentityHeader,
		OrgId:          req.OrgIdHeader,
	}

	patchAppRequestBody := &sources.PatchApplicationRequest{
		AvailabilityStatus:      &availabilityStatus,
		AvailabilityStatusError: &availabilityStatusError,
		Extra:                   extra,
	}

	err := sourcesClient.PatchApplication(ctx, authData, req.ApplicationID, patchAppRequestBody)
	if err != nil {
		return fmt.Errorf("error while updating the application: %w", err)
	}

	l.LogWithContext(ctx).Info(`Application marked as "unavailable"`)

	patchSourceRequestBody := &sources.PatchSourceRequest{AvailabilityStatus: &availabilityStatus}
	err = sourcesClient.PatchSource(ctx, authData, req.SourceID, patchSourceRequestBody)
	if err != nil {
		return fmt.Errorf("error while updating the source: %w", err)
	}

	l.LogWithContext(ctx).Info(`Source marked as "unavailable"`)

	return nil
}
