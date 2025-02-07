package token

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RefreshConfig holds configuration for token refresh behavior
type RefreshConfig struct {
	// MinValidTime is the minimum time a token should be valid for
	// If a token's remaining validity period is less than this,
	// it will be considered unhealthy and trigger a refresh
	MinValidTime time.Duration

	// RefreshInterval is how often to check token health
	RefreshInterval time.Duration

	// RetryAttempts is the number of times to retry refresh on failure
	RetryAttempts int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// RefreshTimeout is the maximum time to wait for a refresh operation
	RefreshTimeout time.Duration

	// ProgressCallback is called to report progress during refresh attempts
	ProgressCallback func(message string)
}

// DefaultRefreshConfig provides sensible defaults for token refresh
var DefaultRefreshConfig = RefreshConfig{
	MinValidTime:     24 * time.Hour,    // Refresh when less than 24h validity remains
	RefreshInterval:  1 * time.Hour,     // Check health every hour
	RetryAttempts:    2,                 // Retry 2 times on failure
	RetryDelay:       1 * time.Second,   // Wait 1s between retries
	RefreshTimeout:   30 * time.Second,  // Maximum time for refresh operation
	ProgressCallback: func(message string) {
		fmt.Print("\r" + message)
	},
}

// RefreshHandler defines the interface for token refresh operations
type RefreshHandler interface {
	// RefreshToken is called when a token needs to be refreshed
	// It should return a new valid token or an error
	RefreshToken(ctx context.Context, current Token) (Token, error)
}

// TokenManager handles token health monitoring and automatic refresh
type TokenManager struct {
	storage   Storage
	handler   RefreshHandler
	config    RefreshConfig
	mu        sync.RWMutex
	monitors  map[string]context.CancelFunc
}

// NewTokenManager creates a new TokenManager with the given configuration
func NewTokenManager(storage Storage, handler RefreshHandler, config RefreshConfig) *TokenManager {
	return &TokenManager{
		storage:  storage,
		handler:  handler,
		config:   config,
		monitors: make(map[string]context.CancelFunc),
	}
}

// CheckHealth verifies if a token is healthy
// A token is considered healthy if:
// 1. It exists and is valid
// 2. It has sufficient remaining validity time (>= MinValidTime)
func (tm *TokenManager) CheckHealth(ctx context.Context, key string) error {
	token, err := tm.storage.Retrieve(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Check if token exists and is basically valid
	if !IsValid(token) {
		return ErrTokenInvalid
	}

	// Check remaining validity time
	if !token.ExpiresAt.IsZero() {
		remainingTime := time.Until(token.ExpiresAt)
		if remainingTime < tm.config.MinValidTime {
			return ErrTokenExpired
		}
	}

	return nil
}

// RefreshToken attempts to refresh a token with proper timeout handling
func (tm *TokenManager) RefreshToken(ctx context.Context, key string) error {
	// Create a timeout context for the entire refresh operation
	refreshCtx, cancel := context.WithTimeout(ctx, tm.config.RefreshTimeout)
	defer cancel()

	currentToken, err := tm.storage.Retrieve(refreshCtx, key)
	if err != nil {
		return fmt.Errorf("failed to retrieve token for refresh: %w", err)
	}

	// Attempt refresh with retries
	var newToken Token
	var refreshErr error

	if tm.config.ProgressCallback != nil {
		tm.config.ProgressCallback("Attempting to retrieve GitHub token...")
	}

	for attempt := 0; attempt <= tm.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-refreshCtx.Done():
				return fmt.Errorf("refresh operation timed out: %w", refreshCtx.Err())
			case <-time.After(tm.config.RetryDelay):
				if tm.config.ProgressCallback != nil {
					tm.config.ProgressCallback("Attempting to retrieve GitHub token...")
				}
			}
		}

		// Create a context for this specific attempt
		attemptCtx, attemptCancel := context.WithTimeout(refreshCtx, tm.config.RefreshTimeout/time.Duration(tm.config.RetryAttempts+1))
		newToken, refreshErr = tm.handler.RefreshToken(attemptCtx, currentToken)
		attemptCancel()

		if refreshErr == nil {
			break
		}

		if refreshCtx.Err() != nil {
			return fmt.Errorf("refresh operation cancelled: %w", refreshCtx.Err())
		}
	}

	if refreshErr != nil {
		return fmt.Errorf("token refresh failed after %d attempts: %w", tm.config.RetryAttempts, refreshErr)
	}

	// Validate the new token
	if !IsValid(newToken) {
		return fmt.Errorf("refreshed token is invalid: %w", ErrTokenInvalid)
	}

	// Store the new token with timeout context
	if err := tm.storage.Store(refreshCtx, key, newToken); err != nil {
		return fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return nil
}

// StartMonitoring begins monitoring a token's health
func (tm *TokenManager) StartMonitoring(ctx context.Context, key string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Cancel any existing monitor for this key
	if cancel, exists := tm.monitors[key]; exists {
		cancel()
	}

	// Create a new context for this monitor
	monitorCtx, cancel := context.WithCancel(ctx)
	tm.monitors[key] = cancel

	// Start monitoring in a goroutine
	go tm.monitor(monitorCtx, key)

	return nil
}

// StopMonitoring stops monitoring a token's health
func (tm *TokenManager) StopMonitoring(key string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if cancel, exists := tm.monitors[key]; exists {
		cancel()
		delete(tm.monitors, key)
	}
}

// monitor is the internal monitoring loop for a token
func (tm *TokenManager) monitor(ctx context.Context, key string) {
	ticker := time.NewTicker(tm.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create a timeout context for health check and refresh operations
			checkCtx, cancel := context.WithTimeout(ctx, tm.config.RefreshTimeout)
			
			// Check token health
			if err := tm.CheckHealth(checkCtx, key); err != nil {
				// Log the health check error but continue to refresh attempt
				if err != ErrTokenExpired {
					fmt.Printf("Token health check failed for %s: %v\n", key, err)
				}
				
				// Attempt to refresh the token if it's unhealthy
				if err := tm.RefreshToken(checkCtx, key); err != nil {
					fmt.Printf("Token refresh failed for %s: %v\n", key, err)
				}
			}
			
			cancel() // Clean up the timeout context
		}
	}
}
