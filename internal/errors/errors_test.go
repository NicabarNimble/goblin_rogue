package errors

import (
	"errors"
	"testing"
)

func TestOperationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		err      error
		expected string
	}{
		{
			name:     "with underlying error",
			op:       "clone",
			err:      errors.New("repository not found"),
			expected: "clone: repository not found",
		},
		{
			name:     "without underlying error",
			op:       "fetch",
			err:      nil,
			expected: "fetch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opErr := &OperationError{
				Op:  tt.op,
				Err: tt.err,
			}
			if got := opErr.Error(); got != tt.expected {
				t.Errorf("OperationError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOperationError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	opErr := &OperationError{
		Op:  "push",
		Err: underlying,
	}

	if got := opErr.Unwrap(); got != underlying {
		t.Errorf("OperationError.Unwrap() = %v, want %v", got, underlying)
	}
}

func TestNew(t *testing.T) {
	op := "pull"
	err := errors.New("network error")

	opErr := New(op, err)

	if opErr.Op != op {
		t.Errorf("New() Op = %v, want %v", opErr.Op, op)
	}
	if opErr.Err != err {
		t.Errorf("New() Err = %v, want %v", opErr.Err, err)
	}
}

func TestOperationError_Is(t *testing.T) {
	tests := []struct {
		name     string
		err1     *OperationError
		err2     error
		expected bool
	}{
		{
			name:     "matching operations",
			err1:     &OperationError{Op: "clone", Err: errors.New("error1")},
			err2:     &OperationError{Op: "clone", Err: errors.New("error2")},
			expected: true,
		},
		{
			name:     "different operations",
			err1:     &OperationError{Op: "clone", Err: errors.New("error")},
			err2:     &OperationError{Op: "push", Err: errors.New("error")},
			expected: false,
		},
		{
			name:     "different error types",
			err1:     &OperationError{Op: "clone", Err: errors.New("error")},
			err2:     errors.New("not an operation error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err1.Is(tt.err2); got != tt.expected {
				t.Errorf("OperationError.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}
