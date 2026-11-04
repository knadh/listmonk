package core

import (
	"net/http"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
	"github.com/lib/pq"
)

var bounceQuerySortFields = []string{"email", "campaign_name", "source", models.FieldCreatedAt, "type"}

// QueryBounces retrieves paginated bounce entries based on the given params.
// It also returns the total number of bounce records within the permitted lists.
func (c *Core) QueryBounces(campID, subID int, source, typ, orderBy, order string, offset, limit int, allLists bool, listIDs []int) ([]models.Bounce, int, error) {
	// Performance optimization: fast check for common default order field without loop,
	// and sanitize order parameters to prevent arbitrary SQL injection while preserving tie-breaker.
	cleanOrderBy := models.FieldCreatedAt
	for _, f := range bounceQuerySortFields {
		if f == orderBy {
			cleanOrderBy = f
			break
		}
	}
	cleanOrder := models.OrderDesc
	if order == models.OrderAsc {
		cleanOrder = models.OrderAsc
	}

	if cleanOrderBy == "campaign_name" {
		cleanOrderBy = "campaigns.name"
	}

	out := []models.Bounce{}
	stmt := strings.ReplaceAll(c.q.QueryBounces, "%order%", cleanOrderBy+" "+cleanOrder+", bounces.id "+cleanOrder)
	if err := c.db.Select(&out, stmt, 0, campID, subID, source, typ, offset, limit, allLists, pq.Array(listIDs)); err != nil {
		c.log.Printf("error fetching bounces: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.bounce}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetBounce retrieves bounce entries based on the given params.
func (c *Core) GetBounce(id int, allLists bool, listIDs []int) (models.Bounce, error) {
	var out []models.Bounce
	stmt := strings.ReplaceAll(c.q.QueryBounces, "%order%", "id "+models.OrderAsc)
	if err := c.db.Select(&out, stmt, id, 0, 0, "", "", 0, 1, allLists, pq.Array(listIDs)); err != nil {
		c.log.Printf("error fetching bounces: %v", err)
		return models.Bounce{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.bounce}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Bounce{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.bounce}"))

	}

	return out[0], nil
}

// RecordBounce records a new bounce.
func (c *Core) RecordBounce(b models.Bounce) error {
	action, ok := c.consts.BounceActions[b.Type]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.invalidData")+": "+b.Type)
	}

	_, err := c.q.RecordBounce.Exec(b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta,
		b.CreatedAt,
		action.Count,
		action.Action)

	if err != nil {
		// Ignore the error if it complained of no subscriber.
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "subscriber_id" {
			c.log.Printf("bounced subscriber (%s / %s) not found", b.SubscriberUUID, b.Email)
			return nil
		}

		c.log.Printf("error recording bounce: %v", err)
	}

	return err
}

// BlocklistBouncedSubscribers blocklists subscribers matching bounce IDs or filters.
func (c *Core) BlocklistBouncedSubscribers(ids []int, campID int, source, typ string, allLists bool, listIDs []int) error {
	if _, err := c.q.BlocklistBouncedSubscribers.Exec(pq.Array(ids), campID, source, typ, allLists, pq.Array(listIDs)); err != nil {
		c.log.Printf("error blocklisting bounced subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
	}

	return nil
}

// DeleteBounce deletes a bounce record.
func (c *Core) DeleteBounce(id int, allLists bool, listIDs []int) error {
	return c.DeleteBounces([]int{id}, false, 0, "", "", allLists, listIDs)
}

// DeleteBounces deletes bounce records by IDs or filters.
func (c *Core) DeleteBounces(ids []int, all bool, campID int, source, typ string, allLists bool, listIDs []int) error {
	if _, err := c.q.DeleteBounces.Exec(pq.Array(ids), all, campID, source, typ, allLists, pq.Array(listIDs)); err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
