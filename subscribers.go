package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

// subProfileData represents a subscriber's collated data in JSON
// for export.
type subProfileData struct {
	Email         string          `db:"email" json:"-"`
	Profile       json.RawMessage `db:"profile" json:"profile,omitempty"`
	Subscriptions json.RawMessage `db:"subscriptions" json:"subscriptions,omitempty"`
	CampaignViews json.RawMessage `db:"campaign_views" json:"campaign_views,omitempty"`
	LinkClicks    json.RawMessage `db:"link_clicks" json:"link_clicks,omitempty"`
}

// subOptin contains the data that's passed to the double opt-in e-mail template.
type subOptin struct {
	*models.Subscriber

	OptinURL string
	Lists    []models.List
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
	}
	if len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Subscriber not found.")
	}
	if err := out.LoadLists(app.Queries.GetSubscriberListsLazy); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error loading lists for subscriber.")
	}

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
	defer tx.Rollback()

	// Run the query.
	if err := tx.Select(&out.Results, stmt, listIDs, "id", pg.Offset, pg.Limit); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error querying subscribers: %v", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.Results.LoadLists(app.Queries.GetSubscriberListsLazy); err != nil {
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
	var (
		newID int
		email = strings.ToLower(strings.TrimSpace(req.Email))
	)
	req.UUID = uuid.NewV4().String()
	err := app.Queries.InsertSubscriber.Get(&newID,
		req.UUID,
		email,
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

	// If the lists are double-optins, send confirmation e-mails.
	go sendOptinConfirmation(req.Subscriber, []int64(req.Lists), app)

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

	if _, err := app.Queries.DeleteSubscribers.Exec(IDs, nil); err != nil {
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

// handleExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in app.Constants.Privacy.
func handleExportSubscriberData(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
	)
	id, _ := strconv.ParseInt(pID, 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	_, b, err := exportSubscriberData(id, "", app.Constants.Privacy.Exportable, app)
	if err != nil {
		app.Logger.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			"Error exporting subscriber data.")
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="profile.json"`)
	return c.Blob(http.StatusOK, "application/json", b)
}

// exportSubscriberData collates the data of a subscriber including profile,
// subscriptions, campaign_views, link_clicks (if they're enabled in the config)
// and returns a formatted, indented JSON payload. Either takes a numeric id
// and an empty subUUID or takes 0 and a string subUUID.
func exportSubscriberData(id int64, subUUID string, exportables map[string]bool, app *App) (subProfileData, []byte, error) {
	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	var (
		data subProfileData
		uu   interface{}
	)
	// UUID should be a valid value or a nil.
	if subUUID != "" {
		uu = subUUID
	}
	if err := app.Queries.ExportSubscriberData.Get(&data, id, uu); err != nil {
		return data, nil, err
	}

	// Filter out the non-exportable items.
	if _, ok := exportables["profile"]; !ok {
		data.Profile = nil
	}
	if _, ok := exportables["subscriptions"]; !ok {
		data.Subscriptions = nil
	}
	if _, ok := exportables["campaign_views"]; !ok {
		data.CampaignViews = nil
	}
	if _, ok := exportables["link_clicks"]; !ok {
		data.LinkClicks = nil
	}

	// Marshal the data into an indented payload.
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return data, nil, err
	}
	return data, b, nil
}

// sendOptinConfirmation sends double opt-in confirmation e-mails to a subscriber
// if at least one of the given listIDs is set to optin=double
func sendOptinConfirmation(sub models.Subscriber, listIDs []int64, app *App) error {
	var lists []models.List

	// Fetch double opt-in lists from the given list IDs.
	err := app.Queries.GetListsByOptin.Select(&lists, models.ListOptinDouble, pq.Int64Array(listIDs))
	if err != nil {
		app.Logger.Printf("error fetching lists for optin: %s", pqErrMsg(err))
		return err
	}

	// None.
	if len(lists) == 0 {
		return nil
	}

	var (
		out      = subOptin{Subscriber: &sub, Lists: lists}
		qListIDs = url.Values{}
	)
	// Construct the opt-in URL with list IDs.
	for _, l := range out.Lists {
		qListIDs.Add("l", l.UUID)
	}
	out.OptinURL = fmt.Sprintf(app.Constants.OptinURL, sub.UUID, qListIDs.Encode())

	// Send the e-mail.
	if err := sendNotification([]string{sub.Email},
		"Confirm subscription",
		notifSubscriberOptin, out, app); err != nil {
		app.Logger.Printf("error e-mailing subscriber profile: %s", err)
		return err
	}

	return nil
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
