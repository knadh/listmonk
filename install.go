package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(app *App, qMap goyesql.Queries) {
	fmt.Println("")
	fmt.Println("** First time installation **")
	fmt.Printf("** IMPORTANT: This will wipe existing listmonk tables and types in the DB '%s' **",
		ko.String("db.database"))
	fmt.Println("")

	var ok string
	fmt.Print("Continue (y/n)?  ")
	if _, err := fmt.Scanf("%s", &ok); err != nil {
		logger.Fatalf("Error reading value from terminal: %v", err)
	}
	if strings.ToLower(ok) != "y" {
		fmt.Println("Installation cancelled.")
		return
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
		pq.StringArray{"test"},
	); err != nil {
		logger.Fatalf("Error creating superadmin user: %v", err)
	}

	// Sample subscriber.
	if _, err := q.UpsertSubscriber.Exec(
		uuid.NewV4(),
		"test@test.com",
		"Test Subscriber",
		`{"type": "known", "good": true}`,
		pq.Int64Array{int64(listID)},
	); err != nil {
		logger.Fatalf("Error creating subscriber: %v", err)
	}

	// Default template.
	tplBody, err := ioutil.ReadFile("templates/default.tpl")
	if err != nil {
		tplBody = []byte(tplTag)
	}

	var tplID int
	if err := q.CreateTemplate.Get(&tplID,
		"Default template",
		string(tplBody),
	); err != nil {
		logger.Fatalf("Error creating default template: %v", err)
	}
	if _, err := q.SetDefaultTemplate.Exec(tplID); err != nil {
		logger.Fatalf("Error setting default template: %v", err)
	}

	logger.Printf("Setup complete")
	logger.Printf(`Run the program view it at %s`, ko.String("app.address"))

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
