package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-run-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a valid config file for testing
	validConfig := &config.SyncConfig{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		Schedule:   "0 0 * * *",
		BranchMappings: map[string]string{
			"main": "master",
			"dev":  "development",
		},
		ErrorHandling: config.ErrorConfig{
			RetryAttempts: 3,
			RetryDelay:    "5m",
			Notify:        true,
			NotifyEmail:   "test@example.com",
		},
	}

	validConfigPath := filepath.Join(tempDir, "valid-config.json")
	data, err := json.MarshalIndent(validConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(validConfigPath, data, 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		setupConfig func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "run with valid config",
			args: []string{"--config", validConfigPath},
			setupConfig: func(t *testing.T) string {
				return validConfigPath
			},
			expectError: false,
		},
		{
			name: "run with non-existent config",
			args: []string{"--config", "non-existent.json"},
			setupConfig: func(t *testing.T) string {
				return filepath.Join(tempDir, "non-existent.json")
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name: "run with invalid config format",
			args: []string{"--config", "invalid-config.json"},
			setupConfig: func(t *testing.T) string {
				path := filepath.Join(tempDir, "invalid-config.json")
				err := os.WriteFile(path, []byte("invalid json"), 0644)
				require.NoError(t, err)
				return path
			},
			expectError: true,
			errorMsg:    "failed to parse config",
		},
		{
			name: "run with missing source repo",
			args: []string{"--config", "missing-source.json"},
			setupConfig: func(t *testing.T) string {
				cfg := *validConfig
				cfg.SourceRepo = ""
				path := filepath.Join(tempDir, "missing-source.json")
				data, err := json.MarshalIndent(cfg, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(path, data, 0644)
				require.NoError(t, err)
				return path
			},
			expectError: true,
			errorMsg:    "invalid source repository",
		},
		{
			name: "run with missing target repo",
			args: []string{"--config", "missing-target.json"},
			setupConfig: func(t *testing.T) string {
				cfg := *validConfig
				cfg.TargetRepo = ""
				path := filepath.Join(tempDir, "missing-target.json")
				data, err := json.MarshalIndent(cfg, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(path, data, 0644)
				require.NoError(t, err)
				return path
			},
			expectError: true,
			errorMsg:    "invalid target repository",
		},
		{
			name: "run with invalid branch mapping",
			args: []string{"--config", "invalid-branch.json"},
			setupConfig: func(t *testing.T) string {
				cfg := *validConfig
				cfg.BranchMappings = map[string]string{
					"": "master", // Invalid empty source branch
				}
				path := filepath.Join(tempDir, "invalid-branch.json")
				data, err := json.MarshalIndent(cfg, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(path, data, 0644)
				require.NoError(t, err)
				return path
			},
			expectError: true,
			errorMsg:    "invalid branch mapping",
		},
		{
			name: "run with specific branches",
			args: []string{
				"--config", validConfigPath,
				"--branch", "main",
				"--branch", "dev",
			},
			setupConfig: func(t *testing.T) string {
				return validConfigPath
			},
			expectError: false,
		},
		{
			name: "run with non-existent branch",
			args: []string{
				"--config", validConfigPath,
				"--branch", "non-existent",
			},
			setupConfig: func(t *testing.T) string {
				return validConfigPath
			},
			expectError: true,
			errorMsg:    "branch not found in mappings",
		},
		{
			name: "run with dry run flag",
			args: []string{
				"--config", validConfigPath,
				"--dry-run",
			},
			setupConfig: func(t *testing.T) string {
				return validConfigPath
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.setupConfig(t) // Setup config file but we use the path from args

			cmd := newRunCmd()
			cmd.SetArgs(tt.args) // Args already contain the config path

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestRunCommandWithProgress(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-run-progress-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a valid config file
	validConfig := &config.SyncConfig{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		BranchMappings: map[string]string{
			"main": "master",
		},
	}

	configPath := filepath.Join(tempDir, "config.json")
	data, err := json.MarshalIndent(validConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Create progress directory
	progressDir := filepath.Join(tempDir, ".gitsync", "progress")
	err = os.MkdirAll(progressDir, 0755)
	require.NoError(t, err)

	// Test that run command creates progress file
	cmd := newRunCmd()
	cmd.SetArgs([]string{"--config", configPath})

	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify progress file was created
	files, err := os.ReadDir(progressDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, files)

	// Verify progress file format
	progressFile := filepath.Join(progressDir, files[0].Name())
	data, err = os.ReadFile(progressFile)
	assert.NoError(t, err)

	var progress struct {
		RunID     string   `json:"run_id"`
		Status    string   `json:"status"`
		Branches  []string `json:"branches"`
		StartTime string   `json:"start_time"`
		EndTime   string   `json:"end_time,omitempty"`
	}

	err = json.Unmarshal(data, &progress)
	assert.NoError(t, err)
	assert.NotEmpty(t, progress.RunID)
	assert.NotEmpty(t, progress.StartTime)
	assert.Equal(t, []string{"main"}, progress.Branches)
}
