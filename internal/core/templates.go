package core

import (
	"database/sql"
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetTemplates retrieves all templates.
func (c *Core) GetTemplates(status string, noBody bool, authID string) ([]models.Template, error) {
	out := []models.Template{}
	if err := c.q.GetTemplates.Select(&out, 0, noBody, status, authID); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetTemplate retrieves a given template.
func (c *Core) GetTemplate(id int, noBody bool, authID string) (models.Template, error) {
	var out []models.Template
	if err := c.q.GetTemplates.Select(&out, id, noBody, "", authID); err != nil {
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
func (c *Core) CreateTemplate(name, typ, subject string, body []byte, authID string) (models.Template, error) {
	var newID int
	if err := c.q.CreateTemplate.Get(&newID, name, typ, subject, body, authID); err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return c.GetTemplate(newID, false, authID)
}

// UpdateTemplate updates a given template.
func (c *Core) UpdateTemplate(id int, name, subject string, body []byte, authID string) (models.Template, error) {
	res, err := c.q.UpdateTemplate.Exec(id, name, subject, body, authID)
	if err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.Template{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	return c.GetTemplate(id, false, authID)
}

// SetDefaultTemplate sets a template as default.
func (c *Core) SetDefaultTemplate(id int, authID string) error {
	if _, err := c.q.SetDefaultTemplate.Exec(id, authID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteTemplate deletes a given template.
func (c *Core) DeleteTemplate(id int, authID string) error {
	var delID int
	if err := c.q.DeleteTemplate.Get(&delID, id, authID); err != nil && err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}
	if delID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("templates.cantDeleteDefault"))
	}

	return nil
}
