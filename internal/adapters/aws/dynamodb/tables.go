package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TablesClient struct {
	client *dynamodb.Client
}

func NewTablesClient(client *dynamodb.Client) *TablesClient {
	return &TablesClient{client: client}
}

type Table struct {
	TableName            string
	TableArn             string
	TableStatus          string
	CreationDateTime     time.Time
	ItemCount            int64
	TableSizeBytes       int64
	BillingModeSummary   string
	KeySchema            []KeySchemaElement
	AttributeDefinitions []AttributeDefinition
	GlobalSecondaryIndexes []GlobalSecondaryIndex
	LocalSecondaryIndexes  []LocalSecondaryIndex
	StreamEnabled        bool
	StreamArn            string
	Tags                 map[string]string
}

type KeySchemaElement struct {
	AttributeName string
	KeyType       string
}

type AttributeDefinition struct {
	AttributeName string
	AttributeType string
}

type GlobalSecondaryIndex struct {
	IndexName      string
	KeySchema      []KeySchemaElement
	Projection     string
	IndexStatus    string
	ProvisionedThroughput *ProvisionedThroughput
}

type LocalSecondaryIndex struct {
	IndexName  string
	KeySchema  []KeySchemaElement
	Projection string
}

type ProvisionedThroughput struct {
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

func (c *TablesClient) ListTables(ctx context.Context) ([]Table, error) {
	var tables []Table
	paginator := dynamodb.NewListTablesPaginator(c.client, &dynamodb.ListTablesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list DynamoDB tables: %w", err)
		}

		for _, tableName := range output.TableNames {
			table, err := c.GetTable(ctx, tableName)
			if err != nil {
				continue
			}
			tables = append(tables, *table)
		}
	}

	return tables, nil
}

func (c *TablesClient) GetTable(ctx context.Context, tableName string) (*Table, error) {
	output, err := c.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	tableDesc := output.Table

	table := &Table{
		TableName:        aws.ToString(tableDesc.TableName),
		TableArn:         aws.ToString(tableDesc.TableArn),
		TableStatus:      string(tableDesc.TableStatus),
		CreationDateTime: aws.ToTime(tableDesc.CreationDateTime),
		ItemCount:        aws.ToInt64(tableDesc.ItemCount),
		TableSizeBytes:   aws.ToInt64(tableDesc.TableSizeBytes),
	}

	if tableDesc.BillingModeSummary != nil {
		table.BillingModeSummary = string(tableDesc.BillingModeSummary.BillingMode)
	} else {
		table.BillingModeSummary = "PROVISIONED"
	}

	for _, ks := range tableDesc.KeySchema {
		table.KeySchema = append(table.KeySchema, KeySchemaElement{
			AttributeName: aws.ToString(ks.AttributeName),
			KeyType:       string(ks.KeyType),
		})
	}

	for _, ad := range tableDesc.AttributeDefinitions {
		table.AttributeDefinitions = append(table.AttributeDefinitions, AttributeDefinition{
			AttributeName: aws.ToString(ad.AttributeName),
			AttributeType: string(ad.AttributeType),
		})
	}

	for _, gsi := range tableDesc.GlobalSecondaryIndexes {
		index := GlobalSecondaryIndex{
			IndexName:   aws.ToString(gsi.IndexName),
			IndexStatus: string(gsi.IndexStatus),
		}

		for _, ks := range gsi.KeySchema {
			index.KeySchema = append(index.KeySchema, KeySchemaElement{
				AttributeName: aws.ToString(ks.AttributeName),
				KeyType:       string(ks.KeyType),
			})
		}

		if gsi.Projection != nil {
			index.Projection = string(gsi.Projection.ProjectionType)
		}

		if gsi.ProvisionedThroughput != nil {
			index.ProvisionedThroughput = &ProvisionedThroughput{
				ReadCapacityUnits:  aws.ToInt64(gsi.ProvisionedThroughput.ReadCapacityUnits),
				WriteCapacityUnits: aws.ToInt64(gsi.ProvisionedThroughput.WriteCapacityUnits),
			}
		}

		table.GlobalSecondaryIndexes = append(table.GlobalSecondaryIndexes, index)
	}

	for _, lsi := range tableDesc.LocalSecondaryIndexes {
		index := LocalSecondaryIndex{
			IndexName: aws.ToString(lsi.IndexName),
		}

		for _, ks := range lsi.KeySchema {
			index.KeySchema = append(index.KeySchema, KeySchemaElement{
				AttributeName: aws.ToString(ks.AttributeName),
				KeyType:       string(ks.KeyType),
			})
		}

		if lsi.Projection != nil {
			index.Projection = string(lsi.Projection.ProjectionType)
		}

		table.LocalSecondaryIndexes = append(table.LocalSecondaryIndexes, index)
	}

	if tableDesc.StreamSpecification != nil {
		table.StreamEnabled = aws.ToBool(tableDesc.StreamSpecification.StreamEnabled)
	}

	if tableDesc.LatestStreamArn != nil {
		table.StreamArn = aws.ToString(tableDesc.LatestStreamArn)
	}

	tagsOutput, err := c.client.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{
		ResourceArn: tableDesc.TableArn,
	})
	if err == nil && tagsOutput.Tags != nil {
		table.Tags = make(map[string]string)
		for _, tag := range tagsOutput.Tags {
			table.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return table, nil
}

func (c *TablesClient) UpdateTableTags(ctx context.Context, tableArn string, tags map[string]string) error {
	var dynamoTags []types.Tag
	for k, v := range tags {
		dynamoTags = append(dynamoTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := c.client.TagResource(ctx, &dynamodb.TagResourceInput{
		ResourceArn: aws.String(tableArn),
		Tags:        dynamoTags,
	})
	if err != nil {
		return fmt.Errorf("failed to update table tags: %w", err)
	}

	return nil
}

func (c *TablesClient) DeleteTable(ctx context.Context, tableName string) error {
	_, err := c.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete table %s: %w", tableName, err)
	}

	return nil
}
