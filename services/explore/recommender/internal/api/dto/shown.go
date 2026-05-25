package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// MarkShownRequest represents a batch request to mark articles as shown
// to the authenticated user. Sent by clients after they have visually
// presented the articles above the viewport mid-point following a scroll
// interaction.
type MarkShownRequest struct {
	ArticleIDs []string `json:"article_ids"`
}

// Validate validates the MarkShownRequest
func (m MarkShownRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.ArticleIDs,
			validation.Required.Error("article_ids is required"),
			validation.Length(1, 100).Error("article_ids must contain between 1 and 100 IDs"),
		),
	)
}
