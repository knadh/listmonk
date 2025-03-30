package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleGetLists retrieves lists with additional metadata like subscriber counts.
func handleGetLists(c echo.Context) error {
	var (
		app  = c.Get("app").(*App)
		user = c.Get(auth.UserKey).(models.User)
		pg   = app.paginator.NewFromURL(c.Request().URL.Query())

		query      = strings.TrimSpace(c.FormValue("query"))
		tags       = c.QueryParams()["tag"]
		orderBy    = c.FormValue("order_by")
		typ        = c.FormValue("type")
		optin      = c.FormValue("optin")
		order      = c.FormValue("order")
		minimal, _ = strconv.ParseBool(c.FormValue("minimal"))

		out models.PageResults
	)

	// Get the list IDs (or blanket permission) the user has access to.
	hasAllPerm, permittedIDs := user.GetPermittedLists(true, false)

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	if minimal {
		res, err := app.core.GetLists("", hasAllPerm, permittedIDs)
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
	res, total, err := app.core.QueryLists(query, typ, optin, tags, orderBy, order, hasAllPerm, permittedIDs, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out.Query = query
	out.Results = res
	out.Total = total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetList retrieves a single list by id.
// It's permission checked by the listPerm middleware.
func handleGetList(c echo.Context) error {
	var (
		app       = c.Get("app").(*App)
		listID, _ = strconv.Atoi(c.Param("id"))
	)

	out, err := app.core.GetList(listID, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateList handles list creation.
func handleCreateList(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		l   = models.List{}
	)

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
// It's permission checked by the listPerm middleware.
func handleUpdateList(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

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
// It's permission checked by the listPerm middleware.
func handleDeleteLists(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)

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

// listPerm is a middleware for wrapping /list/* API calls that take a
// list :id param for validating the list ID against the user's list perms.
func listPerm(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			app   = c.Get("app").(*App)
			user  = c.Get(auth.UserKey).(models.User)
			id, _ = strconv.Atoi(c.Param("id"))
		)

		// Define permissions based on HTTP read/write.
		var (
			permAll = models.PermListManageAll
			perm    = models.PermListManage
		)
		if c.Request().Method == http.MethodGet {
			permAll = models.PermListGetAll
			perm = models.PermListGet
		}

		// Check if the user has permissions for all lists or the specific list.
		if user.HasPerm(permAll) {
			return next(c)
		}

		if id > 0 {
			if user.HasListPerm(id, perm) {
				return next(c)
			}
		}

		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.permissionDenied", "name", "list"))
	}
}
