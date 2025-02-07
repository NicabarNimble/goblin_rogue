package progress

import (
	"errors"
	"testing"
	"time"
)

func TestDefaultTracker_Start(t *testing.T) {
	tracker := &DefaultTracker{}
	op := tracker.Start("test operation")

	if op == nil {
		t.Error("Expected non-nil operation")
	}
	if op.Name != "test operation" {
		t.Errorf("Expected operation name 'test operation', got '%s'", op.Name)
	}
	if op.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}
	if op.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", op.Status)
	}
	if len(op.RateHistory) != 0 {
		t.Errorf("Expected empty rate history, got length %d", len(op.RateHistory))
	}
}

func TestDefaultTracker_Update(t *testing.T) {
	tracker := &DefaultTracker{}
	tracker.Start("test operation")

	// Test initial update
	tracker.Update(50, 100)
	if tracker.CurrentOperation.LastCurrent != 50 {
		t.Errorf("Expected LastCurrent 50, got %d", tracker.CurrentOperation.LastCurrent)
	}
	if tracker.CurrentOperation.LastTotal != 100 {
		t.Errorf("Expected LastTotal 100, got %d", tracker.CurrentOperation.LastTotal)
	}

	// Test progress rate calculation
	time.Sleep(100 * time.Millisecond) // Wait to ensure measurable time difference
	tracker.Update(75, 100)
	if tracker.CurrentOperation.ProgressRate <= 0 {
		t.Error("Expected positive progress rate")
	}
	if len(tracker.CurrentOperation.RateHistory) == 0 {
		t.Error("Expected non-empty rate history")
	}
}

func TestDefaultTracker_Complete(t *testing.T) {
	tracker := &DefaultTracker{}
	tracker.Start("test operation")
	tracker.Update(100, 100)
	tracker.Complete()

	if tracker.CurrentOperation.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", tracker.CurrentOperation.Status)
	}
}

func TestDefaultTracker_Error(t *testing.T) {
	tracker := &DefaultTracker{}
	tracker.Start("test operation")
	testErr := errors.New("test error")
	tracker.Error(testErr)

	if tracker.CurrentOperation.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", tracker.CurrentOperation.Status)
	}
}

func TestDefaultTracker_EdgeCases(t *testing.T) {
	tracker := &DefaultTracker{}

	// Test updates without started operation
	tracker.Update(50, 100)
	if tracker.CurrentOperation != nil {
		t.Error("Expected nil operation when updating without start")
	}

	// Test complete without started operation
	tracker.Complete()
	if tracker.CurrentOperation != nil {
		t.Error("Expected nil operation when completing without start")
	}

	// Test error without started operation
	tracker.Error(errors.New("test error"))
	if tracker.CurrentOperation != nil {
		t.Error("Expected nil operation when error without start")
	}

	// Test rate history size limit
	tracker.Start("test operation")
	for i := 0; i < rateHistorySize+5; i++ {
		time.Sleep(10 * time.Millisecond)
		tracker.Update(int64(i), 100)
		if len(tracker.CurrentOperation.RateHistory) > rateHistorySize {
			t.Errorf("Rate history exceeded max size: %d", len(tracker.CurrentOperation.RateHistory))
		}
	}
}

func TestDefaultTracker_ETACalculation(t *testing.T) {
	tracker := &DefaultTracker{}
	tracker.Start("test operation")

	// Initial update
	tracker.Update(0, 100)
	if !tracker.CurrentOperation.EstimatedETA.IsZero() {
		t.Error("Expected zero ETA on first update")
	}

	// Multiple updates to establish a reliable rate
	for i := 1; i <= 5; i++ {
		time.Sleep(100 * time.Millisecond)
		tracker.Update(int64(i*20), 100)
	}

	if tracker.CurrentOperation.EstimatedETA.IsZero() {
		t.Error("Expected non-zero ETA after progress")
	}

	// Verify ETA is in the future (with 1ms buffer for timing precision)
	now := time.Now().Add(-time.Millisecond)
	if !tracker.CurrentOperation.EstimatedETA.After(now) {
		t.Errorf("Expected ETA %v to be after current time %v",
			tracker.CurrentOperation.EstimatedETA, now)
	}
}

func TestDefaultTracker_MultipleOperations(t *testing.T) {
	tracker := &DefaultTracker{}

	// First operation
	op1 := tracker.Start("operation 1")
	tracker.Update(50, 100)
	tracker.Complete()

	// Second operation
	op2 := tracker.Start("operation 2")
	if op1 == op2 {
		t.Error("Expected different operation instances")
	}

	tracker.Update(75, 100)
	if tracker.CurrentOperation.Name != "operation 2" {
		t.Error("Expected current operation to be 'operation 2'")
	}
}

