package components

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

const (
	fieldName = iota
	fieldValue
	fieldDescription
	fieldTagKey
	fieldTagValue
)

type tagPair struct {
	key   string
	value string
}

type SecretCreator struct {
	theme  styles.Theme
	width  int
	height int

	// Form inputs
	nameInput        textinput.Model
	valueInput       textarea.Model
	descriptionInput textinput.Model
	tagKeyInput      textinput.Model
	tagValueInput    textinput.Model

	// State
	focusedField int
	tags         []tagPair
	errors       map[string]string
}

func NewSecretCreator(theme styles.Theme) *SecretCreator {
	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "my-secret-name"
	nameInput.CharLimit = 512
	nameInput.Width = 50

	// Value textarea
	valueInput := textarea.New()
	valueInput.Placeholder = "Enter secret value..."
	valueInput.CharLimit = 65536
	valueInput.ShowLineNumbers = false
	// Disable paste to avoid clipboard tool requirement
	valueInput.KeyMap.Paste.SetEnabled(false)

	// Description input
	descInput := textinput.New()
	descInput.Placeholder = "Description of this secret (optional)"
	descInput.CharLimit = 2048
	descInput.Width = 50

	// Tag key input
	tagKeyInput := textinput.New()
	tagKeyInput.Placeholder = "Environment"
	tagKeyInput.CharLimit = 128
	tagKeyInput.Width = 20

	// Tag value input
	tagValueInput := textinput.New()
	tagValueInput.Placeholder = "production"
	tagValueInput.CharLimit = 256
	tagValueInput.Width = 20

	return &SecretCreator{
		theme:            theme,
		nameInput:        nameInput,
		valueInput:       valueInput,
		descriptionInput: descInput,
		tagKeyInput:      tagKeyInput,
		tagValueInput:    tagValueInput,
		focusedField:     fieldName,
		tags:             make([]tagPair, 0),
		errors:           make(map[string]string),
	}
}

func (s *SecretCreator) Activate() tea.Cmd {
	s.focusedField = fieldName
	s.nameInput.Focus()
	return textinput.Blink
}

func (s *SecretCreator) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.valueInput.SetWidth(width - 10)
	s.valueInput.SetHeight(8)
}

func (s *SecretCreator) Reset() {
	s.nameInput.SetValue("")
	s.valueInput.SetValue("")
	s.descriptionInput.SetValue("")
	s.tagKeyInput.SetValue("")
	s.tagValueInput.SetValue("")
	s.tags = make([]tagPair, 0)
	s.errors = make(map[string]string)
	s.focusedField = fieldName
}

func (s *SecretCreator) Validate() error {
	s.errors = make(map[string]string)

	// Validate name
	name := strings.TrimSpace(s.nameInput.Value())
	if name == "" {
		s.errors["name"] = "Name is required"
	} else if len(name) > 512 {
		s.errors["name"] = "Name must be 512 characters or less"
	} else if !regexp.MustCompile(`^[a-zA-Z0-9/_+=.@-]+$`).MatchString(name) {
		s.errors["name"] = "Name must contain only alphanumeric, /_+=.@- characters"
	}

	// Validate value
	value := s.valueInput.Value()
	if value == "" {
		s.errors["value"] = "Value is required"
	} else if len(value) > 65536 {
		s.errors["value"] = "Value must be 65536 bytes or less"
	}

	// Validate tags
	if len(s.tags) > 50 {
		s.errors["tags"] = "Maximum 50 tags allowed"
	}

	if len(s.errors) > 0 {
		return fmt.Errorf("validation failed")
	}

	return nil
}

func (s *SecretCreator) GetParams() map[string]interface{} {
	params := make(map[string]interface{})
	params["Name"] = strings.TrimSpace(s.nameInput.Value())
	params["Value"] = s.valueInput.Value()

	if desc := strings.TrimSpace(s.descriptionInput.Value()); desc != "" {
		params["Description"] = desc
	}

	if len(s.tags) > 0 {
		tags := make(map[string]string)
		for _, tag := range s.tags {
			tags[tag.key] = tag.value
		}
		params["Tags"] = tags
	}

	return params
}

