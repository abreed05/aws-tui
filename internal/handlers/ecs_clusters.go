package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ecs"
)

// NavigateToServicesAction is returned by ExecuteAction to trigger navigation to services
type NavigateToServicesAction struct {
	ClusterARN  string
	ClusterName string
}

func (a *NavigateToServicesAction) Error() string {
	return fmt.Sprintf("navigate to services for cluster %s", a.ClusterName)
}

func (a *NavigateToServicesAction) IsActionMsg() {}

// NavigateToTasksAction is returned by ExecuteAction to trigger navigation to tasks
type NavigateToTasksAction struct {
	ClusterARN  string
	ClusterName string
	ServiceARN  string // Optional - if set, filter by service
	ServiceName string // Optional - if set, filter by service
}

func (a *NavigateToTasksAction) Error() string {
	if a.ServiceName != "" {
		return fmt.Sprintf("navigate to tasks for service %s", a.ServiceName)
	}
	return fmt.Sprintf("navigate to tasks for cluster %s", a.ClusterName)
}

func (a *NavigateToTasksAction) IsActionMsg() {}

// ECSClustersHandler handles ECS Cluster resources
type ECSClustersHandler struct {
	BaseHandler
	client *ecsadapter.ClustersClient
	region string
}

// NewECSClustersHandler creates a new ECS clusters handler
func NewECSClustersHandler(ecsClient *ecs.Client, region string) *ECSClustersHandler {
	return &ECSClustersHandler{
		client: ecsadapter.NewClustersClient(ecsClient),
		region: region,
	}
}

func (h *ECSClustersHandler) ResourceType() string { return "ecs:clusters" }
func (h *ECSClustersHandler) ResourceName() string { return "ECS Clusters" }
func (h *ECSClustersHandler) ResourceIcon() string { return "🐳" }
func (h *ECSClustersHandler) ShortcutKey() string  { return "ecs" }

func (h *ECSClustersHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Cluster Name", Width: 30, Sortable: true},
		{Title: "Status", Width: 10, Sortable: true},
		{Title: "Running", Width: 10, Sortable: false},
		{Title: "Pending", Width: 10, Sortable: false},
		{Title: "Services", Width: 10, Sortable: false},
		{Title: "Instances", Width: 10, Sortable: false},
	}
}

func (h *ECSClustersHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	clusters, err := h.client.ListClusters(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list ECS clusters", err)
	}

	resources := make([]Resource, 0, len(clusters))
	for _, cluster := range clusters {
		resource := &ECSClusterResource{
			cluster: cluster,
			region:  h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(cluster.ClusterName)
			status := strings.ToLower(cluster.Status)
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

func (h *ECSClustersHandler) Get(ctx context.Context, id string) (Resource, error) {
	cluster, err := h.client.GetCluster(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get ECS cluster %s", id), err)
	}

	return &ECSClusterResource{
		cluster: *cluster,
		region:  h.region,
	}, nil
}

func (h *ECSClustersHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	cluster, err := h.client.GetCluster(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe ECS cluster %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Cluster"] = map[string]interface{}{
		"ClusterName": cluster.ClusterName,
		"ClusterArn":  cluster.ClusterARN,
		"Status":      cluster.Status,
	}

	// Tasks and Services
	details["Resources"] = map[string]interface{}{
		"RunningTasks":       cluster.RunningTasksCount,
		"PendingTasks":       cluster.PendingTasksCount,
		"ActiveServices":     cluster.ActiveServicesCount,
		"ContainerInstances": cluster.RegisteredContainerInstances,
	}

	// Capacity providers
	if len(cluster.CapacityProviders) > 0 {
		details["CapacityProviders"] = cluster.CapacityProviders
	}

	// Get services for this cluster
	services, err := h.client.ListServices(ctx, cluster.ClusterARN)
	if err == nil && len(services) > 0 {
		serviceList := make([]map[string]interface{}, 0, len(services))
		for _, svc := range services {
			s := map[string]interface{}{
				"ServiceName":    svc.ServiceName,
				"Status":         svc.Status,
				"DesiredCount":   svc.DesiredCount,
				"RunningCount":   svc.RunningCount,
				"LaunchType":     svc.LaunchType,
				"TaskDefinition": svc.TaskDefinition,
			}
			serviceList = append(serviceList, s)
		}
		details["Services"] = serviceList
	}

	// Tags
	if len(cluster.Tags) > 0 {
		details["Tags"] = cluster.Tags
	}

	return details, nil
}

func (h *ECSClustersHandler) Actions() []Action {
	return []Action{
		{Key: "s", Name: "services", Description: "View services"},
		{Key: "t", Name: "tasks", Description: "View tasks"},
	}
}

func (h *ECSClustersHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	cluster, err := h.Get(ctx, resourceID)
	if err != nil {
		return err
	}

	switch action {
	case "services":
		return &NavigateToServicesAction{
			ClusterARN:  cluster.GetARN(),
			ClusterName: cluster.GetName(),
		}
	case "tasks":
		return &NavigateToTasksAction{
			ClusterARN:  cluster.GetARN(),
			ClusterName: cluster.GetName(),
		}
	default:
		return ErrNotSupported
	}
}

// ECSClusterResource implements Resource interface for ECS clusters
type ECSClusterResource struct {
	cluster ecsadapter.Cluster
	region  string
}

func (r *ECSClusterResource) GetID() string   { return r.cluster.ClusterName }
func (r *ECSClusterResource) GetName() string { return r.cluster.ClusterName }
func (r *ECSClusterResource) GetARN() string  { return r.cluster.ClusterARN }
func (r *ECSClusterResource) GetType() string   { return "ecs:clusters" }
func (r *ECSClusterResource) GetRegion() string { return r.region }

func (r *ECSClusterResource) GetCreatedAt() time.Time {
	return time.Time{} // ECS clusters don't have creation time in the API
}

func (r *ECSClusterResource) GetTags() map[string]string {
	return r.cluster.Tags
}

func (r *ECSClusterResource) ToTableRow() []string {
	return []string{
		r.cluster.ClusterName,
		r.cluster.Status,
		fmt.Sprintf("%d", r.cluster.RunningTasksCount),
		fmt.Sprintf("%d", r.cluster.PendingTasksCount),
		fmt.Sprintf("%d", r.cluster.ActiveServicesCount),
		fmt.Sprintf("%d", r.cluster.RegisteredContainerInstances),
	}
}

func (r *ECSClusterResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"ClusterName":        r.cluster.ClusterName,
		"ClusterArn":         r.cluster.ClusterARN,
		"Status":             r.cluster.Status,
		"RunningTasks":       r.cluster.RunningTasksCount,
		"PendingTasks":       r.cluster.PendingTasksCount,
		"ActiveServices":     r.cluster.ActiveServicesCount,
		"ContainerInstances": r.cluster.RegisteredContainerInstances,
	}
}
