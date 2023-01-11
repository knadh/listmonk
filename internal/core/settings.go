package core

import (
	"context"
	"encoding/json"
	"net/http"

	// "github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetSettings returns settings from the DB.
func (c *Core) GetSettings(ctx context.Context) (models.Settings, error) {
	var (
		b   types.JSONText
		out models.Settings
	)

	// xyz, _ := sqlx.PreparexContext(ctx, c.db, `SELECT JSON_OBJECT_AGG(key, value) AS settings
	// FROM (
	//     SELECT * FROM settings ORDER BY key
	// ) t;`)
	if err := c.q.GetSettings.GetContext(ctx, &b); err != nil {
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
func (c *Core) UpdateSettings(ctx context.Context, s models.Settings) error {
	// Marshal settings.
	b, err := json.Marshal(s)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	// Update the settings in the DB.
	if _, err := c.q.UpdateSettings.ExecContext(ctx, b); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	return nil
}
