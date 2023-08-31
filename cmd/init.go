package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/bounce"
	"github.com/knadh/listmonk/internal/bounce/mailbox"
	"github.com/knadh/listmonk/internal/captcha"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/media/providers/filesystem"
	"github.com/knadh/listmonk/internal/media/providers/s3"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/messenger/postback"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	flag "github.com/spf13/pflag"
)

const (
	queryFilePath = "queries.sql"

	// Root URI of the admin frontend.
	adminRoot = "/admin"
)

// constants contains static, constant config values required by the app.
type constants struct {
	SiteName                      string   `koanf:"site_name"`
	RootURL                       string   `koanf:"root_url"`
	LogoURL                       string   `koanf:"logo_url"`
	FaviconURL                    string   `koanf:"favicon_url"`
	FromEmail                     string   `koanf:"from_email"`
	NotifyEmails                  []string `koanf:"notify_emails"`
	EnablePublicSubPage           bool     `koanf:"enable_public_subscription_page"`
	EnablePublicArchive           bool     `koanf:"enable_public_archive"`
	EnablePublicArchiveRSSContent bool     `koanf:"enable_public_archive_rss_content"`
	SendOptinConfirmation         bool     `koanf:"send_optin_confirmation"`
	Lang                          string   `koanf:"lang"`
	DBBatchSize                   int      `koanf:"batch_size"`
	Privacy                       struct {
		IndividualTracking bool            `koanf:"individual_tracking"`
		AllowPreferences   bool            `koanf:"allow_preferences"`
		AllowBlocklist     bool            `koanf:"allow_blocklist"`
		AllowExport        bool            `koanf:"allow_export"`
		AllowWipe          bool            `koanf:"allow_wipe"`
		RecordOptinIP      bool            `koanf:"record_optin_ip"`
		Exportable         map[string]bool `koanf:"-"`
		DomainBlocklist    []string        `koanf:"-"`
	} `koanf:"privacy"`
	Security struct {
		EnableCaptcha bool   `koanf:"enable_captcha"`
		CaptchaKey    string `koanf:"captcha_key"`
		CaptchaSecret string `koanf:"captcha_secret"`
	} `koanf:"security"`
	AdminUsername []byte `koanf:"admin_username"`
	AdminPassword []byte `koanf:"admin_password"`

	Appearance struct {
		AdminCSS  []byte `koanf:"admin.custom_css"`
		AdminJS   []byte `koanf:"admin.custom_js"`
		PublicCSS []byte `koanf:"public.custom_css"`
		PublicJS  []byte `koanf:"public.custom_js"`
	}

	UnsubURL     string
	LinkTrackURL string
	ViewTrackURL string
	OptinURL     string
	MessageURL   string
	ArchiveURL   string

	MediaUpload struct {
		Provider   string
		Extensions []string
	}

	BounceWebhooksEnabled bool
	BounceSESEnabled      bool
	BounceSendgridEnabled bool
	BouncePostmarkEnabled bool
}

type notifTpls struct {
	tpls        *template.Template
	contentType string
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
	f.Bool("install", false, "setup database (first time)")
	f.Bool("idempotent", false, "make --install run only if the database isn't already setup")
	f.Bool("upgrade", false, "upgrade database to the current version")
	f.Bool("version", false, "show current version of the build")
	f.Bool("new-config", false, "generate sample config file")
	f.String("static-dir", "", "(optional) path to directory with static files")
	f.String("i18n-dir", "", "(optional) path to directory with i18n language files")
	f.Bool("yes", false, "assume 'yes' to prompts during --install/upgrade")
	f.Bool("passive", false, "run in passive mode where campaigns are not processed")
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
			lo.Fatalf("error loading config from file: %v.", err)
		}
	}
}

