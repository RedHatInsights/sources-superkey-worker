package sources

import (
	"context"

	"github.com/RedHatInsights/sources-api-go/model"
)

// RestClient represents the Sources' endpoints that are required for the Superkey to be able to talk to the Sources'
// API.
type RestClient interface {
	// TriggerSourceAvailabilityCheck triggers an availability status check in the Sources API for the given source.
	TriggerSourceAvailabilityCheck(ctx context.Context, authData *AuthenticationData, sourceId string) error
	// CreateAuthentication creates an authentication in Sources.
	CreateAuthentication(ctx context.Context, authData *AuthenticationData, sourcesAuthentication *model.AuthenticationCreateRequest) (*model.AuthenticationResponse, error)
	// CreateApplicationAuthentication links the created authentication with an application in Sources.
	CreateApplicationAuthentication(ctx context.Context, authData *AuthenticationData, appAuthCreateRequest *model.ApplicationAuthenticationCreateRequest) error
	// PatchApplication modifies an application in Sources.
	PatchApplication(ctx context.Context, authData *AuthenticationData, appId string, patchApplicationRequest *PatchApplicationRequest) error
	// PatchSource modifies an application in Sources.
	PatchSource(ctx context.Context, authData *AuthenticationData, sourceId string, patchSourceRequest *PatchSourceRequest) error
	// GetInternalAuthentication fetches an authentication using the internal Sources' endpoint, which ensure that the authentication will have the password as well.
	GetInternalAuthentication(ctx context.Context, authData *AuthenticationData, authId string) (*model.AuthenticationInternalResponse, error)
}
