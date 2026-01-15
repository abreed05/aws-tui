package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/handlers"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// TagFilterUpdateMsg is sent when tag filter changes
type TagFilterUpdateMsg struct {
	Tags map[string]string // Selected tag filters
}

// TagFilterClosedMsg is sent when tag filter is closed
type TagFilterClosedMsg struct {
	Tags map[string]string
}

// TagFilter provides tag-based filtering
type TagFilter struct {
	theme  styles.Theme
	active bool

	// Available tags from resources
	availableTags map[string][]string // key -> []values

	// Current filter state
	selectedTags map[string]string // key -> value

	// UI state
	mode       tagFilterMode
	input      textinput.Model
	tagKeys    []string
	tagValues  []string
	cursor     int
	currentKey string

	// Dimensions
	width  int
	height int
}

type tagFilterMode int

const (
	modeSelectKey tagFilterMode = iota
	modeSelectValue
	modeInput
)

// NewTagFilter creates a new tag filter
func NewTagFilter(theme styles.Theme) *TagFilter {
	input := textinput.New()
	input.Placeholder = "Enter tag value..."
	input.CharLimit = 256

	return &TagFilter{
		theme:         theme,
		availableTags: make(map[string][]string),
		selectedTags:  make(map[string]string),
		input:         input,
		mode:          modeSelectKey,
	}
}

// SetResources extracts available tags from resources
func (t *TagFilter) SetResources(resources []handlers.Resource) {
	t.availableTags = make(map[string][]string)
	valueSet := make(map[string]map[string]bool) // key -> set of values

	for _, res := range resources {
		tags := res.GetTags()
		for k, v := range tags {
			if valueSet[k] == nil {
				valueSet[k] = make(map[string]bool)
			}
			valueSet[k][v] = true
		}
	}

	// Convert to sorted slices
	t.tagKeys = make([]string, 0, len(valueSet))
	for k, values := range valueSet {
		t.tagKeys = append(t.tagKeys, k)
		vals := make([]string, 0, len(values))
		for v := range values {
			vals = append(vals, v)
		}
		sort.Strings(vals)
		t.availableTags[k] = vals
	}
	sort.Strings(t.tagKeys)
}

// Activate shows the tag filter
func (t *TagFilter) Activate() tea.Cmd {
	t.active = true
	t.mode = modeSelectKey
	t.cursor = 0
	t.currentKey = ""
	return nil
}

// IsActive returns whether the filter is active
func (t *TagFilter) IsActive() bool {
	return t.active
}

// GetSelectedTags returns the currently selected tag filters
func (t *TagFilter) GetSelectedTags() map[string]string {
	return t.selectedTags
}

// ClearFilters clears all tag filters
func (t *TagFilter) ClearFilters() {
	t.selectedTags = make(map[string]string)
}

// SetSize sets the dimensions
func (t *TagFilter) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Update handles messages
func (t *TagFilter) Update(msg tea.Msg) (*TagFilter, tea.Cmd) {
	if !t.active {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch t.mode {
		case modeSelectKey:
			return t.handleKeySelection(msg)
		case modeSelectValue:
			return t.handleValueSelection(msg)
		case modeInput:
			return t.handleInput(msg)
		}
	}

	return t, nil
}

func (t *TagFilter) handleKeySelection(msg tea.KeyMsg) (*TagFilter, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.active = false
		return t, func() tea.Msg {
			return TagFilterClosedMsg{Tags: t.selectedTags}
		}

	case "enter", "l":
		if len(t.tagKeys) > 0 && t.cursor < len(t.tagKeys) {
			t.currentKey = t.tagKeys[t.cursor]
			t.tagValues = t.availableTags[t.currentKey]
			t.mode = modeSelectValue
			t.cursor = 0
		}
		return t, nil

	case "j", "down":
		if t.cursor < len(t.tagKeys)-1 {
			t.cursor++
		}
		return t, nil

	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
		return t, nil

	case "c":
		// Clear all filters
		t.selectedTags = make(map[string]string)
		return t, func() tea.Msg {
			return TagFilterUpdateMsg{Tags: t.selectedTags}
		}

	case "x":
		// Remove filter for current key
		if len(t.tagKeys) > 0 && t.cursor < len(t.tagKeys) {
			key := t.tagKeys[t.cursor]
			delete(t.selectedTags, key)
			return t, func() tea.Msg {
				return TagFilterUpdateMsg{Tags: t.selectedTags}
			}
		}
	}

	return t, nil
}

