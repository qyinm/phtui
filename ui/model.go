package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	detail    types.ProductDetail
	requestID int
}

// NewModel creates a new Model with the given ProductSource
func NewModel(source types.ProductSource) Model {
	l := list.New([]list.Item{}, NewProductDelegate(), 0, 0)
	l.Title = "🔥 Daily"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = TitleStyle

	vp := viewport.New(0, 0)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(DraculaPink)

	h := help.New()
	h.Styles.ShortKey = HelpKeyStyle
	h.Styles.ShortDesc = HelpDescStyle
	h.Styles.FullKey = HelpKeyStyle
	h.Styles.FullDesc = HelpDescStyle
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(DraculaComment)
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(DraculaComment)

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
		loading:   source != nil,
		requestID: 1,
		statusMsg: "Ready",
	}
}

// Init starts the initial leaderboard fetch
func (m Model) Init() tea.Cmd {
	if m.source == nil {
		return nil
	}
	return tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))
}

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case leaderboardMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "Failed to fetch: " + msg.err.Error()
			return m, nil
		}
		items := make([]list.Item, len(msg.products))
		for i, p := range msg.products {
			items[i] = p
		}
		m.list.SetItems(items)
		m.list.ResetSelected()
		m.list.Title = fmt.Sprintf("🔥 %s — %s", m.periodDisplayName(), m.formatDate())
		m.err = nil
		if len(msg.products) == 0 {
			m.statusMsg = "No products found for this period"
		} else {
			m.statusMsg = fmt.Sprintf("Loaded %d products", len(msg.products))
		}
		return m, nil

	case productDetailMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "Failed to fetch: " + msg.err.Error()
			return m, nil
		}
		m.detail = msg.detail
		m.viewport.SetContent(m.renderDetailContent())
		m.viewport.GotoTop()
		m.state = DetailView
		m.err = nil
		m.statusMsg = m.detail.Product().Name()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		// Block other keys while loading
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.resizePanes()
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			switch m.period {
			case types.Daily:
				m.period = types.Weekly
			case types.Weekly:
				m.period = types.Monthly
			case types.Monthly:
				m.period = types.Daily
			}
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.Daily):
			if m.period == types.Daily {
				return m, nil
			}
			m.period = types.Daily
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.Weekly):
			if m.period == types.Weekly {
				return m, nil
			}
			m.period = types.Weekly
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.Monthly):
			if m.period == types.Monthly {
				return m, nil
			}
			m.period = types.Monthly
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.PrevDate):
			switch m.period {
			case types.Daily:
				m.date = m.date.AddDate(0, 0, -1)
			case types.Weekly:
				m.date = m.date.AddDate(0, 0, -7)
			case types.Monthly:
				m.date = m.date.AddDate(0, -1, 0)
			}
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.NextDate):
			var next time.Time
			switch m.period {
			case types.Daily:
				next = m.date.AddDate(0, 0, 1)
			case types.Weekly:
				next = m.date.AddDate(0, 0, 7)
			case types.Monthly:
				next = m.date.AddDate(0, 1, 0)
			}
			if next.After(time.Now()) {
				return m, nil
			}
			m.date = next
			m.state = ListView
			m.loading = true
			m.statusMsg = "Loading..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.Refresh):
			m.state = ListView
			m.loading = true
			m.statusMsg = "Refreshing..."
			if m.source == nil {
				return m, nil
			}
			m.requestID++
			return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))

		case key.Matches(msg, m.keys.Open):
			var url string
			switch m.state {
			case ListView:
				if item := m.list.SelectedItem(); item != nil {
					if p, ok := item.(types.Product); ok && p.Slug() != "" {
						url = "https://www.producthunt.com/products/" + p.Slug()
					}
				}
			case DetailView:
				if m.detail.Product().Slug() != "" {
					url = "https://www.producthunt.com/products/" + m.detail.Product().Slug()
				}
			}
			if url != "" {
				_ = exec.Command("open", url).Start()
			}
			return m, nil
		}

		switch m.state {
		case ListView:
			if key.Matches(msg, m.keys.Enter) {
				item := m.list.SelectedItem()
				if item == nil {
					return m, nil
				}
				p, ok := item.(types.Product)
				if !ok || p.Slug() == "" {
					return m, nil
				}
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Loading detail..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchProductDetail(m.source, p.Slug(), m.requestID))
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case DetailView:
			if key.Matches(msg, m.keys.Back) {
				m.state = ListView
				m.statusMsg = fmt.Sprintf("%d products", len(m.list.Items()))
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.resizePanes()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Check if terminal is too small
	if m.width < 60 || m.height < 15 {
		return lipgloss.NewStyle().
			Foreground(DraculaOrange).
			Render("Terminal too small. Resize to at least 60x15.")
	}

	var sections []string

	if m.state == ListView || m.loading {
		sections = append(sections, m.renderTabBar())
	}

	if m.loading {
		available := m.height - 3 // tab + status + help
		if available < 1 {
			available = 1
		}
		spin := m.spinner.View() + " Loading..."
		sections = append(sections, lipgloss.Place(m.width, available, lipgloss.Center, lipgloss.Center, spin))
	} else {
		switch m.state {
		case ListView:
			if len(m.list.Items()) == 0 {
				available := m.height - 3 // tab + status + help
				if available < 1 {
					available = 1
				}
				msg := lipgloss.NewStyle().Foreground(DraculaComment).Render("No products found for this period")
				sections = append(sections, lipgloss.Place(m.width, available, lipgloss.Center, lipgloss.Center, msg))
			} else {
				sections = append(sections, m.list.View())
			}
		case DetailView:
			sections = append(sections, m.viewport.View())
		}
	}

	if m.err != nil {
		sections = append(sections, ErrorStyle.Render("Error: "+m.err.Error()))
	} else {
		sections = append(sections, StatusBarStyle.Render(m.statusMsg))
	}

	sections = append(sections, m.help.View(m.keys))

	return strings.Join(sections, "\n")
}

// renderTabBar builds the period tab bar with date
func (m Model) renderTabBar() string {
	tabs := []struct {
		label  string
		period types.Period
	}{
		{"Daily", types.Daily},
		{"Weekly", types.Weekly},
		{"Monthly", types.Monthly},
	}

	var parts []string
	for _, t := range tabs {
		if t.period == m.period {
			parts = append(parts, ActiveTabStyle.Render(t.label))
		} else {
			parts = append(parts, InactiveTabStyle.Render(t.label))
		}
	}

	sep := lipgloss.NewStyle().Foreground(DraculaComment).Render(" — ")
	dateStr := lipgloss.NewStyle().Foreground(DraculaComment).Render(m.formatDate())

	return strings.Join(parts, "") + sep + dateStr
}

// formatDate returns the date formatted for the current period
func (m Model) formatDate() string {
	switch m.period {
	case types.Daily:
		return m.date.Format("January 2, 2006")
	case types.Weekly:
		_, week := m.date.ISOWeek()
		return fmt.Sprintf("Week %d, %d", week, m.date.Year())
	case types.Monthly:
		return m.date.Format("January 2006")
	default:
		return m.date.Format("January 2, 2006")
	}
}

func (m Model) periodDisplayName() string {
	switch m.period {
	case types.Daily:
		return "Daily"
	case types.Weekly:
		return "Weekly"
	case types.Monthly:
		return "Monthly"
	default:
		return "Daily"
	}
}

// renderDetailContent formats ProductDetail for the viewport
func (m Model) renderDetailContent() string {
	d := m.detail
	p := d.Product()

	var b strings.Builder

	b.WriteString(DetailTitleStyle.Render(p.Name()))
	b.WriteString("\n")
	b.WriteString(DetailTaglineStyle.Render(p.Tagline()))
	b.WriteString("\n\n")

	stats := fmt.Sprintf("⭐ %.1f (%d reviews) • %s followers",
		d.Rating(), d.ReviewCount(), formatVoteCount(d.FollowerCount()))
	b.WriteString(stats)
	b.WriteString("\n")

	if d.WebsiteURL() != "" {
		b.WriteString(fmt.Sprintf("🌐 %s\n", d.WebsiteURL()))
	}

	b.WriteString("\n")

	if d.Description() != "" {
		b.WriteString(d.Description())
		b.WriteString("\n")
	}

	if d.MakerComment() != "" {
		b.WriteString("\n--- Maker Comment ---\n")
		b.WriteString(d.MakerComment())
		b.WriteString("\n")
	}

	if len(d.Categories()) > 0 {
		b.WriteString("\nCategories: ")
		b.WriteString(strings.Join(d.Categories(), " • "))
		b.WriteString("\n")
	}

	if len(d.SocialLinks()) > 0 {
		b.WriteString("\nSocial:\n")
		for _, link := range d.SocialLinks() {
			b.WriteString("- ")
			b.WriteString(link)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// resizePanes adjusts dimensions of list and viewport based on window size
func (m *Model) resizePanes() {
	if m.width == 0 {
		return
	}

	// Chrome: tab bar (1) + status bar (1) + help (1) = 3
	chrome := 3
	listHeight := m.height - chrome
	if listHeight < 0 {
		listHeight = 0
	}

	// Detail view has no tab bar — gets 1 extra line
	detailHeight := listHeight + 1
	if detailHeight > m.height {
		detailHeight = m.height
	}

	m.list.SetSize(m.width, listHeight)
	m.viewport.Width = m.width
	m.viewport.Height = detailHeight
}
