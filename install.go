package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(app *App, qMap goyesql.Queries, prompt bool) {
	fmt.Println("")
	fmt.Println("** First time installation **")
	fmt.Printf("** IMPORTANT: This will wipe existing listmonk tables and types in the DB '%s' **",
		ko.String("db.database"))
	fmt.Println("")

	if prompt {
		var ok string
		fmt.Print("Continue (y/n)?  ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			logger.Fatalf("Error reading value from terminal: %v", err)
		}
		if strings.ToLower(ok) != "y" {
			fmt.Println("Installation cancelled.")
			return
		}
	}

	// Migrate the tables.
	err := installMigrate(app.DB, app)
	if err != nil {
		logger.Fatalf("Error migrating DB schema: %v", err)
	}

	// Load the queries.
	var q Queries
	if err := scanQueriesToStruct(&q, qMap, app.DB.Unsafe()); err != nil {
		logger.Fatalf("error loading SQL queries: %v", err)
	}

	// Sample list.
	var listID int
	if err := q.CreateList.Get(&listID,
		uuid.NewV4().String(),
		"Default list",
		models.ListTypePublic,
		models.ListOptinSingle,
		pq.StringArray{"test"},
	); err != nil {
		logger.Fatalf("Error creating list: %v", err)
	}

	// Sample subscriber.
	if _, err := q.UpsertSubscriber.Exec(
		uuid.NewV4(),
		"john@example.com",
		"John Doe",
		`{"type": "known", "good": true, "city": "Bengaluru"}`,
		pq.Int64Array{int64(listID)},
	); err != nil {
		logger.Fatalf("Error creating subscriber: %v", err)
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
		logger.Fatalf("error creating default template: %v", err)
	}
	if _, err := q.SetDefaultTemplate.Exec(tplID); err != nil {
		logger.Fatalf("error setting default template: %v", err)
	}

	// Sample campaign.
	sendAt := time.Now()
	sendAt.Add(time.Minute * 43200)
	if _, err := q.CreateCampaign.Exec(uuid.NewV4(),
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
		logger.Fatalf("error creating sample campaign: %v", err)
	}

	logger.Printf("Setup complete")
	logger.Printf(`Run the program and access the dashboard at %s`, ko.String("app.address"))

}

// installMigrate executes the SQL schema and creates the necessary tables and types.
func installMigrate(db *sqlx.DB, app *App) error {
	q, err := app.FS.Read("/schema.sql")
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
	fs, err := initFileSystem(os.Args[0])
	if err != nil {
		return err
	}

	b, err := fs.Read("config.toml.sample")
	if err != nil {
		return fmt.Errorf("error reading sample config (is binary stuffed?): %v", err)
	}

	return ioutil.WriteFile("config.toml", b, 0644)
}
