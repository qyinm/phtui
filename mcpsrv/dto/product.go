package dto

type Product struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Tagline      string   `json:"tagline"`
	Votes        int      `json:"votes"`
	Comments     int      `json:"comments"`
	Rank         int      `json:"rank"`
	ThumbnailURL string   `json:"thumbnail_url"`
	Categories   []string `json:"categories"`
}
