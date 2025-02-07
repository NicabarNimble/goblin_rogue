package progress

import (
	"fmt"
	"time"
)

// Tracker interface defines methods for tracking operation progress
type Tracker interface {
	Start(operation string) *Operation
	Update(current, total int64)
	Complete()
	Error(err error)
}

// Operation represents a tracked operation
type Operation struct {
	Name          string
	StartTime     time.Time
	Status        string
	LastUpdate    time.Time
	LastCurrent   int64
	LastTotal     int64
	ProgressRate  float64 // operations per second
	RateHistory   []float64
	EstimatedETA  time.Time
}

const (
	rateHistorySize = 10 // Keep last 10 rate measurements for averaging
)

// DefaultTracker provides a basic implementation of the Tracker interface
type DefaultTracker struct {
	CurrentOperation *Operation
}

// Start begins tracking a new operation
func (t *DefaultTracker) Start(operation string) *Operation {
	now := time.Now()
	t.CurrentOperation = &Operation{
		Name:         operation,
		StartTime:    now,
		LastUpdate:   now,
		Status:       "in_progress",
		RateHistory:  make([]float64, 0, rateHistorySize),
	}
	return t.CurrentOperation
}

// Complete marks the operation as completed
func (t *DefaultTracker) Complete() {
	if t.CurrentOperation != nil {
		t.CurrentOperation.Status = "completed"
	}
}

// Error marks the operation as failed with an error
func (t *DefaultTracker) Error(err error) {
	if t.CurrentOperation != nil {
		t.CurrentOperation.Status = "failed"
	}
}

// Update updates the progress of the current operation
func (t *DefaultTracker) Update(current, total int64) {
	if t.CurrentOperation == nil {
		return
	}

	now := time.Now()

	// Calculate progress rate (ops/sec) if we have previous data
	if t.CurrentOperation.LastCurrent > 0 {
		timeDiff := now.Sub(t.CurrentOperation.LastUpdate).Seconds()
		if timeDiff > 0 {
			itemsDiff := float64(current - t.CurrentOperation.LastCurrent)
			currentRate := itemsDiff / timeDiff

			// Add to rate history
			if len(t.CurrentOperation.RateHistory) >= rateHistorySize {
				// Remove oldest rate
				t.CurrentOperation.RateHistory = t.CurrentOperation.RateHistory[1:]
			}
			t.CurrentOperation.RateHistory = append(t.CurrentOperation.RateHistory, currentRate)

			// Calculate average rate
			var totalRate float64
			for _, rate := range t.CurrentOperation.RateHistory {
				totalRate += rate
			}
			t.CurrentOperation.ProgressRate = totalRate / float64(len(t.CurrentOperation.RateHistory))

			// Calculate ETA
			if t.CurrentOperation.ProgressRate > 0 {
				remainingItems := float64(total - current)
				remainingSeconds := remainingItems / t.CurrentOperation.ProgressRate
				t.CurrentOperation.EstimatedETA = now.Add(time.Duration(remainingSeconds) * time.Second)
			}
		}
	}

	// Update last values for next calculation
	t.CurrentOperation.LastUpdate = now
	t.CurrentOperation.LastCurrent = current
	t.CurrentOperation.LastTotal = total
}

// ConsoleTracker implements Tracker for console output
type ConsoleTracker struct {
	currentOperation *Operation
}

// NewConsoleTracker creates a new console-based progress tracker
func NewConsoleTracker() *ConsoleTracker {
	return &ConsoleTracker{}
}

// Start begins tracking a new operation
func (t *ConsoleTracker) Start(operation string) *Operation {
	now := time.Now()
	t.currentOperation = &Operation{
		Name:         operation,
		StartTime:    now,
		LastUpdate:   now,
		RateHistory:  make([]float64, 0, rateHistorySize),
	}
	fmt.Printf("Starting: %s\n", operation)
	return t.currentOperation
}

// Update updates the progress of the current operation
func (t *ConsoleTracker) Update(current, total int64) {
	if t.currentOperation == nil {
		return
	}

	now := time.Now()
	progress := float64(current) / float64(total)

	// Calculate progress rate (ops/sec) if we have previous data
	if t.currentOperation.LastCurrent > 0 {
		timeDiff := now.Sub(t.currentOperation.LastUpdate).Seconds()
		if timeDiff > 0 {
			itemsDiff := float64(current - t.currentOperation.LastCurrent)
			currentRate := itemsDiff / timeDiff

			// Add to rate history
			if len(t.currentOperation.RateHistory) >= rateHistorySize {
				// Remove oldest rate
				t.currentOperation.RateHistory = t.currentOperation.RateHistory[1:]
			}
			t.currentOperation.RateHistory = append(t.currentOperation.RateHistory, currentRate)

			// Calculate average rate
			var totalRate float64
			for _, rate := range t.currentOperation.RateHistory {
				totalRate += rate
			}
			t.currentOperation.ProgressRate = totalRate / float64(len(t.currentOperation.RateHistory))

			// Calculate ETA
			if t.currentOperation.ProgressRate > 0 {
				remainingItems := float64(total - current)
				remainingSeconds := remainingItems / t.currentOperation.ProgressRate
				t.currentOperation.EstimatedETA = now.Add(time.Duration(remainingSeconds) * time.Second)
			}
		}
	}

	// Update last values for next calculation
	t.currentOperation.LastUpdate = now
	t.currentOperation.LastCurrent = current
	t.currentOperation.LastTotal = total

	// Format ETA string
	etaStr := "calculating..."
	if !t.currentOperation.EstimatedETA.IsZero() {
		remaining := time.Until(t.currentOperation.EstimatedETA).Round(time.Second)
		if remaining > 0 {
			etaStr = remaining.String()
		} else {
			etaStr = "almost done"
		}
	}

	fmt.Printf("\r%s: %.2f%% (%.1f ops/sec, ETA: %s)",
		t.currentOperation.Name,
		progress*100,
		t.currentOperation.ProgressRate,
		etaStr)
}

// Complete marks the current operation as completed
func (t *ConsoleTracker) Complete() {
	if t.currentOperation == nil {
		return
	}
	duration := time.Since(t.currentOperation.StartTime)
	fmt.Printf("\nCompleted: %s (took %v)\n", t.currentOperation.Name, duration)
	t.currentOperation = nil
}

// Error marks the current operation as failed
func (t *ConsoleTracker) Error(err error) {
	if t.currentOperation == nil {
		return
	}
	fmt.Printf("\nError: %s - %v\n", t.currentOperation.Name, err)
	t.currentOperation = nil
}
