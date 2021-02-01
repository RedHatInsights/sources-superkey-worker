package amazon

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

// Client the amazon client object, holds credentials and API clients for each service necessary
// which are set when instantiated from the `NewClient` method.
type Client struct {
	AccessKey   string
	SecretKey   string
	Credentials *aws.Config
	Iam         *iam.Client
	S3          *s3.Client
}

// NewClient - takes a key+secret and list of API clients to set up
// returns: new AmazonClient and error
func NewClient(key, sec string, apis ...string) (*Client, error) {
	a := Client{AccessKey: key, SecretKey: sec}

	creds, err := NewAmazonConfig(key, sec)
	if err != nil {
		return nil, err
	}

	a.Credentials = creds

	for _, api := range getRequiredApis(apis) {
		switch api {
		case "s3":
			if a.S3 == nil {
				a.S3 = s3.NewFromConfig(*creds)
			}
		case "iam":
			if a.Iam == nil {
				a.Iam = iam.NewFromConfig(*creds)
			}
		case "reporting":
			l.Log.Warn("Reporting not implemented yet")
		default:
			l.Log.Warnf("Unused api: %v", api)
		}

	}

	return &a, nil
}

func getRequiredApis(steps []string) []string {
	apis := make([]string, 0)
	for _, step := range steps {
		switch step {
		case "s3":
			apis = append(apis, "s3")
		case "role", "policy", "bind_role":
			apis = append(apis, "iam")
		}
	}

	return apis
}
