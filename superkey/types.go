package superkey

import (
	"context"

	sourcesapi "github.com/lindgrenj6/sources-api-client-go"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// CreateRequest - struct representing a request for a superkey
type CreateRequest struct {
	TenantID         string            `json:"tenant_id"`
	SourceID         string            `json:"source_id"`
	AuthenticationID string            `json:"authentication_id"`
	ApplicationType  string            `json:"application_type"`
	Extra            map[string]string `json:"extra"`
	Provider         string            `json:"provider"`
	SuperKeySteps    []Step            `json:"superkey_steps"`
}

// Step - struct representing a step for SuperKey
type Step struct {
	Step          int               `json:"step"`
	Name          string            `json:"name"`
	Payload       string            `json:"payload"`
	Substitutions map[string]string `json:"substitutions"`
}

// DestroyRequest - struct representing a teardown request for an application
// created through superkey
type DestroyRequest struct {
	TenantID         string   `json:"tenant_id"`
	ApplicationID    string   `json:"application_id"`
	AuthenticationID string   `json:"authentication_id"`
	Provider         string   `json:"provider"`
	GUID             string   `json:"guid"`
	Resources        []string `json:"resources"`
}

// App - represents an application that can be posted to sources after being populated
type App struct {
	SourceID    string                                      `json:"source_id"`
	Type        string                                      `json:"application_type"`
	Extra       map[string]interface{}                      `json:"extra"`
	AuthPayload sourcesapi.BulkCreatePayloadAuthentications `json:"authentication_payload"`
}

// Provider the interface for all of the superkey providers
// currently just a single method is needed (ForgeApplication)
type Provider interface {
	ForgeApplication(*CreateRequest) (*ForgedApplication, error)
	TearDown(*ForgedApplication) []error
}

// ForgedApplication - struct to hold the output of a superkey create_application
// request
type ForgedApplication struct {
	Product        *App
	StepsCompleted map[string]map[string]string
	Request        *CreateRequest
	Client         Provider
	GUID           string
}

// MarkCompleted marks a step as completed, storing the passed in hash of data.
func (f *ForgedApplication) MarkCompleted(name string, data map[string]string) {
	f.StepsCompleted[name] = data
}

// CreateInSourcesAPI - creates the forged application in sources
func (f *ForgedApplication) CreateInSourcesAPI() error {
	client := sources.NewAPIClient(f.Request.TenantID)

	applications := []sourcesapi.BulkCreatePayloadApplications{
		{
			Type:       &f.Product.Type,
			Extra:      &f.Product.Extra,
			SourceName: &f.Request.SourceID,
		},
	}
	authentications := []sourcesapi.BulkCreatePayloadAuthentications{
		f.Product.AuthPayload,
	}

	payload := sourcesapi.BulkCreatePayload{Applications: &applications, Authentications: &authentications}

	_, r, err := client.DefaultApi.BulkCreate(context.Background()).BulkCreatePayload(payload).Execute()

	if r.StatusCode != 201 {
		l.Log.Errorf("Failed to create application + authentications %v", err)

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
		Type:     *appType,
		Extra: map[string]interface{}{
			"_superkey": map[string]interface{}{
				"steps": f.StepsCompleted,
				"guid":  f.GUID,
			},
		},
		AuthPayload: sourcesapi.BulkCreatePayloadAuthentications{
			Authtype:     &authtype,
			Username:     username,
			Password:     password,
			ResourceName: appType,
			ResourceType: &resourceType,
		},
	}
}
