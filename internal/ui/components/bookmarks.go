package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/adapters/config"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// BookmarkSelectedMsg is sent when a bookmark is selected
type BookmarkSelectedMsg struct {
	Bookmark config.Bookmark
}

// BookmarkClosedMsg is sent when bookmark selector is closed
type BookmarkClosedMsg struct{}

// BookmarkAddedMsg is sent when a bookmark is added
type BookmarkAddedMsg struct {
	Success bool
	Name    string
	Error   error
}

// BookmarkRemovedMsg is sent when a bookmark is removed
type BookmarkRemovedMsg struct {
	Success bool
	Error   error
}

// BookmarkSelector displays and manages bookmarks
type BookmarkSelector struct {
	theme  styles.Theme
	store  *config.BookmarkStore
	active bool
	cursor int
	width  int
	height int
}

// NewBookmarkSelector creates a new bookmark selector
func NewBookmarkSelector(theme styles.Theme, store *config.BookmarkStore) *BookmarkSelector {
	return &BookmarkSelector{
		theme: theme,
		store: store,
	}
}

// Show activates the bookmark selector
func (b *BookmarkSelector) Show() tea.Cmd {
	b.active = true
	b.cursor = 0
	return nil
}

// Hide deactivates the bookmark selector
func (b *BookmarkSelector) Hide() {
	b.active = false
}

// IsActive returns whether the selector is active
func (b *BookmarkSelector) IsActive() bool {
	return b.active
}

// SetSize sets the dimensions
func (b *BookmarkSelector) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// Update handles messages
func (b *BookmarkSelector) Update(msg tea.Msg) (*BookmarkSelector, tea.Cmd) {
	if !b.active {
		return b, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		bookmarks := b.store.List()

		switch msg.String() {
		case "esc", "q", "'":
			b.active = false
			return b, func() tea.Msg {
				return BookmarkClosedMsg{}
			}

		case "enter", "l":
			if len(bookmarks) > 0 && b.cursor < len(bookmarks) {
				selected := bookmarks[b.cursor]
				b.active = false
				return b, func() tea.Msg {
					return BookmarkSelectedMsg{Bookmark: selected}
				}
			}
			return b, nil

		case "j", "down":
			if b.cursor < len(bookmarks)-1 {
				b.cursor++
			}
			return b, nil

		case "k", "up":
			if b.cursor > 0 {
				b.cursor--
			}
			return b, nil

		case "d", "x":
			// Delete bookmark
			if len(bookmarks) > 0 && b.cursor < len(bookmarks) {
				err := b.store.Remove(b.cursor)
				if b.cursor >= len(b.store.List()) && b.cursor > 0 {
					b.cursor--
				}
				return b, func() tea.Msg {
					return BookmarkRemovedMsg{Success: err == nil, Error: err}
				}
			}
			return b, nil

		case "g":
			b.cursor = 0
			return b, nil

		case "G":
			if len(bookmarks) > 0 {
				b.cursor = len(bookmarks) - 1
			}
			return b, nil
		}
	}

	return b, nil
}

// View renders the bookmark selector
func (b *BookmarkSelector) View() string {
	if !b.active {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(70)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("230"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	typeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	var content strings.Builder

	content.WriteString(titleStyle.Render("Bookmarks"))
	content.WriteString("\n")

	bookmarks := b.store.List()

	if len(bookmarks) == 0 {
		content.WriteString(dimStyle.Render("  (no bookmarks)"))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render("  Press 'm' on a resource to bookmark it"))
	} else {
		// Calculate visible range
		maxVisible := 15
		start := 0
		if b.cursor >= maxVisible {
			start = b.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(bookmarks) {
			end = len(bookmarks)
		}

		for i := start; i < end; i++ {
			bm := bookmarks[i]
			prefix := "  "
			style := normalStyle
			if i == b.cursor {
				prefix = "> "
				style = selectedStyle
			}

			// Format: [type] name (region)
			typeLabel := typeStyle.Render(fmt.Sprintf("[%s]", bm.ResourceType))
			name := style.Render(bm.Name)
			region := dimStyle.Render(fmt.Sprintf("(%s)", bm.Region))

			line := fmt.Sprintf("%s%s %s %s", prefix, typeLabel, name, region)
			content.WriteString(line)
			content.WriteString("\n")
		}

		if len(bookmarks) > maxVisible {
			content.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more", len(bookmarks)-maxVisible)))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(dimStyle.Render("enter:jump  d:delete  esc:close"))

	box := boxStyle.Render(content.String())

	return lipgloss.Place(
		b.width,
		b.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// AddBookmark is a helper to add a bookmark
func AddBookmark(store *config.BookmarkStore, name, resourceType, resourceID, arn, region, profile string) tea.Cmd {
	return func() tea.Msg {
		bookmark := config.Bookmark{
			Name:         name,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			ARN:          arn,
			Region:       region,
			Profile:      profile,
		}

		err := store.Add(bookmark)
		return BookmarkAddedMsg{
			Success: err == nil,
			Name:    name,
			Error:   err,
		}
	}
}
