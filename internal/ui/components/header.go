package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// Header displays the top bar with profile, region, and account info
type Header struct {
	profile   string
	region    string
	accountID string
	width     int
	theme     styles.Theme
}

// NewHeader creates a new header component
func NewHeader(theme styles.Theme) *Header {
	return &Header{
		theme:   theme,
		profile: "default",
		region:  "us-east-1",
	}
}

// SetProfile updates the displayed profile
func (h *Header) SetProfile(profile string) {
	h.profile = profile
	if h.profile == "" {
		h.profile = "default"
	}
}

// SetRegion updates the displayed region
func (h *Header) SetRegion(region string) {
	h.region = region
}

// SetAccountID updates the displayed account ID
func (h *Header) SetAccountID(accountID string) {
	h.accountID = accountID
}

// SetWidth sets the header width
func (h *Header) SetWidth(width int) {
	h.width = width
}

// View renders the header
func (h *Header) View() string {
	// Left side: App name and context
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Render("aws-tui")

	profileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	regionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212"))

	accountStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	left := fmt.Sprintf("%s  %s  %s",
		title,
		profileStyle.Render("⚙ "+h.profile),
		regionStyle.Render("⚑ "+h.region),
	)

	if h.accountID != "" {
		left += "  " + accountStyle.Render("☁ "+h.accountID)
	}

	// Right side: Status indicators
	right := ""

	// Calculate padding
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := h.width - leftWidth - rightWidth - 2

	if padding < 0 {
		padding = 0
	}

	spacer := lipgloss.NewStyle().Width(padding).Render("")

	content := left + spacer + right

	return h.theme.Header.Width(h.width).Render(content)
}
