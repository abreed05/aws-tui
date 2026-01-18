package views

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/handlers"
	"github.com/aaw-tui/aws-tui/internal/ui/components"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// ResourcesLoadedMsg indicates resources have been loaded
type ResourcesLoadedMsg struct {
	Resources []handlers.Resource
	NextToken string
	Error     error
}

// ResourceDetailLoadedMsg indicates resource details have been loaded
type ResourceDetailLoadedMsg struct {
	Details map[string]interface{}
	Error   error
}

// ActionMsg is a message returned by ExecuteAction to trigger navigation
type ActionMsg interface {
	error
	IsActionMsg()
}

// ActionErrorMsg indicates an action failed
type ActionErrorMsg struct {
	Error  error
	Action string
}

// ResourceListView displays a list of resources with optional detail pane
type ResourceListView struct {
	handler handlers.ResourceHandler
	table   *components.Table
	detail  *components.Detail
	search  *components.Search
	tagFilter *components.TagFilter

	// State
	resources       []handlers.Resource
	filteredByTags  []handlers.Resource
	activeTags      map[string]string
	loading         bool
	error           error
	showDetail      bool
	detailFocus     bool

	// Pagination state
	nextToken    string
	prevTokens   []string // Stack of previous tokens for back navigation
	currentPage  int
	hasMore      bool
	totalLoaded  int

	// Dimensions
	width  int
	height int

	// Theme
	theme styles.Theme
}

// NewResourceListView creates a new resource list view
func NewResourceListView(theme styles.Theme) *ResourceListView {
	return &ResourceListView{
		table:      components.NewTable(theme),
		detail:     components.NewDetail(theme),
		search:     components.NewSearch(theme),
		tagFilter:  components.NewTagFilter(theme),
		activeTags: make(map[string]string),
		theme:      theme,
	}
}

// SetHandler sets the resource handler
func (v *ResourceListView) SetHandler(handler handlers.ResourceHandler) {
	v.handler = handler
	v.table.SetColumns(handler.Columns())
	v.resources = nil
	v.filteredByTags = nil
	v.activeTags = make(map[string]string)
	v.tagFilter.ClearFilters()
	v.detail.Clear()
	v.showDetail = false
	// Reset pagination
	v.nextToken = ""
	v.prevTokens = nil
	v.currentPage = 1
	v.hasMore = false
	v.totalLoaded = 0
}

// SetSize sets the view dimensions
func (v *ResourceListView) SetSize(width, height int) {
	v.width = width
	v.height = height

	v.search.SetWidth(width)
	v.tagFilter.SetSize(width, height)

	if v.showDetail {
		// Split view: 60% table, 40% detail
		tableWidth := width * 6 / 10
		detailWidth := width - tableWidth - 1

		v.table.SetSize(tableWidth, height-2)
		v.detail.SetSize(detailWidth, height-2)
	} else {
		v.table.SetSize(width, height-2)
	}
}

// LoadResources loads resources from the handler
func (v *ResourceListView) LoadResources(ctx context.Context, filter string) tea.Cmd {
	return v.loadResourcesWithToken(ctx, filter, "")
}

// loadResourcesWithToken loads resources with a specific pagination token
func (v *ResourceListView) loadResourcesWithToken(ctx context.Context, filter, token string) tea.Cmd {
	if v.handler == nil {
		return nil
	}

	v.loading = true

	return func() tea.Msg {
		result, err := v.handler.List(ctx, handlers.ListOptions{
			Filter:    filter,
			NextToken: token,
			PageSize:  50, // Default page size
		})
		if err != nil {
			return ResourcesLoadedMsg{Error: err}
		}
		return ResourcesLoadedMsg{
			Resources: result.Resources,
			NextToken: result.NextToken,
		}
	}
}

// LoadNextPage loads the next page of resources
func (v *ResourceListView) LoadNextPage() tea.Cmd {
	if !v.hasMore || v.nextToken == "" {
		return nil
	}

	// Save current token for back navigation
	if v.currentPage == 1 {
		v.prevTokens = []string{""}
	} else if len(v.prevTokens) > 0 {
		// We need to track the token that got us to the current page
	}

	// Store the token that gets us to current page before loading next
	currentToken := ""
	if v.currentPage > 1 && len(v.prevTokens) >= v.currentPage-1 {
		currentToken = v.prevTokens[v.currentPage-1]
	}
	if len(v.prevTokens) < v.currentPage {
		v.prevTokens = append(v.prevTokens, currentToken)
	}

	v.currentPage++
	return v.loadResourcesWithToken(context.Background(), "", v.nextToken)
}

