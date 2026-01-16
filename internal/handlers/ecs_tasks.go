package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/ecs"
	"github.com/aaw-tui/aws-tui/internal/ui/messages"
)

// ECSTasksHandler handles ECS Task resources
type ECSTasksHandler struct {
	BaseHandler
	client      *ecsadapter.TasksClient
	region      string
	clusterARN  string
	clusterName string
	serviceARN  string
	serviceName string
}

// NewECSTasksHandlerForService creates a new ECS tasks handler for a specific service
func NewECSTasksHandlerForService(ecsClient *ecs.Client, region, clusterARN, clusterName, serviceARN, serviceName string) *ECSTasksHandler {
	return &ECSTasksHandler{
		client:      ecsadapter.NewTasksClient(ecsClient),
		region:      region,
		clusterARN:  clusterARN,
		clusterName: clusterName,
		serviceARN:  serviceARN,
		serviceName: serviceName,
	}
}

// NewECSTasksHandlerForCluster creates a new ECS tasks handler for all tasks in a cluster
func NewECSTasksHandlerForCluster(ecsClient *ecs.Client, region, clusterARN, clusterName string) *ECSTasksHandler {
	return &ECSTasksHandler{
		client:      ecsadapter.NewTasksClient(ecsClient),
		region:      region,
		clusterARN:  clusterARN,
		clusterName: clusterName,
	}
}

func (h *ECSTasksHandler) ResourceType() string { return "ecs:tasks" }
func (h *ECSTasksHandler) ResourceName() string { return "ECS Tasks" }
func (h *ECSTasksHandler) ResourceIcon() string { return "📦" }
func (h *ECSTasksHandler) ShortcutKey() string  { return "ecs-tasks" }

func (h *ECSTasksHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Task ID", Width: 30, Sortable: true},
		{Title: "Status", Width: 12, Sortable: true},
		{Title: "Launch Type", Width: 12, Sortable: false},
		{Title: "Containers", Width: 10, Sortable: false},
		{Title: "Exec Enabled", Width: 12, Sortable: false},
		{Title: "Started At", Width: 20, Sortable: true},
	}
}

