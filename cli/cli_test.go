package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qyinm/phtui/types"
)

type fakeSource struct {
	leaderboard []types.Product
	detail      types.ProductDetail
	catProducts []types.Product
	catLinks    []types.CategoryLink
	search      []types.Product
	slugDetail  map[string]types.ProductDetail
}

func newFakeSource() *fakeSource {
	product := types.NewProduct(
		"Demo Product",
		"AI product builder",
		[]string{"AI Agents"},
		123,
		9,
		"demo-product",
		"https://img.example/demo.png",
		1,
	)
	product2 := types.NewProduct(
		"Second Product",
		"Another builder",
		[]string{"AI Agents", "Developer Tools"},
		88,
		3,
		"second-product",
		"https://img.example/second.png",
		2,
	)
	detail1 := types.NewProductDetail(
		product,
		"Detailed description",
		4.7,
		11,
		22,
		"Maker note",
		"https://demo.example",
		[]string{"AI Agents"},
		[]string{"https://x.com/demo"},
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		"Maker",
		"https://producthunt.com/@maker",
		[]types.ProConTag{types.NewProConTag("Fast", "Positive", 3)},
		"$9/month",
	)
	detail2 := types.NewProductDetail(
		product2,
		"Second description",
		4.2,
		5,
		15,
		"Another note",
		"https://second.example",
		[]string{"AI Agents", "Developer Tools"},
		[]string{"https://x.com/second"},
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		"Maker Two",
		"https://producthunt.com/@maker2",
		[]types.ProConTag{types.NewProConTag("Cheap", "Positive", 5)},
		"Free",
	)
	return &fakeSource{
		leaderboard: []types.Product{product, product2},
		detail:      detail1,
		catProducts: []types.Product{product, product2},
		catLinks: []types.CategoryLink{
			types.NewCategoryLink("AI Agents", "ai-agents"),
			types.NewCategoryLink("Developer Tools", "developer-tools"),
		},
		search: []types.Product{product, product2},
		slugDetail: map[string]types.ProductDetail{
			"demo-product":   detail1,
			"second-product": detail2,
		},
	}
}

func (f *fakeSource) GetLeaderboard(period types.Period, date time.Time) ([]types.Product, error) {
	return f.leaderboard, nil
}

func (f *fakeSource) GetProductDetail(slug string) (types.ProductDetail, error) {
	if slug == "" {
		return types.ProductDetail{}, errors.New("empty slug")
	}
	if d, ok := f.slugDetail[slug]; ok {
		return d, nil
	}
	return f.detail, nil
}

func (f *fakeSource) GetCategoryProducts(slug string) ([]types.Product, []types.CategoryLink, error) {
	if slug == "" {
		return nil, nil, errors.New("empty category")
	}
	return f.catProducts, f.catLinks, nil
}

func (f *fakeSource) SearchProducts(query string, page int) ([]types.Product, int, bool, bool, int, error) {
	return f.search, page, page > 1, false, 1, nil
}

func TestIdeasCommandOutputsAgentFriendlyJSON(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"ideas", "--category", "ai-agents", "--limit", "1"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run ideas: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}
	if got["source"] != "category" {
		t.Fatalf("unexpected source: %v", got["source"])
	}
	if got["category_slug"] != "ai-agents" {
		t.Fatalf("unexpected category_slug: %v", got["category_slug"])
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items: %#v", got["items"])
	}
	item := items[0].(map[string]any)
	if item["inspiration_reason"] == "" {
		t.Fatalf("expected inspiration_reason")
	}
	if len(item["feature_signals"].([]any)) == 0 {
		t.Fatalf("expected feature_signals")
	}
}

func TestDetailCommandOutputsSignals(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"detail", "demo-product"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run detail: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got["slug"] != "demo-product" {
		t.Fatalf("unexpected slug: %v", got["slug"])
	}
	if got["positioning_hint"] == "" || got["monetization_signal"] == "" {
		t.Fatalf("expected agent signals, got %#v", got)
	}
}

func TestUnknownCommandReturnsUsage(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"unknown"}, &out, newFakeSource())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(out.String(), "phtui ideas") {
		t.Fatalf("expected usage, got %q", out.String())
	}
}

func TestSearchCommandOutputsJSON(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"search", "--query", "demo"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run search: %v", err)
	}

	var got searchOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}
	if got.Query != "demo" {
		t.Fatalf("unexpected query: %q", got.Query)
	}
	if got.Page != 1 {
		t.Fatalf("unexpected page: %d", got.Page)
	}
	if len(got.Items) != 2 {
		t.Fatalf("unexpected items: %d", len(got.Items))
	}
	if got.Items[0].Name != "Demo Product" {
		t.Fatalf("unexpected product name: %s", got.Items[0].Name)
	}
}

func TestSearchCommandEmptyQueryReturnsError(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"search", "--query", ""}, &out, newFakeSource())
	if err == nil {
		t.Fatalf("expected error for empty query")
	}
	if !strings.Contains(err.Error(), "--query is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchCommandPageParam(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"search", "--query", "demo", "--page", "2"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run search page 2: %v", err)
	}

	var got searchOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Page != 2 {
		t.Fatalf("expected page 2, got %d", got.Page)
	}
	if !got.HasPrev {
		t.Fatalf("expected has_prev=true for page 2")
	}
}

func TestCompareCommandOutputsJSON(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"compare", "demo-product", "second-product"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run compare: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, out.String())
	}
	items, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %T", got["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 compare items, got %d", len(items))
	}
	summary, ok := got["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary, got %T", got["summary"])
	}
	if summary["count"] != float64(2) {
		t.Fatalf("expected count 2, got %v", summary["count"])
	}
}

func TestCompareCommandTooFewSlugs(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"compare", "only-one"}, &out, newFakeSource())
	if err == nil {
		t.Fatalf("expected error for <2 slugs")
	}
	if !strings.Contains(err.Error(), "at least 2 product slugs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompareCommandSlugNotFound(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"compare", "demo-product", "nonexistent"}, &out, newFakeSource())
	if err != nil {
		t.Fatalf("run compare with one missing slug: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	items, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("expected items array")
	}
	// Should still return the one that succeeded (nonexistent falls back to fakeSource.detail = detail1)
	if len(items) != 2 {
		t.Fatalf("expected 2 items (one from slugDetail, one fallback), got %d", len(items))
	}
}
