package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/media/providers/filesystem"
	"github.com/knadh/listmonk/internal/media/providers/s3"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo"
)

const (
	queryFilePath = "queries.sql"
)

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
			"frontend/dist/:/frontend",
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

// constants contains static, constant config values required by the app.
type constants struct {
	RootURL      string   `koanf:"root"`
	LogoURL      string   `koanf:"logo_url"`
	FaviconURL   string   `koanf:"favicon_url"`
	FromEmail    string   `koanf:"from_email"`
	NotifyEmails []string `koanf:"notify_emails"`
	Privacy      struct {
		AllowBlacklist bool            `koanf:"allow_blacklist"`
		AllowExport    bool            `koanf:"allow_export"`
		AllowWipe      bool            `koanf:"allow_wipe"`
		Exportable     map[string]bool `koanf:"-"`
	} `koanf:"privacy"`

	UnsubURL     string
	LinkTrackURL string
	ViewTrackURL string
	OptinURL     string
	MessageURL   string

	MediaProvider string
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
		Concurrency:   ko.Int("app.concurrency"),
		MessageRate:   ko.Int("app.message_rate"),
		MaxSendErrors: ko.Int("app.max_send_errors"),
		FromEmail:     cs.FromEmail,
		UnsubURL:      cs.UnsubURL,
		OptinURL:      cs.OptinURL,
		LinkTrackURL:  cs.LinkTrackURL,
		ViewTrackURL:  cs.ViewTrackURL,
		MessageURL:    cs.MessageURL,
	}, newManagerDB(q), campNotifCB, lo)

}

// initImporter initializes the bulk subscriber importer.
func initImporter(q *Queries, db *sqlx.DB, app *App) *subimporter.Importer {
	return subimporter.New(q.UpsertSubscriber.Stmt,
		q.UpsertBlacklistSubscriber.Stmt,
		q.UpdateListsDate.Stmt,
		db.DB,
		func(subject string, data interface{}) error {
			app.sendNotification(app.constants.NotifyEmails, subject, notifTplImport, data)
			return nil
		})
}

// initMessengers initializes various messenger backends.
func initMessengers(m *manager.Manager) messenger.Messenger {
	var (
		mapKeys = ko.MapKeys("smtp")
		srv     = make([]messenger.Server, 0, len(mapKeys))
	)

	// Load the default SMTP messengers.
	for _, name := range mapKeys {
		if !ko.Bool(fmt.Sprintf("smtp.%s.enabled", name)) {
			lo.Printf("skipped SMTP: %s", name)
			continue
		}

		// Read the SMTP config.
		s := messenger.Server{Name: name}
		if err := ko.UnmarshalWithConf("smtp."+name, &s, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error loading SMTP: %v", err)
		}

		srv = append(srv, s)
		lo.Printf("loaded SMTP: %s (%s@%s)", s.Name, s.Username, s.Host)
	}
	if len(srv) == 0 {
		lo.Fatalf("no SMTP servers found in config")
	}

	// Initialize the default e-mail messenger.
	msgr, err := messenger.NewEmailer(srv...)
	if err != nil {
		lo.Fatalf("error loading e-mail messenger: %v", err)
	}
	if err := m.AddMessenger(msgr); err != nil {
		lo.Printf("error registering messenger %s", err)
	}

	return msgr
}

// initMediaStore initializes Upload manager with a custom backend.
func initMediaStore() media.Store {
	switch provider := ko.String("upload.provider"); provider {
	case "s3":
		var opts s3.Opts
		ko.Unmarshal("upload.s3", &opts)
		uplder, err := s3.NewS3Store(opts)
		if err != nil {
			lo.Fatalf("error initializing s3 upload provider %s", err)
		}
		return uplder

	case "filesystem":
		var opts filesystem.Opts
		ko.Unmarshal("upload.filesystem", &opts)
		opts.RootURL = ko.String("app.root")
		opts.UploadPath = filepath.Clean(opts.UploadPath)
		opts.UploadURI = filepath.Clean(opts.UploadURI)
		uplder, err := filesystem.NewDiskStore(opts)
		if err != nil {
			lo.Fatalf("error initializing filesystem upload provider %s", err)
		}
		return uplder

	default:
		lo.Fatalf("unknown provider. please select one of either filesystem or s3")
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
func initHTTPServer(app *App) {
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
	srv.Logger.Fatal(srv.Start(ko.String("app.address")))
}
