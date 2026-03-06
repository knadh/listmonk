package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gdgvda/cron"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	koanfmaps "github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/bounce"
	"github.com/knadh/listmonk/internal/bounce/mailbox"
	"github.com/knadh/listmonk/internal/captcha"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/media/providers/filesystem"
	"github.com/knadh/listmonk/internal/media/providers/s3"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/messenger/postback"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	flag "github.com/spf13/pflag"
	"gopkg.in/volatiletech/null.v6"
)

const (
	// Path to the SQL queries directory in the embedded FS.
	queryFilePath = "/queries"

	emailMsgr = "email"
)

// UrlConfig contains various URL constants used in the app.
type UrlConfig struct {
	RootURL      string `koanf:"root_url"`
	LogoURL      string `koanf:"logo_url"`
	FaviconURL   string `koanf:"favicon_url"`
	LoginURL     string `koanf:"login_url"`
	UnsubURL     string
	LinkTrackURL string
	ViewTrackURL string
	OptinURL     string
	MessageURL   string
	ArchiveURL   string
}

// Config contains static, constant config values required by arbitrary handlers and functions.
type Config struct {
	SiteName                      string   `koanf:"site_name"`
	FromEmail                     string   `koanf:"from_email"`
	NotifyEmails                  []string `koanf:"notify_emails"`
	EnablePublicSubPage           bool     `koanf:"enable_public_subscription_page"`
	EnablePublicArchive           bool     `koanf:"enable_public_archive"`
	EnablePublicArchiveRSSContent bool     `koanf:"enable_public_archive_rss_content"`
	Lang                          string   `koanf:"lang"`
	DBBatchSize                   int      `koanf:"batch_size"`
	Privacy                       struct {
		IndividualTracking bool            `koanf:"individual_tracking"`
		DisableTracking    bool            `koanf:"disable_tracking"`
		AllowPreferences   bool            `koanf:"allow_preferences"`
		AllowBlocklist     bool            `koanf:"allow_blocklist"`
		AllowExport        bool            `koanf:"allow_export"`
		AllowWipe          bool            `koanf:"allow_wipe"`
		RecordOptinIP      bool            `koanf:"record_optin_ip"`
		UnsubHeader        bool            `koanf:"unsubscribe_header"`
		Exportable         map[string]bool `koanf:"-"`
		DomainBlocklist    []string        `koanf:"-"`
		DomainAllowlist    []string        `koanf:"-"`
	} `koanf:"privacy"`
	Security struct {
		OIDC struct {
			Enabled           bool   `koanf:"enabled"`
			ProviderURL       string `koanf:"provider_url"`
			ProviderName      string `koanf:"provider_name"`
			ClientID          string `koanf:"client_id"`
			ClientSecret      string `koanf:"client_secret"`
			AutoCreateUsers   bool   `koanf:"auto_create_users"`
			DefaultUserRoleID int    `koanf:"default_user_role_id"`
			DefaultListRoleID int    `koanf:"default_list_role_id"`
		} `koanf:"oidc"`

		Captcha struct {
			Altcha struct {
				Enabled    bool `koanf:"enabled"`
				Complexity int  `koanf:"complexity"`
			} `koanf:"altcha"`
			HCaptcha struct {
				Enabled bool   `koanf:"enabled"`
				Key     string `koanf:"key"`
				Secret  string `koanf:"secret"`
			} `koanf:"hcaptcha"`
		} `koanf:"captcha"`

		CorsOrigins []string `koanf:"cors_origins"`
	} `koanf:"security"`

	Appearance struct {
		AdminCSS  []byte `koanf:"admin.custom_css"`
		AdminJS   []byte `koanf:"admin.custom_js"`
		PublicCSS []byte `koanf:"public.custom_css"`
		PublicJS  []byte `koanf:"public.custom_js"`
	}

	HasLegacyUser bool
	AssetVersion  string

	MediaUpload struct {
		Provider   string
		Extensions []string
	}

	BounceWebhooksEnabled     bool
	BounceSESEnabled          bool
	BounceSendgridEnabled     bool
	BouncePostmarkEnabled     bool
	BounceForwardemailEnabled bool

	PermissionsRaw json.RawMessage
	Permissions    map[string]struct{}
}

