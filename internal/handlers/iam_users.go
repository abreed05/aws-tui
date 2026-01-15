package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAMUsersHandler handles IAM User resources
type IAMUsersHandler struct {
	BaseHandler
	client *iam.Client
}

// NewIAMUsersHandler creates a new IAM users handler
func NewIAMUsersHandler(client *iam.Client) *IAMUsersHandler {
	return &IAMUsersHandler{client: client}
}

func (h *IAMUsersHandler) ResourceType() string { return "iam:users" }
func (h *IAMUsersHandler) ResourceName() string { return "IAM Users" }
func (h *IAMUsersHandler) ResourceIcon() string { return "👤" }
func (h *IAMUsersHandler) ShortcutKey() string  { return "users" }

func (h *IAMUsersHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 25, Sortable: true},
		{Title: "User ID", Width: 22, Sortable: false},
		{Title: "Created", Width: 12, Sortable: true},
		{Title: "Password Last Used", Width: 18, Sortable: true},
		{Title: "MFA", Width: 5, Sortable: false},
		{Title: "Access Keys", Width: 12, Sortable: false},
	}
}

func (h *IAMUsersHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	input := &iam.ListUsersInput{}

	if opts.PageSize > 0 {
		input.MaxItems = aws.Int32(int32(opts.PageSize))
	}
	if opts.NextToken != "" {
		input.Marker = aws.String(opts.NextToken)
	}

	result, err := h.client.ListUsers(ctx, input)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list IAM users", err)
	}

	resources := make([]Resource, 0, len(result.Users))
	for _, user := range result.Users {
		// Get additional details for each user (MFA, access keys)
		userResource := &IAMUserResource{user: user}

		// Get MFA devices count
		mfaResult, err := h.client.ListMFADevices(ctx, &iam.ListMFADevicesInput{
			UserName: user.UserName,
		})
		if err == nil {
			userResource.mfaCount = len(mfaResult.MFADevices)
		}

		// Get access keys count
		keysResult, err := h.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
			UserName: user.UserName,
		})
		if err == nil {
			userResource.accessKeyCount = len(keysResult.AccessKeyMetadata)
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(aws.ToString(user.UserName))
			if !strings.Contains(name, filter) {
				continue
			}
		}

		resources = append(resources, userResource)
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

func (h *IAMUsersHandler) Get(ctx context.Context, id string) (Resource, error) {
	result, err := h.client.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get IAM user %s", id), err)
	}

	userResource := &IAMUserResource{user: *result.User}

	// Get MFA devices
	mfaResult, _ := h.client.ListMFADevices(ctx, &iam.ListMFADevicesInput{
		UserName: aws.String(id),
	})
	if mfaResult != nil {
		userResource.mfaCount = len(mfaResult.MFADevices)
	}

	// Get access keys
	keysResult, _ := h.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(id),
	})
	if keysResult != nil {
		userResource.accessKeyCount = len(keysResult.AccessKeyMetadata)
	}

	return userResource, nil
}

