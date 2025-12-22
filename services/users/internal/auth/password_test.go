package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewPasswordHasher(t *testing.T) {
	cost := 12
	hasher := NewPasswordHasher(cost)

	if hasher == nil {
		t.Fatal("Expected non-nil hasher")
	}

	if hasher.cost != cost {
		t.Errorf("Expected cost %d, got %d", cost, hasher.cost)
	}
}

func TestPasswordHasher_HashPassword(t *testing.T) {
	hasher := NewPasswordHasher(bcrypt.DefaultCost)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "SecureP@ssw0rd",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "short",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 72), // bcrypt max
			wantErr:  false,
		},
		{
			name:     "password with special characters",
			password: "P@ssw0rd!#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the hash is not empty
				if hash == "" {
					t.Error("Expected non-empty hash")
				}

				// Verify the hash is different from the password
				if hash == tt.password {
					t.Error("Hash should not equal the plain password")
				}

				// Verify the hash starts with bcrypt prefix
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
					t.Error("Hash should have bcrypt prefix")
				}

				// Verify we can compare the password with the hash
				err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
				if err != nil {
					t.Errorf("Generated hash does not match password: %v", err)
				}
			}
		})
	}
}

func TestPasswordHasher_HashPassword_Cost(t *testing.T) {
	tests := []struct {
		name string
		cost int
	}{
		{"minimum cost", bcrypt.MinCost},
		{"default cost", bcrypt.DefaultCost},
		{"cost 12", 12},
		{"cost 14", 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewPasswordHasher(tt.cost)
			password := "TestP@ssw0rd"

			hash, err := hasher.HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}

			// Extract cost from hash and verify
			cost, err := bcrypt.Cost([]byte(hash))
			if err != nil {
				t.Fatalf("Failed to extract cost from hash: %v", err)
			}

			if cost != tt.cost {
				t.Errorf("Expected cost %d, got %d", tt.cost, cost)
			}
		})
	}
}

func TestPasswordHasher_HashPassword_Uniqueness(t *testing.T) {
	hasher := NewPasswordHasher(bcrypt.DefaultCost)
	password := "SameP@ssw0rd"

	hash1, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Bcrypt includes a random salt, so hashes should be different
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (bcrypt should use random salt)")
	}

	// But both should validate against the original password
	if err := bcrypt.CompareHashAndPassword([]byte(hash1), []byte(password)); err != nil {
		t.Error("First hash should validate against password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash2), []byte(password)); err != nil {
		t.Error("Second hash should validate against password")
	}
}

