package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors defines the color palette
type Colors struct {
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Accent      lipgloss.Color
	Background  lipgloss.Color
	Foreground  lipgloss.Color
	Muted       lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	Info        lipgloss.Color
	Border      lipgloss.Color
	Selection   lipgloss.Color
	SelectionFg lipgloss.Color
}

// Theme defines the visual theme
type Theme struct {
	Colors Colors

	// Component styles
	Header       lipgloss.Style
	Footer       lipgloss.Style
	Breadcrumb   lipgloss.Style
	Table        TableStyles
	Detail       DetailStyles
	Search       lipgloss.Style
	Command      lipgloss.Style
	Modal        lipgloss.Style
	Help         lipgloss.Style
	StatusBar    lipgloss.Style
	ErrorMessage lipgloss.Style
}

// TableStyles defines table-specific styles
type TableStyles struct {
	Header   lipgloss.Style
	Row      lipgloss.Style
	Selected lipgloss.Style
	Cell     lipgloss.Style
}

// DetailStyles defines detail view styles
type DetailStyles struct {
	Section lipgloss.Style
	Key     lipgloss.Style
	Value   lipgloss.Style
	Border  lipgloss.Style
}

// DefaultColors returns the default color palette
func DefaultColors() Colors {
	return Colors{
		Primary:     lipgloss.Color("39"),  // Blue
		Secondary:   lipgloss.Color("62"),  // Purple
		Accent:      lipgloss.Color("212"), // Pink
		Background:  lipgloss.Color("235"), // Dark gray
		Foreground:  lipgloss.Color("252"), // Light gray
		Muted:       lipgloss.Color("245"), // Gray
		Success:     lipgloss.Color("78"),  // Green
		Warning:     lipgloss.Color("214"), // Orange
		Error:       lipgloss.Color("196"), // Red
		Info:        lipgloss.Color("81"),  // Cyan
		Border:      lipgloss.Color("240"), // Medium gray
		Selection:   lipgloss.Color("57"),  // Dark purple
		SelectionFg: lipgloss.Color("229"), // Light yellow
	}
}

// DefaultTheme returns the default theme
func DefaultTheme() Theme {
	c := DefaultColors()

	return Theme{
		Colors: c,

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Foreground).
			Background(c.Primary).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Foreground(c.Muted).
			Background(lipgloss.Color("236")).
			Padding(0, 1),

		Breadcrumb: lipgloss.NewStyle().
			Foreground(c.Muted).
			Padding(0, 1),

		Table: TableStyles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.Primary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(c.Border).
				BorderBottom(true),
			Row: lipgloss.NewStyle().
				Foreground(c.Foreground),
			Selected: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.SelectionFg).
				Background(c.Selection),
			Cell: lipgloss.NewStyle().
				Padding(0, 1),
		},

		Detail: DetailStyles{
			Section: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.Accent).
				MarginTop(1),
			Key: lipgloss.NewStyle().
				Foreground(c.Info),
			Value: lipgloss.NewStyle().
				Foreground(c.Foreground),
			Border: lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(c.Border).
				Padding(1, 2),
		},

		Search: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("237")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Primary),

		Command: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("237")).
			Padding(0, 1),

		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Secondary).
			Padding(1, 2),

		Help: lipgloss.NewStyle().
			Foreground(c.Muted),

		StatusBar: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("236")),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(c.Error).
			Bold(true),
	}
}

// Helpers for common styling operations

// Truncate truncates a string to a maximum width with ellipsis
func Truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// PadRight pads a string to a minimum width
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + lipgloss.NewStyle().Width(width-len(s)).Render("")
}

// StatusIcon returns an icon based on status
func StatusIcon(status string) string {
	switch status {
	case "running", "active", "available", "enabled":
		return "●"
	case "stopped", "inactive", "disabled":
		return "○"
	case "pending", "creating", "modifying":
		return "◐"
	case "error", "failed", "deleted":
		return "✗"
	default:
		return "?"
	}
}

// StatusColor returns a color based on status
func StatusColor(status string) lipgloss.Color {
	c := DefaultColors()
	switch status {
	case "running", "active", "available", "enabled":
		return c.Success
	case "stopped", "inactive", "disabled":
		return c.Muted
	case "pending", "creating", "modifying":
		return c.Warning
	case "error", "failed", "deleted":
		return c.Error
	default:
		return c.Foreground
	}
}
