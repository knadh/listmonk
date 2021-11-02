package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"syscall"
	"time"
	"fmt"
	"html/template"
	"sort"
	"bytes"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo"
	"github.com/knadh/listmonk/models"
)

type settings struct {
	AppRootURL            string   `json:"app.root_url"`
	AppLogoURL            string   `json:"app.logo_url"`
	AppFaviconURL         string   `json:"app.favicon_url"`
	AppFromEmail          string   `json:"app.from_email"`
	AppNotifyEmails       []string `json:"app.notify_emails"`
	EnablePublicSubPage   bool     `json:"app.enable_public_subscription_page"`
	SendOptinConfirmation bool     `json:"app.send_optin_confirmation"`
	CheckUpdates          bool     `json:"app.check_updates"`
	AppLang               string   `json:"app.lang"`

	AppBatchSize     int `json:"app.batch_size"`
	AppConcurrency   int `json:"app.concurrency"`
	AppMaxSendErrors int `json:"app.max_send_errors"`
	AppMessageRate   int `json:"app.message_rate"`

	AppMessageSlidingWindow         bool   `json:"app.message_sliding_window"`
	AppMessageSlidingWindowDuration string `json:"app.message_sliding_window_duration"`
	AppMessageSlidingWindowRate     int    `json:"app.message_sliding_window_rate"`

	PrivacyIndividualTracking bool     `json:"privacy.individual_tracking"`
	PrivacyUnsubHeader        bool     `json:"privacy.unsubscribe_header"`
	PrivacyAllowBlocklist     bool     `json:"privacy.allow_blocklist"`
	PrivacyAllowExport        bool     `json:"privacy.allow_export"`
	PrivacyAllowWipe          bool     `json:"privacy.allow_wipe"`
	PrivacyExportable         []string `json:"privacy.exportable"`
	DomainBlocklist           []string `json:"privacy.domain_blocklist"`

	UploadProvider             string `json:"upload.provider"`
	UploadFilesystemUploadPath string `json:"upload.filesystem.upload_path"`
	UploadFilesystemUploadURI  string `json:"upload.filesystem.upload_uri"`
	UploadS3URL                string `json:"upload.s3.url"`
	UploadS3AwsAccessKeyID     string `json:"upload.s3.aws_access_key_id"`
	UploadS3AwsDefaultRegion   string `json:"upload.s3.aws_default_region"`
	UploadS3AwsSecretAccessKey string `json:"upload.s3.aws_secret_access_key,omitempty"`
	UploadS3Bucket             string `json:"upload.s3.bucket"`
	UploadS3BucketDomain       string `json:"upload.s3.bucket_domain"`
	UploadS3BucketPath         string `json:"upload.s3.bucket_path"`
	UploadS3BucketType         string `json:"upload.s3.bucket_type"`
	UploadS3Expiry             string `json:"upload.s3.expiry"`

	SMTP []struct {
		UUID          string              `json:"uuid"`
		Enabled       bool                `json:"enabled"`
		Host          string              `json:"host"`
		HelloHostname string              `json:"hello_hostname"`
		Port          int                 `json:"port"`
		AuthProtocol  string              `json:"auth_protocol"`
		Username      string              `json:"username"`
		Password      string              `json:"password,omitempty"`
		EmailHeaders  []map[string]string `json:"email_headers"`
		MaxConns      int                 `json:"max_conns"`
		MaxMsgRetries int                 `json:"max_msg_retries"`
		IdleTimeout   string              `json:"idle_timeout"`
		WaitTimeout   string              `json:"wait_timeout"`
		TLSEnabled    bool                `json:"tls_enabled"`
		TLSSkipVerify bool                `json:"tls_skip_verify"`
	} `json:"smtp"`

	Messengers []struct {
		UUID          string `json:"uuid"`
		Enabled       bool   `json:"enabled"`
		Name          string `json:"name"`
		RootURL       string `json:"root_url"`
		Username      string `json:"username"`
		Password      string `json:"password,omitempty"`
		MaxConns      int    `json:"max_conns"`
		Timeout       string `json:"timeout"`
		MaxMsgRetries int    `json:"max_msg_retries"`
	} `json:"messengers"`

	BounceEnabled        bool   `json:"bounce.enabled"`
	BounceEnableWebhooks bool   `json:"bounce.webhooks_enabled"`
	BounceCount          int    `json:"bounce.count"`
	BounceAction         string `json:"bounce.action"`
	SESEnabled           bool   `json:"bounce.ses_enabled"`
	SendgridEnabled      bool   `json:"bounce.sendgrid_enabled"`
	SendgridKey          string `json:"bounce.sendgrid_key"`
	BounceBoxes          []struct {
		UUID          string `json:"uuid"`
		Enabled       bool   `json:"enabled"`
		Type          string `json:"type"`
		Host          string `json:"host"`
		Port          int    `json:"port"`
		AuthProtocol  string `json:"auth_protocol"`
		ReturnPath    string `json:"return_path"`
		Username      string `json:"username"`
		Password      string `json:"password,omitempty"`
		TLSEnabled    bool   `json:"tls_enabled"`
		TLSSkipVerify bool   `json:"tls_skip_verify"`
		ScanInterval  string `json:"scan_interval"`
	} `json:"bounce.mailboxes"`

	AdminCustomCSS					string 				`json:"appearance.admin.custom_css"`
	AdminCustomTemplates			map[string]string 	`json:"appearance.admin.custom_templates"`
	PublicCustomCSS					string 				`json:"appearance.public.custom_css"`
	PublicCustomJS					string 				`json:"appearance.public.custom_js"`
}

