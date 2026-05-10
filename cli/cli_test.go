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
	return &fakeSource{
		leaderboard: []types.Product{product},
		detail: types.NewProductDetail(
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
		),
		catProducts: []types.Product{product},
		catLinks:    []types.CategoryLink{types.NewCategoryLink("AI Agents", "ai-agents")},
		search:      []types.Product{product},
	}
}

func (f *fakeSource) GetLeaderboard(period types.Period, date time.Time) ([]types.Product, error) {
	return f.leaderboard, nil
}

func (f *fakeSource) GetProductDetail(slug string) (types.ProductDetail, error) {
	if slug == "" {
		return types.ProductDetail{}, errors.New("empty slug")
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
	if len(got.Items) != 1 {
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
