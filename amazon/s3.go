package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CreateS3Bucket - Creates an s3 bucket from name and config
// returns error if anything went wrong
func CreateS3Bucket(name string, cfg *aws.Config) error {
	s3client := s3.NewFromConfig(*cfg)

	_, err := s3client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: &name,
	})
	if err != nil {
		return err
	}

	return nil
}

// DestroyS3Bucket - Destroys an s3 bucket from name and config
// returns error if anything went wrong
func DestroyS3Bucket(name string, cfg *aws.Config) error {
	s3client := s3.NewFromConfig(*cfg)

	_, err := s3client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: &name,
	})

	if err != nil {
		return err
	}

	return nil
}
