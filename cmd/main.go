package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/bounce"
	"github.com/knadh/listmonk/internal/buflog"
	"github.com/knadh/listmonk/internal/captcha"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/events"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/paginator"
	"github.com/knadh/stuffbin"
)

// =======================
// Global App Structure
// =======================
type App struct {
	cfg        *Config
	urlCfg     *UrlConfig
	fs         stuffbin.FileSystem
	db         *sqlx.DB
	queries    *models.Queries
	core       *core.Core
	manager    *manager.Manager
	messengers []manager.Messenger
	emailMsgr  manager.Messenger
	importer   *subimporter.Importer
	auth       *auth.Auth
	media      media.Store
	bounce     *bounce.Manager
	captcha    *captcha.Captcha
	i18n       *i18n.I18n
	pg         *paginator.Paginator
	events     *events.Events
	log        *log.Logger
	bufLog     *buflog.BufLog

	about         about
	fnOptinNotify func(models.Subscriber, []int) (int, error)

	chReload       chan os.Signal
	needsRestart   bool
	needsUserSetup bool
	update         *AppUpdate
	sync.Mutex
}

// =======================
// Global Variables
// =======================
var (
	evStream = events.New()
	bufLog   = buflog.New(5000)
	lo       = log.New(io.MultiWriter(os.Stdout, bufLog, evStream.ErrWriter()), "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	ko      = koanf.New(".")
	fs      stuffbin.FileSystem
	db      *sqlx.DB
	queries *models.Queries

	buildString   string
	versionString string

	appDir      string = "."
	frontendDir string = "frontend/dist"
)

// =======================
// Initialization
// =======================
func init() {
	// Command-line flags
	initFlags(ko)

	if ko.Bool("version") {
		fmt.Println(buildString)
		os.Exit(0)
	}

	lo.Println(buildString)

	// Generate new config
	if ko.Bool("new-config") {
		path := ko.Strings("config")[0]
		if err := newConfigFile(path); err != nil {
			lo.Println(err)
			os.Exit(1)
		}
		lo.Printf("generated %s. Edit and run --install", path)
		os.Exit(0)
	}

	// Load config files
	initConfigFiles(ko.Strings("config"), ko)

	// Load environment variables
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading config from env: %v", err)
	}

	// Database connection
	db = initDB()

	// Filesystem for frontend & static assets
	fs = initFS(appDir, frontendDir, ko.String("static-dir"), ko.String("i18n-dir"))

	// Installer mode
	if ko.Bool("install") {
		install(migList[len(migList)-1].version, db, fs, !ko.Bool("yes"), ko.Bool("idempotent"))
		os.Exit(0)
	}

	// Check DB schema
	if ok, err := checkSchema(db); err != nil {
		log.Fatalf("error checking schema in DB: %v", err)
	} else if !ok {
		lo.Fatal("the database does not appear to be setup. Run --install.")
	}

	// Upgrade DB if requested
	if ko.Bool("upgrade") {
		upgrade(db, fs, !ko.Bool("yes"))
		os.Exit(0)
	}

	checkUpgrade(db)

	// Read SQL queries
	qMap := readQueries(queryFilePath, fs)

	// Load settings
	if q, ok := qMap["get-settings"]; ok {
		initSettings(q.Query, db, ko)
	}

	// Prepare queries
	queries = prepareQueries(qMap, db, ko)
}

// =======================
// Main Function
// =======================
func main() {
	// Initialize core components
	var (
		cfg               = initConstConfig(ko)
		urlCfg            = initUrlConfig(ko)
		i18n              = initI18n(ko.MustString("app.lang"), fs)
		media             = initMediaStore(ko)
		fbOptinNotify     = makeOptinNotifyHook(ko.Bool("privacy.unsubscribe_header"), urlCfg, queries, i18n)
		core              = initCore(fbOptinNotify, queries, db, i18n, ko)
		msgrs             = append(initSMTPMessengers(), initPostbackMessengers(ko)...)
		mgr               = initCampaignManager(msgrs, queries, urlCfg, core, media, i18n, ko)
		importer          = initImporter(queries, db, core, i18n, ko)
		hasUsers, authSvc = initAuth(core, db.DB, ko)
		bounce            *bounce.Manager
		emailMsgr         *email.Emailer
		chReload          = make(chan os.Signal, 1)
	)

	// Initialize bounce manager
	if ko.Bool("bounce.enabled") {
		bounce = initBounceManager(core.RecordBounce, queries.RecordBounce, lo, ko)
	}

	// Assign email messenger
	for _, m := range msgrs {
		if m.Name() == "email" {
			emailMsgr = m.(*email.Emailer)
		}
	}

	initNotifs(fs, i18n, emailMsgr, urlCfg, ko)
	initTxTemplates(mgr, core)

	if ko.Bool("bounce.enabled") {
		go bounce.Run()
	}

	if ko.Bool("app.cache_slow_queries") {
		initCron(core)
	}

	go mgr.Run()

	// Initialize App struct
	app := &App{
		cfg:        cfg,
		urlCfg:     urlCfg,
		fs:         fs,
		db:         db,
		queries:    queries,
		core:       core,
		manager:    mgr,
		messengers: msgrs,
		emailMsgr:  emailMsgr,
		importer:   importer,
		auth:       authSvc,
		media:      media,
		bounce:     bounce,
		captcha:    initCaptcha(),
		i18n:       i18n,
		log:        lo,
		events:     evStream,
		bufLog:     bufLog,
		pg: paginator.New(paginator.Opt{
			DefaultPerPage: 20,
			MaxPerPage:     50,
			NumPageNums:    10,
			PageParam:      "page",
			PerPageParam:   "per_page",
			AllowAll:       true,
		}),
		fnOptinNotify:  fbOptinNotify,
		about:          initAbout(queries, db),
		chReload:       chReload,
		needsUserSetup: !hasUsers,
	}

	// =========================
	// Failed login logging
	// =========================
	if authSvc != nil {
		authSvc.OnFailedLogin = func(username, ip string) {
			lo.Printf("[FAILED LOGIN] username=%s ip=%s time=%s\n", username, ip, time.Now().Format(time.RFC3339))
		}
		authSvc.OnSuccessfulLogin = func(username, ip string) {
			lo.Printf("[LOGIN SUCCESS] username=%s ip=%s time=%s\n", username, ip, time.Now().Format(time.RFC3339))
		}
	}

	if ko.Bool("app.check_updates") {
		go app.checkUpdates(versionString, time.Hour*24)
	}

	srv := initHTTPServer(cfg, urlCfg, i18n, fs, app)

	// Wait for reload signal
	signal.Notify(chReload, syscall.SIGHUP)
	closerWait := make(chan bool)
	<-awaitReload(chReload, closerWait, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		mgr.Close()
		db.Close()
		for _, m := range app.messengers {
			m.Close()
		}
		closerWait <- true
	})
}
