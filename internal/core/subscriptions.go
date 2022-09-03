package core

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// AddSubscriptions adds list subscriptions to subscribers.
func (c *Core) AddSubscriptions(subIDs, listIDs []int, status string) error {
	if _, err := c.q.AddSubscribersToLists.Exec(pq.Array(subIDs), pq.Array(listIDs), status); err != nil {
		c.log.Printf("error adding subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// AddSubscriptionsByQuery adds list subscriptions to subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) AddSubscriptionsByQuery(query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(sanitizeSQLExp(query), c.q.AddSubscribersToListsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error adding subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscriptions delete list subscriptions from subscribers.
func (c *Core) DeleteSubscriptions(subIDs, listIDs []int) error {
	if _, err := c.q.DeleteSubscriptions.Exec(pq.Array(subIDs), pq.Array(listIDs)); err != nil {
		c.log.Printf("error deleting subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))

	}

	return nil
}

// DeleteSubscriptionsByQuery deletes list subscriptions from subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteSubscriptionsByQuery(query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(sanitizeSQLExp(query), c.q.DeleteSubscriptionsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error deleting subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// UnsubscribeLists sets list subscriptions to 'unsubscribed'.
func (c *Core) UnsubscribeLists(subIDs, listIDs []int) error {
	if _, err := c.q.UnsubscribeSubscribersFromLists.Exec(pq.Array(subIDs), pq.Array(listIDs)); err != nil {
		c.log.Printf("error unsubscribing from lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// UnsubscribeListsByQuery sets list subscriptions to 'ubsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) UnsubscribeListsByQuery(query string, sourceListIDs, targetListIDs []int) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.q.ExecSubQueryTpl(sanitizeSQLExp(query), c.q.UnsubscribeSubscribersFromListsByQuery, sourceListIDs, c.db, pq.Array(targetListIDs))
	if err != nil {
		c.log.Printf("error unsubscriging from lists by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteUnconfirmedSubscriptions sets list subscriptions to 'ubsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteUnconfirmedSubscriptions(beforeDate time.Time) (int, error) {
	res, err := c.q.DeleteUnconfirmedSubscriptions.Exec(beforeDate)
	if err != nil {
		c.log.Printf("error deleting unconfirmed subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	n, _ := res.RowsAffected()
	return int(n), nil
}