func (s *SecretCreator) Update(msg tea.Msg) (*SecretCreator, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			s.nextField()
			return s, nil
		case "shift+tab":
			s.prevField()
			return s, nil
		case "ctrl+a":
			// Add tag
			key := strings.TrimSpace(s.tagKeyInput.Value())
			value := strings.TrimSpace(s.tagValueInput.Value())
			if key != "" && value != "" {
				s.tags = append(s.tags, tagPair{key: key, value: value})
				s.tagKeyInput.SetValue("")
				s.tagValueInput.SetValue("")
			}
			return s, nil
		case "ctrl+d":
			// Remove last tag
			if len(s.tags) > 0 {
				s.tags = s.tags[:len(s.tags)-1]
			}
			return s, nil
		}
	}

	// Update focused field
	var cmd tea.Cmd
	switch s.focusedField {
	case fieldName:
		s.nameInput, cmd = s.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldValue:
		s.valueInput, cmd = s.valueInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldDescription:
		s.descriptionInput, cmd = s.descriptionInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldTagKey:
		s.tagKeyInput, cmd = s.tagKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldTagValue:
		s.tagValueInput, cmd = s.tagValueInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

func (s *SecretCreator) nextField() {
	s.blurAll()
	s.focusedField = (s.focusedField + 1) % 5
	s.focusCurrent()
}

func (s *SecretCreator) prevField() {
	s.blurAll()
	s.focusedField--
	if s.focusedField < 0 {
		s.focusedField = 4
	}
	s.focusCurrent()
}

func (s *SecretCreator) blurAll() {
	s.nameInput.Blur()
	s.valueInput.Blur()
	s.descriptionInput.Blur()
	s.tagKeyInput.Blur()
	s.tagValueInput.Blur()
}

func (s *SecretCreator) focusCurrent() {
	switch s.focusedField {
	case fieldName:
		s.nameInput.Focus()
	case fieldValue:
		s.valueInput.Focus()
	case fieldDescription:
		s.descriptionInput.Focus()
	case fieldTagKey:
		s.tagKeyInput.Focus()
	case fieldTagValue:
		s.tagValueInput.Focus()
	}
}

func (s *SecretCreator) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(s.theme.Colors.Primary).
		Render("Create New Secret")

	// Name field
	nameLabel := lipgloss.NewStyle().
		Foreground(s.theme.Colors.Foreground).
		Render("Name (required):")
	nameView := s.nameInput.View()
	if err, ok := s.errors["name"]; ok {
		nameView += " " + lipgloss.NewStyle().
			Foreground(s.theme.Colors.Error).
			Render(err)
	}

	// Value field
	valueLabel := lipgloss.NewStyle().
		Foreground(s.theme.Colors.Foreground).
		Render("Value (required):")
	valueView := s.valueInput.View()
	if err, ok := s.errors["value"]; ok {
		valueView += "\n" + lipgloss.NewStyle().
			Foreground(s.theme.Colors.Error).
			Render(err)
	}

	// Description field
	descLabel := lipgloss.NewStyle().
		Foreground(s.theme.Colors.Foreground).
		Render("Description (optional):")
	descView := s.descriptionInput.View()

	// Tags section
	tagsLabel := lipgloss.NewStyle().
		Foreground(s.theme.Colors.Foreground).
		Render("Tags (optional):")

	tagInputs := fmt.Sprintf("Key: %s  Value: %s  (Ctrl+A to add)",
		s.tagKeyInput.View(),
		s.tagValueInput.View())

	var tagsList string
	if len(s.tags) > 0 {
		tags := make([]string, 0, len(s.tags))
		for _, tag := range s.tags {
			tags = append(tags, fmt.Sprintf("  • %s: %s", tag.key, tag.value))
		}
		tagsList = "\n" + strings.Join(tags, "\n")
		tagsList += "\n  (Ctrl+D to remove last)"
	}

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(s.theme.Colors.Muted).
		Render("Tab: next field | Ctrl+S: Create | ESC: Cancel")

	return fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s%s\n\n%s",
		title,
		nameLabel, nameView,
		valueLabel, valueView,
		descLabel, descView,
		tagsLabel, tagInputs, tagsList,
		helpText)
}
