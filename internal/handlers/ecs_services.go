package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ecs"
)

// ECSServicesHandler handles ECS Service resources
type ECSServicesHandler struct {
	BaseHandler
	client      *ecsadapter.ClustersClient
	region      string
	clusterARN  string
	clusterName string
}

// NewECSServicesHandlerForCluster creates a new ECS services handler for a specific cluster
func NewECSServicesHandlerForCluster(ecsClient *ecs.Client, region, clusterARN, clusterName string) *ECSServicesHandler {
	return &ECSServicesHandler{
		client:      ecsadapter.NewClustersClient(ecsClient),
		region:      region,
		clusterARN:  clusterARN,
		clusterName: clusterName,
	}
}

func (h *ECSServicesHandler) ResourceType() string { return "ecs:services" }
func (h *ECSServicesHandler) ResourceName() string { return "ECS Services" }
func (h *ECSServicesHandler) ResourceIcon() string { return "⚙" }
func (h *ECSServicesHandler) ShortcutKey() string  { return "ecs-services" }

func (h *ECSServicesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Service Name", Width: 30, Sortable: true},
		{Title: "Status", Width: 12, Sortable: true},
		{Title: "Desired", Width: 10, Sortable: false},
		{Title: "Running", Width: 10, Sortable: false},
		{Title: "Pending", Width: 10, Sortable: false},
		{Title: "Launch Type", Width: 12, Sortable: false},
		{Title: "Task Definition", Width: 40, Sortable: false},
	}
}

func (h *ECSServicesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	services, err := h.client.ListServices(ctx, h.clusterARN)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list ECS services", err)
	}

	resources := make([]Resource, 0, len(services))
	for _, service := range services {
		resource := &ECSServiceResource{
			service: service,
			region:  h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(service.ServiceName)
			status := strings.ToLower(service.Status)
			if !strings.Contains(name, filter) && !strings.Contains(status, filter) {
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

func (h *ECSServicesHandler) Get(ctx context.Context, id string) (Resource, error) {
	services, err := h.client.ListServices(ctx, h.clusterARN)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get ECS service %s", id), err)
	}

	for _, service := range services {
		if service.ServiceName == id || service.ServiceARN == id {
			return &ECSServiceResource{
				service: service,
				region:  h.region,
			}, nil
		}
	}

	return nil, NewHandlerError("NOT_FOUND", fmt.Sprintf("service %s not found", id), nil)
}

func (h *ECSServicesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	resource, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	serviceResource, ok := resource.(*ECSServiceResource)
	if !ok {
		return nil, NewHandlerError("DESCRIBE_FAILED", "failed to convert resource to service", nil)
	}

	service := serviceResource.service
	details := make(map[string]interface{})

	// Basic info
	details["Service"] = map[string]interface{}{
		"ServiceName": service.ServiceName,
		"ServiceArn":  service.ServiceARN,
		"ClusterArn":  service.ClusterARN,
		"Status":      service.Status,
	}

	// Deployment info
	details["Deployment"] = map[string]interface{}{
		"DesiredCount":   service.DesiredCount,
		"RunningCount":   service.RunningCount,
		"PendingCount":   service.PendingCount,
		"LaunchType":     service.LaunchType,
		"TaskDefinition": service.TaskDefinition,
	}

	// Timestamps
	if service.CreatedAt != "" {
		details["CreatedAt"] = service.CreatedAt
	}

	// Tags
	if len(service.Tags) > 0 {
		details["Tags"] = service.Tags
	}

	return details, nil
}

func (h *ECSServicesHandler) Actions() []Action {
	return []Action{
		{Key: "t", Name: "tasks", Description: "View tasks"},
	}
}

func (h *ECSServicesHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	if action != "tasks" {
		return ErrNotSupported
	}

	service, err := h.Get(ctx, resourceID)
	if err != nil {
		return err
	}

	return &NavigateToTasksAction{
		ClusterARN:  h.clusterARN,
		ClusterName: h.clusterName,
		ServiceARN:  service.GetARN(),
		ServiceName: service.GetName(),
	}
}

// ECSServiceResource implements Resource interface for ECS services
type ECSServiceResource struct {
	service ecsadapter.Service
	region  string
}

func (r *ECSServiceResource) GetID() string   { return r.service.ServiceName }
func (r *ECSServiceResource) GetName() string { return r.service.ServiceName }
func (r *ECSServiceResource) GetARN() string  { return r.service.ServiceARN }
func (r *ECSServiceResource) GetType() string   { return "ecs:services" }
func (r *ECSServiceResource) GetRegion() string { return r.region }

func (r *ECSServiceResource) GetCreatedAt() time.Time {
	// Parse CreatedAt string if available
	if r.service.CreatedAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", r.service.CreatedAt)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func (r *ECSServiceResource) GetTags() map[string]string {
	return r.service.Tags
}

func (r *ECSServiceResource) ToTableRow() []string {
	return []string{
		r.service.ServiceName,
		r.service.Status,
		fmt.Sprintf("%d", r.service.DesiredCount),
		fmt.Sprintf("%d", r.service.RunningCount),
		fmt.Sprintf("%d", r.service.PendingCount),
		r.service.LaunchType,
		r.service.TaskDefinition,
	}
}

func (r *ECSServiceResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"ServiceName":    r.service.ServiceName,
		"ServiceArn":     r.service.ServiceARN,
		"ClusterArn":     r.service.ClusterARN,
		"Status":         r.service.Status,
		"DesiredCount":   r.service.DesiredCount,
		"RunningCount":   r.service.RunningCount,
		"PendingCount":   r.service.PendingCount,
		"LaunchType":     r.service.LaunchType,
		"TaskDefinition": r.service.TaskDefinition,
	}
}
