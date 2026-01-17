package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// LogsClient wraps the CloudWatch Logs client
type LogsClient struct {
	client *cloudwatchlogs.Client
}

// NewLogsClient creates a new logs client wrapper
func NewLogsClient(client *cloudwatchlogs.Client) *LogsClient {
	return &LogsClient{client: client}
}

// LogGroup represents a CloudWatch log group
type LogGroup struct {
	Name            string
	Arn             string
	CreatedAt       time.Time
	RetentionInDays int32
	StoredBytes     int64
	Tags            map[string]string
}

// LogStream represents a CloudWatch log stream
type LogStream struct {
	Name              string
	CreatedAt         time.Time
	FirstEventTime    time.Time
	LastEventTime     time.Time
	LastIngestionTime time.Time
	UploadSequenceToken string
	StoredBytes       int64
}

// LogEvent represents a CloudWatch log event
type LogEvent struct {
	Timestamp     time.Time
	Message       string
	IngestionTime time.Time
}

// ListLogGroups lists all log groups with pagination
func (c *LogsClient) ListLogGroups(ctx context.Context) ([]LogGroup, error) {
	var logGroups []LogGroup

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(c.client, &cloudwatchlogs.DescribeLogGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", err)
		}

		for _, group := range page.LogGroups {
			lg := LogGroup{
				Name:            aws.ToString(group.LogGroupName),
				Arn:             aws.ToString(group.Arn),
				CreatedAt:       timeFromMillis(group.CreationTime),
				RetentionInDays: aws.ToInt32(group.RetentionInDays),
				StoredBytes:     aws.ToInt64(group.StoredBytes),
				Tags:            make(map[string]string), // Tags require separate API call
			}

			logGroups = append(logGroups, lg)
		}
	}

	return logGroups, nil
}

// GetLogGroup gets details of a specific log group
func (c *LogsClient) GetLogGroup(ctx context.Context, groupName string) (*LogGroup, error) {
	output, err := c.client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(groupName),
		Limit:              aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe log group %s: %w", groupName, err)
	}

	if len(output.LogGroups) == 0 {
		return nil, fmt.Errorf("log group %s not found", groupName)
	}

	group := output.LogGroups[0]

	// Make sure we got an exact match
	if aws.ToString(group.LogGroupName) != groupName {
		return nil, fmt.Errorf("log group %s not found", groupName)
	}

	lg := &LogGroup{
		Name:            aws.ToString(group.LogGroupName),
		Arn:             aws.ToString(group.Arn),
		CreatedAt:       timeFromMillis(group.CreationTime),
		RetentionInDays: aws.ToInt32(group.RetentionInDays),
		StoredBytes:     aws.ToInt64(group.StoredBytes),
		Tags:            make(map[string]string), // Tags require separate API call
	}

	return lg, nil
}

// ListLogStreams lists all log streams in a log group
func (c *LogsClient) ListLogStreams(ctx context.Context, groupName string) ([]LogStream, error) {
	var logStreams []LogStream

	paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(c.client, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(groupName),
		OrderBy:      types.OrderByLastEventTime,
		Descending:   aws.Bool(true),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log streams for group %s: %w", groupName, err)
		}

		for _, stream := range page.LogStreams {
			ls := LogStream{
				Name:                aws.ToString(stream.LogStreamName),
				CreatedAt:           timeFromMillis(stream.CreationTime),
				LastIngestionTime:   timeFromMillis(stream.LastIngestionTime),
				StoredBytes:         aws.ToInt64(stream.StoredBytes),
				UploadSequenceToken: aws.ToString(stream.UploadSequenceToken),
			}

			if stream.FirstEventTimestamp != nil {
				ls.FirstEventTime = timeFromMillis(stream.FirstEventTimestamp)
			}
			if stream.LastEventTimestamp != nil {
				ls.LastEventTime = timeFromMillis(stream.LastEventTimestamp)
			}

			logStreams = append(logStreams, ls)
		}
	}

	return logStreams, nil
}

// GetLogEvents gets log events from a specific log stream
func (c *LogsClient) GetLogEvents(ctx context.Context, groupName, streamName string, limit int) ([]LogEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	output, err := c.client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(streamName),
		Limit:         aws.Int32(int32(limit)),
		StartFromHead: aws.Bool(false), // Get most recent events
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log events for stream %s in group %s: %w", streamName, groupName, err)
	}

	logEvents := make([]LogEvent, 0, len(output.Events))
	for _, event := range output.Events {
		logEvents = append(logEvents, LogEvent{
			Timestamp:     timeFromMillis(event.Timestamp),
			Message:       aws.ToString(event.Message),
			IngestionTime: timeFromMillis(event.IngestionTime),
		})
	}

	return logEvents, nil
}

// Helper function to convert milliseconds to time.Time
func timeFromMillis(millis *int64) time.Time {
	if millis == nil || *millis == 0 {
		return time.Time{}
	}
	return time.UnixMilli(*millis)
}
