package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCommandExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-logs-test-*")
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

	// Create progress and logs directories
	progressDir := filepath.Join(tempDir, ".gitsync", "progress")
	logsDir := filepath.Join(tempDir, ".gitsync", "logs")
	require.NoError(t, os.MkdirAll(progressDir, 0755))
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Create test progress file
	now := time.Now()
	progress := progressEntry{
		RunID:     "test-run-1",
		Status:    progress.WorkflowCompleted,
		Branches:  []string{"main"},
		StartTime: now.Add(-1 * time.Hour).Format(time.RFC3339),
		EndTime:   now.Add(-30 * time.Minute).Format(time.RFC3339),
	}

	progressData, err := json.MarshalIndent(progress, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(progressDir, "test-run-1.json"), progressData, 0644)
	require.NoError(t, err)

	// Create test log files
	logEntries := []string{
		"[2025-02-01T12:00:00Z] Starting sync operation",
		"[2025-02-01T12:00:01Z] Cloning source repository",
		"[2025-02-01T12:00:05Z] Checking out main branch",
		"[2025-02-01T12:00:10Z] Pushing changes to target repository",
		"[2025-02-01T12:00:15Z] Sync completed successfully",
	}

	err = os.WriteFile(filepath.Join(logsDir, "test-run-1.log"), []byte(joinLogEntries(logEntries)), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		validate    func(*testing.T, string)
		expectError bool
	}{
		{
			name: "view logs for existing run",
			args: []string{
				"--config", configPath,
				"--run", "test-run-1",
			},
			validate: func(t *testing.T, output string) {
				for _, entry := range logEntries {
					assert.Contains(t, output, entry)
				}
			},
		},
		{
			name: "view logs with follow flag",
			args: []string{
				"--config", configPath,
				"--run", "test-run-1",
				"--follow",
			},
			validate: func(t *testing.T, output string) {
				for _, entry := range logEntries {
					assert.Contains(t, output, entry)
				}
			},
		},
		{
			name: "non-existent run ID",
			args: []string{
				"--config", configPath,
				"--run", "non-existent",
			},
			expectError: true,
		},
		{
			name: "missing run ID",
			args: []string{
				"--config", configPath,
			},
			expectError: true,
		},
		{
			name: "non-existent config",
			args: []string{
				"--config", "non-existent.json",
				"--run", "test-run-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newLogsCmd()
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

func TestLogsCommandWithFiltering(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gitsync-logs-filter-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create directories
	progressDir := filepath.Join(tempDir, ".gitsync", "progress")
	logsDir := filepath.Join(tempDir, ".gitsync", "logs")
	require.NoError(t, os.MkdirAll(progressDir, 0755))
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Create test log file with mixed log levels
	logEntries := []string{
		"[2025-02-01T12:00:00Z] INFO: Starting sync operation",
		"[2025-02-01T12:00:01Z] DEBUG: Initializing git client",
		"[2025-02-01T12:00:02Z] ERROR: Failed to authenticate",
		"[2025-02-01T12:00:03Z] WARN: Retrying operation",
		"[2025-02-01T12:00:04Z] INFO: Operation succeeded",
	}

	runID := "test-run-2"
	err = os.WriteFile(filepath.Join(logsDir, runID+".log"), []byte(joinLogEntries(logEntries)), 0644)
	require.NoError(t, err)

	// Create progress file
	progress := progressEntry{
		RunID:     runID,
		Status:    progress.WorkflowCompleted,
		StartTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
	}

	progressData, err := json.MarshalIndent(progress, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(progressDir, runID+".json"), progressData, 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		validate    func(*testing.T, string)
		expectError bool
	}{
		{
			name: "filter by error level",
			args: []string{
				"--run", runID,
				"--level", "error",
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "ERROR: Failed to authenticate")
				assert.NotContains(t, output, "INFO: Starting sync operation")
				assert.NotContains(t, output, "DEBUG: Initializing git client")
			},
		},
		{
			name: "filter by warning and error levels",
			args: []string{
				"--run", runID,
				"--level", "warn",
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "ERROR: Failed to authenticate")
				assert.Contains(t, output, "WARN: Retrying operation")
				assert.NotContains(t, output, "INFO: Starting sync operation")
			},
		},
		{
			name: "filter by time range",
			args: []string{
				"--run", runID,
				"--since", "2025-02-01T12:00:02Z",
				"--until", "2025-02-01T12:00:03Z",
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "ERROR: Failed to authenticate")
				assert.Contains(t, output, "WARN: Retrying operation")
				assert.NotContains(t, output, "INFO: Starting sync operation")
				assert.NotContains(t, output, "INFO: Operation succeeded")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newLogsCmd()
			args := append([]string{"--config", filepath.Join(tempDir, "config.json")}, tt.args...)
			cmd.SetArgs(args)

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

func joinLogEntries(entries []string) string {
	return strings.Join(entries, "\n")
}
