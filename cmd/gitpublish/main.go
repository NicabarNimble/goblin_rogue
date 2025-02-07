package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	gerrors "github.com/NicabarNimble/go-gittools/internal/errors"
	"github.com/NicabarNimble/go-gittools/internal/git"
	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/NicabarNimble/go-gittools/internal/urlutils"
)

type config struct {
	private       string
	publicFork    string
	branch        string
	token         string
	createPR      bool
	prTitle       string
	prDescription string
	targetBranch  string
	createFork    bool
}

func parseFlags() *config {
	cfg := &config{}

	flag.StringVar(&cfg.private, "private", "", "Private repository path")
	flag.StringVar(&cfg.publicFork, "public", "", "Public fork repository URL")
	flag.StringVar(&cfg.branch, "branch", "main", "Branch to publish")
	flag.StringVar(&cfg.token, "token", "", "GitHub token for authentication")

	// PR-related flags
	flag.BoolVar(&cfg.createPR, "pr", false, "Create a pull request after publishing")
	flag.StringVar(&cfg.prTitle, "pr-title", "", "Title for the pull request")
	flag.StringVar(&cfg.prDescription, "pr-desc", "", "Description for the pull request")
	flag.StringVar(&cfg.targetBranch, "target-branch", "main", "Target branch for the pull request")

	// Fork-related flag
	flag.BoolVar(&cfg.createFork, "create-fork", false, "Create a fork if it doesn't exist")

	flag.Parse()

	// In test mode, panic instead of exiting
	isTest := flag.Lookup("test.v") != nil

	if cfg.private == "" || cfg.publicFork == "" {
		msg := "Error: private repository path and public fork URL are required"
		if isTest {
			panic(msg)
		}
		fmt.Println(msg)
		flag.Usage()
		os.Exit(1)
	}

	if cfg.createPR && cfg.prTitle == "" {
		msg := "Error: pr-title is required when creating a pull request"
		if isTest {
			panic(msg)
		}
		fmt.Println(msg)
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func main() {
	cfg := parseFlags()

	// Initialize progress tracker
	tracker := &progress.DefaultTracker{}

	// Perform publish operation
	if err := publishRepository(cfg, tracker); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// parseGitHubURL extracts owner and repo from a GitHub URL
func parseGitHubURL(rawURL string) (owner, repo string, err error) {
	// Only accept HTTPS URLs
	if strings.HasPrefix(rawURL, "git@") {
		return "", "", fmt.Errorf("SSH URLs are not supported, please use HTTPS")
	}

	// Parse and validate the URL
	parsedURL, err := urlutils.ParseHTTPSURL(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid GitHub URL: %w", err)
	}

	// Extract owner and repo from path
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("URL must include owner and repository")
	}

	// Remove .git suffix if present
	repo = strings.TrimSuffix(pathParts[1], ".git")
	return pathParts[0], repo, nil
}

func publishRepository(cfg *config, tracker progress.Tracker) error {
	ctx := context.Background()
	// Create and validate token
	t, err := token.NewToken(cfg.token, time.Time{}, "repo workflow admin:repo")
	if err != nil {
		if errors.Is(err, token.ErrTokenInvalid) {
			return gerrors.New("publish", fmt.Errorf("invalid GitHub token format"))
		}
		return gerrors.New("publish", fmt.Errorf("failed to create token: %w", err))
	}

	// Pre-validate token with GitHub API
	validator := github.NewTokenValidator()
	if err := validator.Validate(ctx, t); err != nil {
		if strings.Contains(err.Error(), "missing required scopes") {
			return gerrors.New("publish", fmt.Errorf("GitHub token is missing required scopes (repo, workflow, admin:repo). Please check token permissions"))
		}
		return gerrors.New("publish", fmt.Errorf("GitHub token validation failed: %w", err))
	}

	// Create GitHub client with validated token
	ghClient, err := github.NewClient(ctx, t)
	if err != nil {
		return gerrors.New("publish", fmt.Errorf("failed to create GitHub client: %w", err))
	}

	// Create fork if requested
	if cfg.createFork {
		targetOwner, targetRepo, err := parseGitHubURL(cfg.private)
		if err != nil {
			return gerrors.New("publish", fmt.Errorf("failed to parse target repository URL: %w", err))
		}
		fmt.Printf("Creating fork of %s/%s...\n", targetOwner, targetRepo)
		if err := ghClient.CreateFork(ctx, fmt.Sprintf("%s/%s", targetOwner, targetRepo)); err != nil {
			return gerrors.New("publish", fmt.Errorf("failed to create fork: %w", err))
		}
	}

	// Clone private repository to temporary location
	tempDir, err := os.MkdirTemp("", "gitpublish-*")
	if err != nil {
		return gerrors.New("publish", fmt.Errorf("failed to create temp directory: %w", err))
	}
	defer os.RemoveAll(tempDir)

	// Clone and push repository
	cloneOpts := git.CloneOptions{
		SourceURL:  cfg.private,
		TargetURL:  cfg.publicFork,
		Token:      cfg.token,
		Progress:   tracker,
	}
	if err := git.CloneRepository(cloneOpts); err != nil {
		return gerrors.New("publish", fmt.Errorf("failed to push to public fork: %w", err))
	}

	fmt.Printf("Successfully published %s to %s\n", cfg.private, cfg.publicFork)

	// Create pull request if requested
	if cfg.createPR {
		sourceOwner, _, err := parseGitHubURL(cfg.publicFork)
		if err != nil {
			return gerrors.New("publish", fmt.Errorf("failed to parse source repository URL: %w", err))
		}

		targetOwner, targetRepo, err := parseGitHubURL(cfg.private)
		if err != nil {
			return gerrors.New("publish", fmt.Errorf("failed to parse target repository URL: %w", err))
		}

		prOpts := github.PROptions{
			Owner: targetOwner,
			Repo:  targetRepo,
			Base:  cfg.targetBranch,
			Head:  fmt.Sprintf("%s:%s", sourceOwner, cfg.branch),
			Title: cfg.prTitle,
			Body:  cfg.prDescription,
		}

		fmt.Printf("Creating pull request from %s to %s/%s...\n", prOpts.Head, targetOwner, targetRepo)
		if err := ghClient.CreatePullRequest(ctx, prOpts); err != nil {
			return gerrors.New("publish", fmt.Errorf("failed to create pull request: %w", err))
		}
		fmt.Printf("Successfully created pull request: %s\n", cfg.prTitle)
	}

	return nil
}
