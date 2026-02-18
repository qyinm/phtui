package scraper

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qyinm/phtui/types"
)

// ParseSearchResults parses Product Hunt search HTML.
// Search page markup differs from leaderboard markup, so parse with
// broader selectors anchored to main content.
func ParseSearchResults(reader io.Reader) ([]types.Product, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	rawText := string(raw)
	if looksLikeCloudflareChallenge(rawText) {
		return nil, fmt.Errorf("search blocked by Cloudflare challenge; interactive browser or API token is required")
	}

	if products := parseHydrationSearchProducts(rawText); len(products) > 0 {
		return products, nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	products := make([]types.Product, 0)
	seen := make(map[string]struct{})

	doc.Find("main a[href^='/products/']").Each(func(_ int, link *goquery.Selection) {
		href, _ := link.Attr("href")
		slug := normalizeProductSlug(href)
		if slug == "" {
			return
		}
		if _, ok := seen[slug]; ok {
			return
		}

		card := link.Closest("article,section,li,div")
		if card.Length() == 0 {
			card = link.Parent()
		}

		name := strings.TrimSpace(link.Text())
		if name == "" {
			name = strings.TrimSpace(card.Find("h1,h2,h3,h4,[data-test*='name']").First().Text())
		}
		if name == "" {
			return
		}

		tagline := extractSearchTagline(card, name)
		var categories []string
		card.Find("a[href^='/topics/']").Each(func(_ int, a *goquery.Selection) {
			cat := strings.TrimSpace(a.Text())
			if cat != "" {
				categories = append(categories, cat)
			}
		})

		thumbnailURL, _ := card.Find("img").First().Attr("src")
		if thumbnailURL == "" {
			thumbnailURL, _ = card.Find("video").First().Attr("poster")
		}

		seen[slug] = struct{}{}
		products = append(products, types.NewProduct(
			name,
			tagline,
			categories,
			0,
			0,
			slug,
			thumbnailURL,
			len(products)+1,
		))
	})

	if len(products) == 0 {
		return nil, fmt.Errorf("no parseable search results from producthunt search page")
	}

	return products, nil
}

var searchBlockRe = regexp.MustCompile(`(?s)"productSearch":\{"__typename":"ProductSearchConnection","edges":\[(.*?)\],"pageInfo":\{`)
var searchNodeRe = regexp.MustCompile(`(?s)"node":\{"__typename":"Product","id":"[^"]+","name":"([^"]+)","tagline":"([^"]*)","slug":"([^"]+)".*?"reviewsCount":([0-9]+).*?"logoUuid":"([^"]*)"`)
var searchPageInfoRe = regexp.MustCompile(`"productSearch":\{"__typename":"ProductSearchConnection","edges":\[.*?\],"pageInfo":\{"__typename":"PageInfo","page":([0-9]+),"hasPreviousPage":(true|false),"hasNextPage":(true|false)\},"pagesCount":([0-9]+)`)

func parseHydrationSearchProducts(raw string) []types.Product {
	blocks := searchBlockRe.FindAllStringSubmatch(raw, -1)
	if len(blocks) == 0 {
		return nil
	}

	products := make([]types.Product, 0)
	seen := make(map[string]struct{})
	for _, b := range blocks {
		if len(b) < 2 {
			continue
		}
		nodes := searchNodeRe.FindAllStringSubmatch(b[1], -1)
		for _, n := range nodes {
			if len(n) < 6 {
				continue
			}
			name := strings.TrimSpace(decodeJSONEscaped(n[1]))
			tagline := strings.TrimSpace(decodeJSONEscaped(n[2]))
			slug := strings.TrimSpace(decodeJSONEscaped(n[3]))
			reviewCount, _ := strconv.Atoi(n[4])
			logo := strings.TrimSpace(decodeJSONEscaped(n[5]))

			if slug == "" || name == "" {
				continue
			}
			if _, ok := seen[slug]; ok {
				continue
			}
			seen[slug] = struct{}{}
			products = append(products, types.NewProduct(
				name,
				tagline,
				nil,
				0,
				reviewCount,
				slug,
				logo,
				len(products)+1,
			))
		}
	}

	return products
}

func parseSearchPageInfo(raw string) (int, bool, bool, int, bool) {
	m := searchPageInfoRe.FindStringSubmatch(raw)
	if len(m) < 5 {
		return 0, false, false, 0, false
	}
	page, _ := strconv.Atoi(m[1])
	hasPrev := m[2] == "true"
	hasNext := m[3] == "true"
	pagesCount, _ := strconv.Atoi(m[4])
	if page <= 0 {
		page = 1
	}
	if pagesCount <= 0 {
		pagesCount = 1
	}
	return page, hasPrev, hasNext, pagesCount, true
}

func looksLikeCloudflareChallenge(html string) bool {
	s := strings.ToLower(html)
	return strings.Contains(s, "<title>just a moment...</title>") &&
		(strings.Contains(s, "cf-challenge") || strings.Contains(s, "_cf_chl_opt"))
}

func normalizeProductSlug(href string) string {
	s := strings.TrimSpace(href)
	if !strings.HasPrefix(s, "/products/") {
		return ""
	}
	s = strings.TrimPrefix(s, "/products/")
	s = strings.SplitN(s, "?", 2)[0]
	s = strings.Trim(s, "/")
	if s == "" {
		return ""
	}
	return strings.SplitN(s, "/", 2)[0]
}

func extractSearchTagline(card *goquery.Selection, name string) string {
	if card == nil || card.Length() == 0 {
		return ""
	}
	candidates := card.Find("p,span")
	nameFold := strings.ToLower(strings.TrimSpace(name))
	for i := 0; i < candidates.Length(); i++ {
		text := strings.TrimSpace(candidates.Eq(i).Text())
		if text == "" {
			continue
		}
		lower := strings.ToLower(text)
		if lower == nameFold {
			continue
		}
		if strings.HasPrefix(text, "#") {
			continue
		}
		return text
	}
	return ""
}
