package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2adapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ec2"
)

// SecurityGroupsHandler handles EC2 Security Group resources
type SecurityGroupsHandler struct {
	BaseHandler
	client *ec2adapter.SecurityGroupsClient
	region string
}

// NewSecurityGroupsHandler creates a new security groups handler
func NewSecurityGroupsHandler(ec2Client *ec2.Client, region string) *SecurityGroupsHandler {
	return &SecurityGroupsHandler{
		client: ec2adapter.NewSecurityGroupsClient(ec2Client),
		region: region,
	}
}

func (h *SecurityGroupsHandler) ResourceType() string { return "ec2:security-groups" }
func (h *SecurityGroupsHandler) ResourceName() string { return "Security Groups" }
func (h *SecurityGroupsHandler) ResourceIcon() string { return "🔒" }
func (h *SecurityGroupsHandler) ShortcutKey() string  { return "sg" }

func (h *SecurityGroupsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 25, Sortable: true},
		{Title: "Group ID", Width: 22, Sortable: false},
		{Title: "VPC ID", Width: 22, Sortable: false},
		{Title: "Inbound", Width: 10, Sortable: false},
		{Title: "Outbound", Width: 10, Sortable: false},
		{Title: "Description", Width: 30, Sortable: false},
	}
}

func (h *SecurityGroupsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	securityGroups, err := h.client.ListSecurityGroups(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list security groups", err)
	}

	resources := make([]Resource, 0, len(securityGroups))
	for _, sg := range securityGroups {
		resource := &SecurityGroupResource{
			sg:     sg,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(sg.GroupName)
			id := strings.ToLower(sg.GroupID)
			desc := strings.ToLower(sg.Description)
			if !strings.Contains(name, filter) && !strings.Contains(id, filter) && !strings.Contains(desc, filter) {
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

func (h *SecurityGroupsHandler) Get(ctx context.Context, id string) (Resource, error) {
	sg, err := h.client.GetSecurityGroup(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get security group %s", id), err)
	}

	return &SecurityGroupResource{
		sg:     *sg,
		region: h.region,
	}, nil
}

func (h *SecurityGroupsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	sg, err := h.client.GetSecurityGroup(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe security group %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["SecurityGroup"] = map[string]interface{}{
		"GroupId":     sg.GroupID,
		"GroupName":   sg.GroupName,
		"Description": sg.Description,
		"VpcId":       sg.VpcID,
		"OwnerId":     sg.OwnerID,
	}

	// Inbound rules
	if len(sg.InboundRules) > 0 {
		inbound := make([]map[string]interface{}, 0, len(sg.InboundRules))
		for _, rule := range sg.InboundRules {
			inbound = append(inbound, formatRule(rule))
		}
		details["InboundRules"] = inbound
	}

	// Outbound rules
	if len(sg.OutboundRules) > 0 {
		outbound := make([]map[string]interface{}, 0, len(sg.OutboundRules))
		for _, rule := range sg.OutboundRules {
			outbound = append(outbound, formatRule(rule))
		}
		details["OutboundRules"] = outbound
	}

	// Tags
	if len(sg.Tags) > 0 {
		details["Tags"] = sg.Tags
	}

	return details, nil
}

func formatRule(rule ec2adapter.SecurityGroupRule) map[string]interface{} {
	result := make(map[string]interface{})

	// Protocol
	protocol := rule.Protocol
	if protocol == "-1" {
		protocol = "All"
	}
	result["Protocol"] = protocol

	// Ports
	if rule.FromPort == rule.ToPort {
		if rule.FromPort == 0 && rule.Protocol == "-1" {
			result["Ports"] = "All"
		} else if rule.FromPort == -1 {
			result["Ports"] = "All"
		} else {
			result["Ports"] = fmt.Sprintf("%d", rule.FromPort)
		}
	} else {
		result["Ports"] = fmt.Sprintf("%d-%d", rule.FromPort, rule.ToPort)
	}

	// Sources
	var sources []string
	sources = append(sources, rule.IPRanges...)
	sources = append(sources, rule.IPv6Ranges...)
	sources = append(sources, rule.PrefixLists...)
	sources = append(sources, rule.SGSources...)
	result["Sources"] = sources

	if rule.Description != "" {
		result["Description"] = rule.Description
	}

	return result
}

func (h *SecurityGroupsHandler) Actions() []Action {
	return []Action{
		// No custom actions - inbound/outbound rules are shown in describe view
	}
}

// SecurityGroupResource implements Resource interface for EC2 security groups
type SecurityGroupResource struct {
	sg     ec2adapter.SecurityGroup
	region string
}

func (r *SecurityGroupResource) GetID() string { return r.sg.GroupID }
func (r *SecurityGroupResource) GetARN() string {
	// Security groups don't have an ARN in the API response, construct it
	return fmt.Sprintf("arn:aws:ec2:%s:%s:security-group/%s", r.region, r.sg.OwnerID, r.sg.GroupID)
}
func (r *SecurityGroupResource) GetName() string {
	// Check for Name tag first
	if name, ok := r.sg.Tags["Name"]; ok && name != "" {
		return name
	}
	return r.sg.GroupName
}
func (r *SecurityGroupResource) GetType() string   { return "ec2:security-groups" }
func (r *SecurityGroupResource) GetRegion() string { return r.region }

func (r *SecurityGroupResource) GetCreatedAt() time.Time {
	// Security groups don't have a creation timestamp
	return time.Time{}
}

func (r *SecurityGroupResource) GetTags() map[string]string {
	return r.sg.Tags
}

func (r *SecurityGroupResource) ToTableRow() []string {
	name := r.sg.GroupName
	if tagName, ok := r.sg.Tags["Name"]; ok && tagName != "" {
		name = tagName
	}

	return []string{
		name,
		r.sg.GroupID,
		r.sg.VpcID,
		fmt.Sprintf("%d rules", len(r.sg.InboundRules)),
		fmt.Sprintf("%d rules", len(r.sg.OutboundRules)),
		truncateString(r.sg.Description, 30),
	}
}

func (r *SecurityGroupResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"GroupId":       r.sg.GroupID,
		"GroupName":     r.sg.GroupName,
		"Description":   r.sg.Description,
		"VpcId":         r.sg.VpcID,
		"OwnerId":       r.sg.OwnerID,
		"InboundRules":  len(r.sg.InboundRules),
		"OutboundRules": len(r.sg.OutboundRules),
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
