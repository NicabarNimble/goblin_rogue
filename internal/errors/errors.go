package errors

import "fmt"

// OperationError represents an error that occurred during a git operation
type OperationError struct {
	Op  string // The operation being performed
	Err error  // The underlying error
}

// Error implements the error interface
func (e *OperationError) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *OperationError) Unwrap() error {
	return e.Err
}

// New creates a new OperationError
func New(op string, err error) *OperationError {
	return &OperationError{
		Op:  op,
		Err: err,
	}
}

// Is implements error matching for OperationError
func (e *OperationError) Is(target error) bool {
	t, ok := target.(*OperationError)
	if !ok {
		return false
	}
	return e.Op == t.Op
}
