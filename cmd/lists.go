package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetLists retrieves lists with additional metadata like subscriber counts.
func (a *App) GetLists(c echo.Context) error {
	var (
		user = auth.GetUser(c)
		pg   = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Get the list IDs (or blanket permission) the user has access to.
	hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeGet)

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	minimal, _ := strconv.ParseBool(c.FormValue("minimal"))
	if minimal {
		res, err := a.core.GetLists("", hasAllPerm, permittedIDs)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		// Meta.
		total := len(res)
		out := models.PageResults{
			Results: res,
			Total:   total,
			Page:    1,
			PerPage: total,
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Full list query.
	var (
		query   = strings.TrimSpace(c.FormValue("query"))
		tags    = c.QueryParams()["tag"]
		orderBy = c.FormValue("order_by")
		typ     = c.FormValue("type")
		optin   = c.FormValue("optin")
		order   = c.FormValue("order")
	)
	res, total, err := a.core.QueryLists(query, typ, optin, tags, orderBy, order, hasAllPerm, permittedIDs, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetList retrieves a single list by id.
// It's permission checked by the listPerm middleware.
func (a *App) GetList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Check if the user has access to the list.
	if err := user.HasListPerm(auth.PermTypeGet, id); err != nil {
		return err
	}

	// Get the list from the DB.
	out, err := a.core.GetList(id, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateList handles list creation.
func (a *App) CreateList(c echo.Context) error {
	l := models.List{}
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	out, err := a.core.CreateList(l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateList handles list modification.
// It's permission checked by the listPerm middleware.
func (a *App) UpdateList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Check if the user has access to the list.
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	// Update the list in the DB.
	out, err := a.core.UpdateList(id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteLists handles list deletion, either a single one (ID in the URI), or a list.
// It's permission checked by the listPerm middleware.
func (a *App) DeleteLists(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var (
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)
	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	// Check if the user has access to the list.
	if err := user.HasListPerm(auth.PermTypeManage, ids...); err != nil {
		return err
	}

	// Delete the lists from the DB.
	if err := a.core.DeleteLists(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
