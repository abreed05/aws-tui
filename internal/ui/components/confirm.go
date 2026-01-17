package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// ConfirmDialog provides a confirmation dialog for sensitive operations
type ConfirmDialog struct {
	message      string
	width        int
	theme        styles.Theme
	requireInput bool
	inputLabel   string
	input        textinput.Model
	inputMin     int
	inputMax     int
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(theme styles.Theme) *ConfirmDialog {
	return &ConfirmDialog{theme: theme}
}

// SetMessage sets the confirmation message
func (c *ConfirmDialog) SetMessage(message string) {
	c.message = message
}

// SetWidth sets the dialog width
func (c *ConfirmDialog) SetWidth(width int) {
	c.width = width
}

// RequireInput enables input field in the dialog
func (c *ConfirmDialog) RequireInput(label string, defaultVal string, min, max int) {
	c.requireInput = true
	c.inputLabel = label
	c.inputMin = min
	c.inputMax = max

	c.input = textinput.New()
	c.input.Placeholder = defaultVal
	c.input.SetValue(defaultVal)
	c.input.CharLimit = 3
	c.input.Width = 10
	c.input.Focus()
}

// GetInput returns the current input value
func (c *ConfirmDialog) GetInput() string {
	return c.input.Value()
}

// HasInput returns whether the dialog has an input field
func (c *ConfirmDialog) HasInput() bool {
	return c.requireInput
}

// Reset clears the input state
func (c *ConfirmDialog) Reset() {
	c.requireInput = false
	c.input.SetValue("")
}

// Update handles messages for the input field
func (c *ConfirmDialog) Update(msg tea.Msg) (*ConfirmDialog, tea.Cmd) {
	if !c.requireInput {
		return c, nil
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

// View renders the confirmation dialog
func (c *ConfirmDialog) View() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.theme.Colors.Warning).
		Padding(1, 2).
		Width(c.width - 20)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(c.theme.Colors.Warning).
		Render("⚠ Warning")

	message := lipgloss.NewStyle().
		Foreground(c.theme.Colors.Foreground).
		Render(c.message)

	var inputSection string
	if c.requireInput {
		inputLabel := lipgloss.NewStyle().
			Foreground(c.theme.Colors.Foreground).
			Render(c.inputLabel + ": ")
		inputSection = "\n\n" + inputLabel + c.input.View()
	}

	help := lipgloss.NewStyle().
		Foreground(c.theme.Colors.Muted).
		Render("\n\nPress 'y' to confirm or 'n' to cancel")

	content := fmt.Sprintf("%s\n\n%s%s%s", title, message, inputSection, help)

	return style.Render(content)
}
