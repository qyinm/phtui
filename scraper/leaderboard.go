package scraper

import (
	"bytes"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qyinm/phtui/types"
)

// ParseLeaderboard parses Product Hunt leaderboard HTML and returns a slice of Products.
// It expects SSR HTML from Product Hunt's Next.js pages.
func ParseLeaderboard(reader io.Reader) ([]types.Product, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	var products []types.Product
	seenBySlug := make(map[string]struct{})

	// Product Hunt can render leaderboard cards with different tag names
	// (e.g. section/article/div), so match by data-test only.
	doc.Find("[data-test^='post-item-']").Each(func(_ int, s *goquery.Selection) {
		p, ok := parseProductCard(s)
		if !ok {
			return
		}
		if _, exists := seenBySlug[p.Slug()]; exists {
			return
		}
		seenBySlug[p.Slug()] = struct{}{}
		products = append(products, p)
	})

	// Fallback path: on some leaderboard variants, certain cards don't expose
	// post-item data-test, but still keep post-name data-test.
	doc.Find("[data-test^='post-name-'] a[href^='/products/']").Each(func(_ int, nameLink *goquery.Selection) {
		href, _ := nameLink.Attr("href")
		slug := strings.TrimPrefix(href, "/products/")
		if slug == "" {
			return
		}
		if _, exists := seenBySlug[slug]; exists {
			return
		}

		card := nameLink.Closest("[data-test^='post-item-']")
		if card.Length() == 0 {
			card = nameLink.Closest("section,article,li")
		}
		if card.Length() == 0 {
			card = nameLink.Parent()
		}
		p, ok := parseProductCard(card)
		if !ok {
			return
		}
		if _, exists := seenBySlug[p.Slug()]; exists {
			return
		}
		seenBySlug[p.Slug()] = struct{}{}
		products = append(products, p)
	})

	// Last-resort fallback: include product cards that don't have data-test attrs
	// (e.g. promoted/expanded list cards), but do have product links.
	doc.Find("main a[href^='/products/']").Each(func(_ int, link *goquery.Selection) {
		href, _ := link.Attr("href")
		slug := strings.TrimPrefix(href, "/products/")
		if slug == "" {
			return
		}
		if _, exists := seenBySlug[slug]; exists {
			return
		}
		card := link.Closest("section,article,li,div")
		if card.Length() == 0 {
			return
		}
		if card.Find("span.text-secondary").Length() == 0 {
			return
		}
		p, ok := parseProductCard(card)
		if !ok {
			return
		}
		if _, exists := seenBySlug[p.Slug()]; exists {
			return
		}
		seenBySlug[p.Slug()] = struct{}{}
		products = append(products, p)
	})

	for i := range products {
		products[i] = types.NewProduct(
			products[i].Name(),
			products[i].Tagline(),
			products[i].Categories(),
			products[i].VoteCount(),
			products[i].CommentCount(),
			products[i].Slug(),
			products[i].ThumbnailURL(),
			i+1,
		)
	}

	// Hydration JSON often includes more leaderboard posts than SSR HTML.
	// Merge any missing posts by slug.
	hydrationProducts := parseHydrationLeaderboardProducts(string(raw))
	indexBySlug := make(map[string]int, len(products))
	for i, p := range products {
		if p.Slug() != "" {
			indexBySlug[p.Slug()] = i
		}
	}
	for _, hp := range hydrationProducts {
		if hp.Slug() == "" {
			continue
		}
		if idx, exists := indexBySlug[hp.Slug()]; exists {
			existing := products[idx]
			name := existing.Name()
			if name == "" {
				name = hp.Name()
			}
			tagline := existing.Tagline()
			if tagline == "" {
				tagline = hp.Tagline()
			}
			categories := existing.Categories()
			if len(categories) == 0 {
				categories = hp.Categories()
			}
			voteCount := existing.VoteCount()
			if hp.VoteCount() > 0 {
				voteCount = hp.VoteCount()
			}
			commentCount := existing.CommentCount()
			if hp.CommentCount() > 0 {
				commentCount = hp.CommentCount()
			}
			rank := existing.Rank()
			if hp.Rank() > 0 {
				rank = hp.Rank()
			}
			products[idx] = types.NewProduct(
				name,
				tagline,
				categories,
				voteCount,
				commentCount,
				existing.Slug(),
				existing.ThumbnailURL(),
				rank,
			)
			continue
		}
		seenBySlug[hp.Slug()] = struct{}{}
		products = append(products, hp)
		indexBySlug[hp.Slug()] = len(products) - 1
	}

	sort.SliceStable(products, func(i, j int) bool {
		ri, rj := products[i].Rank(), products[j].Rank()
		if ri == 0 && rj == 0 {
			return i < j
		}
		if ri == 0 {
			return false
		}
		if rj == 0 {
			return true
		}
		return ri < rj
	})

	// Hydration payload may contain duplicate representations for the same launch
	// (different slugs, same title). Keep the first occurrence.
	deduped := make([]types.Product, 0, len(products))
	seenName := make(map[string]struct{}, len(products))
	for _, p := range products {
		key := strings.ToLower(strings.TrimSpace(p.Name()))
		if _, exists := seenName[key]; exists {
			continue
		}
		seenName[key] = struct{}{}
		deduped = append(deduped, p)
	}

	for i := range deduped {
		p := deduped[i]
		deduped[i] = types.NewProduct(
			p.Name(),
			p.Tagline(),
			p.Categories(),
			p.VoteCount(),
			p.CommentCount(),
			p.Slug(),
			p.ThumbnailURL(),
			i+1,
		)
	}

	return deduped, nil
}

