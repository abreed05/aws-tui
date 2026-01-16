package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// TasksClient wraps the ECS client for task operations
type TasksClient struct {
	client *ecs.Client
}

// NewTasksClient creates a new ECS tasks client
func NewTasksClient(client *ecs.Client) *TasksClient {
	return &TasksClient{client: client}
}

// Task represents an ECS task
type Task struct {
	TaskARN              string
	ClusterARN           string
	TaskDefinitionARN    string
	LastStatus           string
	DesiredStatus        string
	LaunchType           string
	PlatformVersion      string
	EnableExecuteCommand bool
	Containers           []Container
	CreatedAt            string
	StartedAt            string
	Tags                 map[string]string
}

// Container represents a container in a task
type Container struct {
	Name         string
	RuntimeId    string
	LastStatus   string
	HealthStatus string
	Image        string
}

// ListTasks lists tasks in a cluster, optionally filtered by service
func (c *TasksClient) ListTasks(ctx context.Context, clusterARN string, serviceARN string) ([]Task, error) {
	var taskARNs []string
	var nextToken *string

	// List task ARNs
	for {
		input := &ecs.ListTasksInput{
			Cluster:   aws.String(clusterARN),
			NextToken: nextToken,
		}

		// Filter by service if specified
		if serviceARN != "" {
			input.ServiceName = aws.String(serviceARN)
		}

		output, err := c.client.ListTasks(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}

		taskARNs = append(taskARNs, output.TaskArns...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	if len(taskARNs) == 0 {
		return []Task{}, nil
	}

	// Describe tasks in batches of 100 (API limit)
	var tasks []Task
	for i := 0; i < len(taskARNs); i += 100 {
		end := i + 100
		if end > len(taskARNs) {
			end = len(taskARNs)
		}

		describeOutput, err := c.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(clusterARN),
			Tasks:   taskARNs[i:end],
			Include: []types.TaskField{types.TaskFieldTags},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe tasks: %w", err)
		}

		for _, task := range describeOutput.Tasks {
			tasks = append(tasks, convertTask(task))
		}
	}

	return tasks, nil
}

// GetTask gets a single task by ARN
func (c *TasksClient) GetTask(ctx context.Context, clusterARN, taskARN string) (*Task, error) {
	output, err := c.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterARN),
		Tasks:   []string{taskARN},
		Include: []types.TaskField{types.TaskFieldTags},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task %s: %w", taskARN, err)
	}

	if len(output.Tasks) == 0 {
		return nil, fmt.Errorf("task %s not found", taskARN)
	}

	task := convertTask(output.Tasks[0])
	return &task, nil
}

func convertTask(task types.Task) Task {
	result := Task{
		TaskARN:              aws.ToString(task.TaskArn),
		ClusterARN:           aws.ToString(task.ClusterArn),
		TaskDefinitionARN:    aws.ToString(task.TaskDefinitionArn),
		LastStatus:           aws.ToString(task.LastStatus),
		DesiredStatus:        aws.ToString(task.DesiredStatus),
		LaunchType:           string(task.LaunchType),
		PlatformVersion:      aws.ToString(task.PlatformVersion),
		EnableExecuteCommand: task.EnableExecuteCommand,
		Containers:           make([]Container, 0, len(task.Containers)),
		Tags:                 make(map[string]string),
	}

	// Convert containers
	for _, container := range task.Containers {
		c := Container{
			Name:         aws.ToString(container.Name),
			RuntimeId:    aws.ToString(container.RuntimeId),
			LastStatus:   aws.ToString(container.LastStatus),
			Image:        aws.ToString(container.Image),
		}
		if container.HealthStatus != "" {
			c.HealthStatus = string(container.HealthStatus)
		}
		result.Containers = append(result.Containers, c)
	}

	// Convert timestamps
	if task.CreatedAt != nil {
		result.CreatedAt = task.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if task.StartedAt != nil {
		result.StartedAt = task.StartedAt.Format("2006-01-02 15:04:05")
	}

	// Convert tags
	for _, tag := range task.Tags {
		result.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return result
}
