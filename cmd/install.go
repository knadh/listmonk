package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(lastVer string, db *sqlx.DB, fs stuffbin.FileSystem, prompt, idempotent bool) {
	qMap := readQueries(queryFilePath, db, fs)

	fmt.Println("")
	if !idempotent {
		fmt.Println("** first time installation **")
		fmt.Printf("** IMPORTANT: This will wipe existing listmonk tables and types in the DB '%s' **",
			ko.String("db.database"))
	} else {
		fmt.Println("** first time (idempotent) installation **")
	}
	fmt.Println("")

	if prompt {
		var ok string
		fmt.Print("continue (y/N)?  ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			lo.Fatalf("error reading value from terminal: %v", err)
		}
		if strings.ToLower(ok) != "y" {
			fmt.Println("install cancelled.")
			return
		}
	}

	// If idempotence is on, check if the DB is already setup.
	if idempotent {
		if _, err := db.Exec("SELECT count(*) FROM settings"); err != nil {
			// If "settings" doesn't exist, assume it's a fresh install.
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code != "42P01" {
				lo.Fatalf("error checking existing DB schema: %v", err)
			}
		} else {
			lo.Println("skipping install as database appears to be already setup")
			os.Exit(0)
		}
	}

	// Migrate the tables.
	if err := installSchema(lastVer, db, fs); err != nil {
		lo.Fatalf("error migrating DB schema: %v", err)
	}

	// Load the queries.
	q := prepareQueries(qMap, db, ko)

	// Sample list.
	var (
		defList   int
		optinList int
	)
	if err := q.CreateList.Get(&defList,
		uuid.Must(uuid.NewV4()),
		"Default list",
		models.ListTypePrivate,
		models.ListOptinSingle,
		pq.StringArray{"test"},
		"",
	); err != nil {
		lo.Fatalf("error creating list: %v", err)
	}

	if err := q.CreateList.Get(&optinList, uuid.Must(uuid.NewV4()),
		"Opt-in list",
		models.ListTypePublic,
		models.ListOptinDouble,
		pq.StringArray{"test"},
		"",
	); err != nil {
		lo.Fatalf("error creating list: %v", err)
	}

	// Sample subscriber.
	if _, err := q.UpsertSubscriber.Exec(
		uuid.Must(uuid.NewV4()),
		"john@example.com",
		"John Doe",
		`{"type": "known", "good": true, "city": "Bengaluru"}`,
		pq.Int64Array{int64(defList)},
		models.SubscriptionStatusUnconfirmed,
		true); err != nil {
		lo.Fatalf("Error creating subscriber: %v", err)
	}
	if _, err := q.UpsertSubscriber.Exec(
		uuid.Must(uuid.NewV4()),
		"anon@example.com",
		"Anon Doe",
		`{"type": "unknown", "good": true, "city": "Bengaluru"}`,
		pq.Int64Array{int64(optinList)},
		models.SubscriptionStatusUnconfirmed,
		true); err != nil {
		lo.Fatalf("error creating subscriber: %v", err)
	}

	// Default campaign template.
	campTpl, err := fs.Get("/static/email-templates/default.tpl")
	if err != nil {
		lo.Fatalf("error reading default e-mail template: %v", err)
	}

	var campTplID int
	if err := q.CreateTemplate.Get(&campTplID, "Default campaign template", models.TemplateTypeCampaign, "", campTpl.ReadBytes()); err != nil {
		lo.Fatalf("error creating default campaign template: %v", err)
	}
	if _, err := q.SetDefaultTemplate.Exec(campTplID); err != nil {
		lo.Fatalf("error setting default template: %v", err)
	}

	// Default campaign archive template.
	archiveTpl, err := fs.Get("/static/email-templates/default-archive.tpl")
	if err != nil {
		lo.Fatalf("error reading default archive template: %v", err)
	}

	var archiveTplID int
	if err := q.CreateTemplate.Get(&archiveTplID, "Default archive template", models.TemplateTypeCampaign, "", archiveTpl.ReadBytes()); err != nil {
		lo.Fatalf("error creating default campaign template: %v", err)
	}

	// Sample campaign.
	if _, err := q.CreateCampaign.Exec(uuid.Must(uuid.NewV4()),
		models.CampaignTypeRegular,
		"Test campaign",
		"Welcome to listmonk",
		"No Reply <noreply@yoursite.com>",
		`<h3>Hi {{ .Subscriber.FirstName }}!</h3>
		<p>This is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.</p>
		<p>Here is a <a href="https://listmonk.app@TrackLink">tracked link</a>.</p>
		<p>Use the link icon in the editor toolbar or when writing raw HTML or Markdown,
			simply suffix @TrackLink to the end of a URL to turn it into a tracking link. Example:</p>
		<pre>&lt;a href=&quot;https:/&zwnj;/listmonk.app&#064;TrackLink&quot;&gt;&lt;/a&gt;</pre>
		<p>For help, refer to the <a href="https://listmonk.app/docs">documentation</a>.</p>
		`,
		nil,
		"richtext",
		nil,
		json.RawMessage("[]"),
		pq.StringArray{"test-campaign"},
		emailMsgr,
		campTplID,
		pq.Int64Array{1},
		false,
		archiveTplID,
		`{"name": "Subscriber"}`,
	); err != nil {
		lo.Fatalf("error creating sample campaign: %v", err)
	}

	// Sample tx template.
	txTpl, err := fs.Get("/static/email-templates/sample-tx.tpl")
	if err != nil {
		lo.Fatalf("error reading default e-mail template: %v", err)
	}

	if _, err := q.CreateTemplate.Exec("Sample transactional template", models.TemplateTypeTx, "Welcome {{ .Subscriber.Name }}", txTpl.ReadBytes()); err != nil {
		lo.Fatalf("error creating sample transactional template: %v", err)
	}

	lo.Printf("setup complete")
	lo.Printf(`run the program and access the dashboard at %s`, ko.MustString("app.address"))
}

// installSchema executes the SQL schema and creates the necessary tables and types.
func installSchema(curVer string, db *sqlx.DB, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}

	if _, err := db.Exec(string(q)); err != nil {
		return err
	}

	// Insert the current migration version.
	return recordMigrationVersion(curVer, db)
}

// recordMigrationVersion inserts the given version (of DB migration) into the
// `migrations` array in the settings table.
func recordMigrationVersion(ver string, db *sqlx.DB) error {
	_, err := db.Exec(fmt.Sprintf(`INSERT INTO settings (key, value)
	VALUES('migrations', '["%s"]'::JSONB)
	ON CONFLICT (key) DO UPDATE SET value = settings.value || EXCLUDED.value`, ver))
	return err
}

func newConfigFile(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("%s exists. Remove it to generate a new one.", path)
	}

	// Initialize the static file system into which all
	// required static assets (.sql, .js files etc.) are loaded.
	fs := initFS(appDir, "", "", "")
	b, err := fs.Read("config.toml.sample")
	if err != nil {
		return fmt.Errorf("error reading sample config (is binary stuffed?): %v", err)
	}

	// Generate a random admin password.
	pwd, err := generateRandomString(16)
	if err == nil {
		b = regexp.MustCompile(`admin_password\s+?=\s+?(.*)`).
			ReplaceAll(b, []byte(fmt.Sprintf(`admin_password = "%s"`, pwd)))
	}

	return ioutil.WriteFile(path, b, 0644)
}

// checkSchema checks if the DB schema is installed.
func checkSchema(db *sqlx.DB) (bool, error) {
	if _, err := db.Exec(`SELECT id FROM templates LIMIT 1`); err != nil {
		if isTableNotExistErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
