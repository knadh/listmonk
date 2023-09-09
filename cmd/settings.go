package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const pwdMask = "•"

type aboutHost struct {
	OS       string `json:"os"`
	Machine  string `json:"arch"`
	Hostname string `json:"hostname"`
}
type aboutSystem struct {
	NumCPU  int    `json:"num_cpu"`
	AllocMB uint64 `json:"memory_alloc_mb"`
	OSMB    uint64 `json:"memory_from_os_mb"`
}
type about struct {
	Version   string         `json:"version"`
	Build     string         `json:"build"`
	GoVersion string         `json:"go_version"`
	GoArch    string         `json:"go_arch"`
	Database  types.JSONText `json:"database"`
	System    aboutSystem    `json:"system"`
	Host      aboutHost      `json:"host"`
}

var (
	reAlphaNum = regexp.MustCompile(`[^a-z0-9\-]`)
)

// handleGetSettings returns settings from the DB.
func handleGetSettings(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := app.core.GetSettings()
	if err != nil {
		return err
	}

	// Empty out passwords.
	for i := 0; i < len(s.SMTP); i++ {
		s.SMTP[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SMTP[i].Password))
	}
	for i := 0; i < len(s.BounceBoxes); i++ {
		s.BounceBoxes[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceBoxes[i].Password))
	}
	for i := 0; i < len(s.Messengers); i++ {
		s.Messengers[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.Messengers[i].Password))
	}
	s.UploadS3AwsSecretAccessKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.UploadS3AwsSecretAccessKey))
	s.SendgridKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SendgridKey))
	s.SecurityCaptchaSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SecurityCaptchaSecret))
	s.BouncePostmark.Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BouncePostmark.Password))

	return c.JSON(http.StatusOK, okResp{s})
}

// handleUpdateSettings returns settings from the DB.
func handleUpdateSettings(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		set models.Settings
	)

	// Unmarshal and marshal the fields once to sanitize the settings blob.
	if err := c.Bind(&set); err != nil {
		return err
	}

	// Get the existing settings.
	cur, err := app.core.GetSettings()
	if err != nil {
		return err
	}

	// There should be at least one SMTP block that's enabled.
	has := false
	for i, s := range set.SMTP {
		if s.Enabled {
			has = true
		}

		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of SMTP blocks with the array in the DB.
		if s.UUID == "" {
			set.SMTP[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		// If there's no password coming in from the frontend, copy the existing
		// password by matching the UUID.
		if s.Password == "" {
			for _, c := range cur.SMTP {
				if s.UUID == c.UUID {
					set.SMTP[i].Password = c.Password
				}
			}
		}
	}
	if !has {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("settings.errorNoSMTP"))
	}

	// Bounce boxes.
	for i, s := range set.BounceBoxes {
		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of blocks with the array in the DB.
		if s.UUID == "" {
			set.BounceBoxes[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		if d, _ := time.ParseDuration(s.ScanInterval); d.Minutes() < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("settings.bounces.invalidScanInterval"))
		}

		// If there's no password coming in from the frontend, copy the existing
		// password by matching the UUID.
		if s.Password == "" {
			for _, c := range cur.BounceBoxes {
				if s.UUID == c.UUID {
					set.BounceBoxes[i].Password = c.Password
				}
			}
		}
	}

	// Validate and sanitize postback Messenger names. Duplicates are disallowed
	// and "email" is a reserved name.
	names := map[string]bool{emailMsgr: true}

	for i, m := range set.Messengers {
		// UUID to keep track of password changes similar to the SMTP logic above.
		if m.UUID == "" {
			set.Messengers[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		if m.Password == "" {
			for _, c := range cur.Messengers {
				if m.UUID == c.UUID {
					set.Messengers[i].Password = c.Password
				}
			}
		}

		name := reAlphaNum.ReplaceAllString(strings.ToLower(m.Name), "")
		if _, ok := names[name]; ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("settings.duplicateMessengerName", "name", name))
		}
		if len(name) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("settings.invalidMessengerName"))
		}

		set.Messengers[i].Name = name
		names[name] = true
	}

	// S3 password?
	if set.UploadS3AwsSecretAccessKey == "" {
		set.UploadS3AwsSecretAccessKey = cur.UploadS3AwsSecretAccessKey
	}
	if set.SendgridKey == "" {
		set.SendgridKey = cur.SendgridKey
	}
	if set.BouncePostmark.Password == "" {
		set.BouncePostmark.Password = cur.BouncePostmark.Password
	}
	if set.SecurityCaptchaSecret == "" {
		set.SecurityCaptchaSecret = cur.SecurityCaptchaSecret
	}

	for n, v := range set.UploadExtensions {
		set.UploadExtensions[n] = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "."))
	}

	// Domain blocklist.
	doms := make([]string, 0)
	for _, d := range set.DomainBlocklist {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainBlocklist = doms

	// Update the settings in the DB.
	if err := app.core.UpdateSettings(set); err != nil {
		return err
	}

	// If there are any active campaigns, don't do an auto reload and
	// warn the user on the frontend.
	if app.manager.HasRunningCampaigns() {
		app.Lock()
		app.needsRestart = true
		app.Unlock()

		return c.JSON(http.StatusOK, okResp{struct {
			NeedsRestart bool `json:"needs_restart"`
		}{true}})
	}

	// No running campaigns. Reload the app.
	go func() {
		<-time.After(time.Millisecond * 500)
		app.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetLogs returns the log entries stored in the log buffer.
func handleGetLogs(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{app.bufLog.Lines()})
}

// handleTestSMTPSettings returns the log entries stored in the log buffer.
func handleTestSMTPSettings(c echo.Context) error {
	app := c.Get("app").(*App)

	// Copy the raw JSON post body.
	reqBody, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		app.log.Printf("error reading SMTP test: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.internalError"))
	}

	// Load the JSON into koanf to parse SMTP settings properly including timestrings.
	ko := koanf.New(".")
	if err := ko.Load(rawbytes.Provider(reqBody), json.Parser()); err != nil {
		app.log.Printf("error unmarshalling SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.internalError"))
	}

	req := email.Server{}
	if err := ko.UnmarshalWithConf("", &req, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		app.log.Printf("error scanning SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.internalError"))
	}

	to := ko.String("email")
	if to == "" {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.missingFields", "name", "email"))
	}

	// Initialize a new SMTP pool.
	req.MaxConns = 1
	req.IdleTimeout = time.Second * 2
	req.PoolWaitTimeout = time.Second * 2
	msgr, err := email.New(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.errorCreating", "name", "SMTP", "error", err.Error()))
	}

	var b bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&b, "smtp-test", nil); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", "smtp-test", err)
		return err
	}

	m := models.Message{}
	m.ContentType = app.notifTpls.contentType
	m.From = app.constants.FromEmail
	m.To = []string{to}
	m.Subject = app.i18n.T("settings.smtp.testConnection")
	m.Body = b.Bytes()
	if err := msgr.Push(m); err != nil {
		app.log.Printf("error sending SMTP test (%s): %v", m.Subject, err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{app.bufLog.Lines()})
}

func handleGetAboutInfo(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		mem runtime.MemStats
	)

	runtime.ReadMemStats(&mem)

	out := app.about
	out.System.AllocMB = mem.Alloc / 1024 / 1024
	out.System.OSMB = mem.Sys / 1024 / 1024

	return c.JSON(http.StatusOK, out)
}
