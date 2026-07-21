package main

import (
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// listsView is the admin page view.
type listsView struct {
	adminView

	Lists            []models.List
	Page             models.PageProps
	CacheSlowQueries bool
}

type listView struct {
	adminView

	List models.List
}

// publicListForm is the minimal list representation used by the forms view.
type publicListForm struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetLists retrieves lists with additional metadata like subscriber counts.
func (a *App) GetLists(c echo.Context) error {
	lists, props, err := a.getLists(c)
	if err != nil {
		return err
	}

	if props.Param("minimal") == "true" && props.Total == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	return c.JSON(http.StatusOK, okResp{models.PageResults{
		Results:   lists,
		PageProps: props,
	}})
}

// ViewLists renders the HTML view for lists.
func (a *App) ViewLists(c echo.Context) error {
	lists, props, err := a.getLists(c)
	if err != nil {
		return err
	}

	pageID := "lists.all"
	if c.QueryParam("status") == models.ListStatusArchived {
		pageID = "lists.archived"
	}

	data := listsView{
		adminView:        newAdminView(c, a.i18n.T("globals.terms.lists"), "", pageID),
		Lists:            lists,
		Page:             props,
		CacheSlowQueries: ko.Bool("app.cache_slow_queries"),
	}

	return c.Render(http.StatusOK, "admin-lists", data)
}

// ViewList renders the HTML view for editing a list.
func (a *App) ViewList(c echo.Context) error {
	user := auth.GetUser(c)

	// Check if the user has access to the list.
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeGet, id); err != nil {
		return err
	}

	list, err := a.core.GetList(id, "")
	if err != nil {
		return err
	}

	data := listView{
		adminView: newAdminView(c, list.Name, "", "lists.all"),
		List:      list,
	}

	return c.Render(http.StatusOK, "admin-list", data)
}

// ViewForms renders the HTML view for the public subscription form generator.
func (a *App) ViewForms(c echo.Context) error {
	lists, err := a.core.GetLists(models.ListTypePublic, models.ListStatusActive, true, nil)
	if err != nil {
		return err
	}

	out := make([]publicListForm, 0, len(lists))
	for _, l := range lists {
		out = append(out, publicListForm{
			UUID:        l.UUID,
			Name:        l.Name,
			Description: l.Description,
		})
	}

	data := struct {
		adminView
		PublicLists []publicListForm
	}{
		adminView:   newAdminView(c, a.i18n.T("forms.title"), "", "lists.forms"),
		PublicLists: out,
	}

	return c.Render(http.StatusOK, "admin-forms", data)
}

// GetList retrieves a single list by id.
func (a *App) GetList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check if the user has access to the list.
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeGet, id); err != nil {
		return err
	}

	// Get the list from the DB.
	out, err := a.core.GetList(id, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateList handles list creation.
func (a *App) CreateList(c echo.Context) error {
	l := models.List{}
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	out, err := a.core.CreateList(l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateList handles list modification.
func (a *App) UpdateList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check if the user has access to the list.
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	// Update the list in the DB.
	out, err := a.core.UpdateList(id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteList deletes a single list by ID.
func (a *App) DeleteList(c echo.Context) error {
	id := getID(c)

	// Check if the user has manage permission for the list.
	user := auth.GetUser(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	// Delete the list from the DB.
	// Pass getAll=true since we've already verified permissions above.
	if err := a.core.DeleteLists([]int{id}, "", "", "", "", nil, true, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteLists deletes multiple lists by IDs or by query.
func (a *App) DeleteLists(c echo.Context) error {
	user := auth.GetUser(c)

	var (
		ids   []int
		query string
		all   bool
	)

	// Check for IDs in query params.
	if len(c.Request().URL.Query()["id"]) > 0 {
		var err error
		ids, err = parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
	} else {
		// Check for query param.
		query = strings.TrimSpace(c.FormValue("query"))
		all = c.FormValue("all") == "true"
	}

	// Validate that either IDs or query is provided.
	if len(ids) == 0 && (query == "" && !all) {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", "id or query required"))
	}

	// For ID deletion, check if the user has manage permission for the specific lists.
	if len(ids) > 0 {
		if err := user.HasListPerm(auth.PermTypeManage, ids...); err != nil {
			return err
		}

		// Delete the lists from the DB.
		// Pass getAll=true since we've already verified permissions above.
		if err := a.core.DeleteLists(ids, "", "", "", "", nil, true, nil); err != nil {
			return err
		}
	} else {
		// For query deletion, get the list IDs the user has manage permission for.
		hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeManage)

		// Delete the lists from the DB with permission filtering.
		if err := a.core.DeleteLists(nil, query,
			c.FormValue("type"),
			c.FormValue("optin"),
			c.FormValue("status"),
			c.Request().URL.Query()["tag"],
			hasAllPerm, permittedIDs); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) getLists(c echo.Context) ([]models.List, models.PageProps, error) {
	q := makeQuery(c.Request().URL.Query(), map[string]string{
		"page":     "",
		"minimal":  "",
		"query":    "",
		"order_by": "",
		"order":    "",
		"tag":      "",
		"status":   models.ListStatusActive,
	})

	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get the list IDs (or blanket permission) the user has access to.
	hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeGet)

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	minimal := q.Get("minimal") == "true"
	status := q.Get("status")
	if minimal {
		res, err := a.core.GetLists("", status, hasAllPerm, permittedIDs)
		if err != nil {
			return nil, models.PageProps{}, err
		}
		if len(res) == 0 {
			return res, models.NewPageProps(q, 0, 1, 0), nil
		}

		// Meta.
		total := len(res)
		return res, models.NewPageProps(q, total, 1, total), nil
	}

	// Run the DB query.
	pg := a.pg.NewFromURL(q)
	res, total, err := a.core.QueryLists(
		q.Get("query"),
		q.Get("type"),
		q.Get("optin"),
		status,
		q["tag"],
		q.Get("order_by"),
		q.Get("order"),
		hasAllPerm,
		permittedIDs,
		pg.Offset,
		pg.Limit)
	if err != nil {
		return nil, models.PageProps{}, err
	}

	return res, models.NewPageProps(q, total, pg.Page, pg.PerPage), nil
}
