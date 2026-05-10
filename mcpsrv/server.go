package mcpsrv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qyinm/phtui/mcpsrv/dto"
	"github.com/qyinm/phtui/types"
)

type leaderboardGetArgs struct {
	Period string `json:"period" jsonschema:"Leaderboard period: daily, weekly, monthly"`
	Date   string `json:"date,omitempty" jsonschema:"Optional date in YYYY-MM-DD"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Optional maximum number of items"`
}

type productGetDetailArgs struct {
	Slug string `json:"slug" jsonschema:"Product slug"`
}

type productCompareArgs struct {
	Slugs []string `json:"slugs" jsonschema:"Product slugs to compare"`
}

type categoryListArgs struct {
	Query  string `json:"query,omitempty" jsonschema:"Optional category search query"`
	Offset int    `json:"offset,omitempty" jsonschema:"Optional pagination offset"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Optional page size limit"`
}

type categoryGetProductsArgs struct {
	Slug  string `json:"slug" jsonschema:"Category slug"`
	Limit int    `json:"limit,omitempty" jsonschema:"Optional maximum number of products"`
}

type ideaInspirationsArgs struct {
	CategorySlug string `json:"category_slug,omitempty" jsonschema:"Optional Product Hunt category slug to use as the inspiration source"`
	Period       string `json:"period,omitempty" jsonschema:"Leaderboard period when category_slug is omitted: daily, weekly, monthly"`
	Date         string `json:"date,omitempty" jsonschema:"Optional leaderboard date in YYYY-MM-DD when category_slug is omitted"`
	Limit        int    `json:"limit,omitempty" jsonschema:"Optional maximum number of inspiration items"`
}

type searchProductsArgs struct {
	Query string `json:"query" jsonschema:"Search query"`
	Page  int    `json:"page,omitempty" jsonschema:"Page number (1-10)"`
}

type leaderboardGetOutput struct {
	Period string        `json:"period"`
	Date   string        `json:"date"`
	Total  int           `json:"total"`
	Items  []dto.Product `json:"items"`
}

type productGetDetailOutput struct {
	Item dto.ProductDetail `json:"item"`
}

type productCompareOutput struct {
	Items   []dto.CompareItem  `json:"items"`
	Summary dto.CompareSummary `json:"summary"`
}

type categoryListOutput struct {
	Query      string         `json:"query"`
	Offset     int            `json:"offset"`
	Limit      int            `json:"limit"`
	NextOffset int            `json:"next_offset"`
	HasMore    bool           `json:"has_more"`
	Total      int            `json:"total"`
	Items      []dto.Category `json:"items"`
}

type categoryGetProductsOutput struct {
	Slug       string         `json:"slug"`
	Total      int            `json:"total"`
	Categories []dto.Category `json:"categories"`
	Items      []dto.Product  `json:"items"`
}

type ideaInspirationItem struct {
	Product           dto.Product `json:"product"`
	InspirationReason string      `json:"inspiration_reason"`
	FeatureSignals    []string    `json:"feature_signals"`
	MarketSignals     []string    `json:"market_signals"`
}

type ideaInspirationsOutput struct {
	Source       string                `json:"source"`
	Period       string                `json:"period,omitempty"`
	Date         string                `json:"date,omitempty"`
	CategorySlug string                `json:"category_slug,omitempty"`
	Total        int                   `json:"total"`
	Items        []ideaInspirationItem `json:"items"`
}

type searchProductsOutput struct {
	Query      string        `json:"query"`
	Page       int           `json:"page"`
	HasPrev    bool          `json:"has_prev"`
	HasNext    bool          `json:"has_next"`
	PagesCount int           `json:"pages_count"`
	ItemsCount int           `json:"items_count"`
	Items      []dto.Product `json:"items"`
}

type cacheClearOutput struct {
	Status string `json:"status"`
}

type ServerOptions struct {
	EnableAdmin bool
}

type cacheClearSource interface {
	ClearCache()
}

