package core

import (
	"context"
	"net/http"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetSubscriptions retrieves the subscriptions for a subscriber.
func (c *Core) GetSubscriptions(ctx context.Context, subID int, subUUID string, allLists bool) ([]models.Subscription, error) {
	var out []models.Subscription
	err := c.q.GetSubscriptions.SelectContext(ctx, &out, subID, subUUID, allLists)
	if err != nil {
		c.log.Printf("error getting subscriptions: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return out, err
}

// AddSubscriptions adds list subscriptions to subscribers.
func (c *Core) AddSubscriptions(ctx context.Context, subIDs, listIDs []int, status string) error {
	if _, err := c.q.AddSubscribersToLists.ExecContext(ctx, pq.Array(subIDs), pq.Array(listIDs), status); err != nil {
		c.log.Printf("error adding subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// AddSubscriptionsByQuery adds list subscriptions to subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) AddSubscriptionsByQuery(ctx context.Context, query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(ctx, sanitizeSQLExp(query), c.q.AddSubscribersToListsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error adding subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscriptions delete list subscriptions from subscribers.
func (c *Core) DeleteSubscriptions(ctx context.Context, subIDs, listIDs []int) error {
	if _, err := c.q.DeleteSubscriptions.ExecContext(ctx, pq.Array(subIDs), pq.Array(listIDs)); err != nil {
		c.log.Printf("error deleting subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))

	}

	return nil
}

// DeleteSubscriptionsByQuery deletes list subscriptions from subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteSubscriptionsByQuery(ctx context.Context, query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(ctx, sanitizeSQLExp(query), c.q.DeleteSubscriptionsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error deleting subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// UnsubscribeLists sets list subscriptions to 'unsubscribed'.
func (c *Core) UnsubscribeLists(ctx context.Context, subIDs, listIDs []int, listUUIDs []string) error {
	if _, err := c.q.UnsubscribeSubscribersFromLists.ExecContext(ctx, pq.Array(subIDs), pq.Array(listIDs), pq.StringArray(listUUIDs)); err != nil {
		c.log.Printf("error unsubscribing from lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// UnsubscribeListsByQuery sets list subscriptions to 'ubsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) UnsubscribeListsByQuery(ctx context.Context, query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(ctx, sanitizeSQLExp(query), c.q.UnsubscribeSubscribersFromListsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error unsubscriging from lists by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteUnconfirmedSubscriptions sets list subscriptions to 'ubsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteUnconfirmedSubscriptions(ctx context.Context, beforeDate time.Time) (int, error) {
	res, err := c.q.DeleteUnconfirmedSubscriptions.ExecContext(ctx, beforeDate)
	if err != nil {
		c.log.Printf("error deleting unconfirmed subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	n, _ := res.RowsAffected()
	return int(n), nil
}
