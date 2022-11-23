package core

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
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
