package core

import (
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetLists gets all lists optionally filtered by type.
func (c *Core) GetLists(typ string, authID string) ([]models.List, error) {
	out := []models.List{}

	if err := c.q.GetLists.Select(&out, typ, "id", authID); err != nil {
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

// get lists by authid
// GetListsByAuthID gets all lists associated with a particular authid.
// func (c *Core) GetListsByAuthID(authid string) ([]models.List, error) {
// 	out := []models.List{}

// 	// Only pass authid to filter the lists
// 	if err := c.q.GetListsByAuthID.Select(&out, authid); err != nil {
// 		c.log.Printf("error fetching lists by authid: %v", err)
// 		return nil, echo.NewHTTPError(http.StatusInternalServerError,
// 			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
// 	}

// 	// Replace null tags and calculate subscriber count.
// 	for i, l := range out {
// 		if l.Tags == nil {
// 			out[i].Tags = []string{}
// 		}

// 		// Total counts for subscribers.
// 		for _, c := range l.SubscriberCounts {
// 			out[i].SubscriberCount += c
// 		}
// 	}

// 	return out, nil
// }

// QueryLists gets multiple lists based on multiple query params. Along with the  paginated and sliced
// results, the total number of lists in the DB is returned.
func (c *Core) QueryLists(searchStr, typ, optin string, tags []string, orderBy, order string, offset, limit int, authID string) ([]models.List, int, error) {
	_ = c.refreshCache(matListSubStats, false)

	if tags == nil {
		tags = []string{}
	}

	var (
		out            = []models.List{}
		queryStr, stmt = makeSearchQuery(searchStr, orderBy, order, c.q.QueryLists, listQuerySortFields, authID)
	)
	if err := c.db.Select(&out, stmt, 0, "", queryStr, typ, optin, pq.StringArray(tags), offset, limit, authID); err != nil {
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
func (c *Core) GetList(id int, uuid string, authID string) (models.List, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var res []models.List
	queryStr, stmt := makeSearchQuery("", "", "", c.q.QueryLists, nil, authID)
	if err := c.db.Select(&res, stmt, id, uu, queryStr, "", "", pq.StringArray{}, 0, 1, authID); err != nil {
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
func (c *Core) CreateList(l models.List, authID string) (models.List, error) {
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

	l.UUID = uu.String()
	l.AuthID = authID

	// Insert and read ID.
	var newID int

	if err := c.q.CreateList.Get(&newID, l.UUID, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description, l.AuthID); err != nil {
		c.log.Printf("error creating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.GetList(newID, "", authID)
}

// UpdateList updates a given list.
func (c *Core) UpdateList(id int, l models.List, authID string) (models.List, error) {

	l.AuthID = authID

	res, err := c.q.UpdateList.Exec(id, l.Name, l.Type, l.Optin, pq.StringArray(normalizeTags(l.Tags)), l.Description, l.AuthID)
	if err != nil {
		c.log.Printf("error updating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	return c.GetList(id, "", l.AuthID)
}

// DeleteList deletes a list.
func (c *Core) DeleteList(id int, authID string) error {
	return c.DeleteLists([]int{id}, authID)
}

// DeleteLists deletes multiple lists.
func (c *Core) DeleteLists(ids []int, authID string) error {
	if _, err := c.q.DeleteLists.Exec(pq.Array(ids), authID); err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
