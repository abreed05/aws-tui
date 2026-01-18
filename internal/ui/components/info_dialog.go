package components

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// InfoDialog displays structured information in a dialog
type InfoDialog struct {
	theme   styles.Theme
	title   string
	content string
	width   int
	height  int
	visible bool
	scroll  int
	lines   []string
}

// NewInfoDialog creates a new info dialog
func NewInfoDialog(theme styles.Theme) *InfoDialog {
	return &InfoDialog{
		theme:   theme,
		visible: false,
	}
}

// Show displays the dialog with the given title and content
func (d *InfoDialog) Show(title string, data interface{}) {
	d.title = title
	d.visible = true
	d.scroll = 0

	// Convert data to formatted JSON string
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		d.content = fmt.Sprintf("Error formatting data: %v", err)
		d.lines = []string{d.content}
		return
	}

	d.content = string(jsonBytes)
	d.lines = strings.Split(d.content, "\n")
}

// Hide closes the dialog
func (d *InfoDialog) Hide() {
	d.visible = false
	d.title = ""
	d.content = ""
	d.lines = nil
	d.scroll = 0
}

// IsVisible returns whether the dialog is visible
func (d *InfoDialog) IsVisible() bool {
	return d.visible
}

// SetSize sets the dialog dimensions
func (d *InfoDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Update handles messages
func (d *InfoDialog) Update(msg tea.Msg) (*InfoDialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			d.Hide()
			return d, nil

		case "j", "down":
			if d.scroll < len(d.lines)-1 {
				d.scroll++
			}
			return d, nil

		case "k", "up":
			if d.scroll > 0 {
				d.scroll--
			}
			return d, nil

		case "d", "ctrl+d":
			// Page down
			pageSize := d.height - 8
			d.scroll += pageSize
			if d.scroll >= len(d.lines) {
				d.scroll = len(d.lines) - 1
			}
			if d.scroll < 0 {
				d.scroll = 0
			}
			return d, nil

		case "u", "ctrl+u":
			// Page up
			pageSize := d.height - 8
			d.scroll -= pageSize
			if d.scroll < 0 {
				d.scroll = 0
			}
			return d, nil

		case "g":
			// Go to top
			d.scroll = 0
			return d, nil

		case "G":
			// Go to bottom
			d.scroll = len(d.lines) - 1
			if d.scroll < 0 {
				d.scroll = 0
			}
			return d, nil
		}
	}

	return d, nil
}

// View renders the dialog
func (d *InfoDialog) View() string {
	if !d.visible {
		return ""
	}

	dialogWidth := d.width - 10
	if dialogWidth < 40 {
		dialogWidth = 40
	}

	dialogHeight := d.height - 6
	if dialogHeight < 10 {
		dialogHeight = 10
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(d.theme.Colors.Primary).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	title := titleStyle.Render(d.title)

	// Content area
	contentHeight := dialogHeight - 4 // Leave room for title and help text
	visibleLines := make([]string, 0, contentHeight)

	startLine := d.scroll
	endLine := startLine + contentHeight
	if endLine > len(d.lines) {
		endLine = len(d.lines)
	}

	for i := startLine; i < endLine; i++ {
		line := d.lines[i]
		// Truncate long lines
		if len(line) > dialogWidth-6 {
			line = line[:dialogWidth-6] + "..."
		}
		visibleLines = append(visibleLines, line)
	}

	// Pad with empty lines if needed
	for len(visibleLines) < contentHeight {
		visibleLines = append(visibleLines, "")
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(d.theme.Colors.Foreground).
		Width(dialogWidth - 4)

	content := contentStyle.Render(strings.Join(visibleLines, "\n"))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(d.theme.Colors.Muted).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	scrollInfo := ""
	if len(d.lines) > contentHeight {
		scrollInfo = fmt.Sprintf(" (Line %d/%d)", d.scroll+1, len(d.lines))
	}

	help := helpStyle.Render(fmt.Sprintf("j/k: scroll | g/G: top/bottom | esc/q: close%s", scrollInfo))

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Colors.Primary).
		Padding(1, 2).
		Width(dialogWidth)

	dialogContent := fmt.Sprintf("%s\n\n%s\n\n%s", title, content, help)

	dialog := borderStyle.Render(dialogContent)

	// Center the dialog
	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
