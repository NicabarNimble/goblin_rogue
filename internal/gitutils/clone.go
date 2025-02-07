package gitutils

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/token"
)

// CloneOptions contains configuration for repository cloning
type CloneOptions struct {
	SourceURL  string
	TargetURL  string // Optional: will be auto-generated if not provided
	WorkingDir string
	Verbose    bool
	Token      string
	CustomName string // Optional: custom repository name
}

// progressWriter wraps an io.Writer to provide custom output formatting
type progressWriter struct {
	prefix string
	w      io.Writer
}

func newProgressWriter(prefix string, w io.Writer) *progressWriter {
	return &progressWriter{prefix: prefix, w: w}
}

var (
	// Match lines like:
	// Receiving objects:  67% (35484/52960), 236.76 MiB | 78.92 MiB/s
	// Receiving objects: 100% (52960/52960), 298.63 MiB | 81.39 MiB/s, done.
	progressRegex = regexp.MustCompile(`(?:Receiving objects|Resolving deltas):\s*(\d+)%\s*\((\d+)/(\d+)\)(?:,\s*([\d.]+)\s*([^|]+)\|\s*([\d.]+)\s*([^,\n]+))?`)
	// Match completion lines like:
	// Receiving objects: 100% (52960/52960), 298.63 MiB | 81.39 MiB/s, done.
	completionRegex = regexp.MustCompile(`(?:Receiving objects|Resolving deltas):\s*100%.*,\s*([\d.]+)\s*([^|]+).*done`)
)

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "remote: ") {
			line = strings.TrimPrefix(line, "remote: ")
		}

		// Skip the "Cloning into" message as we have our own header
		if strings.HasPrefix(line, "Cloning into") {
			continue
		}

		// Check for completion message first
		if matches := completionRegex.FindStringSubmatch(line); matches != nil {
			size := matches[1]
			unit := strings.TrimSpace(matches[2])
			fmt.Fprintf(pw.w, "%s100%% (Total size: %s %s)\n", pw.prefix, size, unit)
			continue
		}

		// Handle progress lines
		if matches := progressRegex.FindStringSubmatch(line); matches != nil {
			percentage := matches[1]
			current := matches[2]
			total := matches[3]

			// If we have size information (matches[4] through matches[7])
			if len(matches) > 4 && matches[4] != "" {
				size := matches[4]
				sizeUnit := strings.TrimSpace(matches[5])
				speed := matches[6]
				speedUnit := strings.TrimSpace(matches[7])
				fmt.Fprintf(pw.w, "%s%s%% (%s/%s) Size: %s %s, Speed: %s %s\n",
					pw.prefix, percentage, current, total, size, sizeUnit, speed, speedUnit)
			} else {
				fmt.Fprintf(pw.w, "%s%s%% (%s/%s)\n",
					pw.prefix, percentage, current, total)
			}
			continue
		}

		// For all other lines, just prefix them
		fmt.Fprintf(pw.w, "%s%s\n", pw.prefix, line)
	}
	return len(p), nil
}

// extractRepoInfo extracts owner and repo name from a GitHub URL
func extractRepoInfo(repoURL string) (owner string, name string, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	owner = parts[0]
	name = strings.TrimSuffix(parts[1], ".git")
	return owner, name, nil
}

// constructTargetRepoName generates the target repository name
func constructTargetRepoName(sourceName string, customName string) string {
	if customName != "" {
		return customName
	}
	return fmt.Sprintf("private-%s", sourceName)
}

// constructTargetURL generates the target repository URL
func constructTargetURL(username string, repoName string) string {
	return fmt.Sprintf("https://github.com/%s/%s.git", username, repoName)
}

// getStoredToken attempts to retrieve a stored GitHub token with retries
func getStoredToken() (string, error) {
	envStorage := token.NewEnvStorage()
	maxAttempts := 2
	delayBetweenAttempts := time.Second

	fmt.Println("Attempting to retrieve GitHub token...")

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		storedToken, err := envStorage.Retrieve(context.Background(), string(token.ProviderGitHub))
		if err == nil {
			return storedToken.Value, nil
		}

		if err == token.ErrTokenNotFound {
			if attempt == maxAttempts {
				return "", nil // Return empty string but no error if token not found after all attempts
			}
			time.Sleep(delayBetweenAttempts)
			continue
		}

		return "", fmt.Errorf("failed to retrieve stored token: %w", err)
	}

	return "", nil // This line should never be reached but is required for compilation
}

// runTokenSetup runs the token setup process and waits for completion
func runTokenSetup() (string, error) {
	fmt.Println("No GitHub token found. Starting token setup process...")

	// Prompt for token value
	fmt.Print("\nPlease enter your Git token: ")
	var tokenValue string
	fmt.Scanln(&tokenValue)

	if tokenValue == "" {
		return "", fmt.Errorf("no token provided")
	}

	// Create a token object and store it in our current process
	envStorage := token.NewEnvStorage()
	t, err := token.NewToken(tokenValue, time.Time{}, "repo workflow admin:repo")
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	if err := envStorage.Store(context.Background(), string(token.ProviderGitHub), *t); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return t.Value, nil
}

