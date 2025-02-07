package token

import (
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{
			name: "non-expiring token",
			token: Token{
				Value:     "test-token",
				ExpiresAt: time.Time{}, // zero value
			},
			expected: false,
		},
		{
			name: "expired token",
			token: Token{
				Value:     "test-token",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "valid token",
			token: Token{
				Value:     "test-token",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExpired(tt.token); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{
			name: "valid token",
			token: Token{
				Value:     "test-token",
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Scope:     "repo",
				CreatedAt: time.Now(),
			},
			expected: true,
		},
		{
			name: "empty token",
			token: Token{
				Value:     "",
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Scope:     "repo",
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "non-expiring valid token",
			token: Token{
				Value:     "test-token",
				ExpiresAt: time.Time{},
				Scope:     "repo",
				CreatedAt: time.Now(),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.token); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
