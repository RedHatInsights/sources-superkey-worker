package sources

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	sapi "github.com/lindgrenj6/sources-api-client-go"
	"github.com/redhatinsights/sources-superkey-worker/config"
)

var json = `{"identity": {"account_number": "$ACCT$", "user": {"is_org_admin": true}}}`

// NewAPIClient - returns a configured SourcesApiClient
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

// encodedIdentity - base64 decodes a x-rh-identity substituting the account number
// passed in
func encodedIdentity(acct string) string {
	encoded := bytes.NewBuffer([]byte(""))
	encoder := base64.NewEncoder(base64.StdEncoding, encoded)
	identity := strings.Replace(json, "$ACCT$", acct, 1)

	_, err := encoder.Write([]byte(identity))
	if err != nil {
		panic("Failed encoding json x-rh-identity")
	}

	_ = encoder.Close()
	return encoded.String()
}
