package main

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gdgvda/cron"
	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/notifs"
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

// GetSettings returns settings from the DB merged with external config/env values.
func (a *App) GetSettings(c echo.Context) error {
	s, err := a.core.GetSettings()
	if err != nil {
		return err
	}

	// Merge external settings values into the DB settings
	a.mergeExternalSettings(&s)

	// Empty out passwords.
	for i := range s.SMTP {
		s.SMTP[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SMTP[i].Password))
	}
	for i := range s.BounceBoxes {
		s.BounceBoxes[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceBoxes[i].Password))
	}
	for i := range s.Messengers {
		s.Messengers[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.Messengers[i].Password))
	}

	s.UploadS3AwsSecretAccessKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.UploadS3AwsSecretAccessKey))
	s.SendgridKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SendgridKey))
	s.BouncePostmark.Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BouncePostmark.Password))
	s.BounceForwardEmail.Key = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceForwardEmail.Key))
	s.SecurityCaptchaSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SecurityCaptchaSecret))
	s.OIDC.ClientSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.OIDC.ClientSecret))

	// Include external settings metadata in the response
	response := map[string]any{
		"data": s,
		"external_settings": func() []string {
			keys := make([]string, 0, len(a.externalSettings))
			for k := range a.externalSettings {
				keys = append(keys, k)
			}
			return keys
		}(),
	}

	return c.JSON(http.StatusOK, okResp{response})
}

// IsSettingExternallyManaged checks if a specific setting key is configured externally (config file or environment variable).
func (a *App) IsSettingExternallyManaged(key string) bool {
	_, exists := a.externalSettings[key]
	return exists
}

// GetExternallyManagedSettings returns a map of all externally managed settings.
func (a *App) GetExternallyManagedSettings() map[string]any {
	return a.externalSettings
}

