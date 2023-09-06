package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetLists gets all lists optionally filtered by type.
func (c *Core) GetLists(typ string) ([]models.List, error) {
	out := []models.List{}

	if err := c.q.GetLists.Select(&out, typ, "id"); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

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
func (c *Core) QueryLists(query, orderBy, order string, offset, limit int) ([]models.List, int, error) {
	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " WHERE " + query
	}

	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "lists.id"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	stmt := fmt.Sprintf(c.q.QueryListsCount, cond)
	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.log.Printf("error preparing subscriber query: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("lists.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Execute the readonly query and get the count of results.
	total := 0
	if err := tx.Get(&total, stmt); err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	// No results.
	if total == 0 {
		return []models.List{}, 0, nil
	}

	// Run the query again and fetch the actual data. stmt is the raw SQL query.
	var out []models.List
	stmt = strings.ReplaceAll(c.q.QueryLists, "%query%", cond)
	stmt = strings.ReplaceAll(stmt, "%order%", orderBy+" "+order)
	if err := tx.Select(&out, stmt, offset, limit); err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out, total, nil
}

// GetList gets a list by its ID or UUID.
func (c *Core) GetList(id int, uuid string) (models.List, error) {
	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.log.Printf("error preparing subscriber query: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("lists.errorPreparingQuery", "error", pqErrMsg(err)))
	}

	orderBy := "lists.id"
	order := SortDesc

	var res []models.List
	stmt := strings.ReplaceAll(c.q.QueryLists, "%query%", "")
	stmt = strings.ReplaceAll(stmt, "%order%", orderBy+" "+order)
	if err := tx.Select(&res, stmt, 0, 1); err != nil {
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
func (c *Core) GetListsByOptin(ids []int, optinType string) ([]models.List, error) {
	out := []models.List{}
	if err := c.q.GetListsByOptin.Select(&out, optinType, pq.Array(ids), nil); err != nil {
		c.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateList creates a new list.
func (c *Core) CreateList(l models.List) (models.List, error) {
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
	if err := c.q.CreateList.Get(&newID, l.UUID, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description); err != nil {
		c.log.Printf("error creating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.GetList(newID, "")
}

// UpdateList updates a given list.
func (c *Core) UpdateList(id int, l models.List) (models.List, error) {
	res, err := c.q.UpdateList.Exec(id, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description)
	if err != nil {
		c.log.Printf("error updating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	return c.GetList(id, "")
}

// DeleteList deletes a list.
func (c *Core) DeleteList(id int) error {
	return c.DeleteLists([]int{id})
}

// DeleteLists deletes multiple lists.
func (c *Core) DeleteLists(ids []int) error {
	if _, err := c.q.DeleteLists.Exec(pq.Array(ids)); err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
