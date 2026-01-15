package ec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// VPCsClient wraps the EC2 client for VPC operations
type VPCsClient struct {
	client *ec2.Client
}

// NewVPCsClient creates a new VPCs client
func NewVPCsClient(client *ec2.Client) *VPCsClient {
	return &VPCsClient{client: client}
}

// VPC represents a VPC with its metadata
type VPC struct {
	VpcID           string
	Name            string
	CidrBlock       string
	State           string
	IsDefault       bool
	InstanceTenancy string
	DhcpOptionsID   string
	OwnerID         string
	SubnetCount     int
	Tags            map[string]string
}

// Subnet represents a VPC subnet
type Subnet struct {
	SubnetID         string
	Name             string
	VpcID            string
	CidrBlock        string
	AvailabilityZone string
	State            string
	AvailableIPs     int32
	IsDefault        bool
	MapPublicIP      bool
	Tags             map[string]string
}

// ListVPCs lists all VPCs
func (c *VPCsClient) ListVPCs(ctx context.Context) ([]VPC, error) {
	var vpcs []VPC
	var nextToken *string

	for {
		input := &ec2.DescribeVpcsInput{
			NextToken: nextToken,
		}

		output, err := c.client.DescribeVpcs(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe VPCs: %w", err)
		}

		for _, vpc := range output.Vpcs {
			vpcs = append(vpcs, convertVPC(vpc))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	// Get subnet counts for each VPC
	subnetCounts, _ := c.getSubnetCounts(ctx)
	for i := range vpcs {
		if count, ok := subnetCounts[vpcs[i].VpcID]; ok {
			vpcs[i].SubnetCount = count
		}
	}

	return vpcs, nil
}

// GetVPC gets a single VPC by ID
func (c *VPCsClient) GetVPC(ctx context.Context, vpcID string) (*VPC, error) {
	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}

	output, err := c.client.DescribeVpcs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC %s: %w", vpcID, err)
	}

	if len(output.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC %s not found", vpcID)
	}

	vpc := convertVPC(output.Vpcs[0])

	// Get subnet count
	subnetCounts, _ := c.getSubnetCounts(ctx)
	if count, ok := subnetCounts[vpc.VpcID]; ok {
		vpc.SubnetCount = count
	}

	return &vpc, nil
}

// ListSubnets lists all subnets, optionally filtered by VPC
func (c *VPCsClient) ListSubnets(ctx context.Context, vpcID string) ([]Subnet, error) {
	var subnets []Subnet
	var nextToken *string

	for {
		input := &ec2.DescribeSubnetsInput{
			NextToken: nextToken,
		}

		if vpcID != "" {
			input.Filters = []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{vpcID},
				},
			}
		}

		output, err := c.client.DescribeSubnets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnets: %w", err)
		}

		for _, subnet := range output.Subnets {
			subnets = append(subnets, convertSubnet(subnet))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return subnets, nil
}

func (c *VPCsClient) getSubnetCounts(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int)

	subnets, err := c.ListSubnets(ctx, "")
	if err != nil {
		return counts, err
	}

	for _, subnet := range subnets {
		counts[subnet.VpcID]++
	}

	return counts, nil
}

func convertVPC(vpc types.Vpc) VPC {
	result := VPC{
		VpcID:           aws.ToString(vpc.VpcId),
		CidrBlock:       aws.ToString(vpc.CidrBlock),
		State:           string(vpc.State),
		IsDefault:       vpc.IsDefault != nil && *vpc.IsDefault,
		InstanceTenancy: string(vpc.InstanceTenancy),
		DhcpOptionsID:   aws.ToString(vpc.DhcpOptionsId),
		OwnerID:         aws.ToString(vpc.OwnerId),
		Tags:            make(map[string]string),
	}

	for _, tag := range vpc.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		result.Tags[key] = value
		if key == "Name" {
			result.Name = value
		}
	}

	return result
}

func convertSubnet(subnet types.Subnet) Subnet {
	result := Subnet{
		SubnetID:         aws.ToString(subnet.SubnetId),
		VpcID:            aws.ToString(subnet.VpcId),
		CidrBlock:        aws.ToString(subnet.CidrBlock),
		AvailabilityZone: aws.ToString(subnet.AvailabilityZone),
		State:            string(subnet.State),
		IsDefault:        subnet.DefaultForAz != nil && *subnet.DefaultForAz,
		MapPublicIP:      subnet.MapPublicIpOnLaunch != nil && *subnet.MapPublicIpOnLaunch,
		Tags:             make(map[string]string),
	}

	if subnet.AvailableIpAddressCount != nil {
		result.AvailableIPs = *subnet.AvailableIpAddressCount
	}

	for _, tag := range subnet.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		result.Tags[key] = value
		if key == "Name" {
			result.Name = value
		}
	}

	return result
}
