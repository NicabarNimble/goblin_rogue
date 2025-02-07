package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func Test_parseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "valid year",
			input:   "1y",
			want:    365 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "valid days",
			input:   "30d",
			want:    30 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "valid hours",
			input:   "24h",
			want:    24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupTokenTimeout(t *testing.T) {
	// Save original os.Exit and restore it after the test
	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	var exitCode int
	osExit = func(code int) {
		exitCode = code
		panic(fmt.Sprintf("os.Exit(%d)", code))
	}

	t.Run("timeout when no input provided", func(t *testing.T) {
		// Reset global variables
		value = ""
		nonInteractive = false
		exitCode = 0

		// Create a mock cobra command
		cmd := &cobra.Command{}

		// Capture panic
		defer func() {
			r := recover()
			if r == nil {
				t.Error("setupToken() expected timeout error but got none")
			}
			if exitCode != 1 {
				t.Errorf("setupToken() expected exit code 1 on timeout, got %d", exitCode)
			}
		}()

		setupToken(cmd, nil)
	})
}

func TestSetupCommand(t *testing.T) {
	// Save original os.Exit and restore it after the test
	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	var exitCode int
	osExit = func(code int) {
		exitCode = code
		panic(fmt.Sprintf("os.Exit(%d)", code))
	}

	tests := []struct {
		name    string
		token   string
		expires string
		wantErr bool
	}{
		{
			name:    "valid github token",
			token:   "ghp_test123456789",
			expires: "30d",
			wantErr: false,
		},
		{
			name:    "valid gitlab token",
			token:   "glpat-test123456789",
			expires: "1y",
			wantErr: false,
		},
		{
			name:    "invalid token format",
			token:   "invalid-token",
			expires: "30d",
			wantErr: true,
		},
		{
			name:    "invalid expiration",
			token:   "ghp_test123456789",
			expires: "invalid",
			wantErr: true,
		},
		{
			name:    "token too short",
			token:   "ghp_123",
			expires: "30d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables and exit code before each test
			value = tt.token
			expires = tt.expires
			exitCode = 0

			// Create a mock cobra command
			cmd := &cobra.Command{}

			// Capture panic to test error cases
			defer func() {
				r := recover()
				if tt.wantErr {
					if r == nil {
						t.Error("setupToken() expected error but got none")
					}
					if exitCode != 1 {
						t.Errorf("setupToken() expected exit code 1, got %d", exitCode)
					}
				} else {
					if r != nil {
						t.Errorf("setupToken() unexpected error: %v", r)
					}
					if exitCode != 0 {
						t.Errorf("setupToken() unexpected exit code %d", exitCode)
					}
				}
			}()

			setupToken(cmd, nil)
		})
	}
}
