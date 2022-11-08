package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const (
	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// subQueryReq is a "catch all" struct for reading various
// subscriber related requests.
type subQueryReq struct {
	Query         string `json:"query"`
	ListIDs       []int  `json:"list_ids"`
	TargetListIDs []int  `json:"target_list_ids"`
	SubscriberIDs []int  `json:"ids"`
	Action        string `json:"action"`
	Status        string `json:"status"`
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
	models.Subscriber

	OptinURL string
	UnsubURL string
	Lists    []models.List
}

var (
	dummySubscriber = models.Subscriber{
		Email:   "demo@listmonk.app",
		Name:    "Demo Subscriber",
		UUID:    dummyUUID,
		Attribs: models.JSON{"city": "Bengaluru"},
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

	out, err := app.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = app.paginator.NewFromURL(c.Request().URL.Query())

		// The "WHERE ?" bit.
		query   = sanitizeSQLExp(c.FormValue("query"))
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
		out     models.PageResults
	)

	// Limit the subscribers to specific lists?
	listIDs, err := getQueryInts("list_id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	res, total, err := app.core.QuerySubscribers(query, listIDs, order, orderBy, pg.Offset, pg.Limit)
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

// handleExportSubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleExportSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)

		// The "WHERE ?" bit.
		query = sanitizeSQLExp(c.FormValue("query"))
	)

	// Limit the subscribers to specific lists?
	listIDs, err := getQueryInts("list_id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Export only specific subscriber IDs?
	subIDs, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Get the batched export iterator.
	exp, err := app.core.ExportSubscribers(query, subIDs, listIDs, app.constants.DBBatchSize)
	if err != nil {
		return err
	}

	var (
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
	// Iterate in batches until there are no more subscribers to export.
	for {
		out, err := exp()
		if err != nil {
			return err
		}
		if out == nil || len(out) == 0 {
			break
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				app.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}

		// Flush CSV to stream after each batch.
		wr.Flush()
	}

	return nil
}

// handleCreateSubscriber handles the creation of a new subscriber.
func handleCreateSubscriber(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req struct {
			models.Subscriber
			Lists          []int    `json:"lists"`
			ListUUIDs      []string `json:"list_uuids"`
			PreconfirmSubs bool     `json:"preconfirm_subscriptions"`
		}
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate fields.
	if len(req.Email) > 1000 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidEmail"))
	}

	em, err := app.importer.SanitizeEmail(req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = em

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 || len(req.Name) > stdInputMaxLen {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidName"))
	}

	// Insert the subscriber into the DB.
	sub, _, err := app.core.InsertSubscriber(req.Subscriber, req.Lists, req.ListUUIDs, req.PreconfirmSubs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleUpdateSubscriber handles modification of a subscriber.
func handleUpdateSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
		req   struct {
			models.Subscriber
			Lists          []int `json:"lists"`
			PreconfirmSubs bool  `json:"preconfirm_subscriptions"`
		}
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

	out, err := app.core.UpdateSubscriberWithLists(id, req.Subscriber, req.Lists, nil, req.PreconfirmSubs, true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
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
	out, err := app.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	if _, err := sendOptinConfirmationHook(app)(out, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribers handles the blocklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleBlocklistSubscribers(c echo.Context) error {
	var (
		app    = c.Get("app").(*App)
		pID    = c.Param("id")
		subIDs []int
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}

		subIDs = append(subIDs, id)
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

		subIDs = req.SubscriberIDs
	}

	if err := app.core.BlocklistSubscribers(subIDs); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleManageSubscriberLists(c echo.Context) error {
	var (
		app    = c.Get("app").(*App)
		pID    = c.Param("id")
		subIDs []int
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoIDs"))
	}
	if len(subIDs) == 0 {
		subIDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Action.
	var err error
	switch req.Action {
	case "add":
		err = app.core.AddSubscriptions(subIDs, req.TargetListIDs, req.Status)
	case "remove":
		err = app.core.DeleteSubscriptions(subIDs, req.TargetListIDs)
	case "unsubscribe":
		err = app.core.UnsubscribeLists(subIDs, req.TargetListIDs, nil)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribers handles subscriber deletion.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleDeleteSubscribers(c echo.Context) error {
	var (
		app    = c.Get("app").(*App)
		pID    = c.Param("id")
		subIDs []int
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
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
		subIDs = i
	}

	if err := app.core.DeleteSubscribers(subIDs, nil); err != nil {
		return err
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

	if err := app.core.DeleteSubscribersByQuery(req.Query, req.ListIDs); err != nil {
		return err
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

	if err := app.core.BlocklistSubscribersByQuery(req.Query, req.ListIDs); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberListsByQuery bulk adds/removes/unsubscribes subscribers
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
	var err error
	switch req.Action {
	case "add":
		err = app.core.AddSubscriptionsByQuery(req.Query, req.ListIDs, req.TargetListIDs)
	case "remove":
		err = app.core.DeleteSubscriptionsByQuery(req.Query, req.ListIDs, req.TargetListIDs)
	case "unsubscribe":
		err = app.core.UnsubscribeListsByQuery(req.Query, req.ListIDs, req.TargetListIDs)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscriberBounces deletes all the bounces on a subscriber.
func handleDeleteSubscriberBounces(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
	)

	id, _ := strconv.Atoi(pID)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if err := app.core.DeleteSubscriberBounces(id, ""); err != nil {
		return err
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

	id, _ := strconv.Atoi(pID)
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

// exportSubscriberData collates the data of a subscriber including profile,
// subscriptions, campaign_views, link_clicks (if they're enabled in the config)
// and returns a formatted, indented JSON payload. Either takes a numeric id
// and an empty subUUID or takes 0 and a string subUUID.
func exportSubscriberData(id int, subUUID string, exportables map[string]bool, app *App) (models.SubscriberExportProfile, []byte, error) {
	data, err := app.core.GetSubscriberProfileForExport(id, subUUID)
	if err != nil {
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

func getQueryInts(param string, qp url.Values) ([]int, error) {
	var out []int
	if vals, ok := qp[param]; ok {
		for _, v := range vals {
			if v == "" {
				continue
			}

			listID, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			out = append(out, listID)
		}
	}

	return out, nil
}

// sendOptinConfirmationHook returns an enclosed callback that sends optin confirmation e-mails.
// This is plugged into the 'core' package to send optin confirmations when a new subscriber is
// created via `core.CreateSubscriber()`.
func sendOptinConfirmationHook(app *App) func(sub models.Subscriber, listIDs []int) (int, error) {
	return func(sub models.Subscriber, listIDs []int) (int, error) {
		lists, err := app.core.GetSubscriberLists(sub.ID, "", listIDs, nil, models.SubscriptionStatusUnconfirmed, models.ListOptinDouble)
		if err != nil {
			return 0, err
		}

		// None.
		if len(lists) == 0 {
			return 0, nil
		}

		var (
			out      = subOptin{Subscriber: sub, Lists: lists}
			qListIDs = url.Values{}
		)

		// Construct the opt-in URL with list IDs.
		for _, l := range out.Lists {
			qListIDs.Add("l", l.UUID)
		}
		out.OptinURL = fmt.Sprintf(app.constants.OptinURL, sub.UUID, qListIDs.Encode())
		out.UnsubURL = fmt.Sprintf(app.constants.UnsubURL, dummyUUID, sub.UUID)

		// Send the e-mail.
		if err := app.sendNotification([]string{sub.Email}, app.i18n.T("subscribers.optinSubject"), notifSubscriberOptin, out); err != nil {
			app.log.Printf("error sending opt-in e-mail: %s", err)
			return 0, err
		}

		return len(lists), nil
	}
}
