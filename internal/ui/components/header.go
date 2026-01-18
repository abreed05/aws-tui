package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// Header displays the top bar with profile, region, and account info
type Header struct {
	profile     string
	region      string
	accountID   string
	context     string // Current resource context (e.g., "EC2", "DynamoDB", "Home")
	width       int
	theme       styles.Theme
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

// SetContext updates the current resource context
func (h *Header) SetContext(context string) {
	h.context = context
}

// View renders the header
func (h *Header) View() string {
	// Define styles
	boxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("51"))

	contextStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("201"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	valueStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))

	// Build the ASCII art logo (each is 6 chars wide)
	logo := []string{
		"╔═══╗ ",
		"║AWS║ ",
		"╚═══╝ ",
	}

	// Build context section
	contextDisplay := h.context
	if contextDisplay == "" {
		contextDisplay = "Home"
	}

	// Build info lines with proper spacing
	line1 := fmt.Sprintf(" %s %s",
		labelStyle.Render("Profile:"),
		valueStyle.Render(h.profile),
	)

	line2 := fmt.Sprintf(" %s %s",
		labelStyle.Render("Region: "),
		valueStyle.Render(h.region),
	)

	line3 := ""
	if h.accountID != "" {
		line3 = fmt.Sprintf(" %s %s",
			labelStyle.Render("Account:"),
			valueStyle.Render(h.accountID),
		)
	} else {
		line3 = " "
	}

	// Calculate widths for layout (accounting for borders: 4 x "│")
	logoWidth := 6     // Logo is 6 chars
	infoWidth := 36    // Info section width
	contextWidth := h.width - logoWidth - infoWidth - 4  // 4 borders

	// Create the top border
	topBorder := "┌" + strings.Repeat("─", logoWidth) + "┬" +
		strings.Repeat("─", infoWidth) + "┬" +
		strings.Repeat("─", contextWidth) + "┐"

	// Create bottom border
	bottomBorder := "└" + strings.Repeat("─", h.width-2) + "┘"

	// Build the context display centered
	contextText := "[ " + contextDisplay + " ]"
	contextPadding := contextWidth - len(contextText)
	if contextPadding < 0 {
		contextPadding = 0
	}
	leftPad := contextPadding / 2
	rightPad := contextPadding - leftPad
	centeredContext := strings.Repeat(" ", leftPad) + contextText + strings.Repeat(" ", rightPad)

	// Build rows ensuring exact widths
	row1 := "│" + titleStyle.Render(logo[0]) + "│" +
		lipgloss.NewStyle().Width(infoWidth).Render(line1) + "│" +
		contextStyle.Render(centeredContext) + "│"

	row2 := "│" + titleStyle.Render(logo[1]) + "│" +
		lipgloss.NewStyle().Width(infoWidth).Render(line2) + "│" +
		strings.Repeat(" ", contextWidth) + "│"

	row3 := "│" + titleStyle.Render(logo[2]) + "│" +
		lipgloss.NewStyle().Width(infoWidth).Render(line3) + "│" +
		strings.Repeat(" ", contextWidth) + "│"

	// Build title bar
	titleBar := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("235")).
		Width(h.width).
		Align(lipgloss.Center).
		Render("AWS Terminal UI")

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		boxStyle.Render(topBorder),
		boxStyle.Render(row1),
		boxStyle.Render(row2),
		boxStyle.Render(row3),
		boxStyle.Render(bottomBorder),
	)
}