func (t *TagFilter) handleValueSelection(msg tea.KeyMsg) (*TagFilter, tea.Cmd) {
	switch msg.String() {
	case "esc", "h":
		t.mode = modeSelectKey
		t.cursor = 0
		return t, nil

	case "enter", "l":
		if len(t.tagValues) > 0 && t.cursor < len(t.tagValues) {
			value := t.tagValues[t.cursor]
			t.selectedTags[t.currentKey] = value
			t.mode = modeSelectKey
			t.cursor = 0
			return t, func() tea.Msg {
				return TagFilterUpdateMsg{Tags: t.selectedTags}
			}
		}
		return t, nil

	case "j", "down":
		if t.cursor < len(t.tagValues)-1 {
			t.cursor++
		}
		return t, nil

	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
		return t, nil

	case "/":
		// Switch to input mode for custom value
		t.mode = modeInput
		t.input.SetValue("")
		t.input.Focus()
		return t, textinput.Blink
	}

	return t, nil
}

func (t *TagFilter) handleInput(msg tea.KeyMsg) (*TagFilter, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.mode = modeSelectValue
		t.input.Blur()
		return t, nil

	case "enter":
		value := t.input.Value()
		if value != "" {
			t.selectedTags[t.currentKey] = value
		}
		t.mode = modeSelectKey
		t.cursor = 0
		t.input.Blur()
		return t, func() tea.Msg {
			return TagFilterUpdateMsg{Tags: t.selectedTags}
		}
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return t, cmd
}

// View renders the tag filter
func (t *TagFilter) View() string {
	if !t.active {
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
		Width(60)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("230"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	activeFilterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114"))

	var content strings.Builder

	switch t.mode {
	case modeSelectKey:
		content.WriteString(titleStyle.Render("Filter by Tag"))
		content.WriteString("\n")

		// Show active filters
		if len(t.selectedTags) > 0 {
			content.WriteString(dimStyle.Render("Active filters:"))
			content.WriteString("\n")
			for k, v := range t.selectedTags {
				content.WriteString(activeFilterStyle.Render(fmt.Sprintf("  %s = %s", k, v)))
				content.WriteString("\n")
			}
			content.WriteString("\n")
		}

		content.WriteString(dimStyle.Render("Select tag key:"))
		content.WriteString("\n")

		if len(t.tagKeys) == 0 {
			content.WriteString(dimStyle.Render("  (no tags available)"))
		} else {
			for i, key := range t.tagKeys {
				prefix := "  "
				style := normalStyle
				if i == t.cursor {
					prefix = "> "
					style = selectedStyle
				}

				// Show if this key has an active filter
				suffix := ""
				if val, ok := t.selectedTags[key]; ok {
					suffix = dimStyle.Render(fmt.Sprintf(" [=%s]", val))
				}

				content.WriteString(prefix + style.Render(key) + suffix + "\n")
			}
		}

		content.WriteString("\n")
		content.WriteString(dimStyle.Render("enter:select  c:clear all  x:remove  esc:close"))

	case modeSelectValue:
		content.WriteString(titleStyle.Render(fmt.Sprintf("Filter: %s", t.currentKey)))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render("Select value:"))
		content.WriteString("\n")

		for i, val := range t.tagValues {
			prefix := "  "
			style := normalStyle
			if i == t.cursor {
				prefix = "> "
				style = selectedStyle
			}
			content.WriteString(prefix + style.Render(val) + "\n")
		}

		content.WriteString("\n")
		content.WriteString(dimStyle.Render("enter:select  /:custom value  esc:back"))

	case modeInput:
		content.WriteString(titleStyle.Render(fmt.Sprintf("Filter: %s", t.currentKey)))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render("Enter custom value:"))
		content.WriteString("\n")
		content.WriteString(t.input.View())
		content.WriteString("\n\n")
		content.WriteString(dimStyle.Render("enter:apply  esc:cancel"))
	}

	box := boxStyle.Render(content.String())

	return lipgloss.Place(
		t.width,
		t.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// FilterResources filters resources based on selected tags
func FilterByTags(resources []handlers.Resource, tags map[string]string) []handlers.Resource {
	if len(tags) == 0 {
		return resources
	}

	filtered := make([]handlers.Resource, 0)
	for _, res := range resources {
		resTags := res.GetTags()
		matches := true

		for filterKey, filterVal := range tags {
			resVal, ok := resTags[filterKey]
			if !ok || !strings.Contains(strings.ToLower(resVal), strings.ToLower(filterVal)) {
				matches = false
				break
			}
		}

		if matches {
			filtered = append(filtered, res)
		}
	}

	return filtered
}
