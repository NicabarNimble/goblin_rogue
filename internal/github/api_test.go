package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		token     *token.Token
		mockAPI   func(w http.ResponseWriter, r *http.Request)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid token",
			token: &token.Token{
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
			name: "invalid token",
			token: &token.Token{
				Value:     "invalid_token",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message": "Bad credentials"}`))
			},
			wantError: true,
			errorMsg:  "token validation failed",
		},
		{
			name: "expired token",
			token: &token.Token{
				Value:     "expired_token",
				ExpiresAt: time.Now().Add(-24 * time.Hour),
				Scope:    "repo workflow admin:repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("API should not be called for expired token")
			},
			wantError: true,
			errorMsg:  "token validation failed: token has expired",
		},
		{
			name: "missing required scopes",
			token: &token.Token{
				Value:     "limited_token",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Scope:    "repo",
			},
			mockAPI: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantError: true,
			errorMsg:  "token validation failed: invalid token scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockAPI))
			defer server.Close()

			// Create client with test server URL
			client, err := NewClient(context.Background(), tt.token)
			if err == nil {
				client.baseURL = server.URL
				client.token = tt.token.Value
			}

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.token.Value, client.token)
			}
		})
	}

}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOwner   string
		wantRepo    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid repository format",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:        "missing slash",
			input:       "invalidformat",
			wantErr:     true,
			errContains: "invalid repository format",
		},
		{
			name:        "too many slashes",
			input:       "too/many/slashes",
			wantErr:     true,
			errContains: "invalid repository format",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "invalid repository format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepo(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error message %q does not contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestCreateFork(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		statusCode  int
		response    string
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful fork creation",
			repo:      "owner/repo",
			statusCode: http.StatusAccepted,
			response:   `{"id": 123, "name": "repo", "full_name": "new-owner/repo"}`,
			wantErr:    false,
		},
		{
			name:        "invalid repository format",
			repo:        "invalid-format",
			wantErr:     true,
			errContains: "invalid repository format",
		},
		{
			name:        "server error",
			repo:        "owner/repo",
			statusCode:  http.StatusInternalServerError,
			response:    `{"message": "server error"}`,
			wantErr:     true,
			errContains: "failed to create fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
					w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: time.Second * 30,
				},
			}

			err := client.CreateFork(context.Background(), tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error message %q does not contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreatePullRequest(t *testing.T) {
	tests := []struct {
		name        string
		opts        PROptions
		statusCode  int
		response    string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful PR creation",
			opts: PROptions{
				Owner: "owner",
				Repo:  "repo",
				Base:  "main",
				Head:  "feature",
				Title: "Test PR",
				Body:  "PR description",
			},
			statusCode: http.StatusCreated,
			response:   `{"number": 1, "title": "Test PR"}`,
			wantErr:    false,
		},
		{
			name: "server error",
			opts: PROptions{
				Owner: "owner",
				Repo:  "repo",
				Base:  "main",
				Head:  "feature",
				Title: "Test PR",
				Body:  "PR description",
			},
			statusCode:  http.StatusInternalServerError,
			response:    `{"message": "server error"}`,
			wantErr:     true,
			errContains: "failed to create pull request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				var reqBody map[string]string
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}

				if reqBody["title"] != tt.opts.Title {
					t.Errorf("expected title %q, got %q", tt.opts.Title, reqBody["title"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: time.Second * 30,
				},
			}

			err := client.CreatePullRequest(context.Background(), tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error message %q does not contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	var requestCount int
	resetTime := time.Now().Add(10 * time.Millisecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request - return rate limit error
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "API rate limit exceeded"}`))
			return
		}
		// Second request - return success
		w.Header().Set("X-RateLimit-Remaining", "60")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := &Client{
		token:   "test-token",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.makeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d with body: %s", resp.StatusCode, string(body))
	}

	if requestCount != 2 {
		t.Errorf("expected 2 requests (1 rate limited + 1 retry), got %d", requestCount)
	}

	var response struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if response.Message != "success" {
		t.Errorf("expected success message, got: %s", response.Message)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && s != "" && (s == substr || contains_helper(s, substr))
}

// contains_helper is a helper function that checks if s contains substr
func contains_helper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
