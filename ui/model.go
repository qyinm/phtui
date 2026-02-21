package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

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

// tabRegion represents a clickable region in the tab bar.
type tabRegion struct {
	xStart, xEnd int
	period       types.Period
	isCategory   bool // true if this region is the Categories tab
}

// dateRegion represents a clickable region in the date bar.
type dateRegion struct {
	xStart, xEnd int
	action       string    // "prev", "next", or "goto"
	date         time.Time // target date when action is "goto"
}

// Model is the main TUI model
type Model struct {
	source         types.ProductSource
	list           list.Model
	products       []types.Product
	selected       int
	viewport       viewport.Model
	spinner        spinner.Model
	help           help.Model
	keys           keyMap
	state          ViewState
	period         types.Period
	date           time.Time
	width          int
	height         int
	loading        bool
	err            error
	statusMsg      string
	detail         types.ProductDetail
	requestID      int
	dateBarRegions []dateRegion
	searchMode     bool
	searchQuery    string
	searchResults  bool
	searchPage     int
	searchHasPrev  bool
	searchHasNext  bool
	searchPages    int
	// Category browsing
	categoryMode bool
	categorySlug string
	categoryName string
	categoryIdx  int // index within AllCategories for h/l nav
	// Category split pane (left: categories, right: products)
	categorySelectMode bool            // true = split pane mode
	catSelectIdx       int             // left pane cursor position
	catFilterMode      bool            // true = typing category filter query
	catFilterQuery     string          // filter text
	catFilteredIndices []int           // indices into AllCategories matching filter
	splitFocus         int             // 0=left(categories), 1=right(products)
	splitProducts      []types.Product // right pane product list
	splitSelected      int             // right pane product cursor
	splitLoading       bool            // right pane loading
	splitSlug          string          // slug of loaded category in right pane
	splitRequestID     int             // request id for in-flight split-pane category fetch
}

