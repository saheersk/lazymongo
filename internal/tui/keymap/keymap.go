// Package keymap defines the application-wide key bindings.
package keymap

import "github.com/charmbracelet/bubbles/key"

// Map holds every bindable action. Panels store a *Map so all keybindings
// live in one place and are easy to remap via config.
type Map struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	Select  key.Binding
	Back    key.Binding
	Refresh key.Binding
	Filter  key.Binding
	Quit    key.Binding
	Help    key.Binding

	NewDoc    key.Binding
	EditDoc   key.Binding
	DeleteDoc key.Binding
	CopyID    key.Binding
	CopyDoc   key.Binding
}

// Default returns the standard vim-style key bindings.
func Default() *Map {
	return &Map{
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Left:     key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "sidebar")),
		Right:    key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "documents")),
		Top:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		PageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "page up")),
		PageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "page down")),

		Select:  key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "select")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Refresh: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

		NewDoc:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		EditDoc:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		DeleteDoc: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		CopyID:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy id")),
		CopyDoc:   key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy doc")),
	}
}

// ShortHelp returns the minimal hint shown in the footer.
func (m *Map) ShortHelp() []key.Binding {
	return []key.Binding{m.Up, m.Down, m.Left, m.Right, m.Select, m.Filter, m.Quit, m.Help}
}

// FullHelp returns all bindings grouped by category.
func (m *Map) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Up, m.Down, m.Top, m.Bottom},
		{m.Left, m.Right, m.Select, m.Back},
		{m.PageUp, m.PageDown, m.Refresh, m.Filter},
		{m.NewDoc, m.EditDoc, m.DeleteDoc},
		{m.CopyID, m.CopyDoc, m.Quit, m.Help},
	}
}