// mergeExternalSettings merges external config/env values into the settings struct.
func (a *App) mergeExternalSettings(s *models.Settings) {
	// Merge external settings values into the DB settings struct
	for key, value := range a.externalSettings {
		switch key {
		case "app.site_name":
			if v, ok := value.(string); ok {
				s.AppSiteName = v
			}
		case "app.root_url":
			if v, ok := value.(string); ok {
				s.AppRootURL = v
			}
		case "app.logo_url":
			if v, ok := value.(string); ok {
				s.AppLogoURL = v
			}
		case "app.favicon_url":
			if v, ok := value.(string); ok {
				s.AppFaviconURL = v
			}
		case "app.from_email":
			if v, ok := value.(string); ok {
				s.AppFromEmail = v
			}
		case "app.notify_emails":
			if v, ok := value.([]any); ok {
				emails := make([]string, 0, len(v))
				for _, email := range v {
					if str, ok := email.(string); ok {
						emails = append(emails, str)
					}
				}
				s.AppNotifyEmails = emails
			}
		case "app.enable_public_subscription_page":
			if v, ok := value.(bool); ok {
				s.EnablePublicSubPage = v
			}
		case "app.enable_public_archive":
			if v, ok := value.(bool); ok {
				s.EnablePublicArchive = v
			}
		case "app.enable_public_archive_rss_content":
			if v, ok := value.(bool); ok {
				s.EnablePublicArchiveRSSContent = v
			}
		case "app.send_optin_confirmation":
			if v, ok := value.(bool); ok {
				s.SendOptinConfirmation = v
			}
		case "app.check_updates":
			if v, ok := value.(bool); ok {
				s.CheckUpdates = v
			}
		case "app.lang":
			if v, ok := value.(string); ok {
				s.AppLang = v
			}
		case "app.batch_size":
			if v, ok := value.(int); ok {
				s.AppBatchSize = v
			}
		case "app.concurrency":
			if v, ok := value.(int); ok {
				s.AppConcurrency = v
			}
		case "app.max_send_errors":
			if v, ok := value.(int); ok {
				s.AppMaxSendErrors = v
			}
		case "app.message_rate":
			if v, ok := value.(int); ok {
				s.AppMessageRate = v
			}
		case "app.cache_slow_queries":
			if v, ok := value.(bool); ok {
				s.CacheSlowQueries = v
			}
		case "app.cache_slow_queries_interval":
			if v, ok := value.(string); ok {
				s.CacheSlowQueriesInterval = v
			}
		case "app.message_sliding_window":
			if v, ok := value.(bool); ok {
				s.AppMessageSlidingWindow = v
			}
		case "app.message_sliding_window_duration":
			if v, ok := value.(string); ok {
				s.AppMessageSlidingWindowDuration = v
			}
		case "app.message_sliding_window_rate":
			if v, ok := value.(int); ok {
				s.AppMessageSlidingWindowRate = v
			}
		case "privacy.individual_tracking":
			if v, ok := value.(bool); ok {
				s.PrivacyIndividualTracking = v
			}
		case "privacy.unsubscribe_header":
			if v, ok := value.(bool); ok {
				s.PrivacyUnsubHeader = v
			}
		case "privacy.allow_blocklist":
			if v, ok := value.(bool); ok {
				s.PrivacyAllowBlocklist = v
			}
		case "privacy.allow_preferences":
			if v, ok := value.(bool); ok {
				s.PrivacyAllowPreferences = v
			}
		case "privacy.allow_export":
			if v, ok := value.(bool); ok {
				s.PrivacyAllowExport = v
			}
		case "privacy.allow_wipe":
			if v, ok := value.(bool); ok {
				s.PrivacyAllowWipe = v
			}
		case "privacy.exportable":
			if v, ok := value.([]any); ok {
				exportable := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						exportable = append(exportable, str)
					}
				}
				s.PrivacyExportable = exportable
			}
		case "privacy.record_optin_ip":
			if v, ok := value.(bool); ok {
				s.PrivacyRecordOptinIP = v
			}
		case "privacy.domain_blocklist":
			if v, ok := value.([]any); ok {
				domains := make([]string, 0, len(v))
				for _, domain := range v {
					if str, ok := domain.(string); ok {
						domains = append(domains, str)
					}
				}
				s.DomainBlocklist = domains
			}
		case "privacy.domain_allowlist":
			if v, ok := value.([]any); ok {
				domains := make([]string, 0, len(v))
				for _, domain := range v {
					if str, ok := domain.(string); ok {
						domains = append(domains, str)
					}
				}
				s.DomainAllowlist = domains
			}
		case "security.enable_captcha":
			if v, ok := value.(bool); ok {
				s.SecurityEnableCaptcha = v
			}
		case "security.captcha_key":
			if v, ok := value.(string); ok {
				s.SecurityCaptchaKey = v
			}
		case "security.captcha_secret":
			if v, ok := value.(string); ok {
				s.SecurityCaptchaSecret = v
			}
		case "security.oidc":
			if oidcMap, ok := value.(map[string]any); ok {
				if enabled, ok := oidcMap["enabled"].(bool); ok {
					s.OIDC.Enabled = enabled
				}
				if providerURL, ok := oidcMap["provider_url"].(string); ok {
					s.OIDC.ProviderURL = providerURL
				}
				if providerName, ok := oidcMap["provider_name"].(string); ok {
					s.OIDC.ProviderName = providerName
				}
				if clientID, ok := oidcMap["client_id"].(string); ok {
					s.OIDC.ClientID = clientID
				}
				if clientSecret, ok := oidcMap["client_secret"].(string); ok {
					s.OIDC.ClientSecret = clientSecret
				}
			}
		}
	}
}

// GetSettingsState returns the state of settings - which ones are configured externally vs database.
func (a *App) GetSettingsState(c echo.Context) error {
	// Create response with external settings state
	state := map[string]any{
		"external_settings": a.externalSettings,
		"external_keys":     func() []string {
			keys := make([]string, 0, len(a.externalSettings))
			for k := range a.externalSettings {
				keys = append(keys, k)
			}
			return keys
		}(),
	}

	return c.JSON(http.StatusOK, okResp{state})
}

