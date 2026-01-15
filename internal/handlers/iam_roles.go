package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAMRolesHandler handles IAM Role resources
type IAMRolesHandler struct {
	BaseHandler
	client *iam.Client
}

// NewIAMRolesHandler creates a new IAM roles handler
func NewIAMRolesHandler(client *iam.Client) *IAMRolesHandler {
	return &IAMRolesHandler{client: client}
}

func (h *IAMRolesHandler) ResourceType() string { return "iam:roles" }
func (h *IAMRolesHandler) ResourceName() string { return "IAM Roles" }
func (h *IAMRolesHandler) ResourceIcon() string { return "🎭" }
func (h *IAMRolesHandler) ShortcutKey() string  { return "roles" }

func (h *IAMRolesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 35, Sortable: true},
		{Title: "Created", Width: 12, Sortable: true},
		{Title: "Last Used", Width: 12, Sortable: true},
		{Title: "Trust Policy", Width: 30, Sortable: false},
		{Title: "Description", Width: 40, Sortable: false},
	}
}

func (h *IAMRolesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	input := &iam.ListRolesInput{}

	if opts.PageSize > 0 {
		input.MaxItems = aws.Int32(int32(opts.PageSize))
	}
	if opts.NextToken != "" {
		input.Marker = aws.String(opts.NextToken)
	}

	result, err := h.client.ListRoles(ctx, input)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list IAM roles", err)
	}

	resources := make([]Resource, 0, len(result.Roles))
	for _, role := range result.Roles {
		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(aws.ToString(role.RoleName))
			if !strings.Contains(name, filter) {
				continue
			}
		}

		resources = append(resources, &IAMRoleResource{role: role})
	}

	nextToken := ""
	if result.Marker != nil {
		nextToken = aws.ToString(result.Marker)
	}

	return &ListResult{
		Resources: resources,
		NextToken: nextToken,
	}, nil
}

func (h *IAMRolesHandler) Get(ctx context.Context, id string) (Resource, error) {
	result, err := h.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get IAM role %s", id), err)
	}

	return &IAMRoleResource{role: *result.Role}, nil
}

func (h *IAMRolesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	// Get basic role info
	roleResult, err := h.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe IAM role %s", id), err)
	}

	role := roleResult.Role
	details := make(map[string]interface{})

	// Basic info
	details["Role"] = map[string]interface{}{
		"RoleName":                 aws.ToString(role.RoleName),
		"RoleId":                   aws.ToString(role.RoleId),
		"ARN":                      aws.ToString(role.Arn),
		"Path":                     aws.ToString(role.Path),
		"CreateDate":               role.CreateDate.Format(time.RFC3339),
		"Description":              aws.ToString(role.Description),
		"MaxSessionDuration":       aws.ToInt32(role.MaxSessionDuration),
		"PermissionsBoundary":      getPermissionsBoundary(role.PermissionsBoundary),
	}

	// Parse and format trust policy
	if role.AssumeRolePolicyDocument != nil {
		decoded, _ := url.QueryUnescape(aws.ToString(role.AssumeRolePolicyDocument))
		var trustPolicy map[string]interface{}
		if json.Unmarshal([]byte(decoded), &trustPolicy) == nil {
			details["TrustPolicy"] = trustPolicy
		} else {
			details["TrustPolicy"] = decoded
		}
	}

	// Role last used
	if role.RoleLastUsed != nil {
		lastUsed := map[string]string{}
		if role.RoleLastUsed.LastUsedDate != nil {
			lastUsed["LastUsedDate"] = role.RoleLastUsed.LastUsedDate.Format(time.RFC3339)
		}
		if role.RoleLastUsed.Region != nil {
			lastUsed["Region"] = aws.ToString(role.RoleLastUsed.Region)
		}
		details["LastUsed"] = lastUsed
	}

	// Get attached managed policies
	policiesResult, err := h.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(id),
	})
	if err == nil {
		policies := make([]map[string]string, 0, len(policiesResult.AttachedPolicies))
		for _, p := range policiesResult.AttachedPolicies {
			policies = append(policies, map[string]string{
				"PolicyName": aws.ToString(p.PolicyName),
				"PolicyArn":  aws.ToString(p.PolicyArn),
			})
		}
		details["AttachedPolicies"] = policies
	}

	// Get inline policies
	inlineResult, err := h.client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(id),
	})
	if err == nil {
		details["InlinePolicies"] = inlineResult.PolicyNames
	}

	// Get instance profiles
	profilesResult, err := h.client.ListInstanceProfilesForRole(ctx, &iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(id),
	})
	if err == nil {
		profiles := make([]string, 0, len(profilesResult.InstanceProfiles))
		for _, p := range profilesResult.InstanceProfiles {
			profiles = append(profiles, aws.ToString(p.InstanceProfileName))
		}
		details["InstanceProfiles"] = profiles
	}

	// Get tags
	tagsResult, err := h.client.ListRoleTags(ctx, &iam.ListRoleTagsInput{
		RoleName: aws.String(id),
	})
	if err == nil && len(tagsResult.Tags) > 0 {
		tags := make(map[string]string)
		for _, t := range tagsResult.Tags {
			tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
		}
		details["Tags"] = tags
	}

	return details, nil
}

