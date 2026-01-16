package messages

// Navigation messages

// NavigateToMsg requests navigation to a specific view
type NavigateToMsg struct {
	View         string
	ResourceType string
	ResourceID   string
}

// NavigateBackMsg requests navigation back
type NavigateBackMsg struct{}

// Resource messages

// RefreshMsg requests a data refresh
type RefreshMsg struct{}

// ResourcesLoadedMsg indicates resources have been loaded
type ResourcesLoadedMsg struct {
	ResourceType string
	Resources    []interface{}
	Error        error
}

// ResourceDetailLoadedMsg indicates resource details have been loaded
type ResourceDetailLoadedMsg struct {
	ResourceType string
	ResourceID   string
	Details      map[string]interface{}
	Error        error
}

// Error messages

// ErrorMsg indicates an error occurred
type ErrorMsg struct {
	Error   error
	Context string
}

// ClearErrorMsg clears the current error
type ClearErrorMsg struct{}

// Status messages

// StatusMsg updates the status bar
type StatusMsg struct {
	Message string
	IsError bool
}

// LoadingMsg indicates loading state
type LoadingMsg struct {
	Loading bool
	Message string
}

// Profile/Region messages

// ProfilesLoadedMsg indicates profiles have been loaded
type ProfilesLoadedMsg struct {
	Profiles []interface{}
	Error    error
}

// ContextChangedMsg indicates profile or region changed
type ContextChangedMsg struct {
	Profile   string
	Region    string
	AccountID string
}

// Clipboard messages

// CopiedToClipboardMsg indicates something was copied
type CopiedToClipboardMsg struct {
	Content string
	Label   string
}

// Command messages

// CommandMsg represents a command to execute
type CommandMsg struct {
	Command string
	Args    []string
}

// Search messages

// SearchMsg represents a search query
type SearchMsg struct {
	Query string
}

// Window messages

// WindowSizeMsg is sent when window size changes
type WindowSizeMsg struct {
	Width  int
	Height int
}

// ECS Navigation messages

// NavigateToECSServicesMsg requests navigation to ECS services for a cluster
type NavigateToECSServicesMsg struct {
	ClusterARN  string
	ClusterName string
	Breadcrumb  []string
}

// NavigateToECSTasksMsg requests navigation to ECS tasks for a service
type NavigateToECSTasksMsg struct {
	ClusterARN  string
	ClusterName string
	ServiceARN  string
	ServiceName string
	Breadcrumb  []string
}

// ECSExecRequestMsg requests execution into a container
type ECSExecRequestMsg struct {
	ClusterARN string
	TaskARN    string
	Containers []ECSContainer
}

// ECSContainer represents a container in a task
type ECSContainer struct {
	Name         string
	RuntimeId    string
	Status       string
	HealthStatus string
}

// ContainerSelectedMsg is sent when a container is selected from the picker
type ContainerSelectedMsg struct {
	ContainerName string
}

// ECSExecFinishedMsg is sent when the exec command completes
type ECSExecFinishedMsg struct {
	Error error
}
