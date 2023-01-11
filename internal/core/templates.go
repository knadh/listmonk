package core

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetTemplates retrieves all templates.
func (c *Core) GetTemplates(ctx context.Context, status string, noBody bool) ([]models.Template, error) {
	out := []models.Template{}
	if err := c.q.GetTemplates.SelectContext(ctx, &out, 0, noBody, status); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetTemplate retrieves a given template.
func (c *Core) GetTemplate(ctx context.Context, id int, noBody bool) (models.Template, error) {
	var out []models.Template
	if err := c.q.GetTemplates.SelectContext(ctx, &out, id, noBody, ""); err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Template{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	return out[0], nil
}

// CreateTemplate creates a new template.
func (c *Core) CreateTemplate(ctx context.Context, name, typ, subject string, body []byte) (models.Template, error) {
	var newID int
	if err := c.q.CreateTemplate.GetContext(ctx, &newID, name, typ, subject, body); err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return c.GetTemplate(ctx, newID, false)
}

// UpdateTemplate updates a given template.
func (c *Core) UpdateTemplate(ctx context.Context, id int, name, subject string, body []byte) (models.Template, error) {
	res, err := c.q.UpdateTemplate.ExecContext(ctx, id, name, subject, body)
	if err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.Template{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	return c.GetTemplate(ctx, id, false)
}

// SetDefaultTemplate sets a template as default.
func (c *Core) SetDefaultTemplate(ctx context.Context, id int) error {
	if _, err := c.q.SetDefaultTemplate.ExecContext(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteTemplate deletes a given template.
func (c *Core) DeleteTemplate(ctx context.Context, id int) error {
	var delID int
	if err := c.q.DeleteTemplate.GetContext(ctx, &delID, id); err != nil && err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}
	if delID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("templates.cantDeleteDefault"))
	}

	return nil
}
