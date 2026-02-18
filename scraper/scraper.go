package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/qyinm/phtui/types"
)

const (
	baseURL        = "https://www.producthunt.com"
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	searchPageSize = 10
	maxSearchPages = 10
)

// Scraper implements types.ProductSource using HTTP client and in-memory cache.
type Scraper struct {
	client *http.Client
	cache  map[string]cachedResult
	mu     sync.Mutex
}

type cachedResult struct {
	value     any
	timestamp time.Time
}

// Compile-time interface check
var _ types.ProductSource = (*Scraper)(nil)

// New creates a new Scraper with configured HTTP client and empty cache.
func New() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]cachedResult),
	}
}

// GetLeaderboard fetches and parses the Product Hunt Featured leaderboard for the given period and date.
func (s *Scraper) GetLeaderboard(period types.Period, date time.Time) ([]types.Product, error) {
	url := baseURL + period.URLPath(date)

	s.mu.Lock()
	if cached, ok := s.cache[url]; ok {
		s.mu.Unlock()
		if products, ok := cached.value.([]types.Product); ok {
			return products, nil
		}
	}
	s.mu.Unlock()

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

	products, err := ParseLeaderboard(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse leaderboard: %w", err)
	}

	s.mu.Lock()
	s.cache[url] = cachedResult{value: products, timestamp: time.Now()}
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
		detail, ok := cached.value.(types.ProductDetail)
		if ok {
			return detail, nil
		}
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
	s.cache[url] = cachedResult{value: detail, timestamp: time.Now()}
	s.mu.Unlock()

	return detail, nil
}

// SearchProducts fetches Product Hunt global search results for the query.
func (s *Scraper) SearchProducts(query string) ([]types.Product, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	all := make([]types.Product, 0, searchPageSize)
	seen := make(map[string]struct{})

	for page := 1; page <= maxSearchPages; page++ {
		products, _, _, hasNext, _, err := s.SearchProductsPage(q, page)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if len(products) == 0 {
			break
		}

		added := 0
		for _, p := range products {
			if p.Slug() == "" {
				continue
			}
			if _, ok := seen[p.Slug()]; ok {
				continue
			}
			seen[p.Slug()] = struct{}{}
			all = append(all, types.NewProduct(
				p.Name(),
				p.Tagline(),
				p.Categories(),
				p.VoteCount(),
				p.CommentCount(),
				p.Slug(),
				p.ThumbnailURL(),
				len(all)+1,
			))
			added++
		}

		if added == 0 || len(products) < searchPageSize || !hasNext {
			break
		}
	}

	return all, nil
}

// SearchProductsPage fetches a single search results page and paging metadata.
func (s *Scraper) SearchProductsPage(query string, page int) ([]types.Product, int, bool, bool, int, error) {
	if page < 1 {
		page = 1
	}
	escaped := url.QueryEscape(query)
	searchURL := fmt.Sprintf("%s/search?q=%s&page=%d", baseURL, escaped, page)

	s.mu.Lock()
	if cached, ok := s.cache[searchURL]; ok {
		s.mu.Unlock()
		if searchCached, ok := cached.value.(searchPageCache); ok {
			return searchCached.products, searchCached.page, searchCached.hasPrev, searchCached.hasNext, searchCached.pagesCount, nil
		}
		if products, ok := cached.value.([]types.Product); ok {
			return products, page, page > 1, len(products) >= searchPageSize, page, nil
		}
	} else {
		s.mu.Unlock()
	}

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, page, false, false, page, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, page, false, false, page, fmt.Errorf("fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, page, false, false, page, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, page, false, false, page, fmt.Errorf("read search results: %w", err)
	}

	products, err := ParseSearchResults(strings.NewReader(string(body)))
	if err != nil {
		return nil, page, false, false, page, fmt.Errorf("parse search results: %w", err)
	}
	currentPage, hasPrev, hasNext, pagesCount, ok := parseSearchPageInfo(string(body))
	if !ok {
		currentPage = page
		hasPrev = page > 1
		hasNext = len(products) >= searchPageSize
		pagesCount = 0
	}

	s.mu.Lock()
	s.cache[searchURL] = cachedResult{
		value: searchPageCache{
			products:   products,
			page:       currentPage,
			hasPrev:    hasPrev,
			hasNext:    hasNext,
			pagesCount: pagesCount,
		},
		timestamp: time.Now(),
	}
	s.mu.Unlock()

	return products, currentPage, hasPrev, hasNext, pagesCount, nil
}

type searchPageCache struct {
	products   []types.Product
	page       int
	hasPrev    bool
	hasNext    bool
	pagesCount int
}

// ClearCache clears the in-memory cache.
func (s *Scraper) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]cachedResult)
}
