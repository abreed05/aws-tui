package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aaw-tui/aws-tui/internal/handlers"
	"github.com/aaw-tui/aws-tui/internal/ui/styles"
)

// ResourceSelectedMsg is sent when a resource is selected
type ResourceSelectedMsg struct {
	Resource handlers.Resource
}

// Table displays resources in a scrollable table
type Table struct {
	columns   []handlers.ColumnDef
	rows      [][]string
	resources []handlers.Resource

	// State
	cursor     int
	offset     int
	filter     string
	filtered   []int // Indices of filtered rows

	// Dimensions
	width  int
	height int

	// Styling
	theme styles.Theme

	// Focus
	focused bool
}

// NewTable creates a new table component
func NewTable(theme styles.Theme) *Table {
	return &Table{
		theme:    theme,
		focused:  true,
		filtered: make([]int, 0),
	}
}

// SetSize sets the table dimensions
func (t *Table) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetColumns sets the column definitions
func (t *Table) SetColumns(columns []handlers.ColumnDef) {
	t.columns = columns
}

// SetResources updates the table with new resources
func (t *Table) SetResources(resources []handlers.Resource) {
	t.resources = resources
	t.rows = make([][]string, len(resources))

	for i, res := range resources {
		t.rows[i] = res.ToTableRow()
	}

	// Reset filter
	t.filtered = make([]int, len(resources))
	for i := range resources {
		t.filtered[i] = i
	}

	// Reset cursor if out of bounds
	if t.cursor >= len(t.filtered) {
		t.cursor = 0
	}
	t.offset = 0
}

// ApplyFilter filters the displayed rows
func (t *Table) ApplyFilter(filter string) {
	t.filter = strings.ToLower(filter)
	t.filtered = make([]int, 0)

	if t.filter == "" {
		// No filter, show all
		for i := range t.rows {
			t.filtered = append(t.filtered, i)
		}
	} else {
		// Filter rows
		for i, row := range t.rows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), t.filter) {
					t.filtered = append(t.filtered, i)
					break
				}
			}
		}
	}

	// Reset cursor if out of bounds
	if t.cursor >= len(t.filtered) {
		t.cursor = 0
	}
	t.offset = 0
}

// SelectedResource returns the currently selected resource
func (t *Table) SelectedResource() handlers.Resource {
	if len(t.filtered) == 0 || t.cursor >= len(t.filtered) {
		return nil
	}
	idx := t.filtered[t.cursor]
	if idx >= len(t.resources) {
		return nil
	}
	return t.resources[idx]
}

// SelectedIndex returns the index of the selected resource
func (t *Table) SelectedIndex() int {
	if len(t.filtered) == 0 || t.cursor >= len(t.filtered) {
		return -1
	}
	return t.filtered[t.cursor]
}

// Focus sets the focus state
func (t *Table) Focus() {
	t.focused = true
}

// Blur removes focus
func (t *Table) Blur() {
	t.focused = false
}

// IsFocused returns whether the table is focused
func (t *Table) IsFocused() bool {
	return t.focused
}

// Len returns the number of visible rows
func (t *Table) Len() int {
	return len(t.filtered)
}

// Update handles messages
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			t.moveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			t.moveUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g", "home"))):
			t.moveToTop()
		case key.Matches(msg, key.NewBinding(key.WithKeys("G", "end"))):
			t.moveToBottom()
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
			t.moveHalfPageDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
			t.moveHalfPageUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "l"))):
			if res := t.SelectedResource(); res != nil {
				return t, func() tea.Msg {
					return ResourceSelectedMsg{Resource: res}
				}
			}
		}
	}

	return t, nil
}

func (t *Table) moveDown() {
	if len(t.filtered) == 0 {
		return
	}
	if t.cursor < len(t.filtered)-1 {
		t.cursor++
		t.ensureVisible()
	}
}

func (t *Table) moveUp() {
	if t.cursor > 0 {
		t.cursor--
		t.ensureVisible()
	}
}

func (t *Table) moveToTop() {
	t.cursor = 0
	t.offset = 0
}

