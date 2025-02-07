package token

import "testing"

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  Provider
	}{
		{
			name:  "github token with ghp prefix",
			token: "ghp_1234567890abcdef",
			want:  ProviderGitHub,
		},
		{
			name:  "github token with github_pat prefix",
			token: "github_pat_1234567890abcdef",
			want:  ProviderGitHub,
		},
		{
			name:  "gitlab token",
			token: "glpat-1234567890abcdef",
			want:  ProviderGitLab,
		},
		{
			name:  "invalid token format",
			token: "invalid-token",
			want:  "",
		},
		{
			name:  "empty token",
			token: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectProvider(tt.token)
			if got != tt.want {
				t.Errorf("DetectProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}
