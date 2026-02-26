package mcpsrv

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qyinm/phtui/types"
)

type fakeSource struct {
	leaderboard []types.Product
	detail      types.ProductDetail
	catProducts []types.Product
	catLinks    []types.CategoryLink
	search      []types.Product
	cleared     bool
	failLeader  bool
	failDetail  bool
	failCat     bool
	failSearch  bool
}

func newFakeSource() *fakeSource {
	product := types.NewProduct(
		"Demo Product",
		"Tagline",
		[]string{"AI Agents"},
		101,
		5,
		"demo-product",
		"https://img.example/demo.png",
		1,
	)
	detail := types.NewProductDetail(
		product,
		"Description",
		4.5,
		8,
		20,
		"Maker comment",
		"https://demo.example",
		[]string{"AI Agents"},
		[]string{"https://x.com/demo"},
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		"Maker",
		"https://producthunt.com/@maker",
		nil,
		"$9/month",
	)
	return &fakeSource{
		leaderboard: []types.Product{product},
		detail:      detail,
		catProducts: []types.Product{product},
		catLinks: []types.CategoryLink{
			types.NewCategoryLink("AI Agents", "ai-agents"),
			types.NewCategoryLink("Developer Tools", "developer-tools"),
		},
		search: []types.Product{product},
	}
}

func (f *fakeSource) GetLeaderboard(period types.Period, date time.Time) ([]types.Product, error) {
	if f.failLeader {
		return nil, errors.New("upstream leaderboard error")
	}
	return f.leaderboard, nil
}

func (f *fakeSource) GetProductDetail(slug string) (types.ProductDetail, error) {
	if f.failDetail {
		return types.ProductDetail{}, errors.New("upstream detail error")
	}
	return f.detail, nil
}

func (f *fakeSource) GetCategoryProducts(slug string) ([]types.Product, []types.CategoryLink, error) {
	if f.failCat {
		return nil, nil, errors.New("upstream category error")
	}
	return f.catProducts, f.catLinks, nil
}

func (f *fakeSource) SearchProductsPage(query string, page int) ([]types.Product, int, bool, bool, int, error) {
	if f.failSearch {
		return nil, page, false, false, 0, errors.New("upstream search error")
	}
	return f.search, page, page > 1, false, 1, nil
}

func (f *fakeSource) ClearCache() {
	f.cleared = true
}

func TestToolLeaderboardInvalidPeriod(t *testing.T) {
	result, _, err := leaderboardGetHandler(context.Background(), nil, leaderboardGetArgs{Period: "bad"}, newFakeSource())
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError result for invalid period")
	}
}

