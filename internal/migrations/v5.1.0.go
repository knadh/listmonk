package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
	ALTER TABLE campaigns ADD IF NOT EXISTS sliding_window bool NOT NULL DEFAULT false;
	ALTER TABLE campaigns ADD IF NOT EXISTS sliding_window_rate int NOT NULL DEFAULT 1;
	ALTER TABLE campaigns ADD IF NOT EXISTS sliding_window_duration varchar(4) NOT NULL DEFAULT '1h';

	`); err != nil {
		return err
	}

	return nil
}
