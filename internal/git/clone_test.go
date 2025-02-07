package git

import (
	"testing"
)

func mockRunGitCommand(shouldFail bool) func(string, string, ...string) error {
	return func(dir string, token string, args ...string) error {
		if shouldFail {
			return &mockError{msg: "mock command failed"}
		}
		return nil
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestCloneRepository(t *testing.T) {
	// Save original runGitCommand and restore after test
	originalRunGitCommand := runGitCommand
	defer func() {
		runGitCommand = originalRunGitCommand
	}()

	// Set up mock
	runGitCommand = mockRunGitCommand(false)

	tests := []struct {
		name    string
		opts    CloneOptions
		wantErr bool
	}{
		{
			name: "basic clone",
			opts: CloneOptions{
				SourceURL:  "https://github.com/test/repo.git",
				WorkingDir: "testdata",
				Token:     "test-token",
			},
			wantErr: false,
		},
		{
			name: "missing source URL",
			opts: CloneOptions{
				WorkingDir: "testdata",
				Token:     "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing working dir and target URL",
			opts: CloneOptions{
				SourceURL: "https://github.com/test/repo.git",
				Token:    "test-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CloneRepository(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
