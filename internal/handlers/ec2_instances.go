package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2adapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ec2"
)

// EC2InstancesHandler handles EC2 Instance resources
type EC2InstancesHandler struct {
	BaseHandler
	client *ec2adapter.InstancesClient
	region string
}

// NewEC2InstancesHandler creates a new EC2 instances handler
func NewEC2InstancesHandler(ec2Client *ec2.Client, region string) *EC2InstancesHandler {
	return &EC2InstancesHandler{
		client: ec2adapter.NewInstancesClient(ec2Client),
		region: region,
	}
}

func (h *EC2InstancesHandler) ResourceType() string { return "ec2:instances" }
func (h *EC2InstancesHandler) ResourceName() string { return "EC2 Instances" }
func (h *EC2InstancesHandler) ResourceIcon() string { return "💻" }
func (h *EC2InstancesHandler) ShortcutKey() string  { return "ec2" }

func (h *EC2InstancesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Name", Width: 25, Sortable: true},
		{Title: "Instance ID", Width: 20, Sortable: false},
		{Title: "State", Width: 12, Sortable: true},
		{Title: "Type", Width: 12, Sortable: true},
		{Title: "Private IP", Width: 16, Sortable: false},
		{Title: "Public IP", Width: 16, Sortable: false},
		{Title: "AZ", Width: 12, Sortable: false},
	}
}

func (h *EC2InstancesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	instances, err := h.client.ListInstances(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list EC2 instances", err)
	}

	resources := make([]Resource, 0, len(instances))
	for _, inst := range instances {
		resource := &EC2InstanceResource{
			instance: inst,
			region:   h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(inst.Name)
			id := strings.ToLower(inst.InstanceID)
			state := strings.ToLower(inst.State)
			instType := strings.ToLower(inst.InstanceType)
			if !strings.Contains(name, filter) && !strings.Contains(id, filter) &&
				!strings.Contains(state, filter) && !strings.Contains(instType, filter) {
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

func (h *EC2InstancesHandler) Get(ctx context.Context, id string) (Resource, error) {
	inst, err := h.client.GetInstance(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get instance %s", id), err)
	}

	return &EC2InstanceResource{
		instance: *inst,
		region:   h.region,
	}, nil
}

func (h *EC2InstancesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	inst, err := h.client.GetInstance(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe instance %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Instance"] = map[string]interface{}{
		"InstanceId":   inst.InstanceID,
		"Name":         inst.Name,
		"State":        inst.State,
		"InstanceType": inst.InstanceType,
		"Platform":     inst.Platform,
		"ImageId":      inst.ImageID,
		"KeyName":      inst.KeyName,
		"LaunchTime":   inst.LaunchTime.Format(time.RFC3339),
	}

	// Networking
	networking := map[string]interface{}{
		"VpcId":            inst.VpcID,
		"SubnetId":         inst.SubnetID,
		"AvailabilityZone": inst.AvailabilityZone,
		"PrivateIpAddress": inst.PrivateIP,
	}
	if inst.PublicIP != "" {
		networking["PublicIpAddress"] = inst.PublicIP
	}
	details["Networking"] = networking

	// Security
	if len(inst.SecurityGroups) > 0 {
		details["SecurityGroups"] = inst.SecurityGroups
	}

	if inst.IAMRole != "" {
		details["IAMInstanceProfile"] = inst.IAMRole
	}

	// Tags
	if len(inst.Tags) > 0 {
		details["Tags"] = inst.Tags
	}

	return details, nil
}

func (h *EC2InstancesHandler) Actions() []Action {
	return []Action{
		{Key: "s", Name: "start", Description: "Start instance"},
		{Key: "S", Name: "stop", Description: "Stop instance"},
		{Key: "r", Name: "reboot", Description: "Reboot instance"},
		{Key: "c", Name: "connect", Description: "Connection info"},
	}
}

// EC2InstanceResource implements Resource interface for EC2 instances
type EC2InstanceResource struct {
	instance ec2adapter.Instance
	region   string
}

func (r *EC2InstanceResource) GetID() string   { return r.instance.InstanceID }
func (r *EC2InstanceResource) GetName() string {
	if r.instance.Name != "" {
		return r.instance.Name
	}
	return r.instance.InstanceID
}
func (r *EC2InstanceResource) GetARN() string {
	return fmt.Sprintf("arn:aws:ec2:%s::instance/%s", r.region, r.instance.InstanceID)
}
func (r *EC2InstanceResource) GetType() string   { return "ec2:instances" }
func (r *EC2InstanceResource) GetRegion() string { return r.region }

func (r *EC2InstanceResource) GetCreatedAt() time.Time {
	return r.instance.LaunchTime
}

func (r *EC2InstanceResource) GetTags() map[string]string {
	return r.instance.Tags
}

func (r *EC2InstanceResource) ToTableRow() []string {
	name := r.instance.Name
	if name == "" {
		name = "-"
	}

	publicIP := r.instance.PublicIP
	if publicIP == "" {
		publicIP = "-"
	}

	privateIP := r.instance.PrivateIP
	if privateIP == "" {
		privateIP = "-"
	}

	return []string{
		name,
		r.instance.InstanceID,
		r.instance.State,
		r.instance.InstanceType,
		privateIP,
		publicIP,
		r.instance.AvailabilityZone,
	}
}

func (r *EC2InstanceResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"InstanceId":       r.instance.InstanceID,
		"Name":             r.instance.Name,
		"State":            r.instance.State,
		"InstanceType":     r.instance.InstanceType,
		"Platform":         r.instance.Platform,
		"PrivateIpAddress": r.instance.PrivateIP,
		"PublicIpAddress":  r.instance.PublicIP,
		"VpcId":            r.instance.VpcID,
		"AvailabilityZone": r.instance.AvailabilityZone,
	}
}
