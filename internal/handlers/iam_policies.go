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

// IAMPoliciesHandler handles IAM Policy resources
type IAMPoliciesHandler struct {
	BaseHandler
	client *iam.Client
}

// NewIAMPoliciesHandler creates a new IAM policies handler
func NewIAMPoliciesHandler(client *iam.Client) *IAMPoliciesHandler {
	return &IAMPoliciesHandler{client: client}
}

func (h *IAMPoliciesHandler) ResourceType() string { return "iam:policies" }
func (h *IAMPoliciesHandler) ResourceName() string { return "IAM Policies" }
func (h *IAMPoliciesHandler) ResourceIcon() string { return "📜" }
func (h *IAMPoliciesHandler) ShortcutKey() string  { return "policies" }

func (h *IAMPoliciesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 40, Sortable: true},
		{Title: "Type", Width: 12, Sortable: true},
		{Title: "Attached", Width: 10, Sortable: true},
		{Title: "Created", Width: 12, Sortable: true},
		{Title: "Description", Width: 50, Sortable: false},
	}
}

func (h *IAMPoliciesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	input := &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeLocal, // Only customer managed by default
	}

	if opts.PageSize > 0 {
		input.MaxItems = aws.Int32(int32(opts.PageSize))
	}
	if opts.NextToken != "" {
		input.Marker = aws.String(opts.NextToken)
	}

	result, err := h.client.ListPolicies(ctx, input)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list IAM policies", err)
	}

	resources := make([]Resource, 0, len(result.Policies))
	for _, policy := range result.Policies {
		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(aws.ToString(policy.PolicyName))
			if !strings.Contains(name, filter) {
				continue
			}
		}

		policyType := "Customer"
		if strings.HasPrefix(aws.ToString(policy.Arn), "arn:aws:iam::aws:") {
			policyType = "AWS"
		}

		resources = append(resources, &IAMPolicyResource{
			policy:     policy,
			policyType: policyType,
		})
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

func (h *IAMPoliciesHandler) Get(ctx context.Context, id string) (Resource, error) {
	result, err := h.client.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get IAM policy %s", id), err)
	}

	policyType := "Customer"
	if strings.HasPrefix(aws.ToString(result.Policy.Arn), "arn:aws:iam::aws:") {
		policyType = "AWS"
	}

	return &IAMPolicyResource{
		policy:     *result.Policy,
		policyType: policyType,
	}, nil
}

