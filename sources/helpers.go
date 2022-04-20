package sources

import (
	"bytes"
	"encoding/base64"
	"strings"
)

var xRhIdentity = `{"identity": {"account_number": "$ACCT$", "user": {"is_org_admin": true}}}`

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
