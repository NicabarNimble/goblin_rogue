package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		validate    func(*testing.T, *config)
	}{
		{
			name:        "No flags",
			args:        []string{},
			expectError: true,
		},
		{
			name: "Missing public fork",
			args: []string{
				"-private", "https://github.com/user/private-repo",
			},
			expectError: true,
		},
		{
			name: "Missing private repo",
			args: []string{
				"-public", "https://github.com/user/public-fork",
			},
			expectError: true,
		},
		{
			name: "Basic configuration",
			args: []string{
				"-private", "https://github.com/user/private-repo",
				"-public", "https://github.com/user/public-fork",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config) {
				assert.Equal(t, "https://github.com/user/private-repo", cfg.private)
				assert.Equal(t, "https://github.com/user/public-fork", cfg.publicFork)
				assert.Equal(t, "main", cfg.branch)
				assert.False(t, cfg.createPR)
				assert.False(t, cfg.createFork)
			},
		},
		{
			name: "Full configuration with PR",
			args: []string{
				"-private", "https://github.com/user/private-repo",
				"-public", "https://github.com/user/public-fork",
				"-token", "ghp_token123",
				"-branch", "feature",
				"-pr",
				"-pr-title", "New Feature",
				"-pr-desc", "Added new feature",
				"-target-branch", "develop",
				"-create-fork",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config) {
				assert.Equal(t, "https://github.com/user/private-repo", cfg.private)
				assert.Equal(t, "https://github.com/user/public-fork", cfg.publicFork)
				assert.Equal(t, "feature", cfg.branch)
				assert.Equal(t, "ghp_token123", cfg.token)
				assert.True(t, cfg.createPR)
				assert.Equal(t, "New Feature", cfg.prTitle)
				assert.Equal(t, "Added new feature", cfg.prDescription)
				assert.Equal(t, "develop", cfg.targetBranch)
				assert.True(t, cfg.createFork)
			},
		},
		{
			name: "PR without title",
			args: []string{
				"-private", "https://github.com/user/private-repo",
				"-public", "https://github.com/user/public-fork",
				"-pr",
			},
			expectError: true,
		},
	}

	// Helper function to parse flags in test environment
	parseTestFlags := func(args []string) (cfg *config, err error) {
		// Recover from panic and convert to error
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()

		// Save and restore os.Args
		oldArgs := os.Args
		os.Args = append([]string{"go-gitpublish"}, args...)
		defer func() { os.Args = oldArgs }()

		// Save and restore stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() {
			w.Close()
			os.Stdout = oldStdout
			r.Close()
		}()

		// Create new flag set for each test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		flag.CommandLine.SetOutput(w)

		// Set test.v flag to simulate test environment
		flag.CommandLine.Bool("test.v", true, "")
		
		cfg = parseFlags()
		return cfg, nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseTestFlags(tt.args)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		expectError bool
	}{
		{
			name:        "HTTPS URL",
			url:         "https://github.com/user/repo",
			wantOwner:   "user",
			wantRepo:    "repo",
			expectError: false,
		},
		{
			name:        "HTTPS URL with .git",
			url:         "https://github.com/user/repo.git",
			wantOwner:   "user",
			wantRepo:    "repo",
			expectError: false,
		},
		{
			name:        "SSH URL",
			url:         "git@github.com:user/repo",
			expectError: true,
		},
		{
			name:        "SSH URL with .git",
			url:         "git@github.com:user/repo.git",
			expectError: true,
		},
		{
			name:        "GitHub Enterprise URL",
			url:         "https://github.enterprise.com/user/repo",
			wantOwner:   "user",
			wantRepo:    "repo",
			expectError: false,
		},
		{
			name:        "Invalid protocol",
			url:         "http://github.com/user/repo",
			expectError: true,
		},
		{
			name:        "Invalid host",
			url:         "https://gitlab.com/user/repo",
			expectError: true,
		},
		{
			name:        "Invalid URL format",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "Missing repository",
			url:         "https://github.com/user",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.url)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

func TestPublishRepository(t *testing.T) {
	tests := []struct {
		name        string
		config      *config
		mockSetup   func(*mockGitOperations, *mockGitHubClient)
		expectError bool
	}{
		{
			name: "Basic publish without PR",
			config: &config{
				private:    "https://github.com/user/private-repo",
				publicFork: "https://github.com/user/public-fork",
				branch:     "main",
				token:     "test-token",
			},
			mockSetup: func(git *mockGitOperations, gh *mockGitHubClient) {
				git.cloneError = nil
			},
			expectError: false,
		},
		{
			name: "Publish with PR creation",
			config: &config{
				private:       "https://github.com/user/private-repo",
				publicFork:   "https://github.com/user/public-fork",
				branch:       "feature",
				token:       "test-token",
				createPR:    true,
				prTitle:     "New Feature",
				targetBranch: "main",
			},
			mockSetup: func(git *mockGitOperations, gh *mockGitHubClient) {
				git.cloneError = nil
				gh.createPRError = nil
			},
			expectError: false,
		},
		{
			name: "Clone failure",
			config: &config{
				private:    "https://github.com/user/private-repo",
				publicFork: "https://github.com/user/public-fork",
				branch:     "main",
				token:     "test-token",
			},
			mockSetup: func(git *mockGitOperations, gh *mockGitHubClient) {
				git.cloneError = assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock instances
			mockGit := &mockGitOperations{}
			mockGH := &mockGitHubClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockGit, mockGH)
			}

			// Create a test version of publishRepository that uses our mock GitHub client
			testPublishRepository := func(gitOps *mockGitOperations, cfg *config, tracker progress.Tracker) error {
				ctx := context.Background()
				return publishRepositoryWithClient(ctx, gitOps, mockGH, cfg, tracker)
			}

			err := testPublishRepository(mockGit, tt.config, &progress.DefaultTracker{})
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// publishRepositoryWithClient is a test helper that allows injecting a mock GitHub client
func publishRepositoryWithClient(ctx context.Context, gitOps *mockGitOperations, ghClient *mockGitHubClient, cfg *config, tracker progress.Tracker) error {
	if cfg.createFork {
		targetOwner, targetRepo, err := parseGitHubURL(cfg.private)
		if err != nil {
			return err
		}
		if err := ghClient.CreateFork(ctx, fmt.Sprintf("%s/%s", targetOwner, targetRepo)); err != nil {
			return err
		}
	}

	// Create temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "gitpublish-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone and push repository
	cloneOpts := git.CloneOptions{
		SourceURL:  cfg.private,
		TargetURL:  cfg.publicFork,
		Token:      cfg.token,
		Progress:   tracker,
	}
	if err := gitOps.CloneRepository(cloneOpts); err != nil {
		return err
	}

	if cfg.createPR {
		sourceOwner, _, err := parseGitHubURL(cfg.publicFork)
		if err != nil {
			return err
		}

		targetOwner, targetRepo, err := parseGitHubURL(cfg.private)
		if err != nil {
			return err
		}

		prOpts := github.PROptions{
			Owner: targetOwner,
			Repo:  targetRepo,
			Base:  cfg.targetBranch,
			Head:  fmt.Sprintf("%s:%s", sourceOwner, cfg.branch),
			Title: cfg.prTitle,
			Body:  cfg.prDescription,
		}

		if err := ghClient.CreatePullRequest(ctx, prOpts); err != nil {
			return err
		}
	}

	return nil
}

// mockGitOperations provides a mock for git operations
type mockGitOperations struct {
	cloneError error
}

func (m *mockGitOperations) CloneRepository(opts git.CloneOptions) error {
	if m.cloneError != nil {
		return m.cloneError
	}
	return nil
}

// mockGitHubClient implements github.Client interface
type mockGitHubClient struct {
	createForkError error
	createPRError   error
	createRepoError error
}

func (m *mockGitHubClient) CreateRepository(ctx context.Context, opts github.RepoOptions) error {
	return m.createRepoError
}

func (m *mockGitHubClient) CreateFork(ctx context.Context, repo string) error {
	return m.createForkError
}

func (m *mockGitHubClient) CreatePullRequest(ctx context.Context, opts github.PROptions) error {
	return m.createPRError
}

// Implement other required methods of the github.Client interface with no-op implementations
func (m *mockGitHubClient) CreateOrUpdateWorkflow(ctx context.Context, owner, repo, path string, content []byte) error {
	return nil
}

func (m *mockGitHubClient) TriggerWorkflow(ctx context.Context, owner, repo, workflowID string, inputs map[string]interface{}) error {
	return nil
}

func (m *mockGitHubClient) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*github.WorkflowRun, error) {
	return nil, nil
}

func (m *mockGitHubClient) GetWorkflowLogs(ctx context.Context, owner, repo string, runID int64) ([]byte, error) {
	return nil, nil
}

func (m *mockGitHubClient) ListWorkflowRuns(ctx context.Context, owner, repo, workflowID string) ([]github.WorkflowRun, error) {
	return nil, nil
}
