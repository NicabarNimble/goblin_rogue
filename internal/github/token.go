package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/token"
)

// Required scopes for GitHub Actions operations
const (
	ScopeRepo      = "repo"
	ScopeWorkflow  = "workflow"
)

// TokenValidator implements token.Validator for GitHub tokens
type TokenValidator struct {
	baseURL string
}

// NewTokenValidator creates a new GitHub token validator
func NewTokenValidator() *TokenValidator {
	return &TokenValidator{
		baseURL: apiBaseURL,
	}
}

// Validate checks if a token is valid for GitHub Actions operations
func (v *TokenValidator) Validate(ctx context.Context, t *token.Token) error {
	if t.Value == "" {
		return token.ErrTokenInvalid
	}

	if token.IsExpired(*t) {
		return token.ErrTokenExpired
	}

	// Verify token and get scopes from GitHub API
	if err := v.verifyToken(ctx, t); err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	// Check if required scopes are present
	if err := v.validateScopes(t.Scope); err != nil {
		return fmt.Errorf("invalid token scope: %w", err)
	}

	return nil
}

// validateScopes checks if the token has the required scopes
func (v *TokenValidator) validateScopes(scope string) error {
	if scope == "" {
		return fmt.Errorf("no scopes provided")
	}

	scopes := strings.Split(scope, ",")
	// Trim spaces from each scope
	for i, s := range scopes {
		scopes[i] = strings.TrimSpace(s)
	}
	required := map[string]bool{
		ScopeRepo:     false,
		ScopeWorkflow: false,
	}

	for _, s := range scopes {
		if _, ok := required[s]; ok {
			required[s] = true
		}
	}

	// Return detailed scope status
	var missingScopes []string
	scopeStatus := make(map[string]bool)
	for scope, present := range required {
		scopeStatus[scope] = present
		if !present {
			missingScopes = append(missingScopes, scope)
		}
	}

	if len(missingScopes) > 0 {
		return &token.ScopeError{
			Missing: missingScopes,
			Status:  scopeStatus,
		}
	}

	return nil
}

// verifyToken makes a test API call to verify the token and get its scopes
func (v *TokenValidator) verifyToken(ctx context.Context, t *token.Token) error {
	req, err := http.NewRequestWithContext(ctx, "GET", v.baseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.Value)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("invalid token: status %d", resp.StatusCode)
		}
		return fmt.Errorf("invalid token: %s", errorResp.Message)
	}

	// Get scopes from response header
	scopes := resp.Header.Get("X-OAuth-Scopes")
	if scopes == "" {
		return fmt.Errorf("no scopes found in token")
	}

	// Update token scope with actual scopes from GitHub
	t.Scope = scopes

	// Get expiration from response header
	if expStr := resp.Header.Get("GitHub-Authentication-Token-Expiration"); expStr != "" {
		// GitHub returns time in format "2025-03-04 02:13:04 UTC"
		expTime, err := time.Parse("2006-01-02 15:04:05 MST", expStr)
		if err != nil {
			return fmt.Errorf("failed to parse token expiration: %w", err)
		}
		t.ExpiresAt = expTime
	}

	return nil
}
