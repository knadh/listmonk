package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleGetRoles retrieves roles.
func handleGetRoles(c echo.Context) error {
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

// handleCreateRole handles role creation.
func handleCreateRole(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		r   = models.Role{}
	)

	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateRole(r, app); err != nil {
		return err
	}

	out, err := app.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateRole handles role modification.
func handleUpdateRole(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r models.Role
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateRole(r, app); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	out, err := app.core.UpdateRole(id, r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteRole handles role deletion.
func handleDeleteRole(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if err := app.core.DeleteRole(int(id)); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func validateRole(r models.Role, app *App) error {
	// Validate fields.
	if !strHasLen(r.Name.String, 2, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := app.constants.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "permission"))
		}
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != "list:get" && p != "list:manage" {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "list permissions"))
			}
		}
	}

	return nil
}
