package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/media/providers/filesystem"
	"github.com/knadh/listmonk/internal/media/providers/s3"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/messenger/postback"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo"
	flag "github.com/spf13/pflag"
)

const (
	queryFilePath = "queries.sql"
)

// constants contains static, constant config values required by the app.
type constants struct {
	RootURL      string   `koanf:"root_url"`
	LogoURL      string   `koanf:"logo_url"`
	FaviconURL   string   `koanf:"favicon_url"`
	FromEmail    string   `koanf:"from_email"`
	NotifyEmails []string `koanf:"notify_emails"`
	Privacy      struct {
		AllowBlocklist bool            `koanf:"allow_blocklist"`
		AllowExport    bool            `koanf:"allow_export"`
		AllowWipe      bool            `koanf:"allow_wipe"`
		Exportable     map[string]bool `koanf:"-"`
	} `koanf:"privacy"`
	AdminUsername []byte `koanf:"admin_username"`
	AdminPassword []byte `koanf:"admin_password"`

	UnsubURL      string
	LinkTrackURL  string
	ViewTrackURL  string
	OptinURL      string
	MessageURL    string
	MediaProvider string
}

func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		// Register --help handler.
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register the commandline flags.
	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("install", false, "run first time installation")
	f.Bool("upgrade", false, "upgrade database to the current version")
	f.Bool("version", false, "current version of the build")
	f.Bool("new-config", false, "generate sample config file")
	f.String("static-dir", "", "(optional) path to directory with static files")
	f.Bool("yes", false, "assume 'yes' to prompts, eg: during --install")
	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error loading flags: %v", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading config: %v", err)
	}
}

// initConfigFiles loads the given config files into the koanf instance.
func initConfigFiles(files []string, ko *koanf.Koanf) {
	for _, f := range files {
		lo.Printf("reading config: %s", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				lo.Fatal("config file not found. If there isn't one yet, run --new-config to generate one.")
			}
			lo.Fatalf("error loadng config from file: %v.", err)
		}
	}
}

// initFileSystem initializes the stuffbin FileSystem to provide
// access to bunded static assets to the app.
func initFS(staticDir string) stuffbin.FileSystem {
	// Get the executable's path.
	path, err := os.Executable()
	if err != nil {
		lo.Fatalf("error getting executable path: %v", err)
	}

	// Load the static files stuffed in the binary.
	fs, err := stuffbin.UnStuff(path)
	if err != nil {
		// Running in local mode. Load local assets into
		// the in-memory stuffbin.FileSystem.
		lo.Printf("unable to initialize embedded filesystem: %v", err)
		lo.Printf("using local filesystem for static assets")
		files := []string{
			"config.toml.sample",
			"queries.sql",
			"schema.sql",
			"static/email-templates",

			// Alias /static/public to /public for the HTTP fileserver.
			"static/public:/public",

			// The frontend app's static assets are aliased to /frontend
			// so that they are accessible at /frontend/js/* etc.
			// Alias all files inside dist/ and dist/frontend to frontend/*.
			"frontend/dist/favicon.png:/frontend/favicon.png",
			"frontend/dist/frontend:/frontend",
		}

		fs, err = stuffbin.NewLocalFS("/", files...)
		if err != nil {
			lo.Fatalf("failed to initialize local file for assets: %v", err)
		}
	}

	// Optional static directory to override files.
	if staticDir != "" {
		lo.Printf("loading static files from: %v", staticDir)
		fStatic, err := stuffbin.NewLocalFS("/", []string{
			filepath.Join(staticDir, "/email-templates") + ":/static/email-templates",

			// Alias /static/public to /public for the HTTP fileserver.
			filepath.Join(staticDir, "/public") + ":/public",
		}...)
		if err != nil {
			lo.Fatalf("failed reading static directory: %s: %v", staticDir, err)
		}

		if err := fs.Merge(fStatic); err != nil {
			lo.Fatalf("error merging static directory: %s: %v", staticDir, err)
		}
	}
	return fs
}

// initDB initializes the main DB connection pool and parse and loads the app's
// SQL queries into a prepared query map.
func initDB() *sqlx.DB {
	var dbCfg dbConf
	if err := ko.Unmarshal("db", &dbCfg); err != nil {
		lo.Fatalf("error loading db config: %v", err)
	}

	lo.Printf("connecting to db: %s:%d/%s", dbCfg.Host, dbCfg.Port, dbCfg.DBName)
	db, err := connectDB(dbCfg)
	if err != nil {
		lo.Fatalf("error connecting to DB: %v", err)
	}
	return db
}

// initQueries loads named SQL queries from the queries file and optionally
// prepares them.
func initQueries(sqlFile string, db *sqlx.DB, fs stuffbin.FileSystem, prepareQueries bool) (goyesql.Queries, *Queries) {
	// Load SQL queries.
	qB, err := fs.Read(sqlFile)
	if err != nil {
		lo.Fatalf("error reading SQL file %s: %v", sqlFile, err)
	}
	qMap, err := goyesql.ParseBytes(qB)
	if err != nil {
		lo.Fatalf("error parsing SQL queries: %v", err)
	}

	if !prepareQueries {
		return qMap, nil
	}

	// Prepare queries.
	var q Queries
	if err := goyesqlx.ScanToStruct(&q, qMap, db.Unsafe()); err != nil {
		lo.Fatalf("error preparing SQL queries: %v", err)
	}
	return qMap, &q
}