func NewServer(source types.ProductSource, version string, opts *ServerOptions) *mcp.Server {
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	if opts == nil {
		opts = &ServerOptions{}
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "phtui", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "leaderboard_get",
		Description: "Get leaderboard products by period/date.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args leaderboardGetArgs) (*mcp.CallToolResult, leaderboardGetOutput, error) {
		return leaderboardGetHandler(ctx, req, args, source)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "product_get_detail",
		Description: "Get product details by slug.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args productGetDetailArgs) (*mcp.CallToolResult, productGetDetailOutput, error) {
		return productGetDetailHandler(ctx, req, args, source)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "product_compare",
		Description: "Compare multiple products side-by-side by slug.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args productCompareArgs) (*mcp.CallToolResult, productCompareOutput, error) {
		return productCompareHandler(ctx, req, args, source)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_list",
		Description: "List available product categories.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args categoryListArgs) (*mcp.CallToolResult, categoryListOutput, error) {
		return categoryListHandler(ctx, req, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_get_products",
		Description: "Get products for a category slug.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args categoryGetProductsArgs) (*mcp.CallToolResult, categoryGetProductsOutput, error) {
		return categoryGetProductsHandler(ctx, req, args, source)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "idea_inspirations",
		Description: "Get agent-ready inspiration items from a category or leaderboard.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ideaInspirationsArgs) (*mcp.CallToolResult, ideaInspirationsOutput, error) {
		return ideaInspirationsHandler(ctx, req, args, source)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_products",
		Description: "Search products by query.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args searchProductsArgs) (*mcp.CallToolResult, searchProductsOutput, error) {
		return searchProductsHandler(ctx, req, args, source)
	})

	if opts.EnableAdmin {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "cache_clear",
			Description: "Clear scraper cache (admin).",
		}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, cacheClearOutput, error) {
			return cacheClearHandler(ctx, req, source)
		})
	}

	return server
}

func leaderboardGetHandler(_ context.Context, _ *mcp.CallToolRequest, args leaderboardGetArgs, source types.ProductSource) (*mcp.CallToolResult, leaderboardGetOutput, error) {
	period, err := parsePeriod(args.Period)
	if err != nil {
		return errorToolResult(err.Error()), leaderboardGetOutput{}, nil
	}

	date, err := parseDate(args.Date)
	if err != nil {
		return errorToolResult(err.Error()), leaderboardGetOutput{}, nil
	}

	products, err := source.GetLeaderboard(period, date)
	if err != nil {
		return errorToolResult("fetch leaderboard failed"), leaderboardGetOutput{}, nil
	}

	products = applyLimit(products, args.Limit)

	return nil, leaderboardGetOutput{
		Period: period.String(),
		Date:   date.Format(time.DateOnly),
		Total:  len(products),
		Items:  dto.FromProducts(products),
	}, nil
}

func productGetDetailHandler(_ context.Context, _ *mcp.CallToolRequest, args productGetDetailArgs, source types.ProductSource) (*mcp.CallToolResult, productGetDetailOutput, error) {
	slug := strings.TrimSpace(args.Slug)
	if slug == "" {
		return errorToolResult("slug is required"), productGetDetailOutput{}, nil
	}

	detail, err := source.GetProductDetail(slug)
	if err != nil {
		return errorToolResult("fetch product detail failed"), productGetDetailOutput{}, nil
	}

	return nil, productGetDetailOutput{Item: dto.FromProductDetail(detail)}, nil
}

