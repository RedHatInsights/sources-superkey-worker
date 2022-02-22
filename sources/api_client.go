package sources

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

var xRhIdentity = `{"identity": {"account_number": "$ACCT$", "user": {"is_org_admin": true}}}`
var conf = config.Get()

func CheckAvailability(tenant string, sourceID string) error {
	l.Log.Infof("Checking Availability for Source ID: %v", sourceID)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/api/sources/v3.1/sources/%v", conf.SourcesHost, conf.SourcesPort, sourceID,
	))

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: headers(tenant),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 202 {
		return fmt.Errorf("failed to check availability for source %v: %v", sourceID, err)
	}
	defer resp.Body.Close()

	return nil
}

func CreateAuthentication(tenant string, auth *model.AuthenticationCreateRequest) error {
	l.Log.Infof("Creating Authentication: %v", auth)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/api/sources/v3.1/authentications", conf.SourcesHost, conf.SourcesPort,
	))

	body, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Header: headers(tenant),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create Authentication: %v", err)
	} else if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("failed to create Authentication: %v", string(b))
	}
	defer resp.Body.Close()

	return nil
}

func PatchApplication(tenant, appID string, payload map[string]interface{}) error {
	l.Log.Infof("Patching Application %v with Data: %v", appID, payload)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/api/sources/v3.1/applications/%v",
		conf.SourcesHost,
		conf.SourcesPort,
		appID,
	))

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPatch,
		URL:    reqURL,
		Header: headers(tenant),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode > 299 {
		return fmt.Errorf("failed to patch Application %v: %v", appID, err)
	}
	defer resp.Body.Close()

	return nil
}

func PatchSource(tenant, sourceID string, payload map[string]interface{}) error {
	l.Log.Infof("Patching Source %v", sourceID)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/api/sources/v3.1/sources/%v",
		conf.SourcesHost,
		conf.SourcesPort,
		sourceID,
	))

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: http.MethodPatch,
		URL:    reqURL,
		Header: headers(tenant),
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode > 299 {
		return fmt.Errorf("failed to patch Source %v: %v", sourceID, err)
	}
	defer resp.Body.Close()

	return nil
}

// GetInternalAuthentication requests an authentication via the internal sources api
// that way we can expose the password.
// returns: populated sources api Authentication object, error
func GetInternalAuthentication(tenant, authID string) (*model.AuthenticationInternalResponse, error) {
	l.Log.Infof("Requesting SuperKey Authentication: %v", authID)

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/internal/v2.0/authentications/%v?expose_encrypted_attribute[]=password",
		conf.SourcesHost,
		conf.SourcesPort,
		authID,
	))

	req := &http.Request{
		Method: http.MethodGet,
		URL:    reqURL,
		Header: headers(tenant),
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
		l.Log.Warnf("Error getting authentication: %v, tenant: %v, error: %v", authID, tenant, err)
		return nil, fmt.Errorf("failed to get Authentication %v", authID)
	}

	data, _ := io.ReadAll(res.Body)
	auth := model.AuthenticationInternalResponse{}

	// unmarshaling the data from the request, the id comes back as a string which fills `err`
	// we can safely ignore that as long as username/pass are there.
	err = json.Unmarshal(data, &auth)
	if err != nil && (auth.Username == "" || auth.Password == "") {
		l.Log.Warnf("Error unmarshaling authentication: %v, tenant: %v, error: %v", authID, tenant, err)
		return nil, err
	}

	l.Log.Infof("Authentication %v found!", authID)
	return &auth, nil
}

// TODO: This will be removed when the PSK switchover is done.
// encodedIdentity - base64 decodes a x-rh-identity substituting the account number
// passed in
// returns: base64 x-rh-id string
func encodedIdentity(acct string) string {
	encoded := bytes.NewBuffer([]byte(""))
	encoder := base64.NewEncoder(base64.StdEncoding, encoded)
	identity := strings.Replace(xRhIdentity, "$ACCT$", acct, 1)

	_, err := encoder.Write([]byte(identity))
	if err != nil {
		panic("Failed encoding json x-rh-identity")
	}

	_ = encoder.Close()
	return encoded.String()
}

func headers(tenant string) map[string][]string {
	if conf.SourcesPSK == "" {
		return map[string][]string{"x-rh-identity": {encodedIdentity(tenant)}}
	} else {
		return map[string][]string{
			"x-rh-sources-psk":            {conf.SourcesPSK},
			"x-rh-sources-account-number": {tenant},
		}
	}
}
