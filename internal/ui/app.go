package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	StateSecretEditor
	StateSecretCreator
)

// Mode represents vim-like modes
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeCommand
	ModeConfirm
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

	// Secret editing
	secretEditor  *components.SecretEditor
	secretCreator *components.SecretCreator
	confirmDialog *components.ConfirmDialog
	infoDialog    *components.InfoDialog
	pendingAction interface{}

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
		secretEditor:     components.NewSecretEditor(theme),
		secretCreator:    components.NewSecretCreator(theme),
		confirmDialog:    components.NewConfirmDialog(theme),
		infoDialog:       components.NewInfoDialog(theme),
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

	// Register CloudWatch Logs handlers
	a.registry.Register(handlers.NewCloudWatchLogsHandler(a.clientMgr.CloudWatchLogs(), a.clientMgr.Region()))

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
		case ModeConfirm:
			return a.handleConfirmMode(msg)
		default:
			// Handle info dialog if visible
			if a.infoDialog.IsVisible() {
				var cmd tea.Cmd
				a.infoDialog, cmd = a.infoDialog.Update(msg)
				return a, cmd
			}

			// Handle state-specific input in normal mode
			if a.state == StateSecretEditor {
				return a.handleSecretEditorMode(msg)
			}
			if a.state == StateSecretCreator {
				return a.handleSecretCreatorMode(msg)
			}
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

	// CloudWatch Logs Navigation actions
	case *handlers.NavigateToLogStreamsAction:
		handler := handlers.NewCloudWatchLogStreamsHandlerForGroup(
			a.clientMgr.CloudWatchLogs(),
			a.clientMgr.Region(),
			msg.LogGroupName,
		)
		a.state = StateResourceList
		a.breadcrumb.SetPath("CloudWatch Logs", "Log Groups", msg.LogGroupName, "Log Streams")
		a.resourceList.SetHandler(handler)
		a.footer.SetHandlerActions(handler.Actions())
		a.loading = true
		a.footer.SetLoading(true, "Loading log streams...")
		contentHeight := a.calculateContentHeight()
		a.resourceList.SetSize(a.width, contentHeight)
		return a, a.resourceList.LoadResources(context.Background(), "")

	// Secrets Manager actions
	case *handlers.ViewSecretAction:
		// Show confirmation dialog
		a.mode = ModeConfirm
		a.pendingAction = msg
		a.confirmDialog.SetMessage(fmt.Sprintf(
			"You are about to view the secret value for:\n\n%s\n\nThis will display sensitive information.",
			msg.SecretName,
		))
		a.confirmDialog.SetWidth(a.width)
		return a, nil

	case *handlers.EditSecretAction:
		// Load secret value and enter editor
		a.footer.SetLoading(true, "Loading secret...")
		return a, a.loadSecretForEditing(msg.SecretID, msg.SecretName)

	case *handlers.CreateSecretAction:
		// Activate secret creator form
		a.state = StateSecretCreator
		contentHeight := a.calculateContentHeight()
		a.secretCreator.SetSize(a.width, contentHeight)
		return a, a.secretCreator.Activate()

	case *handlers.DeleteSecretAction:
		// Show enhanced confirmation dialog with recovery window input
		a.mode = ModeConfirm
		a.pendingAction = msg
		a.confirmDialog.SetMessage(fmt.Sprintf(
			"You are about to delete the secret:\n\n%s\n\n"+
				"This will schedule the secret for deletion.\n"+
				"It can be recovered within the recovery window.",
			msg.SecretName,
		))
		a.confirmDialog.RequireInput("Recovery window (days, 7-30)", "30", 7, 30)
		a.confirmDialog.SetWidth(a.width)
		return a, nil

	// IAM Users actions
	case *handlers.ViewUserPoliciesAction:
		a.footer.SetLoading(true, "Loading policies...")
		return a, a.loadUserPolicies(msg.UserName)

	case *handlers.ViewUserGroupsAction:
		a.footer.SetLoading(true, "Loading groups...")
		return a, a.loadUserGroups(msg.UserName)

	case *handlers.ViewUserAccessKeysAction:
		a.footer.SetLoading(true, "Loading access keys...")
		return a, a.loadUserAccessKeys(msg.UserName)

	case *handlers.ViewUserMFAAction:
		a.footer.SetLoading(true, "Loading MFA devices...")
		return a, a.loadUserMFA(msg.UserName)

	// EC2 Instance actions
	case *handlers.StartInstanceAction:
		a.footer.SetLoading(true, "Starting instance...")
		return a, a.startEC2Instance(msg.InstanceID)

	case *handlers.StopInstanceAction:
		a.footer.SetLoading(true, "Stopping instance...")
		return a, a.stopEC2Instance(msg.InstanceID)

	case *handlers.RebootInstanceAction:
		a.footer.SetLoading(true, "Rebooting instance...")
		return a, a.rebootEC2Instance(msg.InstanceID)

	case *handlers.ViewConnectionInfoAction:
		a.footer.SetLoading(true, "Loading connection info...")
		return a, a.loadConnectionInfo(msg.InstanceID)

	// S3 Bucket actions
	case *handlers.ViewBucketPolicyAction:
		a.footer.SetLoading(true, "Loading bucket policy...")
		return a, a.loadBucketPolicy(msg.BucketName)

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

	// Secret operation messages
	case SecretLoadedMsg:
		// Show secret value in detail view (could enhance this with a modal)
		a.footer.SetMessage(fmt.Sprintf("Secret value: %s", msg.value), false)
		a.footer.SetLoading(false, "")
		return a, nil

	case SecretLoadedForEditMsg:
		// Enter editor mode
		a.state = StateSecretEditor
		a.secretEditor.SetSecret(msg.id, msg.name, msg.value)
		contentHeight := a.calculateContentHeight()
		a.secretEditor.SetSize(a.width, contentHeight)
		a.footer.SetLoading(false, "")
		return a, nil

	case SecretSavedMsg:
		// Return to list view
		a.state = StateResourceList
		a.footer.SetMessage("Secret updated successfully", false)
		a.footer.SetLoading(false, "")
		// Refresh the list
		return a, a.resourceList.LoadResources(context.Background(), "")

	case SecretSaveErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Failed to save: %v", msg.err), true)
		a.footer.SetLoading(false, "")
		return a, nil

	case SecretLoadErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Failed to load: %v", msg.err), true)
		a.footer.SetLoading(false, "")
		a.mode = ModeNormal
		a.pendingAction = nil
		return a, nil

	case SecretCreatedMsg:
		a.state = StateResourceList
		a.footer.SetMessage(fmt.Sprintf("Secret '%s' created successfully", msg.secretName), false)
		a.footer.SetLoading(false, "")
		a.secretCreator.Reset()
		// Refresh the list
		return a, a.resourceList.LoadResources(context.Background(), "")

	case SecretCreateErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Failed to create secret: %v", msg.err), true)
		a.footer.SetLoading(false, "")
		return a, nil

	case SecretDeletedMsg:
		a.footer.SetMessage(fmt.Sprintf("Secret '%s' scheduled for deletion", msg.secretID), false)
		a.footer.SetLoading(false, "")
		// Refresh the list
		return a, a.resourceList.LoadResources(context.Background(), "")

	case SecretDeleteErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Failed to delete secret: %v", msg.err), true)
		a.footer.SetLoading(false, "")
		return a, nil

	// IAM User data messages
	case UserDataLoadedMsg:
		a.footer.SetLoading(false, "")
		a.infoDialog.SetSize(a.width, a.height)
		a.infoDialog.Show(msg.title, msg.data)
		return a, nil

	case UserDataErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Failed to load data: %v", msg.err), true)
		a.footer.SetLoading(false, "")
		return a, nil

	// EC2 Instance operation messages
	case EC2InstanceOperationSuccessMsg:
		a.footer.SetMessage(msg.message, false)
		a.footer.SetLoading(false, "")
		// Refresh the list to show updated state
		return a, a.resourceList.Refresh()

	case EC2InstanceOperationErrorMsg:
		a.footer.SetMessage(fmt.Sprintf("Operation failed: %v", msg.err), true)
		a.footer.SetLoading(false, "")
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
		a.footer.SetMessage("q:quit  ::command  p:profiles  R:regions  ':bookmarks  :users :roles :policies :logs", false)
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

	case "logs":
		return a.navigateToResource("logs", "CloudWatch Logs", "Log Groups")

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
	case StateSecretEditor:
		content = a.secretEditor.View()
	case StateSecretCreator:
		content = a.secretCreator.View()
	default:
		content = a.renderHome(contentHeight)
	}

	// Add command mode overlay
	if a.mode == ModeCommand {
		content = a.overlayCommand(content, contentHeight)
	}

	// Add confirmation dialog overlay
	if a.mode == ModeConfirm {
		content = a.overlayConfirm(content)
	}

	// Compose the view
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		breadcrumb,
		content,
		footer,
	)

	// Overlay info dialog if visible
	if a.infoDialog.IsVisible() {
		view = a.infoDialog.View()
	}

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
  :logs       - List CloudWatch Log Groups
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