// initFileSystem initializes the stuffbin FileSystem to provide
// access to bundled static assets to the app.
func initFS(appDir, frontendDir, staticDir, i18nDir string) stuffbin.FileSystem {
	var (
		// stuffbin real_path:virtual_alias paths to map local assets on disk
		// when there an embedded filestystem is not found.

		// These paths are joined with appDir.
		appFiles = []string{
			"./config.toml.sample:config.toml.sample",
			"./queries.sql:queries.sql",
			"./schema.sql:schema.sql",
		}

		frontendFiles = []string{
			// Admin frontend's static assets accessible at /admin/* during runtime.
			// These paths are sourced from frontendDir.
			"./:/admin",
		}

		staticFiles = []string{
			// These paths are joined with staticDir.
			"./email-templates:static/email-templates",
			"./public:/public",
		}

		i18nFiles = []string{
			// These paths are joined with i18nDir.
			"./:/i18n",
		}
	)

	// Get the executable's path.
	path, err := os.Executable()
	if err != nil {
		lo.Fatalf("error getting executable path: %v", err)
	}

	// Load embedded files in the executable.
	hasEmbed := true
	fs, err := stuffbin.UnStuff(path)
	if err != nil {
		hasEmbed = false

		// Running in local mode. Load local assets into
		// the in-memory stuffbin.FileSystem.
		lo.Printf("unable to initialize embedded filesystem (%v). Using local filesystem", err)

		fs, err = stuffbin.NewLocalFS("/")
		if err != nil {
			lo.Fatalf("failed to initialize local file for assets: %v", err)
		}
	}

	// If the embed failed, load app and frontend files from the compile-time paths.
	files := []string{}
	if !hasEmbed {
		files = append(files, joinFSPaths(appDir, appFiles)...)
		files = append(files, joinFSPaths(frontendDir, frontendFiles)...)
	}

	// Irrespective of the embeds, if there are user specified static or i18n paths,
	// load files from there and override default files (embedded or picked up from CWD).
	if !hasEmbed || i18nDir != "" {
		if i18nDir == "" {
			// Default dir in cwd.
			i18nDir = "i18n"
		}
		lo.Printf("loading i18n files from: %v", i18nDir)
		files = append(files, joinFSPaths(i18nDir, i18nFiles)...)
	}

	if !hasEmbed || staticDir != "" {
		if staticDir == "" {
			// Default dir in cwd.
			staticDir = "static"
		}
		lo.Printf("loading static files from: %v", staticDir)
		files = append(files, joinFSPaths(staticDir, staticFiles)...)
	}

	// No additional files to load.
	if len(files) == 0 {
		return fs
	}

	// Load files from disk and overlay into the FS.
	fStatic, err := stuffbin.NewLocalFS("/", files...)
	if err != nil {
		lo.Fatalf("failed reading static files from disk: '%s': %v", staticDir, err)
	}

	if err := fs.Merge(fStatic); err != nil {
		lo.Fatalf("error merging static files: '%s': %v", staticDir, err)
	}

	return fs
}

// initDB initializes the main DB connection pool and parse and loads the app's
// SQL queries into a prepared query map.
func initDB() *sqlx.DB {
	var c struct {
		Host        string        `koanf:"host"`
		Port        int           `koanf:"port"`
		User        string        `koanf:"user"`
		Password    string        `koanf:"password"`
		DBName      string        `koanf:"database"`
		SSLMode     string        `koanf:"ssl_mode"`
		Params      string        `koanf:"params"`
		MaxOpen     int           `koanf:"max_open"`
		MaxIdle     int           `koanf:"max_idle"`
		MaxLifetime time.Duration `koanf:"max_lifetime"`
	}
	if err := ko.Unmarshal("db", &c); err != nil {
		lo.Fatalf("error loading db config: %v", err)
	}

	lo.Printf("connecting to db: %s:%d/%s", c.Host, c.Port, c.DBName)
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s %s", c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Params))
	if err != nil {
		lo.Fatalf("error connecting to DB: %v", err)
	}

	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.MaxLifetime)

	return db
}

// readQueries reads named SQL queries from the SQL queries file into a query map.
func readQueries(sqlFile string, db *sqlx.DB, fs stuffbin.FileSystem) goyesql.Queries {
	// Load SQL queries.
	qB, err := fs.Read(sqlFile)
	if err != nil {
		lo.Fatalf("error reading SQL file %s: %v", sqlFile, err)
	}
	qMap, err := goyesql.ParseBytes(qB)
	if err != nil {
		lo.Fatalf("error parsing SQL queries: %v", err)
	}

	return qMap
}

