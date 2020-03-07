package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/listmonk/manager"
	"github.com/knadh/listmonk/media"
	"github.com/knadh/listmonk/messenger"
	"github.com/knadh/listmonk/subimporter"
	"github.com/knadh/stuffbin"
	flag "github.com/spf13/pflag"
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
}

var (
	// Global logger.
	lo = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// Global configuration reader.
	ko = koanf.New(".")

	buildString string
)

func init() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		// Register --help handler.
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register the commandline flags.
	f.StringSlice("config", []string{"config.toml"},
		"Path to one or more config files (will be merged in order)")
	f.Bool("install", false, "Run first time installation")
	f.Bool("version", false, "Current version of the build")
	f.Bool("new-config", false, "Generate sample config file")
	f.Bool("yes", false, "Assume 'yes' to prompts, eg: during --install")

	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error loading flags: %v", err)
	}

	// Display version.
	if v, _ := f.GetBool("version"); v {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Generate new config.
	if ok, _ := f.GetBool("new-config"); ok {
		if err := newConfigFile(); err != nil {
			lo.Println(err)
			os.Exit(1)
		}
		lo.Println("generated config.toml. Edit and run --install")
		os.Exit(0)
	}

	// Load config files.
	cFiles, _ := f.GetStringSlice("config")
	for _, f := range cFiles {
		lo.Printf("reading config: %s", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				lo.Fatal("config file not found. If there isn't one yet, run --new-config to generate one.")
			}
			lo.Fatalf("error loadng config from file: %v.", err)
		}
	}

	// Load environment variables and merge into the loaded config.
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading config from env: %v", err)
	}
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading config: %v", err)
	}
}

func main() {
	// Initialize the DB and the filesystem that are required by the installer
	// and the app.
	var (
		fs = initFS()
		db = initDB()
	)
	defer db.Close()

	// Installer mode? This runs before the SQL queries are loaded and prepared
	// as the installer needs to work on an empty DB.
	if ko.Bool("install") {
		install(db, fs, !ko.Bool("yes"))
		return
	}

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
	app.manager = initCampaignManager(app)
	app.importer = initImporter(app)
	app.messenger = initMessengers(app.manager)
	app.notifTpls = initNotifTemplates("/email-templates/*.html", fs, app.constants)

	// Start the campaign workers.
	go app.manager.Run(time.Second * 5)
	app.manager.SpawnWorkers()

	// Start and run the app server.
	initHTTPServer(app)
}
