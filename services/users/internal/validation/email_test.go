package validation

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		// Valid cases
		{"simple valid email", "user@example.com", false},
		{"subdomain", "user@mail.example.com", false},
		{"plus addressing", "user+tag@example.com", false},
		{"numeric local part", "123@example.com", false},
		{"hyphen in domain", "user@my-domain.com", false},

		// Invalid cases
		{"empty string", "", true},
		{"missing @", "userexample.com", true},
		{"missing domain", "user@", true},
		{"missing local part", "@example.com", true},
		{"multiple @", "user@@example.com", true},
		{"spaces", "us er@example.com", true},
		{"display name form", "Name <user@example.com>", true},
		{"no dot in domain (localhost)", "user@localhost", true},
		{"plain string", "notanemail", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}
