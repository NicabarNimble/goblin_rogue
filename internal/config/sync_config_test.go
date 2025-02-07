package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "0 */6 * * *", cfg.Schedule)
	assert.Equal(t, map[string]string{"main": "main"}, cfg.BranchMappings)
	assert.Equal(t, 3, cfg.ErrorHandling.RetryAttempts)
	assert.Equal(t, "5m", cfg.ErrorHandling.RetryDelay)
	assert.False(t, cfg.ErrorHandling.Notify)
	assert.Empty(t, cfg.ErrorHandling.NotifyEmail)
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		content     string
		expectError bool
		validate    func(*testing.T, *SyncConfig)
	}{
		{
			name: "valid config",
			content: `{
				"source_repo": "owner/source",
				"target_repo": "owner/target",
				"schedule": "0 0 * * *",
				"branch_mappings": {
					"main": "main",
					"develop": "dev"
				},
				"error_handling": {
					"retry_attempts": 5,
					"retry_delay": "10m",
					"notify": true,
					"notify_email": "test@example.com"
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, cfg *SyncConfig) {
				assert.Equal(t, "owner/source", cfg.SourceRepo)
				assert.Equal(t, "owner/target", cfg.TargetRepo)
				assert.Equal(t, "0 0 * * *", cfg.Schedule)
				assert.Equal(t, map[string]string{
					"main":    "main",
					"develop": "dev",
				}, cfg.BranchMappings)
				assert.Equal(t, 5, cfg.ErrorHandling.RetryAttempts)
				assert.Equal(t, "10m", cfg.ErrorHandling.RetryDelay)
				assert.True(t, cfg.ErrorHandling.Notify)
				assert.Equal(t, "test@example.com", cfg.ErrorHandling.NotifyEmail)
			},
		},
		{
			name: "missing required fields",
			content: `{
				"schedule": "0 0 * * *"
			}`,
			expectError: true,
		},
		{
			name: "invalid json",
			content: `{
				"source_repo": "owner/source",
				"invalid json"
			}`,
			expectError: true,
		},
		{
			name: "notification without email",
			content: `{
				"source_repo": "owner/source",
				"target_repo": "owner/target",
				"schedule": "0 0 * * *",
				"branch_mappings": {"main": "main"},
				"error_handling": {
					"notify": true
				}
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, "config.json")
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			assert.NoError(t, err)

			cfg, err := LoadConfig(configPath)
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

func TestSaveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &SyncConfig{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		Schedule:   "0 0 * * *",
		BranchMappings: map[string]string{
			"main": "main",
		},
		ErrorHandling: ErrorConfig{
			RetryAttempts: 3,
			RetryDelay:    "5m",
		},
	}

	configPath := filepath.Join(tempDir, "subdir", "config.json")
	err = SaveConfig(cfg, configPath)
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Read and parse the saved file
	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	var savedCfg SyncConfig
	err = json.Unmarshal(data, &savedCfg)
	assert.NoError(t, err)

	assert.Equal(t, cfg.SourceRepo, savedCfg.SourceRepo)
	assert.Equal(t, cfg.TargetRepo, savedCfg.TargetRepo)
	assert.Equal(t, cfg.Schedule, savedCfg.Schedule)
	assert.Equal(t, cfg.BranchMappings, savedCfg.BranchMappings)
	assert.Equal(t, cfg.ErrorHandling.RetryAttempts, savedCfg.ErrorHandling.RetryAttempts)
	assert.Equal(t, cfg.ErrorHandling.RetryDelay, savedCfg.ErrorHandling.RetryDelay)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *SyncConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &SyncConfig{
				SourceRepo: "owner/source",
				TargetRepo: "owner/target",
				Schedule:   "0 0 * * *",
				BranchMappings: map[string]string{
					"main": "main",
				},
				ErrorHandling: ErrorConfig{
					RetryAttempts: 3,
					RetryDelay:    "5m",
				},
			},
			expectError: false,
		},
		{
			name: "missing source repo",
			config: &SyncConfig{
				TargetRepo: "owner/target",
				Schedule:   "0 0 * * *",
				BranchMappings: map[string]string{
					"main": "main",
				},
			},
			expectError: true,
		},
		{
			name: "missing target repo",
			config: &SyncConfig{
				SourceRepo: "owner/source",
				Schedule:   "0 0 * * *",
				BranchMappings: map[string]string{
					"main": "main",
				},
			},
			expectError: true,
		},
		{
			name: "missing schedule",
			config: &SyncConfig{
				SourceRepo: "owner/source",
				TargetRepo: "owner/target",
				BranchMappings: map[string]string{
					"main": "main",
				},
			},
			expectError: true,
		},
		{
			name: "empty branch mappings",
			config: &SyncConfig{
				SourceRepo:     "owner/source",
				TargetRepo:     "owner/target",
				Schedule:       "0 0 * * *",
				BranchMappings: map[string]string{},
			},
			expectError: true,
		},
		{
			name: "negative retry attempts",
			config: &SyncConfig{
				SourceRepo: "owner/source",
				TargetRepo: "owner/target",
				Schedule:   "0 0 * * *",
				BranchMappings: map[string]string{
					"main": "main",
				},
				ErrorHandling: ErrorConfig{
					RetryAttempts: -1,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeDefaults(t *testing.T) {
	cfg := &SyncConfig{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	cfg.MergeDefaults()

	def := DefaultConfig()
	assert.Equal(t, def.Schedule, cfg.Schedule)
	assert.Equal(t, def.BranchMappings, cfg.BranchMappings)
	assert.Equal(t, def.ErrorHandling.RetryAttempts, cfg.ErrorHandling.RetryAttempts)
	assert.Equal(t, def.ErrorHandling.RetryDelay, cfg.ErrorHandling.RetryDelay)
}
