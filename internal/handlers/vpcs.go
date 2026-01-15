package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2adapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ec2"
)

// VPCsHandler handles VPC resources
type VPCsHandler struct {
	BaseHandler
	client *ec2adapter.VPCsClient
	region string
}

// NewVPCsHandler creates a new VPCs handler
func NewVPCsHandler(ec2Client *ec2.Client, region string) *VPCsHandler {
	return &VPCsHandler{
		client: ec2adapter.NewVPCsClient(ec2Client),
		region: region,
	}
}

func (h *VPCsHandler) ResourceType() string { return "ec2:vpcs" }
func (h *VPCsHandler) ResourceName() string { return "VPCs" }
func (h *VPCsHandler) ResourceIcon() string { return "🌐" }
func (h *VPCsHandler) ShortcutKey() string  { return "vpc" }

func (h *VPCsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 25, Sortable: true},
		{Title: "VPC ID", Width: 22, Sortable: false},
		{Title: "CIDR", Width: 18, Sortable: false},
		{Title: "State", Width: 12, Sortable: true},
		{Title: "Default", Width: 8, Sortable: false},
		{Title: "Subnets", Width: 8, Sortable: false},
		{Title: "Tenancy", Width: 12, Sortable: false},
	}
}

func (h *VPCsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	vpcs, err := h.client.ListVPCs(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list VPCs", err)
	}

	resources := make([]Resource, 0, len(vpcs))
	for _, vpc := range vpcs {
		resource := &VPCResource{
			vpc:    vpc,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(vpc.Name)
			id := strings.ToLower(vpc.VpcID)
			cidr := strings.ToLower(vpc.CidrBlock)
			if !strings.Contains(name, filter) && !strings.Contains(id, filter) && !strings.Contains(cidr, filter) {
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

func (h *VPCsHandler) Get(ctx context.Context, id string) (Resource, error) {
	vpc, err := h.client.GetVPC(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get VPC %s", id), err)
	}

	return &VPCResource{
		vpc:    *vpc,
		region: h.region,
	}, nil
}

func (h *VPCsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	vpc, err := h.client.GetVPC(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe VPC %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["VPC"] = map[string]interface{}{
		"VpcId":           vpc.VpcID,
		"Name":            vpc.Name,
		"CidrBlock":       vpc.CidrBlock,
		"State":           vpc.State,
		"IsDefault":       vpc.IsDefault,
		"InstanceTenancy": vpc.InstanceTenancy,
		"DhcpOptionsId":   vpc.DhcpOptionsID,
		"OwnerId":         vpc.OwnerID,
	}

	// Get subnets for this VPC
	subnets, err := h.client.ListSubnets(ctx, id)
	if err == nil && len(subnets) > 0 {
		subnetList := make([]map[string]interface{}, 0, len(subnets))
		for _, subnet := range subnets {
			s := map[string]interface{}{
				"SubnetId":         subnet.SubnetID,
				"CidrBlock":        subnet.CidrBlock,
				"AvailabilityZone": subnet.AvailabilityZone,
				"AvailableIPs":     subnet.AvailableIPs,
				"State":            subnet.State,
			}
			if subnet.Name != "" {
				s["Name"] = subnet.Name
			}
			if subnet.MapPublicIP {
				s["MapPublicIP"] = true
			}
			subnetList = append(subnetList, s)
		}
		details["Subnets"] = subnetList
	}

	// Tags
	if len(vpc.Tags) > 0 {
		details["Tags"] = vpc.Tags
	}

	return details, nil
}

func (h *VPCsHandler) Actions() []Action {
	return []Action{
		{Key: "s", Name: "subnets", Description: "View subnets"},
		{Key: "r", Name: "routes", Description: "View route tables"},
		{Key: "n", Name: "nacls", Description: "View NACLs"},
	}
}

// VPCResource implements Resource interface for VPCs
type VPCResource struct {
	vpc    ec2adapter.VPC
	region string
}

func (r *VPCResource) GetID() string   { return r.vpc.VpcID }
func (r *VPCResource) GetName() string {
	if r.vpc.Name != "" {
		return r.vpc.Name
	}
	return r.vpc.VpcID
}
func (r *VPCResource) GetARN() string {
	return fmt.Sprintf("arn:aws:ec2:%s:%s:vpc/%s", r.region, r.vpc.OwnerID, r.vpc.VpcID)
}
func (r *VPCResource) GetType() string   { return "ec2:vpcs" }
func (r *VPCResource) GetRegion() string { return r.region }

func (r *VPCResource) GetCreatedAt() time.Time {
	return time.Time{} // VPCs don't have creation time
}

func (r *VPCResource) GetTags() map[string]string {
	return r.vpc.Tags
}

func (r *VPCResource) ToTableRow() []string {
	name := r.vpc.Name
	if name == "" {
		name = "-"
	}

	isDefault := "No"
	if r.vpc.IsDefault {
		isDefault = "Yes"
	}

	return []string{
		name,
		r.vpc.VpcID,
		r.vpc.CidrBlock,
		r.vpc.State,
		isDefault,
		fmt.Sprintf("%d", r.vpc.SubnetCount),
		r.vpc.InstanceTenancy,
	}
}

func (r *VPCResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"VpcId":           r.vpc.VpcID,
		"Name":            r.vpc.Name,
		"CidrBlock":       r.vpc.CidrBlock,
		"State":           r.vpc.State,
		"IsDefault":       r.vpc.IsDefault,
		"InstanceTenancy": r.vpc.InstanceTenancy,
		"SubnetCount":     r.vpc.SubnetCount,
	}
}
