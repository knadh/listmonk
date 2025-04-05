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
func (h *Handlers) GetUserRoles(c echo.Context) error {
	// Get all roles.
	out, err := h.app.core.GetRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GeListRoles retrieves roles.
func (h *Handlers) GeListRoles(c echo.Context) error {
	// Get all roles.
	out, err := h.app.core.GetListRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUserRole handles role creation.
func (h *Handlers) CreateUserRole(c echo.Context) error {
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := h.validateUserRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := h.app.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateListRole handles role creation.
func (h *Handlers) CreateListRole(c echo.Context) error {
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := h.validateListRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := h.app.core.CreateListRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateUserRole handles role modification.
func (h *Handlers) UpdateUserRole(c echo.Context) error {
	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := h.validateUserRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := h.app.core.UpdateUserRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateListRole handles role modification.
func (h *Handlers) UpdateListRole(c echo.Context) error {
	// ID 1 is reserved for the Super Admin role and anything below that is invalid.
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := h.validateListRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := h.app.core.UpdateListRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteRole handles role deletion.
func (h *Handlers) DeleteRole(c echo.Context) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Delete the role from the DB.
	if err := h.app.core.DeleteRole(int(id)); err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (h *Handlers) validateUserRole(r auth.Role) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := h.app.constants.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("permission: %s", p)))
		}
	}

	return nil
}

func (h *Handlers) validateListRole(r auth.ListRole) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
