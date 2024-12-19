package sources

import (
	"bytes"
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

var conf = config.Get()

type SourcesClient struct {
	IdentityHeader string
	OrgId          string
	AccountNumber  string
}

func (sc *SourcesClient) CheckAvailability(tenantId, sourceId string) error {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/sources/%v/check_availability", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, sourceId,
	))

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: sc.headers(),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "request_url": reqURL}).Debugf("Requesting an availability check")

	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return fmt.Errorf(`expecting a 202 status code, got "%d"`, resp.StatusCode)
	}

	return nil
}

func (sc *SourcesClient) CreateAuthentication(tenantId, sourceId, applicationId string, auth *model.AuthenticationCreateRequest) error {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/authentications", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort,
	))

	body, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "application_id": applicationId, "request_url": reqURL, "body": string(body)}).Debugf("Creating authentication in Sources")

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: sc.headers(),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(`expecting a 200 status code, got "%d" with body "%s"`, resp.StatusCode, string(b))
	}

	bytes, _ := io.ReadAll(resp.Body)
	var createdAuth model.AuthenticationResponse
	err = json.Unmarshal(bytes, &createdAuth)
	if err != nil {
		return fmt.Errorf("unable to unmarshal authentication creation response from Sources: %w", err)
	}

	err = sc.createApplicationAuthentication(tenantId, sourceId, applicationId, &model.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    auth.ResourceIDRaw,
		AuthenticationIDRaw: createdAuth.ID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *SourcesClient) PatchApplication(tenantId, sourceId, appID string, payload map[string]interface{}) error {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/applications/%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, appID,
	))

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "application_id": appID, "request_url": reqURL, "body": string(body)}).Debugf("Patching application in Sources")

	req := &http.Request{
		Method: http.MethodPatch,
		URL:    reqURL,
		Header: sc.headers(),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(`expecting a 200 status code, got "%d" with body "%s"`, resp.StatusCode, string(b))
	}

	return nil
}

func (sc *SourcesClient) PatchSource(tenantId, sourceId string, payload map[string]interface{}) error {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/sources/%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, sourceId,
	))

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPatch,
		URL:    reqURL,
		Header: sc.headers(),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "request_url": reqURL, "body": string(body)}).Debugf("Patching source in Sources")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(`expecting a 200 status code, got "%d" with body "%s"`, resp.StatusCode, string(b))
	}

	return nil
}

// GetInternalAuthentication requests an authentication via the internal sources api
// that way we can expose the password.
// returns: populated sources api Authentication object, error
func (sc *SourcesClient) GetInternalAuthentication(tenantId, sourceId, authID string) (*model.AuthenticationInternalResponse, error) {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/internal/v2.0/authentications/%v?expose_encrypted_attribute[]=password", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, authID,
	))

	req := &http.Request{
		Method: http.MethodGet,
		URL:    reqURL,
		Header: sc.headers(),
	}

	var res *http.Response
	var err error
	for retry := 0; retry < 5; retry++ {
		l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "request_url": reqURL}).Debug("Getting internal authentication from Sources")

		res, err = http.DefaultClient.Do(req)

		if err != nil || res.StatusCode == 200 {
			defer res.Body.Close()
			break
		} else {
			l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "authentication_id": authID}).Warn("Unable to fetch internal authentication. Retrying...")
			time.Sleep(3 * time.Second)
		}
	}

	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf(`unable to fetch internal authentication "%s" after 5 retries: %w`, authID, err)
	}

	data, _ := io.ReadAll(res.Body)
	auth := model.AuthenticationInternalResponse{}

	// unmarshaling the data from the request, the id comes back as a string which fills `err`
	// we can safely ignore that as long as username/pass are there.
	err = json.Unmarshal(data, &auth)
	if err != nil && (auth.Username == "" || auth.Password == "") {
		return nil, fmt.Errorf(`internal authentication "%s"'s username or password are empty'`, authID)
	}

	return &auth, nil
}

func (sc *SourcesClient) headers() map[string][]string {
	var headers = make(map[string][]string)

	headers["Content-Type"] = []string{"application/json"}

	if conf.SourcesPSK == "" {
		var xRhId string

		if sc.IdentityHeader == "" {
			xRhId = encodeIdentity(sc.AccountNumber, sc.OrgId)
		} else {
			xRhId = sc.IdentityHeader
		}

		headers["x-rh-identity"] = []string{xRhId}
	} else {
		headers["x-rh-sources-psk"] = []string{conf.SourcesPSK}

		if sc.AccountNumber != "" {
			headers["x-rh-sources-account-number"] = []string{sc.AccountNumber}
		}

		if sc.IdentityHeader != "" {
			headers["x-rh-identity"] = []string{sc.IdentityHeader}
		}

		if sc.OrgId != "" {
			headers["x-rh-org-id"] = []string{sc.OrgId}
		}
	}

	return headers
}

func (sc *SourcesClient) createApplicationAuthentication(tenantId, sourceId, applicationId string, appAuth *model.ApplicationAuthenticationCreateRequest) error {
	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/application_authentications", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort,
	))

	body, err := json.Marshal(appAuth)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: sc.headers(),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	l.Log.WithFields(logrus.Fields{"tenant_id": tenantId, "source_id": sourceId, "application_id": applicationId, "request_url": reqURL, "body": string(body)}).Debugf("Creating application authentication in Sources")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(`expecting a 200 status code, got "%d" with body "%s"`, resp.StatusCode, string(b))
	}

	return nil
}
