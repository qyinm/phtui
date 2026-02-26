package dto

import (
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
		Product:       FromProduct(pd.Product()),
		Description:   pd.Description(),
		Rating:        pd.Rating(),
		ReviewCount:   pd.ReviewCount(),
		FollowerCount: pd.FollowerCount(),
		MakerComment:  pd.MakerComment(),
		WebsiteURL:    pd.WebsiteURL(),
		SocialLinks:   append([]string(nil), pd.SocialLinks()...),
		MakerName:     pd.MakerName(),
		MakerProfile:  pd.MakerProfileURL(),
		PricingInfo:   pd.PricingInfo(),
		PricingType:   pricingType,
		PricingAmount: pricingAmount,
		PricingPeriod: pricingPeriod,
		LaunchDate:    launchDate,
		Pros:          pros,
		Cons:          cons,
	}
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