func (h *IAMUsersHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	// Get basic user info
	userResult, err := h.client.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(id),
	})
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe IAM user %s", id), err)
	}

	user := userResult.User
	details := make(map[string]interface{})

	// Basic info
	details["User"] = map[string]interface{}{
		"UserName":         aws.ToString(user.UserName),
		"UserId":           aws.ToString(user.UserId),
		"ARN":              aws.ToString(user.Arn),
		"Path":             aws.ToString(user.Path),
		"CreateDate":       user.CreateDate.Format(time.RFC3339),
		"PasswordLastUsed": formatTime(user.PasswordLastUsed),
	}

	// Get attached managed policies
	policiesResult, err := h.client.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(id),
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
	inlineResult, err := h.client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
		UserName: aws.String(id),
	})
	if err == nil {
		details["InlinePolicies"] = inlineResult.PolicyNames
	}

	// Get groups
	groupsResult, err := h.client.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{
		UserName: aws.String(id),
	})
	if err == nil {
		groups := make([]string, 0, len(groupsResult.Groups))
		for _, g := range groupsResult.Groups {
			groups = append(groups, aws.ToString(g.GroupName))
		}
		details["Groups"] = groups
	}

	// Get access keys
	keysResult, err := h.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(id),
	})
	if err == nil {
		keys := make([]map[string]string, 0, len(keysResult.AccessKeyMetadata))
		for _, k := range keysResult.AccessKeyMetadata {
			keys = append(keys, map[string]string{
				"AccessKeyId": aws.ToString(k.AccessKeyId),
				"Status":      string(k.Status),
				"CreateDate":  k.CreateDate.Format(time.RFC3339),
			})
		}
		details["AccessKeys"] = keys
	}

	// Get MFA devices
	mfaResult, err := h.client.ListMFADevices(ctx, &iam.ListMFADevicesInput{
		UserName: aws.String(id),
	})
	if err == nil {
		mfaDevices := make([]map[string]string, 0, len(mfaResult.MFADevices))
		for _, m := range mfaResult.MFADevices {
			mfaDevices = append(mfaDevices, map[string]string{
				"SerialNumber": aws.ToString(m.SerialNumber),
				"EnableDate":   m.EnableDate.Format(time.RFC3339),
			})
		}
		details["MFADevices"] = mfaDevices
	}

	// Get tags
	tagsResult, err := h.client.ListUserTags(ctx, &iam.ListUserTagsInput{
		UserName: aws.String(id),
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

func (h *IAMUsersHandler) Actions() []Action {
	return []Action{
		{Key: "p", Name: "policies", Description: "View attached policies"},
		{Key: "g", Name: "groups", Description: "View group memberships"},
		{Key: "k", Name: "access-keys", Description: "View access keys"},
		{Key: "m", Name: "mfa", Description: "View MFA devices"},
	}
}

// IAMUserResource implements Resource interface for IAM users
type IAMUserResource struct {
	user           types.User
	mfaCount       int
	accessKeyCount int
}

func (r *IAMUserResource) GetID() string   { return aws.ToString(r.user.UserName) }
func (r *IAMUserResource) GetARN() string  { return aws.ToString(r.user.Arn) }
func (r *IAMUserResource) GetName() string { return aws.ToString(r.user.UserName) }
func (r *IAMUserResource) GetType() string { return "iam:users" }
func (r *IAMUserResource) GetRegion() string { return "global" }

func (r *IAMUserResource) GetCreatedAt() time.Time {
	if r.user.CreateDate != nil {
		return *r.user.CreateDate
	}
	return time.Time{}
}

func (r *IAMUserResource) GetTags() map[string]string {
	// Tags need to be fetched separately
	return nil
}

func (r *IAMUserResource) ToTableRow() []string {
	passwordLastUsed := "Never"
	if r.user.PasswordLastUsed != nil {
		passwordLastUsed = r.user.PasswordLastUsed.Format("2006-01-02")
	}

	mfaStatus := "No"
	if r.mfaCount > 0 {
		mfaStatus = "Yes"
	}

	accessKeys := fmt.Sprintf("%d", r.accessKeyCount)

	created := ""
	if r.user.CreateDate != nil {
		created = r.user.CreateDate.Format("2006-01-02")
	}

	return []string{
		aws.ToString(r.user.UserName),
		aws.ToString(r.user.UserId),
		created,
		passwordLastUsed,
		mfaStatus,
		accessKeys,
	}
}

func (r *IAMUserResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"UserName":         aws.ToString(r.user.UserName),
		"UserId":           aws.ToString(r.user.UserId),
		"ARN":              aws.ToString(r.user.Arn),
		"Path":             aws.ToString(r.user.Path),
		"CreateDate":       formatTime(r.user.CreateDate),
		"PasswordLastUsed": formatTime(r.user.PasswordLastUsed),
		"MFAEnabled":       r.mfaCount > 0,
		"AccessKeyCount":   r.accessKeyCount,
	}
}

// Helper function to format time pointers
func formatTime(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format(time.RFC3339)
}
