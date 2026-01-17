package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Autocomplete provides command suggestions based on user input
type Autocomplete struct {
	commands    []string
	suggestions []string
	input       string
	selected    int
}

// NewAutocomplete creates a new autocomplete component
func NewAutocomplete() *Autocomplete {
	// List of all available commands
	commands := []string{
		"exit",
		"export",
		"quit",
		"q",
		"home",
		"profile",
		"region",
		"users",
		"roles",
		"policies",
		"sg",
		"kms",
		"secrets",
		"ec2",
		"instances",
		"vpc",
		"vpcs",
		"rds",
		"ecs",
		"lambda",
		"logs",
		"s3",
		"sso",
		"sso-login",
	}

	return &Autocomplete{
		commands:    commands,
		suggestions: []string{},
		selected:    0,
	}
}

// Update updates the autocomplete suggestions based on current input
func (a *Autocomplete) Update(input string) {
	a.input = input
	a.suggestions = []string{}
	a.selected = 0

	// Don't show suggestions if input is empty
	if input == "" {
		return
	}

	// Find matching commands
	for _, cmd := range a.commands {
		if strings.HasPrefix(cmd, input) {
			a.suggestions = append(a.suggestions, cmd)
		}
	}
}

// Next selects the next suggestion
func (a *Autocomplete) Next() {
	if len(a.suggestions) > 0 {
		a.selected = (a.selected + 1) % len(a.suggestions)
	}
}

// Previous selects the previous suggestion
func (a *Autocomplete) Previous() {
	if len(a.suggestions) > 0 {
		a.selected--
		if a.selected < 0 {
			a.selected = len(a.suggestions) - 1
		}
	}
}

// Selected returns the currently selected suggestion
func (a *Autocomplete) Selected() string {
	if len(a.suggestions) > 0 && a.selected < len(a.suggestions) {
		return a.suggestions[a.selected]
	}
	return ""
}

// HasSuggestions returns true if there are suggestions available
func (a *Autocomplete) HasSuggestions() bool {
	return len(a.suggestions) > 0
}

// View renders the autocomplete suggestions
func (a *Autocomplete) View(width int) string {
	if len(a.suggestions) == 0 {
		return ""
	}

	// Limit to 5 suggestions
	maxSuggestions := 5
	displaySuggestions := a.suggestions
	if len(displaySuggestions) > maxSuggestions {
		displaySuggestions = displaySuggestions[:maxSuggestions]
	}

	// Build suggestion list
	var suggestionLines []string
	for i, suggestion := range displaySuggestions {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			PaddingLeft(2)

		if i == a.selected {
			style = style.
				Bold(true).
				Foreground(lipgloss.Color("212")).
				Background(lipgloss.Color("238"))
		}

		suggestionLines = append(suggestionLines, style.Render(suggestion))
	}

	// Add header
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		PaddingLeft(2).
		Render("Suggestions:")

	content := header + "\n" + strings.Join(suggestionLines, "\n")

	// Create a styled box
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Width(width - 4)

	return boxStyle.Render(content)
}
