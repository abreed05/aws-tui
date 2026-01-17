package components

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// SecretEditor provides a textarea-based editor for secrets
type SecretEditor struct {
	textarea     textarea.Model
	secretID     string
	secretName   string
	secretValue  string
	initialValue string
	isJSON       bool
	modified     bool
	width        int
	height       int
	theme        styles.Theme
}

// NewSecretEditor creates a new secret editor
func NewSecretEditor(theme styles.Theme) *SecretEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter secret value..."
	ta.Focus()
	ta.CharLimit = 0 // No character limit
	ta.ShowLineNumbers = false
	// Disable paste to avoid clipboard tool requirement
	ta.KeyMap.Paste.SetEnabled(false)

	return &SecretEditor{
		textarea: ta,
		theme:    theme,
	}
}

// SetSecret sets the secret to edit
func (e *SecretEditor) SetSecret(id, name, value string) {
	e.secretID = id
	e.secretName = name
	e.secretValue = value
	e.initialValue = value
	e.modified = false

	// Try to format as JSON if valid
	var jsonData interface{}
	if json.Unmarshal([]byte(value), &jsonData) == nil {
		e.isJSON = true
		formatted, _ := json.MarshalIndent(jsonData, "", "  ")
		e.textarea.SetValue(string(formatted))
	} else {
		e.isJSON = false
		e.textarea.SetValue(value)
	}
}

// Value returns the current value, validating JSON if needed
func (e *SecretEditor) Value() (string, error) {
	value := e.textarea.Value()

	// If it was JSON, validate it's still valid JSON
	if e.isJSON {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		// Minify JSON before saving
		minified, _ := json.Marshal(jsonData)
		return string(minified), nil
	}

	return value, nil
}

// GetSecretID returns the secret ID being edited
func (e *SecretEditor) GetSecretID() string {
	return e.secretID
}

// IsModified returns whether the secret has been modified
func (e *SecretEditor) IsModified() bool {
	return e.modified
}

// SetSize sets the editor dimensions
func (e *SecretEditor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width - 4)
	e.textarea.SetHeight(height - 10) // Leave room for title and help text
}

// Update handles messages for the editor
func (e *SecretEditor) Update(msg tea.Msg) (*SecretEditor, tea.Cmd) {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)

	// Track if modified
	currentValue := e.textarea.Value()
	// Format for comparison if JSON
	compareValue := currentValue
	if e.isJSON {
		var jsonData interface{}
		if json.Unmarshal([]byte(currentValue), &jsonData) == nil {
			minified, _ := json.Marshal(jsonData)
			compareValue = string(minified)
		}
	}

	initialCompare := e.initialValue
	if e.isJSON {
		var jsonData interface{}
		if json.Unmarshal([]byte(e.initialValue), &jsonData) == nil {
			minified, _ := json.Marshal(jsonData)
			initialCompare = string(minified)
		}
	}

	e.modified = compareValue != initialCompare

	return e, cmd
}

// View renders the editor
func (e *SecretEditor) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(e.theme.Colors.Primary).
		Render(fmt.Sprintf("Editing Secret: %s", e.secretName))

	formatIndicator := "Plain Text"
	if e.isJSON {
		formatIndicator = "JSON"
	}

	modifiedIndicator := ""
	if e.modified {
		modifiedIndicator = " [Modified]"
	}

	subtitle := lipgloss.NewStyle().
		Foreground(e.theme.Colors.Muted).
		Render(fmt.Sprintf("Format: %s%s", formatIndicator, modifiedIndicator))

	helpText := lipgloss.NewStyle().
		Foreground(e.theme.Colors.Muted).
		Render("Ctrl+S: Save | Esc: Cancel")

	editor := e.textarea.View()

	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", title, subtitle, editor, helpText)
}
