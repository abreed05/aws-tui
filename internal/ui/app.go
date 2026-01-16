package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsadapter "github.com/aaw-tui/aws-tui/internal/adapters/aws"
	"github.com/aaw-tui/aws-tui/internal/adapters/config"
	"github.com/aaw-tui/aws-tui/internal/app"
	"github.com/aaw-tui/aws-tui/internal/handlers"
	"github.com/aaw-tui/aws-tui/internal/ui/components"
	"github.com/aaw-tui/aws-tui/internal/ui/keys"
	"github.com/aaw-tui/aws-tui/internal/ui/messages"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
	"github.com/aaw-tui/aws-tui/internal/ui/views"
	"github.com/aaw-tui/aws-tui/internal/utils"
)

// AppState represents the current application state
type AppState int

const (
	StateHome AppState = iota
	StateResourceList
	StateResourceDetail
)

// Mode represents vim-like modes
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeCommand
)

// App is the main Bubbletea model
type App struct {
	// Configuration
	config *app.Config

	// State
	state AppState
	mode  Mode

	// AWS context
	clientMgr     *awsadapter.ClientManager
	profileLoader *config.ProfileLoader
	profiles      []config.Profile
	regions       []config.Region

	// Handler registry
	registry *handlers.Registry

	// Bookmarks
	bookmarkStore    *config.BookmarkStore
	bookmarkSelector *components.BookmarkSelector

	// UI Components
	header       *components.Header
	footer       *components.Footer
	breadcrumb   *components.Breadcrumb
	selector     *components.Selector
	resourceList *views.ResourceListView
	autocomplete *components.Autocomplete

	// Input components for modes
	commandInput textinput.Model

	// Theme and keys
	theme styles.Theme
	keys  keys.KeyMap

	// Dimensions
	width  int
	height int

	// State
	lastError   error
	loading     bool
	loadingMsg  string
	initialized bool
}

// NewApp creates a new application instance
func NewApp(cfg *app.Config) (*App, error) {
	// Load theme from config, fallback to default if not found
	theme, err := styles.LoadTheme(cfg.Theme, cfg.ConfigDir)
	if err != nil {
		// Theme not found, use default (error is non-fatal)
		theme = styles.DefaultTheme()
	}
	keyMap := keys.DefaultKeyMap()

	// Initialize command input
	commandInput := textinput.New()
	commandInput.Placeholder = ""
	commandInput.Prompt = ":"
	commandInput.CharLimit = 200

	// Initialize bookmark store
	bookmarkStore := config.NewBookmarkStore()
	_ = bookmarkStore.Load() // Ignore error on initial load

	a := &App{
		config:           cfg,
		state:            StateHome,
		mode:             ModeNormal,
		clientMgr:        awsadapter.NewClientManager(),
		profileLoader:    config.NewProfileLoader(),
		registry:         handlers.NewRegistry(),
		bookmarkStore:    bookmarkStore,
		bookmarkSelector: components.NewBookmarkSelector(theme, bookmarkStore),
		theme:            theme,
		keys:             keyMap,
		header:           components.NewHeader(theme),
		footer:           components.NewFooter(theme, keyMap),
		breadcrumb:       components.NewBreadcrumb(theme),
		selector:         components.NewSelector(theme),
		resourceList:     views.NewResourceListView(theme),
		autocomplete:     components.NewAutocomplete(),
		commandInput:     commandInput,
	}

	// Load regions (static)
	a.regions = a.profileLoader.ListRegions()

	return a, nil
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.loadProfiles(),
		a.initializeAWS(),
	)
}

// loadProfiles loads AWS profiles
func (a *App) loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, err := a.profileLoader.ListProfiles()
		if err != nil {
			return messages.ErrorMsg{Error: err, Context: "loading profiles"}
		}
		return profilesLoadedMsg{profiles: profiles}
	}
}

