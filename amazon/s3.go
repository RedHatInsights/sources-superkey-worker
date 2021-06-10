package amazon

import (
	"context"
	"fmt"
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
	// First we have to clear any objects that are in the bucket
	objects, err := a.S3.ListObjects(context.Background(), &s3.ListObjectsInput{
		Bucket: &name,
	})
	if err != nil {
		return fmt.Errorf("error during s3 bucket deletion during cleanup: %v", err)
	}

	for _, object := range objects.Contents {
		_, err = a.S3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: &name,
			Key:    object.Key,
		})
		if err != nil {
			return fmt.Errorf("error during s3 bucket deletion during cleanup: %v", err)
		}
	}

	_, err = a.S3.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: &name,
	})
	if err != nil {
		return fmt.Errorf("error during s3 bucket deletion: %v", err)
	}

	return nil
}

// PutBucketPolicy - attaches a policy to a bucket
// returns error
func (a *Client) AttachBucketPolicy(bucket, policy string) error {
	_, err := a.S3.PutBucketPolicy(context.Background(), &s3.PutBucketPolicyInput{
		Bucket: &bucket,
		Policy: &policy,
	})
	if err != nil {
		return err
	}

	return nil
}
