package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateUserContentRequest_Validate_Status(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"unread is valid", "unread", false},
		{"reading is valid", "reading", false},
		{"completed is valid", "completed", false},
		{"archived is valid", "archived", false},
		{"legacy 'read' is no longer valid", "read", true},
		{"unknown status is rejected", "bogus", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.status
			req := UpdateUserContentRequest{Status: &status}
			err := req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
