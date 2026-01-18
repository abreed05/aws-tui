package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	ddbadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/dynamodb"
)

// EditItemAction is returned by ExecuteAction to trigger editing an item
type EditItemAction struct {
	ItemID    string
	ItemKey   string
	TableName string
}

func (a *EditItemAction) Error() string {
	return fmt.Sprintf("edit item %s", a.ItemKey)
}

func (a *EditItemAction) IsActionMsg() {}

// DeleteItemAction is returned by ExecuteAction to trigger deleting an item
type DeleteItemAction struct {
	ItemID    string
	ItemKey   string
	TableName string
}

func (a *DeleteItemAction) Error() string {
	return fmt.Sprintf("delete item %s", a.ItemKey)
}

func (a *DeleteItemAction) IsActionMsg() {}

type DynamoDBItemsHandler struct {
	BaseHandler
	itemsClient  *ddbadapter.ItemsClient
	tablesClient *ddbadapter.TablesClient
	region       string
	tableName    string
}

func NewDynamoDBItemsHandler(ddbClient *dynamodb.Client, region string, tableName string) *DynamoDBItemsHandler {
	return &DynamoDBItemsHandler{
		itemsClient:  ddbadapter.NewItemsClient(ddbClient),
		tablesClient: ddbadapter.NewTablesClient(ddbClient),
		region:       region,
		tableName:    tableName,
	}
}

func (h *DynamoDBItemsHandler) ResourceType() string { return "dynamodb:items" }
func (h *DynamoDBItemsHandler) ResourceName() string {
	return fmt.Sprintf("Items in %s", h.tableName)
}
func (h *DynamoDBItemsHandler) ResourceIcon() string { return "📄" }
func (h *DynamoDBItemsHandler) ShortcutKey() string  { return "dynamodb-items" }

func (h *DynamoDBItemsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Primary Key", Width: 40, Sortable: true},
		{Title: "Sort Key", Width: 30, Sortable: true},
		{Title: "Attributes", Width: 50, Sortable: false},
	}
}

func (h *DynamoDBItemsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	scanOpts := ddbadapter.ScanOptions{
		TableName: h.tableName,
		Limit:     100,
	}

	result, err := h.itemsClient.ScanTable(ctx, scanOpts)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", fmt.Sprintf("failed to scan table %s", h.tableName), err)
	}

	table, err := h.tablesClient.GetTable(ctx, h.tableName)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to get table schema", err)
	}

	resources := make([]Resource, 0, len(result.Items))
	for _, item := range result.Items {
		resource := &DynamoDBItemResource{
			item:      item,
			region:    h.region,
			tableName: h.tableName,
			keySchema: table.KeySchema,
		}

		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			itemJSON, _ := json.Marshal(ddbadapter.ItemToMap(item))
			if !strings.Contains(strings.ToLower(string(itemJSON)), filter) {
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

func (h *DynamoDBItemsHandler) Get(ctx context.Context, id string) (Resource, error) {
	var keyMap map[string]interface{}
	if err := json.Unmarshal([]byte(id), &keyMap); err != nil {
		return nil, NewHandlerError("GET_FAILED", "invalid item key", err)
	}

	key, err := ddbadapter.MapToKey(keyMap)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", "failed to convert key", err)
	}

	item, err := h.itemsClient.GetItem(ctx, h.tableName, key)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get item from %s", h.tableName), err)
	}

	table, err := h.tablesClient.GetTable(ctx, h.tableName)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", "failed to get table schema", err)
	}

	return &DynamoDBItemResource{
		item:      *item,
		region:    h.region,
		tableName: h.tableName,
		keySchema: table.KeySchema,
	}, nil
}

func (h *DynamoDBItemsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	var keyMap map[string]interface{}
	if err := json.Unmarshal([]byte(id), &keyMap); err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", "invalid item key", err)
	}

	key, err := ddbadapter.MapToKey(keyMap)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", "failed to convert key", err)
	}

	item, err := h.itemsClient.GetItem(ctx, h.tableName, key)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to get item from %s", h.tableName), err)
	}

	details := ddbadapter.ItemToMap(*item)
	return details, nil
}

func (h *DynamoDBItemsHandler) CanEdit() bool   { return true }
func (h *DynamoDBItemsHandler) CanDelete() bool { return true }

