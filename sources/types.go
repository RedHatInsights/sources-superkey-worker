package sources

import (
	"encoding/base64"
	"encoding/json"
)

type XRhIdentity struct {
	Identity struct {
		AccountNumber string `json:"account_number"`
	} `json:"identity"`
}

func parseXRhIdentity(header string) (*XRhIdentity, error) {
	raw, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, err
	}

	xrhid := &XRhIdentity{}
	err = json.Unmarshal(raw, xrhid)
	if err != nil {
		return nil, err
	}

	return xrhid, nil
}

func getAccountNumber(raw string) (string, error) {
	xrhid, err := parseXRhIdentity(raw)
	if err != nil {
		return "", err
	}

	return xrhid.Identity.AccountNumber, nil
}
