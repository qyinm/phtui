package scraper

import (
	"os"
	"strings"
	"testing"
)

func TestParseCategoryProducts(t *testing.T) {
	f, err := os.Open("../testdata/category_products.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	products, categories, err := ParseCategoryProducts(f)
	if err != nil {
		t.Fatalf("ParseCategoryProducts: %v", err)
	}

	// Should find products
	if len(products) == 0 {
		t.Fatal("no products found")
	}

	// Should find at least 10 products (fixture has 18 main cards)
	if len(products) < 10 {
		t.Errorf("products count = %d, want >= 10", len(products))
	}

	// First product should be ElevenLabs (top of the page)
	first := products[0]
	if first.Name() != "ElevenLabs" {
		t.Errorf("first product name = %q, want %q", first.Name(), "ElevenLabs")
	}
	if first.Slug() != "elevenlabs" {
		t.Errorf("first product slug = %q, want %q", first.Slug(), "elevenlabs")
	}

	// Each product should have name and slug
	for i, p := range products {
		if p.Name() == "" {
			t.Errorf("product[%d] has empty name", i)
		}
		if p.Slug() == "" {
			t.Errorf("product[%d] has empty slug", i)
		}
		if p.Rank() != i+1 {
			t.Errorf("product[%d] rank = %d, want %d", i, p.Rank(), i+1)
		}
	}

	// Check that known products appear
	slugs := make(map[string]bool)
	for _, p := range products {
		slugs[p.Slug()] = true
	}
	wantSlugs := []string{"elevenlabs", "intercom", "deepgram", "zapier"}
	for _, s := range wantSlugs {
		if !slugs[s] {
			t.Errorf("expected product slug %q not found", s)
		}
	}

	// Should find related categories
	if len(categories) == 0 {
		t.Fatal("no related categories found")
	}

	// Each category should have name and slug
	for i, c := range categories {
		if c.Name() == "" {
			t.Errorf("category[%d] has empty name", i)
		}
		if c.Slug() == "" {
			t.Errorf("category[%d] has empty slug", i)
		}
	}

	// Should NOT include ai-agents (the current category) in related
	for _, c := range categories {
		if c.Slug() == "ai-agents" {
			t.Error("related categories should not include the current category (ai-agents)")
		}
	}

	// Should find known related categories
	catSlugs := make(map[string]bool)
	for _, c := range categories {
		catSlugs[c.Slug()] = true
	}
	// These are visible in the fixture's header nav
	wantCats := []string{"productivity", "llms"}
	for _, s := range wantCats {
		if !catSlugs[s] {
			t.Errorf("expected related category slug %q not found", s)
		}
	}
}

func TestParseCategoryProductsTaglines(t *testing.T) {
	f, err := os.Open("../testdata/category_products.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	products, _, err := ParseCategoryProducts(f)
	if err != nil {
		t.Fatalf("ParseCategoryProducts: %v", err)
	}

	// Most products should have a tagline
	withTagline := 0
	for _, p := range products {
		if p.Tagline() != "" {
			withTagline++
		}
	}
	if withTagline == 0 {
		t.Error("no products have a tagline")
	}

	// ElevenLabs should have a specific tagline
	for _, p := range products {
		if p.Slug() == "elevenlabs" {
			if !strings.Contains(p.Tagline(), "AI voices") {
				t.Errorf("ElevenLabs tagline = %q, expected to contain 'AI voices'", p.Tagline())
			}
		}
	}
}

func TestParseCategoryProductsNoDuplicates(t *testing.T) {
	f, err := os.Open("../testdata/category_products.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	products, _, err := ParseCategoryProducts(f)
	if err != nil {
		t.Fatalf("ParseCategoryProducts: %v", err)
	}

	seen := make(map[string]int)
	for _, p := range products {
		seen[p.Slug()]++
		if seen[p.Slug()] > 1 {
			t.Errorf("duplicate product slug: %q", p.Slug())
		}
	}
}

func TestParseCategoryProductsMinimalHTML(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Test</title></head><body><div></div></body></html>`
	products, categories, err := ParseCategoryProducts(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseCategoryProducts on minimal HTML: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("products count = %d, want 0", len(products))
	}
	if len(categories) != 0 {
		t.Errorf("categories count = %d, want 0", len(categories))
	}
}

func TestParseCategoryProductsSyntheticHTML(t *testing.T) {
	html := `<!DOCTYPE html><html><head>
	<link rel="canonical" href="https://www.producthunt.com/categories/test-cat">
	</head><body>
	<a data-grid-span="1" href="/products/my-product">
		<span class="font-semibold text-primary text-16">My Product</span>
		<span class="text-secondary font-normal text-14">A great tagline</span>
	</a>
	<a data-grid-span="1" href="/products/another-one">
		<span class="font-semibold text-primary text-16">Another One</span>
		<span class="text-secondary font-normal text-14">Another tagline</span>
	</a>
	<a href="/categories/related-cat">Related Category</a>
	<a href="/categories/test-cat">Test Cat</a>
	<a href="/categories/another-cat?page=2#content">Paginated</a>
	</body></html>`

	products, categories, err := ParseCategoryProducts(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseCategoryProducts: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("products count = %d, want 2", len(products))
	}
	if products[0].Name() != "My Product" {
		t.Errorf("product[0] name = %q, want %q", products[0].Name(), "My Product")
	}
	if products[0].Tagline() != "A great tagline" {
		t.Errorf("product[0] tagline = %q, want %q", products[0].Tagline(), "A great tagline")
	}
	if products[0].Slug() != "my-product" {
		t.Errorf("product[0] slug = %q, want %q", products[0].Slug(), "my-product")
	}
	if products[1].Name() != "Another One" {
		t.Errorf("product[1] name = %q, want %q", products[1].Name(), "Another One")
	}

	// Should only include "related-cat" (not current "test-cat", not paginated)
	if len(categories) != 1 {
		t.Fatalf("categories count = %d, want 1", len(categories))
	}
	if categories[0].Slug() != "related-cat" {
		t.Errorf("category[0] slug = %q, want %q", categories[0].Slug(), "related-cat")
	}
	if categories[0].Name() != "Related Category" {
		t.Errorf("category[0] name = %q, want %q", categories[0].Name(), "Related Category")
	}
}
