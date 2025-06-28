package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_1_0 performs the DB migrations for SQL snippets.
func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Create SQL snippets table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sql_snippets (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			query_sql TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_by INTEGER NULL REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for SQL snippets
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sql_snippets_name ON sql_snippets(name);
		CREATE INDEX IF NOT EXISTS idx_sql_snippets_is_active ON sql_snippets(is_active);
	`); err != nil {
		return err
	}

	// Insert default SQL snippets
	if _, err := db.Exec(`
		INSERT INTO sql_snippets (name, description, query_sql) VALUES
		('Enabled Subscribers', 'Subscribers with enabled status', 'subscribers.status = ''enabled'''),
		('Recent Signups', 'Subscribers who joined in the last 30 days', 'subscribers.created_at >= NOW() - INTERVAL ''30 days''')
		ON CONFLICT (name) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}