package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/NicabarNimble/go-gittools/internal/gitutils"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestGitCloneCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "No arguments provided",
			args:           []string{},
			expectedOutput: "Error: accepts 1 arg(s), received 0",
			expectError:    true,
		},
		{
			name:           "Too many arguments",
			args:           []string{"repo1", "repo2"},
			expectedOutput: "Error: accepts 1 arg(s), received 2",
			expectError:    true,
		},
		{
			name:           "Valid source URL",
			args:           []string{"https://github.com/user/repo"},
			expectedOutput: "Starting clone operation...\nSource: https://github.com/user/repo",
			expectError:    false,
		},
		{
			name:           "Valid source URL with custom name",
			args:           []string{"https://github.com/user/repo", "--name", "custom-repo"},
			expectedOutput: "Starting clone operation...\nSource: https://github.com/user/repo",
			expectError:    false,
		},
		{
			name:           "Help flag",
			args:           []string{"--help"},
			expectedOutput: "Clone public repositories to private repositories",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags before each test
			customName = ""
			token = ""

			// Create a new root command for testing
			cmd := &cobra.Command{
				Use:   "go-gitclone [source-repo-url]",
				Short: "Clone public repositories to private repositories",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := cloneRepository(args[0]); err != nil {
						fmt.Printf("Error: %v\n", err)
						return
					}
				},
			}

			cmd.Flags().StringVar(&customName, "name", "", "Custom name for the target repository")
			cmd.Flags().StringVar(&token, "token", "", "GitHub token for authentication")

			// Set args and capture output
			cmd.SetArgs(tt.args)
			output := captureOutput(func() {
				cmd.Execute()
			})

			// Check output contains expected string
			assert.Contains(t, output, tt.expectedOutput)

			// Additional assertions based on test case
			if tt.expectError {
				assert.Contains(t, output, "Error")
			}
		})
	}
}

func TestCloneRepository(t *testing.T) {
	// Save the original implementation
	originalCloneFunc := cloneFunc
	defer func() {
		// Restore the original implementation after the test
		cloneFunc = originalCloneFunc
	}()

	tests := []struct {
		name        string
		sourceURL   string
		customName  string
		expectError bool
		mockError   bool
	}{
		{
			name:        "Empty source URL",
			sourceURL:   "",
			customName:  "",
			expectError: true,
			mockError:   false,
		},
		{
			name:        "Valid source URL",
			sourceURL:   "https://github.com/user/repo",
			customName:  "",
			expectError: false,
			mockError:   false,
		},
		{
			name:        "Valid source URL with custom name",
			sourceURL:   "https://github.com/user/repo",
			customName:  "custom-repo",
			expectError: false,
			mockError:   false,
		},
		{
			name:        "Clone operation fails",
			sourceURL:   "https://github.com/user/repo",
			customName:  "",
			expectError: true,
			mockError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cloneCalled bool
			cloneFunc = func(opts gitutils.CloneOptions) error {
				cloneCalled = true
				if tt.mockError {
					return fmt.Errorf("mock clone error")
				}
				return nil
			}

			customName = tt.customName
			err := cloneRepository(tt.sourceURL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// For valid source URL cases, verify that clone was actually called
			if tt.sourceURL != "" {
				assert.True(t, cloneCalled, "CloneRepository should have been called")
			}
		})
	}
}