var topicNameRe = regexp.MustCompile(`"name":"([^"]+)"`)

// hydrationPostRe extracts individual post fields from the Apollo SSR JSON.
// The SSR data nests each post inside a HomefeedItemEdge node with this shape:
//
//	{"__typename":"Post","id":"...","name":"...","slug":"post-slug","tagline":"...",
//	 ...,"product":{"__typename":"Product","id":"...","slug":"product-slug",...},
//	 ...,"dailyRank":"N","weeklyRank":"N","monthlyRank":"N",
//	 ...,"topics":{"__typename":"TopicConnection","edges":[...]},
//	 ...,"latestScore":N,...,"commentsCount":N}
var hydrationPostRe = regexp.MustCompile(
	`"__typename":"Post","id":"[^"]+","name":"([^"]+)","slug":"[^"]+","tagline":"([^"]*)"`)
var productSlugRe = regexp.MustCompile(
	`"product":\{"__typename":"Product","id":"[^"]+","slug":"([^"]+)"`)
var dailyRankRe = regexp.MustCompile(`"dailyRank":"(\d+)"`)
var weeklyRankRe = regexp.MustCompile(`"weeklyRank":"(\d+)"`)
var monthlyRankRe = regexp.MustCompile(`"monthlyRank":"(\d+)"`)
var latestScoreRe = regexp.MustCompile(`"latestScore":(\d+)`)
var commentsCountRe = regexp.MustCompile(`"commentsCount":(\d+)`)
var topicsEdgesRe = regexp.MustCompile(`"topics":\{"__typename":"TopicConnection","edges":\[(.*?)\]\}`)

