package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/qyinm/phtui/types"
)

// ViewState represents the current view mode
type ViewState int

const (
	ListView ViewState = iota
	DetailView
)

// Model is the main TUI model
type Model struct {
	source    types.ProductSource
	list      list.Model
	viewport  viewport.Model
	spinner   spinner.Model
	help      help.Model
	keys      keyMap
	state     ViewState
	period    types.Period
	date      time.Time
	width     int
	height    int
	loading   bool
	err       error
	statusMsg string
}

// NewModel creates a new Model with the given ProductSource
func NewModel(source types.ProductSource) Model {
	// Create list with custom product delegate
	l := list.New([]list.Item{}, ProductDelegate{}, 0, 0)
	l.Title = "Product Hunt Leaderboard"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = TitleStyle

	// Add placeholder items for testing
	placeholders := []list.Item{
		types.NewProduct("Placeholder 1", "Test product 1", []string{"test"}, 100, 5, "placeholder-1", "", 1),
		types.NewProduct("Placeholder 2", "Test product 2", []string{"test"}, 200, 10, "placeholder-2", "", 2),
		types.NewProduct("Placeholder 3", "Test product 3", []string{"test"}, 300, 15, "placeholder-3", "", 3),
	}
	l.SetItems(placeholders)

	// Create viewport
	vp := viewport.New(0, 0)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot

	// Create help
	h := help.New()

	return Model{
		source:    source,
		list:      l,
		viewport:  vp,
		spinner:   s,
		help:      h,
		keys:      keys,
		state:     ListView,
		period:    types.Daily,
		date:      time.Now(),
		width:     0,
		height:    0,
		loading:   false,
		err:       nil,
		statusMsg: "Ready",
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "q" || msg.String() == "ctrl+c":
			return m, tea.Quit
		}

		// Delegate to list or viewport based on state
		switch m.state {
		case ListView:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case DetailView:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanes()
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.state {
	case ListView:
		return m.list.View()
	case DetailView:
		return m.viewport.View()
	default:
		return "Unknown state\n"
	}
}

// resizePanes adjusts the dimensions of list and viewport based on window size
func (m *Model) resizePanes() {
	// Reserve space for status bar and help
	statusHeight := 1
	helpHeight := 1
	availableHeight := m.height - statusHeight - helpHeight

	if availableHeight < 0 {
		availableHeight = 0
	}

	// Update list dimensions
	m.list.SetSize(m.width, availableHeight)

	// Update viewport dimensions
	m.viewport.Width = m.width
	m.viewport.Height = availableHeight
}
