package main

import (
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// HandleGetSQLSnippets handles the retrieval of SQL snippets.
func (a *App) HandleGetSQLSnippets(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	var (
		pg       = a.pg.NewFromURL(c.Request().URL.Query())
		id, _    = strconv.Atoi(c.Param("id"))
		name     = c.QueryParam("name")
		isActive *bool
	)

	if v := c.QueryParam("is_active"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			isActive = &val
		}
	}

	// Single snippet by ID.
	if id > 0 {
		out, err := a.core.GetSQLSnippet(id, "")
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Multiple snippets.
	limit := pg.Limit
	if limit == 0 {
		limit = 50 // Default limit
	}

	out, err := a.core.GetSQLSnippets(0, name, isActive, pg.Offset, limit)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// HandleCreateSQLSnippet handles the creation of a SQL snippet.
func (a *App) HandleCreateSQLSnippet(c echo.Context) error {
	var s models.SQLSnippet
	if err := c.Bind(&s); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	// Validate the SQL snippet
	if err := a.core.ValidateSQLSnippet(s.QuerySQL); err != nil {
		return err
	}

	out, err := a.core.CreateSQLSnippet(s, user.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, okResp{out})
}

// HandleUpdateSQLSnippet handles the updating of a SQL snippet.
func (a *App) HandleUpdateSQLSnippet(c echo.Context) error {
	var s models.SQLSnippet
	if err := c.Bind(&s); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
	}

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	// Validate the SQL snippet if it's being changed
	if s.QuerySQL != "" {
		if err := a.core.ValidateSQLSnippet(s.QuerySQL); err != nil {
			return err
		}
	}

	out, err := a.core.UpdateSQLSnippet(id, s)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// HandleDeleteSQLSnippet handles the deletion of a SQL snippet.
func (a *App) HandleDeleteSQLSnippet(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
	}

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	if err := a.core.DeleteSQLSnippet(id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// HandleValidateSQLSnippet handles the validation of a SQL snippet.
func (a *App) HandleValidateSQLSnippet(c echo.Context) error {
	var req struct {
		QuerySQL string `json:"query_sql"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	if err := a.core.ValidateSQLSnippet(req.QuerySQL); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{map[string]bool{"valid": true}})
}

// HandleCountSQLSnippet handles counting subscribers that match a SQL snippet.
func (a *App) HandleCountSQLSnippet(c echo.Context) error {
	var req struct {
		QuerySQL string `json:"query_sql"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	// Get total subscriber count (cached)
	totalCount, err := a.core.GetSubscriberCount("", "", "", []int{})
	if err != nil {
		return err
	}

	// Get count for the SQL snippet (if query is provided)
	matchedCount := 0
	if req.QuerySQL != "" {
		matchedCount, err = a.core.GetSubscriberCount("", req.QuerySQL, "", []int{})
		if err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{map[string]int{
		"total":   totalCount,
		"matched": matchedCount,
	}})
}