func parseHydrationLeaderboardProducts(raw string) []types.Product {
	// Product Hunt SSR embeds Apollo cache data in a script element.
	// Leaderboard posts live inside "homefeedItems" connection edges.
	// There can be multiple occurrences (duplicate cache entries); we dedup by product slug.
	const edgesStartMarker = `"homefeedItems":{"__typename":"HomefeedItemConnection","edges":[`

	var products []types.Product
	seen := make(map[string]struct{})

	searchFrom := 0
	for {
		start := strings.Index(raw[searchFrom:], edgesStartMarker)
		if start == -1 {
			break
		}
		start += searchFrom
		edgesStart := start + len(edgesStartMarker)
		rest := raw[edgesStart:]
		endRel := strings.Index(rest, `],"pageInfo":{"__typename":"PageInfo"`)
		if endRel == -1 {
			searchFrom = edgesStart
			continue
		}
		edgesBlob := rest[:endRel]
		searchFrom = edgesStart + endRel

		// Split edges by the HomefeedItemEdge boundary and parse each post
		posts := hydrationPostRe.FindAllStringSubmatchIndex(edgesBlob, -1)
		for i, loc := range posts {
			// Determine the chunk for this post (from current match to next match or end)
			chunkEnd := len(edgesBlob)
			if i+1 < len(posts) {
				chunkEnd = posts[i+1][0]
			}
			chunk := edgesBlob[loc[0]:chunkEnd]

			name := decodeJSONEscaped(edgesBlob[loc[2]:loc[3]])
			tagline := decodeJSONEscaped(edgesBlob[loc[4]:loc[5]])

			// Extract product slug (the one used for /products/ URLs)
			pSlugMatch := productSlugRe.FindStringSubmatch(chunk)
			if len(pSlugMatch) < 2 || pSlugMatch[1] == "" {
				continue
			}
			slug := decodeJSONEscaped(pSlugMatch[1])

			if name == "" || slug == "" {
				continue
			}
			if _, ok := seen[slug]; ok {
				continue
			}

			// Extract ranks
			dailyRank := extractInt(dailyRankRe, chunk)
			weeklyRank := extractInt(weeklyRankRe, chunk)
			monthlyRank := extractInt(monthlyRankRe, chunk)

			rank := dailyRank
			if rank == 0 {
				rank = weeklyRank
			}
			if rank == 0 {
				rank = monthlyRank
			}
			if rank <= 0 {
				continue
			}

			voteCount := extractInt(latestScoreRe, chunk)
			commentCount := extractInt(commentsCountRe, chunk)

			var categories []string
			if tm := topicsEdgesRe.FindStringSubmatch(chunk); len(tm) >= 2 {
				for _, nm := range topicNameRe.FindAllStringSubmatch(tm[1], -1) {
					if len(nm) >= 2 {
						cat := strings.TrimSpace(decodeJSONEscaped(nm[1]))
						if cat != "" {
							categories = append(categories, cat)
						}
					}
				}
			}

			seen[slug] = struct{}{}
			products = append(products, types.NewProduct(
				name,
				tagline,
				categories,
				voteCount,
				commentCount,
				slug,
				"",
				rank,
			))
		}
	}

	return products
}

func extractInt(re *regexp.Regexp, s string) int {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	return parseCount(m[1])
}

func decodeJSONEscaped(s string) string {
	unquoted, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		return s
	}
	return unquoted
}

func parseProductCard(s *goquery.Selection) (types.Product, bool) {
	if s == nil || s.Length() == 0 {
		return types.Product{}, false
	}

	nameLink := s.Find("[data-test^='post-name-'] a[href^='/products/']").First()
	if nameLink.Length() == 0 {
		nameLink = s.Find("a[href^='/products/']").First()
	}
	name := strings.TrimSpace(nameLink.Text())
	href, _ := nameLink.Attr("href")
	slug := strings.TrimPrefix(href, "/products/")
	if name == "" || slug == "" {
		return types.Product{}, false
	}

	tagline := strings.TrimSpace(s.Find("span.text-secondary").First().Text())

	var categories []string
	s.Find("a[href^='/topics/']").Each(func(_ int, a *goquery.Selection) {
		cat := strings.TrimSpace(a.Text())
		if cat != "" {
			categories = append(categories, cat)
		}
	})

	voteBtn := s.Find("button[data-test='vote-button']").First()
	voteCount := 0
	commentCount := 0
	if voteBtn.Length() > 0 {
		voteText := strings.TrimSpace(voteBtn.Find("p").First().Text())
		voteCount = parseCount(voteText)

		commentBtn := voteBtn.Prev()
		commentText := strings.TrimSpace(commentBtn.Find("p").First().Text())
		commentCount = parseCount(commentText)
	} else {
		var counts []int
		s.Find("button p").Each(func(_ int, p *goquery.Selection) {
			n := parseCount(strings.TrimSpace(p.Text()))
			if n > 0 {
				counts = append(counts, n)
			}
		})
		if len(counts) > 0 {
			voteCount = counts[len(counts)-1]
		}
		if len(counts) > 1 {
			commentCount = counts[len(counts)-2]
		}
	}

	thumbnailURL, _ := s.Find("img").First().Attr("src")
	if thumbnailURL == "" {
		thumbnailURL, _ = s.Find("video").First().Attr("poster")
	}

	return types.NewProduct(
		name, tagline, categories,
		voteCount, commentCount,
		slug, thumbnailURL, 0,
	), true
}

// parseCount strips commas and converts a string to int. Returns 0 on failure.
func parseCount(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.Atoi(s)
	return n
}
