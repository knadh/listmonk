package database

import (
	"context"
	"testing"
)

// mockMigrationManager implements MigrationManager for testing
type mockMigrationManager struct{}

func (m *mockMigrationManager) GetCurrentVersion(ctx context.Context) (string, error) {
	return "1.0.0", nil
}

func (m *mockMigrationManager) GetAppliedMigrations(ctx context.Context) ([]string, error) {
	return []string{"1.0.0"}, nil
}

func (m *mockMigrationManager) GetPendingMigrations(ctx context.Context) ([]MigrationScript, error) {
	return []MigrationScript{}, nil
}

func (m *mockMigrationManager) ApplyMigration(ctx context.Context, script MigrationScript) error {
	return nil
}

func (m *mockMigrationManager) RollbackMigration(ctx context.Context, script MigrationScript) error {
	return nil
}

func (m *mockMigrationManager) MigrateToVersion(ctx context.Context, targetVersion string) error {
	return nil
}

func (m *mockMigrationManager) RegisterMigration(script MigrationScript) error {
	return nil
}

func TestMigrationScript(t *testing.T) {
	script := MigrationScript{
		Version:     "1.0.0",
		Description: "Initial migration",
		Up: func(ctx context.Context, adapter Adapter) error {
			return nil
		},
		Down: func(ctx context.Context, adapter Adapter) error {
			return nil
		},
	}
	
	if script.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", script.Version)
	}
	
	if script.Description != "Initial migration" {
		t.Errorf("Expected description 'Initial migration', got %s", script.Description)
	}
	
	// Test that Up and Down functions are callable
	ctx := context.Background()
	adapter := &mockAdapter{dbType: PostgreSQL}
	
	err := script.Up(ctx, adapter)
	if err != nil {
		t.Errorf("Up migration failed: %v", err)
	}
	
	err = script.Down(ctx, adapter)
	if err != nil {
		t.Errorf("Down migration failed: %v", err)
	}
}

func TestMigrationManagerInterface(t *testing.T) {
	var manager MigrationManager = &mockMigrationManager{}
	
	ctx := context.Background()
	
	// Test GetCurrentVersion
	version, err := manager.GetCurrentVersion(ctx)
	if err != nil {
		t.Errorf("GetCurrentVersion failed: %v", err)
	}
	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
	
	// Test GetAppliedMigrations
	migrations, err := manager.GetAppliedMigrations(ctx)
	if err != nil {
		t.Errorf("GetAppliedMigrations failed: %v", err)
	}
	if len(migrations) != 1 || migrations[0] != "1.0.0" {
		t.Errorf("Expected [1.0.0], got %v", migrations)
	}
	
	// Test GetPendingMigrations
	pending, err := manager.GetPendingMigrations(ctx)
	if err != nil {
		t.Errorf("GetPendingMigrations failed: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("Expected empty pending migrations, got %d", len(pending))
	}
	
	// Test RegisterMigration
	script := MigrationScript{
		Version:     "1.1.0",
		Description: "Test migration",
	}
	err = manager.RegisterMigration(script)
	if err != nil {
		t.Errorf("RegisterMigration failed: %v", err)
	}
	
	// Test MigrateToVersion
	err = manager.MigrateToVersion(ctx, "1.1.0")
	if err != nil {
		t.Errorf("MigrateToVersion failed: %v", err)
	}
}

func TestAdapterInterface(t *testing.T) {
	adapter := &mockAdapter{dbType: PostgreSQL}
	
	ctx := context.Background()
	config := Config{Type: PostgreSQL}
	
	// Test Connect
	err := adapter.Connect(ctx, config)
	if err != nil {
		t.Errorf("Connect failed: %v", err)
	}
	
	// Test Ping
	err = adapter.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
	
	// Test Type
	if adapter.Type() != PostgreSQL {
		t.Errorf("Expected type %s, got %s", PostgreSQL, adapter.Type())
	}
	
	// Test IsConnected
	if !adapter.IsConnected() {
		t.Error("Expected adapter to be connected")
	}
	
	// Test GetCurrentVersion
	version, err := adapter.GetCurrentVersion(ctx)
	if err != nil {
		t.Errorf("GetCurrentVersion failed: %v", err)
	}
	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
	
	// Test BeginTx
	tx, err := adapter.BeginTx(ctx)
	if err != nil {
		t.Errorf("BeginTx failed: %v", err)
	}
	if tx != nil {
		t.Error("Expected nil transaction from mock")
	}
	
	// Test GetManager
	manager := adapter.GetManager()
	if manager != nil {
		t.Error("Expected nil manager from mock")
	}
	
	// Test Close
	err = adapter.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestDatabaseTypes(t *testing.T) {
	tests := []struct {
		name   string
		dbType DatabaseType
	}{
		{"PostgreSQL", PostgreSQL},
		{"MongoDB", MongoDB},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dbType) == "" {
				t.Error("Database type should not be empty")
			}
		})
	}
}

func TestAdapterFactoryInterface(t *testing.T) {
	factory := NewFactory()
	
	// Test SupportedTypes initially empty
	types := factory.SupportedTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 supported types initially, got %d", len(types))
	}
	
	// Test CreateAdapter with unregistered type
	_, err := factory.CreateAdapter(PostgreSQL)
	if err == nil {
		t.Error("Expected error for unregistered adapter type")
	}
	
	// Test that we can work with the interface
	var factoryInterface AdapterFactory = factory
	if factoryInterface == nil {
		t.Error("Factory should implement AdapterFactory interface")
	}
}

// Test that interface types compile correctly
func TestInterfaceCompilation(t *testing.T) {
	var (
		_ Adapter           = (*mockAdapter)(nil)
		_ AdapterFactory    = (*factory)(nil)
		_ MigrationManager  = (*mockMigrationManager)(nil)
	)
}

func TestMigrationScriptValidation(t *testing.T) {
	// Test that migration scripts can be created with various configurations
	tests := []struct {
		name        string
		script      MigrationScript
		expectValid bool
	}{
		{
			name: "valid script with both up and down",
			script: MigrationScript{
				Version:     "1.0.0",
				Description: "Test migration",
				Up:          func(ctx context.Context, adapter Adapter) error { return nil },
				Down:        func(ctx context.Context, adapter Adapter) error { return nil },
			},
			expectValid: true,
		},
		{
			name: "valid script with only up",
			script: MigrationScript{
				Version:     "1.0.0",
				Description: "Test migration",
				Up:          func(ctx context.Context, adapter Adapter) error { return nil },
			},
			expectValid: true,
		},
		{
			name: "empty version",
			script: MigrationScript{
				Description: "Test migration",
				Up:          func(ctx context.Context, adapter Adapter) error { return nil },
			},
			expectValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.script.Version != "" && tt.script.Up != nil
			if valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, valid)
			}
		})
	}
}