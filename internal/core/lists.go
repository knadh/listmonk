package core

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetLists gets all lists optionally filtered by type.
func (c *Core) GetLists(typ string) ([]models.List, error) {
	out := []models.List{}

	// TODO: remove orderBy
	if err := c.q.GetLists.Select(&out, typ, "id"); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// QueryLists gets multiple lists based on multiple query params.
func (c *Core) QueryLists(searchStr, orderBy, order string, offset, limit int) ([]models.List, error) {
	out := []models.List{}

	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QueryLists)

	if err := c.db.Select(&out, stmt, 0, "", queryStr, offset, limit); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetList gets a list by its ID or UUID.
func (c *Core) GetList(id int, uuid string) (models.List, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out []models.List
	queryStr, stmt := makeSearchQuery("", "", "", c.q.QueryLists)
	if err := c.db.Select(&out, stmt, id, uu, queryStr, 0, 1); err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	if len(out) == 1 {
		return out[0], nil
	}

	return models.List{}, nil
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
	if err := c.q.CreateList.Get(&newID, l.UUID, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags))); err != nil {
		c.log.Printf("error creating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.GetList(newID, "")
}

// UpdateList updates a given list.
func (c *Core) UpdateList(id int, l models.List) (models.List, error) {
	res, err := c.q.UpdateList.Exec(id, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)))
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