// initSettings loads settings from the DB.
func initSettings(q *Queries) {
	var s types.JSONText
	if err := q.GetSettings.Get(&s); err != nil {
		lo.Fatalf("error reading settings from DB: %s", pqErrMsg(err))
	}

	// Setting keys are dot separated, eg: app.favicon_url. Unflatten them into
	// nested maps {app: {favicon_url}}.
	var out map[string]interface{}
	if err := json.Unmarshal(s, &out); err != nil {
		lo.Fatalf("error unmarshalling settings from DB: %v", err)
	}
	if err := ko.Load(confmap.Provider(out, "."), nil); err != nil {
		lo.Fatalf("error parsing settings from DB: %v", err)
	}
}

func initConstants() *constants {
	// Read constants.
	var c constants
	if err := ko.Unmarshal("app", &c); err != nil {
		lo.Fatalf("error loading app config: %v", err)
	}
	if err := ko.Unmarshal("privacy", &c.Privacy); err != nil {
		lo.Fatalf("error loading app config: %v", err)
	}

	c.RootURL = strings.TrimRight(c.RootURL, "/")
	c.Privacy.Exportable = maps.StringSliceToLookupMap(ko.Strings("privacy.exportable"))
	c.MediaProvider = ko.String("upload.provider")

	// Static URLS.
	// url.com/subscription/{campaign_uuid}/{subscriber_uuid}
	c.UnsubURL = fmt.Sprintf("%s/subscription/%%s/%%s", c.RootURL)

	// url.com/subscription/optin/{subscriber_uuid}
	c.OptinURL = fmt.Sprintf("%s/subscription/optin/%%s?%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
	c.LinkTrackURL = fmt.Sprintf("%s/link/%%s/%%s/%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}
	c.MessageURL = fmt.Sprintf("%s/campaign/%%s/%%s", c.RootURL)

	// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
	c.ViewTrackURL = fmt.Sprintf("%s/campaign/%%s/%%s/px.png", c.RootURL)
	return &c
}

// initCampaignManager initializes the campaign manager.
func initCampaignManager(q *Queries, cs *constants, app *App) *manager.Manager {
	campNotifCB := func(subject string, data interface{}) error {
		return app.sendNotification(cs.NotifyEmails, subject, notifTplCampaign, data)
	}

	if ko.Int("app.concurrency") < 1 {
		lo.Fatal("app.concurrency should be at least 1")
	}
	if ko.Int("app.message_rate") < 1 {
		lo.Fatal("app.message_rate should be at least 1")
	}

	return manager.New(manager.Config{
		BatchSize:     ko.Int("app.batch_size"),
		Concurrency:   ko.Int("app.concurrency"),
		MessageRate:   ko.Int("app.message_rate"),
		MaxSendErrors: ko.Int("app.max_send_errors"),
		FromEmail:     cs.FromEmail,
		UnsubURL:      cs.UnsubURL,
		OptinURL:      cs.OptinURL,
		LinkTrackURL:  cs.LinkTrackURL,
		ViewTrackURL:  cs.ViewTrackURL,
		MessageURL:    cs.MessageURL,
		UnsubHeader:   ko.Bool("privacy.unsubscribe_header"),
	}, newManagerDB(q), campNotifCB, lo)

}

// initImporter initializes the bulk subscriber importer.
func initImporter(q *Queries, db *sqlx.DB, app *App) *subimporter.Importer {
	return subimporter.New(
		subimporter.Options{
			UpsertStmt:         q.UpsertSubscriber.Stmt,
			BlocklistStmt:      q.UpsertBlocklistSubscriber.Stmt,
			UpdateListDateStmt: q.UpdateListsDate.Stmt,
			NotifCB: func(subject string, data interface{}) error {
				app.sendNotification(app.constants.NotifyEmails, subject, notifTplImport, data)
				return nil
			},
		}, db.DB)
}

// initSMTPMessenger initializes the SMTP messenger.
func initSMTPMessenger(m *manager.Manager) messenger.Messenger {
	var (
		mapKeys = ko.MapKeys("smtp")
		servers = make([]email.Server, 0, len(mapKeys))
	)

	items := ko.Slices("smtp")
	if len(items) == 0 {
		lo.Fatalf("no SMTP servers found in config")
	}

	// Load the config for multipme SMTP servers.
	for _, item := range items {
		if !item.Bool("enabled") {
			continue
		}

		// Read the SMTP config.
		var s email.Server
		if err := item.UnmarshalWithConf("", &s, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading SMTP config: %v", err)
		}

		servers = append(servers, s)
		lo.Printf("loaded email (SMTP) messenger: %s@%s",
			item.String("username"), item.String("host"))
	}
	if len(servers) == 0 {
		lo.Fatalf("no SMTP servers enabled in settings")
	}

	// Initialize the e-mail messenger with multiple SMTP servers.
	msgr, err := email.New(servers...)
	if err != nil {
		lo.Fatalf("error loading e-mail messenger: %v", err)
	}

	return msgr
}

// initPostbackMessengers initializes and returns all the enabled
// HTTP postback messenger backends.
func initPostbackMessengers(m *manager.Manager) []messenger.Messenger {
	items := ko.Slices("messengers")
	if len(items) == 0 {
		return nil
	}

	var out []messenger.Messenger
	for _, item := range items {
		if !item.Bool("enabled") {
			continue
		}

		// Read the Postback server config.
		var (
			name = item.String("name")
			o    postback.Options
		)
		if err := item.UnmarshalWithConf("", &o, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading Postback config: %v", err)
		}

		// Initialize the Messenger.
		p, err := postback.New(o)
		if err != nil {
			lo.Fatalf("error initializing Postback messenger %s: %v", name, err)
		}
		out = append(out, p)

		lo.Printf("loaded Postback messenger: %s", name)
	}

	return out
}

// initMediaStore initializes Upload manager with a custom backend.
func initMediaStore() media.Store {
	switch provider := ko.String("upload.provider"); provider {
	case "s3":
		var o s3.Opts
		ko.Unmarshal("upload.s3", &o)
		up, err := s3.NewS3Store(o)
		if err != nil {
			lo.Fatalf("error initializing s3 upload provider %s", err)
		}
		lo.Println("media upload provider: s3")
		return up

	case "filesystem":
		var o filesystem.Opts

		ko.Unmarshal("upload.filesystem", &o)
		o.RootURL = ko.String("app.root_url")
		o.UploadPath = filepath.Clean(o.UploadPath)
		o.UploadURI = filepath.Clean(o.UploadURI)
		up, err := filesystem.NewDiskStore(o)
		if err != nil {
			lo.Fatalf("error initializing filesystem upload provider %s", err)
		}
		lo.Println("media upload provider: filesystem")
		return up

	default:
		lo.Fatalf("unknown provider. select filesystem or s3")
	}
	return nil
}

// initNotifTemplates compiles and returns e-mail notification templates that are
// used for sending ad-hoc notifications to admins and subscribers.
func initNotifTemplates(path string, fs stuffbin.FileSystem, cs *constants) *template.Template {
	// Register utility functions that the e-mail templates can use.
	funcs := template.FuncMap{
		"RootURL": func() string {
			return cs.RootURL
		},
		"LogoURL": func() string {
			return cs.LogoURL
		}}

	tpl, err := stuffbin.ParseTemplatesGlob(funcs, fs, "/static/email-templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing e-mail notif templates: %v", err)
	}
	return tpl
}

// initHTTPServer sets up and runs the app's main HTTP server and blocks forever.
func initHTTPServer(app *App) *echo.Echo {
	// Initialize the HTTP server.
	var srv = echo.New()
	srv.HideBanner = true

	// Register app (*App) to be injected into all HTTP handlers.
	srv.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("app", app)
			return next(c)
		}
	})

	// Parse and load user facing templates.
	tpl, err := stuffbin.ParseTemplatesGlob(nil, app.fs, "/public/templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &tplRenderer{
		templates:  tpl,
		RootURL:    app.constants.RootURL,
		LogoURL:    app.constants.LogoURL,
		FaviconURL: app.constants.FaviconURL}

	// Initialize the static file server.
	fSrv := app.fs.FileServer()
	srv.GET("/public/*", echo.WrapHandler(fSrv))
	srv.GET("/frontend/*", echo.WrapHandler(fSrv))
	if ko.String("upload.provider") == "filesystem" {
		srv.Static(ko.String("upload.filesystem.upload_uri"),
			ko.String("upload.filesystem.upload_path"))
	}

	// Register all HTTP handlers.
	registerHTTPHandlers(srv)

	// Start the server.
	go func() {
		if err := srv.Start(ko.String("app.address")); err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				lo.Println("HTTP server shut down")
			} else {
				lo.Fatalf("error starting HTTP server: %v", err)
			}
		}
	}()

	return srv
}

func awaitReload(sigChan chan os.Signal, closerWait chan bool, closer func()) chan bool {
	// The blocking signal handler that main() waits on.
	out := make(chan bool)

	// Respawn a new process and exit the running one.
	respawn := func() {
		if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
			lo.Fatalf("error spawning process: %v", err)
		}
		os.Exit(0)
	}

	// Listen for reload signal.
	go func() {
		for range sigChan {
			lo.Println("reloading on signal ...")

			go closer()
			select {
			case <-closerWait:
				// Wait for the closer to finish.
				respawn()
			case <-time.After(time.Second * 3):
				// Or timeout and force close.
				respawn()
			}
		}
	}()

	return out
}
