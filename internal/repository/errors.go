package repository

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound indicates that the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrDuplicate indicates that a resource with the same identifier already exists
	ErrDuplicate = errors.New("resource already exists")

	// ErrInvalidInput indicates that the provided input is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrConcurrentAccess indicates that the resource was modified by another process
	ErrConcurrentAccess = errors.New("concurrent access conflict")

	// ErrConnectionFailed indicates that the database connection failed
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrTransactionFailed indicates that a database transaction failed
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrUnsupportedOperation indicates that the operation is not supported by the database adapter
	ErrUnsupportedOperation = errors.New("operation not supported")
)

// DatabaseError wraps database-specific errors with additional context
type DatabaseError struct {
	Operation string
	Entity    string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s on %s: %v", e.Operation, e.Entity, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDuplicate checks if the error is a "duplicate" error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicate)
}

// IsInvalidInput checks if the error is an "invalid input" error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsConcurrentAccess checks if the error is a "concurrent access" error
func IsConcurrentAccess(err error) bool {
	return errors.Is(err, ErrConcurrentAccess)
}

// NewDatabaseError creates a new DatabaseError
func NewDatabaseError(operation, entity string, err error) *DatabaseError {
	return &DatabaseError{
		Operation: operation,
		Entity:    entity,
		Err:       err,
	}
}