// NewModel creates a new Model with the given ProductSource
func NewModel(source types.ProductSource) Model {
	l := newProductListModel(nil, 80, 20)

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
		products:  nil,
		selected:  0,
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

func newProductListModel(items []list.Item, width, height int) list.Model {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}

	l := list.New(items, NewProductDelegate(), width, height)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = TitleStyle
	return l
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
		m.products = msg.products
		m.searchResults = false
		m.searchPage = 0
		m.searchHasPrev = false
		m.searchHasNext = false
		m.searchPages = 0
		m.categoryMode = false
		m.categorySlug = ""
		m.categoryName = ""
		m.selected = 0
		listHeight := m.height - 4
		if listHeight < 1 {
			listHeight = 1
		}
		items := make([]list.Item, len(m.products))
		for i, p := range m.products {
			items[i] = p
		}
		m.list = newProductListModel(items, m.width, listHeight)
		m.list.Paginator.Page = 0
		m.list.Select(0)
		m.list.ResetSelected()
		m.err = nil
		if len(m.products) == 0 {
			m.statusMsg = "No products found for this period"
		} else {
			firstRank := m.products[0].Rank()
			lastRank := m.products[len(m.products)-1].Rank()
			selectedRank := firstRank
			if p, ok := m.selectedProduct(); ok {
				selectedRank = p.Rank()
			}
			m.statusMsg = fmt.Sprintf("Loaded %d products (ranks %d-%d, selected #%d)", len(m.products), firstRank, lastRank, selectedRank)
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

	case searchResultsMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "Search failed: " + msg.err.Error()
			return m, nil
		}
		m.searchQuery = msg.query
		m.searchMode = false
		m.searchResults = true
		m.searchPage = msg.page
		m.searchHasPrev = msg.hasPrev
		m.searchHasNext = msg.hasNext
		m.searchPages = msg.pages
		m.products = msg.products
		m.selected = 0

		listHeight := m.height - 4
		if listHeight < 1 {
			listHeight = 1
		}
		items := make([]list.Item, len(m.products))
		for i, p := range m.products {
			items[i] = p
		}
		m.list = newProductListModel(items, m.width, listHeight)
		m.list.Paginator.Page = 0
		m.list.Select(0)
		m.list.ResetSelected()
		m.err = nil
		m.statusMsg = m.searchStatus()
		return m, nil

	case categoryProductsMsg:
		if m.categorySelectMode {
			if msg.requestID != m.splitRequestID {
				return m, nil
			}
			// Split pane mode — update right pane only
			m.splitLoading = false
			m.loading = false
			if msg.err != nil {
				m.err = msg.err
				m.statusMsg = "Failed to fetch: " + msg.err.Error()
				return m, nil
			}
			m.splitProducts = msg.products
			m.splitSelected = 0
			m.splitSlug = msg.slug
			m.err = nil
			// Derive display name for status
			catName := slugToDisplayName(msg.slug)
			idx := types.CategoryIndexBySlug(msg.slug)
			if idx >= 0 && idx < len(types.AllCategories) && types.AllCategories[idx].Slug() == msg.slug {
				catName = types.AllCategories[idx].Name()
			}
			if len(m.splitProducts) == 0 {
				m.statusMsg = fmt.Sprintf("No products in %s", catName)
			} else {
				m.statusMsg = fmt.Sprintf("%d products in %s", len(m.splitProducts), catName)
			}
			return m, nil
		}
		// Ignore category responses when we are not actively waiting for one.
		// This prevents stale split-pane responses from switching the main view.
		if !m.loading {
			return m, nil
		}
		if msg.requestID != m.requestID {
			return m, nil
		}
		// Standalone category mode (via h/l navigation)
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "Failed to fetch category: " + msg.err.Error()
			return m, nil
		}
		m.categoryMode = true
		m.categorySelectMode = false
		m.categorySlug = msg.slug
		m.searchResults = false
		m.searchPage = 0
		m.searchHasPrev = false
		m.searchHasNext = false
		m.searchPages = 0
		m.categoryIdx = types.CategoryIndexBySlug(msg.slug)
		if m.categoryIdx < 0 {
			m.categoryIdx = 0
		}
		if m.categoryIdx >= 0 && m.categoryIdx < len(types.AllCategories) && types.AllCategories[m.categoryIdx].Slug() == msg.slug {
			m.categoryName = types.AllCategories[m.categoryIdx].Name()
		} else {
			m.categoryName = slugToDisplayName(msg.slug)
		}
		m.products = msg.products
		m.selected = 0

		listHeight := m.height - 4
		if listHeight < 1 {
			listHeight = 1
		}
		items := make([]list.Item, len(m.products))
		for i, p := range m.products {
			items[i] = p
		}
		m.list = newProductListModel(items, m.width, listHeight)
		m.list.Paginator.Page = 0
		m.list.Select(0)
		m.list.ResetSelected()
		m.err = nil
		if len(m.products) == 0 {
			m.statusMsg = fmt.Sprintf("No products in %s", m.categoryName)
		} else {
			m.statusMsg = fmt.Sprintf("%d products in %s", len(m.products), m.categoryName)
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.splitLoading {
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

		if m.state == ListView && m.searchMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.searchMode = false
				m.statusMsg = m.searchStatus()
				return m, nil
			case tea.KeyEnter:
				query := strings.TrimSpace(m.searchQuery)
				m.searchMode = false
				if query == "" {
					m.statusMsg = m.searchStatus()
					return m, nil
				}
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Searching..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchSearchResults(m.source, query, 1, m.requestID))
			case tea.KeyCtrlU:
				m.searchQuery = ""
				m.statusMsg = m.searchStatus()
				return m, nil
			case tea.KeySpace:
				m.searchQuery += " "
				m.statusMsg = m.searchStatus()
				return m, nil
			case tea.KeyBackspace, tea.KeyDelete:
				if m.searchQuery != "" {
					_, size := utf8.DecodeLastRuneInString(m.searchQuery)
					if size > 0 {
						m.searchQuery = m.searchQuery[:len(m.searchQuery)-size]
					}
				}
				m.statusMsg = m.searchStatus()
				return m, nil
			}

			if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
				m.searchQuery += string(msg.Runes)
				m.statusMsg = m.searchStatus()
				return m, nil
			}
		}

		// Category filter text input (typing to filter categories)
		if m.categorySelectMode && m.catFilterMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.catFilterMode = false
				m.catFilterQuery = ""
				m.catFilteredIndices = nil
				m.catSelectIdx = 0
				m.statusMsg = fmt.Sprintf("Select a category (%d categories)", len(types.AllCategories))
				return m, nil
			case tea.KeyEnter:
				// Apply filter and select first match — load products in right pane
				visible := m.catVisibleList()
				if len(visible) > 0 && m.catSelectIdx < len(visible) {
					m.catFilterMode = false
					return m, m.loadSelectedCategory()
				}
				return m, nil
			case tea.KeyCtrlU:
				m.catFilterQuery = ""
				m.updateCatFilter()
				m.catSelectIdx = 0
				m.statusMsg = fmt.Sprintf("Filter: %s", m.catFilterQuery)
				return m, nil
			case tea.KeySpace:
				m.catFilterQuery += " "
				m.updateCatFilter()
				m.statusMsg = fmt.Sprintf("Filter: %s", m.catFilterQuery)
				return m, nil
			case tea.KeyBackspace, tea.KeyDelete:
				if m.catFilterQuery != "" {
					_, size := utf8.DecodeLastRuneInString(m.catFilterQuery)
					if size > 0 {
						m.catFilterQuery = m.catFilterQuery[:len(m.catFilterQuery)-size]
					}
					m.updateCatFilter()
					m.catSelectIdx = 0
				}
				m.statusMsg = fmt.Sprintf("Filter: %s", m.catFilterQuery)
				return m, nil
			}

			if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
				m.catFilterQuery += string(msg.Runes)
				m.updateCatFilter()
				m.catSelectIdx = 0
				m.statusMsg = fmt.Sprintf("Filter: %s", m.catFilterQuery)
				return m, nil
			}

			// Allow j/k navigation while filtering
			if key.Matches(msg, m.keys.Down) {
				visible := m.catVisibleList()
				if m.catSelectIdx < len(visible)-1 {
					m.catSelectIdx++
				}
				return m, nil
			}
			if key.Matches(msg, m.keys.Up) {
				if m.catSelectIdx > 0 {
					m.catSelectIdx--
				}
				return m, nil
			}
		}

		// Split pane mode — right pane focused (product list)
		if m.categorySelectMode && !m.catFilterMode && m.splitFocus == 1 {
			switch {
			case key.Matches(msg, m.keys.Back):
				// Esc → go back to left pane
				m.splitFocus = 0
				return m, nil
			case key.Matches(msg, m.keys.PrevDate):
				// h → go back to left pane
				m.splitFocus = 0
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.splitSelected < len(m.splitProducts)-1 {
					m.splitSelected++
				}
				return m, nil
			case key.Matches(msg, m.keys.Up):
				if m.splitSelected > 0 {
					m.splitSelected--
				}
				return m, nil
			case key.Matches(msg, m.keys.Enter):
				// Open product detail
				if m.splitSelected >= 0 && m.splitSelected < len(m.splitProducts) {
					p := m.splitProducts[m.splitSelected]
					if p.Slug() == "" || m.source == nil {
						return m, nil
					}
					m.loading = true
					m.statusMsg = "Loading detail..."
					m.requestID++
					return m, tea.Batch(m.spinner.Tick, fetchProductDetail(m.source, p.Slug(), m.requestID))
				}
				return m, nil
			case key.Matches(msg, m.keys.Open):
				if m.splitSelected >= 0 && m.splitSelected < len(m.splitProducts) {
					p := m.splitProducts[m.splitSelected]
					if p.Slug() != "" {
						_ = exec.Command("open", "https://www.producthunt.com/products/"+p.Slug()).Start()
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Split pane mode — left pane focused (category list)
		if m.categorySelectMode && !m.catFilterMode && m.splitFocus == 0 {
			switch {
			case key.Matches(msg, m.keys.Back):
				// Esc → exit split pane mode
				m.categorySelectMode = false
				m.splitLoading = false
				m.splitRequestID = 0
				m.requestID++ // invalidate any in-flight split-pane category response
				if m.categoryMode {
					m.statusMsg = fmt.Sprintf("%d products in %s", len(m.products), m.categoryName)
				} else {
					m.statusMsg = fmt.Sprintf("%d products", len(m.products))
				}
				return m, nil
			case key.Matches(msg, m.keys.Search):
				// / → enter filter mode
				m.catFilterMode = true
				m.catFilterQuery = ""
				m.catFilteredIndices = nil
				m.catSelectIdx = 0
				m.statusMsg = "Filter: "
				return m, nil
			case key.Matches(msg, m.keys.NextDate):
				// l → focus right pane
				if len(m.splitProducts) > 0 {
					m.splitFocus = 1
				}
				return m, nil
			case key.Matches(msg, m.keys.Enter):
				// Enter → focus right pane
				if len(m.splitProducts) > 0 {
					m.splitFocus = 1
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				visible := m.catVisibleList()
				if m.catSelectIdx < len(visible)-1 {
					m.catSelectIdx++
					return m, m.loadSelectedCategory()
				}
				return m, nil
			case key.Matches(msg, m.keys.Up):
				if m.catSelectIdx > 0 {
					m.catSelectIdx--
					return m, m.loadSelectedCategory()
				}
				return m, nil
			}
			// Ignore other keys in left pane
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.resizePanes()
			return m, nil

		case m.state == ListView && key.Matches(msg, m.keys.Search):
			m.searchMode = true
			m.statusMsg = m.searchStatus()
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			if m.categoryMode || m.categorySelectMode {
				// From categories/category-select → Daily leaderboard
				return m.switchToLeaderboard(types.Daily)
			}
			switch m.period {
			case types.Daily:
				return m.switchToLeaderboard(types.Weekly)
			case types.Weekly:
				return m.switchToLeaderboard(types.Monthly)
			case types.Monthly:
				// Monthly → Category split pane
				m.state = ListView
				cmd := m.enterCategorySelectMode()
				return m, cmd
			}

		case key.Matches(msg, m.keys.Daily):
			if m.period == types.Daily && !m.categoryMode && !m.categorySelectMode {
				return m, nil
			}
			return m.switchToLeaderboard(types.Daily)

		case key.Matches(msg, m.keys.Weekly):
			if m.period == types.Weekly && !m.categoryMode && !m.categorySelectMode {
				return m, nil
			}
			return m.switchToLeaderboard(types.Weekly)

		case key.Matches(msg, m.keys.Monthly):
			if m.period == types.Monthly && !m.categoryMode && !m.categorySelectMode {
				return m, nil
			}
			return m.switchToLeaderboard(types.Monthly)

		case key.Matches(msg, m.keys.Categories):
			if m.categorySelectMode {
				return m, nil
			}
			m.state = ListView
			cmd := m.enterCategorySelectMode()
			return m, cmd

		case key.Matches(msg, m.keys.PrevDate):
			if m.searchResults {
				if !m.searchHasPrev || m.searchPage <= 1 {
					return m, nil
				}
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Loading search page..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchSearchResults(m.source, m.searchQuery, m.searchPage-1, m.requestID))
			}
			if m.categoryMode {
				// Navigate to previous category in AllCategories
				all := types.AllCategories
				if len(all) == 0 {
					return m, nil
				}
				idx := types.CategoryIndexBySlug(m.categorySlug)
				if idx < 0 {
					idx = 0
				}
				idx--
				if idx < 0 {
					idx = len(all) - 1
				}
				slug := all[idx].Slug()
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Loading category..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchCategoryProducts(m.source, slug, m.requestID))
			}
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
			if m.searchResults {
				if !m.searchHasNext {
					return m, nil
				}
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Loading search page..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchSearchResults(m.source, m.searchQuery, m.searchPage+1, m.requestID))
			}
			if m.categoryMode {
				// Navigate to next category in AllCategories
				all := types.AllCategories
				if len(all) == 0 {
					return m, nil
				}
				idx := types.CategoryIndexBySlug(m.categorySlug)
				if idx < 0 {
					idx = 0
				}
				idx++
				if idx >= len(all) {
					idx = 0
				}
				slug := all[idx].Slug()
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Loading category..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchCategoryProducts(m.source, slug, m.requestID))
			}
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
			if m.searchResults {
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Refreshing search..."
				m.requestID++
				page := m.searchPage
				if page <= 0 {
					page = 1
				}
				return m, tea.Batch(m.spinner.Tick, fetchSearchResults(m.source, m.searchQuery, page, m.requestID))
			}
			if m.categoryMode && m.categorySlug != "" {
				if m.source == nil {
					return m, nil
				}
				m.loading = true
				m.statusMsg = "Refreshing category..."
				m.requestID++
				return m, tea.Batch(m.spinner.Tick, fetchCategoryProducts(m.source, m.categorySlug, m.requestID))
			}
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
				if p, ok := m.selectedProduct(); ok && p.Slug() != "" {
					url = "https://www.producthunt.com/products/" + p.Slug()
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
				p, ok := m.selectedProduct()
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
			if key.Matches(msg, m.keys.Up) {
				if m.selected > 0 {
					m.selected--
				}
				return m, nil
			}
			if key.Matches(msg, m.keys.Down) {
				if m.selected < len(m.products)-1 {
					m.selected++
				}
				return m, nil
			}
			return m, nil

		case DetailView:
			if key.Matches(msg, m.keys.Back) {
				m.state = ListView
				if m.categorySelectMode {
					// Returning to split pane — restore category status
					catName := slugToDisplayName(m.splitSlug)
					idx := types.CategoryIndexBySlug(m.splitSlug)
					if idx >= 0 && idx < len(types.AllCategories) {
						catName = types.AllCategories[idx].Name()
					}
					m.statusMsg = fmt.Sprintf("%d products in %s", len(m.splitProducts), catName)
				} else {
					m.statusMsg = m.searchStatus()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		if m.loading {
			return m, nil
		}
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease && m.state == ListView {
			// Row 0: period tab bar (Daily / Weekly / Monthly / Categories)
			if msg.Y == 0 {
				for _, r := range lastTabBarRegions {
					if msg.X >= r.xStart && msg.X < r.xEnd {
						if r.isCategory {
							if !m.categorySelectMode {
								m.state = ListView
								cmd := m.enterCategorySelectMode()
								return m, cmd
							}
						} else {
							if m.categoryMode || m.categorySelectMode || r.period != m.period {
								return m.switchToLeaderboard(r.period)
							}
						}
						break
					}
				}
			}
			// Row 1: date selector bar
			if msg.Y == 1 {
				for _, r := range lastDateBarRegions {
					if msg.X >= r.xStart && msg.X < r.xEnd {
						return m.handleDateBarClick(r)
					}
				}
			}
		}
		return m, nil

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
		available := m.height - 4 // tab + status + help
		if available < 1 {
			available = 1
		}
		spin := m.spinner.View() + " Loading..."
		sections = append(sections, lipgloss.Place(m.width, available, lipgloss.Center, lipgloss.Center, spin))
	} else {
		switch m.state {
		case ListView:
			if m.categorySelectMode {
				sections = append(sections, m.renderSplitPane())
			} else if len(m.products) == 0 {
				available := m.height - 4 // tab + status + help
				if available < 1 {
					available = 1
				}
				emptyText := "No products found for this period"
				if m.searchResults {
					emptyText = fmt.Sprintf("No results for \"%s\"", m.searchQuery)
				}
				msg := lipgloss.NewStyle().Foreground(DraculaComment).Render(emptyText)
				sections = append(sections, lipgloss.Place(m.width, available, lipgloss.Center, lipgloss.Center, msg))
			} else {
				sections = append(sections, m.renderProductList())
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

// renderTabBar builds the period tab bar (line 1) and date selector bar (line 2).
func (m Model) renderTabBar() string {
	// Line 1: period tabs + Categories
	type tabDef struct {
		label      string
		period     types.Period
		isCategory bool
	}
	tabs := []tabDef{
		{"Daily", types.Daily, false},
		{"Weekly", types.Weekly, false},
		{"Monthly", types.Monthly, false},
		{"Categories", 0, true},
	}

	var parts []string
	var tabRegs []tabRegion
	x := 0
	for _, t := range tabs {
		rendered := lipgloss.Width(t.label) + 2 // Padding(0,1) = 1 left + 1 right
		tabRegs = append(tabRegs, tabRegion{xStart: x, xEnd: x + rendered, period: t.period, isCategory: t.isCategory})

		isActive := false
		if t.isCategory {
			isActive = m.categoryMode || m.categorySelectMode
		} else {
			isActive = !m.categoryMode && !m.categorySelectMode && t.period == m.period
		}

		if isActive {
			parts = append(parts, ActiveTabStyle.Render(t.label))
		} else {
			parts = append(parts, InactiveTabStyle.Render(t.label))
		}
		x += rendered
	}
	line1 := strings.Join(parts, "")
	lastTabBarRegions = tabRegs

	// In split pane mode, skip the date bar to maximize content space
	if m.categorySelectMode {
		lastDateBarRegions = nil
		return line1
	}

	// Line 2: date selector bar
	line2, regions := m.buildDateBar()
	lastDateBarRegions = regions

	return line1 + "\n" + line2
}

// lastTabBarRegions stores click regions from the last render (single-threaded TUI).
var lastTabBarRegions []tabRegion

// lastDateBarRegions stores click regions from the last render (single-threaded TUI).
var lastDateBarRegions []dateRegion

// buildDateBar builds the date selector bar and returns the rendered string and click regions.
func (m Model) buildDateBar() (string, []dateRegion) {
	if m.categorySelectMode {
		return m.buildCategorySelectDateBar()
	}
	if m.searchResults {
		return m.buildSearchPageBar()
	}
	if m.categoryMode {
		return m.buildCategoryDateBar()
	}

	switch m.period {
	case types.Daily:
		return m.buildDailyDateBar()
	case types.Weekly:
		return m.buildWeeklyDateBar()
	case types.Monthly:
		return m.buildMonthlyDateBar()
	default:
		return m.buildDailyDateBar()
	}
}

func (m Model) buildSearchPageBar() (string, []dateRegion) {
	var regions []dateRegion
	var b strings.Builder
	x := 0

	left := "◀ "
	b.WriteString(DateArrowStyle.Render(left))
	leftW := lipgloss.Width(left)
	if m.searchHasPrev && m.searchPage > 1 {
		regions = append(regions, dateRegion{xStart: x, xEnd: x + leftW, action: "search_prev"})
	}
	x += leftW

	page := m.searchPage
	if page <= 0 {
		page = 1
	}
	pages := m.searchPages
	label := fmt.Sprintf(" Search \"%s\" • Page %d ", m.searchQuery, page)
	if pages > 0 {
		label = fmt.Sprintf(" Search \"%s\" • Page %d/%d ", m.searchQuery, page, pages)
	}
	b.WriteString(DateItemActiveStyle.Render(label))
	x += lipgloss.Width(label)

	right := " ▶"
	b.WriteString(DateArrowStyle.Render(right))
	rightW := lipgloss.Width(right)
	if m.searchHasNext {
		regions = append(regions, dateRegion{xStart: x, xEnd: x + rightW, action: "search_next"})
	}

	return b.String(), regions
}

func (m Model) buildCategorySelectDateBar() (string, []dateRegion) {
	var b strings.Builder
	visible := m.catVisibleList()
	total := len(types.AllCategories)
	if m.catFilterMode || m.catFilterQuery != "" {
		b.WriteString(DateItemActiveStyle.Render(fmt.Sprintf(" Filter: %s ", m.catFilterQuery)))
		b.WriteString(DateItemStyle.Render(fmt.Sprintf(" (%d/%d) ", len(visible), total)))
	} else {
		b.WriteString(DateItemActiveStyle.Render(" Select a category "))
		b.WriteString(DateItemStyle.Render(fmt.Sprintf(" (%d categories, / to filter) ", total)))
	}
	return b.String(), nil
}

func (m Model) buildCategoryDateBar() (string, []dateRegion) {
	var regions []dateRegion
	var b strings.Builder
	x := 0
	all := types.AllCategories

	// Left arrow — navigate to previous category
	left := "◀ "
	b.WriteString(DateArrowStyle.Render(left))
	leftW := lipgloss.Width(left)
	if len(all) > 0 {
		regions = append(regions, dateRegion{xStart: x, xEnd: x + leftW, action: "cat_prev"})
	}
	x += leftW

	// Find the current category index
	curIdx := types.CategoryIndexBySlug(m.categorySlug)
	if curIdx < 0 {
		curIdx = 0
	}

	// Calculate how many neighbor categories we can show within terminal width
	// Reserve space for arrows (4 chars) and current category name
	currentLabel := " " + m.categoryName + " "
	currentLabelW := lipgloss.Width(currentLabel)
	arrowSpace := 4 // "◀ " + " ▶"
	availableWidth := m.width - arrowSpace - currentLabelW

	// Collect neighbors to show (categories before and after current)
	type neighborCat struct {
		idx  int
		cat  types.CategoryLink
		side string // "before" or "after"
	}
	var beforeCats, afterCats []neighborCat

	// Gather categories after current
	for i := 1; i < len(all); i++ {
		idx := (curIdx + i) % len(all)
		afterCats = append(afterCats, neighborCat{idx: idx, cat: all[idx], side: "after"})
	}

	// Interleave: show after-cats that fit in the available width
	var visibleAfter []neighborCat
	usedWidth := 0
	for _, nc := range afterCats {
		catLabel := " " + nc.cat.Name() + " "
		catW := lipgloss.Width(catLabel)
		if usedWidth+catW > availableWidth {
			break
		}
		visibleAfter = append(visibleAfter, nc)
		usedWidth += catW
	}
	_ = beforeCats // not used currently — showing only forward neighbors

	// Render current category (active)
	b.WriteString(DateItemActiveStyle.Render(currentLabel))
	x += currentLabelW

	// Render visible neighbor categories
	for _, nc := range visibleAfter {
		catLabel := " " + nc.cat.Name() + " "
		b.WriteString(DateItemStyle.Render(catLabel))
		catWidth := lipgloss.Width(catLabel)
		regions = append(regions, dateRegion{xStart: x, xEnd: x + catWidth, action: "cat_goto:" + nc.cat.Slug()})
		x += catWidth
	}

	// Right arrow — navigate to next category
	right := " ▶"
	b.WriteString(DateArrowStyle.Render(right))
	rightW := lipgloss.Width(right)
	if len(all) > 0 {
		regions = append(regions, dateRegion{xStart: x, xEnd: x + rightW, action: "cat_next"})
	}

	return b.String(), regions
}

func (m Model) buildDailyDateBar() (string, []dateRegion) {
	var regions []dateRegion
	var b strings.Builder
	x := 0

	// Left arrow — navigates to previous month
	arrow := "◀ "
	b.WriteString(DateArrowStyle.Render(arrow))
	aw := lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "prev_month"})
	x += aw

	year, month, _ := m.date.Date()

	// Month/year label so user knows which month they're viewing
	monthLabel := fmt.Sprintf("%s %d ", m.date.Month().String()[:3], year)
	b.WriteString(DateItemStyle.Render(monthLabel))
	x += lipgloss.Width(monthLabel)
	loc := m.date.Location()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	today := time.Now()
	currentDay := m.date.Day()

	for d := 1; d <= daysInMonth; d++ {
		label := fmt.Sprintf("%d", d)
		padded := " " + label + " "
		targetDate := time.Date(year, month, d, 0, 0, 0, 0, loc)
		isFuture := targetDate.After(today)

		var styled string
		if d == currentDay {
			styled = DateItemActiveStyle.Render(padded)
		} else if isFuture {
			styled = DateItemDimStyle.Render(padded)
		} else {
			styled = DateItemStyle.Render(padded)
		}
		b.WriteString(styled)

		cellWidth := lipgloss.Width(padded)
		if !isFuture {
			regions = append(regions, dateRegion{xStart: x, xEnd: x + cellWidth, action: "goto", date: targetDate})
		}
		x += cellWidth
	}

	// Right arrow — navigates to next month
	arrow = " ▶"
	b.WriteString(DateArrowStyle.Render(arrow))
	aw = lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "next_month"})

	return b.String(), regions
}

func (m Model) buildWeeklyDateBar() (string, []dateRegion) {
	var regions []dateRegion
	var b strings.Builder
	x := 0

	arrow := "◀ "
	b.WriteString(DateArrowStyle.Render(arrow))
	aw := lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "prev_month"})
	x += aw

	year, month, _ := m.date.Date()
	loc := m.date.Location()
	_, currentWeek := m.date.ISOWeek()

	// Find weeks that overlap with this month
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	lastOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, loc)

	// Start from the Monday of the week containing the first of the month
	weekStart := firstOfMonth
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	today := time.Now()
	for ws := weekStart; !ws.After(lastOfMonth); ws = ws.AddDate(0, 0, 7) {
		we := ws.AddDate(0, 0, 6) // week end (Sunday)
		_, thisWeek := ws.ISOWeek()

		// Format: "M/D-D" or "M/D-M/D" if crossing month boundary
		var label string
		if ws.Month() == we.Month() {
			label = fmt.Sprintf("%d/%d-%d", int(ws.Month()), ws.Day(), we.Day())
		} else {
			label = fmt.Sprintf("%d/%d-%d/%d", int(ws.Month()), ws.Day(), int(we.Month()), we.Day())
		}
		padded := " " + label + " "

		isFuture := ws.After(today)
		var styled string
		if thisWeek == currentWeek {
			styled = DateItemActiveStyle.Render(padded)
		} else if isFuture {
			styled = DateItemDimStyle.Render(padded)
		} else {
			styled = DateItemStyle.Render(padded)
		}
		b.WriteString(styled)

		cellWidth := lipgloss.Width(padded)
		if !isFuture {
			regions = append(regions, dateRegion{xStart: x, xEnd: x + cellWidth, action: "goto", date: ws})
		}
		x += cellWidth
	}

	arrow = " ▶"
	b.WriteString(DateArrowStyle.Render(arrow))
	aw = lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "next_month"})

	return b.String(), regions
}

