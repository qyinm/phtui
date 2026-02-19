package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

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

	// New fields
	launchDate := parseLaunchDate(doc)
	makerName, makerProfileURL := parseMakerInfo(doc)
	proConTags := parseProConTags(doc)
	pricingInfo := parsePricing(doc)

	product := types.NewProduct(name, tagline, nil, 0, 0, slug, thumbnailURL, 0)
	detail := types.NewProductDetail(product, description, rating, reviewCount, followerCount, makerComment, websiteURL, categories, socialLinks, launchDate, makerName, makerProfileURL, proConTags, pricingInfo)

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

// parseLaunchDate extracts the launch date from "featuredAt" in SSR JSON.
func parseLaunchDate(doc *goquery.Document) time.Time {
	html, err := doc.Html()
	if err != nil {
		return time.Time{}
	}
	re := regexp.MustCompile(`"featuredAt":"([^"]+)"`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return time.Time{}
	}

	var launch time.Time
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		t, parseErr := time.Parse(time.RFC3339, m[1])
		if parseErr != nil {
			continue
		}
		// "featuredAt" can appear multiple times in SSR payloads; choose the earliest.
		if launch.IsZero() || t.Before(launch) {
			launch = t
		}
	}
	return launch
}

// parseMakerInfo extracts maker name and profile URL from meta/link tags.
func parseMakerInfo(doc *goquery.Document) (string, string) {
	name, _ := doc.Find("meta[name='author']").Attr("content")
	profileURL, _ := doc.Find("link[rel='author']").Attr("href")
	return strings.TrimSpace(name), strings.TrimSpace(profileURL)
}

// parseProConTags extracts AI-summarized pro/con tags from SSR JSON.
func parseProConTags(doc *goquery.Document) []types.ProConTag {
	html, err := doc.Html()
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`"__typename":"ReviewAiProConTag","id":"(\d+)","name":"([^"]+)","type":"(\w+)","count":(\d+)`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}

	// Deduplicate by name+type and keep the highest count if duplicates disagree.
	type key struct{ name, tagType string }
	maxCounts := make(map[key]int)
	order := make([]key, 0)

	for _, m := range matches {
		if len(m) < 5 {
			continue
		}
		name := m[2]
		tagType := m[3]
		count, _ := strconv.Atoi(m[4])
		k := key{name, tagType}
		if prev, exists := maxCounts[k]; exists {
			if count > prev {
				maxCounts[k] = count
			}
			continue
		}
		maxCounts[k] = count
		order = append(order, k)
	}

	tags := make([]types.ProConTag, 0, len(order))
	for _, k := range order {
		tags = append(tags, types.NewProConTag(k.name, k.tagType, maxCounts[k]))
	}
	return tags
}

// parsePricing extracts pricing info from SSR JSON "price" field.
func parsePricing(doc *goquery.Document) string {
	if price, ok := parsePricingFromJSONLD(doc); ok {
		return formatPrice(price)
	}

	html, err := doc.Html()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`"price":(\d+)`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return ""
	}

	minPrice := -1
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		price, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			continue
		}
		if minPrice == -1 || price < minPrice {
			minPrice = price
		}
	}
	if minPrice < 0 {
		return ""
	}
	return formatPrice(minPrice)
}

func parsePricingFromJSONLD(doc *goquery.Document) (int, bool) {
	var prices []int
	doc.Find(`script[type='application/ld+json']`).Each(func(_ int, s *goquery.Selection) {
		script := strings.TrimSpace(s.Text())
		if script == "" {
			return
		}
		var payload any
		if err := json.Unmarshal([]byte(script), &payload); err != nil {
			return
		}
		collectProductPrices(payload, &prices)
	})
	if len(prices) == 0 {
		return 0, false
	}
	minPrice := prices[0]
	for _, p := range prices[1:] {
		if p < minPrice {
			minPrice = p
		}
	}
	return minPrice, true
}

func collectProductPrices(node any, prices *[]int) {
	switch v := node.(type) {
	case map[string]any:
		if isProductType(v["@type"]) {
			if price, ok := extractOfferPrice(v["offers"]); ok {
				*prices = append(*prices, price)
			}
		}
		for _, child := range v {
			collectProductPrices(child, prices)
		}
	case []any:
		for _, item := range v {
			collectProductPrices(item, prices)
		}
	}
}

func isProductType(v any) bool {
	switch t := v.(type) {
	case string:
		return t == "Product"
	case []any:
		for _, item := range t {
			s, ok := item.(string)
			if ok && s == "Product" {
				return true
			}
		}
	}
	return false
}

func extractOfferPrice(v any) (int, bool) {
	switch offers := v.(type) {
	case map[string]any:
		return parsePriceValue(offers["price"])
	case []any:
		for _, item := range offers {
			if offer, ok := item.(map[string]any); ok {
				if price, priceOK := parsePriceValue(offer["price"]); priceOK {
					return price, true
				}
			}
		}
	}
	return 0, false
}

func parsePriceValue(v any) (int, bool) {
	switch value := v.(type) {
	case float64:
		return int(value), true
	case string:
		price, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return price, true
	}
	return 0, false
}

func formatPrice(price int) string {
	if price == 0 {
		return "Free"
	}
	return fmt.Sprintf("$%d", price)
}
