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
func (h *Handler) handleGetLists(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)
		pg   = h.app.paginator.NewFromURL(c.Request().URL.Query())

		query      = strings.TrimSpace(c.FormValue("query"))
		tags       = c.QueryParams()["tag"]
		orderBy    = c.FormValue("order_by")
		typ        = c.FormValue("type")
		optin      = c.FormValue("optin")
		order      = c.FormValue("order")
		minimal, _ = strconv.ParseBool(c.FormValue("minimal"))

		out models.PageResults
	)

	var (
		permittedIDs []int
		getAll       = false
	)
	if _, ok := user.PermissionsMap[models.PermListGetAll]; ok {
		getAll = true
	} else {
		permittedIDs = user.GetListIDs
	}

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	if minimal {
		res, err := h.app.core.GetLists("", getAll, permittedIDs)
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
	res, total, err := h.app.core.QueryLists(query, typ, optin, tags, orderBy, order, getAll, permittedIDs, pg.Offset, pg.Limit)
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
func (h *Handler) handleGetList(c echo.Context) error {
	listID, _ := strconv.Atoi(c.Param("id"))

	out, err := h.app.core.GetList(listID, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateList handles list creation.
func (h *Handler) handleCreateList(c echo.Context) error {
	l := models.List{}

	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("lists.invalidName"))
	}

	out, err := h.app.core.CreateList(l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateList handles list modification.
func (h *Handler) handleUpdateList(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("lists.invalidName"))
	}

	out, err := h.app.core.UpdateList(id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteLists handles list deletion, either a single one (ID in the URI), or a list.
func (h *Handler) handleDeleteLists(c echo.Context) error {
	var (
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := h.app.core.DeleteLists(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// listPerm is a middleware for wrapping /list/* API calls that take a
// list :id param for validating the list ID against the user's list perms.
func (h *Handler) listPerm() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
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
			if _, ok := user.PermissionsMap[permAll]; ok {
				return next(c)
			}
			if id > 0 {
				if _, ok := user.ListPermissionsMap[id][perm]; ok {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.permissionDenied", "name", "list"))
		}
	}
}
