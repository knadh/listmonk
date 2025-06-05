package repository

import (
	"errors"
	"testing"
)

func TestDatabaseError(t *testing.T) {
	originalErr := errors.New("connection failed")
	dbErr := NewDatabaseError("create", "subscriber", originalErr)
	
	expected := "database error during create on subscriber: connection failed"
	if dbErr.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, dbErr.Error())
	}
	
	if !errors.Is(dbErr, originalErr) {
		t.Error("DatabaseError should unwrap to original error")
	}
}

func TestErrorCheckers(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checker  func(error) bool
		expected bool
	}{
		{"NotFound check with ErrNotFound", ErrNotFound, IsNotFound, true},
		{"NotFound check with other error", ErrDuplicate, IsNotFound, false},
		{"Duplicate check with ErrDuplicate", ErrDuplicate, IsDuplicate, true},
		{"Duplicate check with other error", ErrNotFound, IsDuplicate, false},
		{"InvalidInput check with ErrInvalidInput", ErrInvalidInput, IsInvalidInput, true},
		{"InvalidInput check with other error", ErrNotFound, IsInvalidInput, false},
		{"ConcurrentAccess check with ErrConcurrentAccess", ErrConcurrentAccess, IsConcurrentAccess, true},
		{"ConcurrentAccess check with other error", ErrNotFound, IsConcurrentAccess, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checker(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestWrappedErrors(t *testing.T) {
	originalErr := ErrNotFound
	dbErr := NewDatabaseError("get", "subscriber", originalErr)
	
	if !IsNotFound(dbErr) {
		t.Error("IsNotFound should work with wrapped errors")
	}
}

func TestDatabaseErrorFields(t *testing.T) {
	originalErr := errors.New("test error")
	dbErr := NewDatabaseError("update", "campaign", originalErr)
	
	if dbErr.Operation != "update" {
		t.Errorf("Expected operation 'update', got %q", dbErr.Operation)
	}
	
	if dbErr.Entity != "campaign" {
		t.Errorf("Expected entity 'campaign', got %q", dbErr.Entity)
	}
	
	if dbErr.Err != originalErr {
		t.Error("Expected wrapped error to match original")
	}
}