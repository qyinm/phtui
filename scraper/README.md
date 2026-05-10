# Scraper maintenance

The scraper parses Product Hunt HTML and SSR hydration payloads. Product Hunt markup can change without notice, so parser changes should be backed by fixtures and focused tests.

## Fixture refresh checklist

1. Save a representative page under `testdata/` for the parser being changed:
   - leaderboard: `leaderboard_daily.html`, `leaderboard_weekly.html`, or `leaderboard_empty.html`
   - category pages: `category_products.html`
   - product detail pages: `product_detail.html`
2. Add or update a focused parser test in `scraper/*_test.go` before changing parser logic.
3. Run the focused test first, then the full scraper suite:

```bash
go test ./scraper -run TestParse -v
go test ./scraper
```

4. If live Product Hunt responses differ from fixtures, prefer adding a new fixture that captures the new shape instead of deleting existing fallback coverage.
5. Keep network calls out of unit tests; use saved fixtures for deterministic CI.
