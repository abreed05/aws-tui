package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	logsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/logs"
)

// CloudWatchLogStreamsHandler handles CloudWatch log stream resources for a specific log group
type CloudWatchLogStreamsHandler struct {
	BaseHandler
	client       *logsadapter.LogsClient
	region       string
	logGroupName string
}

// NewCloudWatchLogStreamsHandlerForGroup creates a new log streams handler for a specific log group
func NewCloudWatchLogStreamsHandlerForGroup(logsClient *cloudwatchlogs.Client, region, logGroupName string) *CloudWatchLogStreamsHandler {
	return &CloudWatchLogStreamsHandler{
		client:       logsadapter.NewLogsClient(logsClient),
		region:       region,
		logGroupName: logGroupName,
	}
}

func (h *CloudWatchLogStreamsHandler) ResourceType() string { return "logs:logstreams" }
func (h *CloudWatchLogStreamsHandler) ResourceName() string { return "Log Streams" }
func (h *CloudWatchLogStreamsHandler) ResourceIcon() string { return "📄" }
func (h *CloudWatchLogStreamsHandler) ShortcutKey() string  { return "log-streams" }

func (h *CloudWatchLogStreamsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Stream Name", Width: 45, Sortable: true},
		{Title: "Last Event", Width: 20, Sortable: true},
		{Title: "Stored (KB)", Width: 12, Sortable: true},
		{Title: "Created", Width: 19, Sortable: true},
	}
}

func (h *CloudWatchLogStreamsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	logStreams, err := h.client.ListLogStreams(ctx, h.logGroupName)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", fmt.Sprintf("failed to list log streams for group %s", h.logGroupName), err)
	}

	resources := make([]Resource, 0, len(logStreams))
	for _, ls := range logStreams {
		resource := &LogStreamResource{
			logStream:    ls,
			region:       h.region,
			logGroupName: h.logGroupName,
			client:       h.client,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(ls.Name)
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

func (h *CloudWatchLogStreamsHandler) Get(ctx context.Context, id string) (Resource, error) {
	logStreams, err := h.client.ListLogStreams(ctx, h.logGroupName)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get log stream %s", id), err)
	}

	for _, ls := range logStreams {
		if ls.Name == id {
			return &LogStreamResource{
				logStream:    ls,
				region:       h.region,
				logGroupName: h.logGroupName,
				client:       h.client,
			}, nil
		}
	}

	return nil, NewHandlerError("NOT_FOUND", fmt.Sprintf("log stream %s not found", id), nil)
}

func (h *CloudWatchLogStreamsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	resource, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	streamResource, ok := resource.(*LogStreamResource)
	if !ok {
		return nil, NewHandlerError("DESCRIBE_FAILED", "failed to convert resource to log stream", nil)
	}

	ls := streamResource.logStream
	details := make(map[string]interface{})

	// Basic info
	details["LogStream"] = map[string]interface{}{
		"Name":         ls.Name,
		"LogGroupName": h.logGroupName,
		"CreatedAt":    formatTimeHelper(ls.CreatedAt),
	}

	// Event times
	eventTimes := make(map[string]interface{})
	if !ls.FirstEventTime.IsZero() {
		eventTimes["FirstEventTime"] = formatTimeHelper(ls.FirstEventTime)
	}
	if !ls.LastEventTime.IsZero() {
		eventTimes["LastEventTime"] = formatTimeHelper(ls.LastEventTime)
	}
	if !ls.LastIngestionTime.IsZero() {
		eventTimes["LastIngestionTime"] = formatTimeHelper(ls.LastIngestionTime)
	}
	if len(eventTimes) > 0 {
		details["EventTimes"] = eventTimes
	}

	// Storage
	details["Storage"] = map[string]interface{}{
		"StoredSize": formatBytesHelper2(ls.StoredBytes),
	}

	// Fetch and display recent log events
	events, err := h.client.GetLogEvents(ctx, h.logGroupName, id, 100)
	if err == nil && len(events) > 0 {
		eventList := make([]map[string]interface{}, 0, len(events))
		for _, event := range events {
			eventList = append(eventList, map[string]interface{}{
				"Timestamp": event.Timestamp.Format("2006-01-02 15:04:05"),
				"Message":   event.Message,
			})
		}
		details["RecentEvents"] = eventList
	} else if err != nil {
		details["RecentEvents"] = fmt.Sprintf("Failed to load events: %v", err)
	} else {
		details["RecentEvents"] = "No events found"
	}

	return details, nil
}

func (h *CloudWatchLogStreamsHandler) Actions() []Action {
	return []Action{
		{Key: "e", Name: "events", Description: "View recent events"},
	}
}

func (h *CloudWatchLogStreamsHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	if action != "events" {
		return ErrNotSupported
	}

	// For events action, just show the describe view which includes recent events
	// The UI will handle showing the detail view
	return ErrNotSupported
}

// LogStreamResource implements Resource interface for log streams
type LogStreamResource struct {
	logStream    logsadapter.LogStream
	region       string
	logGroupName string
	client       *logsadapter.LogsClient
}

func (r *LogStreamResource) GetID() string   { return r.logStream.Name }
func (r *LogStreamResource) GetName() string { return r.logStream.Name }
func (r *LogStreamResource) GetARN() string {
	// CloudWatch log streams don't have ARNs, construct a pseudo-ARN for consistency
	return fmt.Sprintf("arn:aws:logs:%s:::log-group:%s:log-stream:%s",
		r.region, r.logGroupName, r.logStream.Name)
}
func (r *LogStreamResource) GetType() string { return "logs:logstreams" }
func (r *LogStreamResource) GetRegion() string { return r.region }
func (r *LogStreamResource) GetCreatedAt() time.Time { return r.logStream.CreatedAt }
func (r *LogStreamResource) GetTags() map[string]string { return nil }

func (r *LogStreamResource) ToTableRow() []string {
	lastEvent := "-"
	if !r.logStream.LastEventTime.IsZero() {
		lastEvent = r.logStream.LastEventTime.Format("2006-01-02 15:04:05")
	}

	storageKB := r.logStream.StoredBytes / 1024
	if storageKB < 1 && r.logStream.StoredBytes > 0 {
		storageKB = 1
	}

	created := "-"
	if !r.logStream.CreatedAt.IsZero() {
		created = r.logStream.CreatedAt.Format("2006-01-02 15:04:05")
	}

	return []string{
		r.logStream.Name,
		lastEvent,
		fmt.Sprintf("%d", storageKB),
		created,
	}
}

func (r *LogStreamResource) ToDetailMap() map[string]interface{} {
	details := map[string]interface{}{
		"Name":         r.logStream.Name,
		"LogGroupName": r.logGroupName,
		"CreatedAt":    formatTimeHelper(r.logStream.CreatedAt),
		"StoredSize":   formatBytesHelper2(r.logStream.StoredBytes),
	}

	if !r.logStream.FirstEventTime.IsZero() {
		details["FirstEventTime"] = formatTimeHelper(r.logStream.FirstEventTime)
	}
	if !r.logStream.LastEventTime.IsZero() {
		details["LastEventTime"] = formatTimeHelper(r.logStream.LastEventTime)
	}
	if !r.logStream.LastIngestionTime.IsZero() {
		details["LastIngestionTime"] = formatTimeHelper(r.logStream.LastIngestionTime)
	}

	return details
}

// Helper function to format time
func formatTimeHelper(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

// Helper function to format bytes in human-readable format
func formatBytesHelper2(bytes int64) string {
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
