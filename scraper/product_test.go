package scraper

import (
	"os"
	"strings"
	"testing"
)

func TestParseProductDetail(t *testing.T) {
	f, err := os.Open("../testdata/product_detail.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	detail, err := ParseProductDetail(f)
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	// Product name
	if got := detail.Product().Name(); got != "Tanka" {
		t.Errorf("Name = %q, want %q", got, "Tanka")
	}

	// Tagline should contain key terms
	tagline := detail.Product().Tagline()
	if tagline == "" {
		t.Error("Tagline is empty")
	}
	if !strings.Contains(tagline, "Smart Reply") {
		t.Errorf("Tagline = %q, expected to contain 'Smart Reply'", tagline)
	}

	// Rating > 0
	if got := detail.Rating(); got <= 0 {
		t.Errorf("Rating = %v, want > 0", got)
	}
	// Rating should be approximately 4.4
	if got := detail.Rating(); got < 4.0 || got > 5.0 {
		t.Errorf("Rating = %v, want between 4.0 and 5.0", got)
	}

	// Review count > 0
	if got := detail.ReviewCount(); got <= 0 {
		t.Errorf("ReviewCount = %d, want > 0", got)
	}

	// Follower count > 0
	if got := detail.FollowerCount(); got <= 0 {
		t.Errorf("FollowerCount = %d, want > 0", got)
	}
	// 2.6K → should parse to ~2600
	if got := detail.FollowerCount(); got < 2000 || got > 3000 {
		t.Errorf("FollowerCount = %d, want between 2000 and 3000", got)
	}

	// Slug
	if got := detail.Product().Slug(); got != "tanka" {
		t.Errorf("Slug = %q, want %q", got, "tanka")
	}

	// Website URL
	if got := detail.WebsiteURL(); got == "" {
		t.Error("WebsiteURL is empty")
	}
	if got := detail.WebsiteURL(); !strings.Contains(got, "tanka.ai") {
		t.Errorf("WebsiteURL = %q, expected to contain 'tanka.ai'", got)
	}

	if detail.Categories() == nil {
		t.Error("Categories should not be nil")
	}
	if len(detail.SocialLinks()) == 0 {
		t.Error("SocialLinks is empty")
	}
}

func TestParseProductDetailMetadataExtraction(t *testing.T) {
	html := `<!DOCTYPE html><html><head><link rel="canonical" href="https://www.producthunt.com/products/demo"></head><body>
	<div data-test="header">
	  <h1>Demo</h1>
	  <h2 class="text-18">Demo tagline</h2>
	  <a data-test="visit-website-button" href="https://demo.example.com">Visit</a>
	</div>
	<a href="/topics/productivity">Productivity</a>
	<a href="/topics/ai">AI</a>
	<a href="https://x.com/demo">X</a>
	<a href="https://linkedin.com/company/demo">LinkedIn</a>
	</body></html>`

	detail, err := ParseProductDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	if len(detail.Categories()) != 2 {
		t.Errorf("Categories length = %d, want 2", len(detail.Categories()))
	}
	if len(detail.SocialLinks()) != 2 {
		t.Errorf("SocialLinks length = %d, want 2", len(detail.SocialLinks()))
	}
}

func TestParseProductDetailContent(t *testing.T) {
	f, err := os.Open("../testdata/product_detail.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	detail, err := ParseProductDetail(f)
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	// Description should be non-empty and substantial
	desc := detail.Description()
	if desc == "" {
		t.Fatal("Description is empty")
	}
	if len(desc) < 50 {
		t.Errorf("Description length = %d, want >= 50, got %q", len(desc), desc)
	}
	if !strings.Contains(desc, "Tanka") {
		t.Errorf("Description should mention Tanka, got %q", desc)
	}

	// Maker comment should be non-empty and substantial
	mc := detail.MakerComment()
	if mc == "" {
		t.Fatal("MakerComment is empty")
	}
	if len(mc) < 50 {
		t.Errorf("MakerComment length = %d, want >= 50", len(mc))
	}
	// Maker comment should contain key terms from the comment
	if !strings.Contains(mc, "Tanka") {
		t.Errorf("MakerComment should mention Tanka")
	}
}

func TestParseProductDetailMinimalHTML(t *testing.T) {
	// Minimal HTML with no product data — should not panic, return zero values gracefully
	minimal := `<!DOCTYPE html><html><head><title>Test</title></head><body><div></div></body></html>`
	reader := strings.NewReader(minimal)

	detail, err := ParseProductDetail(reader)
	if err != nil {
		t.Fatalf("ParseProductDetail on minimal HTML: %v", err)
	}

	// All fields should be zero/empty, no panic
	if got := detail.Product().Name(); got != "" {
		t.Errorf("Name = %q, want empty", got)
	}
	if got := detail.Rating(); got != 0 {
		t.Errorf("Rating = %v, want 0", got)
	}
	if got := detail.ReviewCount(); got != 0 {
		t.Errorf("ReviewCount = %d, want 0", got)
	}
	if got := detail.FollowerCount(); got != 0 {
		t.Errorf("FollowerCount = %d, want 0", got)
	}
	if got := detail.WebsiteURL(); got != "" {
		t.Errorf("WebsiteURL = %q, want empty", got)
	}
	if got := detail.Description(); got != "" {
		t.Errorf("Description = %q, want empty", got)
	}
	if got := detail.MakerComment(); got != "" {
		t.Errorf("MakerComment = %q, want empty", got)
	}
	if len(detail.Categories()) != 0 {
		t.Errorf("Categories length = %d, want 0", len(detail.Categories()))
	}
	if len(detail.SocialLinks()) != 0 {
		t.Errorf("SocialLinks length = %d, want 0", len(detail.SocialLinks()))
	}
}
