package scraper

import (
	"os"
	"strings"
	"testing"
	"time"
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

	// Launch date should be parsed
	if got := detail.LaunchDate(); got.IsZero() {
		t.Error("LaunchDate is zero, expected valid date")
	}

	// Maker name
	if got := detail.MakerName(); got != "Vincent Zhu" {
		t.Errorf("MakerName = %q, want %q", got, "Vincent Zhu")
	}

	// Maker profile URL
	if got := detail.MakerProfileURL(); got == "" {
		t.Error("MakerProfileURL is empty")
	}

	// Pricing should be "Free" (price:0 in test data)
	if got := detail.PricingInfo(); got != "Free" {
		t.Errorf("PricingInfo = %q, want %q", got, "Free")
	}

	// Pro/Con tags
	if len(detail.ProConTags()) == 0 {
		t.Error("ProConTags is empty, expected at least one")
	}
	// Should have both Positive and Negative tags
	var hasPositive, hasNegative bool
	for _, tag := range detail.ProConTags() {
		if tag.TagType() == "Positive" {
			hasPositive = true
		}
		if tag.TagType() == "Negative" {
			hasNegative = true
		}
		if tag.Count() <= 0 {
			t.Errorf("ProConTag %q has count %d, want > 0", tag.Name(), tag.Count())
		}
	}
	if !hasPositive {
		t.Error("No Positive ProConTags found")
	}
	if !hasNegative {
		t.Error("No Negative ProConTags found")
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
	if got := detail.LaunchDate(); !got.IsZero() {
		t.Errorf("LaunchDate = %v, want zero", got)
	}
	if got := detail.MakerName(); got != "" {
		t.Errorf("MakerName = %q, want empty", got)
	}
	if got := detail.MakerProfileURL(); got != "" {
		t.Errorf("MakerProfileURL = %q, want empty", got)
	}
	if got := detail.PricingInfo(); got != "" {
		t.Errorf("PricingInfo = %q, want empty", got)
	}
	if len(detail.ProConTags()) != 0 {
		t.Errorf("ProConTags length = %d, want 0", len(detail.ProConTags()))
	}
}

func TestParseProductDetailLaunchDateUsesEarliestFeaturedAt(t *testing.T) {
	html := `<!DOCTYPE html><html><head>
	<link rel="canonical" href="https://www.producthunt.com/products/demo">
	</head><body>
	<div data-test="header"><h1>Demo</h1><h2 class="text-18">Demo tagline</h2></div>
	<script>{"featuredAt":"2026-02-05T00:01:00-08:00","featuredAt":"2025-02-18T00:01:00-08:00"}</script>
	</body></html>`

	detail, err := ParseProductDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	got := detail.LaunchDate()
	want, _ := time.Parse(time.RFC3339, "2025-02-18T00:01:00-08:00")
	if !got.Equal(want) {
		t.Errorf("LaunchDate = %v, want %v", got, want)
	}
}

func TestParseProductDetailProConTagUsesMaxCountForDuplicates(t *testing.T) {
	html := `<!DOCTYPE html><html><head>
	<link rel="canonical" href="https://www.producthunt.com/products/demo">
	</head><body>
	<div data-test="header"><h1>Demo</h1><h2 class="text-18">Demo tagline</h2></div>
	<script>{
	"__typename":"ReviewAiProConTag","id":"1","name":"smart replies","type":"Positive","count":2,
	"__typename":"ReviewAiProConTag","id":"2","name":"smart replies","type":"Positive","count":9,
	"__typename":"ReviewAiProConTag","id":"3","name":"mobile","type":"Negative","count":1
	}</script>
	</body></html>`

	detail, err := ParseProductDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	if len(detail.ProConTags()) != 2 {
		t.Fatalf("ProConTags length = %d, want 2", len(detail.ProConTags()))
	}

	counts := map[string]int{}
	for _, tag := range detail.ProConTags() {
		counts[tag.Name()+"|"+tag.TagType()] = tag.Count()
	}
	if got := counts["smart replies|Positive"]; got != 9 {
		t.Errorf("smart replies Positive count = %d, want 9", got)
	}
	if got := counts["mobile|Negative"]; got != 1 {
		t.Errorf("mobile Negative count = %d, want 1", got)
	}
}

func TestParseProductDetailPricingPrefersJSONLDProductPrice(t *testing.T) {
	html := `<!DOCTYPE html><html><head>
	<link rel="canonical" href="https://www.producthunt.com/products/demo">
	<script type="application/ld+json">
	{"@context":"https://schema.org","@type":"Product","name":"Demo","offers":{"@type":"Offer","price":49,"priceCurrency":"USD"}}
	</script>
	</head><body>
	<div data-test="header"><h1>Demo</h1><h2 class="text-18">Demo tagline</h2></div>
	<script>{"price":0,"someOtherNode":{"price":0}}</script>
	</body></html>`

	detail, err := ParseProductDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseProductDetail: %v", err)
	}

	if got := detail.PricingInfo(); got != "$49" {
		t.Errorf("PricingInfo = %q, want %q", got, "$49")
	}
}
