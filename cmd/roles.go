package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v4"
)

// GetUserRoles retrieves roles.
func (a *App) GetUserRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GeListRoles retrieves roles.
func (a *App) GeListRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetListRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUserRole handles role creation.
func (a *App) CreateUserRole(c echo.Context) error {
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateListRole handles role creation.
func (a *App) CreateListRole(c echo.Context) error {
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateListRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateUserRole handles role modification.
func (a *App) UpdateUserRole(c echo.Context) error {
	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateUserRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateListRole handles role modification.
func (a *App) UpdateListRole(c echo.Context) error {
	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateListRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteRole handles role deletion.
func (a *App) DeleteRole(c echo.Context) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the role from the DB.
	if err := a.core.DeleteRole(int(id)); err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateUserRole(r auth.Role) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := a.constants.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("permission: %s", p)))
		}
	}

	return nil
}

func (a *App) validateListRole(r auth.ListRole) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
