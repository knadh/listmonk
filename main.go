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
	"github.com/knadh/listmonk/manager"
	"github.com/knadh/listmonk/messenger"
	"github.com/knadh/listmonk/subimporter"
	"github.com/labstack/echo"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type constants struct {
	AssetPath    string   `mapstructure:"asset_path"`
	RootURL      string   `mapstructure:"root"`
	UploadPath   string   `mapstructure:"upload_path"`
	UploadURI    string   `mapstructure:"upload_uri"`
	FromEmail    string   `mapstructure:"from_email"`
	NotifyEmails []string `mapstructure:"notify_emails"`
}

// App contains the "global" components that are
// passed around, especially through HTTP handlers.
type App struct {
	Constants *constants
	DB        *sqlx.DB
	Queries   *Queries
	Importer  *subimporter.Importer
	Manager   *manager.Manager
	Logger    *log.Logger
	NotifTpls *template.Template
	Messenger messenger.Messenger
}

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "SYS: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Register --help handler.
	flagSet := flag.NewFlagSet("config", flag.ContinueOnError)
	flagSet.Usage = func() {
		fmt.Println(flagSet.FlagUsages())
		os.Exit(0)
	}

	// Setup the default configuration.
	viper.SetConfigName("config")
	flagSet.StringSlice("config", []string{"config.toml"},
		"Path to one or more config files (will be merged in order)")
	flagSet.Bool("install", false, "Run first time installation")
	flagSet.Bool("version", false, "Current version of the build")

	// Process flags.
	flagSet.Parse(os.Args[1:])
	viper.BindPFlags(flagSet)

	// Read the config files.
	cfgs := viper.GetStringSlice("config")
	for _, c := range cfgs {
		logger.Printf("reading config: %s", c)
		viper.SetConfigFile(c)

		if err := viper.MergeInConfig(); err != nil {
			logger.Fatalf("error reading config: %s", err)
		}
	}
}

// registerHandlers registers HTTP handlers.
func registerHandlers(e *echo.Echo) {
	e.GET("/", handleIndexPage)
	e.GET("/api/config.js", handleGetConfigScript)
	e.GET("/api/dashboard/stats", handleGetDashboardStats)
	e.GET("/api/users", handleGetUsers)
	e.POST("/api/users", handleCreateUser)
	e.DELETE("/api/users/:id", handleDeleteUser)

	e.GET("/api/subscribers/:id", handleGetSubscriber)
	e.POST("/api/subscribers", handleCreateSubscriber)
	e.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	e.PUT("/api/subscribers/blacklist", handleBlacklistSubscribers)
	e.PUT("/api/subscribers/:id/blacklist", handleBlacklistSubscribers)
	e.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists)
	e.PUT("/api/subscribers/lists", handleManageSubscriberLists)
	e.DELETE("/api/subscribers/:id", handleDeleteSubscribers)
	e.DELETE("/api/subscribers", handleDeleteSubscribers)

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	e.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery)
	e.PUT("/api/subscribers/query/blacklist", handleBlacklistSubscribersByQuery)
	e.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery)

	e.GET("/api/subscribers", handleQuerySubscribers)

	e.GET("/api/import/subscribers", handleGetImportSubscribers)
	e.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats)
	e.POST("/api/import/subscribers", handleImportSubscribers)
	e.DELETE("/api/import/subscribers", handleStopImportSubscribers)

	e.GET("/api/lists", handleGetLists)
	e.GET("/api/lists/:id", handleGetLists)
	e.POST("/api/lists", handleCreateList)
	e.PUT("/api/lists/:id", handleUpdateList)
	e.DELETE("/api/lists/:id", handleDeleteLists)

	e.GET("/api/campaigns", handleGetCampaigns)
	e.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats)
	e.GET("/api/campaigns/:id", handleGetCampaigns)
	e.GET("/api/campaigns/:id/preview", handlePreviewCampaign)
	e.POST("/api/campaigns/:id/preview", handlePreviewCampaign)
	e.POST("/api/campaigns/:id/test", handleTestCampaign)
	e.POST("/api/campaigns", handleCreateCampaign)
	e.PUT("/api/campaigns/:id", handleUpdateCampaign)
	e.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus)
	e.DELETE("/api/campaigns/:id", handleDeleteCampaign)

	e.GET("/api/media", handleGetMedia)
	e.POST("/api/media", handleUploadMedia)
	e.DELETE("/api/media/:id", handleDeleteMedia)

	e.GET("/api/templates", handleGetTemplates)
	e.GET("/api/templates/:id", handleGetTemplates)
	e.GET("/api/templates/:id/preview", handlePreviewTemplate)
	e.POST("/api/templates/preview", handlePreviewTemplate)
	e.POST("/api/templates", handleCreateTemplate)
	e.PUT("/api/templates/:id", handleUpdateTemplate)
	e.PUT("/api/templates/:id/default", handleTemplateSetDefault)
	e.DELETE("/api/templates/:id", handleDeleteTemplate)

	// Subscriber facing views.
	e.GET("/unsubscribe/:campUUID/:subUUID", handleUnsubscribePage)
	e.POST("/unsubscribe/:campUUID/:subUUID", handleUnsubscribePage)
	e.GET("/link/:linkUUID/:campUUID/:subUUID", handleLinkRedirect)
	e.GET("/campaign/:campUUID/:subUUID/px.png", handleRegisterCampaignView)

	// Static views.
	e.GET("/lists", handleIndexPage)
	e.GET("/subscribers", handleIndexPage)
	e.GET("/subscribers/lists/:listID", handleIndexPage)
	e.GET("/subscribers/import", handleIndexPage)
	e.GET("/campaigns", handleIndexPage)
	e.GET("/campaigns/new", handleIndexPage)
	e.GET("/campaigns/media", handleIndexPage)
	e.GET("/campaigns/templates", handleIndexPage)
	e.GET("/campaigns/:campignID", handleIndexPage)
}

