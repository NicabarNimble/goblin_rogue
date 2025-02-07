package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/token"
)

func TestTokenValidator_Validate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the correct endpoint is being called
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Check request headers
		if r.Header.Get("PRIVATE-TOKEN") != "valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"401 Unauthorized"}`))
			return
		}

		// Set GitLab scopes header
		w.Header().Set("X-Gitlab-Scopes", "api,read_user")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "username": "test"}`))
	}))
	defer server.Close()

	validator := &TokenValidator{
		baseURL: server.URL,
	}

	tests := []struct {
		name      string
		token     token.Token
		wantErr   bool
		wantScope string
	}{
		{
			name: "valid token",
			token: token.Token{
				Value:     "valid-token",
				ExpiresAt: time.Time{},
			},
			wantErr:   false,
			wantScope: "api read_user",
		},
		{
			name: "invalid token",
			token: token.Token{
				Value:     "invalid-token",
				ExpiresAt: time.Time{},
			},
			wantErr: true,
		},
		{
			name: "expired token",
			token: token.Token{
				Value:     "valid-token",
				ExpiresAt: time.Now().Add(-24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "empty token",
			token: token.Token{
				Value:     "",
				ExpiresAt: time.Time{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), &tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.token.Scope != tt.wantScope {
				t.Errorf("TokenValidator.Validate() got scope = %v, want %v", tt.token.Scope, tt.wantScope)
			}
		})
	}
}
