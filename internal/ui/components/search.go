package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// SearchUpdateMsg is sent when search term changes
type SearchUpdateMsg struct {
	Query string
}

// SearchClosedMsg is sent when search is closed
type SearchClosedMsg struct {
	Query string
}

// Search provides search/filter functionality
type Search struct {
	input   textinput.Model
	active  bool
	results int
	total   int
	width   int
	theme   styles.Theme
}

// NewSearch creates a new search component
func NewSearch(theme styles.Theme) *Search {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Prompt = "/ "
	ti.CharLimit = 100
	ti.Width = 40

	return &Search{
		input: ti,
		theme: theme,
	}
}

// SetWidth sets the search box width
func (s *Search) SetWidth(width int) {
	s.width = width
	s.input.Width = width - 20
}

// Activate activates the search
func (s *Search) Activate() tea.Cmd {
	s.active = true
	s.input.Focus()
	return textinput.Blink
}

// Deactivate deactivates the search
func (s *Search) Deactivate() {
	s.active = false
	s.input.Blur()
}

// Clear clears the search input
func (s *Search) Clear() {
	s.input.SetValue("")
	s.results = 0
	s.total = 0
}

// IsActive returns whether search is active
func (s *Search) IsActive() bool {
	return s.active
}

// Value returns the current search value
func (s *Search) Value() string {
	return s.input.Value()
}

// SetResults sets the result count
func (s *Search) SetResults(results, total int) {
	s.results = results
	s.total = total
}

// Update handles messages
func (s *Search) Update(msg tea.Msg) (*Search, tea.Cmd) {
	if !s.active {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			query := s.input.Value()
			s.Deactivate()
			return s, func() tea.Msg {
				return SearchClosedMsg{Query: query}
			}

		case "esc":
			s.Deactivate()
			s.Clear()
			return s, func() tea.Msg {
				return SearchClosedMsg{Query: ""}
			}
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)

	// Send incremental search updates
	return s, tea.Batch(cmd, func() tea.Msg {
		return SearchUpdateMsg{Query: s.input.Value()}
	})
}

// View renders the search component
func (s *Search) View() string {
	if !s.active {
		return ""
	}

	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 1).
		Background(lipgloss.Color("236"))

	resultStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	input := s.input.View()

	var status string
	if s.total > 0 {
		status = resultStyle.Render(fmt.Sprintf(" (%d/%d)", s.results, s.total))
	}

	return searchStyle.Render(input + status)
}
