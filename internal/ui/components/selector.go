package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/adapters/config"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// SelectorMode determines what the selector is selecting
type SelectorMode int

const (
	SelectProfile SelectorMode = iota
	SelectRegion
)

// ProfileSelectedMsg is sent when a profile is selected
type ProfileSelectedMsg struct {
	Profile string
}

// RegionSelectedMsg is sent when a region is selected
type RegionSelectedMsg struct {
	Region string
}

// SelectorClosedMsg is sent when the selector is closed without selection
type SelectorClosedMsg struct{}

// selectorItem implements list.Item
type selectorItem struct {
	title       string
	description string
	value       string
}

func (i selectorItem) Title() string       { return i.title }
func (i selectorItem) Description() string { return i.description }
func (i selectorItem) FilterValue() string { return i.title }

// Selector provides profile and region selection
type Selector struct {
	list     list.Model
	mode     SelectorMode
	active   bool
	width    int
	height   int
	theme    styles.Theme
	selected string
}

// NewSelector creates a new selector component
func NewSelector(theme styles.Theme) *Selector {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("57"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(true)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginLeft(2)

	return &Selector{
		list:  l,
		theme: theme,
	}
}

// SetSize sets the selector dimensions
func (s *Selector) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width-4, height-4)
}

// ShowProfiles shows the profile selector
func (s *Selector) ShowProfiles(profiles []config.Profile, current string) tea.Cmd {
	s.mode = SelectProfile
	s.active = true
	s.selected = current
	s.list.Title = "Select AWS Profile"

	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		desc := p.Region
		if p.IsSSO {
			desc = "SSO: " + p.SSOStartURL
		} else if p.RoleARN != "" {
			desc = "Role: " + p.RoleARN
		}
		if desc == "" {
			desc = "Static credentials"
		}

		items[i] = selectorItem{
			title:       p.Name,
			description: desc,
			value:       p.Name,
		}
	}

	s.list.SetItems(items)

	// Select current item
	for i, item := range items {
		if item.(selectorItem).value == current {
			s.list.Select(i)
			break
		}
	}

	return nil
}

// ShowRegions shows the region selector
func (s *Selector) ShowRegions(regions []config.Region, current string) tea.Cmd {
	s.mode = SelectRegion
	s.active = true
	s.selected = current
	s.list.Title = "Select AWS Region"

	items := make([]list.Item, len(regions))
	for i, r := range regions {
		items[i] = selectorItem{
			title:       r.Name,
			description: r.Description,
			value:       r.Name,
		}
	}

	s.list.SetItems(items)

	// Select current item
	for i, item := range items {
		if item.(selectorItem).value == current {
			s.list.Select(i)
			break
		}
	}

	return nil
}

// IsActive returns whether the selector is active
func (s *Selector) IsActive() bool {
	return s.active
}

// Close closes the selector
func (s *Selector) Close() {
	s.active = false
}

// Update handles messages
func (s *Selector) Update(msg tea.Msg) (*Selector, tea.Cmd) {
	if !s.active {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := s.list.SelectedItem()
			if selected == nil {
				return s, nil
			}
			item := selected.(selectorItem)
			s.active = false

			if s.mode == SelectProfile {
				return s, func() tea.Msg {
					return ProfileSelectedMsg{Profile: item.value}
				}
			}
			return s, func() tea.Msg {
				return RegionSelectedMsg{Region: item.value}
			}

		case "esc", "q":
			s.active = false
			return s, func() tea.Msg {
				return SelectorClosedMsg{}
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

// View renders the selector
func (s *Selector) View() string {
	if !s.active {
		return ""
	}

	// Create a modal box
	content := s.list.View()

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(s.width - 10).
		Height(s.height - 6)

	// Center the modal
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		modal.Render(content),
	)
}

// Helper to build a filter that matches search terms
func matchesFilter(item, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item), strings.ToLower(filter))
}
