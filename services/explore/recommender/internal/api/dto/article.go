package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/andrew-craig/cairn/pkg/models"
)

// ArticlesRequest represents a batch of articles to be stored
type ArticlesRequest struct {
	Articles []models.Article `json:"articles"`
}

// Validate validates the ArticlesRequest
func (a ArticlesRequest) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.Articles,
			validation.Required.Error("articles is required"),
			validation.Length(1, 0).Error("articles array cannot be empty"),
		),
	)
}
