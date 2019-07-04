package main

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

// subQueryReq is a "catch all" struct for reading various
// subscriber related requests.
type subQueryReq struct {
	Query         string        `json:"query"`
	ListIDs       pq.Int64Array `json:"list_ids"`
	TargetListIDs pq.Int64Array `json:"target_list_ids"`
	SubscriberIDs pq.Int64Array `json:"ids"`
	Action        string        `json:"action"`
}

type subsWrap struct {
	Results models.Subscribers `json:"results"`

	Query   string `json:"query"`
	Total   int    `json:"total"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

var dummySubscriber = models.Subscriber{
	Email: "dummy@listmonk.app",
	Name:  "Dummy Subscriber",
	UUID:  "00000000-0000-0000-0000-000000000000",
}

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

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams())

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))

		// The "WHERE ?" bit.
		query = sanitizeSQLExp(c.FormValue("query"))
		out   subsWrap
	)

	listIDs := pq.Int64Array{}
	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid `list_id`.")
	} else if listID > 0 {
		listIDs = append(listIDs, int64(listID))
	}

	// There's an arbitrary query condition from the frontend.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	stmt := fmt.Sprintf(app.Queries.QuerySubscribers, cond)
	// Create a readonly transaction to prevent mutations.
	tx, err := app.DB.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error preparing query: %v", pqErrMsg(err)))
	}

	// Run the query.
	if err := tx.Select(&out.Results, stmt, listIDs, "id", pg.Offset, pg.Limit); err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error querying subscribers: %v", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.Results.LoadLists(app.Queries.GetSubscriberLists); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching subscriber lists: %v", pqErrMsg(err)))
	}

	out.Query = query
	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateSubscriber handles the creation of a new subscriber.
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

	// Insert and read ID.
	var newID int
	err := app.Queries.InsertSubscriber.Get(&newID,
		uuid.NewV4(),
		strings.ToLower(strings.TrimSpace(req.Email)),
		strings.TrimSpace(req.Name),
		req.Status,
		req.Attribs,
		req.Lists)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "subscribers_email_key" {
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

// handleUpdateSubscriber handles modification of a subscriber.
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

	_, err := app.Queries.UpdateSubscriber.Exec(req.ID,
		strings.ToLower(strings.TrimSpace(req.Email)),
		strings.TrimSpace(req.Name),
		req.Status,
		req.Attribs,
		req.Lists)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Update failed: %v", pqErrMsg(err)))
	}

	return handleGetSubscriber(c)
}

// handleBlacklistSubscribers handles the blacklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleBlacklistSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		var req subQueryReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("One or more invalid IDs given: %v", err))
		}
		if len(req.SubscriberIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}
		IDs = req.SubscriberIDs
	}

	if _, err := app.Queries.BlacklistSubscribers.Exec(IDs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error blacklisting: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleManageSubscriberLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
		}
		IDs = append(IDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("One or more invalid IDs given: %v", err))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			"No IDs given.")
	}
	if len(IDs) == 0 {
		IDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No lists given.")
	}

	// Action.
	var err error
	switch req.Action {
	case "add":
		_, err = app.Queries.AddSubscribersToLists.Exec(IDs, req.TargetListIDs)
	case "remove":
		_, err = app.Queries.DeleteSubscriptions.Exec(IDs, req.TargetListIDs)
	case "unsubscribe":
		_, err = app.Queries.UnsubscribeSubscribersFromLists.Exec(IDs, req.TargetListIDs)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid action.")
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error processing lists: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribers handles subscriber deletion.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleDeleteSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("One or more invalid IDs given: %v", err))
		}
		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}
		IDs = i
	}

	if _, err := app.Queries.DeleteSubscribers.Exec(IDs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error deleting: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribersByQuery bulk deletes based on an
// arbitrary SQL expression.
func handleDeleteSubscribersByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}

	err := app.Queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		app.Queries.DeleteSubscribersByQuery,
		req.ListIDs, app.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlacklistSubscribersByQuery bulk blacklists subscribers
// based on an arbitrary SQL expression.
func handleBlacklistSubscribersByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}

	err := app.Queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		app.Queries.BlacklistSubscribersByQuery,
		req.ListIDs, app.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlacklistSubscribersByQuery bulk adds/removes/unsubscribers subscribers
// from one or more lists based on an arbitrary SQL expression.
func handleManageSubscriberListsByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No lists given.")
	}

	// Action.
	var stmt string
	switch req.Action {
	case "add":
		stmt = app.Queries.AddSubscribersToListsByQuery
	case "remove":
		stmt = app.Queries.DeleteSubscriptionsByQuery
	case "unsubscribe":
		stmt = app.Queries.UnsubscribeSubscribersFromListsByQuery
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid action.")
	}

	err := app.Queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query), stmt, req.ListIDs, app.DB, req.TargetListIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// sanitizeSQLExp does basic sanitisation on arbitrary
// SQL query expressions coming from the frontend.
func sanitizeSQLExp(q string) string {
	if len(q) == 0 {
		return ""
	}
	q = strings.TrimSpace(q)

	// Remove semicolon suffix.
	if q[len(q)-1] == ';' {
		q = q[:len(q)-1]
	}
	return q
}