func (t *Table) moveToBottom() {
	if len(t.filtered) > 0 {
		t.cursor = len(t.filtered) - 1
		t.ensureVisible()
	}
}

func (t *Table) moveHalfPageDown() {
	pageSize := t.visibleRows() / 2
	if pageSize < 1 {
		pageSize = 1
	}
	t.cursor += pageSize
	if t.cursor >= len(t.filtered) {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

func (t *Table) moveHalfPageUp() {
	pageSize := t.visibleRows() / 2
	if pageSize < 1 {
		pageSize = 1
	}
	t.cursor -= pageSize
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

func (t *Table) visibleRows() int {
	return t.height - 3 // Account for header and borders
}

func (t *Table) ensureVisible() {
	visible := t.visibleRows()
	if visible <= 0 {
		return
	}

	// Scroll down if cursor is below visible area
	if t.cursor >= t.offset+visible {
		t.offset = t.cursor - visible + 1
	}

	// Scroll up if cursor is above visible area
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
}

// View renders the table
func (t *Table) View() string {
	if t.width == 0 || t.height == 0 {
		return ""
	}

	var sb strings.Builder

	// Render header
	sb.WriteString(t.renderHeader())
	sb.WriteString("\n")

	// Render separator
	sb.WriteString(t.renderSeparator())
	sb.WriteString("\n")

	// Calculate visible rows
	visible := t.visibleRows()
	if visible < 1 {
		visible = 1
	}

	// Render rows
	for i := 0; i < visible; i++ {
		rowIdx := t.offset + i
		if rowIdx >= len(t.filtered) {
			// Empty row
			sb.WriteString(strings.Repeat(" ", t.width))
		} else {
			actualIdx := t.filtered[rowIdx]
			isSelected := rowIdx == t.cursor
			sb.WriteString(t.renderRow(t.rows[actualIdx], isSelected))
		}
		if i < visible-1 {
			sb.WriteString("\n")
		}
	}

	// Status line
	sb.WriteString("\n")
	sb.WriteString(t.renderStatus())

	return sb.String()
}

func (t *Table) renderHeader() string {
	headerStyle := t.theme.Table.Header

	var cells []string
	totalWidth := 0

	for _, col := range t.columns {
		cell := truncateOrPad(col.Title, col.Width)
		cells = append(cells, cell)
		totalWidth += col.Width + 1
	}

	// Fill remaining width
	if totalWidth < t.width {
		cells = append(cells, strings.Repeat(" ", t.width-totalWidth))
	}

	return headerStyle.Render(strings.Join(cells, " "))
}

func (t *Table) renderSeparator() string {
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var parts []string
	for _, col := range t.columns {
		parts = append(parts, strings.Repeat("─", col.Width))
	}

	return sepStyle.Render(strings.Join(parts, "─"))
}

func (t *Table) renderRow(row []string, selected bool) string {
	var style lipgloss.Style
	if selected && t.focused {
		style = t.theme.Table.Selected
	} else {
		style = t.theme.Table.Row
	}

	var cells []string
	totalWidth := 0

	for i, col := range t.columns {
		var cellValue string
		if i < len(row) {
			cellValue = row[i]
		}
		cell := truncateOrPad(cellValue, col.Width)
		cells = append(cells, cell)
		totalWidth += col.Width + 1
	}

	content := strings.Join(cells, " ")

	// Pad to full width for selection highlight
	if totalWidth < t.width {
		content += strings.Repeat(" ", t.width-totalWidth)
	}

	return style.Width(t.width).Render(content)
}

func (t *Table) renderStatus() string {
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	total := len(t.resources)
	filtered := len(t.filtered)
	current := t.cursor + 1

	var status string
	if t.filter != "" {
		status = fmt.Sprintf(" %d/%d (filtered from %d) ", current, filtered, total)
	} else {
		status = fmt.Sprintf(" %d/%d ", current, total)
	}

	return statusStyle.Render(status)
}

// Helper to truncate or pad a string to a specific width
func truncateOrPad(s string, width int) string {
	if len(s) > width {
		if width <= 3 {
			return s[:width]
		}
		return s[:width-3] + "..."
	}
	return s + strings.Repeat(" ", width-len(s))
}
