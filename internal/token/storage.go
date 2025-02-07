// Package token provides token management functionality for git operations.
//
// Storage Strategy
//
// The package implements two primary token storage mechanisms:
//
// 1. Environment Variables (Primary Production Storage):
//   - Recommended for production, headless, and Docker environments
//   - Uses GIT_TOKEN_* prefixed environment variables
//   - Ideal for automated/LLM interactions
//   - No system dependencies or user interaction required
//   - Cross-platform compatible
//
// 2. Memory Storage (Testing/Ephemeral Use):
//   - Suitable for testing and short-lived operations
//   - No persistence between program restarts
//   - Useful for integration tests and local development
//
// Environment Variable Usage:
//   export GIT_TOKEN_GITHUB="your-token-here"  // For GitHub operations
//   export GIT_TOKEN_GITLAB="your-token-here"  // For GitLab operations
//
// Note: System keychain integration was considered but intentionally not implemented
// to optimize for headless and containerized environments where user interaction
// and system integration would be problematic.
package token

import (
	"context"
	"errors"
	"time"
)


// Common errors that may be returned by token operations
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenExpired       = errors.New("token has expired")
	ErrStorageUnavailable = errors.New("token storage is unavailable")
	ErrTokenRefreshFailed = errors.New("token refresh failed")
)

// Token represents an authentication token with metadata
type Token struct {
	// Value is the actual token string
	Value string `json:"Value"` // Store value in JSON for env storage

	// ExpiresAt indicates when the token will expire
	// Zero value means the token does not expire
	ExpiresAt time.Time `json:"ExpiresAt"`

	// Scope defines the permissions granted to this token
	Scope string `json:"Scope"`

	// CreatedAt indicates when the token was created/stored
	CreatedAt time.Time `json:"CreatedAt"`
}

// NewToken creates a new token with validation
func NewToken(value string, expiresAt time.Time, scope string) (*Token, error) {
	if value == "" {
		return nil, errors.New("token value cannot be empty")
	}

	token := &Token{
		Value:     value,
		ExpiresAt: expiresAt,
		Scope:     scope,
		CreatedAt: time.Now(),
	}

	if !IsValid(*token) {
		return nil, ErrTokenInvalid
	}

	return token, nil
}


// Storage defines the interface for token storage implementations
type Storage interface {
	// Store saves a token with the given key
	// If a token already exists for the key, it will be overwritten
	Store(ctx context.Context, key string, token Token) error

	// Retrieve gets a token by its key
	// Returns ErrTokenNotFound if the token doesn't exist
	Retrieve(ctx context.Context, key string) (Token, error)

	// Delete removes a token by its key
	// Returns nil if the token was successfully deleted or didn't exist
	Delete(ctx context.Context, key string) error

	// List returns all stored token keys
	// The returned slice will be empty if no tokens are stored
	List(ctx context.Context) ([]string, error)

	// Close performs any necessary cleanup
	Close(ctx context.Context) error
}

// Validator provides methods to validate tokens
type Validator interface {
	// Validate checks if a token is valid
	// Returns nil if the token is valid, otherwise returns an error
	// explaining why the token is invalid
	Validate(ctx context.Context, token *Token) error
}

// IsExpired checks if a token has expired
func IsExpired(token Token) bool {
	if token.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(token.ExpiresAt)
}

// IsValid performs basic validation of a token
func IsValid(token Token) bool {
	// Only validate that the token has a non-empty value
	return token.Value != ""
}