func (m Model) buildMonthlyDateBar() (string, []dateRegion) {
	var regions []dateRegion
	var b strings.Builder
	x := 0

	arrow := "◀ "
	b.WriteString(DateArrowStyle.Render(arrow))
	aw := lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "prev_month"})
	x += aw

	label := " " + m.date.Format("January 2006") + " "
	b.WriteString(DateItemActiveStyle.Render(label))
	x += lipgloss.Width(label)

	arrow = " ▶"
	b.WriteString(DateArrowStyle.Render(arrow))
	aw = lipgloss.Width(arrow)
	regions = append(regions, dateRegion{xStart: x, xEnd: x + aw, action: "next_month"})

	return b.String(), regions
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

func (m Model) handleDateBarClick(r dateRegion) (tea.Model, tea.Cmd) {
	if r.action == "search_prev" || r.action == "search_next" {
		if !m.searchResults || m.source == nil {
			return m, nil
		}
		targetPage := m.searchPage
		if targetPage <= 0 {
			targetPage = 1
		}
		if r.action == "search_prev" {
			if !m.searchHasPrev || targetPage <= 1 {
				return m, nil
			}
			targetPage--
		} else {
			if !m.searchHasNext {
				return m, nil
			}
			targetPage++
		}
		m.loading = true
		m.statusMsg = "Loading search page..."
		m.requestID++
		return m, tea.Batch(m.spinner.Tick, fetchSearchResults(m.source, m.searchQuery, targetPage, m.requestID))
	}

	// Handle category navigation actions
	if strings.HasPrefix(r.action, "cat_") {
		if m.source == nil {
			return m, nil
		}
		all := types.AllCategories
		var slug string
		switch {
		case r.action == "cat_prev":
			if len(all) == 0 {
				return m, nil
			}
			idx := types.CategoryIndexBySlug(m.categorySlug)
			if idx < 0 {
				idx = 0
			}
			idx--
			if idx < 0 {
				idx = len(all) - 1
			}
			slug = all[idx].Slug()
		case r.action == "cat_next":
			if len(all) == 0 {
				return m, nil
			}
			idx := types.CategoryIndexBySlug(m.categorySlug)
			if idx < 0 {
				idx = 0
			}
			idx++
			if idx >= len(all) {
				idx = 0
			}
			slug = all[idx].Slug()
		case strings.HasPrefix(r.action, "cat_goto:"):
			slug = strings.TrimPrefix(r.action, "cat_goto:")
		}
		if slug == "" {
			return m, nil
		}
		m.loading = true
		m.statusMsg = "Loading category..."
		m.requestID++
		return m, tea.Batch(m.spinner.Tick, fetchCategoryProducts(m.source, slug, m.requestID))
	}

	switch r.action {
	case "prev_month":
		m.date = m.date.AddDate(0, -1, 0)
	case "next_month":
		next := m.date.AddDate(0, 1, 0)
		if next.After(time.Now()) {
			return m, nil
		}
		m.date = next
	case "goto":
		if r.date.After(time.Now()) {
			return m, nil
		}
		m.date = r.date
	default:
		return m, nil
	}

	m.state = ListView
	m.loading = true
	m.statusMsg = "Loading..."
	if m.source == nil {
		return m, nil
	}
	m.requestID++
	return m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))
}

