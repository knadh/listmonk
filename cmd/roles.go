package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleGetUserRoles retrieves roles.
func (h *Handler) handleGetUserRoles(c echo.Context) error {

	// Get all roles.
	out, err := h.app.core.GetRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGeListRoles retrieves roles.
func (h *Handler) handleGetListRoles(c echo.Context) error {
	// Get all roles.
	out, err := h.app.core.GetListRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateUserRole handles role creation.
func (h *Handler) handleCreateUserRole(c echo.Context) error {
	var (
		r = models.Role{}
	)

	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateUserRole(r, h.app); err != nil {
		return err
	}

	out, err := h.app.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateListRole handles role creation.
func (h *Handler) handleCreateListRole(c echo.Context) error {
	var (
		r = models.ListRole{}
	)

	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateListRole(r, h.app); err != nil {
		return err
	}

	out, err := h.app.core.CreateListRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateUserRole handles role modification.
func (h *Handler) handleUpdateUserRole(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r models.Role
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateUserRole(r, h.app); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	out, err := h.app.core.UpdateUserRole(id, r)
	if err != nil {
		return err
	}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateListRole handles role modification.
func (h *Handler) handleUpdateListRole(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r models.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := validateListRole(r, h.app); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	out, err := h.app.core.UpdateListRole(id, r)
	if err != nil {
		return err
	}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteRole handles role deletion.
func (h *Handler) handleDeleteRole(c echo.Context) error {
	var (
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if err := h.app.core.DeleteRole(int(id)); err != nil {
		return err
	}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func validateUserRole(r models.Role, app *App) error {
	// Validate fields.
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

func validateListRole(r models.ListRole, app *App) error {
	// Validate fields.
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != "list:get" && p != "list:manage" {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
