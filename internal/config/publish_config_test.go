package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPublishConfig(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	validConfig := `{
		"privateRepo": "https://github.com/test/private-repo.git",
		"publicFork": "https://github.com/test/public-fork.git",
		"branch": "main",
		"token": "test-token"
	}`

	invalidConfig := `{
		"privateRepo": "",
		"publicFork": "",
		"branch": ""
	}`

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid config",
			content: validConfig,
			wantErr: false,
		},
		{
			name:    "invalid config",
			content: invalidConfig,
			wantErr: true,
		},
		{
			name:    "invalid json",
			content: "{invalid json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config file
			configPath := filepath.Join(tempDir, tt.name+".json")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Test loading config
			config, err := LoadPublishConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadPublishConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadPublishConfig() returned nil config without error")
			}

			// For valid config, verify default branch is set when not provided
			if !tt.wantErr && config != nil && config.Branch == "" {
				t.Error("LoadPublishConfig() did not set default branch")
			}
		})
	}

	// Test loading non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadPublishConfig(filepath.Join(tempDir, "nonexistent.json"))
		if err == nil {
			t.Error("LoadPublishConfig() expected error for non-existent file")
		}
	})
}

func TestSavePublishConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		config  *PublishConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &PublishConfig{
				PrivateRepo: "https://github.com/test/private-repo.git",
				PublicFork: "https://github.com/test/public-fork.git",
				Branch:     "main",
				Token:      "test-token",
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing private repo",
			config: &PublishConfig{
				PublicFork: "https://github.com/test/public-fork.git",
				Branch:     "main",
			},
			wantErr: true,
		},
		{
			name: "invalid config - missing public fork",
			config: &PublishConfig{
				PrivateRepo: "https://github.com/test/private-repo.git",
				Branch:     "main",
			},
			wantErr: true,
		},
		{
			name: "valid config - missing branch should use default",
			config: &PublishConfig{
				PrivateRepo: "https://github.com/test/private-repo.git",
				PublicFork: "https://github.com/test/public-fork.git",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tt.name+".json")
			err := tt.config.SavePublishConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SavePublishConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Error("SavePublishConfig() did not create config file")
				}

				// Try loading the saved config
				loaded, err := LoadPublishConfig(configPath)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
				}

				// Verify loaded config matches original
				if loaded.PrivateRepo != tt.config.PrivateRepo {
					t.Error("Loaded config does not match saved config")
				}

				// Verify default branch is set when not provided
				if tt.config.Branch == "" && loaded.Branch != "main" {
					t.Error("Default branch not set correctly")
				}
			}
		})
	}
}

func TestDefaultPublishConfig(t *testing.T) {
	config := DefaultPublishConfig()

	if config == nil {
		t.Fatal("DefaultPublishConfig() returned nil")
	}

	if config.Branch != "main" {
		t.Errorf("DefaultPublishConfig() branch = %v, want %v", config.Branch, "main")
	}
}
