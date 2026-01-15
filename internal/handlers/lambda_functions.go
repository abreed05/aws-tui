package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	lambdaadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws/lambda"
)

// LambdaFunctionsHandler handles Lambda Function resources
type LambdaFunctionsHandler struct {
	BaseHandler
	client *lambdaadapter.FunctionsClient
	region string
}

// NewLambdaFunctionsHandler creates a new Lambda functions handler
func NewLambdaFunctionsHandler(lambdaClient *lambda.Client, region string) *LambdaFunctionsHandler {
	return &LambdaFunctionsHandler{
		client: lambdaadapter.NewFunctionsClient(lambdaClient),
		region: region,
	}
}

func (h *LambdaFunctionsHandler) ResourceType() string { return "lambda:functions" }
func (h *LambdaFunctionsHandler) ResourceName() string { return "Lambda Functions" }
func (h *LambdaFunctionsHandler) ResourceIcon() string { return "λ" }
func (h *LambdaFunctionsHandler) ShortcutKey() string  { return "lambda" }

func (h *LambdaFunctionsHandler) Columns() []ColumnDef {
	return []ColumnDef{
		{Title: "Function Name", Width: 35, Sortable: true},
		{Title: "Runtime", Width: 15, Sortable: true},
		{Title: "Memory", Width: 8, Sortable: true},
		{Title: "Timeout", Width: 8, Sortable: false},
		{Title: "Code Size", Width: 12, Sortable: false},
		{Title: "Last Modified", Width: 12, Sortable: true},
	}
}

func (h *LambdaFunctionsHandler) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	functions, err := h.client.ListFunctions(ctx)
	if err != nil {
		return nil, NewHandlerError("LIST_FAILED", "failed to list Lambda functions", err)
	}

	resources := make([]Resource, 0, len(functions))
	for _, fn := range functions {
		resource := &LambdaFunctionResource{
			function: fn,
			region:   h.region,
		}

		// Apply filter if specified
		if opts.Filter != "" {
			filter := strings.ToLower(opts.Filter)
			name := strings.ToLower(fn.FunctionName)
			runtime := strings.ToLower(fn.Runtime)
			desc := strings.ToLower(fn.Description)
			if !strings.Contains(name, filter) && !strings.Contains(runtime, filter) && !strings.Contains(desc, filter) {
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

func (h *LambdaFunctionsHandler) Get(ctx context.Context, id string) (Resource, error) {
	fn, err := h.client.GetFunction(ctx, id)
	if err != nil {
		return nil, NewHandlerError("GET_FAILED", fmt.Sprintf("failed to get Lambda function %s", id), err)
	}

	return &LambdaFunctionResource{
		function: *fn,
		region:   h.region,
	}, nil
}

func (h *LambdaFunctionsHandler) Describe(ctx context.Context, id string) (map[string]interface{}, error) {
	fn, err := h.client.GetFunction(ctx, id)
	if err != nil {
		return nil, NewHandlerError("DESCRIBE_FAILED", fmt.Sprintf("failed to describe Lambda function %s", id), err)
	}

	details := make(map[string]interface{})

	// Basic info
	details["Function"] = map[string]interface{}{
		"FunctionName": fn.FunctionName,
		"FunctionArn":  fn.FunctionARN,
		"Description":  fn.Description,
		"Runtime":      fn.Runtime,
		"Handler":      fn.Handler,
		"PackageType":  fn.PackageType,
		"State":        fn.State,
	}

	if fn.StateReason != "" {
		details["Function"].(map[string]interface{})["StateReason"] = fn.StateReason
	}

	// Configuration
	details["Configuration"] = map[string]interface{}{
		"MemorySize":    fmt.Sprintf("%d MB", fn.MemorySize),
		"Timeout":       fmt.Sprintf("%d seconds", fn.Timeout),
		"CodeSize":      formatBytes(fn.CodeSize),
		"Architectures": fn.Architectures,
		"LastModified":  fn.LastModified.Format(time.RFC3339),
	}

	// IAM Role
	details["IAM"] = map[string]interface{}{
		"Role": fn.Role,
	}

	// Environment variables (keys only for security)
	if len(fn.Environment) > 0 {
		envKeys := make([]string, 0, len(fn.Environment))
		for k := range fn.Environment {
			envKeys = append(envKeys, k)
		}
		details["EnvironmentVariables"] = envKeys
	}

	// Tags
	if len(fn.Tags) > 0 {
		details["Tags"] = fn.Tags
	}

	return details, nil
}

func (h *LambdaFunctionsHandler) Actions() []Action {
	return []Action{
		{Key: "i", Name: "invoke", Description: "Invoke function"},
		{Key: "l", Name: "logs", Description: "View CloudWatch logs"},
	}
}

// LambdaFunctionResource implements Resource interface for Lambda functions
type LambdaFunctionResource struct {
	function lambdaadapter.Function
	region   string
}

func (r *LambdaFunctionResource) GetID() string     { return r.function.FunctionName }
func (r *LambdaFunctionResource) GetName() string   { return r.function.FunctionName }
func (r *LambdaFunctionResource) GetARN() string    { return r.function.FunctionARN }
func (r *LambdaFunctionResource) GetType() string   { return "lambda:functions" }
func (r *LambdaFunctionResource) GetRegion() string { return r.region }

func (r *LambdaFunctionResource) GetCreatedAt() time.Time {
	return r.function.LastModified
}

func (r *LambdaFunctionResource) GetTags() map[string]string {
	return r.function.Tags
}

func (r *LambdaFunctionResource) ToTableRow() []string {
	lastMod := "-"
	if !r.function.LastModified.IsZero() {
		lastMod = r.function.LastModified.Format("2006-01-02")
	}

	return []string{
		r.function.FunctionName,
		r.function.Runtime,
		fmt.Sprintf("%d MB", r.function.MemorySize),
		fmt.Sprintf("%ds", r.function.Timeout),
		formatBytes(r.function.CodeSize),
		lastMod,
	}
}

func (r *LambdaFunctionResource) ToDetailMap() map[string]interface{} {
	return map[string]interface{}{
		"FunctionName": r.function.FunctionName,
		"FunctionArn":  r.function.FunctionARN,
		"Runtime":      r.function.Runtime,
		"MemorySize":   r.function.MemorySize,
		"Timeout":      r.function.Timeout,
		"CodeSize":     r.function.CodeSize,
		"Handler":      r.function.Handler,
		"State":        r.function.State,
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
