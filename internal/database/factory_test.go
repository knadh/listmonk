package database

import (
	"context"
	"testing"

	"github.com/knadh/listmonk/internal/repository"
)

// mockAdapter implements the Adapter interface for testing
type mockAdapter struct {
	dbType DatabaseType
}

func (m *mockAdapter) Connect(ctx context.Context, config Config) error {
	return nil
}

func (m *mockAdapter) Close() error {
	return nil
}

func (m *mockAdapter) Ping(ctx context.Context) error {
	return nil
}

func (m *mockAdapter) BeginTx(ctx context.Context) (repository.Transaction, error) {
	return nil, nil
}

func (m *mockAdapter) GetCurrentVersion(ctx context.Context) (string, error) {
	return "1.0.0", nil
}

func (m *mockAdapter) Migrate(ctx context.Context, targetVersion string) error {
	return nil
}

func (m *mockAdapter) GetManager() repository.Manager {
	return nil
}

func (m *mockAdapter) Type() DatabaseType {
	return m.dbType
}

func (m *mockAdapter) IsConnected() bool {
	return true
}

func TestFactoryRegisterAdapter(t *testing.T) {
	factory := NewFactory().(*factory)
	
	// Test successful registration
	err := factory.RegisterAdapter(PostgreSQL, func() Adapter {
		return &mockAdapter{dbType: PostgreSQL}
	})
	
	if err != nil {
		t.Errorf("Unexpected error registering adapter: %v", err)
	}
	
	// Test duplicate registration
	err = factory.RegisterAdapter(PostgreSQL, func() Adapter {
		return &mockAdapter{dbType: PostgreSQL}
	})
	
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
	
	expectedMsg := "adapter for database type postgresql is already registered"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestFactoryCreateAdapter(t *testing.T) {
	factory := NewFactory().(*factory)
	
	// Register adapter
	factory.RegisterAdapter(PostgreSQL, func() Adapter {
		return &mockAdapter{dbType: PostgreSQL}
	})
	
	// Test successful creation
	adapter, err := factory.CreateAdapter(PostgreSQL)
	if err != nil {
		t.Errorf("Unexpected error creating adapter: %v", err)
	}
	
	if adapter == nil {
		t.Error("Expected adapter, got nil")
	}
	
	if adapter.Type() != PostgreSQL {
		t.Errorf("Expected adapter type %s, got %s", PostgreSQL, adapter.Type())
	}
	
	// Test creation of unregistered type
	_, err = factory.CreateAdapter(MongoDB)
	if err == nil {
		t.Error("Expected error for unregistered adapter type")
	}
	
	expectedMsg := "no adapter registered for database type: mongodb"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestFactorySupportedTypes(t *testing.T) {
	factory := NewFactory().(*factory)
	
	// Initially no types
	types := factory.SupportedTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 supported types, got %d", len(types))
	}
	
	// Register adapters
	factory.RegisterAdapter(PostgreSQL, func() Adapter {
		return &mockAdapter{dbType: PostgreSQL}
	})
	
	factory.RegisterAdapter(MongoDB, func() Adapter {
		return &mockAdapter{dbType: MongoDB}
	})
	
	types = factory.SupportedTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(types))
	}
	
	// Check that both types are present
	found := make(map[DatabaseType]bool)
	for _, dbType := range types {
		found[dbType] = true
	}
	
	if !found[PostgreSQL] {
		t.Error("PostgreSQL not found in supported types")
	}
	
	if !found[MongoDB] {
		t.Error("MongoDB not found in supported types")
	}
}

func TestGlobalFactory(t *testing.T) {
	// Save original factory
	originalFactory := globalFactory
	defer func() {
		globalFactory = originalFactory
	}()
	
	// Set a new factory
	newFactory := NewFactory()
	SetFactory(newFactory)
	
	if GetFactory() != newFactory {
		t.Error("GetFactory() should return the factory set by SetFactory()")
	}
	
	// Test global functions
	err := RegisterAdapter(PostgreSQL, func() Adapter {
		return &mockAdapter{dbType: PostgreSQL}
	})
	
	if err != nil {
		t.Errorf("Unexpected error registering adapter: %v", err)
	}
	
	types := SupportedTypes()
	if len(types) != 1 || types[0] != PostgreSQL {
		t.Errorf("Expected 1 supported type (PostgreSQL), got %v", types)
	}
	
	adapter, err := CreateAdapter(PostgreSQL)
	if err != nil {
		t.Errorf("Unexpected error creating adapter: %v", err)
	}
	
	if adapter.Type() != PostgreSQL {
		t.Errorf("Expected adapter type %s, got %s", PostgreSQL, adapter.Type())
	}
}

func TestConcurrentAccess(t *testing.T) {
	factory := NewFactory().(*factory)
	
	// Test concurrent registration
	done := make(chan bool, 2)
	
	go func() {
		factory.RegisterAdapter(PostgreSQL, func() Adapter {
			return &mockAdapter{dbType: PostgreSQL}
		})
		done <- true
	}()
	
	go func() {
		factory.RegisterAdapter(MongoDB, func() Adapter {
			return &mockAdapter{dbType: MongoDB}
		})
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
	
	// Verify both adapters were registered
	types := factory.SupportedTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(types))
	}
	
	// Test concurrent creation
	done = make(chan bool, 2)
	
	go func() {
		_, err := factory.CreateAdapter(PostgreSQL)
		if err != nil {
			t.Errorf("Unexpected error creating PostgreSQL adapter: %v", err)
		}
		done <- true
	}()
	
	go func() {
		_, err := factory.CreateAdapter(MongoDB)
		if err != nil {
			t.Errorf("Unexpected error creating MongoDB adapter: %v", err)
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
}