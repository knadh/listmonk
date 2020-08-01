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

// handleGetLists handles retrieval of lists.
func handleGetLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out listsWrap

		pg        = getPagination(c.QueryParams(), 20, 50)
		listID, _ = strconv.Atoi(c.Param("id"))
		single    = false
	)

	// Fetch one list.
	if listID > 0 {
		single = true
	}

	err := app.queries.GetLists.Select(&out.Results, listID, pg.Offset, pg.Limit)
	if err != nil {
		app.log.Printf("error fetching lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching lists: %s", pqErrMsg(err)))
	}
	if single && len(out.Results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "List not found.")
	}
	if len(out.Results) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Replace null tags.
	for i, v := range out.Results {
		if v.Tags == nil {
			out.Results[i].Tags = make(pq.StringArray, 0)
		}
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out.Results[0]})
	}

	// Meta.
	out.Total = out.Results[0].Total
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
		return echo.NewHTTPError(http.StatusBadRequest,
			"Invalid length for the name field.")
	}

	uu, err := uuid.NewV4()
	if err != nil {
		app.log.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error generating UUID")
	}

	// Insert and read ID.
	var newID int
	o.UUID = uu.String()
	if err := app.queries.CreateList.Get(&newID,
		o.UUID,
		o.Name,
		o.Type,
		o.Optin,
		pq.StringArray(normalizeTags(o.Tags))); err != nil {
		app.log.Printf("error creating list: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error creating list: %s", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", newID))
	return c.JSON(http.StatusOK, handleGetLists(c))
}

// handleUpdateList handles list modification.
func handleUpdateList(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	// Incoming params.
	var o models.List
	if err := c.Bind(&o); err != nil {
		return err
	}

	res, err := app.queries.UpdateList.Exec(id,
		o.Name, o.Type, o.Optin, pq.StringArray(normalizeTags(o.Tags)))
	if err != nil {
		app.log.Printf("error updating list: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error updating list: %s", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "List not found.")
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
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	if id > 0 {
		ids = append(ids, id)
	}

	if _, err := app.queries.DeleteLists.Exec(ids); err != nil {
		app.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error deleting: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}