// initializeAWS initializes the AWS client
func (a *App) initializeAWS() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		profile := a.config.DefaultProfile
		region := a.config.DefaultRegion

		if err := a.clientMgr.Configure(ctx, profile, region); err != nil {
			// Still initialize with error - user can switch profiles
			return awsInitializedMsg{
				profile:   profile,
				region:    region,
				accountID: "",
				err:       err,
			}
		}

		// Try to get account ID - this validates credentials
		accountID, err := a.clientMgr.GetAccountID(ctx)
		if err != nil {
			// Credentials invalid but config loaded - user can switch profiles
			return awsInitializedMsg{
				profile:   profile,
				region:    region,
				accountID: "",
				err:       err,
			}
		}

		return awsInitializedMsg{
			profile:   profile,
			region:    region,
			accountID: accountID,
		}
	}
}

// registerHandlers registers all resource handlers
func (a *App) registerHandlers() {
	// Register IAM handlers
	a.registry.Register(handlers.NewIAMUsersHandler(a.clientMgr.IAM()))
	a.registry.Register(handlers.NewIAMRolesHandler(a.clientMgr.IAM()))
	a.registry.Register(handlers.NewIAMPoliciesHandler(a.clientMgr.IAM()))

	// Register EC2 handlers
	a.registry.Register(handlers.NewSecurityGroupsHandler(a.clientMgr.EC2(), a.clientMgr.Region()))
	a.registry.Register(handlers.NewEC2InstancesHandler(a.clientMgr.EC2(), a.clientMgr.Region()))
	a.registry.Register(handlers.NewVPCsHandler(a.clientMgr.EC2(), a.clientMgr.Region()))

	// Register KMS handlers
	a.registry.Register(handlers.NewKMSKeysHandler(a.clientMgr.KMS(), a.clientMgr.Region()))

	// Register Secrets Manager handlers
	a.registry.Register(handlers.NewSecretsHandler(a.clientMgr.SecretsManager(), a.clientMgr.Region()))

	// Register RDS handlers
	a.registry.Register(handlers.NewRDSInstancesHandler(a.clientMgr.RDS(), a.clientMgr.Region()))

	// Register ECS handlers
	a.registry.Register(handlers.NewECSClustersHandler(a.clientMgr.ECS(), a.clientMgr.Region()))

	// Register Lambda handlers
	a.registry.Register(handlers.NewLambdaFunctionsHandler(a.clientMgr.Lambda(), a.clientMgr.Region()))

	// Register S3 handlers
	a.registry.Register(handlers.NewS3BucketsHandler(a.clientMgr.S3(), a.clientMgr.Region()))
}

// Internal messages
type profilesLoadedMsg struct {
	profiles []config.Profile
}

type awsInitializedMsg struct {
	profile   string
	region    string
	accountID string
	err       error
}

