package utils

import (
	"regexp"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"8 characters", 8},
		{"16 characters", 16},
		{"32 characters", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRandomString(tt.length)
			if len(result) != tt.length {
				t.Errorf("GenerateRandomString(%d) length = %d, want %d", tt.length, len(result), tt.length)
			}
		})
	}

	// Test only valid characters
	t.Run("generates only valid characters", func(t *testing.T) {
		validCharset := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
		for i := 0; i < 10; i++ {
			result := GenerateRandomString(8)
			if !validCharset.MatchString(result) {
				t.Errorf("GenerateRandomString(%d) contains invalid characters: %s", 8, result)
			}
		}
	})

	// Test uniqueness
	t.Run("generates unique strings", func(t *testing.T) {
		results := make(map[string]bool)
		for i := 0; i < 100; i++ {
			result := GenerateRandomString(8)
			if results[result] {
				t.Errorf("GenerateRandomString generated duplicate: %s", result)
			}
			results[result] = true
		}
	})
}
