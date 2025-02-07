package urlutils

import (
	"errors"
	"net/url"
	"testing"
)

func TestParseHTTPSURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr error
	}{
		{
			name:    "valid GitHub URL",
			rawURL:  "https://github.com/owner/repo",
			wantErr: nil,
		},
		{
			name:    "valid GitHub URL with .git suffix",
			rawURL:  "https://github.com/owner/repo.git",
			wantErr: nil,
		},
		{
			name:    "valid GitHub Enterprise URL",
			rawURL:  "https://github.enterprise.com/owner/repo",
			wantErr: nil,
		},
		{
			name:    "valid GitHub Enterprise Cloud URL",
			rawURL:  "https://custom.github.com/owner/repo",
			wantErr: nil,
		},
		{
			name:    "URL with trailing slash",
			rawURL:  "https://github.com/owner/repo/",
			wantErr: nil,
		},
		{
			name:    "SSH URL not supported",
			rawURL:  "git@github.com:owner/repo",
			wantErr: ErrNotHTTPS,
		},
		{
			name:    "invalid protocol",
			rawURL:  "http://github.com/owner/repo",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid host",
			rawURL:  "https://gitlab.com/owner/repo",
			wantErr: ErrInvalidHost,
		},
		{
			name:    "malformed URL",
			rawURL:  "https://github.com:invalid:port/repo",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "missing repository",
			rawURL:  "https://github.com/owner",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "invalid owner name",
			rawURL:  "https://github.com/-owner/repo",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "invalid repository name",
			rawURL:  "https://github.com/owner/repo!invalid",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "owner name too long",
			rawURL:  "https://github.com/thisownernameiswaytoolongandshouldfail/repo",
			wantErr: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHTTPSURL(tt.rawURL)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseHTTPSURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil && got == nil {
				t.Error("ParseHTTPSURL() returned nil URL for valid input")
			}
		})
	}
}

func TestFormatTokenURL(t *testing.T) {
	validURL, _ := url.Parse("https://github.com/owner/repo")

	tests := []struct {
		name     string
		url      *url.URL
		token    string
		wantErr  error
		wantUser string
	}{
		{
			name:     "valid token",
			url:      validURL,
			token:    "abc123",
			wantErr:  nil,
			wantUser: "abc123",
		},
		{
			name:    "nil URL",
			url:     nil,
			token:   "abc123",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "empty token",
			url:     validURL,
			token:   "",
			wantErr: ErrEmptyToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatTokenURL(tt.url, tt.token)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("FormatTokenURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil {
				if got.User == nil {
					t.Error("FormatTokenURL() user is nil")
				} else if got.User.Username() != tt.wantUser {
					t.Errorf("FormatTokenURL() user = %v, want %v", got.User.Username(), tt.wantUser)
				}
				// Verify URL was not modified
				if validURL.String() == got.String() {
					t.Error("FormatTokenURL() modified original URL")
				}
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr error
	}{
		{
			name:    "valid repository URL",
			rawURL:  "https://github.com/owner/repo",
			wantErr: nil,
		},
		{
			name:    "valid URL with .git suffix",
			rawURL:  "https://github.com/owner/repo.git",
			wantErr: nil,
		},
		{
			name:    "SSH URL not supported",
			rawURL:  "git@github.com:owner/repo",
			wantErr: ErrNotHTTPS,
		},
		{
			name:    "missing repository",
			rawURL:  "https://github.com/owner",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "root URL",
			rawURL:  "https://github.com",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "invalid protocol",
			rawURL:  "http://github.com/owner/repo",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid characters in owner",
			rawURL:  "https://github.com/owner$/repo",
			wantErr: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.rawURL)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidGitHubHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{
			name: "public GitHub",
			host: "github.com",
			want: true,
		},
		{
			name: "GitHub Enterprise Cloud",
			host: "enterprise.github.com",
			want: true,
		},
		{
			name: "GitHub Enterprise Server",
			host: "github.enterprise.com",
			want: true,
		},
		{
			name: "invalid host",
			host: "gitlab.com",
			want: false,
		},
		{
			name: "empty host",
			host: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidGitHubHost(tt.host); got != tt.want {
				t.Errorf("isValidGitHubHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParseHTTPSURL(b *testing.B) {
	urls := []string{
		"https://github.com/owner/repo",
		"https://github.com/owner/repo.git",
		"https://github.enterprise.com/owner/repo",
	}

	for _, url := range urls {
		b.Run(url, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = ParseHTTPSURL(url)
			}
		})
	}
}