// prepareQueries queries prepares a query map and returns a *Queries
func prepareQueries(qMap goyesql.Queries, db *sqlx.DB, ko *koanf.Koanf) *models.Queries {
	var (
		countQuery = "get-campaign-analytics-counts"
		linkSel    = "*"
	)
	if ko.Bool("privacy.individual_tracking") {
		countQuery = "get-campaign-analytics-unique-counts"
		linkSel = "DISTINCT subscriber_id"
	}

	// These don't exist in the SQL file but are in the queries struct to be prepared.
	qMap["get-campaign-view-counts"] = &goyesql.Query{
		Query: fmt.Sprintf(qMap[countQuery].Query, "campaign_views"),
		Tags:  map[string]string{"name": "get-campaign-view-counts"},
	}
	qMap["get-campaign-click-counts"] = &goyesql.Query{
		Query: fmt.Sprintf(qMap[countQuery].Query, "link_clicks"),
		Tags:  map[string]string{"name": "get-campaign-click-counts"},
	}
	qMap["get-campaign-link-counts"].Query = fmt.Sprintf(qMap["get-campaign-link-counts"].Query, linkSel)

	// Scan and prepare all queries.
	var q models.Queries
	if err := goyesqlx.ScanToStruct(&q, qMap, db.Unsafe()); err != nil {
		lo.Fatalf("error preparing SQL queries: %v", err)
	}

	return &q
}