// LoadPrevPage loads the previous page of resources
func (v *ResourceListView) LoadPrevPage() tea.Cmd {
	if v.currentPage <= 1 {
		return nil
	}

	v.currentPage--
	token := ""
	if v.currentPage > 1 && len(v.prevTokens) >= v.currentPage-1 {
		token = v.prevTokens[v.currentPage-1]
	}

	return v.loadResourcesWithToken(context.Background(), "", token)
}

// LoadResourceDetail loads details for the selected resource
func (v *ResourceListView) LoadResourceDetail(ctx context.Context) tea.Cmd {
	if v.handler == nil {
		return nil
	}

	selected := v.table.SelectedResource()
	if selected == nil {
		return nil
	}

	return func() tea.Msg {
		details, err := v.handler.Describe(ctx, selected.GetID())
		if err != nil {
			return ResourceDetailLoadedMsg{Error: err}
		}
		return ResourceDetailLoadedMsg{Details: details}
	}
}

// Update handles messages
func (v *ResourceListView) Update(msg tea.Msg) (*ResourceListView, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ResourcesLoadedMsg:
		v.loading = false
		if msg.Error != nil {
			v.error = msg.Error
		} else {
			v.error = nil
			v.resources = msg.Resources
			v.totalLoaded = len(msg.Resources)
			v.nextToken = msg.NextToken
			v.hasMore = msg.NextToken != ""

			v.tagFilter.SetResources(msg.Resources)
			// Apply any existing tag filters
			if len(v.activeTags) > 0 {
				v.filteredByTags = components.FilterByTags(msg.Resources, v.activeTags)
				v.table.SetResources(v.filteredByTags)
				v.search.SetResults(len(v.filteredByTags), len(msg.Resources))
			} else {
				v.filteredByTags = msg.Resources
				v.table.SetResources(msg.Resources)
				v.search.SetResults(len(msg.Resources), len(msg.Resources))
			}
		}
		return v, nil

	case components.TagFilterUpdateMsg:
		v.activeTags = msg.Tags
		if len(msg.Tags) > 0 {
			v.filteredByTags = components.FilterByTags(v.resources, msg.Tags)
		} else {
			v.filteredByTags = v.resources
		}
		v.table.SetResources(v.filteredByTags)
		v.search.SetResults(len(v.filteredByTags), len(v.resources))
		return v, nil

	case components.TagFilterClosedMsg:
		v.activeTags = msg.Tags
		v.table.Focus()
		return v, nil

	case components.ClipboardCopiedMsg:
		// Clipboard message is handled by the main app for status display
		return v, nil

	case ResourceDetailLoadedMsg:
		if msg.Error != nil {
			v.error = msg.Error
		} else {
			v.error = nil
			v.detail.SetContent(msg.Details)
			v.showDetail = true
			v.SetSize(v.width, v.height) // Recalculate sizes
		}
		return v, nil

	case components.SearchUpdateMsg:
		v.table.ApplyFilter(msg.Query)
		v.search.SetResults(v.table.Len(), len(v.resources))
		return v, nil

	case components.SearchClosedMsg:
		// Ensure search is deactivated
		v.search.Deactivate()
		if msg.Query != "" {
			v.table.ApplyFilter(msg.Query)
		} else {
			// Clear filter if query is empty
			v.table.ApplyFilter("")
		}
		v.table.Focus()
		return v, nil

	case components.ResourceSelectedMsg:
		// Resource selected, load details
		v.showDetail = true
		v.SetSize(v.width, v.height)
		return v, v.LoadResourceDetail(context.Background())

	case tea.KeyMsg:
		// Handle search activation
		if msg.String() == "/" && !v.search.IsActive() {
			v.table.Blur()
			return v, v.search.Activate()
		}

		// Handle copy ARN/ID (only if handler doesn't use 'c' for an action)
		if msg.String() == "c" && !v.search.IsActive() {
			// Check if handler has a 'c' action
			hasCreateAction := false
			if v.handler != nil {
				for _, action := range v.handler.Actions() {
					if action.Key == "c" {
						hasCreateAction = true
						break
					}
				}
			}

			// Only copy ARN if handler doesn't use 'c'
			if !hasCreateAction {
				if res := v.table.SelectedResource(); res != nil {
					return v, components.CopyToClipboard(res.GetARN(), "ARN")
				}
				return v, nil
			}
		}

		// Handle copy full resource as JSON
		if msg.String() == "C" && !v.search.IsActive() {
			if v.showDetail && v.detail != nil {
				jsonStr := v.detail.GetJSON()
				if jsonStr != "" {
					return v, components.CopyToClipboard(jsonStr, "JSON")
				}
			} else if res := v.table.SelectedResource(); res != nil {
				// Copy basic resource info if detail not loaded
				return v, components.CopyToClipboard(res.GetARN(), "ARN")
			}
			return v, nil
		}

		// Handle detail toggle
		if msg.String() == "d" && !v.search.IsActive() {
			if v.showDetail {
				v.showDetail = false
				v.detailFocus = false
				v.detail.Clear()
				v.detail.Blur()
				v.table.Focus()
				v.SetSize(v.width, v.height)
			} else {
				return v, v.LoadResourceDetail(context.Background())
			}
			return v, nil
		}

		// Handle tab to switch focus
		if msg.String() == "tab" && v.showDetail {
			v.detailFocus = !v.detailFocus
			if v.detailFocus {
				v.table.Blur()
				v.detail.Focus()
			} else {
				v.detail.Blur()
				v.table.Focus()
			}
			return v, nil
		}

		// Handle escape to close search
		if msg.String() == "esc" && v.search.IsActive() {
			v.search.Deactivate()
			v.search.Clear()
			v.table.Focus()
			return v, nil
		}

		// Handle escape to close detail
		if msg.String() == "esc" && v.showDetail && !v.search.IsActive() {
			v.showDetail = false
			v.detailFocus = false
			v.detail.Clear()
			v.detail.Blur()
			v.table.Focus()
			v.SetSize(v.width, v.height)
			return v, nil
		}

		// Handle refresh with Ctrl+R (universal refresh)
		if msg.String() == "ctrl+r" && !v.search.IsActive() && !v.tagFilter.IsActive() {
			return v, v.Refresh()
		}

		// Handle refresh with 'r' (only if handler doesn't use 'r' for an action)
		if msg.String() == "r" && !v.search.IsActive() && !v.tagFilter.IsActive() {
			// Check if handler has an 'r' action
			hasRotationAction := false
			if v.handler != nil {
				for _, action := range v.handler.Actions() {
					if action.Key == "r" {
						hasRotationAction = true
						break
					}
				}
			}

			// Only refresh if handler doesn't use 'r'
			if !hasRotationAction {
				return v, v.Refresh()
			}
		}

		// Handle actions (s, t, x, etc.) - check handler actions first
		if !v.search.IsActive() && !v.tagFilter.IsActive() && v.handler != nil {
			actions := v.handler.Actions()
			for _, action := range actions {
				if msg.String() == action.Key {
					// Get selected resource
					if res := v.table.SelectedResource(); res != nil {
						// Execute action on handler
						ctx := context.Background()
						err := v.handler.ExecuteAction(ctx, action.Name, res.GetID())
						if err != nil {
							// Check if it's a special navigation action
							if navAction, ok := err.(ActionMsg); ok {
								return v, func() tea.Msg { return navAction }
							}
							// Regular error
							return v, func() tea.Msg {
								return ActionErrorMsg{Error: err, Action: action.Name}
							}
						}
					}
					return v, nil
				}
			}
		}

		// Handle sorting
		if msg.String() == "o" && !v.search.IsActive() && !v.tagFilter.IsActive() {
			v.table.CycleSortColumn()
			return v, nil
		}

		if msg.String() == "O" && !v.search.IsActive() && !v.tagFilter.IsActive() {
			v.table.ToggleSortDirection()
			return v, nil
		}

		// Handle tag filter activation (only if no action conflicts)
		if msg.String() == "t" && !v.search.IsActive() && !v.tagFilter.IsActive() {
			// Check if 't' is used by an action first
			hasTagsAction := false
			if v.handler != nil {
				for _, action := range v.handler.Actions() {
					if action.Key == "t" {
						hasTagsAction = true
						break
					}
				}
			}
			if !hasTagsAction {
				v.table.Blur()
				return v, v.tagFilter.Activate()
			}
		}

		// Handle pagination - next page
		if (msg.String() == "n" || msg.String() == "]") && !v.search.IsActive() && !v.tagFilter.IsActive() {
			if v.hasMore {
				return v, v.LoadNextPage()
			}
			return v, nil
		}

		// Handle pagination - previous page
		if (msg.String() == "N" || msg.String() == "[") && !v.search.IsActive() && !v.tagFilter.IsActive() {
			if v.currentPage > 1 {
				return v, v.LoadPrevPage()
			}
			return v, nil
		}

		// Route to tag filter if active
		if v.tagFilter.IsActive() {
			var cmd tea.Cmd
			v.tagFilter, cmd = v.tagFilter.Update(msg)
			cmds = append(cmds, cmd)
			return v, tea.Batch(cmds...)
		}

		// Route to search if active
		if v.search.IsActive() {
			var cmd tea.Cmd
			v.search, cmd = v.search.Update(msg)
			cmds = append(cmds, cmd)
			return v, tea.Batch(cmds...)
		}

		// Route to detail if focused
		if v.detailFocus && v.showDetail {
			var cmd tea.Cmd
			v.detail, cmd = v.detail.Update(msg)
			cmds = append(cmds, cmd)
			return v, tea.Batch(cmds...)
		}

		// Route to table
		var cmd tea.Cmd
		v.table, cmd = v.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *ResourceListView) View() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Loading state
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
		return lipgloss.Place(
			v.width,
			v.height,
			lipgloss.Center,
			lipgloss.Center,
			loadingStyle.Render("Loading..."),
		)
	}

	// Error state
	if v.error != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		return lipgloss.Place(
			v.width,
			v.height,
			lipgloss.Center,
			lipgloss.Center,
			errorStyle.Render(fmt.Sprintf("Error: %v", v.error)),
		)
	}

	// Build content
	var content string

	if v.showDetail {
		// Split view
		tableView := v.table.View()
		detailView := v.detail.View()

		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("│")

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			tableView,
			separator,
			detailView,
		)
	} else {
		content = v.table.View()
	}

	// Overlay search if active
	if v.search.IsActive() {
		searchView := v.search.View()
		searchBox := lipgloss.Place(
			v.width,
			3,
			lipgloss.Left,
			lipgloss.Top,
			searchView,
		)

		// Combine search overlay with content
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			searchBox,
			content,
		)
	}

	// Overlay tag filter if active
	if v.tagFilter.IsActive() {
		return v.tagFilter.View()
	}

	return content
}

