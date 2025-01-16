package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

// sourcesClient holds the required information to be able to send requests back to the Sources API.
type sourcesClient struct {
	baseV31URL         *url.URL
	baseV20InternalUrl *url.URL
	config             *config.SuperKeyWorkerConfig
}

// AuthenticationData holds the required authentication elements that need to be sent back to the Sources API when
// making a request.
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
			Path:   "/internal/v2.0",
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

	// Set the logging fields.
	ctx = l.WithSourceId(ctx, sourceId)

	return sc.sendRequest(ctx, http.MethodPost, checkAvailabilityUrl, authData, nil, nil)
}

func (sc *sourcesClient) CreateAuthentication(ctx context.Context, authData *AuthenticationData, sourcesAuthentication *model.AuthenticationCreateRequest) (*model.AuthenticationResponse, error) {
	createAuthenticationUrl := sc.baseV31URL.JoinPath("/authentications")

	// Set the logging fields.
	ctx = l.WithResourceType(ctx, sourcesAuthentication.ResourceType)
	ctx = l.WithResourceId(ctx, strconv.FormatInt(sourcesAuthentication.ResourceID, 10))

	var createdAuthentication *model.AuthenticationResponse = nil
	err := sc.sendRequest(ctx, http.MethodPost, createAuthenticationUrl, authData, sourcesAuthentication, &createdAuthentication)
	if err != nil {
		return nil, fmt.Errorf("error while creating authentication: %w", err)
	}

	return createdAuthentication, nil
}

func (sc *sourcesClient) CreateApplicationAuthentication(ctx context.Context, authData *AuthenticationData, appAuthCreateRequest *model.ApplicationAuthenticationCreateRequest) error {
	createApplicationAuthenticationUrl := sc.baseV31URL.JoinPath("/application_authentications")

	// Set the logging fields.
	ctx = l.WithApplicationId(ctx, strconv.FormatInt(appAuthCreateRequest.ApplicationID, 10))
	ctx = l.WithAuthenticationId(ctx, strconv.FormatInt(appAuthCreateRequest.AuthenticationID, 10))

	err := sc.sendRequest(ctx, http.MethodPost, createApplicationAuthenticationUrl, authData, appAuthCreateRequest, nil)
	if err != nil {
		return fmt.Errorf("error while creating the application authentication: %w", err)
	}

	return nil
}

func (sc *sourcesClient) PatchApplication(ctx context.Context, authData *AuthenticationData, appId string, patchApplicationRequest *PatchApplicationRequest) error {
	patchApplicationUrl := sc.baseV31URL.JoinPath("/applications/", url.PathEscape(appId))

	// Set the logging fields.
	ctx = l.WithApplicationId(ctx, appId)

	return sc.sendRequest(ctx, http.MethodPatch, patchApplicationUrl, authData, patchApplicationRequest, nil)
}

func (sc *sourcesClient) PatchSource(ctx context.Context, authData *AuthenticationData, sourceId string, patchSourceRequest *PatchSourceRequest) error {
	patchSourceUrl := sc.baseV31URL.JoinPath("/sources/" + url.PathEscape(sourceId))

	// Set the logging fields.
	ctx = l.WithSourceId(ctx, sourceId)

	return sc.sendRequest(ctx, http.MethodPatch, patchSourceUrl, authData, patchSourceRequest, nil)
}

func (sc *sourcesClient) GetInternalAuthentication(ctx context.Context, authData *AuthenticationData, authId string) (*model.AuthenticationInternalResponse, error) {
	getInternalAuthUrl := sc.baseV20InternalUrl.JoinPath("/authentications/", url.PathEscape(authId), "/?expose_encrypted_attribute[]=password")

	// Set the logging fields.
	ctx = l.WithAuthenticationId(ctx, authId)

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

	// Build the URL for the request. Unfortunately, we cannot simply use "url.String()" because it does escape the
	// special path characters, and for calls like getting the internal authentication, it makes the Sources API to
	// return a "Not found" response. Things like "[]" get escaped and therefore the URL does not match Sources'
	// router, which causes issues.
	urlRaw := fmt.Sprintf("%s://%s:%s%s", url.Scheme, url.Hostname(), url.Port(), url.Path)

	// Create the request. Apparently a nil "*bytes.Buffer" counts as a body, which in turn makes the code panic when
	// creating a new request. That is why we add another "if" statement to guard us against that.
	var request *http.Request
	var err error
	if requestBody != nil {
		request, err = http.NewRequestWithContext(ctx, httpMethod, urlRaw, requestBody)
	} else {
		request, err = http.NewRequestWithContext(ctx, httpMethod, urlRaw, nil)
	}

	if err != nil {
		return fmt.Errorf(`failed to create request: %w`, err)
	}

	// Include the headers in the request.
	sc.addAuthenticationHeaders(request, authData)

	// Add the logging fields to the context.
	ctx = l.WithHttpMethod(ctx, httpMethod)
	ctx = l.WithURL(ctx, urlRaw)
	ctx = l.WithHTTPHeaders(ctx, request.Header)

	// Perform the actual request.
	var response *http.Response
	var responseBody []byte
	for attempt := 0; attempt < sc.config.SourcesRequestsMaxAttempts; attempt++ {
		response, err = http.DefaultClient.Do(request)

		// The "err" check is to avoid nil dereference errors, since if we attempt checking for the status code
		// or attempt reading the response body when an error has occurred, the "response" struct might be nil.
		if err == nil {
			// Declare an error variable to avoid shadowing the response body one.
			var readErr error

			// Read the response body every time to ensure that the body is completely drained when retrying, or that
			// it is available if it needs to be printed or used elsewhere. Draining the body is important so that the
			// connection can be reused.
			responseBody, readErr = io.ReadAll(response.Body)
			if readErr != nil {
				return fmt.Errorf(`failed to read response body: %w`, readErr)
			}

			// Make sure to close the body to avoid memory leaks.
			if closeErr := response.Body.Close(); closeErr != nil {
				l.LogWithContext(ctx).Errorf("Failed to close incoming response's body: %s", closeErr)
			}

			// When the status code is "2xx", we can simply exit the loop. Otherwise, we need to keep retrying until we
			// deplete the attempts.
			if sc.isStatusCodeFamilyOf2xx(response.StatusCode) {
				break
			} else {
				l.LogWithContext(ctx).WithField("response_body", responseBody).Debugf(`Unexpected status code received. Want "2xx", got "%d"`, response.StatusCode)
			}
		} else {
			l.LogWithContext(ctx).Warn("Failed to send request. Retrying...")
			l.LogWithContext(ctx).Debugf("Failed to send request. Retrying... Cause: %s", err)
		}
	}

	// In the case in which we deplete all the attempts, we have to return the error and stop the execution here.
	if err != nil || response == nil {
		return fmt.Errorf("failed to send request: %w", err)
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
