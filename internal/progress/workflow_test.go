package progress

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestWorkflowTracker(t *testing.T) {
	tracker := NewWorkflowTracker()

	// Test workflow start
	workflowID := int64(1234)
	runID := int64(5678)
	workflow := tracker.StartWorkflow("test-workflow", workflowID, runID)

	if workflow == nil {
		t.Fatal("Expected workflow to be created")
	}

	if workflow.WorkflowID != workflowID {
		t.Errorf("Expected workflow ID %d, got %d", workflowID, workflow.WorkflowID)
	}

	if workflow.RunID != runID {
		t.Errorf("Expected run ID %d, got %d", runID, workflow.RunID)
	}

	if workflow.Status != WorkflowQueued {
		t.Errorf("Expected initial status %s, got %s", WorkflowQueued, workflow.Status)
	}

	// Test status updates
	tracker.UpdateWorkflowStatus(WorkflowInProgress)
	if workflow.Status != WorkflowInProgress {
		t.Errorf("Expected status %s, got %s", WorkflowInProgress, workflow.Status)
	}

	// Test log streaming
	logData := "Test log line\nAnother log line"
	logReader := bytes.NewBufferString(logData)
	
	var capturedLogs []string
	logCallback := func(line string) {
		capturedLogs = append(capturedLogs, line)
	}

	tracker.SetLogStream(logReader, logCallback)
	if workflow.LogStream == nil {
		t.Error("Expected log stream to be set")
	}

	// Test workflow completion
	tracker.UpdateWorkflowStatus(WorkflowCompleted)
	if workflow.Status != WorkflowCompleted {
		t.Errorf("Expected status %s, got %s", WorkflowCompleted, workflow.Status)
	}

	// Test error handling
	testError := fmt.Errorf("test error")
	tracker.WorkflowError(testError)
	if workflow.Status != WorkflowFailed {
		t.Errorf("Expected status %s after error, got %s", WorkflowFailed, workflow.Status)
	}
}

func TestWorkflowTrackerNilSafety(t *testing.T) {
	tracker := NewWorkflowTracker()

	// These should not panic when no workflow is active
	tracker.UpdateWorkflowStatus(WorkflowInProgress)
	tracker.SetLogStream(bytes.NewBufferString("test"), func(string) {})
	tracker.WorkflowError(fmt.Errorf("test error"))

	workflow := tracker.GetCurrentWorkflow()
	if workflow != nil {
		t.Error("Expected nil workflow when none started")
	}
}

func TestWorkflowStatusTransitions(t *testing.T) {
	tracker := NewWorkflowTracker()
	workflow := tracker.StartWorkflow("status-test", 1, 1)

	statuses := []WorkflowStatus{
		WorkflowQueued,
		WorkflowInProgress,
		WorkflowCompleted,
	}

	for _, status := range statuses {
		tracker.UpdateWorkflowStatus(status)
		if workflow.Status != status {
			t.Errorf("Expected status %s, got %s", status, workflow.Status)
		}
	}
}

func TestWorkflowDuration(t *testing.T) {
	tracker := NewWorkflowTracker()
	workflow := tracker.StartWorkflow("duration-test", 1, 1)

	startTime := workflow.StartTime
	time.Sleep(time.Millisecond * 10) // Small delay to ensure measurable duration

	tracker.UpdateWorkflowStatus(WorkflowCompleted)
	duration := time.Since(startTime)

	if duration < time.Millisecond*10 {
		t.Error("Expected workflow duration to be at least 10ms")
	}
}
