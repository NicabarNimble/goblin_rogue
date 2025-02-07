// Package urlutils provides utilities for handling GitHub repository URLs.
// It supports parsing and validation of HTTPS URLs for both public GitHub
// and GitHub Enterprise instances.
package urlutils

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// ErrInvalidURL indicates that the provided URL is not valid
	ErrInvalidURL = errors.New("invalid URL format")

	// ErrInvalidHost indicates that the host is not a valid GitHub instance
	ErrInvalidHost = errors.New("invalid GitHub host")

	// ErrInvalidPath indicates that the URL path is not a valid repository path
	ErrInvalidPath = errors.New("invalid repository path")

	// ErrEmptyToken indicates that an empty token was provided
	ErrEmptyToken = errors.New("empty token provided")

	// ErrNotHTTPS indicates that the URL does not use HTTPS protocol
	ErrNotHTTPS = errors.New("URL must use HTTPS protocol")

	// Regular expressions for validation
	ownerRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,35}$`)
	repoRegex  = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`)

	// Allowed GitHub Enterprise Server domains
	// This should be configured by the organization
	allowedGHEDomains = map[string]bool{
		"github.enterprise.com": true,
		"git.company.com":      true,
	}
)

// ParseHTTPSURL parses and validates a GitHub HTTPS URL.
// It accepts URLs in the following formats:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - https://github.enterprise.com/owner/repo
//
// The function validates the URL format, host, and repository path components.
func ParseHTTPSURL(rawURL string) (*url.URL, error) {
	if strings.HasPrefix(rawURL, "git@") {
		return nil, ErrNotHTTPS
	}
	if !strings.HasPrefix(rawURL, "https://") {
		return nil, ErrInvalidURL
	}

	// Remove .git suffix and sanitize URL
	rawURL = sanitizeURL(strings.TrimSuffix(rawURL, ".git"))

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	if !isValidGitHubHost(parsedURL.Host) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidHost, parsedURL.Host)
	}

	// Validate path components
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("%w: URL must include owner and repository", ErrInvalidPath)
	}

	if !ownerRegex.MatchString(pathParts[0]) {
		return nil, fmt.Errorf("%w: invalid owner name format", ErrInvalidPath)
	}

	if !repoRegex.MatchString(pathParts[1]) {
		return nil, fmt.Errorf("%w: invalid repository name format", ErrInvalidPath)
	}

	return parsedURL, nil
}

// FormatTokenURL formats a GitHub URL with the provided token.
// It creates a new URL with the token embedded as the user info component.
// The original URL is not modified.
func FormatTokenURL(parsedURL *url.URL, token string) (*url.URL, error) {
	if parsedURL == nil {
		return nil, fmt.Errorf("%w: nil URL provided", ErrInvalidURL)
	}

	if token == "" {
		return nil, ErrEmptyToken
	}

	// Create a copy and sanitize
	tokenURL := *parsedURL
	tokenURL.User = nil // Clear any existing credentials
	tokenURL.User = url.User(token)

	return &tokenURL, nil
}

// ValidateURL checks if the provided URL is a valid GitHub repository URL.
// It performs comprehensive validation including:
//   - URL format and protocol
//   - GitHub host validation
//   - Owner and repository name format
//   - Path structure
func ValidateURL(rawURL string) error {
	_, err := ParseHTTPSURL(rawURL)
	return err
}

// isValidGitHubHost checks if the host is github.com or an allowed GitHub Enterprise host.
// It supports the following formats:
//   - github.com (Public GitHub)
//   - *.github.com (GitHub Enterprise Cloud)
//   - Explicitly allowed GitHub Enterprise Server domains
func isValidGitHubHost(host string) bool {
	// Public GitHub
	if host == "github.com" {
		return true
	}

	// GitHub Enterprise Cloud
	if strings.HasSuffix(host, ".github.com") {
		return true
	}

	// GitHub Enterprise Server - only allow explicitly configured domains
	return allowedGHEDomains[host]
}

// sanitizeURL removes any sensitive information from the URL
func sanitizeURL(rawURL string) string {
	// Remove any existing credentials
	if u, err := url.Parse(rawURL); err == nil {
		u.User = nil
		return u.String()
	}
	return rawURL
}
