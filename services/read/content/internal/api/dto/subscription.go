package dto

// SubscriptionType represents the type of subscription
type SubscriptionType string

const (
	SubscriptionTypeRSS    SubscriptionType = "rss"
	SubscriptionTypeSocial SubscriptionType = "social" // Future
	SubscriptionTypeEmail  SubscriptionType = "email"  // Future
)

// UnifiedSubscription represents a subscription from any source (RSS, social, email, etc.)
type UnifiedSubscription struct {
	ID           string           `json:"id"`
	Type         SubscriptionType `json:"type"`
	Title        string           `json:"title"`
	Description  string           `json:"description,omitempty"`
	SubscribedAt string           `json:"subscribed_at"`

	// Type-specific data (only one will be populated based on Type)
	RSSData    *RSSSubscriptionData    `json:"rss_data,omitempty"`
	SocialData *SocialSubscriptionData `json:"social_data,omitempty"` // Future
	EmailData  *EmailSubscriptionData  `json:"email_data,omitempty"`  // Future
}

// RSSSubscriptionData contains RSS-specific subscription information
type RSSSubscriptionData struct {
	FeedURL       string `json:"feed_url"`
	SiteURL       string `json:"site_url,omitempty"`
	PollingTier   string `json:"polling_tier,omitempty"`
	LastFetchedAt string `json:"last_fetched_at,omitempty"`
}

// SocialSubscriptionData contains social feed-specific subscription information
// Future implementation
type SocialSubscriptionData struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle"`
}

// EmailSubscriptionData contains email-specific subscription information
// Future implementation
type EmailSubscriptionData struct {
	EmailAddress string `json:"email_address"`
	FilterRules  string `json:"filter_rules,omitempty"`
}

// ListSubscriptionsResponse represents the response for listing all subscriptions
type ListSubscriptionsResponse struct {
	Subscriptions []UnifiedSubscription `json:"subscriptions"`
	TotalCount    int                   `json:"total_count"`
}
