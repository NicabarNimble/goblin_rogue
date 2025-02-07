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

func TestConfigureCommandExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		args        []string
		initialCfg  *config.SyncConfig
		validate    func(*testing.T, *config.SyncConfig)
		expectError bool
	}{
		{
			name: "basic configuration",
			args: []string{
				"--source", "owner/source",
				"--target", "owner/target",
				"--schedule", "0 0 * * *",
			},
			validate: func(t *testing.T, cfg *config.SyncConfig) {
				assert.Equal(t, "owner/source", cfg.SourceRepo)
				assert.Equal(t, "owner/target", cfg.TargetRepo)
				assert.Equal(t, "0 0 * * *", cfg.Schedule)
			},
		},
		{
			name: "branch mappings",
			args: []string{
				"--branch", "main:master",
				"--branch", "dev:development",
			},
			initialCfg: &config.SyncConfig{
				SourceRepo: "owner/source",
				TargetRepo: "owner/target",
			},
			validate: func(t *testing.T, cfg *config.SyncConfig) {
				assert.Equal(t, map[string]string{
					"main": "master",
					"dev":  "development",
				}, cfg.BranchMappings)
			},
		},
		{
			name: "error handling configuration",
			args: []string{
				"--error-notify",
				"--notify-email", "test@example.com",
				"--retry-attempts", "5",
				"--retry-delay", "10m",
			},
			initialCfg: &config.SyncConfig{
				SourceRepo: "owner/source",
				TargetRepo: "owner/target",
			},
			validate: func(t *testing.T, cfg *config.SyncConfig) {
				assert.True(t, cfg.ErrorHandling.Notify)
				assert.Equal(t, "test@example.com", cfg.ErrorHandling.NotifyEmail)
				assert.Equal(t, 5, cfg.ErrorHandling.RetryAttempts)
				assert.Equal(t, "10m", cfg.ErrorHandling.RetryDelay)
			},
		},
		{
			name: "invalid source repo",
			args: []string{
				"--source", "invalid-format",
			},
			expectError: true,
		},
		{
			name: "invalid target repo",
			args: []string{
				"--target", "invalid-format",
			},
			expectError: true,
		},
		{
			name: "invalid schedule",
			args: []string{
				"--schedule", "invalid-cron",
			},
			expectError: true,
		},
		{
			name: "invalid branch mapping",
			args: []string{
				"--branch", "invalid-format",
			},
			expectError: true,
		},
		{
			name: "retry attempts too high",
			args: []string{
				"--retry-attempts", "11",
			},
			expectError: true,
		},
		{
			name: "notification without email",
			args: []string{
				"--error-notify",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique config file for each test
			configFile := filepath.Join(tempDir, tt.name+".json")

			// If there's initial config, write it
			if tt.initialCfg != nil {
				data, err := json.MarshalIndent(tt.initialCfg, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(configFile, data, 0644)
				require.NoError(t, err)
			}

			// Create command with test arguments
			cmd := newConfigureCmd()
			args := append(tt.args, "--config", configFile)
			cmd.SetArgs(args)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Read and validate the saved config
			data, err := os.ReadFile(configFile)
			require.NoError(t, err)

			var cfg config.SyncConfig
			err = json.Unmarshal(data, &cfg)
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, &cfg)
			}
		})
	}
}
