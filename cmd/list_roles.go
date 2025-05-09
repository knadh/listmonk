package main

import (
	"net/http"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v4"
)

// AssignListsToRole handles assigning lists to a role.
func (a *App) AssignListsToRole(c echo.Context) error {
	var data struct {
		RoleID int                   `json:"role_id"`
		Lists  []auth.ListPermission `json:"lists"`
	}
	if err := c.Bind(&data); err != nil {
		return err
	}

	// Validate role ID.
	if data.RoleID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "role_id"))
	}

	// Validate list permissions.
	for _, l := range data.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "list permissions"))
			}
		}
	}

	// Assign lists to the role.
	if err := a.core.UpsertListPermissions(data.RoleID, data.Lists); err != nil {
		return err
	}

	// Get the updated role.
	roles, err := a.core.GetListRoles()
	if err != nil {
		return err
	}

	var updatedRole auth.ListRole
	for _, r := range roles {
		if r.ID == data.RoleID {
			updatedRole = r
			break
		}
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{updatedRole})
}
