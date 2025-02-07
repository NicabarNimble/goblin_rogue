package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/progress"
)

// TestTracker implements a progress tracker for testing
type TestTracker struct {
	progress.DefaultTracker
	operations []string
}

func (t *TestTracker) Start(operation string) *progress.Operation {
	t.operations = append(t.operations, operation)
	return &progress.Operation{
		Name:      operation,
		StartTime: time.Now(),
		Status:    "in_progress",
	}
}

// runCommand executes a command in the specified directory
func runCommand(dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", // Ignore global config
		"GIT_CONFIG_SYSTEM=/dev/null", // Ignore system config
		"GIT_SSL_NO_VERIFY=true",      // Skip SSL verification for tests
	)
	return cmd.Run()
}

// SetupGitConfig configures git for testing with HTTPS authentication
func SetupGitConfig(t *testing.T, dir string) {
	t.Helper()

	commands := [][]string{
		{"config", "user.name", "test"},
		{"config", "user.email", "test@example.com"},
		{"config", "init.defaultBranch", "main"},
	}

	// Add HTTPS configurations
	commands = append(commands,
		[]string{"config", "credential.helper", "store"},
		[]string{"config", "credential.useHttpPath", "true"},
	)

	for _, args := range commands {
		if err := runCommand(dir, "git", args...); err != nil {
			t.Fatalf("Failed to configure git %v: %v", args, err)
		}
	}
}

// SetupCredentials configures HTTPS authentication credentials for testing
func SetupCredentials(t *testing.T) {
	t.Helper()

	credentialsDir := filepath.Join(t.TempDir(), ".git-credentials")
	if err := os.MkdirAll(filepath.Dir(credentialsDir), 0700); err != nil {
		t.Fatalf("Failed to create credentials directory: %v", err)
	}
	// Create mock credentials file
	credentials := []byte("https://test:mock-token@github.com")
	if err := os.WriteFile(credentialsDir, credentials, 0600); err != nil {
		t.Fatalf("Failed to write credentials file: %v", err)
	}

	if err := runCommand("", "git", "config", "--global", "credential.helper", "store --file="+credentialsDir); err != nil {
		t.Fatalf("Failed to configure credential helper: %v", err)
	}
}

// SetupTestRepo creates and initializes a Git repository for testing
func SetupTestRepo(t *testing.T, name string) string {
	t.Helper()

	// Create test directory using t.TempDir()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Initialize git repo
	if err := runCommand(dir, "git", "init"); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Create test file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure git settings
	SetupGitConfig(t, dir)
	SetupCredentials(t)

	// Add and commit file
	if err := runCommand(dir, "git", "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}
	if err := runCommand(dir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}

	// Create and switch to main branch if not already on it
	if err := runCommand(dir, "git", "checkout", "-B", "main"); err != nil {
		t.Fatalf("Failed to create main branch: %v", err)
	}

	return dir
}

// SetupTestRepoWithRemote creates a test repo and sets up a remote
func SetupTestRepoWithRemote(t *testing.T, name, remoteName, remoteURL string) string {
	dir := SetupTestRepo(t, name)

	// Add remote
	if err := runCommand(dir, "git", "remote", "add", remoteName, remoteURL); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	return dir
}

// CreateBranch creates a new branch in the repository
func CreateBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	if err := runCommand(repoPath, "git", "checkout", "-b", branchName); err != nil {
		t.Fatalf("Failed to create branch %s: %v", branchName, err)
	}
}

// AddCommit creates a new commit in the repository
func AddCommit(t *testing.T, repoPath, fileName, content, message string) {
	t.Helper()

	filePath := filepath.Join(repoPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	if err := runCommand(repoPath, "git", "add", fileName); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := runCommand(repoPath, "git", "commit", "-m", message); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}
