package sources

import (
	"encoding/base64"
	"encoding/json"

	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/sirupsen/logrus"
)

// encodeIdentity encodes the provided EBS account number and OrgIds into an XRHID identity structure.
func encodeIdentity(ebsAccountNumber string, orgId string) string {
	id := identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: ebsAccountNumber,
			OrgID:         orgId,
		},
	}

	xRhId, err := json.Marshal(id)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"struct_to_marshal": id}).Errorf(`failed encoding the identity structure: %s`, err)
		return ""
	}

	return base64.StdEncoding.EncodeToString(xRhId)
}
