package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Search   key.Binding
	Enter    key.Binding
	Back     key.Binding
	Tab      key.Binding
	Daily    key.Binding
	Weekly   key.Binding
	Monthly  key.Binding
	PrevDate key.Binding
	NextDate key.Binding
	Open     key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
	Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "period")),
	Daily:    key.NewBinding(key.WithKeys("1")),
	Weekly:   key.NewBinding(key.WithKeys("2")),
	Monthly:  key.NewBinding(key.WithKeys("3")),
	PrevDate: key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/←", "prev")),
	NextDate: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/→", "next")),
	Open:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// ShortHelp returns short help key bindings (for help.Model)
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Search, k.Enter, k.Back, k.Tab, k.Quit}
}

// FullHelp returns full help key bindings
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Search, k.Enter, k.Back},
		{k.Tab, k.Daily, k.Weekly, k.Monthly},
		{k.PrevDate, k.NextDate, k.Open, k.Refresh},
		{k.Help, k.Quit},
	}
}
