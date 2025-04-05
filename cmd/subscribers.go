package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/notifs"
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
	Query              string `json:"query"`
	ListIDs            []int  `json:"list_ids"`
	TargetListIDs      []int  `json:"target_list_ids"`
	SubscriberIDs      []int  `json:"ids"`
	Action             string `json:"action"`
	Status             string `json:"status"`
	SubscriptionStatus string `json:"subscription_status"`
	All                bool   `json:"all"`
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
)

// GetSubscriber handles the retrieval of a single subscriber by ID.
func (a *App) GetSubscriber(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Check if the user has access to at least one of the lists on the subscriber.
	if err := a.hasSubPerm(user, []int{id}); err != nil {
		return err
	}

	// Fetch the subscriber from the DB.
	out, err := a.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// QuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func (a *App) QuerySubscribers(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Filter list IDs by permission.
	listIDs, err := a.filterListQueryByPerm("list_id", c.QueryParams(), user)
	if err != nil {
		return err
	}

	var (
		// The "WHERE ?" bit.
		query     = sanitizeSQLExp(c.FormValue("query"))
		subStatus = c.FormValue("subscription_status")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		pg        = a.pg.NewFromURL(c.Request().URL.Query())
	)
	res, total, err := a.core.QuerySubscribers(query, listIDs, subStatus, order, orderBy, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ExportSubscribers handles querying subscribers based on an arbitrary SQL expression.
func (a *App) ExportSubscribers(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Filter list IDs by permission.
	listIDs, err := a.filterListQueryByPerm("list_id", c.QueryParams(), user)
	if err != nil {
		return err
	}

	// Export only specific subscriber IDs?
	subIDs, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Filter by subscription status
	subStatus := c.QueryParam("subscription_status")

	// Get the batched export iterator.
	query := sanitizeSQLExp(c.FormValue("query"))
	exp, err := a.core.ExportSubscribers(query, subIDs, listIDs, subStatus, a.cfg.DBBatchSize)
	if err != nil {
		return err
	}

	var (
		hdr = c.Response().Header()
		wr  = csv.NewWriter(c.Response())
	)

	hdr.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	hdr.Set("Content-type", "text/csv")
	hdr.Set(echo.HeaderContentDisposition, "attachment; filename="+"subscribers.csv")
	hdr.Set("Content-Transfer-Encoding", "binary")
	hdr.Set("Cache-Control", "no-cache")
	wr.Write([]string{"uuid", "email", "name", "attributes", "status", "created_at", "updated_at"})

loop:
	// Iterate in batches until there are no more subscribers to export.
	for {
		out, err := exp()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			break
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				a.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}

		// Flush CSV to stream after each batch.
		wr.Flush()
	}

	return nil
}

// CreateSubscriber handles the creation of a new subscriber.
func (a *App) CreateSubscriber(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get and validate fields.
	var req subimporter.SubReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate fields.
	req, err := a.importer.ValidateFields(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeManage, req.Lists)

	// Insert the subscriber into the DB.
	sub, _, err := a.core.InsertSubscriber(req.Subscriber, listIDs, nil, req.PreconfirmSubs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// UpdateSubscriber handles modification of a subscriber.
func (a *App) UpdateSubscriber(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Get and validate fields.
	req := struct {
		models.Subscriber
		Lists          []int `json:"lists"`
		PreconfirmSubs bool  `json:"preconfirm_subscriptions"`
	}{}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Sanitize and validate the email field.
	if em, err := a.importer.SanitizeEmail(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req.Email = em
	}

	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidName"))
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeManage, req.Lists)

	// Update the subscriber in the DB.
	out, _, err := a.core.UpdateSubscriberWithLists(id, req.Subscriber, listIDs, nil, req.PreconfirmSubs, true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// SubscriberSendOptin sends an optin confirmation e-mail to a subscriber.
func (a *App) SubscriberSendOptin(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Fetch the subscriber.
	out, err := a.core.GetSubscriber(id, "", "")
	if err != nil {
		return err
	}

	// Trigger the opt-in confirmation e-mail hook.
	if _, err := a.fnOptinNotify(out, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistSubscribers handles the blocklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (a *App) BlocklistSubscribers(c echo.Context) error {
	// Is it a /:id call?
	var (
		subIDs []int
		pID    = c.Param("id")
	)
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}

		subIDs = append(subIDs, id)
	} else {
		// Multiple IDs.
		var req subQueryReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(req.SubscriberIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}

		subIDs = req.SubscriberIDs
	}

	// Update the subscribers in the DB.
	if err := a.core.BlocklistSubscribers(subIDs); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (a *App) ManageSubscriberLists(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Is it an /:id call?
	var (
		pID    = c.Param("id")
		subIDs []int
	)
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.errorNoIDs"))
	}
	if len(subIDs) == 0 {
		subIDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeManage, req.TargetListIDs)

	// Run the action in the DB.
	var err error
	switch req.Action {
	case "add":
		err = a.core.AddSubscriptions(subIDs, listIDs, req.Status)
	case "remove":
		err = a.core.DeleteSubscriptions(subIDs, listIDs)
	case "unsubscribe":
		err = a.core.UnsubscribeLists(subIDs, listIDs, nil)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscribers handles subscriber deletion.
// It takes either an ID in the URI, or a list of IDs in the request body.
func (a *App) DeleteSubscribers(c echo.Context) error {
	// Is it an /:id call?
	var (
		pID    = c.Param("id")
		subIDs []int
	)
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	} else {
		// Multiple IDs.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.errorNoIDs"))
		}
		subIDs = i
	}

	// Delete the subscribers from the DB.
	if err := a.core.DeleteSubscribers(subIDs, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscribersByQuery bulk deletes based on an
// arbitrary SQL expression.
func (a *App) DeleteSubscribersByQuery(c echo.Context) error {
	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.All {
		// If the "all" flag is set, ignore any subquery that may be present.
		req.Query = ""
	} else if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}

	// Delete the subscribers from the DB.
	if err := a.core.DeleteSubscribersByQuery(req.Query, req.ListIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistSubscribersByQuery bulk blocklists subscribers
// based on an arbitrary SQL expression.
func (a *App) BlocklistSubscribersByQuery(c echo.Context) error {
	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}

	// Update the subscribers in the DB.
	if err := a.core.BlocklistSubscribersByQuery(req.Query, req.ListIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ManageSubscriberListsByQuery bulk adds/removes/unsubscribes subscribers
// from one or more lists based on an arbitrary SQL expression.
func (a *App) ManageSubscriberListsByQuery(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Filter lists against the current user's permitted lists.
	sourceListIDs := user.FilterListsByPerm(auth.PermTypeManage, req.ListIDs)
	targetListIDs := user.FilterListsByPerm(auth.PermTypeManage, req.TargetListIDs)

	// Run the action in the DB.
	var err error
	switch req.Action {
	case "add":
		err = a.core.AddSubscriptionsByQuery(req.Query, sourceListIDs, targetListIDs, req.Status, req.SubscriptionStatus)
	case "remove":
		err = a.core.DeleteSubscriptionsByQuery(req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	case "unsubscribe":
		err = a.core.UnsubscribeListsByQuery(req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscriberBounces deletes all the bounces on a subscriber.
func (a *App) DeleteSubscriberBounces(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the bounces from the DB.
	if err := a.core.DeleteSubscriberBounces(id, ""); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in a.Constants.Privacy.
func (a *App) ExportSubscriberData(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	_, b, err := a.exportSubscriberData(id, "", a.cfg.Privacy.Exportable)
	if err != nil {
		a.log.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	// Set headers to force the browser to prompt for download.
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="data.json"`)
	return c.Blob(http.StatusOK, "application/json", b)
}

// exportSubscriberData collates the data of a subscriber including profile,
// subscriptions, campaign_views, link_clicks (if they're enabled in the config)
// and returns a formatted, indented JSON payload. Either takes a numeric id
// and an empty subUUID or takes 0 and a string subUUID.
func (a *App) exportSubscriberData(id int, subUUID string, exportables map[string]bool) (models.SubscriberExportProfile, []byte, error) {
	data, err := a.core.GetSubscriberProfileForExport(id, subUUID)
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
		a.log.Printf("error marshalling subscriber export data: %v", err)
		return data, nil, err
	}

	return data, b, nil
}

// hasSubPerm checks whether the current user has permission to access the given list
// of subscriber IDs.
func (a *App) hasSubPerm(u auth.User, subIDs []int) error {
	allPerm, listIDs := u.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)

	// User has blanket get_all|manage_all permission.
	if allPerm {
		return nil
	}

	// Check whether the subscribers have the list IDs permitted to the user.
	res, err := a.core.HasSubscriberLists(subIDs, listIDs)
	if err != nil {
		return err
	}

	for id, has := range res {
		if !has {
			return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", fmt.Sprintf("subscriber: %d", id)))
		}
	}

	return nil
}

// filterListQueryByPerm filters the list IDs in the query params and returns the list IDs to which the user has access.
func (a *App) filterListQueryByPerm(param string, qp url.Values, user auth.User) ([]int, error) {
	var listIDs []int

	// If there are incoming list query params, filter them by permission.
	if qp.Has(param) {
		ids, err := getQueryInts(param, qp)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}

		listIDs = user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, ids)
	}

	// There are no incoming params. If the user doesn't have permission to get all subscribers,
	// filter by the lists they have access to.
	if len(listIDs) == 0 {
		if _, ok := user.PermissionsMap[auth.PermSubscribersGetAll]; !ok {
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

// sanitizeSQLExp does basic sanitisation on arbitrary
// SQL query expressions coming from the frontend.
func sanitizeSQLExp(q string) string {
	if len(q) == 0 {
		return ""
	}

	// Remove semicolon suffix.
	q = strings.TrimSpace(q)
	if q[len(q)-1] == ';' {
		q = q[:len(q)-1]
	}
	return q
}

// getQueryInts parses the list of given query param values into ints.
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

// makeOptinNotifyHook returns an enclosed callback that sends optin confirmation e-mails.
// This is plugged into the 'core' package to send optin confirmations when a new subscriber is
// created via `core.CreateSubscriber()`.
func makeOptinNotifyHook(unsubHeader bool, u *UrlConfig, q *models.Queries, i *i18n.I18n) func(sub models.Subscriber, listIDs []int) (int, error) {
	return func(sub models.Subscriber, listIDs []int) (int, error) {
		// Fetch double opt-in lists from the given list IDs.
		// Get the list of subscription lists where the subscriber hasn't confirmed.
		var lists = []models.List{}
		if err := q.GetSubscriberLists.Select(&lists, sub.ID, nil, pq.Array(listIDs), nil, models.SubscriptionStatusUnconfirmed, models.ListOptinDouble); err != nil {
			lo.Printf("error fetching lists for opt-in: %s", err)
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
		out.OptinURL = fmt.Sprintf(u.OptinURL, sub.UUID, qListIDs.Encode())
		out.UnsubURL = fmt.Sprintf(u.UnsubURL, dummyUUID, sub.UUID)

		// Unsub headers.
		hdr := textproto.MIMEHeader{}
		hdr.Set(models.EmailHeaderSubscriberUUID, sub.UUID)

		// Attach List-Unsubscribe headers?
		if unsubHeader {
			unsubURL := fmt.Sprintf(u.UnsubURL, dummyUUID, sub.UUID)
			hdr.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
			hdr.Set("List-Unsubscribe", `<`+unsubURL+`>`)
		}

		// Send the e-mail.
		if err := notifs.Notify([]string{sub.Email}, i.T("subscribers.optinSubject"), notifs.TplSubscriberOptin, out, hdr); err != nil {
			lo.Printf("error sending opt-in e-mail for subscriber %d (%s): %s", sub.ID, sub.UUID, err)
			return 0, err
		}

		return len(lists), nil
	}
}
