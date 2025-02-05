package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const (
	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// subQueryReq is a "catch all" struct for reading various
// subscriber related requests.
type subQueryReq struct {
	Query              string `json:"query"`
	ListIDs            []int  `json:"list_ids"`
	TargetListIDs      []int  `json:"target_list_ids"`
	SubscriberIDs      []int  `json:"ids"`
	Action             string `json:"action"`
	Status             string `json:"status"`
	SubscriptionStatus string `json:"subscription_status"`
	All                bool   `json:"all"`
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
func (h *Handler) handleGetSubscriber(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
		user  = c.Get(auth.UserKey).(models.User)
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if err := hasSubPerm(user, []int{id}, h.app); err != nil {
		return err
	}

	out, err := h.app.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func (h *Handler) handleQuerySubscribers(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)
		pg   = h.app.paginator.NewFromURL(c.Request().URL.Query())

		// The "WHERE ?" bit.
		query     = sanitizeSQLExp(c.FormValue("query"))
		subStatus = c.FormValue("subscription_status")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		out       models.PageResults
	)

	// Filter list IDs by permission.
	listIDs, err := filterListQeryByPerm(c.QueryParams(), user, h.app)
	if err != nil {
		return err
	}

	res, total, err := h.app.core.QuerySubscribers(query, listIDs, subStatus, order, orderBy, pg.Offset, pg.Limit)
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
func (h *Handler) handleExportSubscribers(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)

		// The "WHERE ?" bit.
		query = sanitizeSQLExp(c.FormValue("query"))
	)

	// Filter list IDs by permission.
	listIDs, err := filterListQeryByPerm(c.QueryParams(), user, h.app)
	if err != nil {
		return err
	}

	// Export only specific subscriber IDs?
	subIDs, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Filter by subscription status
	subStatus := c.QueryParam("subscription_status")

	// Get the batched export iterator.
	exp, err := h.app.core.ExportSubscribers(query, subIDs, listIDs, subStatus, h.app.constants.DBBatchSize)
	if err != nil {
		return err
	}

	var (
		header = c.Response().Header()
		wr     = csv.NewWriter(c.Response())
	)

	header.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	header.Set("Content-type", "text/csv")
	header.Set(echo.HeaderContentDisposition, "attachment; filename="+"subscribers.csv")
	header.Set("Content-Transfer-Encoding", "binary")
	header.Set("Cache-Control", "no-cache")
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
				h.app.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}

		// Flush CSV to stream after each batch.
		wr.Flush()
	}

	return nil
}

