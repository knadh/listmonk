package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo"
)

type settings struct {
	AppRootURL       string   `json:"app.root_url"`
	AppLogoURL       string   `json:"app.logo_url"`
	AppFaviconURL    string   `json:"app.favicon_url"`
	AppFromEmail     string   `json:"app.from_email"`
	AppNotifyEmails  []string `json:"app.notify_emails"`
	AppBatchSize     int      `json:"app.batch_size"`
	AppConcurrency   int      `json:"app.concurrency"`
	AppMaxSendErrors int      `json:"app.max_send_errors"`
	AppMessageRate   int      `json:"app.message_rate"`

	Messengers []interface{} `json:"messengers"`

	PrivacyUnsubHeader    bool     `json:"privacy.unsubscribe_header"`
	PrivacyAllowBlocklist bool     `json:"privacy.allow_blocklist"`
	PrivacyAllowExport    bool     `json:"privacy.allow_export"`
	PrivacyAllowWipe      bool     `json:"privacy.allow_wipe"`
	PrivacyExportable     []string `json:"privacy.exportable"`

	SMTP []struct {
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

	UploadProvider string `json:"upload.provider"`

	UploadFilesystemUploadPath string `json:"upload.filesystem.upload_path"`
	UploadFilesystemUploadURI  string `json:"upload.filesystem.upload_uri"`

	UploadS3AwsAccessKeyID     string `json:"upload.s3.aws_access_key_id"`
	UploadS3AwsDefaultRegion   string `json:"upload.s3.aws_default_region"`
	UploadS3AwsSecretAccessKey string `json:"upload.s3.aws_secret_access_key,omitempty"`
	UploadS3Bucket             string `json:"upload.s3.bucket"`
	UploadS3BucketDomain       string `json:"upload.s3.bucket_domain"`
	UploadS3BucketPath         string `json:"upload.s3.bucket_path"`
	UploadS3BucketType         string `json:"upload.s3.bucket_type"`
	UploadS3Expiry             string `json:"upload.s3.expiry"`
}

// handleGetSettings returns settings from the DB.
func handleGetSettings(c echo.Context) error {
	app := c.Get("app").(*App)

	s, err := getSettings(app)
	if err != nil {
		return err
	}

	// Empty out passwords.
	for i := 0; i < len(s.SMTP); i++ {
		s.SMTP[i].Password = ""
	}
	s.UploadS3AwsSecretAccessKey = ""

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
	cur, err := getSettings(app)
	if err != nil {
		return err
	}

	// There should be at least one SMTP block that's enabled.
	has := false
	for i, s := range set.SMTP {
		if s.Enabled {
			has = true
		}

		// If there's no password coming in from the frontend, attempt to get the
		// last saved password for the SMTP block at the same position.
		if set.SMTP[i].Password == "" {
			if len(cur.SMTP) > i &&
				set.SMTP[i].Host == cur.SMTP[i].Host &&
				set.SMTP[i].Username == cur.SMTP[i].Username {
				set.SMTP[i].Password = cur.SMTP[i].Password
			}
		}
	}
	if !has {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Minimum one SMTP block should be enabled.")
	}

	// S3 password?
	if set.UploadS3AwsSecretAccessKey == "" {
		set.UploadS3AwsSecretAccessKey = cur.UploadS3AwsSecretAccessKey
	}

	// Marshal settings.
	b, err := json.Marshal(set)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error encoding settings: %v", err))
	}

	// Update the settings in the DB.
	if _, err := app.queries.UpdateSettings.Exec(b); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error updating settings: %s", pqErrMsg(err)))
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

func getSettings(app *App) (settings, error) {
	var (
		b   types.JSONText
		out settings
	)

	if err := app.queries.GetSettings.Get(&b); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching settings: %s", pqErrMsg(err)))
	}

	// Unmarshall the settings and filter out sensitive fields.
	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error parsing settings: %v", err))
	}

	return out, nil
}
