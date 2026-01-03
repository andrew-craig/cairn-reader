package dto

// DetectURLRequest represents the request body for URL detection
type DetectURLRequest struct {
	URL string `json:"url"`
}

// DetectURLResponse represents the response for URL detection
type DetectURLResponse struct {
	URL   string  `json:"url"`
	Type  string  `json:"type"`
	Title *string `json:"title"`
}