// validateAndRestoreExternalSettings checks if any externally managed settings are being changed
// and restores them to their external values. Returns a list of blocked field names.
func (a *App) validateAndRestoreExternalSettings(incoming *models.Settings, current *models.Settings) []string {
	blockedFields := []string{}
	
	// Check each externally managed setting and restore it if it was modified
	for key := range a.externalSettings {
		// For now, we'll implement a simplified version that checks common fields
		// A more comprehensive version would use reflection to check all fields
		switch key {
		case "app.root_url":
			if incoming.AppRootURL != current.AppRootURL {
				incoming.AppRootURL = current.AppRootURL
				blockedFields = append(blockedFields, "App Root URL")
			}
		case "app.site_name":
			if incoming.AppSiteName != current.AppSiteName {
				incoming.AppSiteName = current.AppSiteName
				blockedFields = append(blockedFields, "Site Name")
			}
		case "app.from_email":
			if incoming.AppFromEmail != current.AppFromEmail {
				incoming.AppFromEmail = current.AppFromEmail
				blockedFields = append(blockedFields, "From Email")
			}
		case "app.logo_url":
			if incoming.AppLogoURL != current.AppLogoURL {
				incoming.AppLogoURL = current.AppLogoURL
				blockedFields = append(blockedFields, "Logo URL")
			}
		case "app.favicon_url":
			if incoming.AppFaviconURL != current.AppFaviconURL {
				incoming.AppFaviconURL = current.AppFaviconURL
				blockedFields = append(blockedFields, "Favicon URL")
			}
		// Add more field checks as needed...
		// For complex fields like SMTP, messengers, etc., we'd need more sophisticated checking
		}
	}
	
	return blockedFields
}

