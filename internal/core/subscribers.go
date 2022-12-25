package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var (
	subQuerySortFields = []string{"email", "name", "created_at", "updated_at"}
)

// GetSubscriber fetches a subscriber by one of the given params.
func (c *Core) GetSubscriber(id int, uuid, email string) (models.Subscriber, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out models.Subscribers
	if err := c.q.GetSubscriber.Select(&out, id, uu, email); err != nil {
		c.log.Printf("error fetching subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return models.Subscriber{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name",
				fmt.Sprintf("{globals.terms.subscriber} (%d: %s%s)", id, uuid, email)))
	}
	if err := out.LoadLists(c.q.GetSubscriberListsLazy); err != nil {
		c.log.Printf("error loading subscriber lists: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out[0], nil
}

// GetSubscribersByEmail fetches a subscriber by one of the given params.
func (c *Core) GetSubscribersByEmail(emails []string) (models.Subscribers, error) {
	var out models.Subscribers

	if err := c.q.GetSubscribersByEmails.Select(&out, pq.Array(emails)); err != nil {
		c.log.Printf("error fetching subscriber: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("campaigns.noKnownSubsToTest"))
	}

	if err := out.LoadLists(c.q.GetSubscriberListsLazy); err != nil {
		c.log.Printf("error loading subscriber lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// QuerySubscribers queries and returns paginated subscrribers based on the given params including the total count.
func (c *Core) QuerySubscribers(query string, listIDs []int, order, orderBy string, offset, limit int) (models.Subscribers, int, error) {
	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "subscribers.id"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	// Required for pq.Array()
	if listIDs == nil {
		listIDs = []int{}
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	stmt := fmt.Sprintf(c.q.QuerySubscribersCount, cond)
	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.log.Printf("error preparing subscriber query: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Execute the readonly query and get the count of results.
	total := 0
	if err := tx.Get(&total, stmt, pq.Array(listIDs)); err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// No results.
	if total == 0 {
		return models.Subscribers{}, 0, nil
	}

	// Run the query again and fetch the actual data. stmt is the raw SQL query.
	var out models.Subscribers
	stmt = strings.ReplaceAll(c.q.QuerySubscribers, "%query%", cond)
	stmt = strings.ReplaceAll(stmt, "%order%", orderBy+" "+order)
	if err := tx.Select(&out, stmt, pq.Array(listIDs), offset, limit); err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.LoadLists(c.q.GetSubscriberListsLazy); err != nil {
		c.log.Printf("error fetching subscriber lists: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return out, total, nil
}

// GetSubscriberLists returns a subscriber's lists based on the given conditions.
func (c *Core) GetSubscriberLists(subID int, uuid string, listIDs []int, listUUIDs []string, subStatus string, listType string) ([]models.List, error) {
	if listIDs == nil {
		listIDs = []int{}
	}
	if listUUIDs == nil {
		listUUIDs = []string{}
	}

	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	// Fetch double opt-in lists from the given list IDs.
	// Get the list of subscription lists where the subscriber hasn't confirmed.
	out := []models.List{}
	if err := c.q.GetSubscriberLists.Select(&out, subID, uu, pq.Array(listIDs), pq.Array(listUUIDs), subStatus, listType); err != nil {
		c.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return nil, err
	}

	return out, nil
}

// GetSubscriberProfileForExport returns the subscriber's profile data as a JSON exportable.
// Get the subscriber's data. A single query that gets the profile, list subscriptions, campaign views,
// and link clicks. Names of private lists are replaced with "Private list".
func (c *Core) GetSubscriberProfileForExport(id int, uuid string) (models.SubscriberExportProfile, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out models.SubscriberExportProfile
	if err := c.q.ExportSubscriberData.Get(&out, id, uu); err != nil {
		c.log.Printf("error fetching subscriber export data: %v", err)

		return models.SubscriberExportProfile{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return out, nil
}

// ExportSubscribers returns an iterator function that provides lists of subscribers based
// on the given criteria in an exportable form. The iterator function returned can be called
// repeatedly until there are nil subscribers. It's an iterator because exports can be extremely
// large and may have to be fetched in batches from the DB and streamed somewhere.
func (c *Core) ExportSubscribers(query string, subIDs, listIDs []int, batchSize int) (func() ([]models.SubscriberExport, error), error) {
	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	stmt := fmt.Sprintf(c.q.QuerySubscribersForExport, cond)
	stmt = strings.ReplaceAll(c.q.QuerySubscribersForExport, "%query%", cond)

	// Verify that the arbitrary SQL search expression is read only.
	if cond != "" {
		tx, err := c.db.Unsafe().BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			c.log.Printf("error preparing subscriber query: %v", err)
			return nil, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
		defer tx.Rollback()

		if _, err := tx.Query(stmt, nil, 0, nil, 1); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
	}

	if subIDs == nil {
		subIDs = []int{}
	}
	if listIDs == nil {
		listIDs = []int{}
	}

	// Prepare the actual query statement.
	tx, err := c.db.Preparex(stmt)
	if err != nil {
		c.log.Printf("error preparing subscriber query: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}

	id := 0
	return func() ([]models.SubscriberExport, error) {
		var out []models.SubscriberExport
		if err := tx.Select(&out, pq.Array(listIDs), id, pq.Array(subIDs), batchSize); err != nil {
			c.log.Printf("error exporting subscribers by query: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
		if len(out) == 0 {
			return nil, nil
		}

		id = out[len(out)-1].ID
		return out, nil
	}, nil
}

// InsertSubscriber inserts a subscriber and returns the ID. The first bool indicates if
// it was a new subscriber, and the second bool indicates if the subscriber was sent an optin confirmation.
// bool = optinSent?
func (c *Core) InsertSubscriber(sub models.Subscriber, listIDs []int, listUUIDs []string, preconfirm bool) (models.Subscriber, bool, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}
	sub.UUID = uu.String()

	subStatus := models.SubscriptionStatusUnconfirmed
	if preconfirm {
		subStatus = models.SubscriptionStatusConfirmed
	}
	if sub.Status == "" {
		sub.Status = models.UserStatusEnabled
	}

	// For pq.Array()
	if listIDs == nil {
		listIDs = []int{}
	}
	if listUUIDs == nil {
		listUUIDs = []string{}
	}

	if err = c.q.InsertSubscriber.Get(&sub.ID,
		sub.UUID,
		sub.Email,
		strings.TrimSpace(sub.Name),
		sub.Status,
		sub.Attribs,
		pq.Array(listIDs),
		pq.Array(listUUIDs),
		subStatus); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "subscribers_email_key" {
			return models.Subscriber{}, false, echo.NewHTTPError(http.StatusConflict,
				c.i18n.T("subscribers.emailExists"))
		} else {
			// return sub.Subscriber, errSubscriberExists
			c.log.Printf("error inserting subscriber: %v", err)
			return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorCreating",
					"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
		}
	}

	// Fetch the subscriber'out full data. If the subscriber already existed and wasn't
	// created, the id will be empty. Fetch the details by e-mail then.
	out, err := c.GetSubscriber(sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, false, err
	}

	hasOptin := false
	if !preconfirm && c.constants.SendOptinConfirmation {
		// Send a confirmation e-mail (if there are any double opt-in lists).
		num, _ := c.h.SendOptinConfirmation(out, listIDs)
		hasOptin = num > 0
	}

	return out, hasOptin, nil
}

// UpdateSubscriber updates a subscriber's properties.
func (c *Core) UpdateSubscriber(id int, sub models.Subscriber) (models.Subscriber, error) {
	// Format raw JSON attributes.
	attribs := []byte("{}")
	if len(sub.Attribs) > 0 {
		if b, err := json.Marshal(sub.Attribs); err != nil {
			return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		} else {
			attribs = b
		}
	}

	_, err := c.q.UpdateSubscriber.Exec(id,
		sub.Email,
		strings.TrimSpace(sub.Name),
		sub.Status,
		json.RawMessage(attribs),
	)
	if err != nil {
		c.log.Printf("error updating subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	out, err := c.GetSubscriber(sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, err
	}

	return out, nil
}

// UpdateSubscriberWithLists updates a subscriber's properties.
// If deleteLists is set to true, all existing subscriptions are deleted and only
// the ones provided are added or retained.
func (c *Core) UpdateSubscriberWithLists(id int, sub models.Subscriber, listIDs []int, listUUIDs []string, preconfirm, deleteLists bool) (models.Subscriber, error) {
	subStatus := models.SubscriptionStatusUnconfirmed
	if preconfirm {
		subStatus = models.SubscriptionStatusConfirmed
	}

	// Format raw JSON attributes.
	attribs := []byte("{}")
	if len(sub.Attribs) > 0 {
		if b, err := json.Marshal(sub.Attribs); err != nil {
			return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		} else {
			attribs = b
		}
	}

	_, err := c.q.UpdateSubscriberWithLists.Exec(id,
		sub.Email,
		strings.TrimSpace(sub.Name),
		sub.Status,
		json.RawMessage(attribs),
		pq.Array(listIDs),
		pq.Array(listUUIDs),
		subStatus,
		deleteLists)
	if err != nil {
		c.log.Printf("error updating subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	out, err := c.GetSubscriber(sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, err
	}

	return out, nil
}

// BlocklistSubscribers blocklists the given list of subscribers.
func (c *Core) BlocklistSubscribers(subIDs []int) error {
	if _, err := c.q.BlocklistSubscribers.Exec(pq.Array(subIDs)); err != nil {
		c.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
	}

	return nil
}

// BlocklistSubscribersByQuery blocklists the given list of subscribers.
func (c *Core) BlocklistSubscribersByQuery(query string, listIDs []int) error {
	if err := c.q.ExecSubQueryTpl(sanitizeSQLExp(query), c.q.BlocklistSubscribersByQuery, listIDs, c.db); err != nil {
		c.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscribers deletes the given list of subscribers.
func (c *Core) DeleteSubscribers(subIDs []int, subUUIDs []string) error {
	if subIDs == nil {
		subIDs = []int{}
	}
	if subUUIDs == nil {
		subUUIDs = []string{}
	}

	if _, err := c.q.DeleteSubscribers.Exec(pq.Array(subIDs), pq.Array(subUUIDs)); err != nil {
		c.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscribersByQuery deletes subscribers by a given arbitrary query expression.
func (c *Core) DeleteSubscribersByQuery(query string, listIDs []int) error {
	err := c.q.ExecSubQueryTpl(sanitizeSQLExp(query), c.q.DeleteSubscribersByQuery, listIDs, c.db)
	if err != nil {
		c.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return err
}

// UnsubscribeByCampaign unsubscibers a given subscriber from lists in a given campaign.
func (c *Core) UnsubscribeByCampaign(subUUID, campUUID string, blocklist bool) error {
	if _, err := c.q.UnsubscribeByCampaign.Exec(campUUID, subUUID, blocklist); err != nil {
		c.log.Printf("error unsubscribing: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// ConfirmOptionSubscription confirms a subscriber's optin subscription.
func (c *Core) ConfirmOptionSubscription(subUUID string, listUUIDs []string) error {
	if _, err := c.q.ConfirmSubscriptionOptin.Exec(subUUID, pq.Array(listUUIDs)); err != nil {
		c.log.Printf("error confirming subscription: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscriberBounces deletes the given list of subscribers.
func (c *Core) DeleteSubscriberBounces(id int, uuid string) error {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	if _, err := c.q.DeleteBouncesBySubscriber.Exec(id, uu); err != nil {
		c.log.Printf("error deleting bounces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.bounces}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteOrphanSubscribers deletes orphan subscriber records (subscribers without lists).
func (c *Core) DeleteOrphanSubscribers() (int, error) {
	res, err := c.q.DeleteOrphanSubscribers.Exec()
	if err != nil {
		c.log.Printf("error deleting orphan subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteBlocklistedSubscribers deletes blocklisted subscribers.
func (c *Core) DeleteBlocklistedSubscribers() (int, error) {
	res, err := c.q.DeleteBlocklistedSubscribers.Exec()
	if err != nil {
		c.log.Printf("error deleting blocklisted subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	n, _ := res.RowsAffected()
	return int(n), nil
}
