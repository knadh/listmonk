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
	var (
		app       = c.Get("app").(*App)
		listID, _ = strconv.Atoi(c.Param("id"))
		pg        = app.paginator.NewFromURL(c.Request().URL.Query())

		query      = strings.TrimSpace(c.FormValue("query"))
		tags       = c.QueryParams()["tag"]
		orderBy    = c.FormValue("order_by")
		typ        = c.FormValue("type")
		optin      = c.FormValue("optin")
		order      = c.FormValue("order")
		minimal, _ = strconv.ParseBool(c.FormValue("minimal"))

		out models.PageResults
	)

	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	// Fetch one list.
	single := false
	if listID > 0 {
		single = true
	}

	if single {
		out, err := app.core.GetList(listID, "", authID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	if !single && minimal {
		res, err := app.core.GetLists("", authID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		// Meta.
		out.Results = res
		out.Total = len(res)
		out.Page = 1
		out.PerPage = out.Total

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Full list query.
	res, total, err := app.core.QueryLists(query, typ, optin, tags, orderBy, order, pg.Offset, pg.Limit, authID)
	if err != nil {
		return err
	}

	if single && len(res) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	if single {
		return c.JSON(http.StatusOK, okResp{res[0]})
	}

	out.Query = query
	out.Results = res
	out.Total = total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetListsByAuthID handles retrieving lists associated with particular AuthID
// handleGetListsByAuthID retrieves lists associated with a particular authid.
// func handleGetListsByAuthID(c echo.Context) error {
// 	var (
// 		app    = c.Get("app").(*App)
// 		authID = c.Param("authid") // Assuming you are passing authid in the URL path
// 		out    []models.List
// 	)

// 	// Fetch lists by authid
// 	res, err := app.core.GetListsByAuthID(authID)
// 	if err != nil {
// 		return err
// 	}

// 	// Check if no lists were found
// 	if len(res) == 0 {
// 		return c.JSON(http.StatusOK, okResp{[]struct{}{}}) // Return empty response
// 	}

// 	out = res // Assign results

// 	return c.JSON(http.StatusOK, okResp{out}) // Return lists as JSON
// }

// handleCreateList handles list creation.
func handleCreateList(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		l   = models.List{}
	)

	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	out, err := app.core.CreateList(l, authID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateList handles list modification.
func handleUpdateList(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)
	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

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

	out, err := app.core.UpdateList(id, l, authID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteLists handles list deletion, either a single one (ID in the URI), or a list.
func handleDeleteLists(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)
	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := app.core.DeleteLists(ids, authID); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
