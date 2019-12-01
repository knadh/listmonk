package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/listmonk/manager"
	"github.com/knadh/listmonk/media"
	"github.com/knadh/listmonk/media/providers/filesystem"
	"github.com/knadh/listmonk/media/providers/s3"
	"github.com/knadh/listmonk/messenger"
	"github.com/knadh/listmonk/subimporter"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo"
	flag "github.com/spf13/pflag"
)

type constants struct {
	RootURL        string `koanf:"root"`
	LogoURL        string `koanf:"logo_url"`
	FaviconURL     string `koanf:"favicon_url"`
	UnsubscribeURL string
	LinkTrackURL   string
	ViewTrackURL   string
	OptinURL       string
	FromEmail      string         `koanf:"from_email"`
	NotifyEmails   []string       `koanf:"notify_emails"`
	Privacy        privacyOptions `koanf:"privacy"`
}

type privacyOptions struct {
	AllowBlacklist bool            `koanf:"allow_blacklist"`
	AllowExport    bool            `koanf:"allow_export"`
	AllowWipe      bool            `koanf:"allow_wipe"`
	Exportable     map[string]bool `koanf:"-"`
}

// App contains the "global" components that are
// passed around, especially through HTTP handlers.
type App struct {
	Constants *constants
	DB        *sqlx.DB
	Queries   *Queries
	Importer  *subimporter.Importer
	Manager   *manager.Manager
	FS        stuffbin.FileSystem
	Logger    *log.Logger
	NotifTpls *template.Template
	Messenger messenger.Messenger
	Media     media.Store
}

var (
	// Global logger.
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// Global configuration reader.
	ko = koanf.New(".")

	buildString string
)

func init() {
	// Register --help handler.
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Setup the default configuration.
	f.StringSlice("config", []string{"config.toml"},
		"Path to one or more config files (will be merged in order)")
	f.Bool("install", false, "Run first time installation")
	f.Bool("version", false, "Current version of the build")
	f.Bool("new-config", false, "Generate sample config file")
	f.Bool("yes", false, "Assume 'yes' to prompts, eg: during --install")

	// Process flags.
	if err := f.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("error loading flags: %v", err)
	}

	// Display version.
	if v, _ := f.GetBool("version"); v {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Generate new config.
	if ok, _ := f.GetBool("new-config"); ok {
		if err := newConfigFile(); err != nil {
			logger.Println(err)
			os.Exit(1)
		}
		logger.Println("generated config.toml. Edit and run --install")
		os.Exit(0)
	}

	// Load config files.
	cFiles, _ := f.GetStringSlice("config")
	for _, f := range cFiles {
		logger.Printf("reading config: %s", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				logger.Fatal("config file not found. If there isn't one yet, run --new-config to generate one.")
			}
			logger.Fatalf("error loadng config from file: %v.", err)
		}
	}
	// Load environment variables and merge into the loaded config.
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		logger.Fatalf("error loading config from env: %v", err)
	}
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		logger.Fatalf("error loading config: %v", err)
	}
}

// initFileSystem initializes the stuffbin FileSystem to provide
// access to bunded static assets to the app.
func initFileSystem(binPath string) (stuffbin.FileSystem, error) {
	fs, err := stuffbin.UnStuff(binPath)
	if err == nil {
		return fs, nil
	}

	// Running in local mode. Load the required static assets into
	// the in-memory stuffbin.FileSystem.
	logger.Printf("unable to initialize embedded filesystem: %v", err)
	logger.Printf("using local filesystem for static assets")
	files := []string{
		"config.toml.sample",
		"queries.sql",
		"schema.sql",
		"email-templates",
		"public",

		// The frontend app's static assets are aliased to /frontend
		// so that they are accessible at localhost:port/frontend/static/ ...
		"frontend/build:/frontend",
	}

	fs, err = stuffbin.NewLocalFS("/", files...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local file for assets: %v", err)
	}

	return fs, nil
}

// initMessengers initializes various messaging backends.
func initMessengers(r *manager.Manager) messenger.Messenger {
	// Load SMTP configurations for the default e-mail Messenger.
	var (
		mapKeys = ko.MapKeys("smtp")
		srv     = make([]messenger.Server, 0, len(mapKeys))
	)

	for _, name := range mapKeys {
		if !ko.Bool(fmt.Sprintf("smtp.%s.enabled", name)) {
			logger.Printf("skipped SMTP: %s", name)
			continue
		}

		var s messenger.Server
		if err := ko.Unmarshal("smtp."+name, &s); err != nil {
			logger.Fatalf("error loading SMTP: %v", err)
		}
		s.Name = name
		s.SendTimeout *= time.Millisecond
		srv = append(srv, s)

		logger.Printf("loaded SMTP: %s (%s@%s)", s.Name, s.Username, s.Host)
	}

	msgr, err := messenger.NewEmailer(srv...)
	if err != nil {
		logger.Fatalf("error loading e-mail messenger: %v", err)
	}
	if err := r.AddMessenger(msgr); err != nil {
		logger.Printf("error registering messenger %s", err)
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
			logger.Fatalf("error initializing s3 upload provider %s", err)
		}
		return uplder
	case "filesystem":
		var opts filesystem.Opts
		ko.Unmarshal("upload.filesystem", &opts)
		opts.UploadPath = filepath.Clean(opts.UploadPath)
		opts.UploadURI = filepath.Clean(opts.UploadURI)
		uplder, err := filesystem.NewDiskStore(opts)
		if err != nil {
			logger.Fatalf("error initializing filesystem upload provider %s", err)
		}
		return uplder
	default:
		logger.Fatalf("unknown provider. please select one of either filesystem or s3")
	}
	return nil
}

