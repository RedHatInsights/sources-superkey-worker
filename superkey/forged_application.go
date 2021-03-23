package superkey

import (
	"context"
	"time"

	sourcesapi "github.com/lindgrenj6/sources-api-client-go"
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
func (f *ForgedApplication) CreateInSourcesAPI(identityHeader string) error {
	client := sources.NewAPIClient(identityHeader)

	l.Log.Info("Sleeping to prevent IAM Race Condition")
	// IAM is slow, this prevents the race condition of the POST happening
	// before it's ready.
	time.Sleep(7 * time.Second)

	l.Log.Infof("Posting resources back to Sources API: %v", f)
	err := f.storeSuperKeyData(client)
	if err != nil {
		return err
	}
	err = f.createAuthentications(client)
	if err != nil {
		return err
	}
	err = f.checkAvailability(client)
	if err != nil {
		return err
	}

	l.Log.Infof("Finished posting resources back to Sources API: %v", f)
	return nil
}

func (f *ForgedApplication) createAuthentications(client *sourcesapi.APIClient) error {
	authentications := []sourcesapi.BulkCreatePayloadAuthentications{f.Product.AuthPayload}
	payload := sourcesapi.BulkCreatePayload{Authentications: &authentications}

	_, r, err := client.DefaultApi.BulkCreate(context.Background()).BulkCreatePayload(payload).Execute()

	if r == nil || r.StatusCode != 201 {
		l.Log.Errorf("Failed to create authentications %v", err)
		return err
	}

	return nil
}

func (f *ForgedApplication) storeSuperKeyData(client *sourcesapi.APIClient) error {
	request := client.DefaultApi.UpdateApplication(context.Background(), f.Request.ApplicationID)
	request = request.Application(sourcesapi.Application{Extra: &f.Product.Extra})

	r, err := request.Execute()

	if r == nil || r.StatusCode != 204 {
		l.Log.Errorf("Failed to update application with superkey data %v", err)
		return err
	}

	return nil
}

func (f *ForgedApplication) checkAvailability(client *sourcesapi.APIClient) error {
	request := client.DefaultApi.CheckAvailabilitySource(context.Background(), f.Product.SourceID)
	r, err := request.Execute()

	if r == nil || r.StatusCode != 202 {
		l.Log.Errorf("Failed to check Source availability: %v", err)
		return err
	}

	return nil
}

// CreatePayload - creates + populates the `Product` field on the Forged Application
// based on the steps completed.
func (f *ForgedApplication) CreatePayload(username, password, appType *string) {
	authtype := f.Request.Extra["result_type"]
	resourceType := "application"

	f.Product = &App{
		SourceID: f.Request.SourceID,
		Extra:    f.applicationExtraPayload(),
		AuthPayload: sourcesapi.BulkCreatePayloadAuthentications{
			Authtype:     &authtype,
			Username:     username,
			Password:     password,
			ResourceName: &f.Request.ApplicationID,
			ResourceType: &resourceType,
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
