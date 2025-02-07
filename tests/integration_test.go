package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/NicabarNimble/go-gittools/internal/git"
)

func testPublishWorkflow(t *testing.T) {
	t.Helper()

	// Create test directories
	baseDir := t.TempDir()
	privateDir := filepath.Join(baseDir, "private")
	publicDir := filepath.Join(baseDir, "public")

	// Set up private repository
	if err := os.MkdirAll(privateDir, 0755); err != nil {
		t.Fatalf("Failed to create private directory: %v", err)
	}
	if err := runCommand(privateDir, "git", "init"); err != nil {
		t.Fatalf("Failed to initialize private repository: %v", err)
	}
	SetupGitConfig(t, privateDir)
	
	// Create and set up main branch
	if err := runCommand(privateDir, "git", "checkout", "-b", "main"); err != nil {
		t.Fatalf("Failed to create main branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(privateDir, "test.txt"), []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := runCommand(privateDir, "git", "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}
	if err := runCommand(privateDir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}

	// Set up public repository
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("Failed to create public directory: %v", err)
	}
	if err := runCommand(publicDir, "git", "init", "--bare"); err != nil {
		t.Fatalf("Failed to initialize public repository: %v", err)
	}
	SetupGitConfig(t, publicDir)
	
	// Create a temporary clone of the public repo to set up the initial branch
	publicSetupDir := filepath.Join(baseDir, "public-setup")
	if err := os.MkdirAll(publicSetupDir, 0755); err != nil {
		t.Fatalf("Failed to create public setup directory: %v", err)
	}
	if err := runCommand(publicSetupDir, "git", "clone", "file://"+publicDir, "."); err != nil {
		t.Fatalf("Failed to clone public repository for setup: %v", err)
	}
	// Create and push an initial commit to set up the main branch
	if err := runCommand(publicSetupDir, "git", "checkout", "-b", "main"); err != nil {
		t.Fatalf("Failed to create main branch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicSetupDir, "README.md"), []byte("Initial commit"), 0644); err != nil {
		t.Fatalf("Failed to create README file: %v", err)
	}
	if err := runCommand(publicSetupDir, "git", "add", "README.md"); err != nil {
		t.Fatalf("Failed to add README file: %v", err)
	}
	if err := runCommand(publicSetupDir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	if err := runCommand(publicSetupDir, "git", "push", "-u", "origin", "main"); err != nil {
		t.Fatalf("Failed to push initial commit: %v", err)
	}

	// Create publish config
	cfg := &config.PublishConfig{
		PrivateRepo: privateDir,
		PublicFork:  publicDir,
		Branch:      "main",
	}

	// Initialize tracker
	tracker := &TestTracker{}

	// Create temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "gitpublish-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare clone options
	cloneOpts := git.CloneOptions{
		SourceURL:  "file://" + cfg.PrivateRepo,
		WorkingDir: tempDir,
		Progress:   tracker,
		Token:      "mock-token",
	}

	// Perform publish operations
	err = git.CloneRepository(cloneOpts)
	if err != nil {
		t.Fatalf("Failed to clone private repository: %v", err)
	}

	// Add public as remote and push
	if err := runCommand(tempDir, "git", "remote", "add", "public", "file://" + publicDir); err != nil {
		t.Fatalf("Failed to add public remote: %v", err)
	}

	// Push to public remote with force to handle any conflicts
	cmd := exec.Command("git", "push", "-f", "public", cfg.Branch)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to push to public fork: %v\nOutput: %s", err, output)
	}

	// Verify publish operation
	if len(tracker.operations) == 0 {
		t.Error("No operations tracked during publish")
	}

	// Clone public repo to verify contents
	publicCloneDir := filepath.Join(baseDir, "public-clone")
	if err := runCommand("", "git", "clone", publicDir, publicCloneDir); err != nil {
		t.Fatalf("Failed to clone public repository: %v", err)
	}

	// Debug: List branches and current branch
	cmd = exec.Command("git", "branch", "-a")
	cmd.Dir = publicCloneDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list branches: %v\nOutput: %s", err, output)
	}
	t.Logf("Available branches:\n%s", output)

	// Ensure we're on the main branch
	if err := runCommand(publicCloneDir, "git", "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main branch: %v", err)
	}

	// Debug: List files in the directory
	files, err := os.ReadDir(publicCloneDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	t.Log("Files in public clone:")
	for _, file := range files {
		t.Logf("- %s", file.Name())
	}

	// Verify repository contents
	publicTestFile := filepath.Join(publicCloneDir, "test.txt")
	content, err := os.ReadFile(publicTestFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Unexpected file content: %s", content)
	}
}

func TestPublishWorkflow(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	testPublishWorkflow(t)
}
