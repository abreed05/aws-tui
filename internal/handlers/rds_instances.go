package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	rdsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/rds"
)

// RDSInstancesHandler handles RDS Instance resources
type RDSInstancesHandler struct {
	BaseHandler
	client *rdsadapter.InstancesClient
	region string
}

// NewRDSInstancesHandler creates a new RDS instances handler
func NewRDSInstancesHandler(rdsClient *rds.Client, region string) *RDSInstancesHandler {
	return &RDSInstancesHandler{
		client: rdsadapter.NewInstancesClient(rdsClient),
		region: region,
	}
}

func (h *RDSInstancesHandler) ResourceType() string { return "rds:instances" }
func (h *RDSInstancesHandler) ResourceName() string { return "RDS Instances" }
func (h *RDSInstancesHandler) ResourceIcon() string { return "🗄️" }
func (h *RDSInstancesHandler) ShortcutKey() string  { return "rds" }

func (h *RDSInstancesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "DB Identifier", Width: 25, Sortable: true},
		{Title: "Engine", Width: 15, Sortable: true},
		{Title: "Status", Width: 12, Sortable: true},
		{Title: "Class", Width: 15, Sortable: true},
		{Title: "Storage", Width: 10, Sortable: false},
		{Title: "Multi-AZ", Width: 8, Sortable: false},
		{Title: "Endpoint", Width: 35, Sortable: false},
	}
}

func (h *RDSInstancesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	instances, err := h.client.ListDBInstances(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list RDS instances", err)
	}

	resources := make([]Resource, 0, len(instances))
	for _, inst := range instances {
		resource := &RDSInstanceResource{
			instance: inst,
			region:   h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			id := strings.ToLower(inst.DBInstanceID)
			engine := strings.ToLower(inst.Engine)
			status := strings.ToLower(inst.Status)
			if !strings.Contains(id, filter) && !strings.Contains(engine, filter) && !strings.Contains(status, filter) {
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

func (h *RDSInstancesHandler) Get(ctx context.Context, id string) (Resource, error) {
	inst, err := h.client.GetDBInstance(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get RDS instance %s", id), err)
	}

	return &RDSInstanceResource{
		instance: *inst,
		region:   h.region,
	}, nil
}

func (h *RDSInstancesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	inst, err := h.client.GetDBInstance(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe RDS instance %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Instance"] = map[string]interface{}{
		"DBInstanceIdentifier": inst.DBInstanceID,
		"DBInstanceClass":      inst.DBInstanceClass,
		"Engine":               inst.Engine,
		"EngineVersion":        inst.EngineVersion,
		"Status":               inst.Status,
		"MasterUsername":       inst.MasterUsername,
		"DBName":               inst.DBName,
		"CreatedTime":          inst.CreatedTime.Format(time.RFC3339),
	}

	// Connection
	connection := map[string]interface{}{
		"Endpoint": inst.Endpoint,
		"Port":     inst.Port,
	}
	if inst.PubliclyAccessible {
		connection["PubliclyAccessible"] = true
	}
	details["Connection"] = connection

	// Storage
	details["Storage"] = map[string]interface{}{
		"AllocatedStorage": fmt.Sprintf("%d GB", inst.AllocatedStorage),
		"StorageType":      inst.StorageType,
		"Encrypted":        inst.StorageEncrypted,
	}

	// Availability
	availability := map[string]interface{}{
		"MultiAZ":          inst.MultiAZ,
		"AvailabilityZone": inst.AvailabilityZone,
	}
	if inst.VpcID != "" {
		availability["VpcId"] = inst.VpcID
	}
	details["Availability"] = availability

	// Maintenance
	details["Maintenance"] = map[string]interface{}{
		"AutoMinorVersionUpgrade": inst.AutoMinorVersionUpgrade,
		"BackupRetentionPeriod":   fmt.Sprintf("%d days", inst.BackupRetentionPeriod),
	}

	// Tags
	if len(inst.Tags) > 0 {
		details["Tags"] = inst.Tags
	}

	return details, nil
}

func (h *RDSInstancesHandler) Actions() []Action {
	return []Action{
		{Key: "s", Name: "start", Description: "Start instance"},
		{Key: "S", Name: "stop", Description: "Stop instance"},
		{Key: "r", Name: "reboot", Description: "Reboot instance"},
		{Key: "b", Name: "snapshots", Description: "View snapshots"},
	}
}

// RDSInstanceResource implements Resource interface for RDS instances
type RDSInstanceResource struct {
	instance rdsadapter.DBInstance
	region   string
}

func (r *RDSInstanceResource) GetID() string   { return r.instance.DBInstanceID }
func (r *RDSInstanceResource) GetName() string { return r.instance.DBInstanceID }
func (r *RDSInstanceResource) GetARN() string {
	return fmt.Sprintf("arn:aws:rds:%s::db:%s", r.region, r.instance.DBInstanceID)
}
func (r *RDSInstanceResource) GetType() string   { return "rds:instances" }
func (r *RDSInstanceResource) GetRegion() string { return r.region }

func (r *RDSInstanceResource) GetCreatedAt() time.Time {
	return r.instance.CreatedTime
}

func (r *RDSInstanceResource) GetTags() map[string]string {
	return r.instance.Tags
}

func (r *RDSInstanceResource) ToTableRow() []string {
	engineVersion := r.instance.Engine
	if r.instance.EngineVersion != "" {
		engineVersion = fmt.Sprintf("%s %s", r.instance.Engine, r.instance.EngineVersion)
	}

	multiAZ := "No"
	if r.instance.MultiAZ {
		multiAZ = "Yes"
	}

	storage := fmt.Sprintf("%d GB", r.instance.AllocatedStorage)

	endpoint := r.instance.Endpoint
	if endpoint == "" {
		endpoint = "-"
	}

	return []string{
		r.instance.DBInstanceID,
		engineVersion,
		r.instance.Status,
		r.instance.DBInstanceClass,
		storage,
		multiAZ,
		endpoint,
	}
}

func (r *RDSInstanceResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"DBInstanceIdentifier": r.instance.DBInstanceID,
		"Engine":               r.instance.Engine,
		"EngineVersion":        r.instance.EngineVersion,
		"Status":               r.instance.Status,
		"DBInstanceClass":      r.instance.DBInstanceClass,
		"AllocatedStorage":     r.instance.AllocatedStorage,
		"MultiAZ":              r.instance.MultiAZ,
		"Endpoint":             r.instance.Endpoint,
	}
}
