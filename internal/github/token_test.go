package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/stretchr/testify/assert"
)

func TestTokenValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		token     token.Token
		mockAPI   func(w http.ResponseWriter, r *http.Request)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid token with all scopes",
			token: token.Token{
				Value:     "valid_token",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"login": "testuser"}`))
			},
			wantError: false,
		},
		{
			name: "expired token",
			token: token.Token{
				Value:     "expired_token",
				ExpiresAt: time.Now().Add(-24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantError: true,
			errorMsg:  "token has expired",
		},
		{
			name: "missing required scopes",
			token: token.Token{
				Value:     "limited_token",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantError: true,
			errorMsg:  "missing required scopes: workflow, admin:repo",
		},
		{
			name: "invalid token response",
			token: token.Token{
				Value:     "invalid_token",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message": "Bad credentials"}`))
			},
			wantError: true,
			errorMsg:  "token verification failed: invalid token: Bad credentials",
		},
		{
			name: "empty token",
			token: token.Token{
				Value:     "",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("API should not be called for empty token")
			},
			wantError: true,
			errorMsg:  "token is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.mockAPI))
			defer server.Close()

			// Create validator with test server URL
			v := &TokenValidator{
				baseURL: server.URL,
			}

			// Run validation
			err := v.Validate(context.Background(), &tt.token)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenValidator_ValidateScopes(t *testing.T) {
	v := NewTokenValidator()

	tests := []struct {
		name      string
		scope     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "all required scopes",
			scope:     "repo workflow admin:repo",
			wantError: false,
		},
		{
			name:      "missing workflow scope",
			scope:     "repo admin:repo",
			wantError: true,
			errorMsg:  "missing required scopes: workflow",
		},
		{
			name:      "empty scope",
			scope:     "",
			wantError: true,
			errorMsg:  "no scopes provided",
		},
		{
			name:      "extra scopes",
			scope:     "repo workflow admin:repo user",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validateScopes(tt.scope)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
