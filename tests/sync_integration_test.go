package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncWorkflow(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Create test directories
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	configDir := filepath.Join(baseDir, ".gitsync")

	// Set up source repository
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, runCommand(sourceDir, "git", "init"))
	SetupGitConfig(t, sourceDir)

	// Create and set up main branch in source
	require.NoError(t, runCommand(sourceDir, "git", "checkout", "-b", "main"))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("test content"), 0644))
	require.NoError(t, runCommand(sourceDir, "git", "add", "test.txt"))
	require.NoError(t, runCommand(sourceDir, "git", "commit", "-m", "Initial commit"))

	// Create development branch with different content
	require.NoError(t, runCommand(sourceDir, "git", "checkout", "-b", "dev"))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "dev.txt"), []byte("development content"), 0644))
	require.NoError(t, runCommand(sourceDir, "git", "add", "dev.txt"))
	require.NoError(t, runCommand(sourceDir, "git", "commit", "-m", "Development commit"))

	// Set up target repository
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	require.NoError(t, runCommand(targetDir, "git", "init", "--bare"))
	SetupGitConfig(t, targetDir)

	// Create initial branches in target
	targetSetupDir := filepath.Join(baseDir, "target-setup")
	require.NoError(t, os.MkdirAll(targetSetupDir, 0755))
	require.NoError(t, runCommand(targetSetupDir, "git", "clone", "file://"+targetDir, "."))
	
	// Set up main branch
	require.NoError(t, runCommand(targetSetupDir, "git", "checkout", "-b", "master"))
	require.NoError(t, os.WriteFile(filepath.Join(targetSetupDir, "README.md"), []byte("Target repository"), 0644))
	require.NoError(t, runCommand(targetSetupDir, "git", "add", "README.md"))
	require.NoError(t, runCommand(targetSetupDir, "git", "commit", "-m", "Initial target commit"))
	require.NoError(t, runCommand(targetSetupDir, "git", "push", "-u", "origin", "master"))

	// Set up development branch
	require.NoError(t, runCommand(targetSetupDir, "git", "checkout", "-b", "development"))
	require.NoError(t, runCommand(targetSetupDir, "git", "push", "-u", "origin", "development"))

	// Create sync configuration
	syncConfig := &config.SyncConfig{
		SourceRepo: "file://" + sourceDir,
		TargetRepo: "file://" + targetDir,
		BranchMappings: map[string]string{
			"main": "master",
			"dev":  "development",
		},
		ErrorHandling: config.ErrorConfig{
			RetryAttempts: 3,
			RetryDelay:    "5s",
			Notify:        false,
		},
	}

	// Create config directory and save config
	require.NoError(t, os.MkdirAll(configDir, 0755))
	configPath := filepath.Join(configDir, "config.json")
	configData, err := json.MarshalIndent(syncConfig, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, configData, 0644))

	// Test init command
	require.NoError(t, runCommand("", "./bin/gitsync", "init",
		"--source", syncConfig.SourceRepo,
		"--target", syncConfig.TargetRepo,
		"--branch", "main:master",
		"--branch", "dev:development",
		"--config", configPath,
	))

	// Test run command
	require.NoError(t, runCommand("", "./bin/gitsync", "run",
		"--config", configPath,
	))

	// Wait briefly for sync to complete
	time.Sleep(2 * time.Second)

	// Test status command
	statusOutput, err := runCommandWithOutput("", "./bin/gitsync", "status",
		"--config", configPath,
	)
	require.NoError(t, err)
	assert.Contains(t, statusOutput, "completed")

	// Verify sync results
	verifyDir := filepath.Join(baseDir, "verify")
	require.NoError(t, runCommand("", "git", "clone", "-b", "master", "file://"+targetDir, verifyDir))

	// Verify main branch content
	content, err := os.ReadFile(filepath.Join(verifyDir, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Verify dev branch content
	require.NoError(t, runCommand(verifyDir, "git", "checkout", "development"))
	content, err = os.ReadFile(filepath.Join(verifyDir, "dev.txt"))
	require.NoError(t, err)
	assert.Equal(t, "development content", string(content))

	// Test logs command
	logsOutput, err := runCommandWithOutput("", "./bin/gitsync", "logs",
		"--config", configPath,
		"--run", "latest",
	)
	require.NoError(t, err)
	assert.Contains(t, logsOutput, "Sync completed successfully")

	// Test error handling by making target temporarily unavailable
	require.NoError(t, os.Chmod(targetDir, 0000))
	err = runCommand("", "./bin/gitsync", "run",
		"--config", configPath,
	)
	assert.Error(t, err)
	require.NoError(t, os.Chmod(targetDir, 0755))

	// Verify error was logged
	statusOutput, err = runCommandWithOutput("", "./bin/gitsync", "status",
		"--config", configPath,
		"--status", "failed",
	)
	require.NoError(t, err)
	assert.Contains(t, statusOutput, "failed")
}

func runCommandWithOutput(dir string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}
