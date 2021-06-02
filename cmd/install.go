package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(lastVer string, db *sqlx.DB, fs stuffbin.FileSystem, prompt bool) {
	qMap, _ := initQueries(queryFilePath, db, fs, false)

	fmt.Println("")
	fmt.Println("** first time installation **")
	fmt.Printf("** IMPORTANT: This will wipe existing listmonk tables and types in the DB '%s' **",
		ko.String("db.database"))
	fmt.Println("")

	if prompt {
		var ok string
		fmt.Print("continue (y/n)?  ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			lo.Fatalf("error reading value from terminal: %v", err)
		}
		if strings.ToLower(ok) != "y" {
			fmt.Println("install cancelled.")
			return
		}
	}

	// Migrate the tables.
	err := installSchema(lastVer, db, fs)
	if err != nil {
		lo.Fatalf("Error migrating DB schema: %v", err)
	}

	// Load the queries.
	var q Queries
	if err := goyesqlx.ScanToStruct(&q, qMap, db.Unsafe()); err != nil {
		lo.Fatalf("error loading SQL queries: %v", err)
	}

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
	); err != nil {
		lo.Fatalf("Error creating list: %v", err)
	}

	if err := q.CreateList.Get(&optinList, uuid.Must(uuid.NewV4()),
		"Opt-in list",
		models.ListTypePublic,
		models.ListOptinDouble,
		pq.StringArray{"test"},
	); err != nil {
		lo.Fatalf("Error creating list: %v", err)
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
		lo.Fatalf("Error creating subscriber: %v", err)
	}

	// Default template.
	tplBody, err := fs.Get("/static/email-templates/default.tpl")
	if err != nil {
		lo.Fatalf("error reading default e-mail template: %v", err)
	}

	var tplID int
	if err := q.CreateTemplate.Get(&tplID,
		"Default template",
		string(tplBody.ReadBytes()),
	); err != nil {
		lo.Fatalf("error creating default template: %v", err)
	}
	if _, err := q.SetDefaultTemplate.Exec(tplID); err != nil {
		lo.Fatalf("error setting default template: %v", err)
	}

	// Sample campaign.
	if _, err := q.CreateCampaign.Exec(uuid.Must(uuid.NewV4()),
		models.CampaignTypeRegular,
		"Test campaign",
		"Welcome to listmonk",
		"No Reply <noreply@yoursite.com>",
		`<h3>Hi {{ .Subscriber.FirstName }}!</h3>
			This is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.`,
		nil,
		"richtext",
		nil,
		pq.StringArray{"test-campaign"},
		emailMsgr,
		1,
		pq.Int64Array{1},
	); err != nil {
		lo.Fatalf("error creating sample campaign: %v", err)
	}

	lo.Printf("Setup complete")
	lo.Printf(`Run the program and access the dashboard at %s`, ko.MustString("app.address"))
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

func newConfigFile() error {
	if _, err := os.Stat("config.toml"); !os.IsNotExist(err) {
		return errors.New("config.toml exists. Remove it to generate a new one")
	}

	// Initialize the static file system into which all
	// required static assets (.sql, .js files etc.) are loaded.
	fs := initFS("", "")
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

	return ioutil.WriteFile("config.toml", b, 0644)
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
