package token

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_Store(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	validToken, err := NewToken("valid-token", time.Now().Add(time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create valid token: %v", err)
	}

	tests := []struct {
		name      string
		key       string
		token     Token
		wantError error
	}{
		{
			name:      "valid token",
			key:       "test1",
			token:     *validToken,
			wantError: nil,
		},
		{
			name: "invalid token",
			key:  "test2",
			token: Token{
				Value:     "", // Invalid: empty value
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantError: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Store(ctx, tt.key, tt.token)
			if err != tt.wantError {
				t.Errorf("Store() error = %v, want %v", err, tt.wantError)
			}

			if err == nil {
				// Verify token was stored
				stored, err := storage.Retrieve(ctx, tt.key)
				if err != nil {
					t.Errorf("Retrieve() error = %v", err)
				}
				if stored.Value != tt.token.Value {
					t.Errorf("Stored token value = %v, want %v", stored.Value, tt.token.Value)
				}
			}
		})
	}
}

func TestMemoryStorage_Retrieve(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	validToken, err := NewToken("valid-token", time.Now().Add(time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create valid token: %v", err)
	}

	expiredToken, err := NewToken("expired-token", time.Now().Add(-time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	// Store test tokens
	_ = storage.Store(ctx, "valid", *validToken)
	_ = storage.Store(ctx, "expired", *expiredToken)

	tests := []struct {
		name      string
		key       string
		wantToken Token
		wantError error
	}{
		{
			name:      "valid token",
			key:       "valid",
			wantToken: *validToken,
			wantError: nil,
		},
		{
			name:      "expired token",
			key:       "expired",
			wantToken: Token{},
			wantError: ErrTokenExpired,
		},
		{
			name:      "non-existent token",
			key:       "missing",
			wantToken: Token{},
			wantError: ErrTokenNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := storage.Retrieve(ctx, tt.key)
			if err != tt.wantError {
				t.Errorf("Retrieve() error = %v, want %v", err, tt.wantError)
			}
			if err == nil && token.Value != tt.wantToken.Value {
				t.Errorf("Retrieve() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	// Store a token
	token, err := NewToken("test-token", time.Now().Add(time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	_ = storage.Store(ctx, "test", *token)

	// Delete the token
	err = storage.Delete(ctx, "test")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify token was deleted
	_, err = storage.Retrieve(ctx, "test")
	if err != ErrTokenNotFound {
		t.Errorf("Retrieve() after delete error = %v, want %v", err, ErrTokenNotFound)
	}

	// Delete non-existent token should not error
	err = storage.Delete(ctx, "missing")
	if err != nil {
		t.Errorf("Delete() non-existent token error = %v, want nil", err)
	}
}

func TestMemoryStorage_List(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	// Store some tokens
	tokens := map[string]Token{}
	for i := 1; i <= 3; i++ {
		token, err := NewToken(fmt.Sprintf("value%d", i), time.Now().Add(time.Hour), "repo")
		if err != nil {
			t.Fatalf("Failed to create token %d: %v", i, err)
		}
		key := fmt.Sprintf("token%d", i)
		tokens[key] = *token
		_ = storage.Store(ctx, key, *token)
	}

	// List tokens
	keys, err := storage.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(keys) != len(tokens) {
		t.Errorf("List() returned %d keys, want %d", len(keys), len(tokens))
	}

	// Verify all keys are present
	for _, k := range keys {
		if _, exists := tokens[k]; !exists {
			t.Errorf("List() returned unexpected key: %s", k)
		}
	}
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store some tokens
	token, err := NewToken("test-token", time.Now().Add(time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}
	_ = storage.Store(ctx, "test", *token)

	// Close storage
	if err := storage.Close(ctx); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify storage is empty
	keys, err := storage.List(ctx)
	if err != nil {
		t.Errorf("List() after close error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Storage not empty after close, contains %d keys", len(keys))
	}
}

func TestToken_JSONMarshaling(t *testing.T) {
	token, err := NewToken("secret-value", time.Now().Add(time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Test marshaling
	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Failed to marshal token: %v", err)
	}

	// Verify token value is redacted
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal token JSON: %v", err)
	}

	if value, ok := jsonMap["value"].(string); !ok || value != "REDACTED" {
		t.Errorf("Token value not properly redacted in JSON, got %v", jsonMap["value"])
	}

	// Test unmarshaling
	var unmarshaledToken Token
	if err := json.Unmarshal(data, &unmarshaledToken); err != nil {
		t.Fatalf("Failed to unmarshal token: %v", err)
	}

	// Verify fields except Value (which is redacted)
	if !unmarshaledToken.ExpiresAt.Equal(token.ExpiresAt) {
		t.Errorf("ExpiresAt not preserved in JSON roundtrip")
	}
	if unmarshaledToken.Scope != token.Scope {
		t.Errorf("Scope not preserved in JSON roundtrip")
	}
}

func TestMemoryStorage_Concurrent(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("token-%d-%d", id, j)
				token, err := NewToken(
					fmt.Sprintf("value-%d-%d", id, j),
					time.Now().Add(time.Hour),
					"repo",
				)
				if err != nil {
					t.Errorf("Failed to create token: %v", err)
					continue
				}

				// Store
				if err := storage.Store(ctx, key, *token); err != nil {
					t.Errorf("Concurrent Store() error = %v", err)
				}

				// Retrieve
				if _, err := storage.Retrieve(ctx, key); err != nil {
					t.Errorf("Concurrent Retrieve() error = %v", err)
				}

				// Delete
				if err := storage.Delete(ctx, key); err != nil {
					t.Errorf("Concurrent Delete() error = %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
}
