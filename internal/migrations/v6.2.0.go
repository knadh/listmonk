package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Add MJML to content_type enum if not exists.
	if _, err := db.Exec(`ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'mjml';`); err != nil {
		return err
	}

	// Add campaign_mjml to template_type enum if not exists.
	if _, err := db.Exec(`ALTER TYPE template_type ADD VALUE IF NOT EXISTS 'campaign_mjml';`); err != nil {
		return err
	}

	// Insert sample MJML template.
	tpl, err := fs.Get("/static/email-templates/sample-mjml.tpl")
	if err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT INTO templates (name, type, subject, body) VALUES($1, $2, $3, $4)`,
		"Sample MJML template", "campaign_mjml", "", tpl.ReadBytes()); err != nil {
		return err
	}

	return nil
}
