package dto

type ProductDetail struct {
	Product
	Description        string   `json:"description"`
	Rating             float64  `json:"rating"`
	ReviewCount        int      `json:"review_count"`
	FollowerCount      int      `json:"follower_count"`
	MakerComment       string   `json:"maker_comment"`
	WebsiteURL         string   `json:"website_url"`
	SocialLinks        []string `json:"social_links"`
	MakerName          string   `json:"maker_name"`
	MakerProfile       string   `json:"maker_profile_url"`
	PricingInfo        string   `json:"pricing_info"`
	PricingType        string   `json:"pricing_type"`
	PricingAmount      string   `json:"pricing_amount"`
	PricingPeriod      string   `json:"pricing_period"`
	LaunchDate         string   `json:"launch_date"`
	Pros               []ProCon `json:"pros"`
	Cons               []ProCon `json:"cons"`
	PositioningHint    string   `json:"positioning_hint"`
	TargetUserSignal   string   `json:"target_user_signal"`
	MonetizationSignal string   `json:"monetization_signal"`
	FeatureSignals     []string `json:"feature_signals"`
}

type ProCon struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
