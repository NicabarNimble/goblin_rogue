package token

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockRefreshHandler implements RefreshHandler for testing
type mockRefreshHandler struct {
	refreshFunc func(ctx context.Context, current Token) (Token, error)
}

func (m *mockRefreshHandler) RefreshToken(ctx context.Context, current Token) (Token, error) {
	return m.refreshFunc(ctx, current)
}

func TestTokenManager_CheckHealth(t *testing.T) {
	storage := NewMemoryStorage()
	handler := &mockRefreshHandler{}
	config := DefaultRefreshConfig

	manager := NewTokenManager(storage, handler, config)
	ctx := context.Background()
	defer storage.Close(ctx)

	// Create test tokens
	healthyToken, err := NewToken("valid-token", time.Now().Add(48*time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create healthy token: %v", err)
	}

	expiringSoonToken, err := NewToken("expiring-token", time.Now().Add(1*time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create expiring token: %v", err)
	}

	expiredToken, err := NewToken("expired-token", time.Now().Add(-1*time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	tests := []struct {
		name      string
		token     *Token
		wantError error
	}{
		{
			name:      "healthy token",
			token:     healthyToken,
			wantError: nil,
		},
		{
			name:      "expiring soon token",
			token:     expiringSoonToken,
			wantError: ErrTokenExpired,
		},
		{
			name:      "expired token",
			token:     expiredToken,
			wantError: ErrTokenExpired,
		},
		{
			name:      "invalid token",
			token:     nil, // Don't store any token for this test
			wantError: ErrTokenNotFound, // Checking health of non-existent token should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("test-token-%s", tt.name)
			if tt.token != nil {
				if err := storage.Store(ctx, key, *tt.token); err != nil {
					t.Fatalf("Failed to store token: %v", err)
				}
			}

			err := manager.CheckHealth(ctx, key)
			if !errors.Is(err, tt.wantError) {
				t.Errorf("CheckHealth() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTokenManager_RefreshToken(t *testing.T) {
	storage := NewMemoryStorage()
	config := RefreshConfig{
		MinValidTime:    24 * time.Hour,
		RetryAttempts:   2,
		RetryDelay:      time.Millisecond,
		RefreshTimeout:  100 * time.Millisecond,
	}

	ctx := context.Background()
	defer storage.Close(ctx)

	currentToken, err := NewToken("old-token", time.Now().Add(1*time.Hour), "repo")
	if err != nil {
		t.Fatalf("Failed to create current token: %v", err)
	}

	tests := []struct {
		name         string
		setupHandler func() *mockRefreshHandler
		wantError    string
		timeout      time.Duration
	}{
		{
			name: "successful refresh",
			setupHandler: func() *mockRefreshHandler {
				return &mockRefreshHandler{
					refreshFunc: func(ctx context.Context, current Token) (Token, error) {
						token, err := NewToken("new-token", time.Now().Add(48*time.Hour), "repo")
						if err != nil {
							return Token{}, fmt.Errorf("failed to create new token: %w", err)
						}
						return *token, nil
					},
				}
			},
			wantError: "",
		},
		{
			name: "refresh fails all retries",
			setupHandler: func() *mockRefreshHandler {
				return &mockRefreshHandler{
					refreshFunc: func(ctx context.Context, current Token) (Token, error) {
						return Token{}, errors.New("refresh failed")
					},
				}
			},
			wantError: "token refresh failed after 2 attempts: refresh failed",
		},
		{
			name: "context timeout",
			setupHandler: func() *mockRefreshHandler {
				return &mockRefreshHandler{
					refreshFunc: func(ctx context.Context, current Token) (Token, error) {
						select {
						case <-time.After(50 * time.Millisecond):
							token, err := NewToken("new-token", time.Now().Add(48*time.Hour), "repo")
							if err != nil {
								return Token{}, fmt.Errorf("failed to create new token: %w", err)
							}
							return *token, nil
						case <-ctx.Done():
							return Token{}, ctx.Err()
						}
					},
				}
			},
			timeout:   10 * time.Millisecond,
			wantError: "context deadline exceeded",
		},
		{
			name: "invalid refresh token",
			setupHandler: func() *mockRefreshHandler {
				return &mockRefreshHandler{
					refreshFunc: func(ctx context.Context, current Token) (Token, error) {
						return Token{Value: ""}, nil // Invalid token
					},
				}
			},
			wantError: "refreshed token is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			manager := NewTokenManager(storage, handler, config)

			key := fmt.Sprintf("test-token-%s", tt.name)
			if err := storage.Store(ctx, key, *currentToken); err != nil {
				t.Fatalf("Failed to store token: %v", err)
			}

			testCtx := ctx
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				testCtx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			}

			err := manager.RefreshToken(testCtx, key)
			if tt.wantError == "" {
				if err != nil {
					t.Errorf("RefreshToken() unexpected error: %v", err)
				}
				// Verify the token was actually updated
				newToken, err := storage.Retrieve(ctx, key)
				if err != nil {
					t.Fatalf("Failed to retrieve token after refresh: %v", err)
				}
				if newToken.Value == currentToken.Value {
					t.Error("Token was not updated after successful refresh")
				}
			} else {
				if err == nil {
					t.Errorf("RefreshToken() expected error, got nil")
				} else {
					if !strings.Contains(err.Error(), tt.wantError) {
						t.Errorf("RefreshToken() error = %v, want error containing %q", err, tt.wantError)
					}
				}
			}
		})
	}
}

func TestTokenManager_ConcurrentMonitoring(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	defer storage.Close(ctx)

	var refreshCount int32
	handler := &mockRefreshHandler{
		refreshFunc: func(ctx context.Context, current Token) (Token, error) {
			atomic.AddInt32(&refreshCount, 1)
			token, err := NewToken(
				fmt.Sprintf("refreshed-token-%d", atomic.LoadInt32(&refreshCount)),
				time.Now().Add(48*time.Hour),
				"repo",
			)
			if err != nil {
				return Token{}, fmt.Errorf("failed to create refreshed token: %w", err)
			}
			return *token, nil
		},
	}

	config := RefreshConfig{
		MinValidTime:    24 * time.Hour,
		RefreshInterval: 10 * time.Millisecond,
		RetryAttempts:   1,
		RetryDelay:      time.Millisecond,
		RefreshTimeout:  50 * time.Millisecond,
	}

	manager := NewTokenManager(storage, handler, config)
	monitorCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Start monitoring multiple tokens concurrently
	const numTokens = 5
	var wg sync.WaitGroup
	wg.Add(numTokens)

	// Use a channel to collect errors from goroutines
	errChan := make(chan error, numTokens*3) // *3 for potential errors in create, store, and monitor

	for i := 0; i < numTokens; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("monitored-token-%d", id)
			
			// Create an expiring token
			token, err := NewToken(
				fmt.Sprintf("original-token-%d", id),
				time.Now().Add(1*time.Hour),
				"repo",
			)
			if err != nil {
				errChan <- fmt.Errorf("failed to create token %d: %v", id, err)
				return
			}

			if err := storage.Store(ctx, key, *token); err != nil {
				errChan <- fmt.Errorf("failed to store token %d: %v", id, err)
				return
			}

			// Start monitoring
			if err := manager.StartMonitoring(monitorCtx, key); err != nil {
				errChan <- fmt.Errorf("failed to start monitoring token %d: %v", id, err)
				return
			}

			// Wait for monitoring to run
			<-monitorCtx.Done()

			// Stop monitoring
			manager.StopMonitoring(key)

			// Verify the token was updated
			finalToken, err := storage.Retrieve(ctx, key)
			if err != nil {
				errChan <- fmt.Errorf("failed to retrieve final token %d: %v", id, err)
				return
			}
			if finalToken.Value == token.Value {
				errChan <- fmt.Errorf("token %d was not updated during monitoring", id)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		t.Errorf("Test encountered errors:\n%s", strings.Join(errors, "\n"))
	}

	// Check if any tokens were refreshed
	if atomic.LoadInt32(&refreshCount) == 0 {
		t.Error("No tokens were refreshed during monitoring")
	}
}
