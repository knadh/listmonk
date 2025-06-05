package database

import (
	"context"

	"github.com/knadh/listmonk/internal/repository"
)

// Adapter defines the interface that all database adapters must implement
type Adapter interface {
	// Connection management
	Connect(ctx context.Context, config Config) error
	Close() error
	Ping(ctx context.Context) error
	
	// Transaction management
	BeginTx(ctx context.Context) (repository.Transaction, error)
	
	// Migration management
	GetCurrentVersion(ctx context.Context) (string, error)
	Migrate(ctx context.Context, targetVersion string) error
	
	// Repository access
	GetManager() repository.Manager
	
	// Database type
	Type() DatabaseType
	
	// Health check
	IsConnected() bool
}

// AdapterFactory defines the factory interface for creating database adapters
type AdapterFactory interface {
	CreateAdapter(dbType DatabaseType) (Adapter, error)
	SupportedTypes() []DatabaseType
}

// MigrationScript represents a database migration script
type MigrationScript struct {
	Version     string
	Description string
	Up          func(ctx context.Context, adapter Adapter) error
	Down        func(ctx context.Context, adapter Adapter) error
}

// MigrationManager manages database migrations
type MigrationManager interface {
	GetCurrentVersion(ctx context.Context) (string, error)
	GetAppliedMigrations(ctx context.Context) ([]string, error)
	GetPendingMigrations(ctx context.Context) ([]MigrationScript, error)
	ApplyMigration(ctx context.Context, script MigrationScript) error
	RollbackMigration(ctx context.Context, script MigrationScript) error
	MigrateToVersion(ctx context.Context, targetVersion string) error
	RegisterMigration(script MigrationScript) error
}