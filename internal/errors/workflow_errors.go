package errors

import (
	"fmt"
	"net/http"
)

// WorkflowError represents an error that occurred during workflow operations
type WorkflowError struct {
	Op      string // Operation that failed
	Message string // Error message
	Status  int    // HTTP status code (if applicable)
	Err     error  // Underlying error
}

func (e *WorkflowError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s: %s (HTTP %d)", e.Op, e.Message, e.Status)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// NewWorkflowError creates a new WorkflowError
func NewWorkflowError(op, message string, err error) *WorkflowError {
	return &WorkflowError{
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// NewWorkflowHTTPError creates a new WorkflowError with HTTP status
func NewWorkflowHTTPError(op string, status int, message string, err error) *WorkflowError {
	return &WorkflowError{
		Op:      op,
		Status:  status,
		Message: message,
		Err:     err,
	}
}

// Common workflow error types
var (
	ErrWorkflowNotFound = &WorkflowError{
		Message: "workflow not found",
		Status:  http.StatusNotFound,
	}

	ErrWorkflowRunNotFound = &WorkflowError{
		Message: "workflow run not found",
		Status:  http.StatusNotFound,
	}

	ErrWorkflowDisabled = &WorkflowError{
		Message: "workflow is disabled",
		Status:  http.StatusConflict,
	}

	ErrWorkflowInProgress = &WorkflowError{
		Message: "workflow is already in progress",
		Status:  http.StatusConflict,
	}

	ErrInvalidWorkflowFile = &WorkflowError{
		Message: "invalid workflow file",
		Status:  http.StatusBadRequest,
	}

	ErrRateLimitExceeded = &WorkflowError{
		Message: "GitHub API rate limit exceeded",
		Status:  http.StatusTooManyRequests,
	}
)

// IsWorkflowError checks if an error is a WorkflowError
func IsWorkflowError(err error) bool {
	_, ok := err.(*WorkflowError)
	return ok
}

// IsNotFound checks if the error indicates a resource was not found
func IsNotFound(err error) bool {
	if we, ok := err.(*WorkflowError); ok {
		return we.Status == http.StatusNotFound
	}
	return false
}

// IsRateLimitExceeded checks if the error indicates rate limit was exceeded
func IsRateLimitExceeded(err error) bool {
	if we, ok := err.(*WorkflowError); ok {
		return we.Status == http.StatusTooManyRequests
	}
	return false
}

// IsRetryable checks if the error is potentially retryable
func IsRetryable(err error) bool {
	if we, ok := err.(*WorkflowError); ok {
		switch we.Status {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}
