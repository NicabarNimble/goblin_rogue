package gitutils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NicabarNimble/go-gittools/internal/github"
)

// Store original functions
var (
	originalRunGitCommand = runGitCommand
	originalOsExit       = osExit
)

type mockGitCommand struct {
	commands []string
}

func (m *mockGitCommand) run(dir string, args ...string) error {
	cmd := strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	return nil
}

type mockGitHubClient struct {
	shouldRepoExist bool
}

func (m *mockGitHubClient) CreateRepository(_ github.RepoOptions) error {
	if m.shouldRepoExist {
		return fmt.Errorf("repository already exists")
	}
	return nil
}

func TestCloneRepository(t *testing.T) {
	// Restore original functions after test
	defer func() {
		runGitCommand = originalRunGitCommand
		osExit = originalOsExit
	}()

	tests := []struct {
		name           string
		opts           CloneOptions
		repoExists     bool
		wantExitCode   int
		wantPushForce  bool
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "new repository clone",
			opts: CloneOptions{
				SourceURL: "https://github.com/source/repo.git",
				TargetURL: "https://github.com/target/repo.git",
				Token:     "test-token",
			},
			repoExists:    false,
			wantExitCode:  0,
			wantPushForce: false,
			wantErr:       false,
		},
		{
			name: "existing repository",
			opts: CloneOptions{
				SourceURL: "https://github.com/source/repo.git",
				TargetURL: "https://github.com/target/repo.git",
				Token:     "test-token",
			},
			repoExists:    true,
			wantExitCode:  2,
			wantPushForce: false,
			wantErr:       false,
		},
		{
			name: "missing source URL",
			opts: CloneOptions{
				TargetURL: "https://github.com/target/repo.git",
				Token:     "test-token",
			},
			repoExists:     false,
			wantExitCode:   0,
			wantPushForce:  false,
			wantErr:        true,
			wantErrMessage: "both source and target URLs must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mocks
			mock := &mockGitCommand{}
			runGitCommand = mock.run

			// Create a channel to capture exit code
			exitCodeChan := make(chan int, 1)
			osExit = func(code int) {
				exitCodeChan <- code
				panic("exit") // Use panic to stop execution as os.Exit would
			}

			// Run test and recover from expected panic
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if r != "exit" {
							t.Errorf("unexpected panic: %v", r)
						}
					}
				}()
				err = CloneRepository(tt.opts)
			}()

			// Check exit code if repository exists
			if tt.repoExists {
				select {
				case exitCode := <-exitCodeChan:
					if exitCode != tt.wantExitCode {
						t.Errorf("CloneRepository() exit code = %v, want %v", exitCode, tt.wantExitCode)
					}
				default:
					if tt.wantExitCode != 0 {
						t.Errorf("Expected os.Exit(%d) to be called", tt.wantExitCode)
					}
				}
			}

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrMessage) {
				t.Errorf("CloneRepository() error message = %v, want to contain %v", err, tt.wantErrMessage)
			}

			// Check if force flag was used in push command
			if !tt.repoExists {
				for _, cmd := range mock.commands {
					if strings.HasPrefix(cmd, "push") {
						hasForce := strings.Contains(cmd, "--force")
						if hasForce != tt.wantPushForce {
							t.Errorf("push command force flag = %v, want %v", hasForce, tt.wantPushForce)
						}
					}
				}
			}
		})
	}
}
