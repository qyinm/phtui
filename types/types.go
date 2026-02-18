package types

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

// Period represents the leaderboard time period
type Period int

const (
	Daily Period = iota
	Weekly
	Monthly
)

// String returns the string representation of the period
func (p Period) String() string {
	switch p {
	case Daily:
		return "daily"
	case Weekly:
		return "weekly"
	case Monthly:
		return "monthly"
	default:
		return "unknown"
	}
}

// URLPath returns the Product Hunt leaderboard URL path for the given date
// Daily: /leaderboard/daily/YYYY/M/DD (month and day without leading zeros)
// Weekly: /leaderboard/weekly/YYYY/W (ISO week number)
// Monthly: /leaderboard/monthly/YYYY/M (month without leading zero)
func (p Period) URLPath(date time.Time) string {
	year := date.Year()
	month := int(date.Month())
	day := date.Day()

	switch p {
	case Daily:
		return fmt.Sprintf("/leaderboard/daily/%d/%d/%d", year, month, day)
	case Weekly:
		_, week := date.ISOWeek()
		return fmt.Sprintf("/leaderboard/weekly/%d/%d", year, week)
	case Monthly:
		return fmt.Sprintf("/leaderboard/monthly/%d/%d", year, month)
	default:
		return ""
	}
}

// Product represents a PH leaderboard entry
type Product struct {
	name         string
	tagline      string
	categories   []string
	voteCount    int
	commentCount int
	slug         string
	thumbnailURL string
	rank         int
}

// NewProduct creates a new Product with the given fields
func NewProduct(name, tagline string, categories []string, voteCount, commentCount int, slug, thumbnailURL string, rank int) Product {
	return Product{
		name:         name,
		tagline:      tagline,
		categories:   categories,
		voteCount:    voteCount,
		commentCount: commentCount,
		slug:         slug,
		thumbnailURL: thumbnailURL,
		rank:         rank,
	}
}

// Getters for Product fields
func (p Product) Name() string         { return p.name }
func (p Product) Tagline() string      { return p.tagline }
func (p Product) Categories() []string { return p.categories }
func (p Product) VoteCount() int       { return p.voteCount }
func (p Product) CommentCount() int    { return p.commentCount }
func (p Product) Slug() string         { return p.slug }
func (p Product) ThumbnailURL() string { return p.thumbnailURL }
func (p Product) Rank() int            { return p.rank }

// list.Item interface implementation
func (p Product) Title() string       { return p.name }
func (p Product) Description() string { return p.tagline }
func (p Product) FilterValue() string { return p.name }

// Compile-time check that Product implements list.Item
var _ list.Item = Product{}

// ProductDetail extends Product with full detail page data
type ProductDetail struct {
	product       Product
	description   string
	rating        float64
	reviewCount   int
	followerCount int
	makerComment  string
	websiteURL    string
	categories    []string
	socialLinks   []string
}

// NewProductDetail creates a new ProductDetail
func NewProductDetail(product Product, description string, rating float64, reviewCount, followerCount int, makerComment, websiteURL string, categories, socialLinks []string) ProductDetail {
	return ProductDetail{
		product:       product,
		description:   description,
		rating:        rating,
		reviewCount:   reviewCount,
		followerCount: followerCount,
		makerComment:  makerComment,
		websiteURL:    websiteURL,
		categories:    categories,
		socialLinks:   socialLinks,
	}
}

// Getters for ProductDetail fields
func (pd ProductDetail) Product() Product      { return pd.product }
func (pd ProductDetail) Description() string   { return pd.description }
func (pd ProductDetail) Rating() float64       { return pd.rating }
func (pd ProductDetail) ReviewCount() int      { return pd.reviewCount }
func (pd ProductDetail) FollowerCount() int    { return pd.followerCount }
func (pd ProductDetail) MakerComment() string  { return pd.makerComment }
func (pd ProductDetail) WebsiteURL() string    { return pd.websiteURL }
func (pd ProductDetail) Categories() []string  { return pd.categories }
func (pd ProductDetail) SocialLinks() []string { return pd.socialLinks }

type LeaderboardEntry = Product

// ProductSource is the core abstraction for data access.
// Sync methods only — no bubbletea dependency.
// Future: MCP server, CLI can call these directly.
type ProductSource interface {
	GetLeaderboard(period Period, date time.Time) ([]Product, error)
	GetProductDetail(slug string) (ProductDetail, error)
}
