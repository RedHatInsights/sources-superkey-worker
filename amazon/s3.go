package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CreateS3Bucket - Creates an s3 bucket from name and config
// returns error if anything went wrong
func (a *Client) CreateS3Bucket(name string) error {
	_, err := a.S3.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: &name,
	})
	if err != nil {
		return err
	}

	return nil
}

// DestroyS3Bucket - Destroys an s3 bucket from name and config
// returns error if anything went wrong
func (a *Client) DestroyS3Bucket(name string) error {
	_, err := a.S3.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: &name,
	})

	if err != nil {
		return err
	}

	return nil
}