func (h *DynamoDBItemsHandler) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	var keyMap map[string]interface{}
	if err := json.Unmarshal([]byte(id), &keyMap); err != nil {
		return NewHandlerError("UPDATE_FAILED", "invalid item key", err)
	}

	key, err := ddbadapter.MapToKey(keyMap)
	if err != nil {
		return NewHandlerError("UPDATE_FAILED", "failed to convert key", err)
	}

	itemMap, ok := updates["item"].(map[string]interface{})
	if !ok {
		return NewHandlerError("UPDATE_FAILED", "invalid item data", fmt.Errorf("item field is required"))
	}

	// Convert the item map to DynamoDB attribute values
	attributeValues, err := ddbadapter.MapToKey(itemMap)
	if err != nil {
		return NewHandlerError("UPDATE_FAILED", "failed to convert item to DynamoDB format", err)
	}

	// Ensure the key attributes are present in the item
	for k, v := range key {
		attributeValues[k] = v
	}

	return h.itemsClient.PutItem(ctx, h.tableName, attributeValues)
}

func (h *DynamoDBItemsHandler) Delete(ctx context.Context, id string) error {
	var keyMap map[string]interface{}
	if err := json.Unmarshal([]byte(id), &keyMap); err != nil {
		return NewHandlerError("DELETE_FAILED", "invalid item key", err)
	}

	key, err := ddbadapter.MapToKey(keyMap)
	if err != nil {
		return NewHandlerError("DELETE_FAILED", "failed to convert key", err)
	}

	return h.itemsClient.DeleteItem(ctx, h.tableName, key)
}

func (h *DynamoDBItemsHandler) Actions() []Action {
	return []Action{
		{Key: "e", Name: "edit", Description: "Edit item"},
		{Key: "x", Name: "delete", Description: "Delete item"},
	}
}

func (h *DynamoDBItemsHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	resource, err := h.Get(ctx, resourceID)
	if err != nil {
		return err
	}

	switch action {
	case "edit":
		return &EditItemAction{
			ItemID:    resourceID,
			ItemKey:   resource.GetName(),
			TableName: h.tableName,
		}
	case "delete":
		return &DeleteItemAction{
			ItemID:    resourceID,
			ItemKey:   resource.GetName(),
			TableName: h.tableName,
		}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

type DynamoDBItemResource struct {
	item      ddbadapter.Item
	region    string
	tableName string
	keySchema []ddbadapter.KeySchemaElement
}

func (r *DynamoDBItemResource) GetID() string {
	keyData := make(map[string]types.AttributeValue)
	for _, keyElem := range r.keySchema {
		if val, ok := r.item.Attributes[keyElem.AttributeName]; ok {
			keyData[keyElem.AttributeName] = val
		}
	}

	// Convert to plain map for JSON serialization
	keyMap, err := ddbadapter.KeyToMap(keyData)
	if err != nil {
		return ""
	}

	keyJSON, _ := json.Marshal(keyMap)
	return string(keyJSON)
}

func (r *DynamoDBItemResource) GetARN() string {
	return fmt.Sprintf("arn:aws:dynamodb:%s::table/%s/item/%s", r.region, r.tableName, r.GetID())
}

func (r *DynamoDBItemResource) GetName() string {
	if len(r.keySchema) > 0 {
		keyAttr := r.keySchema[0].AttributeName
		if val, ok := r.item.Attributes[keyAttr]; ok {
			return fmt.Sprintf("%v", ddbadapter.AttributeValueToInterface(val))
		}
	}
	return "Unknown"
}

func (r *DynamoDBItemResource) GetType() string              { return "dynamodb:item" }
func (r *DynamoDBItemResource) GetRegion() string            { return r.region }
func (r *DynamoDBItemResource) GetCreatedAt() time.Time      { return time.Time{} }
func (r *DynamoDBItemResource) GetTags() map[string]string   { return nil }

func (r *DynamoDBItemResource) ToTableRow() []string {
	var primaryKey, sortKey string
	var otherAttrs []string

	for i, keyElem := range r.keySchema {
		if val, ok := r.item.Attributes[keyElem.AttributeName]; ok {
			valStr := fmt.Sprintf("%v", ddbadapter.AttributeValueToInterface(val))
			if i == 0 {
				primaryKey = valStr
			} else if i == 1 {
				sortKey = valStr
			}
		}
	}

	for attrName := range r.item.Attributes {
		isKey := false
		for _, keyElem := range r.keySchema {
			if keyElem.AttributeName == attrName {
				isKey = true
				break
			}
		}
		if !isKey {
			otherAttrs = append(otherAttrs, attrName)
		}
	}

	attributes := strings.Join(otherAttrs, ", ")
	if len(attributes) > 47 {
		attributes = attributes[:47] + "..."
	}

	return []string{
		primaryKey,
		sortKey,
		attributes,
	}
}

func (r *DynamoDBItemResource) ToDetailMap() map[string]interface{} {
	return ddbadapter.ItemToMap(r.item)
}
