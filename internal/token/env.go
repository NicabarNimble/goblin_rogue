package token

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// EnvPrefix is the prefix used for all token environment variables
	EnvPrefix = "GIT_TOKEN_"
)

// EnvStorage implements Storage using environment variables.
// This is the primary recommended storage implementation for production use,
// especially in headless and containerized environments. It stores tokens as
// JSON-encoded strings in environment variables with the GIT_TOKEN_ prefix.
//
// Example Usage:
//   export GIT_TOKEN_GITHUB='{"Value":"ghp_abc...","Scope":"repo,workflow"}'
//   export GIT_TOKEN_GITLAB='{"Value":"glpat_xyz...","Scope":"api"}'
//
// In Docker:
//   docker run -e GIT_TOKEN_GITHUB='{"Value":"..."}'
//
// Benefits:
//   - No system dependencies or user interaction required
//   - Works consistently across platforms
//   - Ideal for automation and LLM interaction
//   - Native support in CI/CD and container environments
type EnvStorage struct{}

// NewEnvStorage creates a new environment variable-based token storage
func NewEnvStorage() *EnvStorage {
	return &EnvStorage{}
}

// Store saves a token with the given key as an environment variable
// The token is stored as a JSON string to preserve metadata
func (e *EnvStorage) Store(ctx context.Context, key string, token Token) error {
	if !IsValid(token) {
		return ErrTokenInvalid
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	envKey := e.FormatEnvKey(key)
	if err := os.Setenv(envKey, string(data)); err != nil {
		return fmt.Errorf("failed to set environment variable: %w", err)
	}

	return nil
}

// Retrieve gets a token by its key from environment variables
func (e *EnvStorage) Retrieve(ctx context.Context, key string) (Token, error) {
	envKey := e.FormatEnvKey(key)
	data := os.Getenv(envKey)
	if data == "" {
		return Token{}, ErrTokenNotFound
	}

	var token Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return Token{}, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Validate the token before returning
	if !IsValid(token) {
		return Token{}, ErrTokenInvalid
	}

	// Check expiration
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return Token{}, ErrTokenExpired
	}

	return token, nil
}

// Delete removes a token by unsetting its environment variable
func (e *EnvStorage) Delete(ctx context.Context, key string) error {
	envKey := e.FormatEnvKey(key)
	if err := os.Unsetenv(envKey); err != nil {
		return fmt.Errorf("failed to unset environment variable: %w", err)
	}
	return nil
}

// List returns all stored token keys from environment variables
func (e *EnvStorage) List(ctx context.Context) ([]string, error) {
	var keys []string
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) > 0 {
			key := parts[0]
			// Only include keys that start with prefix and don't end with _VALUE
			if strings.HasPrefix(key, EnvPrefix) && !strings.HasSuffix(key, "_VALUE") {
				// Strip the prefix to get the original key
				keys = append(keys, strings.TrimPrefix(key, EnvPrefix))
			}
		}
	}
	return keys, nil
}

// FormatEnvKey converts a token key into an environment variable name
// This is exported to allow users to predict and verify environment variable names
func (e *EnvStorage) FormatEnvKey(key string) string {
	// Convert the key to uppercase and replace any non-alphanumeric characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, strings.ToUpper(key))

	return EnvPrefix + sanitized
}

// Close implements Storage.Close
func (e *EnvStorage) Close(ctx context.Context) error {
	// Nothing to clean up for environment variables
	return nil
}
