package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"syscall"

	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/knadh/listmonk/models"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

// install runs the first time setup of creating and
// migrating the database and creating the super user.
func install(app *App, qMap goyesql.Queries) {
	var (
		email, pw, pw2 []byte
		err            error

		// Pseudo e-mail validation using Regexp, well ...
		emRegex, _ = regexp.Compile("(.+?)@(.+?)")
	)

	fmt.Println("** First time installation. **")
	fmt.Println("** IMPORTANT: This will wipe existing listmonk tables and types. **")
	fmt.Println("\n")

	for len(email) == 0 {
		fmt.Print("Enter the superadmin login e-mail: ")
		if _, err = fmt.Scanf("%s", &email); err != nil {
			logger.Fatalf("Error reading e-mail from the terminal: %v", err)
		}

		if !emRegex.Match(email) {
			logger.Println("Please enter a valid e-mail")
			email = []byte{}
		}
	}

	for len(pw) < 8 {
		fmt.Print("Enter the superadmin password (min 8 chars): ")
		if pw, err = terminal.ReadPassword(int(syscall.Stdin)); err != nil {
			logger.Fatalf("Error reading password from the terminal: %v", err)
		}

		fmt.Println("")
		if len(pw) < 8 {
			logger.Println("Password should be min 8 characters")
			pw = []byte{}
		}
	}

	for len(pw2) < 8 {
		fmt.Print("Repeat the superadmin password: ")
		if pw2, err = terminal.ReadPassword(int(syscall.Stdin)); err != nil {
			logger.Fatalf("Error reading password from the terminal: %v", err)
		}

		fmt.Println("")
		if len(pw2) < 8 {
			logger.Println("Password should be min 8 characters")
			pw2 = []byte{}
		}
	}

	// Validate.
	if !bytes.Equal(pw, pw2) {
		logger.Fatalf("Passwords don't match")
	}

	// Hash the password.
	hash, err := bcrypt.GenerateFromPassword(pw, bcrypt.DefaultCost)
	if err != nil {
		logger.Fatalf("Error hashing password: %v", err)
	}

	// Migrate the tables.
	err = installMigrate(app.DB)
	if err != nil {
		logger.Fatalf("Error migrating DB schema: %v", err)
	}

	// Load the queries.
	var q Queries
	if err := scanQueriesToStruct(&q, qMap, app.DB.Unsafe()); err != nil {
		logger.Fatalf("error loading SQL queries: %v", err)
	}

	// Create the superadmin user.
	if _, err := q.CreateUser.Exec(
		string(email),
		models.UserTypeSuperadmin, // name
		string(hash),
		models.UserTypeSuperadmin,
		models.UserStatusEnabled,
	); err != nil {
		logger.Fatalf("Error creating superadmin user: %v", err)
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
	name := bytes.Split(email, []byte("@"))
	if _, err := q.UpsertSubscriber.Exec(
		uuid.NewV4(),
		email,
		bytes.Title(name[0]),
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
	logger.Printf(`Run the program and login with the username "superadmin" and your password at %s`,
		viper.GetString("server.address"))

}

// installMigrate executes the SQL schema and creates the necessary tables and types.
func installMigrate(db *sqlx.DB) error {
	q, err := ioutil.ReadFile("schema.sql")
	if err != nil {
		return err
	}

	_, err = db.Query(string(q))
	if err != nil {
		return err
	}

	return nil
}
