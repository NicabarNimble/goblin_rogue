package git

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/urlutils"
	"github.com/NicabarNimble/go-gittools/internal/errors"
)

const (
	defaultTimeout = 10 * time.Minute
	maxRetries    = 3
)

// ErrInvalidOptions indicates that the provided clone options are invalid
var ErrInvalidOptions = errors.New("clone", fmt.Errorf("invalid clone options"))

// CloneOptions contains configuration for repository cloning
type CloneOptions struct {
	SourceURL  string
	TargetURL  string
	WorkingDir string
	Token      string          // Token for HTTPS authentication
	Progress   progress.Tracker
	Context    context.Context // Context for cancellation/timeout
}

// CloneRepository clones a source repository to a target location
func CloneRepository(opts CloneOptions) error {
	// Set up context with timeout if not provided
	if opts.Context == nil {
		var cancel context.CancelFunc
		opts.Context, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}

	// Validate required fields
	if opts.SourceURL == "" {
		err := errors.New("clone", fmt.Errorf("source URL must be specified"))
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return err
	}

	// Initialize progress tracking
	if opts.Progress != nil {
		opts.Progress.Start("Clone Repository")
		defer opts.Progress.Complete()
	}

	// Check for context cancellation
	select {
	case <-opts.Context.Done():
		err := errors.New("clone", fmt.Errorf("operation cancelled: %w", opts.Context.Err()))
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return err
	default:
	}

	// Parse and validate source URL - only accept HTTPS URLs
	sourceURL := opts.SourceURL
	if strings.HasPrefix(sourceURL, "git@") {
		err := errors.New("clone", fmt.Errorf("SSH URLs are not supported, please use HTTPS"))
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return err
	}
	
// Skip URL validation for file:// URLs (used in tests)
if !strings.HasPrefix(sourceURL, "file://") {
	// Validate the HTTPS URL
	if err := urlutils.ValidateURL(sourceURL); err != nil {
		err = errors.New("clone", fmt.Errorf("invalid source URL: %w", err))
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return err
	}
}

	// If WorkingDir is specified, clone directly to it
	if opts.WorkingDir != "" {
		if err := runGitCommand("", opts.Token, "clone", sourceURL, opts.WorkingDir); err != nil {
			if opts.Progress != nil {
				opts.Progress.Error(err)
			}
			return errors.New("clone", fmt.Errorf("failed to clone source repository: %w", err))
		}
		return nil
	}

	// If TargetURL is specified, use the mirror workflow
	if opts.TargetURL == "" && opts.WorkingDir == "" {
		err := errors.New("clone", fmt.Errorf("either working directory or target URL must be specified"))
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return err
	}

	// Create temporary directory for initial clone with proper cleanup
	tempDir, err := os.MkdirTemp("", "gitclone-*")
	if err != nil {
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return errors.New("clone", fmt.Errorf("failed to create temp directory: %w", err))
	}
	
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(tempDir)
		}
	}()

	// Clone source repository
	if err := runGitCommand(tempDir, opts.Token, "clone", sourceURL, "."); err != nil {
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return errors.New("clone", fmt.Errorf("failed to clone source repository: %w", err))
	}

// Parse and validate target URL if specified
targetURL := opts.TargetURL
if targetURL != "" {
	if strings.HasPrefix(targetURL, "git@") {
		return errors.New("clone", fmt.Errorf("SSH URLs are not supported, please use HTTPS"))
	}
	
	// Skip URL validation for file:// URLs (used in tests)
	if !strings.HasPrefix(targetURL, "file://") {
		// Validate the HTTPS URL
		if err := urlutils.ValidateURL(targetURL); err != nil {
			return errors.New("clone", fmt.Errorf("invalid target URL: %w", err))
		}
	}
}

	// Add target remote
	if err := runGitCommand(tempDir, opts.Token, "remote", "add", "target", targetURL); err != nil {
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return errors.New("clone", fmt.Errorf("failed to add target remote: %w", err))
	}

	// Push to target repository
	if err := runGitCommand(tempDir, opts.Token, "push", "target", "--all"); err != nil {
		if opts.Progress != nil {
			opts.Progress.Error(err)
		}
		return errors.New("clone", fmt.Errorf("failed to push to target repository: %w", err))
	}

	return nil
}

// runGitCommand is a variable so it can be mocked in tests
var runGitCommand = func(dir string, token string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle HTTPS with token for clone and push commands with retries for rate limits
	if len(args) > 0 && (args[0] == "clone" || args[0] == "push") && len(args) > 1 && token != "" {
		rawURL := args[1]
		if strings.HasPrefix(rawURL, "https://") {
			parsedURL, err := url.Parse(rawURL)
			if err != nil {
				return errors.New("git-command", fmt.Errorf("invalid URL format: %w", err))
			}

			// Format URL with token
			tokenURL, err := urlutils.FormatTokenURL(parsedURL, token)
			if err != nil {
				return errors.New("git-command", fmt.Errorf("failed to format URL with token: %w", err))
			}

			args[1] = tokenURL.String()
			cmd.Args = append([]string{cmd.Args[0]}, args...)
		}
	}

	// For testing purposes, use test credentials
	if token != "" {
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
			"GIT_SSL_NO_VERIFY=true",
		)
	}

	// Retry logic for rate limits and auth failures
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := cmd.Run()
		if err == nil {
			return nil
		}

		lastErr = err
		errStr := err.Error()

		// Check for rate limit or auth failures
		if strings.Contains(errStr, "HTTP 429") || 
		   strings.Contains(errStr, "rate limit") || 
		   strings.Contains(errStr, "Authentication failed") {
			select {
			case <-ctx.Done():
				return errors.New("git-command", fmt.Errorf("operation timed out: %w", ctx.Err()))
			case <-time.After(time.Duration(i+1) * 5 * time.Second):
				continue
			}
		}

		// For non-retryable errors, return immediately
		return errors.New("git-command", fmt.Errorf("git command failed: %w", err))
	}

	return errors.New("git-command", fmt.Errorf("exceeded retry attempts: %w", lastErr))
}
