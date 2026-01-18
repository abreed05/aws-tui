package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	ddbadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/dynamodb"
)

// NavigateToItemsAction is returned by ExecuteAction to trigger navigation to table items
type NavigateToItemsAction struct {
	TableName string
}

func (a *NavigateToItemsAction) Error() string {
	return fmt.Sprintf("navigate to items for table %s", a.TableName)
}

func (a *NavigateToItemsAction) IsActionMsg() {}

type DynamoDBTablesHandler struct {
	BaseHandler
	client *ddbadapter.TablesClient
	region string
}

func NewDynamoDBTablesHandler(ddbClient *dynamodb.Client, region string) *DynamoDBTablesHandler {
	return &DynamoDBTablesHandler{
		client: ddbadapter.NewTablesClient(ddbClient),
		region: region,
	}
}

func (h *DynamoDBTablesHandler) ResourceType() string { return "dynamodb:tables" }
func (h *DynamoDBTablesHandler) ResourceName() string { return "DynamoDB Tables" }
func (h *DynamoDBTablesHandler) ResourceIcon() string { return "🗄️" }
func (h *DynamoDBTablesHandler) ShortcutKey() string  { return "dynamodb" }

func (h *DynamoDBTablesHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Table Name", Width: 35, Sortable: true},
		{Title: "Status", Width: 12, Sortable: true},
		{Title: "Billing Mode", Width: 15, Sortable: true},
		{Title: "Item Count", Width: 12, Sortable: true},
		{Title: "Size (MB)", Width: 12, Sortable: true},
		{Title: "Created", Width: 14, Sortable: true},
	}
}

