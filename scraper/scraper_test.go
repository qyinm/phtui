package scraper

import (
	"testing"
	"time"

	"github.com/qyinm/phtui/types"
)

func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		period   types.Period
		date     time.Time
		expected string
	}{
		{
			name:     "Daily 2025-02-18",
			period:   types.Daily,
			date:     time.Date(2025, 2, 18, 0, 0, 0, 0, time.UTC),
			expected: "https://www.producthunt.com/leaderboard/daily/2025/2/18",
		},
		{
			name:     "Weekly 2025-02-18",
			period:   types.Weekly,
			date:     time.Date(2025, 2, 18, 0, 0, 0, 0, time.UTC),
			expected: "https://www.producthunt.com/leaderboard/weekly/2025/8",
		},
		{
			name:     "Monthly 2025-02-18",
			period:   types.Monthly,
			date:     time.Date(2025, 2, 18, 0, 0, 0, 0, time.UTC),
			expected: "https://www.producthunt.com/leaderboard/monthly/2025/2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := baseURL + tt.period.URLPath(tt.date)
			if url != tt.expected {
				t.Errorf("URL mismatch:\ngot:  %s\nwant: %s", url, tt.expected)
			}
		})
	}
}

func TestProductDetailURL(t *testing.T) {
	slug := "example-product"
	expected := "https://www.producthunt.com/products/example-product"
	url := baseURL + "/products/" + slug

	if url != expected {
		t.Errorf("Product detail URL mismatch:\ngot:  %s\nwant: %s", url, expected)
	}
}