// initFlags initializes the commandline flags into the Koanf instance.
func initFlags(ko *koanf.Koanf) {
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
	f.Bool("new-config", false, "generate sample config file (at path given in --config)")
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
			"./queries:queries",
			"./schema.sql:schema.sql",
			"./permissions.json:permissions.json",
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

	// Get the executable's execPath.
	execPath, err := os.Executable()
	if err != nil {
		lo.Fatalf("error getting executable path: %v", err)
	}

	// Load embedded files in the executable.
	hasEmbed := true
	fs, err := stuffbin.UnStuff(execPath)
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
		} else {
			// There is a custom static directory. Any paths that aren't in it, exclude.
			sf := []string{}
			for _, def := range staticFiles {
				s := strings.Split(def, ":")[0]
				if _, err := os.Stat(path.Join(staticDir, s)); err == nil {
					sf = append(sf, def)
				}
			}
			staticFiles = sf
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

	return db.Unsafe()
}

func readQueries(dir string, fs stuffbin.FileSystem) goyesql.Queries {
	out := goyesql.Queries{}

	// Glob all the .sql files in the queries directory.
	qPath := path.Join(dir, "/*.sql")
	files, err := fs.Glob(qPath)
	if err != nil {
		lo.Fatalf("error reading *.sql query files from %s: %v", qPath, err)
	}

	// Read and merge queries from all files into one map.
	for _, file := range files {
		// Read the SQL file.
		b, err := fs.Read(file)
		if err != nil {
			lo.Fatalf("error reading SQL file %s: %v", file, err)
		}

		// Parse queries in it into a map.
		mp, err := goyesql.ParseBytes(b)
		if err != nil {
			lo.Fatalf("error parsing SQL queries: %v", err)
		}

		// Merge into the main query map.
		maps.Copy(out, mp)
	}

	return out
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
	if err := goyesqlx.ScanToStruct(&q, qMap, db); err != nil {
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
	var out map[string]any
	if err := json.Unmarshal(s, &out); err != nil {
		lo.Fatalf("error unmarshalling settings from DB: %v", err)
	}
	if err := ko.Load(confmap.Provider(out, "."), nil); err != nil {
		lo.Fatalf("error parsing settings from DB: %v", err)
	}
}

func initUrlConfig(ko *koanf.Koanf) *UrlConfig {
	root := strings.TrimSuffix(ko.String("app.root_url"), "/")

	return &UrlConfig{
		RootURL:    root,
		LogoURL:    ko.String("app.logo_url"),
		FaviconURL: ko.String("app.favicon_url"),
		LoginURL:   path.Join(uriAdmin, "/login"),

		// Static URLS.
		// url.com/subscription/{campaign_uuid}/{subscriber_uuid}
		UnsubURL: fmt.Sprintf("%s/subscription/%%s/%%s", root),

		// url.com/subscription/optin/{subscriber_uuid}
		OptinURL: fmt.Sprintf("%s/subscription/optin/%%s?%%s", root),

		// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
		LinkTrackURL: fmt.Sprintf("%s/link/%%s/%%s/%%s", root),

		// url.com/link/{campaign_uuid}/{subscriber_uuid}
		MessageURL: fmt.Sprintf("%s/campaign/%%s/%%s", root),

		// url.com/archive
		ArchiveURL: root + "/archive",

		// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
		ViewTrackURL: fmt.Sprintf("%s/campaign/%%s/%%s/px.png", root),
	}
}

// initConstConfig initializes the app's global constants from the given koanf instance.
func initConstConfig(ko *koanf.Koanf) *Config {
	// Read constants.
	var c Config
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

	c.Lang = ko.String("app.lang")
	c.Privacy.Exportable = koanfmaps.StringSliceToLookupMap(ko.Strings("privacy.exportable"))
	c.MediaUpload.Provider = ko.String("upload.provider")
	c.MediaUpload.Extensions = ko.Strings("upload.extensions")
	c.Privacy.DomainBlocklist = ko.Strings("privacy.domain_blocklist")
	c.Privacy.DomainAllowlist = ko.Strings("privacy.domain_allowlist")

	c.BounceWebhooksEnabled = ko.Bool("bounce.webhooks_enabled")
	c.BounceSESEnabled = ko.Bool("bounce.ses_enabled")
	c.BounceSendgridEnabled = ko.Bool("bounce.sendgrid_enabled")
	c.BouncePostmarkEnabled = ko.Bool("bounce.postmark.enabled")
	c.BounceForwardemailEnabled = ko.Bool("bounce.forwardemail.enabled")
	c.HasLegacyUser = ko.Exists("app.admin_username") || ko.Exists("app.admin_password")

	b := md5.Sum([]byte(time.Now().String()))
	c.AssetVersion = fmt.Sprintf("%x", b)[0:10]

	pm, err := fs.Read("/permissions.json")
	if err != nil {
		lo.Fatalf("error reading permissions file: %v", err)
	}
	c.PermissionsRaw = pm

	// Make a lookup map of permissions.
	permGroups := []struct {
		Group       string   `json:"group"`
		Permissions []string `json:"permissions"`
	}{}
	if err := json.Unmarshal(pm, &permGroups); err != nil {
		lo.Fatalf("error loading permissions file: %v", err)
	}

	c.Permissions = map[string]struct{}{}
	for _, group := range permGroups {
		for _, g := range group.Permissions {
			c.Permissions[g] = struct{}{}
		}
	}

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

// initCore initializes the CRUD DB core .
func initCore(fnNotify func(sub models.Subscriber, listIDs []int) (int, error), queries *models.Queries, db *sqlx.DB, i *i18n.I18n, ko *koanf.Koanf) *core.Core {
	opt := &core.Opt{
		Constants: core.Constants{
			SendOptinConfirmation: ko.Bool("app.send_optin_confirmation"),
			CacheSlowQueries:      ko.Bool("app.cache_slow_queries"),
		},
		Queries: queries,
		DB:      db,
		I18n:    i,
		Log:     lo,
	}

	// Load bounce config.
	if err := ko.Unmarshal("bounce.actions", &opt.Constants.BounceActions); err != nil {
		lo.Fatalf("error unmarshalling bounce config: %v", err)
	}

	// Initialize the CRUD core.
	return core.New(opt, &core.Hooks{
		SendOptinConfirmation: fnNotify,
	})
}

// initCampaignManager initializes the campaign manager.
func initCampaignManager(msgrs []manager.Messenger, q *models.Queries, u *UrlConfig, co *core.Core, md media.Store, i *i18n.I18n, ko *koanf.Koanf) *manager.Manager {
	if ko.Bool("passive") {
		lo.Println("running in passive mode. won't process campaigns.")
	}

	mgr := manager.New(manager.Config{
		BatchSize:             ko.Int("app.batch_size"),
		Concurrency:           ko.Int("app.concurrency"),
		MessageRate:           ko.Int("app.message_rate"),
		MaxSendErrors:         ko.Int("app.max_send_errors"),
		FromEmail:             ko.String("app.from_email"),
		IndividualTracking:    ko.Bool("privacy.individual_tracking"),
		DisableTracking:       ko.Bool("privacy.disable_tracking"),
		UnsubURL:              u.UnsubURL,
		OptinURL:              u.OptinURL,
		LinkTrackURL:          u.LinkTrackURL,
		ViewTrackURL:          u.ViewTrackURL,
		MessageURL:            u.MessageURL,
		ArchiveURL:            u.ArchiveURL,
		RootURL:               u.RootURL,
		UnsubHeader:           ko.Bool("privacy.unsubscribe_header"),
		SlidingWindow:         ko.Bool("app.message_sliding_window"),
		SlidingWindowDuration: ko.Duration("app.message_sliding_window_duration"),
		SlidingWindowRate:     ko.Int("app.message_sliding_window_rate"),
		ScanInterval:          time.Second * 5,
		ScanCampaigns:         !ko.Bool("passive"),
	}, newManagerStore(q, co, md), i, lo)

	// Attach all messengers to the campaign manager.
	for _, m := range msgrs {
		mgr.AddMessenger(m)
	}

	return mgr
}

// initTxTemplates initializes and compiles the transactional templates and caches them in-memory.
func initTxTemplates(m *manager.Manager, co *core.Core) {
	tpls, err := co.GetTemplates(models.TemplateTypeTx, false)
	if err != nil {
		lo.Fatalf("error loading transactional templates: %v", err)
	}

	for _, t := range tpls {
		tpl := t
		if err := tpl.Compile(m.GenericTemplateFuncs()); err != nil {
			lo.Printf("error compiling transactional template %d: %v", tpl.ID, err)
			continue
		}
		m.CacheTpl(tpl.ID, &tpl)
	}
}

// initImporter initializes the bulk subscriber importer.
func initImporter(q *models.Queries, db *sqlx.DB, core *core.Core, i *i18n.I18n, ko *koanf.Koanf) *subimporter.Importer {
	return subimporter.New(
		subimporter.Options{
			DomainBlocklist:    ko.Strings("privacy.domain_blocklist"),
			DomainAllowlist:    ko.Strings("privacy.domain_allowlist"),
			UpsertStmt:         q.UpsertSubscriber.Stmt,
			BlocklistStmt:      q.UpsertBlocklistSubscriber.Stmt,
			UpdateListDateStmt: q.UpdateListsDate.Stmt,

			// Hook for triggering admin notifications and refreshing stats materialized
			// views after a successful import.
			PostCB: func(subject string, data any) error {
				// Refresh cached subscriber counts and stats.
				core.RefreshMatViews(true)

				// Send admin notification.
				notifs.NotifySystem(subject, notifs.TplImport, data, nil)
				return nil
			},
		}, db.DB, i)
}

// initSMTPMessenger initializes the combined and individual SMTP messengers.
func initSMTPMessengers() []manager.Messenger {
	var (
		servers = []email.Server{}
		out     = []manager.Messenger{}
	)

	// Load the config for multiple SMTP servers.
	for _, item := range ko.Slices("smtp") {
		if !item.Bool("enabled") {
			continue
		}

		// Read the SMTP config.
		var s email.Server
		if err := item.UnmarshalWithConf("", &s, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading SMTP config: %v", err)
		}

		servers = append(servers, s)
		lo.Printf("initialized email (SMTP) messenger: %s@%s", item.String("username"), item.String("host"))

		// If the server has a name, initialize it as a standalone e-mail messenger
		// allowing campaigns to select individual SMTPs. In the UI and config, it'll appear as `email / $name`.
		if s.Name != "" {
			msgr, err := email.New(s.Name, s)
			if err != nil {
				lo.Fatalf("error initializing e-mail messenger: %v", err)
			}
			out = append(out, msgr)
		}
	}

	// Initialize the 'email' messenger with all SMTP servers.
	msgr, err := email.New(email.MessengerName, servers...)
	if err != nil {
		lo.Fatalf("error initializing e-mail messenger: %v", err)
	}

	// If it's just one server, return the default "email" messenger.
	if len(servers) == 1 {
		return []manager.Messenger{msgr}
	}

	// If there are multiple servers, prepend the group "email" to be the first one.
	out = append([]manager.Messenger{msgr}, out...)

	return out
}

// initPostbackMessengers initializes and returns all the enabled
// HTTP postback messenger backends.
func initPostbackMessengers(ko *koanf.Koanf) []manager.Messenger {
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
func initMediaStore(ko *koanf.Koanf) media.Store {
	switch provider := ko.String("upload.provider"); provider {
	case "s3":
		var o s3.Opt
		ko.Unmarshal("upload.s3", &o)
		o.RootURL = ko.String("app.root_url")

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

// initNotifs initializes the notifier with the system e-mail templates.
func initNotifs(fs stuffbin.FileSystem, i *i18n.I18n, em *email.Emailer, u *UrlConfig, ko *koanf.Koanf) {
	tpls, err := stuffbin.ParseTemplatesGlob(initTplFuncs(i, u), fs, "/static/email-templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing e-mail notif templates: %v", err)
	}

	// Read the notification templates.
	html, err := fs.Read("/static/email-templates/base.html")
	if err != nil {
		lo.Fatalf("error reading static/email-templates/base.html: %v", err)
	}

	// Determine whether the notification templates are HTML or plaintext.
	// Copy the first few (arbitrary) bytes of the template and check if has the <!doctype html> tag.
	ln := min(len(html), 256)
	h := make([]byte, ln)
	copy(h, html[0:ln])

	contentType := models.CampaignContentTypeHTML
	if !bytes.Contains(bytes.ToLower(h), []byte("<!doctype html")) {
		contentType = models.CampaignContentTypePlain
		lo.Println("system e-mail templates are plaintext")
	}

	notifs.Initialize(notifs.Opt{
		FromEmail:    ko.String("app.from_email"),
		SystemEmails: ko.Strings("app.notify_emails"),
		ContentType:  contentType,
	}, tpls, em, lo)
}

// initBounceManager initializes the bounce manager that scans mailboxes and listens to webhooks
// for incoming bounce events.
func initBounceManager(cb func(models.Bounce) error, stmt *sqlx.Stmt, lo *log.Logger, ko *koanf.Koanf) *bounce.Manager {
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
		ForwardEmail: struct {
			Enabled bool
			Key     string
		}{
			ko.Bool("bounce.forwardemail.enabled"),
			ko.String("bounce.forwardemail.key"),
		},
		RecordBounceCB: cb,
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

	// Initialize the bounce manager.
	b, err := bounce.New(opt, &bounce.Queries{RecordQuery: stmt}, lo)
	if err != nil {
		lo.Fatalf("error initializing bounce manager: %v", err)
	}

	return b
}

// initAbout initializes the app's /about API endpoint with the app and system info.
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
func initHTTPServer(cfg *Config, urlCfg *UrlConfig, i *i18n.I18n, fs stuffbin.FileSystem, app *App) *echo.Echo {
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

	tpl, err := stuffbin.ParseTemplatesGlob(initTplFuncs(i, urlCfg), fs, "/public/templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &tplRenderer{
		templates:           tpl,
		SiteName:            cfg.SiteName,
		RootURL:             urlCfg.RootURL,
		LogoURL:             urlCfg.LogoURL,
		FaviconURL:          urlCfg.FaviconURL,
		AssetVersion:        cfg.AssetVersion,
		EnablePublicSubPage: cfg.EnablePublicSubPage,
		EnablePublicArchive: cfg.EnablePublicArchive,
		IndividualTracking:  cfg.Privacy.IndividualTracking,
	}

	// Initialize the static file server.
	fSrv := fs.FileServer()

	// Public (subscriber) facing static files.
	srv.GET("/public/static/*", echo.WrapHandler(fSrv))

	// Admin (frontend) facing static files.
	srv.GET("/admin/static/*", echo.WrapHandler(fSrv))

	// Public (subscriber) facing media upload files.
	var (
		uploadProvider = ko.String("upload.provider")
		uploadFsURI    = ko.String("upload.filesystem.upload_uri")
		publicURL      = ko.String("upload.s3.public_url")
	)
	switch {
	case uploadProvider == "filesystem" && uploadFsURI != "":
		srv.Static(uploadFsURI, ko.String("upload.filesystem.upload_path"))
	case uploadProvider == "s3" && strings.HasPrefix(publicURL, "/"):
		srv.GET(path.Join(publicURL, "/:filepath"), app.ServeS3Media)
	}

	// Register all HTTP handlers.
	initHTTPHandlers(srv, app)

	// Start the server.
	go func() {
		if err := srv.Start(ko.String("app.address")); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				lo.Println("HTTP server shut down")
			} else {
				lo.Fatalf("error starting HTTP server: %v", err)
			}
		}
	}()

	return srv
}

// initCaptcha initializes the captcha service.
func initCaptcha() *captcha.Captcha {
	var opt captcha.Opt
	if err := ko.Unmarshal("security.captcha", &opt); err != nil {
		lo.Fatalf("error loading captcha config: %v", err)
	}

	return captcha.New(opt)
}

// initCron initializes cron jobs for slow query cache refresh and database vacuum.
func initCron(co *core.Core, db *sqlx.DB) {
	c := cron.New(cron.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))

	// Slow query cache cron job.
	if ko.Bool("app.cache_slow_queries") {
		intval := ko.String("app.cache_slow_queries_interval")
		if intval == "" {
			lo.Println("error: invalid cron interval string for slow query cache")
		} else {
			_, err := c.Add(intval, func() {
				lo.Println("refreshing slow query cache")
				_ = co.RefreshMatViews(true)
				lo.Println("done refreshing slow query cache")
			})
			if err != nil {
				lo.Printf("error initializing slow cache query cron: %v", err)
			} else {
				lo.Printf("IMPORTANT: database slow query caching is enabled. Aggregate numbers and stats will not be realtime. Next refresh at: %v", c.Entries()[len(c.Entries())-1].Next)
			}
		}
	}

	// Database vacuum cron job.
	if ko.Bool("maintenance.db.vacuum") {
		intval := ko.String("maintenance.db.vacuum_cron_interval")
		if intval == "" {
			lo.Println("error: invalid cron interval string for database vacuum")
		} else {
			_, err := c.Add(intval, func() {
				RunDBVacuum(db, lo)
			})
			if err != nil {
				lo.Printf("error initializing database vacuum cron: %v", err)
			} else {
				lo.Printf("database VACUUM cron enabled at interval: %s", intval)
			}
		}
	}

	if len(c.Entries()) > 0 {
		c.Start()
	}
}

// awaitReload waits for a SIGHUP signal to reload the app. Every setting change on the UI causes a reload.
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

// initTplFuncs returns a generic template func map with custom template
// functions and sprig template functions.
func initTplFuncs(i *i18n.I18n, u *UrlConfig) template.FuncMap {
	funcs := template.FuncMap{
		"RootURL": func() string {
			return u.RootURL
		},
		"LogoURL": func() string {
			return u.LogoURL
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

	// Copy spring functions.
	sprigFuncs := sprig.GenericFuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")
	delete(sprigFuncs, "getHostByName")

	maps.Copy(funcs, sprigFuncs)

	return funcs
}

// initAuth initializes the auth module with the given DB connection and
func initAuth(co *core.Core, db *sql.DB, ko *koanf.Koanf) (bool, *auth.Auth) {
	var oidcCfg auth.OIDCConfig

	// If OIDC is enabled, set up the OIDC config.
	if ko.Bool("security.oidc.enabled") {
		oidcCfg = auth.OIDCConfig{
			Enabled:           true,
			ProviderURL:       ko.String("security.oidc.provider_url"),
			ClientID:          ko.String("security.oidc.client_id"),
			ClientSecret:      ko.String("security.oidc.client_secret"),
			AutoCreateUsers:   ko.Bool("security.oidc.auto_create_users"),
			DefaultUserRoleID: ko.Int("security.oidc.default_user_role_id"),
			DefaultListRoleID: ko.Int("security.oidc.default_list_role_id"),
			RedirectURL:       fmt.Sprintf("%s/auth/oidc", strings.TrimRight(ko.String("app.root_url"), "/")),
		}
	}

	// Setup the sessio manager callbacks for getting and setting cookies.
	cb := &auth.Callbacks{
		GetCookie: func(name string, r any) (*http.Cookie, error) {
			c := r.(echo.Context)
			cookie, err := c.Cookie(name)
			return cookie, err
		},
		SetCookie: func(cookie *http.Cookie, w any) error {
			c := w.(echo.Context)
			cookie.SameSite = http.SameSiteLaxMode
			c.SetCookie(cookie)
			return nil
		},
		GetUser: func(id int) (auth.User, error) {
			return co.GetUser(id, "", "")
		},
	}

	// Initiaize the auth module.
	a, err := auth.New(auth.Config{OIDC: oidcCfg}, db, cb, lo)
	if err != nil {
		lo.Fatalf("error initializing auth: %v", err)
	}

	// Cache all API users in-memory for token auth.
	hasUsers, err := cacheUsers(co, a)
	if err != nil {
		lo.Fatalf("error loading API users to cache: %v", err)
	}

	// If the legacy username+password is set in the TOML file, use that as an API
	// access token in the auth module to preserve backwards compatibility for existing
	// API integrations. The presence of these values show a red banner on the admin UI
	// prompting the creation of new API credentials and the removal of values from
	// the TOML config.
	var (
		username = ko.String("app.admin_username")
		password = ko.String("app.admin_password")
	)
	if len(username) > 2 && len(password) > 6 {
		u := auth.User{
			Username:      username,
			Password:      null.String{Valid: true, String: password},
			PasswordLogin: true,
			HasPassword:   true,
			Status:        auth.UserStatusEnabled,
			Type:          auth.UserTypeAPI,
		}
		u.UserRole.ID = auth.SuperAdminRoleID
		a.CacheAPIUser(u)

		lo.Println(`WARNING: Remove the admin_username and admin_password fields from the TOML configuration file. If you are using APIs, create and use new credentials. Users are now managed via the Admin -> Settings -> Users dashboard.`)
	}

	return hasUsers, a
}

// joinFSPaths joins the given paths with the root path and returns the full paths.
func joinFSPaths(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		// real_path:stuffbin_alias
		f := strings.Split(p, ":")

		out = append(out, path.Join(root, f[0])+":"+f[1])
	}

	return out
}
