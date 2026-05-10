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