// Update handles all messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle selector if active
		if a.selector.IsActive() {
			newSelector, cmd := a.selector.Update(msg)
			a.selector = newSelector
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		// Handle bookmark selector if active
		if a.bookmarkSelector.IsActive() {
			var cmd tea.Cmd
			a.bookmarkSelector, cmd = a.bookmarkSelector.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		// Handle mode-specific input
		switch a.mode {
		case ModeCommand:
			return a.handleCommandInput(msg)
		default:
			return a.handleNormalMode(msg)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.header.SetWidth(msg.Width)
		a.footer.SetWidth(msg.Width)
		a.breadcrumb.SetWidth(msg.Width)
		a.selector.SetSize(msg.Width, msg.Height)
		a.bookmarkSelector.SetSize(msg.Width, msg.Height)

		// Update resource list size
		contentHeight := a.calculateContentHeight()
		a.resourceList.SetSize(msg.Width, contentHeight)
		return a, nil

	case profilesLoadedMsg:
		a.profiles = msg.profiles
		return a, nil

	case awsInitializedMsg:
		a.header.SetProfile(msg.profile)
		a.header.SetRegion(msg.region)
		a.header.SetAccountID(msg.accountID)
		a.initialized = true

		// Register handlers now that AWS is configured
		a.registerHandlers()

		// Show error if credentials failed
		if msg.err != nil {
			a.footer.SetMessage(fmt.Sprintf("AWS Error: %v. Press 'p' to select a profile.", msg.err), true)
		}
		return a, nil

	case ssoLoginFinishedMsg:
		if msg.err != nil {
			a.footer.SetMessage(fmt.Sprintf("SSO login failed: %v", msg.err), true)
			return a, nil
		}
		a.footer.SetMessage("SSO session refreshed successfully", false)
		// Re-initialize the client to pick up new credentials
		return a, a.switchProfile(a.clientMgr.Profile())

	case components.ProfileSelectedMsg:
		return a, a.switchProfile(msg.Profile)

	case components.RegionSelectedMsg:
		return a, a.switchRegion(msg.Region)

	case components.SelectorClosedMsg:
		return a, nil

	case messages.ErrorMsg:
		a.lastError = msg.Error
		a.footer.SetMessage(fmt.Sprintf("Error: %v", msg.Error), true)
		return a, nil

	case messages.LoadingMsg:
		a.loading = msg.Loading
		a.loadingMsg = msg.Message
		a.footer.SetLoading(msg.Loading, msg.Message)
		return a, nil

	// Resource list messages
	case views.ResourcesLoadedMsg:
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		a.loading = false
		a.footer.SetLoading(false, "")
		if msg.Error != nil {
			a.footer.SetMessage(fmt.Sprintf("Error: %v", msg.Error), true)
		} else {
			a.footer.ClearMessage()
			// Update pagination info
			page, hasMore, count := a.resourceList.GetPaginationInfo()
			a.footer.SetPagination(page, hasMore, count)
		}
		return a, cmd

	case views.ResourceDetailLoadedMsg:
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		return a, cmd

	case components.ResourceSelectedMsg:
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		return a, cmd

	case components.SearchUpdateMsg, components.SearchClosedMsg:
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		return a, cmd

	case components.ClipboardCopiedMsg:
		if msg.Success {
			a.footer.SetMessage(fmt.Sprintf("Copied %s to clipboard", msg.Label), false)
		} else {
			a.footer.SetMessage(fmt.Sprintf("Failed to copy: %v", msg.Error), true)
		}
		return a, nil

	case components.BookmarkAddedMsg:
		if msg.Success {
			a.footer.SetMessage(fmt.Sprintf("Bookmarked: %s", msg.Name), false)
		} else {
			a.footer.SetMessage(fmt.Sprintf("Failed to bookmark: %v", msg.Error), true)
		}
		return a, nil

	case components.BookmarkRemovedMsg:
		if msg.Success {
			a.footer.SetMessage("Bookmark removed", false)
		} else {
			a.footer.SetMessage(fmt.Sprintf("Failed to remove bookmark: %v", msg.Error), true)
		}
		return a, nil

	case components.BookmarkClosedMsg:
		return a, nil

	case components.BookmarkSelectedMsg:
		// Navigate to the bookmarked resource
		return a.navigateToBookmark(msg.Bookmark)

	// ECS Navigation actions
	case *handlers.NavigateToServicesAction:
		handler := handlers.NewECSServicesHandlerForCluster(
			a.clientMgr.ECS(),
			a.clientMgr.Region(),
			msg.ClusterARN,
			msg.ClusterName,
		)
		a.state = StateResourceList
		a.breadcrumb.SetPath("ECS", "Clusters", msg.ClusterName, "Services")
		a.resourceList.SetHandler(handler)
		a.footer.SetHandlerActions(handler.Actions())
		a.loading = true
		a.footer.SetLoading(true, "Loading services...")
		contentHeight := a.calculateContentHeight()
		a.resourceList.SetSize(a.width, contentHeight)
		return a, a.resourceList.LoadResources(context.Background(), "")

	case *handlers.NavigateToTasksAction:
		var handler *handlers.ECSTasksHandler
		if msg.ServiceARN != "" {
			handler = handlers.NewECSTasksHandlerForService(
				a.clientMgr.ECS(),
				a.clientMgr.Region(),
				msg.ClusterARN,
				msg.ClusterName,
				msg.ServiceARN,
				msg.ServiceName,
			)
			a.breadcrumb.SetPath("ECS", "Clusters", msg.ClusterName, "Services", msg.ServiceName, "Tasks")
		} else {
			handler = handlers.NewECSTasksHandlerForCluster(
				a.clientMgr.ECS(),
				a.clientMgr.Region(),
				msg.ClusterARN,
				msg.ClusterName,
			)
			a.breadcrumb.SetPath("ECS", "Clusters", msg.ClusterName, "Tasks")
		}
		a.state = StateResourceList
		a.resourceList.SetHandler(handler)
		a.footer.SetHandlerActions(handler.Actions())
		a.loading = true
		a.footer.SetLoading(true, "Loading tasks...")
		contentHeight := a.calculateContentHeight()
		a.resourceList.SetSize(a.width, contentHeight)
		return a, a.resourceList.LoadResources(context.Background(), "")

	case *handlers.ExecRequestAction:
		// For now, auto-select first container (can add picker later)
		containerName := msg.Containers[0].Name
		if len(msg.Containers) > 1 {
			a.footer.SetMessage(fmt.Sprintf("Multiple containers found, using: %s", containerName), false)
		}
		return a, a.executeECSExec(msg.ClusterARN, msg.TaskARN, containerName)

	case ecsExecFinishedMsg:
		if msg.err != nil {
			a.footer.SetMessage(fmt.Sprintf("Exec failed: %v", msg.err), true)
		} else {
			a.footer.SetMessage("Exec session completed", false)
		}
		return a, nil

	case views.ActionErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Action failed: %v", msg.Error), true)
		return a, nil
	}

	// Route to resource list if in that state
	if a.state == StateResourceList {
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) calculateContentHeight() int {
	if a.height == 0 {
		return 0
	}
	// Header + breadcrumb + footer = roughly 3 lines
	return a.height - 3
}

func (a *App) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If in resource list state, route navigation to resource list first
	if a.state == StateResourceList {
		switch msg.String() {
		case "esc", "h":
			// If detail is open, close it first
			if a.resourceList.HasOpenDetail() {
				a.resourceList.CloseDetail()
				return a, nil
			}
			// Go back to home
			a.state = StateHome
			a.breadcrumb.SetPath("Home")
			a.footer.ClearPagination()
			a.footer.ClearHandlerActions()
			return a, nil
		case "q":
			return a, tea.Quit
		case ":":
			a.mode = ModeCommand
			a.commandInput.SetValue("")
			a.commandInput.Focus()
			return a, textinput.Blink
		case "m":
			// Bookmark current resource
			if res := a.resourceList.GetSelectedResource(); res != nil {
				handler := a.resourceList.Handler()
				return a, components.AddBookmark(
					a.bookmarkStore,
					res.GetName(),
					handler.ResourceType(),
					res.GetID(),
					res.GetARN(),
					a.clientMgr.Region(),
					a.clientMgr.Profile(),
				)
			}
			return a, nil
		case "'":
			// Show bookmarks
			return a, a.bookmarkSelector.Show()
		}

		// Route to resource list
		var cmd tea.Cmd
		a.resourceList, cmd = a.resourceList.Update(msg)
		return a, cmd
	}

	// Home state key handling
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		return a, tea.Quit

	case msg.String() == ":":
		a.mode = ModeCommand
		a.commandInput.SetValue("")
		a.commandInput.Focus()
		return a, textinput.Blink

	case msg.String() == "p":
		a.selector.ShowProfiles(a.profiles, a.clientMgr.Profile())
		return a, nil

	case msg.String() == "R":
		a.selector.ShowRegions(a.regions, a.clientMgr.Region())
		return a, nil

	case msg.String() == "?":
		a.footer.SetMessage("q:quit  ::command  p:profiles  R:regions  ':bookmarks  :users :roles :policies", false)
		return a, nil

	case msg.String() == "'":
		// Show bookmarks from home
		return a, a.bookmarkSelector.Show()
	}

	return a, nil
}