// UpdateSettings returns settings from the DB.
func (a *App) UpdateSettings(c echo.Context) error {
	// Unmarshal and marshal the fields once to sanitize the settings blob.
	var set models.Settings
	if err := c.Bind(&set); err != nil {
		return err
	}

	// Get the existing settings.
	cur, err := a.core.GetSettings()
	if err != nil {
		return err
	}

	// Validate and sanitize postback Messenger names along with SMTP names
	// (where each SMTP is also considered as a standalone messenger).
	// Duplicates are disallowed and "email" is a reserved name.
	names := map[string]bool{emailMsgr: true}

	// There should be at least one SMTP block that's enabled.
	has := false
	for i, s := range set.SMTP {
		if s.Enabled {
			has = true
		}

		// Sanitize and normalize the SMTP server name.
		name := reAlphaNum.ReplaceAllString(strings.ToLower(strings.TrimSpace(s.Name)), "-")
		if name != "" {
			if !strings.HasPrefix(name, "email-") {
				name = "email-" + name
			}

			if _, ok := names[name]; ok {
				return echo.NewHTTPError(http.StatusBadRequest,
					a.i18n.Ts("settings.duplicateMessengerName", "name", name))
			}

			names[name] = true
		}
		set.SMTP[i].Name = name

		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of SMTP blocks with the array in the DB.
		if s.UUID == "" {
			set.SMTP[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		// Ensure the HOST is trimmed of any whitespace.
		// This is a common mistake when copy-pasting SMTP settings.
		set.SMTP[i].Host = strings.TrimSpace(s.Host)

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
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.errorNoSMTP"))
	}

	// Always remove the trailing slash from the app root URL.
	set.AppRootURL = strings.TrimRight(set.AppRootURL, "/")

	// Bounce boxes.
	for i, s := range set.BounceBoxes {
		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of blocks with the array in the DB.
		if s.UUID == "" {
			set.BounceBoxes[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		// Ensure the HOST is trimmed of any whitespace.
		// This is a common mistake when copy-pasting SMTP settings.
		set.BounceBoxes[i].Host = strings.TrimSpace(s.Host)

		if d, _ := time.ParseDuration(s.ScanInterval); d.Minutes() < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.bounces.invalidScanInterval"))
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
				a.i18n.Ts("settings.duplicateMessengerName", "name", name))
		}
		if len(name) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.invalidMessengerName"))
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
	if set.BounceForwardEmail.Key == "" {
		set.BounceForwardEmail.Key = cur.BounceForwardEmail.Key
	}
	if set.SecurityCaptchaSecret == "" {
		set.SecurityCaptchaSecret = cur.SecurityCaptchaSecret
	}
	if set.OIDC.ClientSecret == "" {
		set.OIDC.ClientSecret = cur.OIDC.ClientSecret
	}

	for n, v := range set.UploadExtensions {
		set.UploadExtensions[n] = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "."))
	}

	// Domain blocklist / allowlist.
	doms := make([]string, 0, len(set.DomainBlocklist))
	for _, d := range set.DomainBlocklist {
		if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainBlocklist = doms

	doms = make([]string, 0, len(set.DomainAllowlist))
	for _, d := range set.DomainAllowlist {
		if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainAllowlist = doms

	// Validate slow query caching cron.
	if set.CacheSlowQueries {
		if _, err := cron.ParseStandard(set.CacheSlowQueriesInterval); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidData")+": slow query cron: "+err.Error())
		}
	}

	// Prevent updates to externally managed settings.
	// We need to restore the external values and warn about blocked fields.
	blockedFields := a.validateAndRestoreExternalSettings(&set, &cur)
	if len(blockedFields) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, 
			"Cannot update externally configured settings: "+strings.Join(blockedFields, ", "))
	}

	// Update the settings in the DB.
	if err := a.core.UpdateSettings(set); err != nil {
		return err
	}

	// If there are any active campaigns, don't do an auto reload and
	// warn the user on the frontend.
	if a.manager.HasRunningCampaigns() {
		a.Lock()
		a.needsRestart = true
		a.Unlock()

		return c.JSON(http.StatusOK, okResp{struct {
			NeedsRestart bool `json:"needs_restart"`
		}{true}})
	}

	// No running campaigns. Reload the a.
	go func() {
		<-time.After(time.Millisecond * 500)
		a.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}

// GetLogs returns the log entries stored in the log buffer.
func (a *App) GetLogs(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{a.bufLog.Lines()})
}

// TestSMTPSettings returns the log entries stored in the log buffer.
func (a *App) TestSMTPSettings(c echo.Context) error {
	// Copy the raw JSON post body.
	reqBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading SMTP test: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	// Load the JSON into koanf to parse SMTP settings properly including timestrings.
	ko := koanf.New(".")
	if err := ko.Load(rawbytes.Provider(reqBody), json.Parser()); err != nil {
		a.log.Printf("error unmarshalling SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	req := email.Server{}
	if err := ko.UnmarshalWithConf("", &req, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		a.log.Printf("error scanning SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	to := ko.String("email")
	if to == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.missingFields", "name", "email"))
	}

	// Initialize a new SMTP pool.
	req.MaxConns = 1
	req.IdleTimeout = time.Second * 2
	req.PoolWaitTimeout = time.Second * 2
	msgr, err := email.New("", req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorCreating", "name", "SMTP", "error", err.Error()))
	}

	// Render the test email template body.
	var b bytes.Buffer
	if err := notifs.Tpls.ExecuteTemplate(&b, "smtp-test", nil); err != nil {
		a.log.Printf("error compiling notification template '%s': %v", "smtp-test", err)
		return err
	}

	m := models.Message{}
	m.From = a.cfg.FromEmail
	m.To = []string{to}
	m.Subject = a.i18n.T("settings.smtp.testConnection")
	m.Body = b.Bytes()
	if err := msgr.Push(m); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{a.bufLog.Lines()})
}

func (a *App) GetAboutInfo(c echo.Context) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	out := a.about
	out.System.AllocMB = mem.Alloc / 1024 / 1024
	out.System.OSMB = mem.Sys / 1024 / 1024

	return c.JSON(http.StatusOK, out)
}
