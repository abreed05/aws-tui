package lambda

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// FunctionsClient wraps the Lambda client
type FunctionsClient struct {
	client *lambda.Client
}

// NewFunctionsClient creates a new Lambda functions client
func NewFunctionsClient(client *lambda.Client) *FunctionsClient {
	return &FunctionsClient{client: client}
}

// Function represents a Lambda function
type Function struct {
	FunctionName    string
	FunctionARN     string
	Runtime         string
	Handler         string
	CodeSize        int64
	Description     string
	Timeout         int32
	MemorySize      int32
	LastModified    time.Time
	Role            string
	State           string
	StateReason     string
	PackageType     string
	Architectures   []string
	Environment     map[string]string
	Tags            map[string]string
}

// ListFunctions lists all Lambda functions
func (c *FunctionsClient) ListFunctions(ctx context.Context) ([]Function, error) {
	var functions []Function
	var marker *string

	for {
		input := &lambda.ListFunctionsInput{
			Marker: marker,
		}

		output, err := c.client.ListFunctions(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions: %w", err)
		}

		for _, fn := range output.Functions {
			functions = append(functions, convertFunction(fn))
		}

		if output.NextMarker == nil {
			break
		}
		marker = output.NextMarker
	}

	return functions, nil
}

// GetFunction gets a single Lambda function
func (c *FunctionsClient) GetFunction(ctx context.Context, functionName string) (*Function, error) {
	output, err := c.client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function %s: %w", functionName, err)
	}

	fn := convertFunctionConfig(*output.Configuration)

	// Get tags
	tagsOutput, err := c.client.ListTags(ctx, &lambda.ListTagsInput{
		Resource: output.Configuration.FunctionArn,
	})
	if err == nil {
		fn.Tags = tagsOutput.Tags
	}

	return &fn, nil
}

func convertFunction(fn types.FunctionConfiguration) Function {
	return convertFunctionConfig(fn)
}

func convertFunctionConfig(fn types.FunctionConfiguration) Function {
	result := Function{
		FunctionName: aws.ToString(fn.FunctionName),
		FunctionARN:  aws.ToString(fn.FunctionArn),
		Runtime:      string(fn.Runtime),
		Handler:      aws.ToString(fn.Handler),
		CodeSize:     fn.CodeSize,
		Description:  aws.ToString(fn.Description),
		Role:         aws.ToString(fn.Role),
		State:        string(fn.State),
		StateReason:  aws.ToString(fn.StateReason),
		PackageType:  string(fn.PackageType),
		Tags:         make(map[string]string),
	}

	if fn.Timeout != nil {
		result.Timeout = *fn.Timeout
	}

	if fn.MemorySize != nil {
		result.MemorySize = *fn.MemorySize
	}

	if fn.LastModified != nil {
		// Lambda returns time as string in ISO format
		t, err := time.Parse("2006-01-02T15:04:05.000+0000", *fn.LastModified)
		if err == nil {
			result.LastModified = t
		}
	}

	for _, arch := range fn.Architectures {
		result.Architectures = append(result.Architectures, string(arch))
	}

	if fn.Environment != nil {
		result.Environment = fn.Environment.Variables
	}

	return result
}