// CloneRepository clones a source repository to a target location
func CloneRepository(opts CloneOptions) error {
	if opts.SourceURL == "" {
		return fmt.Errorf("source URL must be specified")
	}

	fmt.Printf("\n🔄 Starting clone operation...\n")
	fmt.Printf("📂 Source: %s\n", opts.SourceURL)

	// If no token provided via flag, try to get stored token
	if opts.Token == "" {
		token, err := getStoredToken()
		if err != nil || token == "" {
			// No valid token found, run setup
			token, err = runTokenSetup()
			if err != nil {
				return fmt.Errorf("token setup failed: %w", err)
			}
		}
		opts.Token = token
	}

	// Verify we have a token
	if opts.Token == "" {
		return fmt.Errorf("GitHub token is required and could not be obtained")
	}

	t, err := token.NewToken(opts.Token, time.Time{}, "repo workflow admin:repo")
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	ghClient, err := github.NewClient(context.Background(), t)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Extract source repository information
	_, sourceName, err := extractRepoInfo(opts.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to parse source URL: %w", err)
	}

	// Generate target repository name and URL if not provided
	targetName := constructTargetRepoName(sourceName, opts.CustomName)
	if opts.TargetURL == "" {
		opts.TargetURL = constructTargetURL(ghClient.GetUsername(), targetName)
	}

	// Create the repository if it doesn't exist
	repoOpts := github.RepoOptions{
		Name:        targetName,
		Description: fmt.Sprintf("Private clone of %s", opts.SourceURL),
		Private:     true,
	}

	fmt.Printf("\n🔨 Creating private repository...\n")
	fmt.Printf("   %s\n", opts.TargetURL)

	if err := ghClient.CreateRepository(context.Background(), repoOpts); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			fmt.Printf("\n⚠️  Repository already exists at %s\n", opts.TargetURL)
			fmt.Printf("   For automated syncing, use gitsync with this repository\n")
			os.Exit(2) // Exit code 2 indicates repository exists
		}
		return fmt.Errorf("failed to create target repository: %w", err)
	}

	// Create temporary directory for initial clone
	tempDir, err := os.MkdirTemp("", "gitclone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone source repository
	if err := runGitCommand(tempDir, "clone", opts.SourceURL, "."); err != nil {
		return fmt.Errorf("failed to clone source repository: %w", err)
	}

	// Add target remote
	if err := runGitCommand(tempDir, "remote", "add", "target", opts.TargetURL); err != nil {
		return fmt.Errorf("failed to add target remote: %w", err)
	}

	// Set up authentication for the target repository
	targetWithAuth := strings.Replace(opts.TargetURL, "https://", fmt.Sprintf("https://%s@", opts.Token), 1)
	if err := runGitCommand(tempDir, "remote", "set-url", "target", targetWithAuth); err != nil {
		return fmt.Errorf("failed to set authenticated remote URL: %w", err)
	}

	// Configure git user for commits
	if err := runGitCommand(tempDir, "config", "user.name", "go-gitclone"); err != nil {
		return fmt.Errorf("failed to configure git user name: %w", err)
	}
	if err := runGitCommand(tempDir, "config", "user.email", "go-gitclone@github.com"); err != nil {
		return fmt.Errorf("failed to configure git user email: %w", err)
	}

	fmt.Printf("\n🔒 Removing workflow files for security...\n")
	// Remove workflow files before pushing
	if err := runGitCommand(tempDir, "rm", "-rf", ".github/workflows"); err != nil {
		// Ignore error if workflows directory doesn't exist
		if !strings.Contains(err.Error(), "pathspec '.github/workflows' did not match any files") {
			return fmt.Errorf("failed to remove workflow files: %w", err)
		}
	}

	// Commit the removal of workflow files if any were removed
	if err := runGitCommand(tempDir, "commit", "-m", "Remove workflow files for security", "--allow-empty"); err != nil {
		return fmt.Errorf("failed to commit workflow removal: %w", err)
	}

	// Push to target repository (without force flag)
	if err := runGitCommand(tempDir, "push", "-u", "target", "--all"); err != nil {
		return fmt.Errorf("failed to push to target repository: %w", err)
	}

	fmt.Printf("\n✨ Clone operation completed successfully!\n")
	return nil
}

// For testing purposes
var (
	runGitCommand = defaultRunGitCommand
	osExit       = os.Exit
)

func defaultRunGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	// Special handling for different git commands
	switch args[0] {
	case "clone":
		fmt.Printf("\n📦 Cloning repository...\n")
		cmd.Stdout = newProgressWriter("   ", os.Stdout)
		cmd.Stderr = newProgressWriter("   ", os.Stderr)
	case "rm":
		// Suppress output for rm command
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	case "push":
		fmt.Printf("\n📤 Pushing to target repository...\n")
		cmd.Stdout = newProgressWriter("   ", os.Stdout)
		cmd.Stderr = newProgressWriter("   ", os.Stderr)
	default:
		// For other commands, discard output unless there's an error
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "GIT_CREDENTIAL_HELPER=")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command failed: %w", err)
	}
	return nil
}