func (a *App) overlayConfirm(content string) string {
	// Center the dialog
	lines := strings.Split(content, "\n")
	if len(lines) > 5 {
		lines = lines[5:]
	}

	dialog := a.confirmDialog.View()
	result := dialog + "\n" + strings.Join(lines, "\n")

	return result
}

// Message types for secret operations
type SecretLoadedMsg struct {
	name  string
	value string
}

type SecretLoadedForEditMsg struct {
	id    string
	name  string
	value string
}

type SecretLoadErrorMsg struct {
	err error
}

type SecretSavedMsg struct {
	secretID string
}

type SecretSaveErrorMsg struct {
	err error
}

// Secret creation messages
type SecretCreatedMsg struct {
	secretName string
}

type SecretCreateErrorMsg struct {
	err error
}

// Secret deletion messages
type SecretDeletedMsg struct {
	secretID string
}

type SecretDeleteErrorMsg struct {
	err error
}

// IAM User data messages
type UserDataLoadedMsg struct {
	title string
	data  interface{}
}

type UserDataErrorMsg struct {
	err error
}

// EC2 Instance operation messages
type EC2InstanceOperationSuccessMsg struct {
	message string
}

type EC2InstanceOperationErrorMsg struct {
	err error
}

// handleConfirmMode handles confirmation dialog input
func (a *App) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// User confirmed
		a.mode = ModeNormal

		if deleteAction, ok := a.pendingAction.(*handlers.DeleteSecretAction); ok {
			// Get recovery window from dialog input
			recoveryWindow := 30 // default
			if input := a.confirmDialog.GetInput(); input != "" {
				if val, err := strconv.Atoi(input); err == nil {
					if val < 7 || val > 30 {
						a.footer.SetMessage("Recovery window must be 7-30 days", true)
						return a, nil
					}
					recoveryWindow = val
				} else {
					a.footer.SetMessage("Invalid recovery window (must be a number)", true)
					return a, nil
				}
			}
			a.pendingAction = nil
			a.confirmDialog.Reset()
			return a, a.deleteSecret(deleteAction.SecretID, deleteAction.SecretName, recoveryWindow)
		}

		if viewAction, ok := a.pendingAction.(*handlers.ViewSecretAction); ok {
			a.pendingAction = nil
			a.confirmDialog.Reset()
			return a, a.loadAndViewSecret(viewAction.SecretID, viewAction.SecretName)
		}

		a.pendingAction = nil
		a.confirmDialog.Reset()
		return a, nil

	case "n", "N", "esc":
		// User cancelled
		a.mode = ModeNormal
		a.pendingAction = nil
		a.confirmDialog.Reset()
		return a, nil

	default:
		// Route input to confirm dialog if it has input field
		if a.confirmDialog.HasInput() {
			var cmd tea.Cmd
			a.confirmDialog, cmd = a.confirmDialog.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

// handleSecretEditorMode handles secret editor input
func (a *App) handleSecretEditorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel editing
		a.state = StateResourceList
		return a, nil

	case "ctrl+s":
		// Save secret
		a.footer.SetLoading(true, "Saving secret...")
		return a, a.saveSecret()
	}

	// Pass other keys to editor
	var cmd tea.Cmd
	a.secretEditor, cmd = a.secretEditor.Update(msg)
	return a, cmd
}