func (a *App) handleCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = ModeNormal
		a.commandInput.Blur()
		a.commandInput.SetValue("")
		a.autocomplete.Update("")
		return a, nil

	case "enter":
		cmd := a.commandInput.Value()
		a.mode = ModeNormal
		a.commandInput.Blur()
		a.commandInput.SetValue("")
		a.autocomplete.Update("")
		return a.executeCommand(cmd)

	case "tab":
		// Cycle through autocomplete suggestions
		if a.autocomplete.HasSuggestions() {
			a.autocomplete.Next()
			selected := a.autocomplete.Selected()
			if selected != "" {
				a.commandInput.SetValue(selected)
			}
		}
		return a, nil

	case "shift+tab":
		// Cycle backwards through autocomplete suggestions
		if a.autocomplete.HasSuggestions() {
			a.autocomplete.Previous()
			selected := a.autocomplete.Selected()
			if selected != "" {
				a.commandInput.SetValue(selected)
			}
		}
		return a, nil
	}

	var cmd tea.Cmd
	a.commandInput, cmd = a.commandInput.Update(msg)

	// Update autocomplete suggestions based on current input
	a.autocomplete.Update(a.commandInput.Value())

	return a, cmd
}

func (a *App) executeCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return a, nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "q", "quit", "exit":
		return a, tea.Quit

	case "home":
		a.state = StateHome
		a.breadcrumb.SetPath("Home")
		return a, nil

	case "profile":
		if len(args) > 0 {
			return a, a.switchProfile(args[0])
		}
		a.selector.ShowProfiles(a.profiles, a.clientMgr.Profile())
		return a, nil

	case "region":
		if len(args) > 0 {
			return a, a.switchRegion(args[0])
		}
		a.selector.ShowRegions(a.regions, a.clientMgr.Region())
		return a, nil

	case "users":
		return a.navigateToResource("users", "IAM", "Users")

	case "roles":
		return a.navigateToResource("roles", "IAM", "Roles")

	case "policies":
		return a.navigateToResource("policies", "IAM", "Policies")

	case "sg":
		return a.navigateToResource("sg", "EC2", "Security Groups")

	case "kms":
		return a.navigateToResource("kms", "KMS", "Keys")

	case "secrets":
		return a.navigateToResource("secrets", "Secrets Manager", "Secrets")

	case "ec2", "instances":
		return a.navigateToResource("ec2", "EC2", "Instances")

	case "vpc", "vpcs":
		return a.navigateToResource("vpc", "VPC", "VPCs")

	case "rds":
		return a.navigateToResource("rds", "RDS", "Instances")

	case "ecs":
		return a.navigateToResource("ecs", "ECS", "Clusters")

	case "lambda":
		return a.navigateToResource("lambda", "Lambda", "Functions")

	case "s3":
		return a.navigateToResource("s3", "S3", "Buckets")

	case "export":
		if len(args) == 0 {
			a.footer.SetMessage("Usage: :export json|yaml", true)
			return a, nil
		}
		return a.exportCurrentResource(args[0])

	case "sso", "sso-login":
		return a, a.refreshSSOSession()

	default:
		a.footer.SetMessage(fmt.Sprintf("Unknown command: %s", command), true)
		return a, nil
	}
}

