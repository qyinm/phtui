package scraper

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qyinm/phtui/types"
)

// ParseCategoryProducts parses a Product Hunt category page
// (e.g. /categories/ai-agents) and extracts the product list
// plus related category links.
func ParseCategoryProducts(reader io.Reader) ([]types.Product, []types.CategoryLink, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("parse HTML: %w", err)
	}

	products := parseCategoryProductCards(doc)
	relatedCategories := parseCategoryRelatedCategories(doc)

	return products, relatedCategories, nil
}

// parseCategoryProductCards extracts products from the category page HTML.
// Each product card is an <a data-grid-span="1" href="/products/{slug}">
// containing a name span and tagline span. Rating and review count appear
// nearby in the same row/grid container.
func parseCategoryProductCards(doc *goquery.Document) []types.Product {
	var products []types.Product
	seen := make(map[string]struct{})

	// Primary approach: find product cards by the grid-span link pattern.
	doc.Find(`a[data-grid-span="1"][href^="/products/"]`).Each(func(_ int, link *goquery.Selection) {
		href, _ := link.Attr("href")
		slug := normalizeProductSlug(href)
		if slug == "" {
			return
		}
		if _, ok := seen[slug]; ok {
			return
		}

		// Product name: <span class="font-semibold text-primary text-16">
		name := strings.TrimSpace(link.Find("span.font-semibold").First().Text())
		if name == "" {
			name = strings.TrimSpace(link.Find("span").First().Text())
		}
		if name == "" {
			return
		}

		// Tagline: <span class="text-secondary font-normal text-14">
		tagline := strings.TrimSpace(link.Find("span.text-secondary").First().Text())

		// Thumbnail: look for image in sibling/parent grid context
		thumbnailURL := findCategoryThumbnail(doc, name, slug)

		// Find the parent grid row to extract rating/reviews
		row := link.Closest(`div,li,section,article`)
		reviewCount := 0
		if row.Length() > 0 {
			reviewCount = parseCategoryReviewCount(row, slug)
		}

		seen[slug] = struct{}{}
		products = append(products, types.NewProduct(
			name,
			tagline,
			nil,
			0,
			reviewCount,
			slug,
			thumbnailURL,
			len(products)+1,
		))
	})

	// Fallback: if no grid-span cards found, try broader product link matching.
	if len(products) == 0 {
		doc.Find(`a[href^="/products/"]`).Each(func(_ int, link *goquery.Selection) {
			href, _ := link.Attr("href")
			slug := normalizeProductSlug(href)
			if slug == "" {
				return
			}
			if _, ok := seen[slug]; ok {
				return
			}

			// Skip review/shoutout sub-page links
			if strings.Contains(href, "/reviews") || strings.Contains(href, "?filter=") {
				return
			}

			name := strings.TrimSpace(link.Text())
			if name == "" {
				return
			}
			// Skip very short names that are likely just icons or labels
			if len(name) < 2 {
				return
			}

			card := link.Closest("div,li,section,article")
			tagline := ""
			if card.Length() > 0 {
				tagline = extractSearchTagline(card, name)
			}

			seen[slug] = struct{}{}
			products = append(products, types.NewProduct(
				name,
				tagline,
				nil,
				0,
				0,
				slug,
				"",
				len(products)+1,
			))
		})
	}

	return products
}

// parseCategoryReviewCount extracts the review count from text like "155 reviews"
// near a product's review link.
func parseCategoryReviewCount(container *goquery.Selection, slug string) int {
	reviewLinkHref := fmt.Sprintf("/products/%s/reviews", slug)
	var count int
	container.Find(fmt.Sprintf(`a[href="%s"]`, reviewLinkHref)).Each(func(_ int, s *goquery.Selection) {
		if count > 0 {
			return
		}
		text := strings.TrimSpace(s.Text())
		count = extractReviewNumber(text)
	})
	// Also check parent for the review link
	if count == 0 {
		container.Parent().Find(fmt.Sprintf(`a[href="%s"]`, reviewLinkHref)).Each(func(_ int, s *goquery.Selection) {
			if count > 0 {
				return
			}
			text := strings.TrimSpace(s.Text())
			count = extractReviewNumber(text)
		})
	}
	return count
}

var categoryReviewRe = regexp.MustCompile(`(\d+)\s*reviews?`)

// extractReviewNumber pulls the numeric count from text like "155 reviews".
func extractReviewNumber(text string) int {
	m := categoryReviewRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// findCategoryThumbnail finds the product thumbnail by looking for
// an img with data-test="{Name}-thumbnail".
func findCategoryThumbnail(doc *goquery.Document, name, slug string) string {
	// Try data-test attribute first
	selector := fmt.Sprintf(`img[data-test="%s-thumbnail"]`, name)
	if img := doc.Find(selector).First(); img.Length() > 0 {
		if src, ok := img.Attr("src"); ok {
			return src
		}
	}
	return ""
}

// parseCategoryRelatedCategories extracts related category links from the page.
// It looks for a[href^="/categories/"] links, excluding pagination links
// (which contain ?page=) and the current category's own canonical link.
func parseCategoryRelatedCategories(doc *goquery.Document) []types.CategoryLink {
	seen := make(map[string]struct{})
	var categories []types.CategoryLink

	// Find the current category slug from the canonical URL or h1
	currentSlug := ""
	if canonical, ok := doc.Find("link[rel='canonical']").Attr("href"); ok {
		if idx := strings.Index(canonical, "/categories/"); idx >= 0 {
			currentSlug = canonical[idx+len("/categories/"):]
			currentSlug = strings.SplitN(currentSlug, "?", 2)[0]
			currentSlug = strings.Trim(currentSlug, "/")
		}
	}

	doc.Find(`a[href^="/categories/"]`).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")

		// Skip pagination links
		if strings.Contains(href, "?page=") || strings.Contains(href, "?order=") || strings.Contains(href, "?ref=") {
			return
		}

		slug := strings.TrimPrefix(href, "/categories/")
		slug = strings.SplitN(slug, "?", 2)[0]
		slug = strings.Trim(slug, "/")
		if slug == "" {
			return
		}

		// Skip the current category
		if slug == currentSlug {
			return
		}

		if _, ok := seen[slug]; ok {
			return
		}

		name := strings.TrimSpace(s.Text())
		if name == "" {
			return
		}

		seen[slug] = struct{}{}
		categories = append(categories, types.NewCategoryLink(name, slug))
	})

	return categories
}
