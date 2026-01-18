package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/handlers"
	"github.com/aaw-tui/aws-tui/internal/ui/keys"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// Footer displays the bottom bar with help and status
type Footer struct {
	width      int
	theme      styles.Theme
	keys       keys.KeyMap
	message    string
	messageErr bool
	loading    bool
	loadingMsg string
	// Pagination
	page    int
	hasMore bool
	count   int
	// Handler actions for context-specific hints
	handlerActions []handlers.Action
}

// NewFooter creates a new footer component
func NewFooter(theme styles.Theme, keyMap keys.KeyMap) *Footer {
	return &Footer{
		theme: theme,
		keys:  keyMap,
	}
}

// SetWidth sets the footer width
func (f *Footer) SetWidth(width int) {
	f.width = width
}

// SetMessage sets a status message
func (f *Footer) SetMessage(msg string, isError bool) {
	f.message = msg
	f.messageErr = isError
}

// ClearMessage clears the status message
func (f *Footer) ClearMessage() {
	f.message = ""
	f.messageErr = false
}

// SetLoading sets the loading state
func (f *Footer) SetLoading(loading bool, msg string) {
	f.loading = loading
	f.loadingMsg = msg
}

// SetPagination sets the pagination info
func (f *Footer) SetPagination(page int, hasMore bool, count int) {
	f.page = page
	f.hasMore = hasMore
	f.count = count
}

// ClearPagination clears the pagination info
func (f *Footer) ClearPagination() {
	f.page = 0
	f.hasMore = false
	f.count = 0
}

// SetHandlerActions sets the handler actions for context-specific hints
func (f *Footer) SetHandlerActions(actions []handlers.Action) {
	f.handlerActions = actions
}

// ClearHandlerActions clears the handler actions
func (f *Footer) ClearHandlerActions() {
	f.handlerActions = nil
}

// View renders the footer
func (f *Footer) View() string {
	// If loading, show loading indicator
	if f.loading {
		spinner := "⣾⣽⣻⢿⡿⣟⣯⣷"
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
		msg := f.loadingMsg
		if msg == "" {
			msg = "Loading..."
		}
		content := loadingStyle.Render(string(spinner[0])) + " " + msg
		return f.theme.Footer.Width(f.width).Render(content)
	}

	// If there's a message, show it
	if f.message != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if f.messageErr {
			style = style.Foreground(lipgloss.Color("196"))
		}
		return f.theme.Footer.Width(f.width).Render(style.Render(f.message))
	}

	// Show help hints
	hints := f.buildHelpHints()
	return f.theme.Footer.Width(f.width).Render(hints)
}

func (f *Footer) buildHelpHints() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	hints := []string{
		fmt.Sprintf("%s %s", keyStyle.Render("j/k"), descStyle.Render("nav")),
		fmt.Sprintf("%s %s", keyStyle.Render("Ctrl+R"), descStyle.Render("refresh")),
	}

	// Add handler-specific action hints if available
	if len(f.handlerActions) > 0 {
		for _, action := range f.handlerActions {
			hints = append(hints, fmt.Sprintf("%s %s", keyStyle.Render(action.Key), descStyle.Render(action.Description)))
		}
	}

	// Add common hints
	hints = append(hints,
		fmt.Sprintf("%s %s", keyStyle.Render("/"), descStyle.Render("search")),
		fmt.Sprintf("%s %s", keyStyle.Render("o"), descStyle.Render("sort")),
		fmt.Sprintf("%s %s", keyStyle.Render(":"), descStyle.Render("cmd")),
		fmt.Sprintf("%s %s", keyStyle.Render("d"), descStyle.Render("describe")),
		fmt.Sprintf("%s %s", keyStyle.Render("c"), descStyle.Render("copy")),
		fmt.Sprintf("%s %s", keyStyle.Render("?"), descStyle.Render("help")),
		fmt.Sprintf("%s %s", keyStyle.Render("q"), descStyle.Render("quit")),
	)

	helpHints := strings.Join(hints, sepStyle.Render(" │ "))

	// Add pagination info if present
	if f.page > 0 {
		pageStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

		pageInfo := fmt.Sprintf("Page %d", f.page)
		if f.hasMore {
			pageInfo += "+"
		}
		pageInfo += fmt.Sprintf(" (%d items)", f.count)

		// Add pagination hints
		if f.hasMore || f.page > 1 {
			navHint := ""
			if f.hasMore {
				navHint = keyStyle.Render("n") + descStyle.Render("/") + keyStyle.Render("]") + descStyle.Render(":next")
			}
			if f.page > 1 {
				if navHint != "" {
					navHint += " "
				}
				navHint += keyStyle.Render("N") + descStyle.Render("/") + keyStyle.Render("[") + descStyle.Render(":prev")
			}
			pageInfo = pageStyle.Render(pageInfo) + " " + navHint
		} else {
			pageInfo = pageStyle.Render(pageInfo)
		}

		// Right-align pagination info
		helpLen := lipgloss.Width(helpHints)
		pageLen := lipgloss.Width(pageInfo)
		padding := f.width - helpLen - pageLen - 4
		if padding > 0 {
			return helpHints + strings.Repeat(" ", padding) + pageInfo
		}
		return helpHints + " " + pageInfo
	}

	return helpHints
}