func (a *App) navigateToResource(shortcut string, breadcrumbParts ...string) (tea.Model, tea.Cmd) {
	handler, ok := a.registry.Get(shortcut)
	if !ok {
		a.footer.SetMessage(fmt.Sprintf("Handler not found: %s", shortcut), true)
		return a, nil
	}

	a.state = StateResourceList
	a.breadcrumb.SetPath(breadcrumbParts...)
	a.resourceList.SetHandler(handler)
	a.footer.SetHandlerActions(handler.Actions())
	a.loading = true
	a.footer.SetLoading(true, fmt.Sprintf("Loading %s...", handler.ResourceName()))

	// Update size
	contentHeight := a.calculateContentHeight()
	a.resourceList.SetSize(a.width, contentHeight)

	return a, a.resourceList.LoadResources(context.Background(), "")
}

func (a *App) switchProfile(profile string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := a.clientMgr.SwitchProfile(ctx, profile); err != nil {
			return messages.ErrorMsg{Error: err, Context: "switching profile"}
		}

		accountID, _ := a.clientMgr.GetAccountID(ctx)

		return awsInitializedMsg{
			profile:   profile,
			region:    a.clientMgr.Region(),
			accountID: accountID,
		}
	}
}

func (a *App) switchRegion(region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := a.clientMgr.SwitchRegion(ctx, region); err != nil {
			return messages.ErrorMsg{Error: err, Context: "switching region"}
		}

		return awsInitializedMsg{
			profile:   a.clientMgr.Profile(),
			region:    region,
			accountID: "",
		}
	}
}

