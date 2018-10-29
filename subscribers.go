package main

// !!!!!!!!!!! TODO
// For non-flat JSON attribs, show the advanced editor instead of the key-value editor

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/subimporter"
	"github.com/labstack/echo"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

type subsWrap struct {
	Results models.Subscribers `json:"results"`

	Query   string `json:"query"`
	Total   int    `json:"total"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

type queryAddResp struct {
	Count int64 `json:"count"`
}

type queryAddReq struct {
	Query       string        `json:"query"`
	SourceList  int           `json:"source_list"`
	TargetLists pq.Int64Array `json:"target_lists"`
}

var jsonMap = []byte("{}")

// handleGetSubscriber handles the retrieval of a single subscriber by ID.
func handleGetSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))

		out models.Subscribers
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid subscriber ID.")
	}

	err := app.Queries.GetSubscriber.Select(&out, id, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching subscriber: %s", pqErrMsg(err)))
	} else if len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Subscriber not found.")
	}
	out.LoadLists(app.Queries.GetSubscriberLists)

	return c.JSON(http.StatusOK, okResp{out[0]})
}

// handleQuerySubscribers handles querying subscribers based on arbitrary conditions in SQL.
func handleQuerySubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams())

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))
		hasList   bool

		// The "WHERE ?" bit.
		query = c.FormValue("query")

		out subsWrap
	)

	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `list_id`.")
	} else if listID > 0 {
		hasList = true
	}

	// There's an arbitrary query condition from the frontend.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	// The SQL queries to be executed are different for global subscribers
	// and subscribers belonging to a specific list.
	var (
		stmt      = ""
		stmtCount = ""
	)
	if hasList {
		stmt = fmt.Sprintf(app.Queries.QuerySubscribersByList,
			listID, cond, pg.Offset, pg.Limit)
		stmtCount = fmt.Sprintf(app.Queries.QuerySubscribersByListCount,
			listID, cond)
	} else {
		stmt = fmt.Sprintf(app.Queries.QuerySubscribers,
			cond, pg.Offset, pg.Limit)
		stmtCount = fmt.Sprintf(app.Queries.QuerySubscribersCount, cond)
	}

	// Create a readonly transaction to prevent mutations.
	tx, err := app.DB.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error preparing query: %v", pqErrMsg(err)))
	}

	// Run the actual query.
	if err := tx.Select(&out.Results, stmt); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error querying subscribers: %v", pqErrMsg(err)))
	}

	// Run the query count.
	if err := tx.Get(&out.Total, stmtCount); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error running count query: %v", pqErrMsg(err)))
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error in subscriber query transaction: %v", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.Results.LoadLists(app.Queries.GetSubscriberLists); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching subscriber lists: %v", pqErrMsg(err)))
	}

	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Query = query
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateSubscriber handles subscriber creation.
func handleCreateSubscriber(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subimporter.SubReq
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	} else if err := subimporter.ValidateFields(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Insert and read ID.
	var newID int
	err := app.Queries.UpsertSubscriber.Get(&newID,
		uuid.NewV4(),
		req.Email,
		req.Name,
		req.Status,
		req.Attribs,
		true,
		req.Lists)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest, "The e-mail already exists.")
		}

		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error creating subscriber: %v", err))
	}

	// Hand over to the GET handler to return the last insertion.
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", newID))
	return c.JSON(http.StatusOK, handleGetSubscriber(c))
}

// handleUpdateSubscriber handles subscriber modification.
func handleUpdateSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		req   subimporter.SubReq
	)
	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}
	if req.Email != "" && !govalidator.IsEmail(req.Email) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `email`.")
	}
	if req.Name != "" && !govalidator.IsByteLength(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid length for `name`.")
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	_, err := app.Queries.UpdateSubscriber.Exec(req.ID,
		req.Email,
		req.Name,
		req.Status,
		req.Attribs,
		req.Lists)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Update failed: %v", pqErrMsg(err)))
	}

	return handleGetSubscriber(c)
}

// handleDeleteSubscribers handles subscriber deletion,
// either a single one (ID in the URI), or a list.
func handleDeleteSubscribers(c echo.Context) error {
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

	if _, err := app.Queries.DeleteSubscribers.Exec(ids); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Delete failed: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleQuerySubscribersIntoLists handles querying subscribers based on arbitrary conditions in SQL
// and adding them to given lists.
func handleQuerySubscribersIntoLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req queryAddReq
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error parsing request: %v", err))
	}

	if len(req.TargetLists) < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `target_lists`.")
	}

	if req.SourceList < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `source_list`.")
	}

	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid subscriber `query`.")
	}
	cond := " AND " + req.Query

	// The SQL queries to be executed are different for global subscribers
	// and subscribers belonging to a specific list.
	var (
		stmt    = ""
		stmtDry = ""
	)
	if req.SourceList > 0 {
		stmt = fmt.Sprintf(app.Queries.QuerySubscribersByList, req.SourceList, cond)
		stmtDry = fmt.Sprintf(app.Queries.QuerySubscribersByList, req.SourceList, cond, 0, 1)
	} else {
		stmt = fmt.Sprintf(app.Queries.QuerySubscribersIntoLists, cond)
		stmtDry = fmt.Sprintf(app.Queries.QuerySubscribers, cond, 0, 1)
	}

	// Create a readonly transaction to prevent mutations.
	// This is used to dry-run the arbitrary query before it's used to
	// insert subscriptions.
	tx, err := app.DB.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error preparing query (dry-run): %v", pqErrMsg(err)))
	}

	// Perform the dry run.
	if _, err := tx.Exec(stmtDry); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error querying (dry-run) subscribers: %v", pqErrMsg(err)))
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error in subscriber dry-run query transaction: %v", pqErrMsg(err)))
	}

	// Prepare the query.
	q, err := app.DB.Preparex(stmt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error preparing query: %v", pqErrMsg(err)))
	}

	// Run the query.
	res, err := q.Exec(req.TargetLists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error adding subscribers to lists: %v", pqErrMsg(err)))
	}

	num, _ := res.RowsAffected()
	return c.JSON(http.StatusOK, okResp{queryAddResp{num}})
}