type dummyStruct struct {
	models.Subscriber

	OptinURL 		string
	OptinURLAttr 	template.HTMLAttr
	Lists    		[]models.List
	Imported 		int
	Total			int
}

var (
	reAlphaNum = regexp.MustCompile(`[^a-z0-9\-]`)

	dummySub = models.Subscriber{
		Email:   	"demo@listmonk.app",
		Name:    	"Demo Subscriber",
		UUID:    	dummyUUID,
		Attribs: 	models.SubscriberAttribs{"city": "Bengaluru"},
		Status:	 	"enabled",
	}

	dummyList = models.List{
		UUID:            	dummyUUID,
		Name:            	"Demo list",
		Type:            	"public",
		Optin:           	"double",
		SubscriberCount: 	100,
		SubscriberID:    	1,
		SubscriptionStatus: "confirmed",
		Total: 				500,
	}

	dummyData = dummyStruct{
		Subscriber: 	dummySub,
		OptinURL:		"https://demo.url",
		OptinURLAttr:	template.HTMLAttr(fmt.Sprintf(`href="OptinURL"`)),
		Lists:			[]models.List {dummyList},
		Imported:		100,
		Total:			150,
	}

	dummyCampaign = map[string]interface{}{
			"ID":     	1,
			"Name":   	"Demo Campaign",
			"Status": 	"running",
			"Sent":   	1000,
			"ToSend": 	1200,
			"Reason": 	"",
	}
)

// handleGetSettings returns settings from the DB.
func handleGetSettings(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := GetSettings(app)
	if err != nil {
		return err
	}

	// Empty out passwords.
	for i := 0; i < len(s.SMTP); i++ {
		s.SMTP[i].Password = ""
	}
	for i := 0; i < len(s.BounceBoxes); i++ {
		s.BounceBoxes[i].Password = ""
	}
	for i := 0; i < len(s.Messengers); i++ {
		s.Messengers[i].Password = ""
	}
	s.UploadS3AwsSecretAccessKey = ""
	s.SendgridKey = ""

	return c.JSON(http.StatusOK, okResp{s})
}

// handleUpdateSettings returns settings from the DB.
func handleUpdateSettings(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		set settings
	)

	// Unmarshal and marshal the fields once to sanitize the settings blob.
	if err := c.Bind(&set); err != nil {
		return err
	}

	// Get the existing settings.
	cur, err := GetSettings(app)
	if err != nil {
		return err
	}

	// There should be at least one SMTP block that's enabled.
	has := false
	for i, s := range set.SMTP {
		if s.Enabled {
			has = true
		}

		// Assign a UUID. The frontend only sends a password when the user explictly
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
		// Assign a UUID. The frontend only sends a password when the user explictly
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

	// Domain blocklist.
	doms := make([]string, 0)
	for _, d := range set.DomainBlocklist {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainBlocklist = doms

	// Marshal settings.
	b, err := json.Marshal(set)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	// Update the settings in the DB.
	if _, err := app.queries.UpdateSettings.Exec(b); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.settings}", "error", pqErrMsg(err)))
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
		app.sigChan <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetLogs returns the log entries stored in the log buffer.
func handleGetLogs(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{app.bufLog.Lines()})
}

