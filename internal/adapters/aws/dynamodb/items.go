package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ItemsClient struct {
	client *dynamodb.Client
}

func NewItemsClient(client *dynamodb.Client) *ItemsClient {
	return &ItemsClient{client: client}
}

type Item struct {
	TableName  string
	Attributes map[string]types.AttributeValue
	Keys       map[string]types.AttributeValue
}

type ScanOptions struct {
	TableName      string
	Limit          int32
	ExclusiveStartKey map[string]types.AttributeValue
}

type ScanResult struct {
	Items            []Item
	LastEvaluatedKey map[string]types.AttributeValue
	Count            int32
	ScannedCount     int32
}

func (c *ItemsClient) ScanTable(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(opts.TableName),
	}

	if opts.Limit > 0 {
		input.Limit = aws.Int32(opts.Limit)
	} else {
		input.Limit = aws.Int32(100)
	}

	if opts.ExclusiveStartKey != nil {
		input.ExclusiveStartKey = opts.ExclusiveStartKey
	}

	output, err := c.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan table %s: %w", opts.TableName, err)
	}

	var items []Item
	for _, item := range output.Items {
		items = append(items, Item{
			TableName:  opts.TableName,
			Attributes: item,
		})
	}

	return &ScanResult{
		Items:            items,
		LastEvaluatedKey: output.LastEvaluatedKey,
		Count:            output.Count,
		ScannedCount:     output.ScannedCount,
	}, nil
}

func (c *ItemsClient) GetItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) (*Item, error) {
	output, err := c.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from table %s: %w", tableName, err)
	}

	if output.Item == nil {
		return nil, fmt.Errorf("item not found")
	}

	return &Item{
		TableName:  tableName,
		Attributes: output.Item,
		Keys:       key,
	}, nil
}

func (c *ItemsClient) PutItem(ctx context.Context, tableName string, item map[string]types.AttributeValue) error {
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in table %s: %w", tableName, err)
	}

	return nil
}

func (c *ItemsClient) UpdateItem(ctx context.Context, tableName string, key map[string]types.AttributeValue,
	updateExpression string, expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue) error {

	input := &dynamodb.UpdateItemInput{
		TableName:        aws.String(tableName),
		Key:              key,
		UpdateExpression: aws.String(updateExpression),
	}

	if expressionAttributeNames != nil {
		input.ExpressionAttributeNames = expressionAttributeNames
	}

	if expressionAttributeValues != nil {
		input.ExpressionAttributeValues = expressionAttributeValues
	}

	_, err := c.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update item in table %s: %w", tableName, err)
	}

	return nil
}

func (c *ItemsClient) DeleteItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) error {
	_, err := c.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete item from table %s: %w", tableName, err)
	}

	return nil
}

func AttributeValueToInterface(av types.AttributeValue) interface{} {
	var result interface{}
	attributevalue.Unmarshal(av, &result)
	return result
}

func ItemToMap(item Item) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range item.Attributes {
		result[k] = AttributeValueToInterface(v)
	}
	return result
}

// KeyToMap converts a DynamoDB key to a plain map for serialization
func KeyToMap(key map[string]types.AttributeValue) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range key {
		result[k] = AttributeValueToInterface(v)
	}
	return result, nil
}

// MapToKey converts a plain map back to a DynamoDB key
func MapToKey(keyMap map[string]interface{}) (map[string]types.AttributeValue, error) {
	key := make(map[string]types.AttributeValue)
	for k, v := range keyMap {
		av, err := attributevalue.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key attribute %s: %w", k, err)
		}
		key[k] = av
	}
	return key, nil
}
