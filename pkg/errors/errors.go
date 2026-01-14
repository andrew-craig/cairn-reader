// Package errors provides sentinel errors and error handling utilities
// for the Cairn application. These errors enable programmatic error checking
// across services using errors.Is() and errors.As().
package errors

import "errors"

// Common repository errors
var (
	// ErrNotFound indicates that a requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates that a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates that provided input is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates that authentication is required
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates that the user lacks permission
	ErrForbidden = errors.New("forbidden")
)

// Domain-specific errors
var (
	// ErrArticleNotFound indicates that an article was not found
	ErrArticleNotFound = errors.New("article not found")

	// ErrUserNotFound indicates that a user was not found
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists indicates that a user already exists with the given email or device ID
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidUserData indicates that user data is invalid
	ErrInvalidUserData = errors.New("invalid user data")

	// ErrFeedNotFound indicates that a feed was not found
	ErrFeedNotFound = errors.New("feed not found")

	// ErrInvalidVoteType indicates that the vote type is not valid
	ErrInvalidVoteType = errors.New("invalid vote type")

	// ErrInvalidUserID indicates that the user ID format is invalid
	ErrInvalidUserID = errors.New("invalid user ID format")

	// ErrTokenNotFound indicates that a refresh token was not found
	ErrTokenNotFound = errors.New("refresh token not found")

	// ErrTokenExpired indicates that a refresh token has expired
	ErrTokenExpired = errors.New("refresh token expired")

	// ErrInvalidTokenData indicates that refresh token data is invalid
	ErrInvalidTokenData = errors.New("invalid refresh token data")
)