func TestPasswordHasher_ComparePassword(t *testing.T) {
	hasher := NewPasswordHasher(bcrypt.DefaultCost)
	password := "SecureP@ssw0rd"

	hash, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
	}{
		{
			name:           "correct password",
			hashedPassword: hash,
			password:       password,
			wantErr:        false,
		},
		{
			name:           "incorrect password",
			hashedPassword: hash,
			password:       "WrongP@ssw0rd",
			wantErr:        true,
		},
		{
			name:           "case sensitive - uppercase",
			hashedPassword: hash,
			password:       strings.ToUpper(password),
			wantErr:        true,
		},
		{
			name:           "case sensitive - lowercase",
			hashedPassword: hash,
			password:       strings.ToLower(password),
			wantErr:        true,
		},
		{
			name:           "empty password",
			hashedPassword: hash,
			password:       "",
			wantErr:        true,
		},
		{
			name:           "password with extra characters",
			hashedPassword: hash,
			password:       password + "extra",
			wantErr:        true,
		},
		{
			name:           "password missing characters",
			hashedPassword: hash,
			password:       password[:len(password)-1],
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ComparePassword(tt.hashedPassword, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComparePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordHasher_ComparePassword_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher(bcrypt.DefaultCost)

	tests := []struct {
		name           string
		hashedPassword string
		password       string
	}{
		{
			name:           "invalid hash format",
			hashedPassword: "not-a-valid-hash",
			password:       "password",
		},
		{
			name:           "empty hash",
			hashedPassword: "",
			password:       "password",
		},
		{
			name:           "corrupted hash",
			hashedPassword: "$2a$10$invalid",
			password:       "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ComparePassword(tt.hashedPassword, tt.password)
			if err == nil {
				t.Error("Expected error for invalid hash")
			}
		})
	}
}

func TestValidatePasswordStrength_MinLength(t *testing.T) {
	tests := []struct {
		name              string
		password          string
		minLength         int
		requireComplexity bool
		wantErr           bool
		errContains       string
	}{
		{
			name:              "meets minimum length",
			password:          "12345678",
			minLength:         8,
			requireComplexity: false,
			wantErr:           false,
		},
		{
			name:              "below minimum length",
			password:          "1234567",
			minLength:         8,
			requireComplexity: false,
			wantErr:           true,
			errContains:       "at least 8 characters",
		},
		{
			name:              "exactly minimum length",
			password:          "12345678",
			minLength:         8,
			requireComplexity: false,
			wantErr:           false,
		},
		{
			name:              "above minimum length",
			password:          "123456789",
			minLength:         8,
			requireComplexity: false,
			wantErr:           false,
		},
		{
			name:              "empty password",
			password:          "",
			minLength:         8,
			requireComplexity: false,
			wantErr:           true,
			errContains:       "at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, tt.minLength, tt.requireComplexity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestValidatePasswordStrength_Complexity(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "meets all complexity requirements",
			password: "SecureP@ssw0rd123",
			wantErr:  false,
		},
		{
			name:        "missing uppercase",
			password:    "securep@ssw0rd123",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "missing lowercase",
			password:    "SECUREP@SSW0RD123",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "missing digit",
			password:    "SecureP@ssword",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "missing special character",
			password:    "SecurePassword123",
			wantErr:     true,
			errContains: "special character",
		},
		{
			name:        "only lowercase",
			password:    "password",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "only uppercase",
			password:    "PASSWORD",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "only digits",
			password:    "12345678",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:        "only special characters",
			password:    "!@#$%^&*",
			wantErr:     true,
			errContains: "uppercase, lowercase, and numeric",
		},
		{
			name:     "various special characters",
			password: "SecureP!ssw0rd",
			wantErr:  false,
		},
		{
			name:     "special characters: !@#$%",
			password: "SecureP@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "special characters: ^&*()",
			password: "SecureP^ssw0rd&",
			wantErr:  false,
		},
		{
			name:     "special characters: -_=+",
			password: "SecureP_ssw0rd=",
			wantErr:  false,
		},
		{
			name:     "special characters: []{}",
			password: "SecureP[ssw0rd]",
			wantErr:  false,
		},
		{
			name:     "special characters: |\\:;",
			password: "SecureP|ssw0rd\\",
			wantErr:  false,
		},
		{
			name:     "special characters: '<>",
			password: "SecureP<ssw0rd>",
			wantErr:  false,
		},
		{
			name:     "special characters: ,./",
			password: "SecureP,ssw0rd.",
			wantErr:  false,
		},
		{
			name:     "special characters: ?",
			password: "SecureP?ssw0rd1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, 8, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestValidatePasswordStrength_NoComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "only lowercase letters - valid when complexity not required",
			password: "password",
			wantErr:  false,
		},
		{
			name:     "only uppercase letters - valid when complexity not required",
			password: "PASSWORD",
			wantErr:  false,
		},
		{
			name:     "only digits - valid when complexity not required",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "mixed without special - valid when complexity not required",
			password: "Password123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, 8, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordStrength_Combined(t *testing.T) {
	tests := []struct {
		name              string
		password          string
		minLength         int
		requireComplexity bool
		wantErr           bool
	}{
		{
			name:              "valid with complexity",
			password:          "SecureP@ssw0rd",
			minLength:         8,
			requireComplexity: true,
			wantErr:           false,
		},
		{
			name:              "valid without complexity",
			password:          "simplepassword",
			minLength:         8,
			requireComplexity: false,
			wantErr:           false,
		},
		{
			name:              "too short but meets complexity",
			password:          "Sec@1",
			minLength:         8,
			requireComplexity: true,
			wantErr:           true,
		},
		{
			name:              "long enough but missing complexity",
			password:          "longpasswordbutnocomplexity",
			minLength:         8,
			requireComplexity: true,
			wantErr:           true,
		},
		{
			name:              "exactly 8 chars with complexity",
			password:          "Sec@1234",
			minLength:         8,
			requireComplexity: true,
			wantErr:           false,
		},
		{
			name:              "very long with complexity",
			password:          strings.Repeat("SecureP@ssw0rd", 5),
			minLength:         8,
			requireComplexity: true,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, tt.minLength, tt.requireComplexity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordHasher_Integration(t *testing.T) {
	// Test the full workflow: hash, store, retrieve, compare
	hasher := NewPasswordHasher(12)
	originalPassword := "MySecureP@ssw0rd123"

	// Step 1: Validate password strength
	err := ValidatePasswordStrength(originalPassword, 8, true)
	if err != nil {
		t.Fatalf("Password validation failed: %v", err)
	}

	// Step 2: Hash the password
	hashedPassword, err := hasher.HashPassword(originalPassword)
	if err != nil {
		t.Fatalf("Password hashing failed: %v", err)
	}

	// Step 3: Simulate storage and retrieval (just use the hash as-is)
	retrievedHash := hashedPassword

	// Step 4: Compare with correct password
	err = hasher.ComparePassword(retrievedHash, originalPassword)
	if err != nil {
		t.Errorf("Password comparison failed for correct password: %v", err)
	}

	// Step 5: Compare with incorrect password
	err = hasher.ComparePassword(retrievedHash, "WrongPassword123!")
	if err == nil {
		t.Error("Expected error when comparing with incorrect password")
	}
}

func BenchmarkPasswordHasher_HashPassword(b *testing.B) {
	hasher := NewPasswordHasher(12)
	password := "SecureP@ssw0rd123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.HashPassword(password)
	}
}

func BenchmarkPasswordHasher_ComparePassword(b *testing.B) {
	hasher := NewPasswordHasher(12)
	password := "SecureP@ssw0rd123"
	hash, _ := hasher.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.ComparePassword(hash, password)
	}
}

func BenchmarkValidatePasswordStrength(b *testing.B) {
	password := "SecureP@ssw0rd123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePasswordStrength(password, 8, true)
	}
}
