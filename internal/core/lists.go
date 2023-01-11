package core

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetLists gets all lists optionally filtered by type.
func (c *Core) GetLists(ctx context.Context, typ string) ([]models.List, error) {
	out := []models.List{}

	// xyz, _ := sqlx.PreparexContext(ctx, c.db, "WITH subs AS ( SELECT subscriber_id, JSON_AGG( ROW_TO_JSON( (SELECT l FROM (SELECT subscriber_lists.status AS subscription_status, lists.*) l) ) ) AS lists FROM lists LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id) WHERE subscriber_lists.subscriber_id = ANY($1) GROUP BY subscriber_id ) SELECT id as subscriber_id, COALESCE(s.lists, '[]') AS lists FROM (SELECT id FROM UNNEST($1) AS id) x LEFT JOIN subs AS s ON (s.subscriber_id = id) ORDER BY ARRAY_POSITION($1, id);")
	if err := c.q.GetLists.SelectContext(ctx, &out, typ, "id"); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	// c.q.GetLists.Stmt.Close()

	// Replace null tags.
	for i, l := range out {
		if l.Tags == nil {
			out[i].Tags = []string{}
		}

		// Total counts.
		for _, c := range l.SubscriberCounts {
			out[i].SubscriberCount += c
		}
	}

	return out, nil
}

// QueryLists gets multiple lists based on multiple query params. Along with the  paginated and sliced
// results, the total number of lists in the DB is returned.
func (c *Core) QueryLists(ctx context.Context, searchStr, orderBy, order string, offset, limit int) ([]models.List, int, error) {
	out := []models.List{}

	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QueryLists)

	if err := c.db.SelectContext(ctx, &out, stmt, 0, "", queryStr, offset, limit); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total

		// Replace null tags.
		for i, l := range out {
			if l.Tags == nil {
				out[i].Tags = []string{}
			}

			// Total counts.
			for _, c := range l.SubscriberCounts {
				out[i].SubscriberCount += c
			}
		}
	}

	return out, total, nil
}

// GetList gets a list by its ID or UUID.
func (c *Core) GetList(ctx context.Context, id int, uuid string) (models.List, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var res []models.List
	queryStr, stmt := makeSearchQuery("", "", "", c.q.QueryLists)
	if err := c.db.SelectContext(ctx, &res, stmt, id, uu, queryStr, 0, 1); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	if len(res) == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	out := res[0]
	if out.Tags == nil {
		out.Tags = []string{}
	}
	// Total counts.
	for _, c := range out.SubscriberCounts {
		out.SubscriberCount += c
	}

	return out, nil
}

// GetListsByOptin returns lists by optin type.
func (c *Core) GetListsByOptin(ctx context.Context, ids []int, optinType string) ([]models.List, error) {
	out := []models.List{}
	if err := c.q.GetListsByOptin.SelectContext(ctx, &out, optinType, pq.Array(ids), nil); err != nil {
		c.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateList creates a new list.
func (c *Core) CreateList(ctx context.Context, l models.List) (models.List, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	if l.Type == "" {
		l.Type = models.ListTypePrivate
	}
	if l.Optin == "" {
		l.Optin = models.ListOptinSingle
	}

	// Insert and read ID.
	var newID int
	l.UUID = uu.String()
	if err := c.q.CreateList.GetContext(ctx, &newID, l.UUID, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description); err != nil {
		c.log.Printf("error creating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.GetList(ctx, newID, "")
}

// UpdateList updates a given list.
func (c *Core) UpdateList(ctx context.Context, id int, l models.List) (models.List, error) {
	res, err := c.q.UpdateList.ExecContext(ctx, id, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description)
	if err != nil {
		c.log.Printf("error updating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	return c.GetList(ctx, id, "")
}

// DeleteList deletes a list.
func (c *Core) DeleteList(ctx context.Context, id int) error {
	return c.DeleteLists(ctx, []int{id})
}

// DeleteLists deletes multiple lists.
func (c *Core) DeleteLists(ctx context.Context, ids []int) error {
	if _, err := c.q.DeleteLists.ExecContext(ctx, pq.Array(ids)); err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
