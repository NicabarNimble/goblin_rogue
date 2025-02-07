package progress

import (
	"fmt"
	"io"
	"time"
)

// WorkflowStatus represents the current status of a GitHub Actions workflow
type WorkflowStatus string

const (
	WorkflowQueued     WorkflowStatus = "queued"
	WorkflowInProgress WorkflowStatus = "in_progress"
	WorkflowCompleted  WorkflowStatus = "completed"
	WorkflowFailed     WorkflowStatus = "failed"
)

// WorkflowOperation represents a GitHub Actions workflow operation
type WorkflowOperation struct {
	*Operation
	RunID       int64
	WorkflowID  int64
	Status      WorkflowStatus
	LogStream   io.Reader
	LogCallback func(string) // Callback for handling log lines
}

// WorkflowTracker provides tracking specifically for GitHub Actions workflows
type WorkflowTracker struct {
	*ConsoleTracker
	currentWorkflow *WorkflowOperation
}

// NewWorkflowTracker creates a new tracker for GitHub Actions workflows
func NewWorkflowTracker() *WorkflowTracker {
	return &WorkflowTracker{
		ConsoleTracker: NewConsoleTracker(),
	}
}

// StartWorkflow begins tracking a new workflow operation
func (t *WorkflowTracker) StartWorkflow(name string, workflowID, runID int64) *WorkflowOperation {
	op := t.Start(name)
	t.currentWorkflow = &WorkflowOperation{
		Operation:  op,
		WorkflowID: workflowID,
		RunID:      runID,
		Status:     WorkflowQueued,
	}
	fmt.Printf("Starting workflow %s (Run ID: %d)\n", name, runID)
	return t.currentWorkflow
}

// UpdateWorkflowStatus updates the status of the current workflow
func (t *WorkflowTracker) UpdateWorkflowStatus(status WorkflowStatus) {
	if t.currentWorkflow == nil {
		return
	}

	t.currentWorkflow.Status = status
	statusStr := string(status)
	
	switch status {
	case WorkflowCompleted:
		duration := time.Since(t.currentWorkflow.StartTime)
		fmt.Printf("\nWorkflow completed successfully (took %v)\n", duration)
	case WorkflowFailed:
		duration := time.Since(t.currentWorkflow.StartTime)
		fmt.Printf("\nWorkflow failed (after %v)\n", duration)
	default:
		fmt.Printf("\rWorkflow status: %s", statusStr)
	}
}

// SetLogStream sets up log streaming for the current workflow
func (t *WorkflowTracker) SetLogStream(reader io.Reader, callback func(string)) {
	if t.currentWorkflow == nil {
		return
	}
	t.currentWorkflow.LogStream = reader
	t.currentWorkflow.LogCallback = callback
}

// GetCurrentWorkflow returns the current workflow operation being tracked
func (t *WorkflowTracker) GetCurrentWorkflow() *WorkflowOperation {
	return t.currentWorkflow
}

// WorkflowError handles workflow-specific errors
func (t *WorkflowTracker) WorkflowError(err error) {
	if t.currentWorkflow == nil {
		return
	}
	t.currentWorkflow.Status = WorkflowFailed
	fmt.Printf("\nWorkflow error: %v\n", err)
}
