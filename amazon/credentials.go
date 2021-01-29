package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// NewConfig - returns an aws config struct with access key + secret + region set
func NewConfig(key, sec string) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		// Hard coded credentials.
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     key,
				SecretAccessKey: sec,
				Source:          "SourcesSuperKeyWorker",
			},
		}))

	if err != nil {
		return nil, err
	}

	// defaulting for now. maybe we'll support setting it eventually.
	cfg.Region = "us-east-1"

	return &cfg, nil
}
