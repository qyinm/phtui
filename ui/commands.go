package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qyinm/phtui/types"
)

// Message types for async operations

type leaderboardMsg struct {
	requestID int
	products  []types.Product
	err       error
}

type productDetailMsg struct {
	requestID int
	detail    types.ProductDetail
	err       error
}

// fetchLeaderboard returns a tea.Cmd that fetches the leaderboard asynchronously
func fetchLeaderboard(source types.ProductSource, period types.Period, date time.Time, requestID int) tea.Cmd {
	return func() tea.Msg {
		products, err := source.GetLeaderboard(period, date)
		return leaderboardMsg{requestID: requestID, products: products, err: err}
	}
}

// fetchProductDetail returns a tea.Cmd that fetches product detail asynchronously
func fetchProductDetail(source types.ProductSource, slug string, requestID int) tea.Cmd {
	return func() tea.Msg {
		detail, err := source.GetProductDetail(slug)
		return productDetailMsg{requestID: requestID, detail: detail, err: err}
	}
}
