package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	s3adapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/s3"
)

// S3BucketsHandler handles S3 Bucket resources
type S3BucketsHandler struct {
	BaseHandler
	client *s3adapter.BucketsClient
	region string
}

// NewS3BucketsHandler creates a new S3 buckets handler
func NewS3BucketsHandler(s3Client *s3.Client, region string) *S3BucketsHandler {
	return &S3BucketsHandler{
		client: s3adapter.NewBucketsClient(s3Client, region),
		region: region,
	}
}

func (h *S3BucketsHandler) ResourceType() string { return "s3:buckets" }
func (h *S3BucketsHandler) ResourceName() string { return "S3 Buckets" }
func (h *S3BucketsHandler) ResourceIcon() string { return "🪣" }
func (h *S3BucketsHandler) ShortcutKey() string  { return "s3" }

func (h *S3BucketsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Bucket Name", Width: 45, Sortable: true},
		{Title: "Region", Width: 15, Sortable: true},
		{Title: "Created", Width: 12, Sortable: true},
	}
}

func (h *S3BucketsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	buckets, err := h.client.ListBuckets(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list S3 buckets", err)
	}

	resources := make([]Resource, 0, len(buckets))
	for _, bucket := range buckets {
		resource := &S3BucketResource{
			bucket: bucket,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(bucket.Name)
			if !strings.Contains(name, filter) {
				continue
			}
		}

		resources = append(resources, resource)
	}

	return &ListResult{
		Resources: resources,
		NextToken: "",
	}, nil
}

func (h *S3BucketsHandler) Get(ctx context.Context, id string) (Resource, error) {
	bucket, err := h.client.GetBucket(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get S3 bucket %s", id), err)
	}

	return &S3BucketResource{
		bucket: *bucket,
		region: bucket.Region,
	}, nil
}

func (h *S3BucketsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	bucket, err := h.client.GetBucket(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe S3 bucket %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Bucket"] = map[string]interface{}{
		"Name":   bucket.Name,
		"Region": bucket.Region,
	}

	if !bucket.CreationDate.IsZero() {
		details["Bucket"].(map[string]interface{})["CreationDate"] = bucket.CreationDate.Format(time.RFC3339)
	}

	// Security settings
	publicBlocked := "No"
	if bucket.PublicAccessBlock {
		publicBlocked = "Yes"
	}
	details["Security"] = map[string]interface{}{
		"Versioning":         bucket.Versioning,
		"Encryption":         bucket.Encryption,
		"PublicAccessBlocked": publicBlocked,
	}

	// Get bucket policy if exists
	policy, err := h.client.GetBucketPolicy(ctx, id)
	if err == nil && policy != "" {
		var policyDoc map[string]interface{}
		if json.Unmarshal([]byte(policy), &policyDoc) == nil {
			details["BucketPolicy"] = policyDoc
		}
	}

	// Get lifecycle rules if exist
	lifecycle, err := h.client.GetBucketLifecycle(ctx, id)
	if err == nil && len(lifecycle) > 0 {
		rules := make([]map[string]interface{}, 0, len(lifecycle))
		for _, rule := range lifecycle {
			r := map[string]interface{}{
				"ID":     *rule.ID,
				"Status": string(rule.Status),
			}
			if rule.Filter != nil {
				// Simplified - just show that there's a filter
				r["HasFilter"] = true
			}
			rules = append(rules, r)
		}
		details["LifecycleRules"] = rules
	}

	// Tags
	if len(bucket.Tags) > 0 {
		details["Tags"] = bucket.Tags
	}

	return details, nil
}

func (h *S3BucketsHandler) Actions() []Action {
	return []Action{
		{Key: "p", Name: "policy", Description: "View bucket policy"},
	}
}

func (h *S3BucketsHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	switch action {
	case "policy":
		return &ViewBucketPolicyAction{
			BucketName: resourceID,
		}
	default:
		return ErrNotSupported
	}
}

// GetBucketPolicyForView retrieves bucket policy for viewing
func (h *S3BucketsHandler) GetBucketPolicyForView(ctx context.Context, bucketName string) (interface{}, error) {
	policy, err := h.client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket policy: %w", err)
	}

	if policy == "" {
		return map[string]string{"message": "No bucket policy configured"}, nil
	}

	// Parse JSON policy
	var policyDoc map[string]interface{}
	if err := json.Unmarshal([]byte(policy), &policyDoc); err != nil {
		// Return as string if not valid JSON
		return map[string]string{"policy": policy}, nil
	}

	return policyDoc, nil
}

// ViewBucketPolicyAction triggers viewing bucket policy
type ViewBucketPolicyAction struct {
	BucketName string
}

func (a *ViewBucketPolicyAction) Error() string {
	return fmt.Sprintf("view policy for bucket %s", a.BucketName)
}

func (a *ViewBucketPolicyAction) IsActionMsg() {}

// S3BucketResource implements Resource interface for S3 buckets
type S3BucketResource struct {
	bucket s3adapter.Bucket
	region string
}

func (r *S3BucketResource) GetID() string     { return r.bucket.Name }
func (r *S3BucketResource) GetName() string   { return r.bucket.Name }
func (r *S3BucketResource) GetARN() string    { return fmt.Sprintf("arn:aws:s3:::%s", r.bucket.Name) }
func (r *S3BucketResource) GetType() string   { return "s3:buckets" }
func (r *S3BucketResource) GetRegion() string { return r.bucket.Region }

func (r *S3BucketResource) GetCreatedAt() time.Time {
	return r.bucket.CreationDate
}

func (r *S3BucketResource) GetTags() map[string]string {
	return r.bucket.Tags
}

func (r *S3BucketResource) ToTableRow() []string {
	created := "-"
	if !r.bucket.CreationDate.IsZero() {
		created = r.bucket.CreationDate.Format("2006-01-02")
	}

	region := r.bucket.Region
	if region == "" {
		region = "-"
	}

	return []string{
		r.bucket.Name,
		region,
		created,
	}
}

func (r *S3BucketResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"Name":              r.bucket.Name,
		"Region":            r.bucket.Region,
		"Versioning":        r.bucket.Versioning,
		"Encryption":        r.bucket.Encryption,
		"PublicAccessBlock": r.bucket.PublicAccessBlock,
	}
}