// initSettings loads settings from the DB into the given Koanf map.
func initSettings(query string, db *sqlx.DB, ko *koanf.Koanf) {
	var s types.JSONText
	if err := db.Get(&s, query); err != nil {
		msg := err.Error()
		if err, ok := err.(*pq.Error); ok {
			if err.Detail != "" {
				msg = fmt.Sprintf("%s. %s", err, err.Detail)
			}
		}

		lo.Fatalf("error reading settings from DB: %s", msg)
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
		lo.Fatalf("error loading app.privacy config: %v", err)
	}
	if err := ko.Unmarshal("security", &c.Security); err != nil {
		lo.Fatalf("error loading app.security config: %v", err)
	}
	if err := ko.UnmarshalWithConf("appearance", &c.Appearance, koanf.UnmarshalConf{FlatPaths: true}); err != nil {
		lo.Fatalf("error loading app.appearance config: %v", err)
	}

	c.RootURL = strings.TrimRight(c.RootURL, "/")
	c.Lang = ko.String("app.lang")
	c.Privacy.Exportable = maps.StringSliceToLookupMap(ko.Strings("privacy.exportable"))
	c.MediaUpload.Provider = ko.String("upload.provider")
	c.MediaUpload.Extensions = ko.Strings("upload.extensions")
	c.Privacy.DomainBlocklist = ko.Strings("privacy.domain_blocklist")

	// Static URLS.
	// url.com/subscription/{campaign_uuid}/{subscriber_uuid}
	c.UnsubURL = fmt.Sprintf("%s/subscription/%%s/%%s", c.RootURL)

	// url.com/subscription/optin/{subscriber_uuid}
	c.OptinURL = fmt.Sprintf("%s/subscription/optin/%%s?%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
	c.LinkTrackURL = fmt.Sprintf("%s/link/%%s/%%s/%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}
	c.MessageURL = fmt.Sprintf("%s/campaign/%%s/%%s", c.RootURL)

	// url.com/archive
	c.ArchiveURL = c.RootURL + "/archive"

	// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
	c.ViewTrackURL = fmt.Sprintf("%s/campaign/%%s/%%s/px.png", c.RootURL)

	c.BounceWebhooksEnabled = ko.Bool("bounce.webhooks_enabled")
	c.BounceSESEnabled = ko.Bool("bounce.ses_enabled")
	c.BounceSendgridEnabled = ko.Bool("bounce.sendgrid_enabled")
	c.BouncePostmarkEnabled = ko.Bool("bounce.postmark.enabled")

	return &c
}

// initI18n initializes a new i18n instance with the selected language map
// loaded from the filesystem. English is a loaded first as the default map
// and then the selected language is loaded on top of it so that if there are
// missing translations in it, the default English translations show up.
func initI18n(lang string, fs stuffbin.FileSystem) *i18n.I18n {
	i, ok, err := getI18nLang(lang, fs)
	if err != nil {
		if ok {
			lo.Println(err)
		} else {
			lo.Fatal(err)
		}
	}
	return i
}

// initCampaignManager initializes the campaign manager.
func initCampaignManager(q *models.Queries, cs *constants, app *App) *manager.Manager {
	campNotifCB := func(subject string, data interface{}) error {
		return app.sendNotification(cs.NotifyEmails, subject, notifTplCampaign, data)
	}

	if ko.Int("app.concurrency") < 1 {
		lo.Fatal("app.concurrency should be at least 1")
	}
	if ko.Int("app.message_rate") < 1 {
		lo.Fatal("app.message_rate should be at least 1")
	}

	if ko.Bool("passive") {
		lo.Println("running in passive mode. won't process campaigns.")
	}

	return manager.New(manager.Config{
		BatchSize:             ko.Int("app.batch_size"),
		Concurrency:           ko.Int("app.concurrency"),
		MessageRate:           ko.Int("app.message_rate"),
		MaxSendErrors:         ko.Int("app.max_send_errors"),
		FromEmail:             cs.FromEmail,
		IndividualTracking:    ko.Bool("privacy.individual_tracking"),
		UnsubURL:              cs.UnsubURL,
		OptinURL:              cs.OptinURL,
		LinkTrackURL:          cs.LinkTrackURL,
		ViewTrackURL:          cs.ViewTrackURL,
		MessageURL:            cs.MessageURL,
		ArchiveURL:            cs.ArchiveURL,
		UnsubHeader:           ko.Bool("privacy.unsubscribe_header"),
		SlidingWindow:         ko.Bool("app.message_sliding_window"),
		SlidingWindowDuration: ko.Duration("app.message_sliding_window_duration"),
		SlidingWindowRate:     ko.Int("app.message_sliding_window_rate"),
		ScanInterval:          time.Second * 5,
		ScanCampaigns:         !ko.Bool("passive"),
	}, newManagerStore(q, app.core, app.media), campNotifCB, app.i18n, lo)
}

func initTxTemplates(m *manager.Manager, app *App) {
	tpls, err := app.core.GetTemplates(models.TemplateTypeTx, false)
	if err != nil {
		lo.Fatalf("error loading transactional templates: %v", err)
	}

	for _, t := range tpls {
		tpl := t
		if err := tpl.Compile(app.manager.GenericTemplateFuncs()); err != nil {
			lo.Printf("error compiling transactional template %d: %v", tpl.ID, err)
			continue
		}
		m.CacheTpl(tpl.ID, &tpl)
	}
}

// initImporter initializes the bulk subscriber importer.
func initImporter(q *models.Queries, db *sqlx.DB, app *App) *subimporter.Importer {
	return subimporter.New(
		subimporter.Options{
			DomainBlocklist:    app.constants.Privacy.DomainBlocklist,
			UpsertStmt:         q.UpsertSubscriber.Stmt,
			BlocklistStmt:      q.UpsertBlocklistSubscriber.Stmt,
			UpdateListDateStmt: q.UpdateListsDate.Stmt,
			NotifCB: func(subject string, data interface{}) error {
				app.sendNotification(app.constants.NotifyEmails, subject, notifTplImport, data)
				return nil
			},
		}, db.DB, app.i18n)
}

// initSMTPMessenger initializes the SMTP messenger.
func initSMTPMessenger(m *manager.Manager) manager.Messenger {
	var (
		mapKeys = ko.MapKeys("smtp")
		servers = make([]email.Server, 0, len(mapKeys))
	)

	items := ko.Slices("smtp")
	if len(items) == 0 {
		lo.Fatalf("no SMTP servers found in config")
	}

	// Load the config for multiple SMTP servers.
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
func initPostbackMessengers(m *manager.Manager) []manager.Messenger {
	items := ko.Slices("messengers")
	if len(items) == 0 {
		return nil
	}

	var out []manager.Messenger
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
		var o s3.Opt
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
		up, err := filesystem.New(o)
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
func initNotifTemplates(path string, fs stuffbin.FileSystem, i *i18n.I18n, cs *constants) *notifTpls {
	// Register utility functions that the e-mail templates can use.
	funcs := template.FuncMap{
		"RootURL": func() string {
			return cs.RootURL
		},
		"LogoURL": func() string {
			return cs.LogoURL
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
		"L": func() *i18n.I18n {
			return i
		},
		"Safe": func(safeHTML string) template.HTML {
			return template.HTML(safeHTML)
		},
	}

	for k, v := range sprig.GenericFuncMap() {
		funcs[k] = v
	}

	tpls, err := stuffbin.ParseTemplatesGlob(funcs, fs, "/static/email-templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing e-mail notif templates: %v", err)
	}

	html, err := fs.Read("/static/email-templates/base.html")
	if err != nil {
		lo.Fatalf("error reading static/email-templates/base.html: %v", err)
	}

	out := &notifTpls{
		tpls:        tpls,
		contentType: models.CampaignContentTypeHTML,
	}

	// Determine whether the notification templates are HTML or plaintext.
	// Copy the first few (arbitrary) bytes of the template and check if has the <!doctype html> tag.
	ln := 256
	if len(html) < ln {
		ln = len(html)
	}
	h := make([]byte, ln)
	copy(h, html[0:ln])

	if !bytes.Contains(bytes.ToLower(h), []byte("<!doctype html")) {
		out.contentType = models.CampaignContentTypePlain
		lo.Println("system e-mail templates are plaintext")
	}

	return out
}

// initBounceManager initializes the bounce manager that scans mailboxes and listens to webhooks
// for incoming bounce events.
func initBounceManager(app *App) *bounce.Manager {
	opt := bounce.Opt{
		WebhooksEnabled: ko.Bool("bounce.webhooks_enabled"),
		SESEnabled:      ko.Bool("bounce.ses_enabled"),
		SendgridEnabled: ko.Bool("bounce.sendgrid_enabled"),
		SendgridKey:     ko.String("bounce.sendgrid_key"),
		Postmark: struct {
			Enabled  bool
			Username string
			Password string
		}{
			ko.Bool("bounce.postmark.enabled"),
			ko.String("bounce.postmark.username"),
			ko.String("bounce.postmark.password"),
		},
		RecordBounceCB: app.core.RecordBounce,
	}

	// For now, only one mailbox is supported.
	for _, b := range ko.Slices("bounce.mailboxes") {
		if !b.Bool("enabled") {
			continue
		}

		var boxOpt mailbox.Opt
		if err := b.UnmarshalWithConf("", &boxOpt, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading bounce mailbox config: %v", err)
		}

		opt.MailboxType = b.String("type")
		opt.MailboxEnabled = true
		opt.Mailbox = boxOpt
		break
	}

	b, err := bounce.New(opt, &bounce.Queries{
		RecordQuery: app.queries.RecordBounce,
	}, app.log)
	if err != nil {
		lo.Fatalf("error initializing bounce manager: %v", err)
	}

	return b
}

func initAbout(q *models.Queries, db *sqlx.DB) about {
	var (
		mem runtime.MemStats
	)

	// Memory / alloc stats.
	runtime.ReadMemStats(&mem)

	info := types.JSONText(`{}`)
	if err := db.QueryRow(q.GetDBInfo).Scan(&info); err != nil {
		lo.Printf("WARNING: error getting database version: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		lo.Printf("WARNING: error getting hostname: %v", err)
	}

	return about{
		Version:   versionString,
		Build:     buildString,
		GoArch:    runtime.GOARCH,
		GoVersion: runtime.Version(),
		Database:  info,
		System: aboutSystem{
			NumCPU: runtime.NumCPU(),
		},
		Host: aboutHost{
			OS:       runtime.GOOS,
			Machine:  runtime.GOARCH,
			Hostname: hostname,
		},
	}

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
	tpl, err := stuffbin.ParseTemplatesGlob(template.FuncMap{
		"L": func() *i18n.I18n {
			return app.i18n
		}}, app.fs, "/public/templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &tplRenderer{
		templates:           tpl,
		SiteName:            app.constants.SiteName,
		RootURL:             app.constants.RootURL,
		LogoURL:             app.constants.LogoURL,
		FaviconURL:          app.constants.FaviconURL,
		EnablePublicSubPage: app.constants.EnablePublicSubPage,
		EnablePublicArchive: app.constants.EnablePublicArchive,
	}

	// Initialize the static file server.
	fSrv := app.fs.FileServer()

	// Public (subscriber) facing static files.
	srv.GET("/public/static/*", echo.WrapHandler(fSrv))

	// Admin (frontend) facing static files.
	srv.GET("/admin/static/*", echo.WrapHandler(fSrv))

	// Public (subscriber) facing media upload files.
	if ko.String("upload.provider") == "filesystem" && ko.String("upload.filesystem.upload_uri") != "" {
		srv.Static(ko.String("upload.filesystem.upload_uri"), ko.String("upload.filesystem.upload_path"))
	}

	// Register all HTTP handlers.
	initHTTPHandlers(srv, app)

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

func initCaptcha() *captcha.Captcha {
	return captcha.New(captcha.Opt{
		CaptchaSecret: ko.String("security.captcha_secret"),
	})
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

func joinFSPaths(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		// real_path:stuffbin_alias
		f := strings.Split(p, ":")

		out = append(out, path.Join(root, f[0])+":"+f[1])
	}

	return out
}
