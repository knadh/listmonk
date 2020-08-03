package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/stuffbin"
)

// App contains the "global" components that are
// passed around, especially through HTTP handlers.
type App struct {
	fs        stuffbin.FileSystem
	db        *sqlx.DB
	queries   *Queries
	constants *constants
	manager   *manager.Manager
	importer  *subimporter.Importer
	messenger messenger.Messenger
	media     media.Store
	notifTpls *template.Template
	log       *log.Logger

	// Channel for passing reload signals.
	sigChan chan os.Signal

	// Global variable that stores the state indicating that a restart is required
	// after a settings update.
	needsRestart bool
	sync.Mutex
}

var (
	lo = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	ko = koanf.New(".")

	fs      stuffbin.FileSystem
	db      *sqlx.DB
	queries *Queries

	buildString string
)

func init() {
	lo.Println(buildString)
	initFlags()

	// Display version.
	if ko.Bool("version") {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Generate new config.
	if ko.Bool("new-config") {
		if err := newConfigFile(); err != nil {
			lo.Println(err)
			os.Exit(1)
		}
		lo.Println("generated config.toml. Edit and run --install")
		os.Exit(0)
	}

	// Load config files to pick up the database settings first.
	initConfigFiles(ko.Strings("config"), ko)

	// Connect to the database, load the filesystem to read SQL queries.
	db = initDB()
	fs = initFS(ko.String("static-dir"))

	// Installer mode? This runs before the SQL queries are loaded and prepared
	// as the installer needs to work on an empty DB.
	if ko.Bool("install") {
		// Save the version of the last listed migration.
		install(migList[len(migList)-1].version, db, fs, !ko.Bool("yes"))
		os.Exit(0)
	}
	if ko.Bool("upgrade") {
		upgrade(db, fs, !ko.Bool("yes"))
		os.Exit(0)
	}

	// Before the queries are prepared, see if there are pending upgrades.
	checkUpgrade(db)

	// Load the SQL queries from the filesystem.
	_, queries := initQueries(queryFilePath, db, fs, true)

	// Load settings from DB.
	initSettings(queries)

	// Load environment variables and merge into the loaded config.
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading config from env: %v", err)
	}
}

func main() {
	// Initialize the main app controller that wraps all of the app's
	// components. This is passed around HTTP handlers.
	app := &App{
		fs:        fs,
		db:        db,
		constants: initConstants(),
		media:     initMediaStore(),
		log:       lo,
	}
	_, app.queries = initQueries(queryFilePath, db, fs, true)
	app.manager = initCampaignManager(app.queries, app.constants, app)
	app.importer = initImporter(app.queries, db, app)
	app.messenger = initMessengers(app.manager)
	app.notifTpls = initNotifTemplates("/email-templates/*.html", fs, app.constants)

	// Start the campaign workers. The campaign batches (fetch from DB, push out
	// messages) get processed at the specified interval.
	go app.manager.Run(time.Second * 5)

	// Start the app server.
	srv := initHTTPServer(app)

	// Wait for the reload signal with a callback to gracefully shut down resources.
	// The `wait` channel is passed to awaitReload to wait for the callback to finish
	// within N seconds, or do a force reload.
	app.sigChan = make(chan os.Signal)
	signal.Notify(app.sigChan, syscall.SIGHUP)

	closerWait := make(chan bool)
	<-awaitReload(app.sigChan, closerWait, func() {
		// Stop the HTTP server.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		// Close the campaign manager.
		app.manager.Close()

		// Close the DB pool.
		app.db.DB.Close()

		// Close the messenger pool.
		app.messenger.Close()

		// Signal the close.
		closerWait <- true
	})
}
