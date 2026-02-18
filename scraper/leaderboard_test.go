package scraper

import (
	"os"
	"strings"
	"testing"
)

func TestParseLeaderboard_Daily(t *testing.T) {
	f, err := os.Open("../testdata/leaderboard_daily.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	products, err := ParseLeaderboard(f)
	if err != nil {
		t.Fatalf("ParseLeaderboard returned error: %v", err)
	}

	if len(products) < 10 {
		t.Fatalf("expected at least 10 products, got %d", len(products))
	}

	// First product assertions
	first := products[0]
	if first.Name() == "" {
		t.Error("first product name is empty")
	}
	if first.Tagline() == "" {
		t.Error("first product tagline is empty")
	}
	if first.Slug() == "" {
		t.Error("first product slug is empty")
	}
	if first.VoteCount() <= 0 {
		t.Errorf("first product vote count should be > 0, got %d", first.VoteCount())
	}
	if first.CommentCount() < 0 {
		t.Errorf("first product comment count should be >= 0, got %d", first.CommentCount())
	}
	if first.Rank() != 1 {
		t.Errorf("first product rank should be 1, got %d", first.Rank())
	}
	if first.ThumbnailURL() == "" {
		t.Error("first product thumbnail URL is empty")
	}
	if len(first.Categories()) == 0 {
		t.Error("first product has no categories")
	}

	// Verify rank ordering
	for i, p := range products {
		if p.Rank() != i+1 {
			t.Errorf("product %d rank should be %d, got %d", i, i+1, p.Rank())
		}
	}
}

func TestParseLeaderboard_Weekly(t *testing.T) {
	f, err := os.Open("../testdata/leaderboard_weekly.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	products, err := ParseLeaderboard(f)
	if err != nil {
		t.Fatalf("ParseLeaderboard returned error: %v", err)
	}

	if len(products) < 10 {
		t.Fatalf("expected at least 10 products, got %d", len(products))
	}

	first := products[0]
	if first.Name() == "" {
		t.Error("first product name is empty")
	}
	if first.Slug() == "" {
		t.Error("first product slug is empty")
	}
	if first.VoteCount() <= 0 {
		t.Errorf("first product vote count should be > 0, got %d", first.VoteCount())
	}
	if first.ThumbnailURL() == "" {
		t.Error("first product thumbnail URL is empty")
	}
	if len(first.Categories()) == 0 {
		t.Error("first product has no categories")
	}

	// Verify same structure works for weekly
	last := products[len(products)-1]
	if last.Name() == "" {
		t.Error("last product name is empty")
	}
	if last.Rank() != len(products) {
		t.Errorf("last product rank should be %d, got %d", len(products), last.Rank())
	}
}

func TestParseLeaderboard_Empty(t *testing.T) {
	f, err := os.Open("../testdata/leaderboard_empty.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	products, err := ParseLeaderboard(f)
	if err != nil {
		t.Fatalf("ParseLeaderboard should not error on empty HTML, got: %v", err)
	}

	if len(products) != 0 {
		t.Errorf("expected 0 products for empty HTML, got %d", len(products))
	}
}

func TestParseLeaderboard_Malformed(t *testing.T) {
	r := strings.NewReader("<html><body><div>not a leaderboard</div></body></html>")

	products, err := ParseLeaderboard(r)
	if err != nil {
		t.Fatalf("ParseLeaderboard should not error on malformed HTML, got: %v", err)
	}

	if len(products) != 0 {
		t.Errorf("expected 0 products for malformed HTML, got %d", len(products))
	}
}
