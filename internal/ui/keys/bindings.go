package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all application key bindings
type KeyMap struct {
	// Navigation
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Top          key.Binding
	Bottom       key.Binding

	// Mode switching
	Search  key.Binding
	Command key.Binding
	Escape  key.Binding

	// Actions
	Enter    key.Binding
	Describe key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Refresh  key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding

	// View toggles
	ToggleYAML key.Binding
	ToggleWrap key.Binding

	// Clipboard
	CopyID   key.Binding
	CopyJSON key.Binding

	// Bookmarks
	Bookmark     key.Binding
	GoToBookmark key.Binding

	// Tags
	FilterByTag key.Binding

	// Tab
	Tab key.Binding
}

// DefaultKeyMap returns the default vim-style key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation (vim-style)
		Up:           key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Left:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "back")),
		Right:        key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "enter")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("C-u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("C-d", "½ page down")),
		Top:          key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:       key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),

		// Mode switching
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Command: key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
		Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),

		// Actions
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Describe: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "describe")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("C-d", "delete")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

		// View toggles
		ToggleYAML: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yaml")),
		ToggleWrap: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "wrap")),

		// Clipboard
		CopyID:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy ARN")),
		CopyJSON: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "copy JSON")),

		// Bookmarks
		Bookmark:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "bookmark")),
		GoToBookmark: key.NewBinding(key.WithKeys("'"), key.WithHelp("'", "go to mark")),

		// Tags
		FilterByTag: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "filter tags")),

		// Tab navigation
		Tab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	}
}

// ShortHelp returns keybindings to show in the help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Search, k.Command, k.Describe, k.Quit, k.Help}
}

// FullHelp returns all keybindings for the help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.HalfPageUp, k.HalfPageDown, k.Top, k.Bottom},
		{k.Search, k.Command, k.Escape},
		{k.Enter, k.Describe, k.Edit, k.Refresh},
		{k.CopyID, k.CopyJSON, k.ToggleYAML},
		{k.Bookmark, k.GoToBookmark, k.FilterByTag},
		{k.Quit, k.Help},
	}
}
