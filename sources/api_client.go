package sources

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	sapi "github.com/lindgrenj6/sources-api-client-go"
	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

var xRhIdentity = `{"identity": {"account_number": "$ACCT$", "user": {"is_org_admin": true}}}`

// NewAPIClient - creates a sources api client with default header for account
// returns: Sources API Client
func NewAPIClient(acct string) *sapi.APIClient {
	conf := config.Get()

	return sapi.NewAPIClient(&sapi.Configuration{
		DefaultHeader: map[string]string{"x-rh-identity": encodedIdentity(acct)},
		Servers: []sapi.ServerConfiguration{
			{
				URL: fmt.Sprintf("%s://%s:%d/api/sources/v3.1", conf.SourcesScheme, conf.SourcesHost, conf.SourcesPort),
			},
		},
	})
}

// GetInternalAuthentication requests an authentication via the internal sources api
// that way we can expose the password.
// returns: populated sources api Authentication object, error
func GetInternalAuthentication(tenant, authID string) (*sapi.Authentication, error) {
	conf := config.Get()

	reqURL, _ := url.Parse(fmt.Sprintf(
		"http://%v:%v/internal/v1.0/authentications/%v?expose_encrypted_attribute[]=password",
		conf.SourcesHost,
		conf.SourcesPort,
		authID,
	))

	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: map[string][]string{
			"x-rh-identity": {encodedIdentity(tenant)},
		},
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		l.Log.Warnf("Error getting authentication: %v, tenant: %v, error: %v", authID, tenant, err)
		return nil, err
	}
	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	auth := sapi.Authentication{}

	// unmarshaling the data from the request, the id comes back as a string which fills `err`
	// we can safely ignore that as long as username/pass are there.
	err = json.Unmarshal(data, &auth)
	if err != nil && (auth.Username == nil || auth.Password == nil) {
		l.Log.Warnf("Error unmarshaling authentication: %v, tenant: %v, error: %v", authID, tenant, err)
		return nil, err
	}

	return &auth, nil
}

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