func TestToolCategoryListPaging(t *testing.T) {
	_, out, err := categoryListHandler(context.Background(), nil, categoryListArgs{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != len(types.AllCategories) {
		t.Fatalf("unexpected total: got %d want %d", out.Total, len(types.AllCategories))
	}
	if len(out.Items) != 10 {
		t.Fatalf("unexpected items len: %d", len(out.Items))
	}
	if out.NextOffset != 10 {
		t.Fatalf("unexpected next offset: %d", out.NextOffset)
	}
}

func TestSearchToolEmptyQuery(t *testing.T) {
	result, _, err := searchProductsHandler(context.Background(), nil, searchProductsArgs{Query: "  "}, newFakeSource())
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError for empty query")
	}
}

func TestToolUpstreamFailuresIsError(t *testing.T) {
	f1 := newFakeSource()
	f1.failLeader = true
	r1, _, _ := leaderboardGetHandler(context.Background(), nil, leaderboardGetArgs{Period: "daily"}, f1)
	if r1 == nil || !r1.IsError {
		t.Fatalf("leaderboard failure must return IsError")
	}

	f2 := newFakeSource()
	f2.failDetail = true
	r2, _, _ := productGetDetailHandler(context.Background(), nil, productGetDetailArgs{Slug: "demo-product"}, f2)
	if r2 == nil || !r2.IsError {
		t.Fatalf("detail failure must return IsError")
	}

	f3 := newFakeSource()
	f3.failCat = true
	r3, _, _ := categoryGetProductsHandler(context.Background(), nil, categoryGetProductsArgs{Slug: "ai-agents"}, f3)
	if r3 == nil || !r3.IsError {
		t.Fatalf("category failure must return IsError")
	}

	f4 := newFakeSource()
	f4.failSearch = true
	r4, _, _ := searchProductsHandler(context.Background(), nil, searchProductsArgs{Query: "demo", Page: 1}, f4)
	if r4 == nil || !r4.IsError {
		t.Fatalf("search failure must return IsError")
	}
}

func TestSearchToolGating(t *testing.T) {
	ctx := context.Background()
	srvWithout := startTestServer(newFakeSource(), Config{}, &ServerOptions{EnableSearch: false})
	defer srvWithout.Close()

	sessionWithout := connectTestClient(t, ctx, srvWithout.URL+"/mcp")
	toolsWithout, err := sessionWithout.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools (without search): %v", err)
	}
	sessionWithout.Close()
	if containsTool(toolsWithout.Tools, "search_products") {
		t.Fatalf("search_products should be absent when disabled")
	}

	srvWith := startTestServer(newFakeSource(), Config{}, &ServerOptions{EnableSearch: true})
	defer srvWith.Close()
	sessionWith := connectTestClient(t, ctx, srvWith.URL+"/mcp")
	toolsWith, err := sessionWith.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools (with search): %v", err)
	}
	sessionWith.Close()
	if !containsTool(toolsWith.Tools, "search_products") {
		t.Fatalf("search_products should be present when enabled")
	}
}

func TestAdminCacheClearGating(t *testing.T) {
	ctx := context.Background()
	src := newFakeSource()

	srvWithout := startTestServer(src, Config{}, &ServerOptions{EnableAdmin: false})
	defer srvWithout.Close()
	sessionWithout := connectTestClient(t, ctx, srvWithout.URL+"/mcp")
	toolsWithout, err := sessionWithout.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools without admin: %v", err)
	}
	sessionWithout.Close()
	if containsTool(toolsWithout.Tools, "cache_clear") {
		t.Fatalf("cache_clear should be absent when admin disabled")
	}

	srvWith := startTestServer(src, Config{}, &ServerOptions{EnableAdmin: true, APIKey: "secret"})
	defer srvWith.Close()
	sessionWith := connectTestClient(t, ctx, srvWith.URL+"/mcp")
	toolsWith, err := sessionWith.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools with admin: %v", err)
	}
	sessionWith.Close()
	if !containsTool(toolsWith.Tools, "cache_clear") {
		t.Fatalf("cache_clear should be present when admin enabled")
	}
}

func TestAdminCacheClearCallsSource(t *testing.T) {
	ctx := context.Background()
	src := newFakeSource()
	srv := startTestServer(src, Config{}, &ServerOptions{EnableAdmin: true, APIKey: "secret"})
	defer srv.Close()

	session := connectTestClient(t, ctx, srv.URL+"/mcp")
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "cache_clear", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call cache_clear: %v", err)
	}
	if result.IsError {
		t.Fatalf("cache_clear returned tool error")
	}
	if !src.cleared {
		t.Fatalf("expected source.ClearCache to be called")
	}
}

