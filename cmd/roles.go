package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v4"
)

// handleGetUserRoles retrieves roles.
func handleGetUserRoles(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// Get all roles.
	out, err := app.core.GetRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGeListRoles retrieves roles.
func handleGeListRoles(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// Get all roles.
	out, err := app.core.GetListRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateUserRole handles role creation.
func handleCreateUserRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := validateUserRole(r, app); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := app.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateListRole handles role creation.
func handleCreateListRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := validateListRole(r, app); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := app.core.CreateListRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateUserRole handles role modification.
func handleUpdateUserRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := validateUserRole(r, app); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := app.core.UpdateUserRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(app.core, app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateListRole handles role modification.
func handleUpdateListRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateListRole(r, app); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := app.core.UpdateListRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(app.core, app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteRole handles role deletion.
func handleDeleteRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Delete the role from the DB.
	if err := app.core.DeleteRole(int(id)); err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(app.core, app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func validateUserRole(r auth.Role, app *App) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := app.constants.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("permission: %s", p)))
		}
	}

	return nil
}

func validateListRole(r auth.ListRole, app *App) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
