package token

import (
	"fmt"
	"strings"
)

// Provider represents a Git provider type
type Provider string

const (
	ProviderGitHub Provider = "GITHUB"
	ProviderGitLab Provider = "GITLAB"
)

// DetectProvider attempts to determine the token provider from the token format
func DetectProvider(tokenValue string) Provider {
	switch {
	case strings.HasPrefix(tokenValue, "ghp_"),
		strings.HasPrefix(tokenValue, "github_pat_"):
		return ProviderGitHub
	case strings.HasPrefix(tokenValue, "glpat-"):
		return ProviderGitLab
	default:
		return ""
	}
}

// ScopeError represents a token scope validation error with detailed status
type ScopeError struct {
	Missing []string         // List of missing required scopes
	Status  map[string]bool  // Status of all required scopes (present/missing)
}

func (e *ScopeError) Error() string {
	return fmt.Sprintf("missing required scopes: %s", strings.Join(e.Missing, ", "))
}
