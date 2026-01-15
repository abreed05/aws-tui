package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// BucketsClient wraps the S3 client
type BucketsClient struct {
	client *s3.Client
	region string
}

// NewBucketsClient creates a new S3 buckets client
func NewBucketsClient(client *s3.Client, region string) *BucketsClient {
	return &BucketsClient{client: client, region: region}
}

// Bucket represents an S3 bucket
type Bucket struct {
	Name             string
	CreationDate     time.Time
	Region           string
	Versioning       string
	Encryption       string
	PublicAccessBlock bool
	Tags             map[string]string
}

// ListBuckets lists all S3 buckets
func (c *BucketsClient) ListBuckets(ctx context.Context) ([]Bucket, error) {
	output, err := c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	buckets := make([]Bucket, 0, len(output.Buckets))
	for _, b := range output.Buckets {
		bucket := Bucket{
			Name:   aws.ToString(b.Name),
			Region: c.region, // Default to current region
			Tags:   make(map[string]string),
		}

		if b.CreationDate != nil {
			bucket.CreationDate = *b.CreationDate
		}

		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// GetBucket gets details for a single S3 bucket
func (c *BucketsClient) GetBucket(ctx context.Context, bucketName string) (*Bucket, error) {
	bucket := &Bucket{
		Name: bucketName,
		Tags: make(map[string]string),
	}

	// Get bucket location
	locOutput, err := c.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		loc := string(locOutput.LocationConstraint)
		if loc == "" {
			loc = "us-east-1" // Empty means us-east-1
		}
		bucket.Region = loc
	}

	// Get versioning status
	verOutput, err := c.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		bucket.Versioning = string(verOutput.Status)
		if bucket.Versioning == "" {
			bucket.Versioning = "Disabled"
		}
	}

	// Get encryption
	encOutput, err := c.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && encOutput.ServerSideEncryptionConfiguration != nil {
		for _, rule := range encOutput.ServerSideEncryptionConfiguration.Rules {
			if rule.ApplyServerSideEncryptionByDefault != nil {
				bucket.Encryption = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				break
			}
		}
	}
	if bucket.Encryption == "" {
		bucket.Encryption = "None"
	}

	// Get public access block
	pabOutput, err := c.client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && pabOutput.PublicAccessBlockConfiguration != nil {
		pab := pabOutput.PublicAccessBlockConfiguration
		bucket.PublicAccessBlock = (pab.BlockPublicAcls != nil && *pab.BlockPublicAcls) &&
			(pab.BlockPublicPolicy != nil && *pab.BlockPublicPolicy) &&
			(pab.IgnorePublicAcls != nil && *pab.IgnorePublicAcls) &&
			(pab.RestrictPublicBuckets != nil && *pab.RestrictPublicBuckets)
	}

	// Get tags
	tagsOutput, err := c.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		for _, tag := range tagsOutput.TagSet {
			bucket.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return bucket, nil
}

// GetBucketPolicy gets the bucket policy
func (c *BucketsClient) GetBucketPolicy(ctx context.Context, bucketName string) (string, error) {
	output, err := c.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(output.Policy), nil
}

// GetBucketLifecycle gets lifecycle rules
func (c *BucketsClient) GetBucketLifecycle(ctx context.Context, bucketName string) ([]types.LifecycleRule, error) {
	output, err := c.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, err
	}
	return output.Rules, nil
}
