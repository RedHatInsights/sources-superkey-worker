package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	cost "github.com/aws/aws-sdk-go-v2/service/costandusagereportservice"
	costtypes "github.com/aws/aws-sdk-go-v2/service/costandusagereportservice/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
)

var CostS3Policy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "billingreports.amazonaws.com"
      },
      "Action": [
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy"
      ],
      "Resource": "arn:aws:s3:::S3BUCKET"
    },
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "billingreports.amazonaws.com"
      },
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::S3BUCKET/*"
    }
  ]
}`

// Client the amazon client object, holds credentials and API clients for each service necessary
// which are set when instantiated from the `NewClient` method.
type Client struct {
	AccessKey     string
	SecretKey     string
	Credentials   *aws.Config
	Iam           *iam.Client
	S3            *s3.Client
	CostReporting *cost.Client
}

// NewClient - takes a key+secret and list of API clients to set up
// returns: new AmazonClient and error
func NewClient(ctx context.Context, key, sec string, apis ...string) (*Client, error) {
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
		case "cost_report":
			if a.CostReporting == nil {
				a.CostReporting = cost.NewFromConfig(*creds)
			}
		default:
			l.LogWithContext(ctx).Errorf(`Unsupported "%s" API requested when creating an Amazon client`, api)
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
		case "cost_report":
			apis = append(apis, "cost_report")
		}
	}

	return apis
}

type CostReport struct {
	AdditionalArtifacts      []costtypes.AdditionalArtifact `json:"additional_artifacts"`
	AdditionalSchemaElements []costtypes.SchemaElement      `json:"additional_schema_elements"`
	Compression              costtypes.CompressionFormat    `json:"compression"`
	Format                   costtypes.ReportFormat         `json:"format"`
	TimeUnit                 costtypes.TimeUnit             `json:"time_unit"`
	ReportName               string                         `json:"report_name"`
	S3Prefix                 string                         `json:"s3_prefix"`
	S3Region                 costtypes.AWSRegion            `json:"s3_region"`
	S3Bucket                 string                         `json:"s3_bucket"`
}
