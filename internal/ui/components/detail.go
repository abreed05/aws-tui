package components

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// Detail displays resource details in a scrollable view
type Detail struct {
	viewport viewport.Model
	content  map[string]interface{}
	yamlView bool
	rawJSON  string

	// Dimensions
	width  int
	height int

	// Styling
	theme styles.Theme

	// Focus
	focused bool
}

// NewDetail creates a new detail component
func NewDetail(theme styles.Theme) *Detail {
	vp := viewport.New(80, 20)

	return &Detail{
		viewport: vp,
		theme:    theme,
		focused:  false,
	}
}

// SetSize sets the detail view dimensions
func (d *Detail) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width - 4
	d.viewport.Height = height - 2
	d.renderContent()
}

// SetContent updates the detail view with new content
func (d *Detail) SetContent(content map[string]interface{}) {
	d.content = content
	d.renderContent()
}

// Clear clears the detail view
func (d *Detail) Clear() {
	d.content = nil
	d.viewport.SetContent("")
}

// ToggleYAML switches between formatted and YAML view
func (d *Detail) ToggleYAML() {
	d.yamlView = !d.yamlView
	d.renderContent()
}

// IsYAMLView returns whether YAML view is active
func (d *Detail) IsYAMLView() bool {
	return d.yamlView
}

// GetJSON returns the content as JSON string
func (d *Detail) GetJSON() string {
	if d.content == nil {
		return ""
	}
	data, err := json.MarshalIndent(d.content, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// Focus sets the focus state
func (d *Detail) Focus() {
	d.focused = true
}

// Blur removes focus
func (d *Detail) Blur() {
	d.focused = false
}

// IsFocused returns whether the detail view is focused
func (d *Detail) IsFocused() bool {
	return d.focused
}

// Update handles messages
func (d *Detail) Update(msg tea.Msg) (*Detail, tea.Cmd) {
	if !d.focused {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
			d.ToggleYAML()
			return d, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			d.viewport.LineDown(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			d.viewport.LineUp(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
			d.viewport.HalfViewDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
			d.viewport.HalfViewUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g", "home"))):
			d.viewport.GotoTop()
		case key.Matches(msg, key.NewBinding(key.WithKeys("G", "end"))):
			d.viewport.GotoBottom()
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *Detail) renderContent() {
	if d.content == nil {
		d.viewport.SetContent("No resource selected")
		return
	}

	var content string
	if d.yamlView {
		content = d.renderYAML()
	} else {
		content = d.renderFormatted()
	}

	d.viewport.SetContent(content)
}

func (d *Detail) renderYAML() string {
	data, err := yaml.Marshal(d.content)
	if err != nil {
		return fmt.Sprintf("Error rendering YAML: %v", err)
	}

	// Syntax highlight YAML
	lines := strings.Split(string(data), "\n")
	var highlighted []string

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	stringStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("78"))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			highlighted = append(highlighted, line)
			continue
		}

		// Simple YAML highlighting
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]

				// Check if value is a string (has quotes or content after colon)
				if strings.TrimSpace(value) != "" {
					if strings.HasPrefix(strings.TrimSpace(value), "\"") ||
						strings.HasPrefix(strings.TrimSpace(value), "'") {
						highlighted = append(highlighted, keyStyle.Render(key)+":"+stringStyle.Render(value))
					} else {
						highlighted = append(highlighted, keyStyle.Render(key)+":"+valueStyle.Render(value))
					}
				} else {
					highlighted = append(highlighted, keyStyle.Render(key)+":")
				}
				continue
			}
		}

		// List items
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			highlighted = append(highlighted, valueStyle.Render(line))
			continue
		}

		highlighted = append(highlighted, line)
	}

	return strings.Join(highlighted, "\n")
}

func (d *Detail) renderFormatted() string {
	var sb strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(25)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Sort sections for consistent ordering
	sections := make([]string, 0, len(d.content))
	for section := range d.content {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	for _, section := range sections {
		value := d.content[section]

		sb.WriteString(sectionStyle.Render(section))
		sb.WriteString("\n")

		switch v := value.(type) {
		case map[string]interface{}:
			d.renderMap(&sb, v, keyStyle, valueStyle, "  ")

		case map[string]string:
			for k, val := range v {
				sb.WriteString("  ")
				sb.WriteString(keyStyle.Render(k + ":"))
				sb.WriteString(valueStyle.Render(val))
				sb.WriteString("\n")
			}

		case []interface{}:
			d.renderSlice(&sb, v, keyStyle, valueStyle, "  ")

		case []map[string]string:
			for _, item := range v {
				for k, val := range item {
					sb.WriteString("  ")
					sb.WriteString(keyStyle.Render(k + ":"))
					sb.WriteString(valueStyle.Render(val))
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}

		case []map[string]interface{}:
			for _, item := range v {
				d.renderMap(&sb, item, keyStyle, valueStyle, "  ")
				sb.WriteString("\n")
			}

		case []string:
			for _, s := range v {
				sb.WriteString("  • ")
				sb.WriteString(valueStyle.Render(s))
				sb.WriteString("\n")
			}

		default:
			sb.WriteString("  ")
			sb.WriteString(valueStyle.Render(fmt.Sprintf("%v", v)))
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (d *Detail) renderMap(sb *strings.Builder, m map[string]interface{}, keyStyle, valueStyle lipgloss.Style, indent string) {
	// Sort keys for consistent ordering
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		sb.WriteString(indent)
		sb.WriteString(keyStyle.Render(k + ":"))

		switch val := v.(type) {
		case map[string]interface{}:
			sb.WriteString("\n")
			d.renderMap(sb, val, keyStyle, valueStyle, indent+"  ")
		case []interface{}:
			sb.WriteString("\n")
			d.renderSlice(sb, val, keyStyle, valueStyle, indent+"  ")
		default:
			sb.WriteString(valueStyle.Render(fmt.Sprintf("%v", val)))
			sb.WriteString("\n")
		}
	}
}

func (d *Detail) renderSlice(sb *strings.Builder, s []interface{}, keyStyle, valueStyle lipgloss.Style, indent string) {
	for _, item := range s {
		switch v := item.(type) {
		case map[string]interface{}:
			sb.WriteString(indent + "•\n")
			d.renderMap(sb, v, keyStyle, valueStyle, indent+"  ")
		default:
			sb.WriteString(indent + "• ")
			sb.WriteString(valueStyle.Render(fmt.Sprintf("%v", v)))
			sb.WriteString("\n")
		}
	}
}

// View renders the detail view
func (d *Detail) View() string {
	if d.width == 0 || d.height == 0 {
		return ""
	}

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("237")).
		Padding(0, 1).
		Width(d.width)

	viewMode := "Formatted"
	if d.yamlView {
		viewMode = "YAML"
	}

	title := fmt.Sprintf("Details (%s) - Press 'y' to toggle", viewMode)

	// Border style based on focus
	borderColor := lipgloss.Color("240")
	if d.focused {
		borderColor = lipgloss.Color("62")
	}

	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(d.width - 2).
		Height(d.height - 3)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(d.viewport.View()),
	)
}