func (h *IAMRolesHandler) Actions() []Action {
	return []Action{
		{Key: "p", Name: "policies", Description: "View attached policies"},
		{Key: "t", Name: "trust", Description: "View trust policy"},
		{Key: "i", Name: "instance-profiles", Description: "View instance profiles"},
	}
}

// IAMRoleResource implements Resource interface for IAM roles
type IAMRoleResource struct {
	role types.Role
}

func (r *IAMRoleResource) GetID() string   { return aws.ToString(r.role.RoleName) }
func (r *IAMRoleResource) GetARN() string  { return aws.ToString(r.role.Arn) }
func (r *IAMRoleResource) GetName() string { return aws.ToString(r.role.RoleName) }
func (r *IAMRoleResource) GetType() string { return "iam:roles" }
func (r *IAMRoleResource) GetRegion() string { return "global" }

func (r *IAMRoleResource) GetCreatedAt() time.Time {
	if r.role.CreateDate != nil {
		return *r.role.CreateDate
	}
	return time.Time{}
}

func (r *IAMRoleResource) GetTags() map[string]string {
	return nil
}

func (r *IAMRoleResource) ToTableRow() []string {
	created := ""
	if r.role.CreateDate != nil {
		created = r.role.CreateDate.Format("2006-01-02")
	}

	lastUsed := "Never"
	if r.role.RoleLastUsed != nil && r.role.RoleLastUsed.LastUsedDate != nil {
		lastUsed = r.role.RoleLastUsed.LastUsedDate.Format("2006-01-02")
	}

	trustPrincipal := extractTrustPrincipal(r.role.AssumeRolePolicyDocument)
	description := aws.ToString(r.role.Description)
	if len(description) > 40 {
		description = description[:37] + "..."
	}

	return []string{
		aws.ToString(r.role.RoleName),
		created,
		lastUsed,
		trustPrincipal,
		description,
	}
}

func (r *IAMRoleResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"RoleName":    aws.ToString(r.role.RoleName),
		"RoleId":      aws.ToString(r.role.RoleId),
		"ARN":         aws.ToString(r.role.Arn),
		"Path":        aws.ToString(r.role.Path),
		"CreateDate":  formatTime(r.role.CreateDate),
		"Description": aws.ToString(r.role.Description),
	}
}

// Helper to extract trust principal from policy document
func extractTrustPrincipal(policyDoc *string) string {
	if policyDoc == nil {
		return "Unknown"
	}

	decoded, err := url.QueryUnescape(aws.ToString(policyDoc))
	if err != nil {
		return "Unknown"
	}

	var policy struct {
		Statement []struct {
			Principal interface{} `json:"Principal"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal([]byte(decoded), &policy); err != nil {
		return "Unknown"
	}

	if len(policy.Statement) == 0 {
		return "Unknown"
	}

	principal := policy.Statement[0].Principal
	switch p := principal.(type) {
	case string:
		return p
	case map[string]interface{}:
		if service, ok := p["Service"]; ok {
			switch s := service.(type) {
			case string:
				return s
			case []interface{}:
				if len(s) > 0 {
					return fmt.Sprintf("%v", s[0])
				}
			}
		}
		if aws, ok := p["AWS"]; ok {
			switch a := aws.(type) {
			case string:
				// Truncate ARNs
				if strings.Contains(a, ":") {
					parts := strings.Split(a, ":")
					if len(parts) > 0 {
						return parts[len(parts)-1]
					}
				}
				return a
			case []interface{}:
				if len(a) > 0 {
					return fmt.Sprintf("%v (+%d)", a[0], len(a)-1)
				}
			}
		}
		if federated, ok := p["Federated"]; ok {
			return fmt.Sprintf("Federated: %v", federated)
		}
	}

	return "Unknown"
}

func getPermissionsBoundary(pb *types.AttachedPermissionsBoundary) string {
	if pb == nil || pb.PermissionsBoundaryArn == nil {
		return "None"
	}
	arn := aws.ToString(pb.PermissionsBoundaryArn)
	// Extract just the policy name from ARN
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}
