package git

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/progress"
)

// mockProgressTracker implements progress.Tracker for testing
type mockProgressTracker struct {
	started   bool
	completed bool
	lastError error
	operation *progress.Operation
}

func (m *mockProgressTracker) Start(operation string) *progress.Operation {
	m.started = true
	m.operation = &progress.Operation{
		Name:      operation,
		StartTime: time.Now(),
		Status:    "in_progress",
	}
	return m.operation
}

func (m *mockProgressTracker) Complete() {
	m.completed = true
	if m.operation != nil {
		m.operation.Status = "completed"
	}
}

func (m *mockProgressTracker) Error(err error) {
	m.lastError = err
	if m.operation != nil {
		m.operation.Status = "failed"
	}
}

func (m *mockProgressTracker) Update(current, total int64) {
	if m.operation != nil {
		m.operation.LastCurrent = current
		m.operation.LastTotal = total
	}
}

func TestCloneRepositoryExtended(t *testing.T) {
	// Save original runGitCommand and restore after test
	originalRunGitCommand := runGitCommand
	defer func() {
		runGitCommand = originalRunGitCommand
	}()

	tests := []struct {
		name           string
		opts           CloneOptions
		mockShouldFail bool
		mockError      string
		wantErr        bool
		checkProgress  bool
	}{
		{
			name: "context cancellation",
			opts: CloneOptions{
				SourceURL:  "https://github.com/test/repo.git",
				WorkingDir: "testdata",
				Token:     "test-token",
				Context:   func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel() // Cancel immediately
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "ssh url rejection",
			opts: CloneOptions{
				SourceURL:  "git@github.com:test/repo.git",
				WorkingDir: "testdata",
				Token:     "test-token",
			},
			wantErr: true,
		},
		{
			name: "rate limit retry",
			opts: CloneOptions{
				SourceURL:  "https://github.com/test/repo.git",
				WorkingDir: "testdata",
				Token:     "test-token",
			},
			mockShouldFail: true,
			mockError:      "HTTP 429 rate limit exceeded",
			wantErr:        true,
		},
		{
			name: "auth failure",
			opts: CloneOptions{
				SourceURL:  "https://github.com/test/repo.git",
				WorkingDir: "testdata",
				Token:     "invalid-token",
			},
			mockShouldFail: true,
			mockError:      "Authentication failed",
			wantErr:        true,
		},
		{
			name: "progress tracking",
			opts: CloneOptions{
				SourceURL:  "https://github.com/test/repo.git",
				WorkingDir: "testdata",
				Token:     "test-token",
				Progress:  &mockProgressTracker{},
			},
			wantErr:       false,
			checkProgress: true,
		},
		{
			name: "target url validation",
			opts: CloneOptions{
				SourceURL: "https://github.com/test/repo.git",
				TargetURL: "git@github.com:test/target.git", // Invalid SSH URL
				Token:    "test-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockShouldFail {
				runGitCommand = func(dir string, token string, args ...string) error {
					return fmt.Errorf(tt.mockError)
				}
			} else {
				runGitCommand = mockRunGitCommand(false)
			}

			err := CloneRepository(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneRepository() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkProgress {
				progress := tt.opts.Progress.(*mockProgressTracker)
				if !progress.started {
					t.Error("Progress tracking was not started")
				}
				if !progress.completed {
					t.Error("Progress tracking was not completed")
				}
				if tt.wantErr && progress.lastError == nil {
					t.Error("Expected error in progress tracking")
				}
			}
		})
	}
}

func TestCloneRepositoryTimeout(t *testing.T) {
	originalRunGitCommand := runGitCommand
	defer func() {
		runGitCommand = originalRunGitCommand
	}()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Mock a command that checks context cancellation
	runGitCommand = func(dir string, token string, args ...string) error {
		// Sleep briefly to ensure context gets cancelled
		time.Sleep(200 * time.Millisecond)
		return context.DeadlineExceeded
	}

	opts := CloneOptions{
		SourceURL:  "https://github.com/test/repo.git",
		WorkingDir: "testdata",
		Token:     "test-token",
		Context:   ctx,
	}

	err := CloneRepository(opts)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected deadline exceeded error, got: %v", err)
	}
}
