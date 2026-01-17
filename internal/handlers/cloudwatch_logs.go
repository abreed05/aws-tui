package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	logsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/logs"
)

// NavigateToLogStreamsAction is returned by ExecuteAction to trigger navigation to log streams
type NavigateToLogStreamsAction struct {
	LogGroupName string
}

func (a *NavigateToLogStreamsAction) Error() string {
	return fmt.Sprintf("navigate to log streams for %s", a.LogGroupName)
}

func (a *NavigateToLogStreamsAction) IsActionMsg() {}

// CloudWatchLogsHandler handles CloudWatch log group resources
type CloudWatchLogsHandler struct {
	BaseHandler
	client *logsadapter.LogsClient
	region string
}

// NewCloudWatchLogsHandler creates a new CloudWatch Logs handler
func NewCloudWatchLogsHandler(logsClient *cloudwatchlogs.Client, region string) *CloudWatchLogsHandler {
	return &CloudWatchLogsHandler{
		client: logsadapter.NewLogsClient(logsClient),
		region: region,
	}
}

func (h *CloudWatchLogsHandler) ResourceType() string { return "logs:loggroups" }
func (h *CloudWatchLogsHandler) ResourceName() string { return "Log Groups" }
func (h *CloudWatchLogsHandler) ResourceIcon() string { return "📋" }
func (h *CloudWatchLogsHandler) ShortcutKey() string  { return "logs" }

func (h *CloudWatchLogsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Log Group Name", Width: 50, Sortable: true},
		{Title: "Retention (days)", Width: 15, Sortable: true},
		{Title: "Stored (MB)", Width: 12, Sortable: true},
		{Title: "Created", Width: 19, Sortable: true},
	}
}

func (h *CloudWatchLogsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	logGroups, err := h.client.ListLogGroups(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list log groups", err)
	}

	resources := make([]Resource, 0, len(logGroups))
	for _, lg := range logGroups {
		resource := &LogGroupResource{
			logGroup: lg,
			region:   h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(lg.Name)
			if !strings.Contains(name, filter) {
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

func (h *CloudWatchLogsHandler) Get(ctx context.Context, id string) (Resource, error) {
	lg, err := h.client.GetLogGroup(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get log group %s", id), err)
	}

	return &LogGroupResource{
		logGroup: *lg,
		region:   h.region,
	}, nil
}

func (h *CloudWatchLogsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	lg, err := h.client.GetLogGroup(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe log group %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["LogGroup"] = map[string]interface{}{
		"Name":      lg.Name,
		"Arn":       lg.Arn,
		"CreatedAt": lg.CreatedAt.Format(time.RFC3339),
	}

	// Configuration
	retention := "Never expire"
	if lg.RetentionInDays > 0 {
		retention = fmt.Sprintf("%d days", lg.RetentionInDays)
	}

	storedSize := formatBytesHelper(lg.StoredBytes)

	details["Configuration"] = map[string]interface{}{
		"Retention":  retention,
		"StoredSize": storedSize,
	}

	// Tags
	if len(lg.Tags) > 0 {
		details["Tags"] = lg.Tags
	}

	return details, nil
}

func (h *CloudWatchLogsHandler) Actions() []Action {
	return []Action{
		{Key: "s", Name: "streams", Description: "View log streams"},
	}
}

func (h *CloudWatchLogsHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	if action != "streams" {
		return ErrNotSupported
	}

	return &NavigateToLogStreamsAction{
		LogGroupName: resourceID,
	}
}

// LogGroupResource implements Resource interface for log groups
type LogGroupResource struct {
	logGroup logsadapter.LogGroup
	region   string
}

func (r *LogGroupResource) GetID() string   { return r.logGroup.Name }
func (r *LogGroupResource) GetName() string { return r.logGroup.Name }
func (r *LogGroupResource) GetARN() string  { return r.logGroup.Arn }
func (r *LogGroupResource) GetType() string { return "logs:loggroups" }
func (r *LogGroupResource) GetRegion() string { return r.region }
func (r *LogGroupResource) GetCreatedAt() time.Time { return r.logGroup.CreatedAt }
func (r *LogGroupResource) GetTags() map[string]string { return r.logGroup.Tags }

func (r *LogGroupResource) ToTableRow() []string {
	retention := "Never"
	if r.logGroup.RetentionInDays > 0 {
		retention = fmt.Sprintf("%d", r.logGroup.RetentionInDays)
	}

	storageMB := r.logGroup.StoredBytes / (1024 * 1024)
	if storageMB < 1 && r.logGroup.StoredBytes > 0 {
		storageMB = 1
	}

	created := "-"
	if !r.logGroup.CreatedAt.IsZero() {
		created = r.logGroup.CreatedAt.Format("2006-01-02 15:04:05")
	}

	return []string{
		r.logGroup.Name,
		retention,
		fmt.Sprintf("%d", storageMB),
		created,
	}
}

func (r *LogGroupResource) ToDetailMap() map[string]interface{} {
	retention := "Never expire"
	if r.logGroup.RetentionInDays > 0 {
		retention = fmt.Sprintf("%d days", r.logGroup.RetentionInDays)
	}

	return map[string]interface{}{
		"Name":       r.logGroup.Name,
		"Arn":        r.logGroup.Arn,
		"CreatedAt":  r.logGroup.CreatedAt.Format(time.RFC3339),
		"Retention":  retention,
		"StoredSize": formatBytesHelper(r.logGroup.StoredBytes),
	}
}

// Helper function to format bytes in human-readable format
func formatBytesHelper(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	case bytes < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	}
}
