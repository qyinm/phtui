package dto

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/qyinm/phtui/types"
)

var amountRe = regexp.MustCompile(`\$\s*[0-9]+(?:\.[0-9]{1,2})?`)

func FromProduct(p types.Product) Product {
	return Product{
		Slug:         p.Slug(),
		Name:         p.Name(),
		Tagline:      p.Tagline(),
		Votes:        p.VoteCount(),
		Comments:     p.CommentCount(),
		Rank:         p.Rank(),
		ThumbnailURL: p.ThumbnailURL(),
		Categories:   append([]string(nil), p.Categories()...),
	}
}

func FromProducts(products []types.Product) []Product {
	out := make([]Product, 0, len(products))
	for _, p := range products {
		out = append(out, FromProduct(p))
	}
	return out
}

func FromCategory(c types.CategoryLink) Category {
	return Category{Slug: c.Slug(), Name: c.Name()}
}

func FromCategories(categories []types.CategoryLink) []Category {
	out := make([]Category, 0, len(categories))
	for _, c := range categories {
		out = append(out, FromCategory(c))
	}
	return out
}

func FromProductDetail(pd types.ProductDetail) ProductDetail {
	pricingType, pricingAmount, pricingPeriod := parsePricing(pd.PricingInfo())
	pros := make([]ProCon, 0)
	cons := make([]ProCon, 0)
	for _, tag := range pd.ProConTags() {
		pc := ProCon{Name: tag.Name(), Count: tag.Count()}
		if strings.EqualFold(tag.TagType(), "negative") {
			cons = append(cons, pc)
			continue
		}
		pros = append(pros, pc)
	}

	launchDate := ""
	if !pd.LaunchDate().IsZero() {
		launchDate = pd.LaunchDate().Format(time.DateOnly)
	}

	return ProductDetail{
		Product:            FromProduct(pd.Product()),
		Description:        pd.Description(),
		Rating:             pd.Rating(),
		ReviewCount:        pd.ReviewCount(),
		FollowerCount:      pd.FollowerCount(),
		MakerComment:       pd.MakerComment(),
		WebsiteURL:         pd.WebsiteURL(),
		SocialLinks:        append([]string(nil), pd.SocialLinks()...),
		MakerName:          pd.MakerName(),
		MakerProfile:       pd.MakerProfileURL(),
		PricingInfo:        pd.PricingInfo(),
		PricingType:        pricingType,
		PricingAmount:      pricingAmount,
		PricingPeriod:      pricingPeriod,
		LaunchDate:         launchDate,
		Pros:               pros,
		Cons:               cons,
		PositioningHint:    positioningHint(pd),
		TargetUserSignal:   targetUserSignal(pd),
		MonetizationSignal: monetizationSignal(pricingType, pricingAmount, pricingPeriod, pd.PricingInfo()),
		FeatureSignals:     detailFeatureSignals(pd, pros, cons),
	}
}

func FromCompareDetails(details []types.ProductDetail) CompareOutput {
	items := make([]CompareItem, 0, len(details))
	for i, d := range details {
		items = append(items, CompareItem{
			Index: i + 1,
			Item:  FromProductDetail(d),
		})
	}
	return CompareOutput{
		Items:   items,
		Summary: buildCompareSummary(items),
	}
}