// handleGetAdminCustomCSS returns the Admin custom CSS from the DB.
func handleGetAdminCustomCSS(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := GetSettings(app)
	if err != nil {
		return err
	}

	css := []byte(s.AdminCustomCSS)
	return c.Blob(http.StatusOK, "text/css", css)
}

// handleGetPublicCustomCSS returns the Admin custom CSS from the DB.
func handleGetPublicCustomCSS(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := GetSettings(app)
	if err != nil {
		return err
	}

	css := []byte(s.PublicCustomCSS)
	return c.Blob(http.StatusOK, "text/css", css)
}

// handleGetPublicCustomJS returns the Admin custom CSS from the DB.
func handleGetPublicCustomJS(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := GetSettings(app)
	if err != nil {
		return err
	}

	js := []byte(s.PublicCustomJS)
	return c.Blob(http.StatusOK, "text/javascript", js)
}

func GetSettings(app *App) (settings, error) {
	var (
		b   types.JSONText
		out settings
	)

	if err := app.queries.GetSettings.Get(&b); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	// Unmarshall the settings and filter out sensitive fields.
	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	return out, nil
}

// handleGetAdminCustomTemplate returns the default notification template given the block name.
func handleGetNotifTemplate(c echo.Context) error {
	app := c.Get("app").(*App)
	name := c.Param("name")

	template := GetDefaultTemplate(app, name)
	return c.JSON(http.StatusOK, okResp{string(template)})
}

// handleGetTemplates returns the currently defined templates.
func handleGetDefinedTemplates(c echo.Context) error {
	app := c.Get("app").(*App)

	defined := GetDefinedTemplates(app)
	return c.JSON(http.StatusOK, okResp{defined})
}

// handleGenerateNotifPreview returns a constructed notifications template from the db
func handleGenerateNotifPreview(c echo.Context) error {
	app := c.Get("app").(*App)
	name := c.Param("name")

	//use dummy data to generate preview
	body, err := GenerateEmailTemplate(app, name, dummyData)
	if err != nil {

		//try using dummy campaign data before returning error
		cBody, cErr := GenerateEmailTemplate(app, name, dummyCampaign)
		if cErr != nil {
			app.log.Printf("error generating notification preview '%s': %v", name, cErr)
			return cErr
		}
		return c.Blob(http.StatusOK, "text/html", []byte(cBody))
	}

	return c.Blob(http.StatusOK, "text/html", []byte(body))
}

func GetDefinedTemplates(app *App) []string {
	templs := app.notifTpls.tpls.Templates()
	defined := []string{}
	for _, tmpl := range templs {
		name := tmpl.Name()

		//ignore any .html templates
		if !strings.HasSuffix(name, "html") {
			defined = append(defined, name)
		}
	}

	sort.Strings(defined)
	return defined
}

func GenerateEmailTemplate(app *App, tplName string, data interface{}) ([]byte, error) {
	//get settings
	s, err := GetSettings(app)
	if err != nil {
		return nil, err
	}
	
	//duplicate the default template
	dupTpl, _ := app.notifTpls.tpls.Clone()

	//get all defined template names
	dfltTempls := GetDefinedTemplates(app)

	//check to see if we have any custom templates defined in the Admin Dashboard, then override the default template
	cstmTmplsJSON := map[string]string(s.AdminCustomTemplates)
	for _, name := range dfltTempls {
			val, ok := cstmTmplsJSON[name]
			if ok {
				newTemplate, _ := template.New(name).Parse(val)
				dupTpl.AddParseTree(name, newTemplate.Tree)
			}
	}

	var b bytes.Buffer
	if err := dupTpl.ExecuteTemplate(&b, tplName, data); err != nil {
		app.log.Printf("error generating notification template '%s': %v", tplName, err)
		return nil, err
	}

	return b.Bytes(), nil
}

func GetDefaultTemplate(app *App, tplName string) []byte {
	tmpl := app.notifTpls.tpls.Lookup(tplName)
	return []byte(tmpl.Tree.Root.String())
}
