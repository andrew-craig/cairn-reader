// Package dto provides data transfer objects for API requests and responses.
package dto

// CreateAddressResponse represents the response for email address creation
type CreateAddressResponse struct {
	Data struct {
		EmailAddress string `json:"email_address"`
		CreatedAt    string `json:"created_at"`
	} `json:"data"`
}

// GetAddressResponse represents the response for getting an email address
type GetAddressResponse struct {
	Data struct {
		EmailAddress string `json:"email_address"`
		CreatedAt    string `json:"created_at"`
	} `json:"data"`
}
