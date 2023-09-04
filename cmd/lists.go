package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleGetLists retrieves lists with additional metadata like subscriber counts. This may be slow.
func handleGetLists(c echo.Context) error {
	app := c.Get("app").(*App)
	pg := app.paginator.NewFromURL(c.Request().URL.Query())

	query := strings.TrimSpace(c.FormValue("query"))
	orderBy := c.FormValue("order_by")
	order := c.FormValue("order")
	minimal, _ := strconv.ParseBool(c.FormValue("minimal"))
	listID, _ := strconv.Atoi(c.Param("id"))

	// Fetch one list.
	if listID > 0 {
		out, err := app.core.GetList(listID, "")
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	if minimal {
		res, err := app.core.GetLists("")
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		// Meta.
		out := models.PageResults{
			Results: res,
			Total:   len(res),
			Page:    1,
			PerPage: len(res),
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Full list query.
	res, total, err := app.core.QueryLists(query, orderBy, order, pg.Offset, pg.Limit)
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

// handleCreateList handles list creation.
func handleCreateList(c echo.Context) error {
	app := c.Get("app").(*App)
	var l models.List

	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	out, err := app.core.CreateList(l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateList handles list modification.
func handleUpdateList(c echo.Context) error {
	app := c.Get("app").(*App)
	id, _ := strconv.Atoi(c.Param("id"))

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	out, err := app.core.UpdateList(id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteLists handles list deletion, either a single one (ID in the URI), or a list.
func handleDeleteLists(c echo.Context) error {
	app := c.Get("app").(*App)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ids := []int{}

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := app.core.DeleteLists(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