// ssoLoginFinishedMsg is sent when the SSO login process completes
type ssoLoginFinishedMsg struct {
	err error
}

func (a *App) refreshSSOSession() tea.Cmd {
	profile := a.clientMgr.Profile()
	c := exec.Command("aws", "sso", "login", "--profile", profile)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ssoLoginFinishedMsg{err: err}
	})
}

// ecsExecFinishedMsg is sent when the ECS exec process completes
type ecsExecFinishedMsg struct {
	err error
}

func (a *App) executeECSExec(clusterARN, taskARN, containerName string) tea.Cmd {
	cmd := exec.Command(
		"aws", "ecs", "execute-command",
		"--cluster", clusterARN,
		"--task", taskARN,
		"--container", containerName,
		"--command", "/bin/bash",
		"--interactive",
	)

	// Set AWS environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AWS_REGION=%s", a.clientMgr.Region()),
		fmt.Sprintf("AWS_PROFILE=%s", a.clientMgr.Profile()),
	)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ecsExecFinishedMsg{err: err}
	})
}

// exportCurrentResource exports the selected resource or list to a file
func (a *App) exportCurrentResource(formatStr string) (tea.Model, tea.Cmd) {
	if a.state != StateResourceList {
		a.footer.SetMessage("Export is only available in resource list view", true)
		return a, nil
	}

	var format utils.ExportFormat
	switch strings.ToLower(formatStr) {
	case "json":
		format = utils.ExportJSON
	case "yaml", "yml":
		format = utils.ExportYAML
	default:
		a.footer.SetMessage(fmt.Sprintf("Unknown format: %s. Use json or yaml", formatStr), true)
		return a, nil
	}

	handler := a.resourceList.Handler()
	if handler == nil {
		a.footer.SetMessage("No resource handler active", true)
		return a, nil
	}

	// Get selected resource or export list
	selected := a.resourceList.GetSelectedResource()
	exporter := utils.NewExporter(".")

	if selected != nil {
		// Export single resource detail
		ctx := context.Background()
		details, err := handler.Describe(ctx, selected.GetID())
		if err != nil {
			a.footer.SetMessage(fmt.Sprintf("Failed to get resource details: %v", err), true)
			return a, nil
		}

		filepath, err := exporter.Export(details, handler.ResourceType(), selected.GetID(), format)
		if err != nil {
			a.footer.SetMessage(fmt.Sprintf("Export failed: %v", err), true)
			return a, nil
		}

		a.footer.SetMessage(fmt.Sprintf("Exported to %s", filepath), false)
	} else {
		a.footer.SetMessage("No resource selected to export", true)
	}

	return a, nil
}

// navigateToBookmark navigates to a bookmarked resource
func (a *App) navigateToBookmark(bookmark config.Bookmark) (tea.Model, tea.Cmd) {
	// Get the shortcut key from resource type (e.g., "iam:users" -> "users")
	shortcut := bookmark.ResourceType
	parts := strings.Split(bookmark.ResourceType, ":")
	if len(parts) > 1 {
		shortcut = parts[1]
	}

	handler, ok := a.registry.Get(shortcut)
	if !ok {
		// Try with full type
		handler, ok = a.registry.Get(bookmark.ResourceType)
		if !ok {
			a.footer.SetMessage(fmt.Sprintf("Handler not found for: %s", bookmark.ResourceType), true)
			return a, nil
		}
	}

	// Check if we need to switch region
	if bookmark.Region != "" && bookmark.Region != a.clientMgr.Region() {
		ctx := context.Background()
		if err := a.clientMgr.SwitchRegion(ctx, bookmark.Region); err != nil {
			a.footer.SetMessage(fmt.Sprintf("Failed to switch region: %v", err), true)
			return a, nil
		}
		a.header.SetRegion(bookmark.Region)
		// Re-register handlers for new region
		a.registerHandlers()

		// Get handler again after re-registering
		handler, ok = a.registry.Get(shortcut)
		if !ok {
			handler, ok = a.registry.Get(bookmark.ResourceType)
			if !ok {
				a.footer.SetMessage(fmt.Sprintf("Handler not found for: %s", bookmark.ResourceType), true)
				return a, nil
			}
		}
	}

	// Navigate to the resource type
	a.state = StateResourceList
	a.breadcrumb.SetPath(handler.ResourceName())
	a.resourceList.SetHandler(handler)
	a.footer.SetHandlerActions(handler.Actions())
	a.loading = true
	a.footer.SetLoading(true, fmt.Sprintf("Loading %s...", handler.ResourceName()))

	// Update size
	contentHeight := a.calculateContentHeight()
	a.resourceList.SetSize(a.width, contentHeight)

	return a, a.resourceList.LoadResources(context.Background(), "")
}

