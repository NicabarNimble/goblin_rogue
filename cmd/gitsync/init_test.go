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

func TestInitCommandExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		args        []string
		validate    func(*testing.T, string)
		expectError bool
	}{
		{
			name: "basic initialization",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
			},
			validate: func(t *testing.T, configPath string) {
				// Verify config file was created
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var cfg config.SyncConfig
				err = json.Unmarshal(data, &cfg)
				require.NoError(t, err)

				assert.Equal(t, "owner/source", cfg.SourceRepo)
				assert.Equal(t, "owner/target", cfg.TargetRepo)
				assert.Equal(t, config.DefaultConfig().Schedule, cfg.Schedule)
				assert.Equal(t, config.DefaultConfig().BranchMappings, cfg.BranchMappings)
			},
		},
		{
			name: "initialization with custom schedule",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
				"--schedule", "0 0 * * *",
			},
			validate: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var cfg config.SyncConfig
				err = json.Unmarshal(data, &cfg)
				require.NoError(t, err)

				assert.Equal(t, "0 0 * * *", cfg.Schedule)
			},
		},
		{
			name: "initialization with branch mappings",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
				"--branch", "main:master",
				"--branch", "dev:development",
			},
			validate: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var cfg config.SyncConfig
				err = json.Unmarshal(data, &cfg)
				require.NoError(t, err)

				assert.Equal(t, map[string]string{
					"main": "master",
					"dev":  "development",
				}, cfg.BranchMappings)
			},
		},
		{
			name: "missing source repository",
			args: []string{
				"--target", "owner/target",
			},
			expectError: true,
		},
		{
			name: "missing target repository",
			args: []string{
				"--source", "owner/source",
			},
			expectError: true,
		},
		{
			name: "invalid source repository format",
			args: []string{
				"--source", "invalid-format",
				"--target", "owner/target",
			},
			expectError: true,
		},
		{
			name: "invalid target repository format",
			args: []string{
				"--source", "owner/source",
				"--target", "invalid-format",
			},
			expectError: true,
		},
		{
			name: "invalid schedule format",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
				"--schedule", "invalid-cron",
			},
			expectError: true,
		},
		{
			name: "invalid branch mapping format",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
				"--branch", "invalid-format",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique config file for each test
			configFile := filepath.Join(tempDir, tt.name+".json")

			// Create command with test arguments
			cmd := newInitCmd()
			args := append(tt.args, "--config", configFile)
			cmd.SetArgs(args)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, configFile)
			}

			// Verify the config file exists
			_, err = os.Stat(configFile)
			assert.NoError(t, err)
		})
	}
}

func TestInitCommandWithExistingConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-init-existing-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.json")

	// Create an existing config file
	existingCfg := &config.SyncConfig{
		SourceRepo: "existing/source",
		TargetRepo: "existing/target",
		Schedule:   "0 0 * * *",
		BranchMappings: map[string]string{
			"main": "master",
		},
	}
	data, err := json.MarshalIndent(existingCfg, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configFile, data, 0644)
	require.NoError(t, err)

	// Try to initialize with new config
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--source", "new/source",
		"--target", "new/target",
		"--config", configFile,
	})

	// Should fail because config already exists
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Verify original config was not modified
	data, err = os.ReadFile(configFile)
	require.NoError(t, err)

	var cfg config.SyncConfig
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err)

	assert.Equal(t, existingCfg.SourceRepo, cfg.SourceRepo)
	assert.Equal(t, existingCfg.TargetRepo, cfg.TargetRepo)
	assert.Equal(t, existingCfg.Schedule, cfg.Schedule)
	assert.Equal(t, existingCfg.BranchMappings, cfg.BranchMappings)
}