func (m Model) searchStatus() string {
	if m.searchMode {
		return fmt.Sprintf("Search (global): %s", m.searchQuery)
	}
	if m.searchResults {
		page := m.searchPage
		if page <= 0 {
			page = 1
		}
		pages := m.searchPages
		if pages > 0 {
			return fmt.Sprintf("Search \"%s\" • page %d/%d • %d results", m.searchQuery, page, pages, len(m.products))
		}
		return fmt.Sprintf("Search \"%s\" • page %d • %d results", m.searchQuery, page, len(m.products))
	}
	return fmt.Sprintf("%d products", len(m.products))
}

func (m Model) selectedProduct() (types.Product, bool) {
	if len(m.products) == 0 {
		return types.Product{}, false
	}
	if m.selected < 0 || m.selected >= len(m.products) {
		return types.Product{}, false
	}
	return m.products[m.selected], true
}

func (m Model) renderProductList() string {
	available := m.height - 4 // tab + status + help
	if available < 1 {
		available = 1
	}

	itemHeight := 3
	visibleCount := available / itemHeight
	if visibleCount < 1 {
		visibleCount = 1
	}

	start := 0
	if m.selected >= visibleCount {
		start = m.selected - visibleCount + 1
	}
	end := start + visibleCount
	if end > len(m.products) {
		end = len(m.products)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(renderProductItem(m.products[i], i == m.selected, m.width))
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderProductItem(product types.Product, isSelected bool, width int) string {
	// Line 1: Rank + Name + Votes
	rankStr := fmt.Sprintf("#%-2d", product.Rank())
	nameStr := product.Name()
	voteDisplay := fmt.Sprintf("▲ %s", formatVoteCount(product.VoteCount()))

	rankWidth := lipgloss.Width(rankStr)
	voteWidth := lipgloss.Width(voteDisplay) + 1
	availableForName := width - rankWidth - voteWidth
	if availableForName <= 1 {
		availableForName = 0
	}
	nameStr = padOrTruncate(nameStr, availableForName)

	var line1 string
	if isSelected {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaCyan).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaPink).Bold(true)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen).Bold(true)
		line1 = lipgloss.JoinHorizontal(lipgloss.Left, rankStyle.Render(rankStr), nameStyle.Render(nameStr), voteStyle.Render(voteDisplay))
	} else {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaComment)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaCyan)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen)
		line1 = lipgloss.JoinHorizontal(lipgloss.Left, rankStyle.Render(rankStr), nameStyle.Render(nameStr), voteStyle.Render(voteDisplay))
	}

	// Line 2: Tagline
	tagline := product.Tagline()
	taglineIndent := "    "
	taglineAvailable := width - lipgloss.Width(taglineIndent)
	if taglineAvailable < 0 {
		taglineAvailable = 0
	}
	tagline = truncateToWidth(tagline, taglineAvailable)
	line2 := taglineIndent + lipgloss.NewStyle().Foreground(DraculaForeground).Render(tagline)

	// Line 3: Categories
	categoryStr := strings.Join(product.Categories(), " • ")
	categoryIndent := "    "
	categoryAvailable := width - lipgloss.Width(categoryIndent)
	if categoryAvailable < 0 {
		categoryAvailable = 0
	}
	categoryStr = truncateToWidth(categoryStr, categoryAvailable)
	line3 := categoryIndent + lipgloss.NewStyle().Foreground(DraculaComment).Render(categoryStr)

	output := line1 + "\n" + line2 + "\n" + line3
	if isSelected {
		return SelectedItemStyle.Render(output)
	}
	return output
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

	if !d.LaunchDate().IsZero() {
		b.WriteString(fmt.Sprintf("🚀 Launched: %s\n", d.LaunchDate().Format("January 2, 2006")))
	}

	if d.MakerName() != "" {
		maker := fmt.Sprintf("👤 Maker: %s", d.MakerName())
		if d.MakerProfileURL() != "" {
			maker += fmt.Sprintf(" (%s)", d.MakerProfileURL())
		}
		b.WriteString(maker + "\n")
	}

	if d.PricingInfo() != "" {
		b.WriteString(fmt.Sprintf("💰 %s\n", d.PricingInfo()))
	}

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

	if len(d.ProConTags()) > 0 {
		var pros, cons, others []string
		for _, tag := range d.ProConTags() {
			label := fmt.Sprintf("%s (%d)", tag.Name(), tag.Count())
			switch tag.TagType() {
			case "Positive":
				pros = append(pros, label)
			case "Negative":
				cons = append(cons, label)
			default:
				others = append(others, label)
			}
		}
		if len(pros) > 0 {
			b.WriteString("\n👍 Pros:\n")
			for _, p := range pros {
				b.WriteString("  + " + p + "\n")
			}
		}
		if len(cons) > 0 {
			b.WriteString("\n👎 Cons:\n")
			for _, c := range cons {
				b.WriteString("  - " + c + "\n")
			}
		}
		if len(others) > 0 {
			b.WriteString("\nℹ️ Other:\n")
			for _, o := range others {
				b.WriteString("  * " + o + "\n")
			}
		}
	}

	if len(d.Categories()) > 0 {
		catStyle := lipgloss.NewStyle().Foreground(DraculaCyan).Underline(true)
		b.WriteString("\nCategories: ")
		for i, cat := range d.Categories() {
			if i > 0 {
				b.WriteString(" • ")
			}
			b.WriteString(catStyle.Render(cat))
		}
		b.WriteString("  (press 4 to browse categories)")
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

	// Chrome: tab bar (1) + date bar (1) + status bar (1) + help (1) = 4
	chrome := 4
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

// truncateToWidth truncates a string to fit within maxWidth display columns,
// appending "…" if truncated. Uses rune-aware iteration to avoid cutting
// multibyte characters (Korean, Japanese, emoji, etc.) mid-sequence.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}
	// Need to truncate — reserve 1 column for "…"
	target := maxWidth - 1
	if target <= 0 {
		return "…"
	}
	var result strings.Builder
	currentWidth := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if currentWidth+rw > target {
			break
		}
		result.WriteRune(r)
		currentWidth += rw
	}
	result.WriteString("…")
	return result.String()
}