// handleSecretCreatorMode handles secret creator input
func (a *App) handleSecretCreatorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel creation
		a.state = StateResourceList
		a.secretCreator.Reset()
		return a, nil

	case "ctrl+s":
		// Submit form
		if err := a.secretCreator.Validate(); err != nil {
			a.footer.SetMessage("Please fix validation errors", true)
			return a, nil
		}
		a.footer.SetLoading(true, "Creating secret...")
		params := a.secretCreator.GetParams()
		return a, a.createSecret(params)
	}

	// Pass to creator for field handling
	var cmd tea.Cmd
	a.secretCreator, cmd = a.secretCreator.Update(msg)
	return a, cmd
}

// loadAndViewSecret loads a secret value for viewing
func (a *App) loadAndViewSecret(secretID, secretName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("secrets")
		if !ok {
			return SecretLoadErrorMsg{err: fmt.Errorf("secrets handler not found")}
		}

		secretsHandler, ok := handler.(*handlers.SecretsHandler)
		if !ok {
			return SecretLoadErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		value, err := secretsHandler.GetSecretValueForView(ctx, secretID)
		if err != nil {
			return SecretLoadErrorMsg{err: err}
		}

		return SecretLoadedMsg{
			name:  secretName,
			value: value,
		}
	}
}

// loadSecretForEditing loads a secret value for editing
func (a *App) loadSecretForEditing(secretID, secretName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("secrets")
		if !ok {
			return SecretLoadErrorMsg{err: fmt.Errorf("secrets handler not found")}
		}

		secretsHandler, ok := handler.(*handlers.SecretsHandler)
		if !ok {
			return SecretLoadErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		value, err := secretsHandler.GetSecretValueForEdit(ctx, secretID)
		if err != nil {
			return SecretLoadErrorMsg{err: err}
		}

		return SecretLoadedForEditMsg{
			id:    secretID,
			name:  secretName,
			value: value,
		}
	}
}