// View renders the UI
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Build layout
	header := a.header.View()
	breadcrumb := a.breadcrumb.View()
	footer := a.footer.View()

	// Calculate content height
	headerHeight := lipgloss.Height(header)
	breadcrumbHeight := lipgloss.Height(breadcrumb)
	footerHeight := lipgloss.Height(footer)
	contentHeight := a.height - headerHeight - breadcrumbHeight - footerHeight

	// Render main content
	var content string
	switch a.state {
	case StateHome:
		content = a.renderHome(contentHeight)
	case StateResourceList:
		content = a.resourceList.View()
	default:
		content = a.renderHome(contentHeight)
	}

	// Add command mode overlay
	if a.mode == ModeCommand {
		content = a.overlayCommand(content, contentHeight)
	}

	// Compose the view
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		breadcrumb,
		content,
		footer,
	)

	// Overlay selector if active
	if a.selector.IsActive() {
		view = a.selector.View()
	}

	// Overlay bookmark selector if active
	if a.bookmarkSelector.IsActive() {
		view = a.bookmarkSelector.View()
	}

	return view
}

func (a *App) renderHome(height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Render("Welcome to aws-tui")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("A terminal UI for AWS resource management")

	commands := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginTop(2).
		Render(`Commands:
  :users      - List IAM Users
  :roles      - List IAM Roles
  :policies   - List IAM Policies
  :ec2        - List EC2 Instances
  :vpc        - List VPCs
  :sg         - List Security Groups
  :rds        - List RDS Instances
  :ecs        - List ECS Clusters
  :lambda     - List Lambda Functions
  :s3         - List S3 Buckets
  :kms        - List KMS Keys
  :secrets    - List Secrets
  :profile    - Switch AWS Profile
  :region     - Switch AWS Region
  :export     - Export resource (json|yaml)
  :q          - Quit

Shortcuts:
  p           - Profile selector
  R           - Region selector
  ?           - Help

Navigation:
  j/k         - Move up/down
  enter/l     - Select/Enter
  esc/h       - Back
  d           - Describe resource
  /           - Search
  t           - Filter by tags
  r           - Refresh list
  n/]         - Next page
  N/[         - Previous page
  m           - Bookmark resource
  '           - Show bookmarks
  c           - Copy ARN to clipboard
  C           - Copy JSON to clipboard`)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		subtitle,
		commands,
	)

	return lipgloss.Place(
		a.width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (a *App) overlayCommand(content string, height int) string {
	commandBox := a.theme.Command.Width(a.width).Render(a.commandInput.View())

	// Get autocomplete suggestions if available
	var autocompleteBox string
	if a.autocomplete.HasSuggestions() {
		autocompleteBox = a.autocomplete.View(a.width)
	}

	lines := strings.Split(content, "\n")
	linesToRemove := 1
	if autocompleteBox != "" {
		// Count lines in autocomplete box and remove that many additional lines
		autocompleteLines := strings.Count(autocompleteBox, "\n") + 1
		linesToRemove += autocompleteLines
	}

	if len(lines) > linesToRemove {
		lines = lines[linesToRemove:] // Remove lines to make room for command and autocomplete
	}

	result := commandBox
	if autocompleteBox != "" {
		result += "\n" + autocompleteBox
	}
	result += "\n" + strings.Join(lines, "\n")

	return result
}