// padOrTruncate pads the string with spaces to exactly targetWidth display columns,
// or truncates with "…" if it exceeds targetWidth.
func padOrTruncate(s string, targetWidth int) string {
	w := lipgloss.Width(s)
	if w > targetWidth {
		return truncateToWidth(s, targetWidth)
	}
	if w < targetWidth {
		return s + strings.Repeat(" ", targetWidth-w)
	}
	return s
}

// switchToLeaderboard resets category/split-pane state and fetches the leaderboard for the given period.
func (m *Model) switchToLeaderboard(period types.Period) (tea.Model, tea.Cmd) {
	m.categoryMode = false
	m.categorySelectMode = false
	m.splitLoading = false
	m.splitRequestID = 0
	m.period = period
	m.state = ListView
	m.loading = true
	m.statusMsg = "Loading..."
	if m.source == nil {
		return *m, nil
	}
	m.requestID++
	return *m, tea.Batch(m.spinner.Tick, fetchLeaderboard(m.source, m.period, m.date, m.requestID))
}

// slugToDisplayName converts a category slug like "ai-agents" to "AI Agents".
func slugToDisplayName(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w == "ai" || w == "llm" || w == "llms" || w == "api" || w == "saas" {
			words[i] = strings.ToUpper(w)
		} else if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// enterCategorySelectMode switches to the split pane mode and returns a Cmd to load the initial category.
func (m *Model) enterCategorySelectMode() tea.Cmd {
	m.categorySelectMode = true
	m.catFilterMode = false
	m.catFilterQuery = ""
	m.catFilteredIndices = nil
	m.splitFocus = 0
	m.splitProducts = nil
	m.splitSelected = 0
	m.splitLoading = false
	m.splitSlug = ""
	m.splitRequestID = 0
	// If we were viewing a category, position cursor there
	if m.categorySlug != "" {
		idx := types.CategoryIndexBySlug(m.categorySlug)
		if idx >= 0 {
			m.catSelectIdx = idx
		}
	}
	m.statusMsg = fmt.Sprintf("Select a category (%d categories)", len(types.AllCategories))
	// Trigger initial load for the selected category
	return m.loadSelectedCategory()
}

// loadSelectedCategory triggers a fetch for the currently selected category in the left pane.
func (m *Model) loadSelectedCategory() tea.Cmd {
	visible := m.catVisibleList()
	if len(visible) == 0 || m.catSelectIdx >= len(visible) || m.source == nil {
		return nil
	}
	catIdx := visible[m.catSelectIdx]
	slug := types.AllCategories[catIdx].Slug()
	if slug == m.splitSlug {
		return nil // already loaded
	}
	m.splitLoading = true
	m.requestID++
	m.splitRequestID = m.requestID
	return tea.Batch(m.spinner.Tick, fetchCategoryProducts(m.source, slug, m.requestID))
}

// catVisibleList returns the list of AllCategories indices to show.
// allCategoryIndices is a pre-computed slice [0, 1, 2, ..., len(AllCategories)-1]
// to avoid allocating a new slice on every catVisibleList call.
var allCategoryIndices = func() []int {
	indices := make([]int, len(types.AllCategories))
	for i := range indices {
		indices[i] = i
	}
	return indices
}()

// catVisibleList returns the list of AllCategories indices to show.
// When filtering, returns only matching indices; otherwise all indices.
func (m Model) catVisibleList() []int {
	if m.catFilterMode || m.catFilterQuery != "" {
		if len(m.catFilteredIndices) > 0 {
			return m.catFilteredIndices
		}
		return nil
	}
	return allCategoryIndices
}

// updateCatFilter updates the filtered category indices based on the query.
func (m *Model) updateCatFilter() {
	if m.catFilterQuery == "" {
		m.catFilteredIndices = nil
		return
	}
	query := strings.ToLower(m.catFilterQuery)
	m.catFilteredIndices = nil
	for i, cat := range types.AllCategories {
		if strings.Contains(strings.ToLower(cat.Name()), query) || strings.Contains(cat.Slug(), query) {
			m.catFilteredIndices = append(m.catFilteredIndices, i)
		}
	}
	// Reset cursor if out of range
	if m.catSelectIdx >= len(m.catFilteredIndices) {
		m.catSelectIdx = 0
	}
}

// renderSplitPane renders the left (categories) + right (products) split layout.
func (m Model) renderSplitPane() string {
	available := m.height - 3 // tab + status + help (no date bar in split mode)
	if available < 1 {
		available = 1
	}

	// Calculate pane widths
	leftWidth := m.width * 30 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	if leftWidth > 35 {
		leftWidth = 35
	}
	sepWidth := 1 // "│"
	rightWidth := m.width - leftWidth - sepWidth
	if rightWidth < 30 {
		rightWidth = 30
	}

	leftContent := m.renderCategoryPane(leftWidth, available)
	rightContent := m.renderProductPane(rightWidth, available)

	// Join panes line by line
	leftLines := strings.Split(leftContent, "\n")
	rightLines := strings.Split(rightContent, "\n")

	sepStyle := lipgloss.NewStyle().Foreground(DraculaComment)

	var result strings.Builder
	for i := 0; i < available; i++ {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		// Pad left to leftWidth
		leftVisual := lipgloss.Width(left)
		if leftVisual < leftWidth {
			left += strings.Repeat(" ", leftWidth-leftVisual)
		}

		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}

		result.WriteString(left)
		result.WriteString(sepStyle.Render("│"))
		result.WriteString(right)
		if i < available-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderCategoryPane renders the left pane with the category list.
func (m Model) renderCategoryPane(width, height int) string {
	visible := m.catVisibleList()
	if len(visible) == 0 {
		emptyText := "No categories"
		if m.catFilterQuery != "" {
			emptyText = "No match"
		}
		msg := lipgloss.NewStyle().Foreground(DraculaComment).Render(emptyText)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
	}

	visibleCount := height
	if visibleCount < 1 {
		visibleCount = 1
	}

	sel := m.catSelectIdx
	if sel >= len(visible) {
		sel = len(visible) - 1
	}
	if sel < 0 {
		sel = 0
	}

	start := 0
	if sel >= visibleCount {
		start = sel - visibleCount + 1
	}
	end := start + visibleCount
	if end > len(visible) {
		end = len(visible)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	isLeftFocused := m.splitFocus == 0

	var b strings.Builder
	for i := start; i < end; i++ {
		catIdx := visible[i]
		cat := types.AllCategories[catIdx]
		isSelected := i == sel

		name := cat.Name()
		// Truncate name if too long for pane (display-width aware)
		maxName := width - 3 // padding/border space
		if maxName < 5 {
			maxName = 5
		}
		name = truncateToWidth(name, maxName)

		if isSelected && isLeftFocused {
			line := lipgloss.NewStyle().
				Foreground(DraculaPink).Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(DraculaPink).
				PaddingLeft(1).
				Render(name)
			b.WriteString(line)
		} else if isSelected {
			// Selected but not focused — dim highlight
			line := lipgloss.NewStyle().
				Foreground(DraculaPink).
				PaddingLeft(2).
				Render(name)
			b.WriteString(line)
		} else {
			line := lipgloss.NewStyle().
				Foreground(DraculaComment).
				PaddingLeft(2).
				Render(name)
			b.WriteString(line)
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderProductPane renders the right pane with the product list for the selected category.
func (m Model) renderProductPane(width, height int) string {
	if m.splitLoading {
		spin := m.spinner.View() + " Loading..."
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, spin)
	}

	if len(m.splitProducts) == 0 {
		emptyText := "Select a category"
		if m.splitSlug != "" {
			emptyText = "No products"
		}
		msg := lipgloss.NewStyle().Foreground(DraculaComment).Render(emptyText)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
	}

	isRightFocused := m.splitFocus == 1
	itemHeight := 3
	visibleCount := height / itemHeight
	if visibleCount < 1 {
		visibleCount = 1
	}

	sel := m.splitSelected
	if sel >= len(m.splitProducts) {
		sel = len(m.splitProducts) - 1
	}
	if sel < 0 {
		sel = 0
	}

	start := 0
	if sel >= visibleCount {
		start = sel - visibleCount + 1
	}
	end := start + visibleCount
	if end > len(m.splitProducts) {
		end = len(m.splitProducts)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		isSelected := i == sel && isRightFocused
		b.WriteString(renderProductItem(m.splitProducts[i], isSelected, width))
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
