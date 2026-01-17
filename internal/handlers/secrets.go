package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	smadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/secretsmanager"
)

// SecretsHandler handles Secrets Manager resources
type SecretsHandler struct {
	BaseHandler
	client *smadapter.SecretsClient
	region string
}

// NewSecretsHandler creates a new secrets handler
func NewSecretsHandler(smClient *secretsmanager.Client, region string) *SecretsHandler {
	return &SecretsHandler{
		client: smadapter.NewSecretsClient(smClient),
		region: region,
	}
}

func (h *SecretsHandler) ResourceType() string { return "secretsmanager:secrets" }
func (h *SecretsHandler) ResourceName() string { return "Secrets" }
func (h *SecretsHandler) ResourceIcon() string { return "🔐" }
func (h *SecretsHandler) ShortcutKey() string  { return "secrets" }

func (h *SecretsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 40, Sortable: true},
		{Title: "Rotation", Width: 10, Sortable: false},
		{Title: "Last Changed", Width: 14, Sortable: true},
		{Title: "Last Accessed", Width: 14, Sortable: true},
		{Title: "Description", Width: 35, Sortable: false},
	}
}

func (h *SecretsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	secrets, err := h.client.ListSecrets(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list secrets", err)
	}

	resources := make([]Resource, 0, len(secrets))
	for _, secret := range secrets {
		resource := &SecretResource{
			secret: secret,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(secret.Name)
			desc := strings.ToLower(secret.Description)
			if !strings.Contains(name, filter) && !strings.Contains(desc, filter) {
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

func (h *SecretsHandler) Get(ctx context.Context, id string) (Resource, error) {
	secret, err := h.client.GetSecret(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get secret %s", id), err)
	}

	return &SecretResource{
		secret: *secret,
		region: h.region,
	}, nil
}

func (h *SecretsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	secret, err := h.client.GetSecret(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe secret %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	secretInfo := map[string]interface{}{
		"Name":            secret.Name,
		"ARN":             secret.ARN,
		"Description":     secret.Description,
		"RotationEnabled": secret.RotationEnabled,
	}

	if secret.KmsKeyID != "" {
		secretInfo["KmsKeyId"] = secret.KmsKeyID
	}
	if secret.OwningService != "" {
		secretInfo["OwningService"] = secret.OwningService
	}
	if secret.PrimaryRegion != "" {
		secretInfo["PrimaryRegion"] = secret.PrimaryRegion
	}

	details["Secret"] = secretInfo

	// Dates
	dates := make(map[string]string)
	if !secret.LastChangedDate.IsZero() {
		dates["LastChanged"] = secret.LastChangedDate.Format(time.RFC3339)
	}
	if !secret.LastAccessedDate.IsZero() {
		dates["LastAccessed"] = secret.LastAccessedDate.Format(time.RFC3339)
	}
	if !secret.LastRotatedDate.IsZero() {
		dates["LastRotated"] = secret.LastRotatedDate.Format(time.RFC3339)
	}
	if !secret.DeletedDate.IsZero() {
		dates["DeletedDate"] = secret.DeletedDate.Format(time.RFC3339)
	}
	if len(dates) > 0 {
		details["Dates"] = dates
	}

	// Rotation info
	if secret.RotationEnabled {
		rotation := map[string]interface{}{
			"Enabled": true,
		}
		if secret.RotationLambdaARN != "" {
			rotation["LambdaARN"] = secret.RotationLambdaARN
		}
		details["Rotation"] = rotation
	}

	// Get resource policy if exists
	policy, err := h.client.GetSecretResourcePolicy(ctx, id)
	if err == nil && policy != "" {
		var policyDoc map[string]interface{}
		if json.Unmarshal([]byte(policy), &policyDoc) == nil {
			details["ResourcePolicy"] = policyDoc
		} else {
			details["ResourcePolicy"] = policy
		}
	}

	// Get version IDs
	versionIDs, err := h.client.GetSecretVersionIDs(ctx, id)
	if err == nil && len(versionIDs) > 0 {
		details["Versions"] = versionIDs
	}

	// Tags
	if len(secret.Tags) > 0 {
		details["Tags"] = secret.Tags
	}

	// NOTE: We intentionally do NOT retrieve the secret value for security
	details["_Note"] = "Secret value is not displayed for security reasons. Use AWS CLI or Console to view."

	return details, nil
}

func (h *SecretsHandler) Actions() []Action {
	return []Action{
		{Key: "v", Name: "view", Description: "View secret value"},
		{Key: "e", Name: "edit", Description: "Edit secret value"},
		{Key: "c", Name: "create", Description: "Create new secret"},
		{Key: "x", Name: "delete", Description: "Delete secret"},
		{Key: "r", Name: "rotation", Description: "View rotation configuration"},
	}
}

func (h *SecretsHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	switch action {
	case "view":
		return &ViewSecretAction{
			SecretID:   resourceID,
			SecretName: resourceID,
		}
	case "edit":
		return &EditSecretAction{
			SecretID:   resourceID,
			SecretName: resourceID,
		}
	case "create":
		return &CreateSecretAction{}
	case "delete":
		return &DeleteSecretAction{
			SecretID:   resourceID,
			SecretName: resourceID,
		}
	default:
		return ErrNotSupported
	}
}

func (h *SecretsHandler) CanEdit() bool {
	return true
}

func (h *SecretsHandler) CanDelete() bool {
	return true
}

func (h *SecretsHandler) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// Extract secret value from updates
	secretValue, ok := updates["SecretValue"].(string)
	if !ok {
		return fmt.Errorf("invalid secret value")
	}

	return h.client.UpdateSecretValue(ctx, id, secretValue)
}

func (h *SecretsHandler) Create(ctx context.Context, params map[string]interface{}) (Resource, error) {
	// Extract parameters
	name, ok := params["Name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	value, ok := params["Value"].(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("value is required")
	}

	description, _ := params["Description"].(string)

	tags := make(map[string]string)
	if tagsInterface, ok := params["Tags"].(map[string]string); ok {
		tags = tagsInterface
	}

	createParams := smadapter.CreateSecretParams{
		Name:        name,
		SecretValue: value,
		Description: description,
		Tags:        tags,
	}

	secret, err := h.client.CreateSecret(ctx, createParams)
	if err != nil {
		return nil, NewHandlerError("CREATE_FAILED", "failed to create secret", err)
	}

	return &SecretResource{
		secret: *secret,
		region: h.region,
	}, nil
}

func (h *SecretsHandler) Delete(ctx context.Context, id string) error {
	// Default recovery window
	return h.DeleteWithRecoveryWindow(ctx, id, 30)
}

// DeleteWithRecoveryWindow allows specifying recovery window
func (h *SecretsHandler) DeleteWithRecoveryWindow(ctx context.Context, id string, recoveryWindowDays int) error {
	err := h.client.DeleteSecret(ctx, id, int32(recoveryWindowDays))
	if err != nil {
		return NewHandlerError("DELETE_FAILED", fmt.Sprintf("failed to delete secret %s", id), err)
	}
	return nil
}

// GetSecretValueForView retrieves secret value for viewing
func (h *SecretsHandler) GetSecretValueForView(ctx context.Context, secretID string) (string, error) {
	return h.client.GetSecretValue(ctx, secretID)
}

// GetSecretValueForEdit retrieves secret value for editing
func (h *SecretsHandler) GetSecretValueForEdit(ctx context.Context, secretID string) (string, error) {
	return h.client.GetSecretValue(ctx, secretID)
}

// SecretResource implements Resource interface for Secrets Manager secrets
type SecretResource struct {
	secret smadapter.Secret
	region string
}

func (r *SecretResource) GetID() string     { return r.secret.Name }
func (r *SecretResource) GetARN() string    { return r.secret.ARN }
func (r *SecretResource) GetName() string   { return r.secret.Name }
func (r *SecretResource) GetType() string   { return "secretsmanager:secrets" }
func (r *SecretResource) GetRegion() string { return r.region }

func (r *SecretResource) GetCreatedAt() time.Time {
	// Secrets don't have a creation timestamp, use last changed as proxy
	return r.secret.LastChangedDate
}

func (r *SecretResource) GetTags() map[string]string {
	return r.secret.Tags
}

func (r *SecretResource) ToTableRow() []string {
	rotation := "No"
	if r.secret.RotationEnabled {
		rotation = "Yes"
	}

	lastChanged := ""
	if !r.secret.LastChangedDate.IsZero() {
		lastChanged = r.secret.LastChangedDate.Format("2006-01-02")
	}

	lastAccessed := ""
	if !r.secret.LastAccessedDate.IsZero() {
		lastAccessed = r.secret.LastAccessedDate.Format("2006-01-02")
	}

	return []string{
		r.secret.Name,
		rotation,
		lastChanged,
		lastAccessed,
		truncateString(r.secret.Description, 35),
	}
}

func (r *SecretResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"Name":            r.secret.Name,
		"ARN":             r.secret.ARN,
		"Description":     r.secret.Description,
		"KmsKeyId":        r.secret.KmsKeyID,
		"RotationEnabled": r.secret.RotationEnabled,
		"OwningService":   r.secret.OwningService,
	}
}

// ViewSecretAction is returned by ExecuteAction to trigger viewing a secret
type ViewSecretAction struct {
	SecretID   string
	SecretName string
}

func (a *ViewSecretAction) Error() string {
	return fmt.Sprintf("view secret %s", a.SecretName)
}

func (a *ViewSecretAction) IsActionMsg() {}

// EditSecretAction is returned by ExecuteAction to trigger editing a secret
type EditSecretAction struct {
	SecretID   string
	SecretName string
}

func (a *EditSecretAction) Error() string {
	return fmt.Sprintf("edit secret %s", a.SecretName)
}

func (a *EditSecretAction) IsActionMsg() {}

// CreateSecretAction triggers the secret creation form
type CreateSecretAction struct{}

func (a *CreateSecretAction) Error() string {
	return "create secret"
}

func (a *CreateSecretAction) IsActionMsg() {}

// DeleteSecretAction triggers the delete confirmation
type DeleteSecretAction struct {
	SecretID   string
	SecretName string
}

func (a *DeleteSecretAction) Error() string {
	return fmt.Sprintf("delete secret %s", a.SecretName)
}

func (a *DeleteSecretAction) IsActionMsg() {}