// saveSecret saves the current secret being edited
func (a *App) saveSecret() tea.Cmd {
	return func() tea.Msg {
		value, err := a.secretEditor.Value()
		if err != nil {
			return SecretSaveErrorMsg{err: err}
		}

		ctx := context.Background()
		handler, ok := a.registry.Get("secrets")
		if !ok {
			return SecretSaveErrorMsg{err: fmt.Errorf("secrets handler not found")}
		}

		secretsHandler, ok := handler.(*handlers.SecretsHandler)
		if !ok {
			return SecretSaveErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		secretID := a.secretEditor.GetSecretID()
		updates := map[string]interface{}{
			"SecretValue": value,
		}

		if err := secretsHandler.Update(ctx, secretID, updates); err != nil {
			return SecretSaveErrorMsg{err: err}
		}

		return SecretSavedMsg{secretID: secretID}
	}
}

func (a *App) createSecret(params map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("secrets")
		if !ok {
			return SecretCreateErrorMsg{err: fmt.Errorf("secrets handler not found")}
		}

		secretsHandler, ok := handler.(*handlers.SecretsHandler)
		if !ok {
			return SecretCreateErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		_, err := secretsHandler.Create(ctx, params)
		if err != nil {
			return SecretCreateErrorMsg{err: err}
		}

		secretName, _ := params["Name"].(string)
		return SecretCreatedMsg{secretName: secretName}
	}
}

func (a *App) deleteSecret(secretID, secretName string, recoveryWindowDays int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("secrets")
		if !ok {
			return SecretDeleteErrorMsg{err: fmt.Errorf("secrets handler not found")}
		}

		secretsHandler, ok := handler.(*handlers.SecretsHandler)
		if !ok {
			return SecretDeleteErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		err := secretsHandler.DeleteWithRecoveryWindow(ctx, secretID, recoveryWindowDays)
		if err != nil {
			return SecretDeleteErrorMsg{err: err}
		}

		return SecretDeletedMsg{secretID: secretID}
	}
}

// IAM User data loading functions

func (a *App) loadUserPolicies(userName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("users")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("users handler not found")}
		}

		usersHandler, ok := handler.(*handlers.IAMUsersHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := usersHandler.GetUserPolicies(ctx, userName)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("Policies for User: %s", userName),
			data:  data,
		}
	}
}

