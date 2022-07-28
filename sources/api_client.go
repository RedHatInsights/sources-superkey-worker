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
)

var conf = config.Get()

type SourcesClient struct {
	IdentityHeader string
	OrgId          string
	AccountNumber  string
}

func (sc *SourcesClient) CheckAvailability(sourceID string) error {
	l.Log.Infof("Checking Availability for Source ID: %v", sourceID)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/sources/%v/check_availability", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, sourceID,
	))

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: sc.headers(),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check availability for source %v: %v", sourceID, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return fmt.Errorf("failed to check availability for source %v: bad return code %v", sourceID, resp.StatusCode)
	}

	return nil
}

func (sc *SourcesClient) CreateAuthentication(auth *model.AuthenticationCreateRequest) error {
	l.Log.Infof("Creating Authentication: %v", auth)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/authentications", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort,
	))

	body, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: sc.headers(),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create Authentication: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create Authentication: %v", string(b))
	}

	bytes, _ := io.ReadAll(resp.Body)
	var createdAuth model.AuthenticationResponse
	err = json.Unmarshal(bytes, &createdAuth)
	if err != nil {
		return err
	}

	l.Log.Infof("Creating ApplicationAuthentication for [%v:%v]", auth.ResourceIDRaw, createdAuth.ID)
	err = sc.createApplicationAuthentication(&model.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    auth.ResourceIDRaw,
		AuthenticationIDRaw: createdAuth.ID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *SourcesClient) PatchApplication(tenant, appID string, payload map[string]interface{}) error {
	l.Log.Infof("Patching Application %v with Data: %v", appID, payload)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/applications/%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, appID,
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to patch Application %v: %v", appID, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to patch Application %v: %v", appID, string(b))
	}

	return nil
}

func (sc *SourcesClient) PatchSource(tenant, sourceID string, payload map[string]interface{}) error {
	l.Log.Infof("Patching Source %v", sourceID)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"%v://%v:%v/api/sources/v3.1/sources/%v", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort, sourceID,
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to patch Source %v: %v", sourceID, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to patch Source %v: %v", sourceID, string(b))
	}

	return nil
}

// GetInternalAuthentication requests an authentication via the internal sources api
// that way we can expose the password.
// returns: populated sources api Authentication object, error
func (sc *SourcesClient) GetInternalAuthentication(authID string) (*model.AuthenticationInternalResponse, error) {
	l.Log.Infof("Requesting SuperKey Authentication: %v", authID)

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
		res, err = http.DefaultClient.Do(req)

		if err != nil || res.StatusCode == 200 {
			defer res.Body.Close()
			break
		} else {
			l.Log.Warnf("Authentication %v unavailable, retrying...", authID)
			time.Sleep(3 * time.Second)
		}
	}

	if err != nil || res.StatusCode != 200 {
		l.Log.Warnf("Error getting authentication: %v, tenant: %v, error: %v", authID, sc.AccountNumber, err)
		return nil, fmt.Errorf("failed to get Authentication %v", authID)
	}

	data, _ := io.ReadAll(res.Body)
	auth := model.AuthenticationInternalResponse{}

	// unmarshaling the data from the request, the id comes back as a string which fills `err`
	// we can safely ignore that as long as username/pass are there.
	err = json.Unmarshal(data, &auth)
	if err != nil && (auth.Username == "" || auth.Password == "") {
		l.Log.Warnf("Error unmarshaling authentication: %v, tenant: %v, error: %v", authID, sc.AccountNumber, err)
		return nil, err
	}

	l.Log.Infof("Authentication %v found!", authID)
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

func (sc *SourcesClient) createApplicationAuthentication(appAuth *model.ApplicationAuthenticationCreateRequest) error {
	l.Log.Infof("Creating ApplicationAuthentication: %v", appAuth)

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create Authentication: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create ApplicationAuthentication: %v", string(b))
	}

	return nil
}