func main() {
	// Connect to the DB.
	var dbCfg dbConf
	if err := ko.Unmarshal("db", &dbCfg); err != nil {
		log.Fatalf("error loading db config: %v", err)
	}
	db, err := connectDB(dbCfg)
	if err != nil {
		logger.Fatalf("error connecting to DB: %v", err)
	}
	defer db.Close()

	var c constants
	if err := ko.Unmarshal("app", &c); err != nil {
		log.Fatalf("error loading app config: %v", err)
	}
	if err := ko.Unmarshal("privacy", &c.Privacy); err != nil {
		log.Fatalf("error loading app config: %v", err)
	}
	c.RootURL = strings.TrimRight(c.RootURL, "/")
	c.Privacy.Exportable = maps.StringSliceToLookupMap(ko.Strings("privacy.exportable"))

	// Initialize the static file system into which all
	// required static assets (.sql, .js files etc.) are loaded.
	fs, err := initFileSystem(os.Args[0])
	if err != nil {
		logger.Fatal(err)
	}

	// Initialize the app context that's passed around.
	app := &App{
		Constants: &c,
		DB:        db,
		Logger:    logger,
		FS:        fs,
	}

	// Load SQL queries.
	qB, err := fs.Read("/queries.sql")
	if err != nil {
		logger.Fatalf("error reading queries.sql: %v", err)
	}
	qMap, err := goyesql.ParseBytes(qB)
	if err != nil {
		logger.Fatalf("error parsing SQL queries: %v", err)
	}

	// Run the first time installation.
	if ko.Bool("install") {
		install(app, qMap, !ko.Bool("yes"))
		return
	}

	// Map queries to the query container.
	q := &Queries{}
	if err := scanQueriesToStruct(q, qMap, db.Unsafe()); err != nil {
		logger.Fatalf("no SQL queries loaded: %v", err)
	}
	app.Queries = q

	// Initialize the bulk subscriber importer.
	importNotifCB := func(subject string, data interface{}) error {
		go sendNotification(app.Constants.NotifyEmails, subject, notifTplImport, data, app)
		return nil
	}
	app.Importer = subimporter.New(q.UpsertSubscriber.Stmt,
		q.UpsertBlacklistSubscriber.Stmt,
		q.UpdateListsDate.Stmt,
		db.DB,
		importNotifCB)

	// Prepare notification e-mail templates.
	notifTpls, err := compileNotifTpls("/email-templates/*.html", fs, app)
	if err != nil {
		logger.Fatalf("error loading e-mail notification templates: %v", err)
	}
	app.NotifTpls = notifTpls

	// Static URLS.
	// url.com/subscription/{campaign_uuid}/{subscriber_uuid}
	c.UnsubscribeURL = fmt.Sprintf("%s/subscription/%%s/%%s", app.Constants.RootURL)

	// url.com/subscription/optin/{subscriber_uuid}
	c.OptinURL = fmt.Sprintf("%s/subscription/optin/%%s?%%s", app.Constants.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
	c.LinkTrackURL = fmt.Sprintf("%s/link/%%s/%%s/%%s", app.Constants.RootURL)

	// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
	c.ViewTrackURL = fmt.Sprintf("%s/campaign/%%s/%%s/px.png", app.Constants.RootURL)

	// Initialize the campaign manager.
	campNotifCB := func(subject string, data interface{}) error {
		return sendNotification(app.Constants.NotifyEmails, subject, notifTplCampaign, data, app)
	}
	m := manager.New(manager.Config{
		Concurrency:   ko.Int("app.concurrency"),
		MaxSendErrors: ko.Int("app.max_send_errors"),
		FromEmail:     app.Constants.FromEmail,
		UnsubURL:      c.UnsubscribeURL,
		OptinURL:      c.OptinURL,
		LinkTrackURL:  c.LinkTrackURL,
		ViewTrackURL:  c.ViewTrackURL,
	}, newManagerDB(q), campNotifCB, logger)
	app.Manager = m

	// Add messengers.
	app.Messenger = initMessengers(app.Manager)

	// Add uploader
	app.Media = initMediaStore()

	// Initialize the workers that push out messages.
	go m.Run(time.Second * 5)
	m.SpawnWorkers()

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

	// Parse user facing templates.
	tpl, err := stuffbin.ParseTemplatesGlob(nil, fs, "/public/templates/*.html")
	if err != nil {
		logger.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &tplRenderer{
		templates:  tpl,
		RootURL:    c.RootURL,
		LogoURL:    c.LogoURL,
		FaviconURL: c.FaviconURL}

	// Register HTTP handlers and static file servers.
	fSrv := app.FS.FileServer()
	srv.GET("/public/*", echo.WrapHandler(fSrv))
	srv.GET("/frontend/*", echo.WrapHandler(fSrv))
	if ko.String("upload.provider") == "filesystem" {
		srv.Static(ko.String("upload.filesystem.upload_uri"), ko.String("upload.filesystem.upload_path"))
	}
	registerHandlers(srv)
	srv.Logger.Fatal(srv.Start(ko.String("app.address")))
}
