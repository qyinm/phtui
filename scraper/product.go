package scraper

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qyinm/phtui/types"
)

// ParseProductDetail parses a Product Hunt product detail page and extracts
// product information from the rendered HTML.
func ParseProductDetail(reader io.Reader) (types.ProductDetail, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return types.ProductDetail{}, fmt.Errorf("parse HTML: %w", err)
	}

	header := doc.Find("[data-test='header']")

	// Product name from h1
	name := strings.TrimSpace(header.Find("h1").First().Text())

	// Tagline from first h2.text-18
	tagline := strings.TrimSpace(header.Find("h2.text-18").First().Text())

	// Slug from canonical URL: /products/{slug}
	slug := parseSlugFromDoc(doc)

	// Rating: <span class="text-14 font-medium">4.4</span> inside reviews link
	rating := parseRating(header)

	// Review count: "11 reviews" link text
	reviewCount := parseReviewCount(header)

	// Follower count: "2.6K followers" paragraph text
	followerCount := parseFollowerCount(header)

	// Website URL from visit-website button
	websiteURL, _ := doc.Find("a[data-test='visit-website-button']").Attr("href")
	categories := parseDetailCategories(doc)
	socialLinks := parseSocialLinks(doc)

	// Description: short product blurb
	description := strings.TrimSpace(
		header.Find("div.relative.text-16.font-normal.text-gray-700").First().Text(),
	)

	// Thumbnail from video poster
	thumbnailURL := parseThumbnail(header, name)

	// Maker comment from "Maker Comment" section
	makerComment := parseMakerComment(doc)

	product := types.NewProduct(name, tagline, nil, 0, 0, slug, thumbnailURL, 0)
	detail := types.NewProductDetail(product, description, rating, reviewCount, followerCount, makerComment, websiteURL, categories, socialLinks)

	return detail, nil
}

// parseSlugFromDoc extracts the product slug from the canonical URL.
func parseSlugFromDoc(doc *goquery.Document) string {
	href, exists := doc.Find("link[rel='canonical']").Attr("href")
	if !exists {
		return ""
	}
	parts := strings.Split(href, "/products/")
	if len(parts) < 2 {
		return ""
	}
	return strings.SplitN(parts[1], "/", 2)[0]
}

// parseRating extracts the numeric rating (e.g. 4.4) from the star rating area.
func parseRating(header *goquery.Selection) float64 {
	var rating float64
	header.Find("a[href*='/reviews'] span.text-14").Each(func(i int, s *goquery.Selection) {
		if rating > 0 {
			return
		}
		text := strings.TrimSpace(s.Text())
		if r, err := strconv.ParseFloat(text, 64); err == nil {
			rating = r
		}
	})
	return rating
}

// parseReviewCount extracts the review count from text like "11 reviews".
func parseReviewCount(header *goquery.Selection) int {
	reviewRe := regexp.MustCompile(`(\d+)\s*reviews?`)
	var count int
	header.Find("a[href*='/reviews']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if matches := reviewRe.FindStringSubmatch(text); len(matches) > 1 {
			count, _ = strconv.Atoi(matches[1])
		}
	})
	return count
}

// parseFollowerCount extracts the follower count from text like "2.6K followers".
func parseFollowerCount(header *goquery.Selection) int {
	var count int
	header.Find("p.text-14").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(text, "followers") {
			count = parseCountWithSuffix(text)
		}
	})
	return count
}

// parseCountWithSuffix parses a number with optional K/M suffix.
// Examples: "2.6K" → 2600, "1.5M" → 1500000, "42" → 42
func parseCountWithSuffix(text string) int {
	re := regexp.MustCompile(`([\d.]+)\s*([KkMm]?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	if len(matches) > 2 {
		switch strings.ToUpper(matches[2]) {
		case "K":
			val *= 1000
		case "M":
			val *= 1000000
		}
	}
	return int(math.Round(val))
}

// parseThumbnail extracts the thumbnail URL from the video poster attribute.
func parseThumbnail(header *goquery.Selection, name string) string {
	var url string
	header.Find("video[aria-label]").Each(func(i int, s *goquery.Selection) {
		if url != "" {
			return
		}
		if label, _ := s.Attr("aria-label"); label == name {
			url, _ = s.Attr("poster")
		}
	})
	return url
}

// parseMakerComment extracts the maker's comment from the "Maker Comment" section.
func parseMakerComment(doc *goquery.Document) string {
	var comment string
	doc.Find("h2").Each(func(i int, s *goquery.Selection) {
		if comment != "" {
			return
		}
		if strings.TrimSpace(s.Text()) != "Maker Comment" {
			return
		}
		threadDiv := s.Next()
		proseDiv := threadDiv.Find(".prose")
		if proseDiv.Length() == 0 {
			return
		}
		var parts []string
		proseDiv.First().Find("p").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" {
				parts = append(parts, text)
			}
		})
		comment = strings.Join(parts, "\n\n")
	})
	return comment
}

func parseDetailCategories(doc *goquery.Document) []string {
	seen := make(map[string]struct{})
	categories := make([]string, 0)
	doc.Find("a[href^='/topics/']").Each(func(_ int, s *goquery.Selection) {
		cat := strings.TrimSpace(s.Text())
		if cat == "" {
			return
		}
		if _, ok := seen[cat]; ok {
			return
		}
		seen[cat] = struct{}{}
		categories = append(categories, cat)
	})
	return categories
}

func parseSocialLinks(doc *goquery.Document) []string {
	seen := make(map[string]struct{})
	links := make([]string, 0)
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		h := strings.TrimSpace(href)
		if h == "" {
			return
		}
		if strings.Contains(h, "producthunt.com") {
			return
		}
		if strings.Contains(h, "linkedin.com") || strings.Contains(h, "x.com") || strings.Contains(h, "twitter.com") {
			if _, exists := seen[h]; exists {
				return
			}
			seen[h] = struct{}{}
			links = append(links, h)
		}
	})
	return links
}
