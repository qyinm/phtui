package ui

import (
	"fmt"
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

type searchResultsMsg struct {
	requestID int
	query     string
	page      int
	hasPrev   bool
	hasNext   bool
	pages     int
	products  []types.Product
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

type searchableSource interface {
	SearchProductsPage(query string, page int) ([]types.Product, int, bool, bool, int, error)
}

func fetchSearchResults(source types.ProductSource, query string, page int, requestID int) tea.Cmd {
	return func() tea.Msg {
		searchable, ok := source.(searchableSource)
		if !ok {
			return searchResultsMsg{
				requestID: requestID,
				query:     query,
				page:      page,
				err:       fmt.Errorf("search not supported by source"),
			}
		}
		products, currentPage, hasPrev, hasNext, pagesCount, err := searchable.SearchProductsPage(query, page)
		return searchResultsMsg{
			requestID: requestID,
			query:     query,
			page:      currentPage,
			hasPrev:   hasPrev,
			hasNext:   hasNext,
			pages:     pagesCount,
			products:  products,
			err:       err,
		}
	}
}

type categoryProductsMsg struct {
	requestID  int
	slug       string
	products   []types.Product
	categories []types.CategoryLink
	err        error
}

func fetchCategoryProducts(source types.ProductSource, slug string, requestID int) tea.Cmd {
	return func() tea.Msg {
		products, categories, err := source.GetCategoryProducts(slug)
		return categoryProductsMsg{
			requestID:  requestID,
			slug:       slug,
			products:   products,
			categories: categories,
			err:        err,
		}
	}
}
