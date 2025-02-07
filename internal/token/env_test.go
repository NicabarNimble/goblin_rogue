package token

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEnvStorage(t *testing.T) {
	storage := NewEnvStorage()
	ctx := context.Background()

	// Clean up any existing test environment variables
	cleanup := func() {
		for _, env := range os.Environ() {
			if len(env) > len(EnvPrefix) && env[:len(EnvPrefix)] == EnvPrefix {
				os.Unsetenv(env[:strings.Index(env, "=")])
			}
		}
	}
	cleanup()
	defer cleanup()

	t.Run("Store and Retrieve", func(t *testing.T) {
		cleanup()
		token, err := NewToken("test-token", time.Now().Add(time.Hour), "repo,workflow")
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		err = storage.Store(ctx, "test-key", *token)
		if err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		retrieved, err := storage.Retrieve(ctx, "test-key")
		if err != nil {
			t.Fatalf("Failed to retrieve token: %v", err)
		}

		if retrieved.Value != token.Value {
			t.Errorf("Retrieved token value mismatch: got %s, want %s", retrieved.Value, token.Value)
		}
		if retrieved.Scope != token.Scope {
			t.Errorf("Retrieved token scope mismatch: got %s, want %s", retrieved.Scope, token.Scope)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cleanup()
		token, err := NewToken("delete-test-token", time.Now().Add(time.Hour), "repo")
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		key := "delete-test-key"
		if err := storage.Store(ctx, key, *token); err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		if err := storage.Delete(ctx, key); err != nil {
			t.Fatalf("Failed to delete token: %v", err)
		}

		_, err = storage.Retrieve(ctx, key)
		if !errors.Is(err, ErrTokenNotFound) {
			t.Errorf("Expected ErrTokenNotFound, got: %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		cleanup()
		tokens := make(map[Provider]*Token)
		
		// Create tokens for different providers
		providers := []Provider{ProviderGitHub, ProviderGitLab}
		for _, provider := range providers {
			token, err := NewToken(
				"token-"+string(provider),
				time.Now().Add(time.Hour),
				"repo",
			)
			if err != nil {
				t.Fatalf("Failed to create token for %s: %v", provider, err)
			}
			tokens[provider] = token
			if err := storage.Store(ctx, string(provider), *token); err != nil {
				t.Fatalf("Failed to store token %s: %v", provider, err)
			}
		}

		keys, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(keys) != len(tokens) {
			t.Errorf("Expected %d keys, got %d", len(tokens), len(keys))
		}

		// Verify all keys are present
		for _, key := range keys {
			if _, exists := tokens[Provider(key)]; !exists {
				t.Errorf("Unexpected key in list: %s", key)
			}
		}
	})

	t.Run("Invalid Token", func(t *testing.T) {
		cleanup()
		token := Token{
			Value:     "",  // Invalid: empty value
			ExpiresAt: time.Now().Add(-time.Hour), // Invalid: already expired
		}

		key := "invalid-token-key"
		err := storage.Store(ctx, key, token)
		if err == nil {
			t.Error("Expected error storing invalid token, got nil")
		}
		if !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("Expected ErrTokenInvalid, got: %v", err)
		}
	})

	t.Run("Key Formatting", func(t *testing.T) {
		cleanup()
		token, err := NewToken("format-test", time.Now().Add(time.Hour), "repo")
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		key := "test/key.with-special@chars"
		if err := storage.Store(ctx, key, *token); err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		retrieved, err := storage.Retrieve(ctx, key)
		if err != nil {
			t.Fatalf("Failed to retrieve token with special chars key: %v", err)
		}

		if retrieved.Value != token.Value {
			t.Errorf("Retrieved token value mismatch for special chars key")
		}

		// Verify the key was properly formatted
		formattedKey := storage.FormatEnvKey(key)
		if !strings.HasPrefix(formattedKey, EnvPrefix) {
			t.Errorf("Formatted key missing prefix: %s", formattedKey)
		}
		if strings.ContainsAny(formattedKey[len(EnvPrefix):], "@/.-") {
			t.Errorf("Formatted key contains special characters: %s", formattedKey)
		}
	})

	t.Run("JSON Storage Format", func(t *testing.T) {
		cleanup()
		token, err := NewToken("test-token-value", time.Now().Add(time.Hour), "repo")
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		key := "json-test"
		if err := storage.Store(ctx, key, *token); err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		// Get the raw environment variable
		envKey := storage.FormatEnvKey(key)
		rawJSON := os.Getenv(envKey)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		if err := json.Unmarshal([]byte(rawJSON), &jsonMap); err != nil {
			t.Fatalf("Failed to unmarshal stored JSON: %v", err)
		}

		// Check all expected fields are present
		expectedFields := []string{"Value", "ExpiresAt", "Scope", "CreatedAt"}
		for _, field := range expectedFields {
			if _, ok := jsonMap[field]; !ok {
				t.Errorf("Missing expected field in JSON: %s", field)
			}
		}

		// Verify the stored value matches
		if value, ok := jsonMap["Value"].(string); !ok || value != token.Value {
			t.Errorf("Token value mismatch in JSON, got %v, want %v", value, token.Value)
		}

		// Verify we can retrieve the token
		retrieved, err := storage.Retrieve(ctx, key)
		if err != nil {
			t.Fatalf("Failed to retrieve token: %v", err)
		}
		if retrieved.Value != token.Value {
			t.Errorf("Retrieved token value mismatch: got %v, want %v", retrieved.Value, token.Value)
		}
	})

	t.Run("Close", func(t *testing.T) {
		cleanup()
		if err := storage.Close(ctx); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})
}
