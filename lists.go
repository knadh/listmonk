package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"

	"github.com/asaskevich/govalidator"
	"github.com/labstack/echo"
)

// handleGetLists handles retrieval of lists.
func handleGetLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []models.List

		listID, _ = strconv.Atoi(c.Param("id"))
		single    = false
	)

	// Fetch one list.
	if listID > 0 {
		single = true
	}

	err := app.Queries.GetLists.Select(&out, listID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching lists: %s", pqErrMsg(err)))
	} else if single && len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "List not found.")
	} else if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Replace null tags.
	for i, v := range out {
		if v.Tags == nil {
			out[i].Tags = make(pq.StringArray, 0)
		}
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out[0]})
	}

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
	if !govalidator.IsByteLength(o.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Invalid length for the name field.")
	}

	// Insert and read ID.
	var newID int
	o.UUID = uuid.NewV4().String()
	if err := app.Queries.CreateList.Get(&newID,
		o.UUID,
		o.Name,
		o.Type,
		pq.StringArray(normalizeTags(o.Tags))); err != nil {
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

	res, err := app.Queries.UpdateList.Exec(id, o.Name, o.Type, pq.StringArray(normalizeTags(o.Tags)))
	if err != nil {
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

	// Read the list IDs if they were sent in the body.
	c.Bind(&ids)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	if id > 0 {
		ids = append(ids, id)
	}

	if _, err := app.Queries.DeleteLists.Exec(ids); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Delete failed: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}
