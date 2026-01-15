package handlers

import (
	"context"
	"time"
)

// Resource represents a generic AWS resource
type Resource interface {
	// Identity
	GetID() string
	GetARN() string
	GetName() string
	GetType() string

	// Metadata
	GetRegion() string
	GetCreatedAt() time.Time
	GetTags() map[string]string

	// Display
	ToTableRow() []string
	ToDetailMap() map[string]interface{}
}

// ColumnDef defines a table column
type ColumnDef struct {
	Title    string
	Width    int
	Sortable bool
}

// Action defines a resource-specific action
type Action struct {
	Key         string
	Name        string
	Description string
	Dangerous   bool
}

// ListOptions defines options for listing resources
type ListOptions struct {
	Filter    string
	PageSize  int
	NextToken string
	SortField string
	SortAsc   bool
}

// ListResult contains the result of a list operation
type ListResult struct {
	Resources []Resource
	NextToken string
	Total     int
}

// ResourceHandler defines the interface for resource type handlers
type ResourceHandler interface {
	// Metadata
	ResourceType() string
	ResourceName() string
	ResourceIcon() string
	ShortcutKey() string

	// Column definitions for table view
	Columns() []ColumnDef

	// Data operations
	List(ctx context.Context, opts ListOptions) (*ListResult, error)
	Get(ctx context.Context, id string) (Resource, error)
	Describe(ctx context.Context, id string) (map[string]interface{}, error)

	// Mutation capabilities
	CanEdit() bool
	CanDelete() bool

	// Mutations (optional - check CanEdit/CanDelete first)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error

	// Resource-specific actions
	Actions() []Action
	ExecuteAction(ctx context.Context, action string, resourceID string) error
}

// BaseHandler provides default implementations for optional methods
type BaseHandler struct{}

func (b *BaseHandler) CanEdit() bool   { return false }
func (b *BaseHandler) CanDelete() bool { return false }

func (b *BaseHandler) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return ErrNotSupported
}

func (b *BaseHandler) Delete(ctx context.Context, id string) error {
	return ErrNotSupported
}

func (b *BaseHandler) Actions() []Action {
	return nil
}

func (b *BaseHandler) ExecuteAction(ctx context.Context, action string, resourceID string) error {
	return ErrNotSupported
}

// Common errors
var (
	ErrNotSupported = &HandlerError{Code: "NOT_SUPPORTED", Message: "operation not supported"}
	ErrNotFound     = &HandlerError{Code: "NOT_FOUND", Message: "resource not found"}
	ErrUnauthorized = &HandlerError{Code: "UNAUTHORIZED", Message: "access denied"}
)

// HandlerError represents a handler-specific error
type HandlerError struct {
	Code    string
	Message string
	Cause   error
}

func (e *HandlerError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *HandlerError) Unwrap() error {
	return e.Cause
}

// NewHandlerError creates a new handler error
func NewHandlerError(code, message string, cause error) *HandlerError {
	return &HandlerError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
