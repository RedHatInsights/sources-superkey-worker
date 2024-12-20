package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/sirupsen/logrus"
)

// sourcesClient holds the required information to be able to send requests back to the Sources API.
type sourcesClient struct {
	baseV31URL         *url.URL
	baseV20InternalUrl *url.URL
	config             *config.SuperKeyWorkerConfig
}

// AuthenticationData
type AuthenticationData struct {
	IdentityHeader string
	OrgId          string
	AccountNumber  string
}

// PatchApplicationRequest represents the fields that we might want to update when updating the application's details.
//
// The AvailabilityStatus field represents the current application's availability status.
// The AvailabilityStatusError field gives information about why the status might not be "available".
// The Extra field allows adding extra fields to the application, such as the Superkey key.
type PatchApplicationRequest struct {
	AvailabilityStatus      *string                `json:"availability_status"`
	AvailabilityStatusError *string                `json:"availability_status_error"`
	Extra                   map[string]interface{} `json:"extra"`
}

// PatchSourceRequest represents the availability status field that we might want to update in a Source.
//
// The AvailabilityStatus field represents the current sources' availability status.
type PatchSourceRequest struct {
	AvailabilityStatus *string `json:"availability_status"`
}

// NewSourcesClient initializes a new SourcesClient to be able to communicate with the Sources API.
func NewSourcesClient(config *config.SuperKeyWorkerConfig) *sourcesClient {
	return &sourcesClient{
		baseV20InternalUrl: &url.URL{
			Host:   fmt.Sprintf("%s:%d", config.SourcesHost, config.SourcesPort),
			Path:   "/internal/v2.0/",
			Scheme: config.SourcesScheme,
		},
		baseV31URL: &url.URL{
			Host:   fmt.Sprintf("%s:%d", config.SourcesHost, config.SourcesPort),
			Path:   "/api/sources/v3.1",
			Scheme: config.SourcesScheme,
		},
		config: config,
	}
}

func (sc *sourcesClient) TriggerSourceAvailabilityCheck(ctx context.Context, authData *AuthenticationData, sourceId string) error {
	checkAvailabilityUrl := sc.baseV31URL.JoinPath("/sources/", url.PathEscape(sourceId), "/check_availability")

	return sc.sendRequest(ctx, http.MethodPost, checkAvailabilityUrl, authData, nil, nil)
}

func (sc *sourcesClient) CreateAuthentication(ctx context.Context, authData *AuthenticationData, sourcesAuthentication *model.AuthenticationCreateRequest) (*model.AuthenticationResponse, error) {
	createAuthenticationUrl := sc.baseV31URL.JoinPath("/authentications")

	var createdAuthentication model.AuthenticationResponse
	err := sc.sendRequest(ctx, http.MethodPost, createAuthenticationUrl, authData, sourcesAuthentication, createdAuthentication)
	if err != nil {
		return nil, fmt.Errorf("error while creating authentication: %w", err)
	}

	return &createdAuthentication, nil
}

func (sc *sourcesClient) CreateApplicationAuthentication(ctx context.Context, authData *AuthenticationData, appAuthCreateRequest *model.ApplicationAuthenticationCreateRequest) error {
	createApplicationAuthenticationUrl := sc.baseV31URL.JoinPath("/application_authentications")

	err := sc.sendRequest(ctx, http.MethodPost, createApplicationAuthenticationUrl, authData, appAuthCreateRequest, nil)
	if err != nil {
		return fmt.Errorf("error while creating the application authentication: %w", err)
	}

	return nil
}

func (sc *sourcesClient) PatchApplication(ctx context.Context, authData *AuthenticationData, appId string, patchApplicationRequest *PatchApplicationRequest) error {
	patchApplicationUrl := sc.baseV31URL.JoinPath("/applications/", url.PathEscape(appId))

	return sc.sendRequest(ctx, http.MethodPatch, patchApplicationUrl, authData, patchApplicationRequest, nil)
}

func (sc *sourcesClient) PatchSource(ctx context.Context, authData *AuthenticationData, sourceId string, patchSourceRequest *PatchSourceRequest) error {
	patchSourceUrl := sc.baseV31URL.JoinPath("/sources/" + url.PathEscape(sourceId))

	return sc.sendRequest(ctx, http.MethodPatch, patchSourceUrl, authData, patchSourceRequest, nil)
}

func (sc *sourcesClient) GetInternalAuthentication(ctx context.Context, authData *AuthenticationData, authId string) (*model.AuthenticationInternalResponse, error) {
	getInternalAuthUrl := sc.baseV20InternalUrl.JoinPath("/authentications/", url.PathEscape(authId), "/?expose_encrypted_attribute[]=password")

	var authInternalResponse *model.AuthenticationInternalResponse = nil
	err := sc.sendRequest(ctx, http.MethodGet, getInternalAuthUrl, authData, nil, &authInternalResponse)

	return authInternalResponse, err
}

