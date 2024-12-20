package superkey

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// ReconstructForgedApplication - returns a ForgedApplication with the fields set
// during the initial creation, notably steps completed and the superkey id
func ReconstructForgedApplication(request *DestroyRequest) *ForgedApplication {
	return &ForgedApplication{
		StepsCompleted: request.StepsCompleted,
		Request: &CreateRequest{
			TenantID:      request.TenantID,
			SuperKey:      request.SuperKey,
			Provider:      request.Provider,
			SuperKeySteps: request.SuperKeySteps,
		},
		GUID: request.GUID,
	}
}

// MarkCompleted marks a step as completed, storing the passed in hash of data.
func (f *ForgedApplication) MarkCompleted(name string, data map[string]string) {
	f.StepsCompleted[name] = data
}

// CreateInSourcesAPI - creates the forged application in sources
func (f *ForgedApplication) CreateInSourcesAPI(ctx context.Context) error {
	l.LogWithContext(ctx).Debug("Sleeping to prevent IAM Race Condition")

	// IAM is slow, this prevents the race condition of the POST happening
	// before it's ready.
	time.Sleep(waitTime() * time.Second)

	sourcesClient := sources.NewSourcesClient(config.Get())

	l.LogWithContext(ctx).Debugf("Posting resources back to Sources API: %v", f)
	err := f.storeSuperKeyData(ctx, sourcesClient)
	if err != nil {
		return fmt.Errorf("error while storing the superkey data in Sources: %w", err)
	}

	l.LogWithContext(ctx).Info("Superkey data stored in Sources")

	err = f.createAuthentications(ctx, sourcesClient)
	if err != nil {
		return fmt.Errorf("error while creating the authentications in Sources: %w", err)
	}

	l.LogWithContext(ctx).Info("Authentications created in Sources")

	err = f.checkAvailability(ctx, sourcesClient)
	if err != nil {
		return fmt.Errorf("error while triggering an availability check in Sources: %w", err)
	}

	l.LogWithContext(ctx).Info("Availability check requested in Sources")
	l.LogWithContext(ctx).Debug("Finished creating and updating resources in Sources")

	return nil
}

func (f *ForgedApplication) createAuthentications(ctx context.Context, sourcesRestClient sources.RestClient) error {
	extra := map[string]interface{}{}
	externalID, ok := f.Request.Extra["external_id"]
	if ok {
		extra["external_id"] = externalID
	}

	authData := sources.AuthenticationData{
		IdentityHeader: f.Request.IdentityHeader,
		OrgId:          f.Request.OrgIdHeader,
	}

	auth := model.AuthenticationCreateRequest{
		AuthType:      f.Product.AuthPayload.AuthType,
		Username:      f.Product.AuthPayload.Username,
		ResourceType:  f.Product.AuthPayload.ResourceType,
		ResourceIDRaw: f.Request.ApplicationID,
		Extra:         extra,
	}

	createdAuthentication, err := sourcesRestClient.CreateAuthentication(ctx, &authData, &auth)
	if err != nil {
		return fmt.Errorf("error while creating the authentication in Sources: %w", err)
	}

	appAuthBody := model.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    f.Request.ApplicationID,
		AuthenticationIDRaw: createdAuthentication.ID,
	}

	err = sourcesRestClient.CreateApplicationAuthentication(ctx, &authData, &appAuthBody)
	if err != nil {
		return fmt.Errorf("error while associating the authentication with an application in Sources: %w", err)
	}

	return nil
}

func (f *ForgedApplication) storeSuperKeyData(ctx context.Context, sourcesRestClient sources.RestClient) error {
	authData := sources.AuthenticationData{
		IdentityHeader: f.Request.IdentityHeader,
		OrgId:          f.Request.OrgIdHeader,
	}

	err := sourcesRestClient.PatchApplication(ctx, &authData, f.Request.ApplicationID, &sources.PatchApplicationRequest{Extra: f.Product.Extra})
	if err != nil {
		return fmt.Errorf("failed to update application with superkey data: %w", err)
	}

	return nil
}

func (f *ForgedApplication) checkAvailability(ctx context.Context, sourcesRestClient sources.RestClient) error {
	authData := sources.AuthenticationData{
		IdentityHeader: f.Request.IdentityHeader,
		OrgId:          f.Request.OrgIdHeader,
	}

	err := sourcesRestClient.TriggerSourceAvailabilityCheck(ctx, &authData, f.Product.SourceID)
	if err != nil {
		return err
	}

	return nil
}

// CreatePayload - creates + populates the `Product` field on the Forged Application
// based on the steps completed.
func (f *ForgedApplication) CreatePayload(username, password, appType *string) {
	authtype := f.Request.Extra["result_type"]
	resourceId, _ := strconv.ParseInt(f.Request.ApplicationID, 10, 64)

	f.Product = &App{
		SourceID: f.Request.SourceID,
		Extra:    f.applicationExtraPayload(),
		AuthPayload: model.AuthenticationCreateRequest{
			AuthType:      authtype,
			Username:      username,
			ResourceIDRaw: resourceId,
			ResourceType:  "Application",
		},
	}
}

func (f *ForgedApplication) applicationExtraPayload() map[string]interface{} {
	extra := map[string]interface{}{
		"_superkey": map[string]interface{}{
			"steps":    f.StepsCompleted,
			"guid":     f.GUID,
			"provider": f.Request.Provider,
		},
	}

	// return the s3 bucket if we had it set during resource creation
	if f.StepsCompleted["s3"] != nil {
		extra["bucket"] = f.StepsCompleted["s3"]["output"]
	}

	return extra
}

const DEFAULT_SLEEP_TIME = 7

// read from the ENV first - if there isn't anything there fall back to the old
// default which is 7 seconds. defined ^^
func waitTime() time.Duration {
	raw := os.Getenv("AWS_WAIT_TIME")
	if raw == "" {
		return DEFAULT_SLEEP_TIME // chosen by fair dice roll
	}

	i, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		l.Log.Errorf("Failed to parse %q as sleep time - defaulting to %v", raw, DEFAULT_SLEEP_TIME)
		return DEFAULT_SLEEP_TIME
	}

	return time.Duration(i)
}