func TestAuthMiddleware(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{APIKey: "secret", RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	resp, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddlewareSuccess(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{APIKey: "secret", RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	headers := map[string]string{"Authorization": "Bearer secret"}
	resp, err := postInitialize(srv.URL+"/mcp", headers)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthMiddlewareXAPIKeySuccess(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{APIKey: "secret", RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	headers := map[string]string{"X-API-Key": "secret"}
	resp, err := postInitialize(srv.URL+"/mcp", headers)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthMiddlewareMalformedBearer(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{APIKey: "secret", RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	headers := map[string]string{"Authorization": "Bearer"}
	resp, err := postInitialize(srv.URL+"/mcp", headers)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestOriginAllowlistMiddleware(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	headers := map[string]string{"Origin": "https://evil.example"}
	resp, err := postInitialize(srv.URL+"/mcp", headers)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestOriginAllowlistMiddlewareAllowed(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{AllowedOrigins: []string{"https://app.example"}, RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	headers := map[string]string{"Origin": "https://app.example"}
	resp, err := postInitialize(srv.URL+"/mcp", headers)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestOriginAllowlistPreflight(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{AllowedOrigins: []string{"https://app.example"}, RPS: 100, Burst: 100}, &ServerOptions{})
	defer srv.Close()

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{RPS: 1, Burst: 1}, &ServerOptions{})
	defer srv.Close()

	resp1, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", resp1.StatusCode)
	}

	resp2, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", resp2.StatusCode)
	}
}

func TestRateLimitRefill(t *testing.T) {
	srv := startTestServer(newFakeSource(), Config{RPS: 20, Burst: 1}, &ServerOptions{})
	defer srv.Close()

	resp1, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp1.Body.Close()

	resp2, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", resp2.StatusCode)
	}

	time.Sleep(60 * time.Millisecond)
	resp3, err := postInitialize(srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected third request 200 after refill, got %d", resp3.StatusCode)
	}
}

func TestStatelessGetMethod(t *testing.T) {
	handler := NewHandler(NewServer(newFakeSource(), "dev", &ServerOptions{}), StreamableOptions(Config{Stateless: true}))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestMCPListTools(t *testing.T) {
	ctx := context.Background()
	srv := startTestServer(newFakeSource(), Config{}, &ServerOptions{})
	defer srv.Close()

	session := connectTestClient(t, ctx, srv.URL+"/mcp")
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	for _, name := range []string{"leaderboard_get", "product_get_detail", "category_list", "category_get_products"} {
		if !containsTool(tools.Tools, name) {
			t.Fatalf("missing tool %q", name)
		}
	}
}

func TestMCPCoreTools(t *testing.T) {
	ctx := context.Background()
	srv := startTestServer(newFakeSource(), Config{}, &ServerOptions{})
	defer srv.Close()

	session := connectTestClient(t, ctx, srv.URL+"/mcp")
	defer session.Close()

	cases := []mcp.CallToolParams{
		{Name: "leaderboard_get", Arguments: map[string]any{"period": "daily"}},
		{Name: "product_get_detail", Arguments: map[string]any{"slug": "demo-product"}},
		{Name: "category_list", Arguments: map[string]any{"offset": 0, "limit": 5}},
		{Name: "category_get_products", Arguments: map[string]any{"slug": "ai-agents"}},
	}

	for _, tc := range cases {
		result, err := session.CallTool(ctx, &tc)
		if err != nil {
			t.Fatalf("call tool %s failed: %v", tc.Name, err)
		}
		if result.IsError {
			t.Fatalf("tool %s returned IsError=true", tc.Name)
		}
	}
}

func TestSearchToolSuccess(t *testing.T) {
	ctx := context.Background()
	srv := startTestServer(newFakeSource(), Config{}, &ServerOptions{EnableSearch: true})
	defer srv.Close()

	session := connectTestClient(t, ctx, srv.URL+"/mcp")
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search_products", Arguments: map[string]any{"query": "demo", "page": 1}})
	if err != nil {
		t.Fatalf("search tool call failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("search tool returned IsError=true")
	}
}

func startTestServer(source types.ProductSource, cfg Config, opts *ServerOptions) *httptest.Server {
	if cfg.RPS <= 0 {
		cfg.RPS = 100
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 100
	}
	server := NewServer(source, "test", opts)
	mux := http.NewServeMux()
	mux.Handle("/mcp", WrapMCPHandler(NewHandler(server, StreamableOptions(cfg)), cfg))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return httptest.NewServer(mux)
}

func connectTestClient(t *testing.T, ctx context.Context, endpoint string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	return session
}

func containsTool(tools []*mcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool != nil && tool.Name == name {
			return true
		}
	}
	return false
}

func postInitialize(url string, headers map[string]string) (*http.Response, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test",
				"version": "1",
			},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return http.DefaultClient.Do(req)
}