// sendRequest sends a request with the provided method and body to the given url, performing a maximum number of
// attempts and marshaling the incoming response's body. You can leave the body and the marshalTarget arguments empty
// if you do not require them.
func (sc *sourcesClient) sendRequest(ctx context.Context, httpMethod string, url *url.URL, authData *AuthenticationData, body interface{}, marshalTarget interface{}) error {
	// Set up a timeout so that the requests don't hang up forever.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// When a body is specified, attempt to marshal it as JSON.
	var requestBody *bytes.Buffer = nil
	if body != nil {
		tmp, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}

		requestBody = bytes.NewBuffer(tmp)
	}

	// Create the request.
	request, err := http.NewRequestWithContext(ctx, httpMethod, url.String(), requestBody)
	if err != nil {
		return fmt.Errorf(`failed to create request: %w`, err)
	}

	// Include the headers in the request.
	sc.addAuthenticationHeaders(request, authData)

	// Perform the actual request.
	var response *http.Response
	for attempt := 0; attempt < sc.config.SourcesRequestsMaxAttempts; attempt++ {
		response, err = http.DefaultClient.Do(request)

		// The "err" check is to avoid nil dereference errors, since if we attempt checking for the status code
		// directly when an error has occurred, the "response" struct might be nil.
		if err == nil && sc.isStatusCodeFamilyOf2xx(response.StatusCode) {
			break
		}

		// When there are no errors but the status code is not the expected one, we attempt to drain the body so that
		// the default client can reuse the connection, and then we close the body to avoid memory leaks.
		if err == nil && !sc.isStatusCodeFamilyOf2xx(response.StatusCode) {
			_, drainErr := io.Copy(io.Discard, response.Body)

			if drainErr != nil {
				l.Log.WithFields(logrus.Fields{}).Warnf("Unable to drain response body. The connection will not be reused by the default HTTP client: %s", drainErr)
			}

			if closeErr := response.Body.Close(); closeErr != nil {
				l.Log.WithFields(logrus.Fields{}).Errorf("Failed to close incoming response's body: %s", closeErr)
			}

			l.Log.WithFields(logrus.Fields{}).Debugf(`Unexpected status code received. Want "2xx", got "%d"`, response.StatusCode)
			continue
		}

		if err != nil {
			l.Log.WithFields(logrus.Fields{}).Warn("Failed to send request. Retrying...")
			l.Log.WithFields(logrus.Fields{}).Debugf("Failed to send request. Retrying... Cause: %s", err)
		}
	}

	// In the case in which we deplete all the attempts, we have to return the error and stop the execution here.
	if err != nil || response == nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Always read the response body, in case we need to return it in an error or marshal it to a struct.
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf(`failed to read response body: %w`, err)
	}

	// Make sure that the status code is a "2xx" one.
	if !sc.isStatusCodeFamilyOf2xx(response.StatusCode) {
		return fmt.Errorf(`unexpected status code received. Want "2xx", got "%d". Response body: %s`, response.StatusCode, string(responseBody))
	}

	// We might need to marshal the incoming response in the specified struct.
	if marshalTarget != nil {
		err = json.Unmarshal(responseBody, &marshalTarget)
		if err != nil {
			return fmt.Errorf(`failed to unmarshal response body: %w`, err)
		}
	}

	return nil
}

// isStatusCodeFamilyOf2xx returns true if the given status code is a 2xx status code.
func (sc *sourcesClient) isStatusCodeFamilyOf2xx(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func (sc *sourcesClient) addAuthenticationHeaders(request *http.Request, authData *AuthenticationData) {
	request.Header.Add("Content-Type", "application/json")

	if sc.config.SourcesPSK == "" {
		var xRhId string

		if authData.IdentityHeader == "" {
			xRhId = encodeIdentity(authData.AccountNumber, authData.OrgId)
		} else {
			xRhId = authData.IdentityHeader
		}

		request.Header.Add("x-rh-identity", xRhId)
	} else {
		request.Header.Add("x-rh-sources-psk", sc.config.SourcesPSK)

		if authData.AccountNumber != "" {
			request.Header.Add("x-rh-sources-account-number", authData.AccountNumber)
		}

		if authData.IdentityHeader != "" {
			request.Header.Add("x-rh-identity", authData.IdentityHeader)
		}

		if authData.OrgId != "" {
			request.Header.Add("x-rh-org-id", authData.OrgId)
		}
	}
}