func buildCompareSummary(items []CompareItem) CompareSummary {
	summary := CompareSummary{Count: len(items)}
	if len(items) == 0 {
		return summary
	}

	// Rating range
	var minRating, maxRating float64
	hasRating := false
	for _, item := range items {
		r := item.Item.Rating
		if r > 0 {
			if !hasRating || r < minRating {
				minRating = r
			}
			if !hasRating || r > maxRating {
				maxRating = r
			}
			hasRating = true
		}
	}
	if hasRating {
		if minRating == maxRating {
			summary.RatingRange = formatFloat(minRating)
		} else {
			summary.RatingRange = formatFloat(minRating) + "–" + formatFloat(maxRating)
		}
	}

	// Pricing types and range
	seenTypes := make(map[string]struct{})
	var pricingAmounts []string
	for _, item := range items {
		t := item.Item.PricingType
		if t != "" && t != "unknown" {
			if _, ok := seenTypes[t]; !ok {
				seenTypes[t] = struct{}{}
				summary.PricingTypes = append(summary.PricingTypes, t)
			}
		}
		if a := item.Item.PricingAmount; a != "" {
			pricingAmounts = append(pricingAmounts, a)
		}
	}
	if len(pricingAmounts) > 0 {
		if len(pricingAmounts) == 1 {
			summary.PricingRange = pricingAmounts[0]
		} else {
			minA, maxA := pricingAmounts[0], pricingAmounts[0]
			for _, a := range pricingAmounts[1:] {
				if len(a) < len(minA) || (len(a) == len(minA) && a < minA) {
					minA = a
				}
				if len(a) > len(maxA) || (len(a) == len(maxA) && a > maxA) {
					maxA = a
				}
			}
			summary.PricingRange = minA + "–" + maxA
		}
	}

	// Common category (most frequent)
	catCounts := make(map[string]int)
	for _, item := range items {
		for _, c := range item.Item.Categories {
			if c != "" {
				catCounts[c]++
			}
		}
	}
	maxCount := 0
	commonCat := ""
	for cat, count := range catCounts {
		if count > maxCount {
			maxCount = count
			commonCat = cat
		}
	}
	if maxCount >= 2 && commonCat != "" {
		summary.CommonCategory = commonCat
	}

	return summary
}

func formatFloat(f float64) string {
	s := strings.TrimRight(strings.TrimRight(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					fmt.Sprintf("%.1f", f), ".0", "", 1),
				".", ".", 1),
			",", ".", 1),
		"0"), ".")
	if s == "" {
		s = "0"
	}
	return s
}

func positioningHint(pd types.ProductDetail) string {
	if s := strings.TrimSpace(pd.Product().Tagline()); s != "" {
		return s
	}
	return strings.TrimSpace(pd.Description())
}

func targetUserSignal(pd types.ProductDetail) string {
	for _, c := range pd.Product().Categories() {
		if s := strings.TrimSpace(c); s != "" {
			return s + " users"
		}
	}
	for _, c := range pd.Categories() {
		if s := strings.TrimSpace(c); s != "" {
			return s + " users"
		}
	}
	return "Product Hunt users"
}

func monetizationSignal(pricingType, pricingAmount, pricingPeriod, pricingInfo string) string {
	switch pricingType {
	case "paid":
		parts := []string{"paid"}
		if pricingAmount != "" {
			parts = append(parts, pricingAmount)
		}
		if pricingPeriod != "" {
			parts[len(parts)-1] = parts[len(parts)-1] + "/" + pricingPeriod
		}
		return strings.Join(parts, " ")
	case "free":
		return "free"
	}
	return strings.TrimSpace(pricingInfo)
}

func detailFeatureSignals(pd types.ProductDetail, pros, cons []ProCon) []string {
	signals := make([]string, 0, 4)
	for _, p := range pros {
		if strings.TrimSpace(p.Name) != "" {
			signals = append(signals, "Positive signal: "+p.Name)
		}
	}
	for _, c := range cons {
		if strings.TrimSpace(c.Name) != "" {
			signals = append(signals, "Constraint signal: "+c.Name)
		}
	}
	if pd.Rating() > 0 {
		signals = append(signals, "Rating signal")
	}
	if len(signals) == 0 && pd.Description() != "" {
		signals = append(signals, "Description available for feature extraction")
	}
	return signals
}

func parsePricing(pricingInfo string) (string, string, string) {
	s := strings.TrimSpace(pricingInfo)
	if s == "" {
		return "", "", ""
	}

	lower := strings.ToLower(s)
	pricingType := "unknown"
	if strings.Contains(lower, "free") {
		pricingType = "free"
	}
	if strings.Contains(s, "$") {
		pricingType = "paid"
	}

	pricingAmount := strings.TrimSpace(amountRe.FindString(s))

	pricingPeriod := ""
	switch {
	case strings.Contains(lower, "/month"), strings.Contains(lower, "per month"), strings.Contains(lower, "/mo"):
		pricingPeriod = "month"
	case strings.Contains(lower, "/year"), strings.Contains(lower, "per year"), strings.Contains(lower, "/yr"):
		pricingPeriod = "year"
	case strings.Contains(lower, "/week"), strings.Contains(lower, "per week"):
		pricingPeriod = "week"
	case strings.Contains(lower, "/day"), strings.Contains(lower, "per day"):
		pricingPeriod = "day"
	}

	return pricingType, pricingAmount, pricingPeriod
}
