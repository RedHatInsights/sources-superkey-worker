package superkey

import (
	"os"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
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
func (f *ForgedApplication) CreateInSourcesAPI() error {
	l.Log.Info("Sleeping to prevent IAM Race Condition")
	// IAM is slow, this prevents the race condition of the POST happening
	// before it's ready.
	time.Sleep(waitTime() * time.Second)

	// create a sources client for our identity + account number
	if f.SourcesClient == nil {
		f.SourcesClient = &sources.SourcesClient{IdentityHeader: f.Request.IdentityHeader, OrgId: f.Request.OrgIdHeader, AccountNumber: f.Request.TenantID}
	}

	l.Log.Infof("Posting resources back to Sources API: %v", f)
	err := f.storeSuperKeyData()
	if err != nil {
		return err
	}
	err = f.createAuthentications()
	if err != nil {
		return err
	}
	err = f.checkAvailability()
	if err != nil {
		return err
	}

	l.Log.Infof("Finished posting resources back to Sources API: %v", f)
	return nil
}

func (f *ForgedApplication) createAuthentications() error {
	extra := f.Product.Extra
	externalID, ok := f.Request.Extra["external_id"]
	if ok {
		extra["external_id"] = externalID
	}

	auth := model.AuthenticationCreateRequest{
		AuthType:      f.Product.AuthPayload.AuthType,
		Username:      f.Product.AuthPayload.Username,
		ResourceType:  f.Product.AuthPayload.ResourceType,
		ResourceIDRaw: f.Request.ApplicationID,
		Extra:         extra,
	}

	err := f.SourcesClient.CreateAuthentication(&auth)
	if err != nil {
		l.Log.Errorf("Failed to create authentication: %v", err)
		return err
	}

	return nil
}

func (f *ForgedApplication) storeSuperKeyData() error {
	err := f.SourcesClient.PatchApplication(f.Request.TenantID, f.Request.ApplicationID, map[string]interface{}{
		"extra": f.Product.Extra,
	})

	if err != nil {
		l.Log.Errorf("Failed to update application with superkey data %v", err)
		return err
	}

	return nil
}

func (f *ForgedApplication) checkAvailability() error {
	err := f.SourcesClient.CheckAvailability(f.Product.SourceID)
	if err != nil {
		l.Log.Errorf("Failed to check Source availability: %v", err)
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