func (h *DynamoDBTablesHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	tables, err := h.client.ListTables(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list DynamoDB tables", err)
	}

	resources := make([]Resource, 0, len(tables))
	for _, table := range tables {
		resource := &DynamoDBTableResource{
			table:  table,
			region: h.region,
		}

		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(table.TableName)
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

func (h *DynamoDBTablesHandler) Get(ctx context.Context, id string) (Resource, error) {
	table, err := h.client.GetTable(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get table %s", id), err)
	}

	return &DynamoDBTableResource{
		table:  *table,
		region: h.region,
	}, nil
}

func (h *DynamoDBTablesHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	table, err := h.client.GetTable(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe table %s", id), err)
	}

	details := make(map[string]interface{})

	tableInfo := map[string]interface{}{
		"TableName":       table.TableName,
		"TableArn":        table.TableArn,
		"TableStatus":     table.TableStatus,
		"BillingMode":     table.BillingModeSummary,
		"ItemCount":       table.ItemCount,
		"TableSizeBytes":  table.TableSizeBytes,
		"CreationDateTime": table.CreationDateTime.Format(time.RFC3339),
	}
	details["Table"] = tableInfo

	if len(table.KeySchema) > 0 {
		keySchema := make([]map[string]string, 0, len(table.KeySchema))
		for _, key := range table.KeySchema {
			keySchema = append(keySchema, map[string]string{
				"AttributeName": key.AttributeName,
				"KeyType":       key.KeyType,
			})
		}
		details["KeySchema"] = keySchema
	}

	if len(table.AttributeDefinitions) > 0 {
		attributes := make([]map[string]string, 0, len(table.AttributeDefinitions))
		for _, attr := range table.AttributeDefinitions {
			attributes = append(attributes, map[string]string{
				"AttributeName": attr.AttributeName,
				"AttributeType": attr.AttributeType,
			})
		}
		details["AttributeDefinitions"] = attributes
	}

	if len(table.GlobalSecondaryIndexes) > 0 {
		gsis := make([]map[string]interface{}, 0, len(table.GlobalSecondaryIndexes))
		for _, gsi := range table.GlobalSecondaryIndexes {
			gsiMap := map[string]interface{}{
				"IndexName":   gsi.IndexName,
				"IndexStatus": gsi.IndexStatus,
				"Projection":  gsi.Projection,
			}
			if gsi.ProvisionedThroughput != nil {
				gsiMap["ProvisionedThroughput"] = map[string]int64{
					"ReadCapacityUnits":  gsi.ProvisionedThroughput.ReadCapacityUnits,
					"WriteCapacityUnits": gsi.ProvisionedThroughput.WriteCapacityUnits,
				}
			}
			gsis = append(gsis, gsiMap)
		}
		details["GlobalSecondaryIndexes"] = gsis
	}

	if len(table.LocalSecondaryIndexes) > 0 {
		lsis := make([]map[string]interface{}, 0, len(table.LocalSecondaryIndexes))
		for _, lsi := range table.LocalSecondaryIndexes {
			lsis = append(lsis, map[string]interface{}{
				"IndexName":  lsi.IndexName,
				"Projection": lsi.Projection,
			})
		}
		details["LocalSecondaryIndexes"] = lsis
	}

	if table.StreamEnabled {
		details["Streams"] = map[string]interface{}{
			"Enabled":   true,
			"StreamArn": table.StreamArn,
		}
	}

	if len(table.Tags) > 0 {
		details["Tags"] = table.Tags
	}

	return details, nil
}

func (h *DynamoDBTablesHandler) CanEdit() bool   { return false }
func (h *DynamoDBTablesHandler) CanDelete() bool { return true }

func (h *DynamoDBTablesHandler) Delete(ctx context.Context, id string) error {
	return h.client.DeleteTable(ctx, id)
}

func (h *DynamoDBTablesHandler) Actions() []Action {
	return []Action{
		{Key: "v", Name: "view-items", Description: "View table items"},
	}
}

func (h *DynamoDBTablesHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	table, err := h.Get(ctx, resourceID)
	if err != nil {
		return err
	}

	switch action {
	case "view-items":
		return &NavigateToItemsAction{
			TableName: table.GetName(),
		}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

type DynamoDBTableResource struct {
	table  ddbadapter.Table
	region string
}

func (r *DynamoDBTableResource) GetID() string   { return r.table.TableName }
func (r *DynamoDBTableResource) GetARN() string  { return r.table.TableArn }
func (r *DynamoDBTableResource) GetName() string { return r.table.TableName }
func (r *DynamoDBTableResource) GetType() string { return "dynamodb:table" }

func (r *DynamoDBTableResource) GetRegion() string              { return r.region }
func (r *DynamoDBTableResource) GetCreatedAt() time.Time        { return r.table.CreationDateTime }
func (r *DynamoDBTableResource) GetTags() map[string]string     { return r.table.Tags }

func (r *DynamoDBTableResource) ToTableRow() []string {
	itemCount := fmt.Sprintf("%d", r.table.ItemCount)
	sizeInMB := fmt.Sprintf("%.2f", float64(r.table.TableSizeBytes)/(1024*1024))
	created := r.table.CreationDateTime.Format("2006-01-02")

	return []string{
		r.table.TableName,
		r.table.TableStatus,
		r.table.BillingModeSummary,
		itemCount,
		sizeInMB,
		created,
	}
}

func (r *DynamoDBTableResource) ToDetailMap() map[string]interface{} {
	details := map[string]interface{}{
		"TableName":      r.table.TableName,
		"TableArn":       r.table.TableArn,
		"TableStatus":    r.table.TableStatus,
		"BillingMode":    r.table.BillingModeSummary,
		"ItemCount":      r.table.ItemCount,
		"TableSizeBytes": r.table.TableSizeBytes,
		"Created":        r.table.CreationDateTime.Format(time.RFC3339),
	}

	if len(r.table.KeySchema) > 0 {
		keySchema := make([]string, 0, len(r.table.KeySchema))
		for _, key := range r.table.KeySchema {
			keySchema = append(keySchema, fmt.Sprintf("%s (%s)", key.AttributeName, key.KeyType))
		}
		details["KeySchema"] = strings.Join(keySchema, ", ")
	}

	if r.table.StreamEnabled {
		details["StreamEnabled"] = true
	}

	return details
}
