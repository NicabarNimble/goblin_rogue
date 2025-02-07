package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/NicabarNimble/go-gittools/internal/token"
)

const (
	apiBaseURL = "https://gitlab.com/api/v4"
	userAgent  = "go-gittools"
)

// TokenValidator implements token.Validator for GitLab tokens
type TokenValidator struct {
	baseURL string
}

// NewTokenValidator creates a new GitLab token validator
func NewTokenValidator() *TokenValidator {
	return &TokenValidator{
		baseURL: apiBaseURL,
	}
}

// Validate checks if a token is valid for GitLab operations
func (v *TokenValidator) Validate(ctx context.Context, t *token.Token) error {
	if t.Value == "" {
		return token.ErrTokenInvalid
	}

	if token.IsExpired(*t) {
		return token.ErrTokenExpired
	}

	// Verify token and get scopes from GitLab API
	if err := v.verifyToken(ctx, t); err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}
	return nil
}

// verifyToken makes a test API call to verify the token and get its scopes
func (v *TokenValidator) verifyToken(ctx context.Context, t *token.Token) error {
	req, err := http.NewRequestWithContext(ctx, "GET", v.baseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", t.Value)
	req.Header.Set("Accept", "application/json")
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
	scopes := resp.Header.Get("X-Gitlab-Scopes")
	if scopes == "" {
		scopes = "api" // Default scope for personal access tokens
	}

	// Update token scope with actual scopes from GitLab
	// Split by comma, trim spaces, and join with spaces
	scopesList := strings.Split(scopes, ",")
	for i, s := range scopesList {
		scopesList[i] = strings.TrimSpace(s)
	}
	t.Scope = strings.Join(scopesList, " ")

	return nil
}