func (h *ECSTasksHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	tasks, err := h.client.ListTasks(ctx, h.clusterARN, h.serviceARN)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list ECS tasks", err)
	}

	resources := make([]Resource, 0, len(tasks))
	for _, task := range tasks {
		resource := &ECSTaskResource{
			task:   task,
			region: h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			taskID := strings.ToLower(getTaskIDFromARN(task.TaskARN))
			status := strings.ToLower(task.LastStatus)
			if !strings.Contains(taskID, filter) && !strings.Contains(status, filter) {
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

func (h *ECSTasksHandler) Get(ctx context.Context, id string) (Resource, error) {
	// If id is short form, we need the full ARN
	taskARN := id
	if !strings.Contains(id, "arn:") {
		// List all tasks and find matching one
		tasks, err := h.client.ListTasks(ctx, h.clusterARN, h.serviceARN)
		if err != nil {
			return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get ECS task %s", id), err)
		}
		for _, task := range tasks {
			if getTaskIDFromARN(task.TaskARN) == id || task.TaskARN == id {
				return &ECSTaskResource{
					task:   task,
					region: h.region,
				}, nil
			}
		}
		return nil, NewHandlerError("NOT_FOUND", fmt.Sprintf("task %s not found", id), nil)
	}

	task, err := h.client.GetTask(ctx, h.clusterARN, taskARN)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get ECS task %s", id), err)
	}

	return &ECSTaskResource{
		task:   *task,
		region: h.region,
	}, nil
}

func (h *ECSTasksHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	resource, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	taskResource, ok := resource.(*ECSTaskResource)
	if !ok {
		return nil, NewHandlerError("DESCRIBE_FAILED", "failed to convert resource to task", nil)
	}

	task := taskResource.task
	details := make(map[string]interface{})

	// Basic info
	details["Task"] = map[string]interface{}{
		"TaskID":            getTaskIDFromARN(task.TaskARN),
		"TaskArn":           task.TaskARN,
		"ClusterArn":        task.ClusterARN,
		"TaskDefinitionArn": task.TaskDefinitionARN,
	}

	// Status
	details["Status"] = map[string]interface{}{
		"LastStatus":           task.LastStatus,
		"DesiredStatus":        task.DesiredStatus,
		"LaunchType":           task.LaunchType,
		"PlatformVersion":      task.PlatformVersion,
		"EnableExecuteCommand": task.EnableExecuteCommand,
	}

	// Containers
	if len(task.Containers) > 0 {
		containerList := make([]map[string]interface{}, 0, len(task.Containers))
		for _, container := range task.Containers {
			c := map[string]interface{}{
				"Name":       container.Name,
				"Status":     container.LastStatus,
				"Image":      container.Image,
				"RuntimeId":  container.RuntimeId,
			}
			if container.HealthStatus != "" {
				c["HealthStatus"] = container.HealthStatus
			}
			containerList = append(containerList, c)
		}
		details["Containers"] = containerList
	}

	// Timestamps
	if task.CreatedAt != "" {
		details["CreatedAt"] = task.CreatedAt
	}
	if task.StartedAt != "" {
		details["StartedAt"] = task.StartedAt
	}

	// Tags
	if len(task.Tags) > 0 {
		details["Tags"] = task.Tags
	}

	return details, nil
}

func (h *ECSTasksHandler) Actions() []Action {
	return []Action{
		{Key: "x", Name: "exec", Description: "exec shell"},
	}
}

func (h *ECSTasksHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	if action != "exec" {
		return ErrNotSupported
	}

	// Get task details
	resource, err := h.Get(ctx, resourceID)
	if err != nil {
		return err
	}

	taskResource, ok := resource.(*ECSTaskResource)
	if !ok {
		return fmt.Errorf("failed to convert resource to task")
	}

	task := taskResource.task

	// Pre-flight checks
	if !task.EnableExecuteCommand {
		return fmt.Errorf("execute command not enabled for this task")
	}

	if task.LastStatus != "RUNNING" {
		return fmt.Errorf("task is not running (status: %s)", task.LastStatus)
	}

	// Filter to running containers
	runningContainers := make([]messages.ECSContainer, 0)
	for _, container := range task.Containers {
		if container.LastStatus == "RUNNING" {
			runningContainers = append(runningContainers, messages.ECSContainer{
				Name:         container.Name,
				RuntimeId:    container.RuntimeId,
				Status:       container.LastStatus,
				HealthStatus: container.HealthStatus,
			})
		}
	}

	if len(runningContainers) == 0 {
		return fmt.Errorf("no running containers found in this task")
	}

	// Return exec request message
	return &ExecRequestAction{
		ClusterARN: h.clusterARN,
		TaskARN:    task.TaskARN,
		Containers: runningContainers,
	}
}

// ExecRequestAction is returned by ExecuteAction to trigger exec
type ExecRequestAction struct {
	ClusterARN string
	TaskARN    string
	Containers []messages.ECSContainer
}

func (a *ExecRequestAction) Error() string {
	return fmt.Sprintf("exec request for task %s", getTaskIDFromARN(a.TaskARN))
}

func (a *ExecRequestAction) IsActionMsg() {}

// ECSTaskResource implements Resource interface for ECS tasks
type ECSTaskResource struct {
	task   ecsadapter.Task
	region string
}

func (r *ECSTaskResource) GetID() string   { return getTaskIDFromARN(r.task.TaskARN) }
func (r *ECSTaskResource) GetName() string { return getTaskIDFromARN(r.task.TaskARN) }
func (r *ECSTaskResource) GetARN() string  { return r.task.TaskARN }
func (r *ECSTaskResource) GetType() string   { return "ecs:tasks" }
func (r *ECSTaskResource) GetRegion() string { return r.region }

func (r *ECSTaskResource) GetCreatedAt() time.Time {
	if r.task.CreatedAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", r.task.CreatedAt)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func (r *ECSTaskResource) GetTags() map[string]string {
	return r.task.Tags
}

func (r *ECSTaskResource) ToTableRow() []string {
	execEnabled := "No"
	if r.task.EnableExecuteCommand {
		execEnabled = "Yes"
	}

	return []string{
		getTaskIDFromARN(r.task.TaskARN),
		r.task.LastStatus,
		r.task.LaunchType,
		fmt.Sprintf("%d", len(r.task.Containers)),
		execEnabled,
		r.task.StartedAt,
	}
}

func (r *ECSTaskResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"TaskID":              getTaskIDFromARN(r.task.TaskARN),
		"TaskArn":             r.task.TaskARN,
		"ClusterArn":          r.task.ClusterARN,
		"TaskDefinitionArn":   r.task.TaskDefinitionARN,
		"LastStatus":          r.task.LastStatus,
		"DesiredStatus":       r.task.DesiredStatus,
		"LaunchType":          r.task.LaunchType,
		"EnableExecuteCommand": r.task.EnableExecuteCommand,
	}
}

// Helper function to extract task ID from ARN
// e.g., "arn:aws:ecs:us-east-1:123456789012:task/cluster-name/abc123def456" -> "abc123def456"
func getTaskIDFromARN(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}
