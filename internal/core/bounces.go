package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var bounceQuerySortFields = []string{"email", "campaign_name", "source", "created_at"}

// QueryBounces retrieves paginated bounce entries based on the given params.
// It also returns the total number of bounce records in the DB.
func (c *Core) QueryBounces(campID, subID int, source, orderBy, order string, offset, limit int) ([]models.Bounce, int, error) {
	if !strSliceContains(orderBy, bounceQuerySortFields) {
		orderBy = "created_at"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	out := []models.Bounce{}
	stmt := strings.ReplaceAll(c.q.QueryBounces, "%order%", orderBy+" "+order)
	if err := c.db.Select(&out, stmt, 0, campID, subID, source, offset, limit); err != nil {
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
func (c *Core) GetBounce(id int) (models.Bounce, error) {
	var out []models.Bounce
	stmt := fmt.Sprintf(c.q.QueryBounces, "id", SortAsc)
	if err := c.db.Select(&out, stmt, id, 0, 0, "", 0, 1); err != nil {
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
	_, err := c.q.RecordBounce.Exec(b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta,
		b.CreatedAt,
		c.constants.MaxBounceCount,
		c.constants.BounceAction)

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

// DeleteBounce deletes a list.
func (c *Core) DeleteBounce(id int) error {
	return c.DeleteBounces([]int{id})
}

// DeleteBounces deletes multiple lists.
func (c *Core) DeleteBounces(ids []int) error {
	if _, err := c.q.DeleteBounces.Exec(pq.Array(ids)); err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
