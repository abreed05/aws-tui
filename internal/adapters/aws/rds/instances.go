package rds

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// InstancesClient wraps the RDS client for instance operations
type InstancesClient struct {
	client *rds.Client
}

// NewInstancesClient creates a new RDS instances client
func NewInstancesClient(client *rds.Client) *InstancesClient {
	return &InstancesClient{client: client}
}

// DBInstance represents an RDS instance
type DBInstance struct {
	DBInstanceID            string
	DBInstanceClass         string
	Engine                  string
	EngineVersion           string
	Status                  string
	Endpoint                string
	Port                    int32
	MasterUsername          string
	DBName                  string
	AllocatedStorage        int32
	StorageType             string
	StorageEncrypted        bool
	MultiAZ                 bool
	AvailabilityZone        string
	VpcID                   string
	PubliclyAccessible      bool
	AutoMinorVersionUpgrade bool
	BackupRetentionPeriod   int32
	CreatedTime             time.Time
	Tags                    map[string]string
}

// ListDBInstances lists all RDS instances
func (c *InstancesClient) ListDBInstances(ctx context.Context) ([]DBInstance, error) {
	var instances []DBInstance
	var marker *string

	for {
		input := &rds.DescribeDBInstancesInput{
			Marker: marker,
		}

		output, err := c.client.DescribeDBInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances: %w", err)
		}

		for _, db := range output.DBInstances {
			instances = append(instances, convertDBInstance(db))
		}

		if output.Marker == nil {
			break
		}
		marker = output.Marker
	}

	return instances, nil
}

// GetDBInstance gets a single RDS instance by ID
func (c *InstancesClient) GetDBInstance(ctx context.Context, dbInstanceID string) (*DBInstance, error) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceID),
	}

	output, err := c.client.DescribeDBInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB instance %s: %w", dbInstanceID, err)
	}

	if len(output.DBInstances) == 0 {
		return nil, fmt.Errorf("DB instance %s not found", dbInstanceID)
	}

	inst := convertDBInstance(output.DBInstances[0])
	return &inst, nil
}

func convertDBInstance(db types.DBInstance) DBInstance {
	result := DBInstance{
		DBInstanceID:            aws.ToString(db.DBInstanceIdentifier),
		DBInstanceClass:         aws.ToString(db.DBInstanceClass),
		Engine:                  aws.ToString(db.Engine),
		EngineVersion:           aws.ToString(db.EngineVersion),
		Status:                  aws.ToString(db.DBInstanceStatus),
		MasterUsername:          aws.ToString(db.MasterUsername),
		DBName:                  aws.ToString(db.DBName),
		StorageType:             aws.ToString(db.StorageType),
		AvailabilityZone:        aws.ToString(db.AvailabilityZone),
		Tags:                    make(map[string]string),
	}

	if db.Endpoint != nil {
		result.Endpoint = aws.ToString(db.Endpoint.Address)
		if db.Endpoint.Port != nil {
			result.Port = *db.Endpoint.Port
		}
	}

	if db.AllocatedStorage != nil {
		result.AllocatedStorage = *db.AllocatedStorage
	}

	if db.StorageEncrypted != nil {
		result.StorageEncrypted = *db.StorageEncrypted
	}

	if db.MultiAZ != nil {
		result.MultiAZ = *db.MultiAZ
	}

	if db.PubliclyAccessible != nil {
		result.PubliclyAccessible = *db.PubliclyAccessible
	}

	if db.AutoMinorVersionUpgrade != nil {
		result.AutoMinorVersionUpgrade = *db.AutoMinorVersionUpgrade
	}

	if db.BackupRetentionPeriod != nil {
		result.BackupRetentionPeriod = *db.BackupRetentionPeriod
	}

	if db.InstanceCreateTime != nil {
		result.CreatedTime = *db.InstanceCreateTime
	}

	if db.DBSubnetGroup != nil {
		result.VpcID = aws.ToString(db.DBSubnetGroup.VpcId)
	}

	for _, tag := range db.TagList {
		result.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return result
}