func (a *App) loadUserGroups(userName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("users")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("users handler not found")}
		}

		usersHandler, ok := handler.(*handlers.IAMUsersHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := usersHandler.GetUserGroups(ctx, userName)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("Groups for User: %s", userName),
			data:  data,
		}
	}
}

func (a *App) loadUserAccessKeys(userName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("users")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("users handler not found")}
		}

		usersHandler, ok := handler.(*handlers.IAMUsersHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := usersHandler.GetUserAccessKeys(ctx, userName)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("Access Keys for User: %s", userName),
			data:  data,
		}
	}
}

func (a *App) loadUserMFA(userName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("users")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("users handler not found")}
		}

		usersHandler, ok := handler.(*handlers.IAMUsersHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := usersHandler.GetUserMFADevices(ctx, userName)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("MFA Devices for User: %s", userName),
			data:  data,
		}
	}
}

// EC2 Instance operation functions

func (a *App) startEC2Instance(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("ec2")
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("EC2 handler not found")}
		}

		ec2Handler, ok := handler.(*handlers.EC2InstancesHandler)
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		err := ec2Handler.StartInstance(ctx, instanceID)
		if err != nil {
			return EC2InstanceOperationErrorMsg{err: err}
		}

		return EC2InstanceOperationSuccessMsg{
			message: fmt.Sprintf("Instance %s is starting", instanceID),
		}
	}
}

func (a *App) stopEC2Instance(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("ec2")
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("EC2 handler not found")}
		}

		ec2Handler, ok := handler.(*handlers.EC2InstancesHandler)
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		err := ec2Handler.StopInstance(ctx, instanceID)
		if err != nil {
			return EC2InstanceOperationErrorMsg{err: err}
		}

		return EC2InstanceOperationSuccessMsg{
			message: fmt.Sprintf("Instance %s is stopping", instanceID),
		}
	}
}

func (a *App) rebootEC2Instance(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("ec2")
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("EC2 handler not found")}
		}

		ec2Handler, ok := handler.(*handlers.EC2InstancesHandler)
		if !ok {
			return EC2InstanceOperationErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		err := ec2Handler.RebootInstance(ctx, instanceID)
		if err != nil {
			return EC2InstanceOperationErrorMsg{err: err}
		}

		return EC2InstanceOperationSuccessMsg{
			message: fmt.Sprintf("Instance %s is rebooting", instanceID),
		}
	}
}

func (a *App) loadConnectionInfo(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("ec2")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("EC2 handler not found")}
		}

		ec2Handler, ok := handler.(*handlers.EC2InstancesHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := ec2Handler.GetConnectionInfo(ctx, instanceID)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("Connection Info for Instance: %s", instanceID),
			data:  data,
		}
	}
}

// S3 Bucket operation functions

func (a *App) loadBucketPolicy(bucketName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		handler, ok := a.registry.Get("s3")
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("S3 handler not found")}
		}

		s3Handler, ok := handler.(*handlers.S3BucketsHandler)
		if !ok {
			return UserDataErrorMsg{err: fmt.Errorf("invalid handler type")}
		}

		data, err := s3Handler.GetBucketPolicyForView(ctx, bucketName)
		if err != nil {
			return UserDataErrorMsg{err: err}
		}

		return UserDataLoadedMsg{
			title: fmt.Sprintf("Bucket Policy for: %s", bucketName),
			data:  data,
		}
	}
}