// initMessengers initializes various messaging backends.
func initMessengers(r *manager.Manager) messenger.Messenger {
	// Load SMTP configurations for the default e-mail Messenger.
	var srv []messenger.Server
	for name := range viper.GetStringMapString("smtp") {
		if !viper.GetBool(fmt.Sprintf("smtp.%s.enabled", name)) {
			logger.Printf("skipped SMTP config %s", name)
			continue
		}

		var s messenger.Server
		viper.UnmarshalKey("smtp."+name, &s)
		s.Name = name
		s.SendTimeout = s.SendTimeout * time.Millisecond
		srv = append(srv, s)

		logger.Printf("loaded SMTP config %s (%s@%s)", s.Name, s.Username, s.Host)
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

func main() {
	// Connect to the DB.
	db, err := connectDB(viper.GetString("db.host"),
		viper.GetInt("db.port"),
		viper.GetString("db.user"),
		viper.GetString("db.password"),
		viper.GetString("db.database"))
	if err != nil {
		logger.Fatalf("error connecting to DB: %v", err)
	}
	defer db.Close()

	var c constants
	viper.UnmarshalKey("app", &c)
	c.RootURL = strings.TrimRight(c.RootURL, "/")
	c.UploadURI = filepath.Clean(c.UploadURI)
	c.AssetPath = filepath.Clean(c.AssetPath)

	// Initialize the app context that's passed around.
	app := &App{
		Constants: &c,
		DB:        db,
		Logger:    logger,
	}

	// Load SQL queries.
	qMap, err := goyesql.ParseFile("queries.sql")
	if err != nil {
		logger.Fatalf("error loading SQL queries: %v", err)
	}

	// First time installation.
	if viper.GetBool("install") {
		install(app, qMap)
		return
	}

	// Map queries to the query container.
	q := &Queries{}
	if err := scanQueriesToStruct(q, qMap, db.Unsafe()); err != nil {
		logger.Fatalf("no SQL queries loaded: %v", err)
	}
	app.Queries = q

	// Importer.
	importNotifCB := func(subject string, data map[string]interface{}) error {
		return sendNotification(notifTplImport, subject, data, app)
	}
	app.Importer = subimporter.New(q.UpsertSubscriber.Stmt,
		q.UpsertBlacklistSubscriber.Stmt,
		db.DB,
		importNotifCB)

	// System e-mail templates.
	notifTpls, err := template.ParseGlob("templates/*.html")
	if err != nil {
		logger.Fatalf("error loading system templates: %v", err)
	}
	app.NotifTpls = notifTpls

	// Campaign daemon.
	campNotifCB := func(subject string, data map[string]interface{}) error {
		return sendNotification(notifTplCampaign, subject, data, app)
	}
	m := manager.New(manager.Config{
		Concurrency:   viper.GetInt("app.concurrency"),
		MaxSendErrors: viper.GetInt("app.max_send_errors"),
		FromEmail:     app.Constants.FromEmail,

		// url.com/unsubscribe/{campaign_uuid}/{subscriber_uuid}
		UnsubscribeURL: fmt.Sprintf("%s/unsubscribe/%%s/%%s", app.Constants.RootURL),

		// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
		LinkTrackURL: fmt.Sprintf("%s/link/%%s/%%s/%%s", app.Constants.RootURL),

		// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
		ViewTrackURL: fmt.Sprintf("%s/campaign/%%s/%%s/px.png", app.Constants.RootURL),
	}, newManagerDB(q), campNotifCB, logger)
	app.Manager = m

	// Add messengers.
	app.Messenger = initMessengers(app.Manager)

	go m.Run(time.Duration(time.Second * 5))
	m.SpawnWorkers()

	// Initialize the server.
	var srv = echo.New()
	srv.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("app", app)
			return next(c)
		}
	})

	// User facing templates.
	tpl, err := template.ParseGlob("public/templates/*.html")
	if err != nil {
		logger.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &Template{
		templates: tpl,
	}
	srv.HideBanner = true

	// Register HTTP middleware.
	// e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	// e.Use(authSession)
	srv.Static("/static", filepath.Join(filepath.Clean(viper.GetString("app.asset_path")), "static"))
	srv.Static("/static/public", "frontend/my/public")
	srv.Static("/public/static", "public/static")
	srv.Static(filepath.Clean(viper.GetString("app.upload_uri")),
		filepath.Clean(viper.GetString("app.upload_path")))
	registerHandlers(srv)

	srv.Logger.Fatal(srv.Start(viper.GetString("app.address")))
}
