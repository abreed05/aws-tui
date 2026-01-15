package ec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SecurityGroupsClient wraps the EC2 client for security group operations
type SecurityGroupsClient struct {
	client *ec2.Client
}

// NewSecurityGroupsClient creates a new security groups client
func NewSecurityGroupsClient(client *ec2.Client) *SecurityGroupsClient {
	return &SecurityGroupsClient{client: client}
}

// SecurityGroup represents a security group with its rules
type SecurityGroup struct {
	GroupID       string
	GroupName     string
	Description   string
	VpcID         string
	OwnerID       string
	InboundRules  []SecurityGroupRule
	OutboundRules []SecurityGroupRule
	Tags          map[string]string
}

// SecurityGroupRule represents an inbound or outbound rule
type SecurityGroupRule struct {
	Protocol    string
	FromPort    int32
	ToPort      int32
	IPRanges    []string
	IPv6Ranges  []string
	PrefixLists []string
	SGSources   []string // Security group sources
	Description string
}

// ListSecurityGroups lists all security groups
func (c *SecurityGroupsClient) ListSecurityGroups(ctx context.Context) ([]SecurityGroup, error) {
	var securityGroups []SecurityGroup
	var nextToken *string

	for {
		input := &ec2.DescribeSecurityGroupsInput{
			NextToken: nextToken,
		}

		output, err := c.client.DescribeSecurityGroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe security groups: %w", err)
		}

		for _, sg := range output.SecurityGroups {
			securityGroups = append(securityGroups, convertSecurityGroup(sg))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return securityGroups, nil
}

// GetSecurityGroup gets a single security group by ID
func (c *SecurityGroupsClient) GetSecurityGroup(ctx context.Context, groupID string) (*SecurityGroup, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
	}

	output, err := c.client.DescribeSecurityGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe security group %s: %w", groupID, err)
	}

	if len(output.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group %s not found", groupID)
	}

	sg := convertSecurityGroup(output.SecurityGroups[0])
	return &sg, nil
}

// GetSecurityGroupRules gets detailed rules for a security group
func (c *SecurityGroupsClient) GetSecurityGroupRules(ctx context.Context, groupID string) ([]types.SecurityGroupRule, error) {
	var rules []types.SecurityGroupRule
	var nextToken *string

	for {
		input := &ec2.DescribeSecurityGroupRulesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("group-id"),
					Values: []string{groupID},
				},
			},
			NextToken: nextToken,
		}

		output, err := c.client.DescribeSecurityGroupRules(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe security group rules: %w", err)
		}

		rules = append(rules, output.SecurityGroupRules...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return rules, nil
}

func convertSecurityGroup(sg types.SecurityGroup) SecurityGroup {
	result := SecurityGroup{
		GroupID:     aws.ToString(sg.GroupId),
		GroupName:   aws.ToString(sg.GroupName),
		Description: aws.ToString(sg.Description),
		VpcID:       aws.ToString(sg.VpcId),
		OwnerID:     aws.ToString(sg.OwnerId),
		Tags:        make(map[string]string),
	}

	// Convert tags
	for _, tag := range sg.Tags {
		result.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	// Convert inbound rules
	for _, perm := range sg.IpPermissions {
		result.InboundRules = append(result.InboundRules, convertPermission(perm))
	}

	// Convert outbound rules
	for _, perm := range sg.IpPermissionsEgress {
		result.OutboundRules = append(result.OutboundRules, convertPermission(perm))
	}

	return result
}

func convertPermission(perm types.IpPermission) SecurityGroupRule {
	rule := SecurityGroupRule{
		Protocol: aws.ToString(perm.IpProtocol),
	}

	if perm.FromPort != nil {
		rule.FromPort = *perm.FromPort
	}
	if perm.ToPort != nil {
		rule.ToPort = *perm.ToPort
	}

	// IP ranges
	for _, ipRange := range perm.IpRanges {
		cidr := aws.ToString(ipRange.CidrIp)
		if ipRange.Description != nil {
			cidr += " (" + aws.ToString(ipRange.Description) + ")"
		}
		rule.IPRanges = append(rule.IPRanges, cidr)
	}

	// IPv6 ranges
	for _, ipv6Range := range perm.Ipv6Ranges {
		cidr := aws.ToString(ipv6Range.CidrIpv6)
		if ipv6Range.Description != nil {
			cidr += " (" + aws.ToString(ipv6Range.Description) + ")"
		}
		rule.IPv6Ranges = append(rule.IPv6Ranges, cidr)
	}

	// Prefix lists
	for _, pl := range perm.PrefixListIds {
		rule.PrefixLists = append(rule.PrefixLists, aws.ToString(pl.PrefixListId))
	}

	// Security group sources
	for _, sgSource := range perm.UserIdGroupPairs {
		source := aws.ToString(sgSource.GroupId)
		if sgSource.GroupName != nil {
			source = aws.ToString(sgSource.GroupName) + " (" + source + ")"
		}
		rule.SGSources = append(rule.SGSources, source)
	}

	return rule
}
