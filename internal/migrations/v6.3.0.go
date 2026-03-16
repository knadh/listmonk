package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS subject_prefix TEXT NOT NULL DEFAULT '';
		ALTER TABLE campaign_lists ADD COLUMN IF NOT EXISTS subject_prefix TEXT NOT NULL DEFAULT '';
		UPDATE campaign_lists SET subject_prefix = l.subject_prefix FROM lists l WHERE l.id = campaign_lists.list_id;
	`); err != nil {
		return err
	}

	return nil
}
