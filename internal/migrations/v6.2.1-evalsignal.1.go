package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_2_1_Evalsignal_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS send_until TIMESTAMP WITH TIME ZONE`)
	return err
}