// IsLoading returns whether the view is loading
func (v *ResourceListView) IsLoading() bool {
	return v.loading
}

// GetError returns any error
func (v *ResourceListView) GetError() error {
	return v.error
}

// GetSelectedResource returns the selected resource
func (v *ResourceListView) GetSelectedResource() handlers.Resource {
	return v.table.SelectedResource()
}

// Handler returns the current handler
func (v *ResourceListView) Handler() handlers.ResourceHandler {
	return v.handler
}

// Refresh reloads the current resources from the first page
func (v *ResourceListView) Refresh() tea.Cmd {
	if v.handler == nil {
		return nil
	}
	v.loading = true
	v.detailFocus = false
	v.detail.Clear()
	v.showDetail = false
	v.table.Focus()
	// Reset pagination
	v.currentPage = 1
	v.prevTokens = nil
	v.nextToken = ""
	v.hasMore = false
	v.SetSize(v.width, v.height)
	return v.LoadResources(context.Background(), "")
}

// HasOpenDetail returns true if the detail pane is currently visible
func (v *ResourceListView) HasOpenDetail() bool {
	return v.showDetail
}

// CloseDetail closes the detail pane and returns focus to the table
func (v *ResourceListView) CloseDetail() {
	if v.showDetail {
		v.showDetail = false
		v.detailFocus = false
		v.detail.Clear()
		v.detail.Blur()
		v.table.Focus()
		v.SetSize(v.width, v.height)
	}
}

// GetPaginationInfo returns current page, hasMore, and count for display
func (v *ResourceListView) GetPaginationInfo() (page int, hasMore bool, count int) {
	return v.currentPage, v.hasMore, v.totalLoaded
}

// HasPagination returns true if there's more than one page of data
func (v *ResourceListView) HasPagination() bool {
	return v.hasMore || v.currentPage > 1
}
