package scraper

import (
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qyinm/phtui/types"
)

// ParseLeaderboard parses Product Hunt leaderboard HTML and returns a slice of Products.
// It expects SSR HTML from Product Hunt's Next.js pages.
func ParseLeaderboard(reader io.Reader) ([]types.Product, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	var products []types.Product
	rank := 0

	doc.Find("section[data-test^='post-item-']").Each(func(_ int, s *goquery.Selection) {
		rank++

		// Product name and slug from the post-name link
		nameLink := s.Find("span[data-test^='post-name-'] a").First()
		name := strings.TrimSpace(nameLink.Text())
		href, _ := nameLink.Attr("href")
		slug := strings.TrimPrefix(href, "/products/")

		// Tagline
		tagline := strings.TrimSpace(s.Find("span.text-secondary").First().Text())

		// Categories from topic links
		var categories []string
		s.Find("a[href^='/topics/']").Each(func(_ int, a *goquery.Selection) {
			cat := strings.TrimSpace(a.Text())
			if cat != "" {
				categories = append(categories, cat)
			}
		})

		// Vote count from vote button
		voteBtn := s.Find("button[data-test='vote-button']")
		voteText := strings.TrimSpace(voteBtn.Find("p").First().Text())
		voteCount := parseCount(voteText)

		// Comment count from the button before vote button
		commentBtn := voteBtn.Prev()
		commentText := strings.TrimSpace(commentBtn.Find("p").First().Text())
		commentCount := parseCount(commentText)

		// Thumbnail URL: try img src first, fallback to video poster
		thumbnailURL, _ := s.Find("img").First().Attr("src")
		if thumbnailURL == "" {
			thumbnailURL, _ = s.Find("video").First().Attr("poster")
		}

		products = append(products, types.NewProduct(
			name, tagline, categories,
			voteCount, commentCount,
			slug, thumbnailURL, rank,
		))
	})

	return products, nil
}

// parseCount strips commas and converts a string to int. Returns 0 on failure.
func parseCount(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.Atoi(s)
	return n
}
