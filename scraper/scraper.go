package scraper

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/qyinm/phtui/types"
)

const (
	baseURL   = "https://www.producthunt.com"
	userAgent = "phtui/1.0"
)

// Scraper implements types.ProductSource using HTTP client and in-memory cache.
type Scraper struct {
	client *http.Client
	cache  map[string]any
	mu     sync.Mutex
}

// Compile-time interface check
var _ types.ProductSource = (*Scraper)(nil)

// New creates a new Scraper with configured HTTP client and empty cache.
func New() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]any),
	}
}

// GetLeaderboard fetches and parses the Product Hunt leaderboard for the given period and date.
func (s *Scraper) GetLeaderboard(period types.Period, date time.Time) ([]types.Product, error) {
	url := baseURL + period.URLPath(date)

	// Check cache
	s.mu.Lock()
	if cached, ok := s.cache[url]; ok {
		s.mu.Unlock()
		return cached.([]types.Product), nil
	}
	s.mu.Unlock()

	// HTTP GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch leaderboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse
	products, err := ParseLeaderboard(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse leaderboard: %w", err)
	}

	// Cache result
	s.mu.Lock()
	s.cache[url] = products
	s.mu.Unlock()

	return products, nil
}

// GetProductDetail fetches and parses the Product Hunt product detail page for the given slug.
func (s *Scraper) GetProductDetail(slug string) (types.ProductDetail, error) {
	url := baseURL + "/products/" + slug

	// Check cache
	s.mu.Lock()
	if cached, ok := s.cache[url]; ok {
		s.mu.Unlock()
		return cached.(types.ProductDetail), nil
	}
	s.mu.Unlock()

	// HTTP GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return types.ProductDetail{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return types.ProductDetail{}, fmt.Errorf("fetch product detail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read body for error context
		body, _ := io.ReadAll(resp.Body)
		return types.ProductDetail{}, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse
	detail, err := ParseProductDetail(resp.Body)
	if err != nil {
		return types.ProductDetail{}, fmt.Errorf("parse product detail: %w", err)
	}

	// Cache result
	s.mu.Lock()
	s.cache[url] = detail
	s.mu.Unlock()

	return detail, nil
}

// ClearCache clears the in-memory cache.
func (s *Scraper) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]any)
}
