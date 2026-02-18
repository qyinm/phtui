package scraper

import (
	"strings"
	"testing"
)

func TestParseSearchResults(t *testing.T) {
	html := `
<!doctype html>
<html><body>
  <main>
    <section>
      <article>
        <a href="/products/alpha-ai"><h3>Alpha AI</h3></a>
        <p>AI agent for support teams</p>
        <a href="/topics/ai-agents">AI Agents</a>
      </article>
      <article>
        <a href="/products/beta-note?ref=search">Beta Note</a>
        <span>Write docs fast</span>
      </article>
    </section>
  </main>
</body></html>`

	got, err := ParseSearchResults(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseSearchResults error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 products, got %d", len(got))
	}

	if got[0].Slug() != "alpha-ai" || got[0].Name() != "Alpha AI" {
		t.Fatalf("unexpected first product: slug=%q name=%q", got[0].Slug(), got[0].Name())
	}
	if got[1].Slug() != "beta-note" {
		t.Fatalf("unexpected second slug: %q", got[1].Slug())
	}
}

func TestNormalizeProductSlug(t *testing.T) {
	cases := map[string]string{
		"/products/demo":               "demo",
		"/products/demo/":              "demo",
		"/products/demo?ref=search":    "demo",
		"/products/demo/reviews":       "demo",
		"https://example.com/products": "",
	}
	for in, want := range cases {
		if got := normalizeProductSlug(in); got != want {
			t.Fatalf("normalizeProductSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLooksLikeCloudflareChallenge(t *testing.T) {
	html := `<html><head><title>Just a moment...</title></head><body><script>window._cf_chl_opt={};</script></body></html>`
	if !looksLikeCloudflareChallenge(html) {
		t.Fatalf("expected challenge page detection to be true")
	}
}

func TestParseHydrationSearchProducts(t *testing.T) {
	raw := `"productSearch":{"__typename":"ProductSearchConnection","edges":[` +
		`{"__typename":"ProductEdge","node":{"__typename":"Product","id":"1","name":"Claude by Anthropic","tagline":"A family of foundational AI models","slug":"claude","reviewsRating":4.96,"reviewsCount":627,"logoUuid":"logo1.png","isNoLongerOnline":false}},` +
		`{"__typename":"ProductEdge","node":{"__typename":"Product","id":"2","name":"Claude Code","tagline":"Anthropic’s deep-context AI coder","slug":"claude-code","reviewsRating":5,"reviewsCount":191,"logoUuid":"logo2.png","isNoLongerOnline":false}}` +
		`],"pageInfo":{"__typename":"PageInfo"}}`

	got := parseHydrationSearchProducts(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 products, got %d", len(got))
	}
	if got[0].Slug() != "claude" || got[1].Slug() != "claude-code" {
		t.Fatalf("unexpected slugs: %q %q", got[0].Slug(), got[1].Slug())
	}
}