func TestDefaultTracker_StatusTransitions(t *testing.T) {
	tracker := &DefaultTracker{}

	// Test normal flow: in_progress -> completed
	op := tracker.Start("normal operation")
	if op.Status != "in_progress" {
		t.Errorf("Expected initial status 'in_progress', got '%s'", op.Status)
	}

	tracker.Complete()
	if op.Status != "completed" {
		t.Errorf("Expected final status 'completed', got '%s'", op.Status)
	}

	// Test error flow: in_progress -> failed
	op = tracker.Start("failing operation")
	if op.Status != "in_progress" {
		t.Errorf("Expected initial status 'in_progress', got '%s'", op.Status)
	}

	tracker.Error(errors.New("test error"))
	if op.Status != "failed" {
		t.Errorf("Expected final status 'failed', got '%s'", op.Status)
	}
}

// ConsoleTracker Tests

func TestConsoleTracker_Start(t *testing.T) {
	tracker := NewConsoleTracker()
	op := tracker.Start("test operation")

	if op == nil {
		t.Error("Expected non-nil operation")
	}
	if op.Name != "test operation" {
		t.Errorf("Expected operation name 'test operation', got '%s'", op.Name)
	}
	if op.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}
	if len(op.RateHistory) != 0 {
		t.Errorf("Expected empty rate history, got length %d", len(op.RateHistory))
	}
}

func TestConsoleTracker_Update(t *testing.T) {
	tracker := NewConsoleTracker()
	tracker.Start("test operation")

	// Test initial update
	tracker.Update(50, 100)
	if tracker.currentOperation.LastCurrent != 50 {
		t.Errorf("Expected LastCurrent 50, got %d", tracker.currentOperation.LastCurrent)
	}
	if tracker.currentOperation.LastTotal != 100 {
		t.Errorf("Expected LastTotal 100, got %d", tracker.currentOperation.LastTotal)
	}

	// Test progress rate calculation
	time.Sleep(100 * time.Millisecond) // Wait to ensure measurable time difference
	tracker.Update(75, 100)
	if tracker.currentOperation.ProgressRate <= 0 {
		t.Error("Expected positive progress rate")
	}
	if len(tracker.currentOperation.RateHistory) == 0 {
		t.Error("Expected non-empty rate history")
	}
}

func TestConsoleTracker_Complete(t *testing.T) {
	tracker := NewConsoleTracker()
	tracker.Start("test operation")
	tracker.Update(100, 100)
	tracker.Complete()

	if tracker.currentOperation != nil {
		t.Error("Expected nil current operation after completion")
	}
}

func TestConsoleTracker_Error(t *testing.T) {
	tracker := NewConsoleTracker()
	tracker.Start("test operation")
	testErr := errors.New("test error")
	tracker.Error(testErr)

	if tracker.currentOperation != nil {
		t.Error("Expected nil current operation after error")
	}
}

func TestConsoleTracker_EdgeCases(t *testing.T) {
	tracker := NewConsoleTracker()

	// Test updates without started operation
	tracker.Update(50, 100)
	if tracker.currentOperation != nil {
		t.Error("Expected nil operation when updating without start")
	}

	// Test complete without started operation
	tracker.Complete()
	if tracker.currentOperation != nil {
		t.Error("Expected nil operation when completing without start")
	}

	// Test error without started operation
	tracker.Error(errors.New("test error"))
	if tracker.currentOperation != nil {
		t.Error("Expected nil operation when error without start")
	}

	// Test rate history size limit
	tracker.Start("test operation")
	for i := 0; i < rateHistorySize+5; i++ {
		time.Sleep(10 * time.Millisecond)
		tracker.Update(int64(i), 100)
		if len(tracker.currentOperation.RateHistory) > rateHistorySize {
			t.Errorf("Rate history exceeded max size: %d", len(tracker.currentOperation.RateHistory))
		}
	}
}

func TestConsoleTracker_ETACalculation(t *testing.T) {
	tracker := NewConsoleTracker()
	tracker.Start("test operation")

	// Initial update
	tracker.Update(0, 100)
	if !tracker.currentOperation.EstimatedETA.IsZero() {
		t.Error("Expected zero ETA on first update")
	}

	// Multiple updates to establish a reliable rate
	for i := 1; i <= 5; i++ {
		time.Sleep(100 * time.Millisecond)
		tracker.Update(int64(i*20), 100)
	}

	if tracker.currentOperation.EstimatedETA.IsZero() {
		t.Error("Expected non-zero ETA after progress")
	}

	// Verify ETA is in the future (with 1ms buffer for timing precision)
	now := time.Now().Add(-time.Millisecond)
	if !tracker.currentOperation.EstimatedETA.After(now) {
		t.Errorf("Expected ETA %v to be after current time %v",
			tracker.currentOperation.EstimatedETA, now)
	}
}

func TestConsoleTracker_MultipleOperations(t *testing.T) {
	tracker := NewConsoleTracker()

	// First operation
	op1 := tracker.Start("operation 1")
	tracker.Update(50, 100)
	tracker.Complete()

	// Second operation
	op2 := tracker.Start("operation 2")
	if op1 == op2 {
		t.Error("Expected different operation instances")
	}

	tracker.Update(75, 100)
	if tracker.currentOperation.Name != "operation 2" {
		t.Error("Expected current operation to be 'operation 2'")
	}
}
