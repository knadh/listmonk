package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

const (
	dummyUUID = "00000000-0000-0000-0000-000000000000"
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

type subUpdateReq struct {
	models.Subscriber
	RawAttribs     json.RawMessage `json:"attribs"`
	Lists          pq.Int64Array   `json:"lists"`
	ListUUIDs      pq.StringArray  `json:"list_uuids"`
	PreconfirmSubs bool            `json:"preconfirm_subscriptions"`
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

var (
	dummySubscriber = models.Subscriber{
		Email:   "demo@listmonk.app",
		Name:    "Demo Subscriber",
		UUID:    dummyUUID,
		Attribs: models.SubscriberAttribs{"city": "Bengaluru"},
	}

	subQuerySortFields = []string{"email", "name", "created_at", "updated_at"}

	errSubscriberExists = errors.New("subscriber already exists")
)

// handleGetSubscriber handles the retrieval of a single subscriber by ID.
func handleGetSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	sub, err := getSubscriber(id, "", "", app)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 30)

		// The "WHERE ?" bit.
		query   = sanitizeSQLExp(c.FormValue("query"))
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
		out     = subsWrap{Results: make([]models.Subscriber, 0, 1)}
	)

	// Limit the subscribers to sepcific lists?
	listIDs, err := getQueryListIDs(c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "subscribers.id"
	}
	if order != sortAsc && order != sortDesc {
		order = sortDesc
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	stmt := fmt.Sprintf(app.queries.QuerySubscribersCount, cond)
	tx, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Execute the readonly query and get the count of results.
	var total = 0
	if err := tx.Get(&total, stmt, listIDs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// No results.
	if total == 0 {
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Run the query again and fetch the actual data. stmt is the raw SQL query.
	stmt = fmt.Sprintf(app.queries.QuerySubscribers, cond, orderBy, order)
	if err := tx.Select(&out.Results, stmt, listIDs, pg.Offset, pg.Limit); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.Results.LoadLists(app.queries.GetSubscriberListsLazy); err != nil {
		app.log.Printf("error fetching subscriber lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	out.Query = query
	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Total = total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleExportSubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleExportSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)

		// The "WHERE ?" bit.
		query = sanitizeSQLExp(c.FormValue("query"))
	)

	// Limit the subscribers to sepcific lists?
	listIDs, err := getQueryListIDs(c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	stmt := fmt.Sprintf(app.queries.QuerySubscribersForExport, cond)

	// Verify that the arbitrary SQL search expression is read only.
	if cond != "" {
		tx, err := app.db.Unsafe().BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			app.log.Printf("error preparing subscriber query: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
		defer tx.Rollback()

		if _, err := tx.Query(stmt, nil, 0, 1); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
	}

	// Prepare the actual query statement.
	tx, err := db.Preparex(stmt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}

	// Run the query until all rows are exhausted.
	var (
		id = 0

		h  = c.Response().Header()
		wr = csv.NewWriter(c.Response())
	)

	h.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	h.Set("Content-type", "text/csv")
	h.Set(echo.HeaderContentDisposition, "attachment; filename="+"subscribers.csv")
	h.Set("Content-Transfer-Encoding", "binary")
	h.Set("Cache-Control", "no-cache")
	wr.Write([]string{"uuid", "email", "name", "attributes", "status", "created_at", "updated_at"})

loop:
	for {
		var out []models.SubscriberExport
		if err := tx.Select(&out, listIDs, id, app.constants.DBBatchSize); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
		if len(out) == 0 {
			break loop
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				app.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}
		wr.Flush()

		id = out[len(out)-1].ID
	}

	return nil
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
	}

	r, err := app.importer.ValidateFields(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req = r
	}

	// Insert the subscriber into the DB.
	sub, isNew, _, err := insertSubscriber(req, app)
	if err != nil {
		return err
	}
	if !isNew {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.emailExists"))
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleUpdateSubscriber handles modification of a subscriber.
func handleUpdateSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		req   subUpdateReq
	)
	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if em, err := app.importer.SanitizeEmail(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req.Email = em
	}

	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidName"))
	}

	// If there's an attribs value, validate it.
	if len(req.RawAttribs) > 0 {
		var a models.SubscriberAttribs
		if err := json.Unmarshal(req.RawAttribs, &a); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		}
	}

	subStatus := models.SubscriptionStatusUnconfirmed
	if req.PreconfirmSubs {
		subStatus = models.SubscriptionStatusConfirmed
	}

	_, err := app.queries.UpdateSubscriber.Exec(id,
		strings.ToLower(strings.TrimSpace(req.Email)),
		strings.TrimSpace(req.Name),
		req.Status,
		req.RawAttribs,
		req.Lists,
		subStatus)
	if err != nil {
		app.log.Printf("error updating subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	// Send a confirmation e-mail (if there are any double opt-in lists).
	sub, err := getSubscriber(int(id), "", "", app)
	if err != nil {
		return err
	}

	if !req.PreconfirmSubs && app.constants.SendOptinConfirmation {
		_, _ = sendOptinConfirmation(sub, []int64(req.Lists), app)
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleGetSubscriberSendOptin sends an optin confirmation e-mail to a subscriber.
func handleSubscriberSendOptin(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Fetch the subscriber.
	out, err := getSubscriber(id, "", "", app)
	if err != nil {
		app.log.Printf("error fetching subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	if _, err := sendOptinConfirmation(out, nil, app); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribers handles the blocklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleBlocklistSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		var req subQueryReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(req.SubscriberIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}
		IDs = req.SubscriberIDs
	}

	if _, err := app.queries.BlocklistSubscribers.Exec(IDs); err != nil {
		app.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
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
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoIDs"))
	}
	if len(IDs) == 0 {
		IDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Action.
	var err error
	switch req.Action {
	case "add":
		_, err = app.queries.AddSubscribersToLists.Exec(IDs, req.TargetListIDs)
	case "remove":
		_, err = app.queries.DeleteSubscriptions.Exec(IDs, req.TargetListIDs)
	case "unsubscribe":
		_, err = app.queries.UnsubscribeSubscribersFromLists.Exec(IDs, req.TargetListIDs)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		app.log.Printf("error updating subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscribers}", "error", err.Error()))
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
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorNoIDs", "error", err.Error()))
		}
		IDs = i
	}

	if _, err := app.queries.DeleteSubscribers.Exec(IDs, nil); err != nil {
		app.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
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

	err := app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		app.queries.DeleteSubscribersByQuery,
		req.ListIDs, app.db)
	if err != nil {
		app.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribersByQuery bulk blocklists subscribers
// based on an arbitrary SQL expression.
func handleBlocklistSubscribersByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}

	err := app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		app.queries.BlocklistSubscribersByQuery,
		req.ListIDs, app.db)
	if err != nil {
		app.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberListsByQuery bulk adds/removes/unsubscribers subscribers
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
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Action.
	var stmt string
	switch req.Action {
	case "add":
		stmt = app.queries.AddSubscribersToListsByQuery
	case "remove":
		stmt = app.queries.DeleteSubscriptionsByQuery
	case "unsubscribe":
		stmt = app.queries.UnsubscribeSubscribersFromListsByQuery
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	err := app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		stmt, req.ListIDs, app.db, req.TargetListIDs)
	if err != nil {
		app.log.Printf("error updating subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscriberBounces deletes all the bounces on a subscriber.
func handleDeleteSubscriberBounces(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
	)

	id, _ := strconv.ParseInt(pID, 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if _, err := app.queries.DeleteBouncesBySubscriber.Exec(id, nil); err != nil {
		app.log.Printf("error deleting bounces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.bounces}", "error", pqErrMsg(err)))
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
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	_, b, err := exportSubscriberData(id, "", app.constants.Privacy.Exportable, app)
	if err != nil {
		app.log.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="data.json"`)
	return c.Blob(http.StatusOK, "application/json", b)
}

// insertSubscriber inserts a subscriber and returns the ID. The first bool indicates if
// it was a new subscriber, and the second bool indicates if the subscriber was sent an optin confirmation.
func insertSubscriber(req subimporter.SubReq, app *App) (models.Subscriber, bool, bool, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		return req.Subscriber, false, false, err
	}
	req.UUID = uu.String()

	var (
		isNew     = true
		subStatus = models.SubscriptionStatusUnconfirmed
	)
	if req.PreconfirmSubs {
		subStatus = models.SubscriptionStatusConfirmed
	}

	if req.Status == "" {
		req.Status = models.UserStatusEnabled
	}

	if err = app.queries.InsertSubscriber.Get(&req.ID,
		req.UUID,
		req.Email,
		strings.TrimSpace(req.Name),
		req.Status,
		req.Attribs,
		req.Lists,
		req.ListUUIDs,
		subStatus); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "subscribers_email_key" {
			isNew = false
		} else {
			// return req.Subscriber, errSubscriberExists
			app.log.Printf("error inserting subscriber: %v", err)
			return req.Subscriber, false, false, echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorCreating",
					"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
		}
	}

	// Fetch the subscriber's full data. If the subscriber already existed and wasn't
	// created, the id will be empty. Fetch the details by e-mail then.
	sub, err := getSubscriber(req.ID, "", strings.ToLower(req.Email), app)
	if err != nil {
		return sub, false, false, err
	}

	hasOptin := false
	if !req.PreconfirmSubs && app.constants.SendOptinConfirmation {
		// Send a confirmation e-mail (if there are any double opt-in lists).
		num, _ := sendOptinConfirmation(sub, []int64(req.Lists), app)
		hasOptin = num > 0
	}
	return sub, isNew, hasOptin, nil
}

// getSubscriber gets a single subscriber by ID, uuid, or e-mail in that order.
// Only one of these params should have a value.
func getSubscriber(id int, uuid, email string, app *App) (models.Subscriber, error) {
	var out models.Subscribers

	if err := app.queries.GetSubscriber.Select(&out, id, uuid, email); err != nil {
		app.log.Printf("error fetching subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return models.Subscriber{}, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.subscriber}"))
	}
	if err := out.LoadLists(app.queries.GetSubscriberListsLazy); err != nil {
		app.log.Printf("error loading subscriber lists: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out[0], nil
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
	if err := app.queries.ExportSubscriberData.Get(&data, id, uu); err != nil {
		app.log.Printf("error fetching subscriber export data: %v", err)
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
		app.log.Printf("error marshalling subscriber export data: %v", err)
		return data, nil, err
	}
	return data, b, nil
}

// sendOptinConfirmation sends a double opt-in confirmation e-mail to a subscriber
// if at least one of the given listIDs is set to optin=double. It returns the number of
// opt-in lists that were found.
func sendOptinConfirmation(sub models.Subscriber, listIDs []int64, app *App) (int, error) {
	var lists []models.List

	// Fetch double opt-in lists from the given list IDs.
	// Get the list of subscription lists where the subscriber hasn't confirmed.
	if err := app.queries.GetSubscriberLists.Select(&lists, sub.ID, nil,
		pq.Int64Array(listIDs), nil, models.SubscriptionStatusUnconfirmed, models.ListOptinDouble); err != nil {
		app.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return 0, err
	}

	// None.
	if len(lists) == 0 {
		return 0, nil
	}

	var (
		out      = subOptin{Subscriber: &sub, Lists: lists}
		qListIDs = url.Values{}
	)
	// Construct the opt-in URL with list IDs.
	for _, l := range out.Lists {
		qListIDs.Add("l", l.UUID)
	}
	out.OptinURL = fmt.Sprintf(app.constants.OptinURL, sub.UUID, qListIDs.Encode())

	// Send the e-mail.
	if err := app.sendNotification([]string{sub.Email},
		app.i18n.T("subscribers.optinSubject"), notifSubscriberOptin, out); err != nil {
		app.log.Printf("error sending opt-in e-mail: %s", err)
		return 0, err
	}
	return len(lists), nil
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

func getQueryListIDs(qp url.Values) (pq.Int64Array, error) {
	out := pq.Int64Array{}
	if vals, ok := qp["list_id"]; ok {
		for _, v := range vals {
			if v == "" {
				continue
			}

			listID, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			out = append(out, int64(listID))
		}
	}

	return out, nil
}
