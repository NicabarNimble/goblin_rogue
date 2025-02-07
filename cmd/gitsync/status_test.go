package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// progressEntry represents a progress file entry for testing
type progressEntry struct {
	RunID     string                `json:"run_id"`
	Status    progress.WorkflowStatus `json:"status"`
	Branches  []string              `json:"branches"`
	StartTime string                `json:"start_time"`
	EndTime   string                `json:"end_time,omitempty"`
	Error     string                `json:"error,omitempty"`
}

func TestStatusCommandExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-status-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a valid config file
	validConfig := &config.SyncConfig{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		BranchMappings: map[string]string{
			"main": "master",
			"dev":  "development",
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

	// Create test progress files
	now := time.Now()
	progressFiles := []struct {
		name     string
		progress progressEntry
	}{
		{
			name: "completed-sync.json",
			progress: progressEntry{
				RunID:     "run-1",
				Status:    progress.WorkflowCompleted,
				Branches:  []string{"main", "dev"},
				StartTime: now.Add(-1 * time.Hour).Format(time.RFC3339),
				EndTime:   now.Add(-30 * time.Minute).Format(time.RFC3339),
			},
		},
		{
			name: "active-sync.json",
			progress: progressEntry{
				RunID:     "run-2",
				Status:    progress.WorkflowInProgress,
				Branches:  []string{"main"},
				StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339),
			},
		},
		{
			name: "failed-sync.json",
			progress: progressEntry{
				RunID:     "run-3",
				Status:    progress.WorkflowFailed,
				Branches:  []string{"dev"},
				StartTime: now.Add(-2 * time.Hour).Format(time.RFC3339),
				EndTime:   now.Add(-2 * time.Hour).Format(time.RFC3339),
				Error:     "failed to push changes",
			},
		},
	}

	for _, pf := range progressFiles {
		data, err := json.MarshalIndent(pf.progress, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(progressDir, pf.name), data, 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		args        []string
		validate    func(*testing.T, string)
		expectError bool
	}{
		{
			name: "list all syncs",
			args: []string{"--config", configPath},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "run-1")
				assert.Contains(t, output, "run-2")
				assert.Contains(t, output, "run-3")
				assert.Contains(t, output, string(progress.WorkflowCompleted))
				assert.Contains(t, output, string(progress.WorkflowInProgress))
				assert.Contains(t, output, string(progress.WorkflowFailed))
			},
		},
		{
			name: "filter by status completed",
			args: []string{"--config", configPath, "--status", "completed"},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "run-1")
				assert.NotContains(t, output, "run-2")
				assert.NotContains(t, output, "run-3")
			},
		},
		{
			name: "filter by status failed",
			args: []string{"--config", configPath, "--status", "failed"},
			validate: func(t *testing.T, output string) {
				assert.NotContains(t, output, "run-1")
				assert.NotContains(t, output, "run-2")
				assert.Contains(t, output, "run-3")
				assert.Contains(t, output, "failed to push changes")
			},
		},
		{
			name: "filter by branch",
			args: []string{"--config", configPath, "--branch", "dev"},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "run-1")
				assert.NotContains(t, output, "run-2")
				assert.Contains(t, output, "run-3")
			},
		},
		{
			name: "filter by run ID",
			args: []string{"--config", configPath, "--run", "run-2"},
			validate: func(t *testing.T, output string) {
				assert.NotContains(t, output, "run-1")
				assert.Contains(t, output, "run-2")
				assert.NotContains(t, output, "run-3")
			},
		},
		{
			name:        "invalid status filter",
			args:        []string{"--config", configPath, "--status", "invalid"},
			expectError: true,
		},
		{
			name:        "non-existent config",
			args:        []string{"--config", "non-existent.json"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newStatusCmd()
			cmd.SetArgs(tt.args)

			// Capture command output
			output := new(bytes.Buffer)
			cmd.SetOut(output)

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, output.String())
			}
		})
	}
}

func TestStatusCommandWithNoProgress(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-status-empty-*")
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

	// Create empty progress directory
	progressDir := filepath.Join(tempDir, ".gitsync", "progress")
	err = os.MkdirAll(progressDir, 0755)
	require.NoError(t, err)

	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--config", configPath})

	// Capture command output
	output := new(bytes.Buffer)
	cmd.SetOut(output)

	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "No sync operations found")
}
