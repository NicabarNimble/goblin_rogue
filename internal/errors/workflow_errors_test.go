package errors

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowError(t *testing.T) {
	tests := []struct {
		name           string
		err            *WorkflowError
		expectedString string
	}{
		{
			name: "error with status",
			err: &WorkflowError{
				Op:      "CreateWorkflow",
				Message: "failed to create workflow",
				Status:  http.StatusBadRequest,
			},
			expectedString: "CreateWorkflow: failed to create workflow (HTTP 400)",
		},
		{
			name: "error without status",
			err: &WorkflowError{
				Op:      "GetWorkflow",
				Message: "workflow not found",
			},
			expectedString: "GetWorkflow: workflow not found",
		},
		{
			name: "error with underlying error",
			err: &WorkflowError{
				Op:      "TriggerWorkflow",
				Message: "failed to trigger workflow",
				Status:  http.StatusInternalServerError,
				Err:     fmt.Errorf("network error"),
			},
			expectedString: "TriggerWorkflow: failed to trigger workflow (HTTP 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, tt.err.Error())
		})
	}
}

func TestNewWorkflowError(t *testing.T) {
	underlying := fmt.Errorf("underlying error")
	err := NewWorkflowError("TestOp", "test message", underlying)

	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, underlying, err.Err)
	assert.Equal(t, 0, err.Status)
}

func TestNewWorkflowHTTPError(t *testing.T) {
	underlying := fmt.Errorf("underlying error")
	err := NewWorkflowHTTPError("TestOp", http.StatusBadRequest, "test message", underlying)

	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, underlying, err.Err)
	assert.Equal(t, http.StatusBadRequest, err.Status)
}

func TestErrorUnwrap(t *testing.T) {
	underlying := fmt.Errorf("underlying error")
	err := NewWorkflowError("TestOp", "test message", underlying)

	unwrapped := err.Unwrap()
	assert.Equal(t, underlying, unwrapped)
}

func TestIsWorkflowError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "workflow error",
			err:      NewWorkflowError("TestOp", "test message", nil),
			expected: true,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsWorkflowError(tt.err))
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not found error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusNotFound, "not found", nil),
			expected: true,
		},
		{
			name:     "other workflow error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusBadRequest, "bad request", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNotFound(tt.err))
		})
	}
}

func TestIsRateLimitExceeded(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusTooManyRequests, "rate limit exceeded", nil),
			expected: true,
		},
		{
			name:     "other workflow error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusBadRequest, "bad request", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRateLimitExceeded(tt.err))
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusTooManyRequests, "rate limit exceeded", nil),
			expected: true,
		},
		{
			name:     "internal server error",
			err:      NewWorkflowHTTPError("TestOp", http.StatusInternalServerError, "server error", nil),
			expected: true,
		},
		{
			name:     "bad gateway",
			err:      NewWorkflowHTTPError("TestOp", http.StatusBadGateway, "bad gateway", nil),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      NewWorkflowHTTPError("TestOp", http.StatusServiceUnavailable, "service unavailable", nil),
			expected: true,
		},
		{
			name:     "gateway timeout",
			err:      NewWorkflowHTTPError("TestOp", http.StatusGatewayTimeout, "gateway timeout", nil),
			expected: true,
		},
		{
			name:     "bad request",
			err:      NewWorkflowHTTPError("TestOp", http.StatusBadRequest, "bad request", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *WorkflowError
		status int
	}{
		{
			name:   "workflow not found",
			err:    ErrWorkflowNotFound,
			status: http.StatusNotFound,
		},
		{
			name:   "workflow run not found",
			err:    ErrWorkflowRunNotFound,
			status: http.StatusNotFound,
		},
		{
			name:   "workflow disabled",
			err:    ErrWorkflowDisabled,
			status: http.StatusConflict,
		},
		{
			name:   "workflow in progress",
			err:    ErrWorkflowInProgress,
			status: http.StatusConflict,
		},
		{
			name:   "invalid workflow file",
			err:    ErrInvalidWorkflowFile,
			status: http.StatusBadRequest,
		},
		{
			name:   "rate limit exceeded",
			err:    ErrRateLimitExceeded,
			status: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.status, tt.err.Status)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}
