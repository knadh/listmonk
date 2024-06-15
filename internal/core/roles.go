package core

import (
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetRoles retrieves all roles.
func (c *Core) GetRoles() ([]models.Role, error) {
	out := []models.Role{}
	if err := c.q.GetRoles.Select(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{users.roles}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateRole creates a new role.
func (c *Core) CreateRole(r models.Role) (models.Role, error) {
	var out models.Role

	if err := c.q.CreateRole.Get(&out, r.Name, pq.Array(r.Permissions)); err != nil {
		return models.Role{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// UpdateRole updates a given role.
func (c *Core) UpdateRole(id int, r models.Role) (models.Role, error) {
	var out models.Role

	if err := c.q.UpdateRole.Get(&out, id, r.Name, pq.Array(r.Permissions)); err != nil {
		return models.Role{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	if out.ID == 0 {
		return models.Role{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{users.role}"))
	}

	return out, nil
}

// DeleteRole deletes a given role.
func (c *Core) DeleteRole(id int) error {
	if _, err := c.q.DeleteRole.Exec(id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return nil
}
