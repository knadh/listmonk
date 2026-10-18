package core

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
)

// GetSettings returns settings from the DB.
func (c *Core) GetSettings() (models.Settings, error) {
	var (
		b   types.JSONText
		out models.Settings
	)

	if err := c.q.GetSettings.Get(&b); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	// Unmarshal the settings and filter out sensitive fields.
	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	return out, nil
}

// UpdateSettings updates settings.
func (c *Core) UpdateSettings(s models.Settings) error {
	// Persist empty collections as arrays, not JSON null.
	s.AppFromEmail = utils.NilToSlice(s.AppFromEmail)
	s.AppNotifyEmails = utils.NilToSlice(s.AppNotifyEmails)
	s.PrivacyExportable = utils.NilToSlice(s.PrivacyExportable)
	s.DomainBlocklist = utils.NilToSlice(s.DomainBlocklist)
	s.DomainAllowlist = utils.NilToSlice(s.DomainAllowlist)
	s.SecurityTrustedURLs = utils.NilToSlice(s.SecurityTrustedURLs)
	s.UploadExtensions = utils.NilToSlice(s.UploadExtensions)
	s.SMTP = utils.NilToSlice(s.SMTP)
	s.Messengers = utils.NilToSlice(s.Messengers)
	s.BounceBoxes = utils.NilToSlice(s.BounceBoxes)

	// Marshal settings.
	b, err := json.Marshal(s)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	// Update the settings in the DB.
	if _, err := c.q.UpdateSettings.Exec(b); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	return nil
}

// UpdateSettingsByKey updates a single setting by key.
func (c *Core) UpdateSettingsByKey(key string, value json.RawMessage) error {
	switch key {
	case "app.from_email",
		"app.notify_emails",
		"privacy.exportable",
		"privacy.domain_blocklist",
		"privacy.domain_allowlist",
		"security.trusted_urls",
		"upload.extensions",
		"smtp",
		"messengers",
		"bounce.mailboxes":
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			value = json.RawMessage("[]")
		}
	}

	if _, err := c.q.UpdateSettingsByKey.Exec(key, value); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	return nil
}
