package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ClustersClient wraps the ECS client for cluster operations
type ClustersClient struct {
	client *ecs.Client
}

// NewClustersClient creates a new ECS clusters client
func NewClustersClient(client *ecs.Client) *ClustersClient {
	return &ClustersClient{client: client}
}

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterARN                    string
	ClusterName                   string
	Status                        string
	RunningTasksCount             int32
	PendingTasksCount             int32
	ActiveServicesCount           int32
	RegisteredContainerInstances  int32
	CapacityProviders             []string
	Tags                          map[string]string
}

// Service represents an ECS service
type Service struct {
	ServiceARN        string
	ServiceName       string
	ClusterARN        string
	Status            string
	DesiredCount      int32
	RunningCount      int32
	PendingCount      int32
	LaunchType        string
	TaskDefinition    string
	CreatedAt         string
	Tags              map[string]string
}

// ListClusters lists all ECS clusters
func (c *ClustersClient) ListClusters(ctx context.Context) ([]Cluster, error) {
	var clusterARNs []string
	var nextToken *string

	// First, list all cluster ARNs
	for {
		input := &ecs.ListClustersInput{
			NextToken: nextToken,
		}

		output, err := c.client.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		clusterARNs = append(clusterARNs, output.ClusterArns...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	if len(clusterARNs) == 0 {
		return []Cluster{}, nil
	}

	// Then describe all clusters to get details
	describeOutput, err := c.client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: clusterARNs,
		Include:  []types.ClusterField{types.ClusterFieldTags},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe clusters: %w", err)
	}

	clusters := make([]Cluster, 0, len(describeOutput.Clusters))
	for _, cluster := range describeOutput.Clusters {
		clusters = append(clusters, convertCluster(cluster))
	}

	return clusters, nil
}

// GetCluster gets a single ECS cluster by name or ARN
func (c *ClustersClient) GetCluster(ctx context.Context, clusterID string) (*Cluster, error) {
	output, err := c.client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterID},
		Include:  []types.ClusterField{types.ClusterFieldTags},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster %s: %w", clusterID, err)
	}

	if len(output.Clusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster := convertCluster(output.Clusters[0])
	return &cluster, nil
}

// ListServices lists services in a cluster
func (c *ClustersClient) ListServices(ctx context.Context, clusterARN string) ([]Service, error) {
	var serviceARNs []string
	var nextToken *string

	for {
		input := &ecs.ListServicesInput{
			Cluster:   aws.String(clusterARN),
			NextToken: nextToken,
		}

		output, err := c.client.ListServices(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}

		serviceARNs = append(serviceARNs, output.ServiceArns...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	if len(serviceARNs) == 0 {
		return []Service{}, nil
	}

	// Describe services in batches of 10 (API limit)
	var services []Service
	for i := 0; i < len(serviceARNs); i += 10 {
		end := i + 10
		if end > len(serviceARNs) {
			end = len(serviceARNs)
		}

		describeOutput, err := c.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterARN),
			Services: serviceARNs[i:end],
			Include:  []types.ServiceField{types.ServiceFieldTags},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe services: %w", err)
		}

		for _, svc := range describeOutput.Services {
			services = append(services, convertService(svc))
		}
	}

	return services, nil
}

func convertCluster(cluster types.Cluster) Cluster {
	result := Cluster{
		ClusterARN:                   aws.ToString(cluster.ClusterArn),
		ClusterName:                  aws.ToString(cluster.ClusterName),
		Status:                       aws.ToString(cluster.Status),
		RunningTasksCount:            cluster.RunningTasksCount,
		PendingTasksCount:            cluster.PendingTasksCount,
		ActiveServicesCount:          cluster.ActiveServicesCount,
		RegisteredContainerInstances: cluster.RegisteredContainerInstancesCount,
		CapacityProviders:            cluster.CapacityProviders,
		Tags:                         make(map[string]string),
	}

	for _, tag := range cluster.Tags {
		result.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return result
}

func convertService(svc types.Service) Service {
	result := Service{
		ServiceARN:     aws.ToString(svc.ServiceArn),
		ServiceName:    aws.ToString(svc.ServiceName),
		ClusterARN:     aws.ToString(svc.ClusterArn),
		Status:         aws.ToString(svc.Status),
		DesiredCount:   svc.DesiredCount,
		RunningCount:   svc.RunningCount,
		PendingCount:   svc.PendingCount,
		LaunchType:     string(svc.LaunchType),
		TaskDefinition: aws.ToString(svc.TaskDefinition),
		Tags:           make(map[string]string),
	}

	if svc.CreatedAt != nil {
		result.CreatedAt = svc.CreatedAt.Format("2006-01-02 15:04:05")
	}

	for _, tag := range svc.Tags {
		result.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return result
}