func (h *IAMPoliciesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	// Get policy info
	policyResult, err := h.client.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe IAM policy %s", id), err)
	}

	policy := policyResult.Policy
	details := make(map[string]interface{})

	policyType := "Customer Managed"
	if strings.HasPrefix(aws.ToString(policy.Arn), "arn:aws:iam::aws:") {
		policyType = "AWS Managed"
	}

	// Basic info
	details["Policy"] = map[string]interface{}{
		"PolicyName":       aws.ToString(policy.PolicyName),
		"PolicyId":         aws.ToString(policy.PolicyId),
		"ARN":              aws.ToString(policy.Arn),
		"Path":             aws.ToString(policy.Path),
		"Type":             policyType,
		"DefaultVersionId": aws.ToString(policy.DefaultVersionId),
		"AttachmentCount":  aws.ToInt32(policy.AttachmentCount),
		"CreateDate":       policy.CreateDate.Format(time.RFC3339),
		"UpdateDate":       formatTime(policy.UpdateDate),
		"Description":      aws.ToString(policy.Description),
	}

	// Get policy document (default version)
	if policy.DefaultVersionId != nil {
		versionResult, err := h.client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
			PolicyArn: aws.String(id),
			VersionId: policy.DefaultVersionId,
		})
		if err == nil && versionResult.PolicyVersion != nil {
			decoded, _ := url.QueryUnescape(aws.ToString(versionResult.PolicyVersion.Document))
			var policyDoc map[string]interface{}
			if json.Unmarshal([]byte(decoded), &policyDoc) == nil {
				details["PolicyDocument"] = policyDoc
			} else {
				details["PolicyDocument"] = decoded
			}
		}
	}

	// Get entities this policy is attached to
	entitiesResult, err := h.client.ListEntitiesForPolicy(ctx, &iam.ListEntitiesForPolicyInput{
		PolicyArn: aws.String(id),
	})
	if err == nil {
		if len(entitiesResult.PolicyUsers) > 0 {
			users := make([]string, 0, len(entitiesResult.PolicyUsers))
			for _, u := range entitiesResult.PolicyUsers {
				users = append(users, aws.ToString(u.UserName))
			}
			details["AttachedUsers"] = users
		}

		if len(entitiesResult.PolicyGroups) > 0 {
			groups := make([]string, 0, len(entitiesResult.PolicyGroups))
			for _, g := range entitiesResult.PolicyGroups {
				groups = append(groups, aws.ToString(g.GroupName))
			}
			details["AttachedGroups"] = groups
		}

		if len(entitiesResult.PolicyRoles) > 0 {
			roles := make([]string, 0, len(entitiesResult.PolicyRoles))
			for _, r := range entitiesResult.PolicyRoles {
				roles = append(roles, aws.ToString(r.RoleName))
			}
			details["AttachedRoles"] = roles
		}
	}

	// Get policy versions
	versionsResult, err := h.client.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(id),
	})
	if err == nil {
		versions := make([]map[string]interface{}, 0, len(versionsResult.Versions))
		for _, v := range versionsResult.Versions {
			versions = append(versions, map[string]interface{}{
				"VersionId":        aws.ToString(v.VersionId),
				"IsDefaultVersion": v.IsDefaultVersion,
				"CreateDate":       v.CreateDate.Format(time.RFC3339),
			})
		}
		details["Versions"] = versions
	}

	// Get tags (only for customer managed policies)
	if !strings.HasPrefix(id, "arn:aws:iam::aws:") {
		tagsResult, err := h.client.ListPolicyTags(ctx, &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(id),
		})
		if err == nil && len(tagsResult.Tags) > 0 {
			tags := make(map[string]string)
			for _, t := range tagsResult.Tags {
				tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}
			details["Tags"] = tags
		}
	}

	return details, nil
}

func (h *IAMPoliciesHandler) Actions() []Action {
	return []Action{
		{Key: "v", Name: "view-document", Description: "View policy document"},
		{Key: "e", Name: "entities", Description: "View attached entities"},
		{Key: "V", Name: "versions", Description: "View policy versions"},
	}
}

// IAMPolicyResource implements Resource interface for IAM policies
type IAMPolicyResource struct {
	policy     types.Policy
	policyType string
}

func (r *IAMPolicyResource) GetID() string   { return aws.ToString(r.policy.Arn) }
func (r *IAMPolicyResource) GetARN() string  { return aws.ToString(r.policy.Arn) }
func (r *IAMPolicyResource) GetName() string { return aws.ToString(r.policy.PolicyName) }
func (r *IAMPolicyResource) GetType() string { return "iam:policies" }
func (r *IAMPolicyResource) GetRegion() string { return "global" }

func (r *IAMPolicyResource) GetCreatedAt() time.Time {
	if r.policy.CreateDate != nil {
		return *r.policy.CreateDate
	}
	return time.Time{}
}

func (r *IAMPolicyResource) GetTags() map[string]string {
	return nil
}

func (r *IAMPolicyResource) ToTableRow() []string {
	created := ""
	if r.policy.CreateDate != nil {
		created = r.policy.CreateDate.Format("2006-01-02")
	}

	attached := fmt.Sprintf("%d", aws.ToInt32(r.policy.AttachmentCount))

	description := aws.ToString(r.policy.Description)
	if len(description) > 50 {
		description = description[:47] + "..."
	}

	return []string{
		aws.ToString(r.policy.PolicyName),
		r.policyType,
		attached,
		created,
		description,
	}
}

func (r *IAMPolicyResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"PolicyName":      aws.ToString(r.policy.PolicyName),
		"PolicyId":        aws.ToString(r.policy.PolicyId),
		"ARN":             aws.ToString(r.policy.Arn),
		"Type":            r.policyType,
		"AttachmentCount": aws.ToInt32(r.policy.AttachmentCount),
		"CreateDate":      formatTime(r.policy.CreateDate),
		"Description":     aws.ToString(r.policy.Description),
	}
}
