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

	UPDATE campaigns SET 
		sliding_window = (select value::boolean  from settings where key = 'app.message_sliding_window'),
		sliding_window_rate = (select value::integer from settings where key = 'app.message_sliding_window_rate'),
		sliding_window_duration = (select value#>>'{}' from settings where key = 'app.message_sliding_window_duration');

	DELETE FROM settings where key IN ('app.message_sliding_window', 'app.message_sliding_window_rate', 'app.message_sliding_window_duration');

	`); err != nil {
		return err
	}

	return nil
}
