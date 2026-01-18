package ec2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// InstancesClient wraps the EC2 client for instance operations
type InstancesClient struct {
	client *ec2.Client
}

// NewInstancesClient creates a new instances client
func NewInstancesClient(client *ec2.Client) *InstancesClient {
	return &InstancesClient{client: client}
}

// Instance represents an EC2 instance with its metadata
type Instance struct {
	InstanceID       string
	Name             string
	State            string
	InstanceType     string
	Platform         string
	PrivateIP        string
	PublicIP         string
	VpcID            string
	SubnetID         string
	AvailabilityZone string
	LaunchTime       time.Time
	ImageID          string
	KeyName          string
	SecurityGroups   []string
	IAMRole          string
	Tags             map[string]string
}

// ListInstances lists all EC2 instances
func (c *InstancesClient) ListInstances(ctx context.Context) ([]Instance, error) {
	var instances []Instance
	var nextToken *string

	for {
		input := &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		}

		output, err := c.client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range output.Reservations {
			for _, inst := range reservation.Instances {
				instances = append(instances, convertInstance(inst))
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return instances, nil
}

// GetInstance gets a single EC2 instance by ID
func (c *InstancesClient) GetInstance(ctx context.Context, instanceID string) (*Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	output, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance %s: %w", instanceID, err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	inst := convertInstance(output.Reservations[0].Instances[0])
	return &inst, nil
}

// GetInstanceStatus gets the status checks for an instance
func (c *InstancesClient) GetInstanceStatus(ctx context.Context, instanceID string) (*types.InstanceStatus, error) {
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceID},
	}

	output, err := c.client.DescribeInstanceStatus(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance status: %w", err)
	}

	if len(output.InstanceStatuses) == 0 {
		return nil, nil
	}

	return &output.InstanceStatuses[0], nil
}

func convertInstance(inst types.Instance) Instance {
	result := Instance{
		InstanceID:   aws.ToString(inst.InstanceId),
		State:        string(inst.State.Name),
		InstanceType: string(inst.InstanceType),
		PrivateIP:    aws.ToString(inst.PrivateIpAddress),
		PublicIP:     aws.ToString(inst.PublicIpAddress),
		VpcID:        aws.ToString(inst.VpcId),
		SubnetID:     aws.ToString(inst.SubnetId),
		ImageID:      aws.ToString(inst.ImageId),
		KeyName:      aws.ToString(inst.KeyName),
		Tags:         make(map[string]string),
	}

	if inst.Placement != nil {
		result.AvailabilityZone = aws.ToString(inst.Placement.AvailabilityZone)
	}

	if inst.LaunchTime != nil {
		result.LaunchTime = *inst.LaunchTime
	}

	if inst.Platform != "" {
		result.Platform = string(inst.Platform)
	} else {
		result.Platform = "Linux/UNIX"
	}

	if inst.IamInstanceProfile != nil {
		result.IAMRole = aws.ToString(inst.IamInstanceProfile.Arn)
	}

	// Extract security groups
	for _, sg := range inst.SecurityGroups {
		sgName := aws.ToString(sg.GroupName)
		if sgName != "" {
			result.SecurityGroups = append(result.SecurityGroups, sgName)
		}
	}

	// Extract tags including Name
	for _, tag := range inst.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		result.Tags[key] = value
		if key == "Name" {
			result.Name = value
		}
	}

	return result
}

// StartInstance starts a stopped EC2 instance
func (c *InstancesClient) StartInstance(ctx context.Context, instanceID string) error {
	_, err := c.client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return nil
}

// StopInstance stops a running EC2 instance
func (c *InstancesClient) StopInstance(ctx context.Context, instanceID string) error {
	_, err := c.client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}
	return nil
}

// RebootInstance reboots an EC2 instance
func (c *InstancesClient) RebootInstance(ctx context.Context, instanceID string) error {
	_, err := c.client.RebootInstances(ctx, &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}
	return nil
}

// GetInstanceConnectionInfo retrieves connection information for an instance
func (c *InstancesClient) GetInstanceConnectionInfo(ctx context.Context, instanceID string) (map[string]interface{}, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	inst := result.Reservations[0].Instances[0]

	info := make(map[string]interface{})
	info["InstanceId"] = aws.ToString(inst.InstanceId)
	info["State"] = string(inst.State.Name)

	if inst.PublicIpAddress != nil {
		info["PublicIP"] = aws.ToString(inst.PublicIpAddress)
	}

	if inst.PrivateIpAddress != nil {
		info["PrivateIP"] = aws.ToString(inst.PrivateIpAddress)
	}

	if inst.PublicDnsName != nil && aws.ToString(inst.PublicDnsName) != "" {
		info["PublicDNS"] = aws.ToString(inst.PublicDnsName)
	}

	if inst.PrivateDnsName != nil && aws.ToString(inst.PrivateDnsName) != "" {
		info["PrivateDNS"] = aws.ToString(inst.PrivateDnsName)
	}

	if inst.KeyName != nil {
		info["KeyName"] = aws.ToString(inst.KeyName)
	}

	// Platform information
	platform := "Linux/Unix"
	if inst.Platform == types.PlatformValuesWindows {
		platform = "Windows"
	}
	info["Platform"] = platform

	// SSH/RDP command examples
	if inst.PublicIpAddress != nil {
		publicIP := aws.ToString(inst.PublicIpAddress)
		if platform == "Windows" {
			info["ConnectionCommand"] = fmt.Sprintf("RDP to: %s:3389", publicIP)
		} else {
			keyName := aws.ToString(inst.KeyName)
			if keyName != "" {
				info["ConnectionCommand"] = fmt.Sprintf("ssh -i ~/.ssh/%s.pem ec2-user@%s", keyName, publicIP)
			} else {
				info["ConnectionCommand"] = fmt.Sprintf("ssh ec2-user@%s", publicIP)
			}
		}
	}

	return info, nil
}
