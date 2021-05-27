package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"

	"github.com/labstack/echo"
)

type listsWrap struct {
	Results []models.List `json:"results"`

	Total   int `json:"total"`
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
}

var (
	listQuerySortFields = []string{"name", "type", "subscriber_count", "created_at", "updated_at"}
)

// handleGetLists handles retrieval of lists.
func handleGetLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out listsWrap

		pg        = getPagination(c.QueryParams(), 20, 50)
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		listID, _ = strconv.Atoi(c.Param("id"))
		single    = false
	)

	// Fetch one list.
	if listID > 0 {
		single = true
	}

	// Sort params.
	if !strSliceContains(orderBy, listQuerySortFields) {
		orderBy = "created_at"
	}
	if order != sortAsc && order != sortDesc {
		order = sortAsc
	}

	results, err := app.store.QueryLists(listID, pg.Offset, pg.Limit, orderBy, order)
	if err != nil {
		app.log.Printf("error fetching lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	if single && len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}
	if len(results) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Replace null tags.
	for i, v := range results {
		if v.Tags == nil {
			results[i].Tags = make(pq.StringArray, 0)
		}
	}

	if single {
		return c.JSON(http.StatusOK, okResp{results[0]})
	}

	// Meta.
	out.Results = results
	out.Total = results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateList handles list creation.
func handleCreateList(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		o   = models.List{}
	)

	if err := c.Bind(&o); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(o.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("lists.invalidName"))
	}

	uu, err := uuid.NewV4()
	if err != nil {
		app.log.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Insert and read ID.
	var newID int
	o.UUID = uu.String()
	if err := app.store.CreateList(&newID,
		o.UUID,
		o.Name,
		o.Type,
		o.Optin,
		pq.StringArray(normalizeTags(o.Tags))); err != nil {
		app.log.Printf("error creating list: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorCreating",
				"name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	return handleGetLists(copyEchoCtx(c, map[string]string{
		"id": fmt.Sprintf("%d", newID),
	}))
}

// handleUpdateList handles list modification.
func handleUpdateList(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var o models.List
	if err := c.Bind(&o); err != nil {
		return err
	}

	res, err := app.store.UpdateList(id,
		o.Name, o.Type, o.Optin, pq.StringArray(normalizeTags(o.Tags)))
	if err != nil {
		app.log.Printf("error updating list: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	return handleGetLists(c)
}

// handleDeleteLists handles deletion deletion,
// either a single one (ID in the URI), or a list.
func handleDeleteLists(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   pq.Int64Array
	)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, id)
	}

	if err := app.store.DeleteLists(ids); err != nil {
		app.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}
