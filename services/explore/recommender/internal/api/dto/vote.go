package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// VoteRequest represents a vote submission request
type VoteRequest struct {
	VoteType string `json:"vote_type"`
}

// Validate validates the VoteRequest
func (v VoteRequest) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.VoteType,
			validation.Required.Error("vote_type is required"),
			validation.In("upvote", "downvote").Error("vote_type must be 'upvote' or 'downvote'"),
		),
	)
}