func productCompareHandler(_ context.Context, _ *mcp.CallToolRequest, args productCompareArgs, source types.ProductSource) (*mcp.CallToolResult, productCompareOutput, error) {
	if len(args.Slugs) < 2 {
		return errorToolResult("at least 2 slugs are required"), productCompareOutput{}, nil
	}

	details := make([]types.ProductDetail, 0, len(args.Slugs))
	var firstErr error
	for _, slug := range args.Slugs {
		s := strings.TrimSpace(slug)
		if s == "" {
			continue
		}
		detail, err := source.GetProductDetail(s)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		details = append(details, detail)
	}

	if len(details) < 2 {
		msg := "could not fetch enough product details for comparison"
		if firstErr != nil {
			msg = msg + ": " + firstErr.Error()
		}
		return errorToolResult(msg), productCompareOutput{}, nil
	}

	return nil, productCompareOutput(dto.FromCompareDetails(details)), nil
}

func categoryListHandler(_ context.Context, _ *mcp.CallToolRequest, args categoryListArgs) (*mcp.CallToolResult, categoryListOutput, error) {
	query := strings.TrimSpace(strings.ToLower(args.Query))
	all := types.AllCategories
	filtered := make([]types.CategoryLink, 0, len(all))
	for _, c := range all {
		if query == "" {
			filtered = append(filtered, c)
			continue
		}
		if strings.Contains(strings.ToLower(c.Name()), query) || strings.Contains(strings.ToLower(c.Slug()), query) {
			filtered = append(filtered, c)
		}
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := args.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[offset:end]
	nextOffset := end
	hasMore := end < len(filtered)
	if !hasMore {
		nextOffset = -1
	}

	return nil, categoryListOutput{
		Query:      args.Query,
		Offset:     offset,
		Limit:      limit,
		NextOffset: nextOffset,
		HasMore:    hasMore,
		Total:      len(filtered),
		Items:      dto.FromCategories(page),
	}, nil
}

func categoryGetProductsHandler(_ context.Context, _ *mcp.CallToolRequest, args categoryGetProductsArgs, source types.ProductSource) (*mcp.CallToolResult, categoryGetProductsOutput, error) {
	slug := strings.TrimSpace(args.Slug)
	if slug == "" {
		return errorToolResult("slug is required"), categoryGetProductsOutput{}, nil
	}

	products, categories, err := source.GetCategoryProducts(slug)
	if err != nil {
		return errorToolResult("fetch category products failed"), categoryGetProductsOutput{}, nil
	}

	products = applyLimit(products, args.Limit)

	return nil, categoryGetProductsOutput{
		Slug:       slug,
		Total:      len(products),
		Categories: dto.FromCategories(categories),
		Items:      dto.FromProducts(products),
	}, nil
}

func ideaInspirationsHandler(_ context.Context, _ *mcp.CallToolRequest, args ideaInspirationsArgs, source types.ProductSource) (*mcp.CallToolResult, ideaInspirationsOutput, error) {
	limit := args.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 25 {
		limit = 25
	}

	categorySlug := strings.TrimSpace(args.CategorySlug)
	if categorySlug != "" {
		products, _, err := source.GetCategoryProducts(categorySlug)
		if err != nil {
			return errorToolResult("fetch category inspirations failed"), ideaInspirationsOutput{}, nil
		}
		products = applyLimit(products, limit)
		return nil, ideaInspirationsOutput{
			Source:       "category",
			CategorySlug: categorySlug,
			Total:        len(products),
			Items:        buildInspirationItems(products),
		}, nil
	}

	period, err := parsePeriod(args.Period)
	if err != nil {
		return errorToolResult(err.Error()), ideaInspirationsOutput{}, nil
	}
	date, err := parseDate(args.Date)
	if err != nil {
		return errorToolResult(err.Error()), ideaInspirationsOutput{}, nil
	}
	products, err := source.GetLeaderboard(period, date)
	if err != nil {
		return errorToolResult("fetch leaderboard inspirations failed"), ideaInspirationsOutput{}, nil
	}
	products = applyLimit(products, limit)
	return nil, ideaInspirationsOutput{
		Source: "leaderboard",
		Period: period.String(),
		Date:   date.Format(time.DateOnly),
		Total:  len(products),
		Items:  buildInspirationItems(products),
	}, nil
}

func buildInspirationItems(products []types.Product) []ideaInspirationItem {
	items := make([]ideaInspirationItem, 0, len(products))
	for _, p := range products {
		items = append(items, ideaInspirationItem{
			Product:           dto.FromProduct(p),
			InspirationReason: inspirationReason(p),
			FeatureSignals:    featureSignals(p),
			MarketSignals:     marketSignals(p),
		})
	}
	return items
}

func inspirationReason(p types.Product) string {
	if len(p.Categories()) > 0 {
		return fmt.Sprintf("Rank #%d in Product Hunt context with traction in %s; useful for feature and positioning inspiration.", p.Rank(), strings.Join(p.Categories(), ", "))
	}
	return fmt.Sprintf("Rank #%d Product Hunt launch; useful for feature and positioning inspiration.", p.Rank())
}

func featureSignals(p types.Product) []string {
	signals := make([]string, 0, 3)
	if p.Tagline() != "" {
		signals = append(signals, "Positioning: "+p.Tagline())
	}
	for _, c := range p.Categories() {
		if strings.TrimSpace(c) != "" {
			signals = append(signals, "Category pattern: "+c)
		}
	}
	if len(signals) == 0 {
		signals = append(signals, "Review product detail for feature patterns")
	}
	return signals
}

func marketSignals(p types.Product) []string {
	signals := []string{fmt.Sprintf("Product Hunt rank #%d", p.Rank())}
	if p.VoteCount() > 0 {
		signals = append(signals, fmt.Sprintf("%d votes", p.VoteCount()))
	}
	if p.CommentCount() > 0 {
		signals = append(signals, fmt.Sprintf("%d comments", p.CommentCount()))
	}
	return signals
}

func searchProductsHandler(_ context.Context, _ *mcp.CallToolRequest, args searchProductsArgs, source types.ProductSource) (*mcp.CallToolResult, searchProductsOutput, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return errorToolResult("query is required"), searchProductsOutput{}, nil
	}
	page := args.Page
	if page == 0 {
		page = 1
	}
	if page < 1 || page > 10 {
		return errorToolResult("page must be between 1 and 10"), searchProductsOutput{}, nil
	}

	products, currentPage, hasPrev, hasNext, pagesCount, err := source.SearchProducts(query, page)
	if err != nil {
		msg := "search failed"
		if strings.Contains(strings.ToLower(err.Error()), "cloudflare") {
			msg = "search blocked by Cloudflare challenge; retryable=false"
		}
		return errorToolResult(msg), searchProductsOutput{}, nil
	}

	return nil, searchProductsOutput{
		Query:      query,
		Page:       currentPage,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		PagesCount: pagesCount,
		ItemsCount: len(products),
		Items:      dto.FromProducts(products),
	}, nil
}

func cacheClearHandler(_ context.Context, _ *mcp.CallToolRequest, source types.ProductSource) (*mcp.CallToolResult, cacheClearOutput, error) {
	clearable, ok := source.(cacheClearSource)
	if !ok {
		return errorToolResult("cache clear is not supported by this source"), cacheClearOutput{}, nil
	}
	clearable.ClearCache()
	return nil, cacheClearOutput{Status: "ok"}, nil
}

func errorToolResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func applyLimit(items []types.Product, limit int) []types.Product {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}

func parsePeriod(raw string) (types.Period, error) {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "" {
		return types.Daily, nil
	}
	switch v {
	case "daily":
		return types.Daily, nil
	case "weekly":
		return types.Weekly, nil
	case "monthly":
		return types.Monthly, nil
	default:
		return types.Daily, fmt.Errorf("invalid period %q; expected daily|weekly|monthly", raw)
	}
}

func parseDate(raw string) (time.Time, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return time.Now(), nil
	}
	if d, err := time.Parse(time.DateOnly, v); err == nil {
		return d, nil
	}
	if ts, err := time.Parse(time.RFC3339, v); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q; expected YYYY-MM-DD or RFC3339", raw)
}
