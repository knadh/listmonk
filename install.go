package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(db *sqlx.DB, fs stuffbin.FileSystem, prompt bool) {
	qMap, _ := initQueries(queryFilePath, db, fs, false)

	fmt.Println("")
	fmt.Println("** First time installation **")
	fmt.Printf("** IMPORTANT: This will wipe existing listmonk tables and types in the DB '%s' **",
		ko.String("db.database"))
	fmt.Println("")

	if prompt {
		var ok string
		fmt.Print("Continue (y/n)?  ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			lo.Fatalf("Error reading value from terminal: %v", err)
		}
		if strings.ToLower(ok) != "y" {
			fmt.Println("Installation cancelled.")
			return
		}
	}

	// Migrate the tables.
	err := installMigrate(db, fs)
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
	); err != nil {
		lo.Fatalf("Error creating subscriber: %v", err)
	}
	if _, err := q.UpsertSubscriber.Exec(
		uuid.Must(uuid.NewV4()),
		"anon@example.com",
		"Anon Doe",
		`{"type": "unknown", "good": true, "city": "Bengaluru"}`,
		pq.Int64Array{int64(optinList)},
	); err != nil {
		lo.Fatalf("Error creating subscriber: %v", err)
	}

	// Default template.
	tplBody, err := ioutil.ReadFile("email-templates/default.tpl")
	if err != nil {
		tplBody = []byte(tplTag)
	}

	var tplID int
	if err := q.CreateTemplate.Get(&tplID,
		"Default template",
		string(tplBody),
	); err != nil {
		lo.Fatalf("error creating default template: %v", err)
	}
	if _, err := q.SetDefaultTemplate.Exec(tplID); err != nil {
		lo.Fatalf("error setting default template: %v", err)
	}

	// Sample campaign.
	sendAt := time.Now()
	sendAt.Add(time.Minute * 43200)
	if _, err := q.CreateCampaign.Exec(uuid.Must(uuid.NewV4()),
		models.CampaignTypeRegular,
		"Test campaign",
		"Welcome to listmonk",
		"No Reply <noreply@yoursite.com>",
		`<h3>Hi {{ .Subscriber.FirstName }}!</h3>
			This is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.`,
		"richtext",
		sendAt,
		pq.StringArray{"test-campaign"},
		"email",
		1,
		pq.Int64Array{1},
	); err != nil {
		lo.Fatalf("error creating sample campaign: %v", err)
	}

	lo.Printf("Setup complete")
	lo.Printf(`Run the program and access the dashboard at %s`, ko.MustString("app.address"))

}

// installMigrate executes the SQL schema and creates the necessary tables and types.
func installMigrate(db *sqlx.DB, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}

	_, err = db.Query(string(q))
	if err != nil {
		return err
	}

	return nil
}

func newConfigFile() error {
	if _, err := os.Stat("config.toml"); !os.IsNotExist(err) {
		return errors.New("config.toml exists. Remove it to generate a new one")
	}

	// Initialize the static file system into which all
	// required static assets (.sql, .js files etc.) are loaded.
	fs := initFS()
	b, err := fs.Read("config.toml.sample")
	if err != nil {
		return fmt.Errorf("error reading sample config (is binary stuffed?): %v", err)
	}

	return ioutil.WriteFile("config.toml", b, 0644)
}