// handleCreateSubscriber handles the creation of a new subscriber.
func (h *Handler) handleCreateSubscriber(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)

		req subimporter.SubReq
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate fields.
	req, err := h.app.importer.ValidateFields(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(req.Lists, false, true)

	// Insert the subscriber into the DB.
	sub, _, err := h.app.core.InsertSubscriber(req.Subscriber, listIDs, nil, req.PreconfirmSubs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleUpdateSubscriber handles modification of a subscriber.
func (h *Handler) handleUpdateSubscriber(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)

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
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if em, err := h.app.importer.SanitizeEmail(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req.Email = em
	}

	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.invalidName"))
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(req.Lists, false, true)

	out, _, err := h.app.core.UpdateSubscriberWithLists(id, req.Subscriber, listIDs, nil, req.PreconfirmSubs, true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleSubscriberSendOptin sends an optin confirmation e-mail to a subscriber.
func (h *Handler) handleSubscriberSendOptin(c echo.Context) error {
	var id, _ = strconv.Atoi(c.Param("id"))

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Fetch the subscriber.
	out, err := h.app.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	if _, err := sendOptinConfirmationHook(h.app)(out, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribers handles the blocklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (h *Handler) handleBlocklistSubscribers(c echo.Context) error {
	var (
		pID    = c.Param("id")
		subIDs []int
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
		}

		subIDs = append(subIDs, id)
	} else {
		// Multiple IDs.
		var req subQueryReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(req.SubscriberIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}

		subIDs = req.SubscriberIDs
	}

	if err := h.app.core.BlocklistSubscribers(subIDs); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (h *Handler) handleManageSubscriberLists(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)

		pID    = c.Param("id")
		subIDs []int
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.errorNoIDs"))
	}
	if len(subIDs) == 0 {
		subIDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(req.TargetListIDs, false, true)

	// Action.
	var err error
	switch req.Action {
	case "add":
		err = h.app.core.AddSubscriptions(subIDs, listIDs, req.Status)
	case "remove":
		err = h.app.core.DeleteSubscriptions(subIDs, listIDs)
	case "unsubscribe":
		err = h.app.core.UnsubscribeLists(subIDs, listIDs, nil)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribers handles subscriber deletion.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (h *Handler) handleDeleteSubscribers(c echo.Context) error {
	var (
		pID    = c.Param("id")
		subIDs []int
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	} else {
		// Multiple IDs.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("subscribers.errorNoIDs", "error", err.Error()))
		}
		subIDs = i
	}

	if err := h.app.core.DeleteSubscribers(subIDs, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribersByQuery bulk deletes based on an
// arbitrary SQL expression.
func (h *Handler) handleDeleteSubscribersByQuery(c echo.Context) error {
	var req subQueryReq

	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.All {
		req.Query = ""
	} else if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}

	if err := h.app.core.DeleteSubscribersByQuery(req.Query, req.ListIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribersByQuery bulk blocklists subscribers
// based on an arbitrary SQL expression.
func (h *Handler) handleBlocklistSubscribersByQuery(c echo.Context) error {
	var req subQueryReq

	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}

	if err := h.app.core.BlocklistSubscribersByQuery(req.Query, req.ListIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberListsByQuery bulk adds/removes/unsubscribes subscribers
// from one or more lists based on an arbitrary SQL expression.
func (h *Handler) handleManageSubscriberListsByQuery(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)

		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Filter lists against the current user's permitted lists.
	sourceListIDs := user.FilterListsByPerm(req.ListIDs, false, true)
	targetListIDs := user.FilterListsByPerm(req.TargetListIDs, false, true)

	// Action.
	var err error
	switch req.Action {
	case "add":
		err = h.app.core.AddSubscriptionsByQuery(req.Query, sourceListIDs, targetListIDs, req.Status, req.SubscriptionStatus)
	case "remove":
		err = h.app.core.DeleteSubscriptionsByQuery(req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	case "unsubscribe":
		err = h.app.core.UnsubscribeListsByQuery(req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscriberBounces deletes all the bounces on a subscriber.
func (h *Handler) handleDeleteSubscriberBounces(c echo.Context) error {
	var pID = c.Param("id")

	id, _ := strconv.Atoi(pID)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if err := h.app.core.DeleteSubscriberBounces(id, ""); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in app.Constants.Privacy.
func (h *Handler) handleExportSubscriberData(c echo.Context) error {
	var (
		pID = c.Param("id")
	)

	id, _ := strconv.Atoi(pID)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	_, b, err := exportSubscriberData(id, "", h.app.constants.Privacy.Exportable, h.app)
	if err != nil {
		h.app.log.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			h.app.i18n.Ts("globals.messages.errorFetching",
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

		// Unsub headers.
		h := textproto.MIMEHeader{}
		h.Set(models.EmailHeaderSubscriberUUID, sub.UUID)

		// Attach List-Unsubscribe headers?
		if app.constants.Privacy.UnsubHeader {
			unsubURL := fmt.Sprintf(app.constants.UnsubURL, dummyUUID, sub.UUID)
			h.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
			h.Set("List-Unsubscribe", `<`+unsubURL+`>`)
		}

		// Send the e-mail.
		if err := app.sendNotification([]string{sub.Email}, app.i18n.T("subscribers.optinSubject"), notifSubscriberOptin, out, h); err != nil {
			app.log.Printf("error sending opt-in e-mail for subscriber %d (%s): %s", sub.ID, sub.UUID, err)
			return 0, err
		}

		return len(lists), nil
	}
}

// hasSubPerm checks whether the current user has permission to access the given list
// of subscriber IDs.
func hasSubPerm(u models.User, subIDs []int, app *App) error {
	if u.UserRoleID == auth.SuperAdminRoleID {
		return nil
	}

	if _, ok := u.PermissionsMap[models.PermSubscribersGetAll]; ok {
		return nil
	}

	res, err := app.core.HasSubscriberLists(subIDs, u.GetListIDs)
	if err != nil {
		return err
	}

	for id, has := range res {
		if !has {
			return echo.NewHTTPError(http.StatusForbidden, app.i18n.Ts("globals.messages.permissionDenied", "name", fmt.Sprintf("subscriber: %d", id)))
		}
	}

	return nil
}

func filterListQeryByPerm(qp url.Values, user models.User, app *App) ([]int, error) {
	var listIDs []int

	// If there are incoming list query params, filter them by permission.
	if qp.Has("list_id") {
		ids, err := getQueryInts("list_id", qp)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}

		listIDs = user.FilterListsByPerm(ids, true, true)
	}

	// There are no incoming params. If the user doesn't have permission to get all subscribers,
	// filter by the lists they have access to.
	if len(listIDs) == 0 {
		if _, ok := user.PermissionsMap[models.PermSubscribersGetAll]; !ok {
			if len(user.GetListIDs) > 0 {
				listIDs = user.GetListIDs
			} else {
				// User doesn't have access to any lists.
				listIDs = []int{-1}
			}
		}
	}

	return listIDs, nil
}
