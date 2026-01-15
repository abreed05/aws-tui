package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// Breadcrumb displays navigation path
type Breadcrumb struct {
	path  []string
	width int
	theme styles.Theme
}

// NewBreadcrumb creates a new breadcrumb component
func NewBreadcrumb(theme styles.Theme) *Breadcrumb {
	return &Breadcrumb{
		theme: theme,
		path:  []string{"Home"},
	}
}

// SetPath sets the breadcrumb path
func (b *Breadcrumb) SetPath(path ...string) {
	if len(path) == 0 {
		b.path = []string{"Home"}
	} else {
		b.path = path
	}
}

// Push adds an item to the path
func (b *Breadcrumb) Push(item string) {
	b.path = append(b.path, item)
}

// Pop removes the last item from the path
func (b *Breadcrumb) Pop() string {
	if len(b.path) <= 1 {
		return ""
	}
	last := b.path[len(b.path)-1]
	b.path = b.path[:len(b.path)-1]
	return last
}

// Current returns the current (last) item in the path
func (b *Breadcrumb) Current() string {
	if len(b.path) == 0 {
		return ""
	}
	return b.path[len(b.path)-1]
}

// SetWidth sets the breadcrumb width
func (b *Breadcrumb) SetWidth(width int) {
	b.width = width
}

// View renders the breadcrumb
func (b *Breadcrumb) View() string {
	if len(b.path) == 0 {
		return ""
	}

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)

	separator := separatorStyle.Render(" › ")

	var parts []string
	for i, item := range b.path {
		if i == len(b.path)-1 {
			// Current item
			parts = append(parts, currentStyle.Render(item))
		} else {
			parts = append(parts, itemStyle.Render(item))
		}
	}

	content := strings.Join(parts, separator)

	return b.theme.Breadcrumb.Width(b.width).Render(content)
}
