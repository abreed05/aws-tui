package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"

	kmsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/kms"
)

// KMSKeysHandler handles KMS Key resources
type KMSKeysHandler struct {
	BaseHandler
	client *kmsadapter.KeysClient
	region string
}

// NewKMSKeysHandler creates a new KMS keys handler
func NewKMSKeysHandler(kmsClient *kms.Client, region string) *KMSKeysHandler {
	return &KMSKeysHandler{
		client: kmsadapter.NewKeysClient(kmsClient),
		region: region,
	}
}

func (h *KMSKeysHandler) ResourceType() string { return "kms:keys" }
func (h *KMSKeysHandler) ResourceName() string { return "KMS Keys" }
func (h *KMSKeysHandler) ResourceIcon() string { return "🔑" }
func (h *KMSKeysHandler) ShortcutKey() string  { return "kms" }

func (h *KMSKeysHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Alias / ID", Width: 35, Sortable: true},
		{Title: "State", Width: 12, Sortable: true},
		{Title: "Usage", Width: 18, Sortable: false},
		{Title: "Origin", Width: 10, Sortable: false},
		{Title: "Created", Width: 12, Sortable: true},
		{Title: "Description", Width: 30, Sortable: false},
	}
}

func (h *KMSKeysHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	keys, err := h.client.ListKeys(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list KMS keys", err)
	}

	resources := make([]Resource, 0, len(keys))
	for _, key := range keys {
		// Skip AWS managed keys unless explicitly requested
		if !key.CustomerOwned && opts.Filter == "" {
			continue
		}

		resource := &KMSKeyResource{
			key:    key,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			alias := strings.ToLower(key.AliasName)
			id := strings.ToLower(key.KeyID)
			desc := strings.ToLower(key.Description)
			if !strings.Contains(alias, filter) && !strings.Contains(id, filter) && !strings.Contains(desc, filter) {
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

func (h *KMSKeysHandler) Get(ctx context.Context, id string) (Resource, error) {
	key, err := h.client.GetKey(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get KMS key %s", id), err)
	}

	return &KMSKeyResource{
		key:    *key,
		region: h.region,
	}, nil
}

func (h *KMSKeysHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	key, err := h.client.GetKey(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe KMS key %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Key"] = map[string]interface{}{
		"KeyId":       key.KeyID,
		"ARN":         key.KeyARN,
		"Alias":       key.AliasName,
		"Description": key.Description,
		"State":       key.KeyState,
		"Usage":       key.KeyUsage,
		"Spec":        key.KeySpec,
		"Origin":      key.Origin,
		"MultiRegion": key.MultiRegion,
		"Enabled":     key.Enabled,
		"CreatedAt":   key.CreationDate.Format(time.RFC3339),
	}

	// Try to get rotation status
	rotationEnabled, err := h.client.GetKeyRotationStatus(ctx, id)
	if err == nil {
		details["Rotation"] = map[string]interface{}{
			"Enabled": rotationEnabled,
		}
	}

	// Try to get key policy
	policy, err := h.client.GetKeyPolicy(ctx, id)
	if err == nil && policy != "" {
		// Parse and include the policy
		var policyDoc map[string]interface{}
		if json.Unmarshal([]byte(policy), &policyDoc) == nil {
			details["KeyPolicy"] = policyDoc
		} else {
			details["KeyPolicy"] = policy
		}
	}

	// Tags
	if len(key.Tags) > 0 {
		details["Tags"] = key.Tags
	}

	return details, nil
}

func (h *KMSKeysHandler) Actions() []Action {
	return []Action{
		{Key: "p", Name: "policy", Description: "View key policy"},
		{Key: "r", Name: "rotation", Description: "View rotation status"},
	}
}

// KMSKeyResource implements Resource interface for KMS keys
type KMSKeyResource struct {
	key    kmsadapter.Key
	region string
}

func (r *KMSKeyResource) GetID() string     { return r.key.KeyID }
func (r *KMSKeyResource) GetARN() string    { return r.key.KeyARN }
func (r *KMSKeyResource) GetType() string   { return "kms:keys" }
func (r *KMSKeyResource) GetRegion() string { return r.region }

func (r *KMSKeyResource) GetName() string {
	if r.key.AliasName != "" {
		return r.key.AliasName
	}
	return r.key.KeyID
}

func (r *KMSKeyResource) GetCreatedAt() time.Time {
	return r.key.CreationDate
}

func (r *KMSKeyResource) GetTags() map[string]string {
	return r.key.Tags
}

func (r *KMSKeyResource) ToTableRow() []string {
	displayName := r.key.KeyID
	if r.key.AliasName != "" {
		displayName = r.key.AliasName
	}

	created := ""
	if !r.key.CreationDate.IsZero() {
		created = r.key.CreationDate.Format("2006-01-02")
	}

	return []string{
		displayName,
		r.key.KeyState,
		r.key.KeyUsage,
		r.key.Origin,
		created,
		truncateString(r.key.Description, 30),
	}
}

func (r *KMSKeyResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"KeyId":       r.key.KeyID,
		"ARN":         r.key.KeyARN,
		"Alias":       r.key.AliasName,
		"Description": r.key.Description,
		"State":       r.key.KeyState,
		"Usage":       r.key.KeyUsage,
		"Spec":        r.key.KeySpec,
		"Origin":      r.key.Origin,
		"MultiRegion": r.key.MultiRegion,
		"Enabled":     r.key.Enabled,
	}
}
