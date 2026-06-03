package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

type listTpl struct {
	Title            string
	Description      string
	Lists            []models.List
	Page             pageTpl
	CacheSlowQueries bool
	Profile          auth.User
}

type pageTpl struct {
	models.PageResults
	Query    string
	OrderBy  string
	Order    string
	Status   string
	Pages    []int
	PrevPage int
	NextPage int
	HasPrev  bool
	HasNext  bool
}

func (l listTpl) Can(perms ...string) bool {
	return can(l.Profile, perms...)
}

func (l listTpl) CanManageList(id int) bool {
	return canManageList(l.Profile, id)
}

// GetLists retrieves lists with additional metadata like subscriber counts.
func (a *App) GetLists(c echo.Context) error {
	out, err := a.getLists(c)
	if err != nil {
		return err
	}
	if minimal, _ := strconv.ParseBool(c.FormValue("minimal")); minimal && out.Total == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ViewLists renders the server-side HTML view for lists.
func (a *App) ViewLists(c echo.Context) error {
	out, err := a.getLists(c)
	if err != nil {
		return err
	}

	lists, _ := out.Results.([]models.List)
	user := auth.GetUser(c)
	data := listTpl{
		Title:       a.i18n.T("globals.terms.lists"),
		Description: a.i18n.T("globals.terms.lists"),
		Lists:       lists,
		Page: makePage(out,
			formValue(c, "order_by", "id"),
			formValue(c, "order", "asc"),
			formValue(c, "status", models.ListStatusActive)),
		CacheSlowQueries: ko.Bool("app.cache_slow_queries"),
		Profile:          user,
	}

	return c.Render(http.StatusOK, "admin-lists", data)
}

func (a *App) getLists(c echo.Context) (models.PageResults, error) {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get the list IDs (or blanket permission) the user has access to.
	hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeGet)

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	minimal, _ := strconv.ParseBool(c.FormValue("minimal"))
	if minimal {
		status := c.FormValue("status")
		res, err := a.core.GetLists("", status, hasAllPerm, permittedIDs)
		if err != nil {
			return models.PageResults{}, err
		}
		if len(res) == 0 {
			return models.PageResults{Results: []models.List{}, Total: 0, Page: 1, PerPage: 0}, nil
		}

		// Meta.
		total := len(res)
		out := models.PageResults{
			Results: res,
			Total:   total,
			Page:    1,
			PerPage: total,
		}

		return out, nil
	}

	// Full list query.
	var (
		query   = strings.TrimSpace(c.FormValue("query"))
		tags    = c.QueryParams()["tag"]
		orderBy = c.FormValue("order_by")
		typ     = c.FormValue("type")
		optin   = c.FormValue("optin")
		status  = c.FormValue("status")
		order   = c.FormValue("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)
	res, total, err := a.core.QueryLists(query, typ, optin, status, tags, orderBy, order, hasAllPerm, permittedIDs, pg.Offset, pg.Limit)
	if err != nil {
		return models.PageResults{}, err
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return out, nil
}

func formValue(c echo.Context, key, fallback string) string {
	if v := strings.TrimSpace(c.FormValue(key)); v != "" {
		return v
	}
	return fallback
}

func can(user auth.User, perms ...string) bool {
	for _, perm := range perms {
		if strings.HasSuffix(perm, "*") {
			prefix := strings.TrimSuffix(perm, "*")
			for _, p := range user.UserRole.Permissions {
				if strings.HasPrefix(p, prefix) {
					return true
				}
			}
			continue
		}
		if user.HasPerm(perm) {
			return true
		}
	}
	return false
}

func canManageList(user auth.User, id int) bool {
	return user.HasListPerm(auth.PermTypeManage, id) == nil
}

func makePage(results models.PageResults, orderBy, order, status string) pageTpl {
	p := pageTpl{
		PageResults: results,
		Query:       results.Query,
		OrderBy:     orderBy,
		Order:       order,
		Status:      status,
	}

	if p.PerPage <= 0 {
		return p
	}

	totalPages := p.Total / p.PerPage
	if p.Total%p.PerPage != 0 {
		totalPages++
	}
	if totalPages <= 1 {
		return p
	}

	start := p.PageResults.Page - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > totalPages {
		end = totalPages
		start = end - 4
		if start < 1 {
			start = 1
		}
	}

	p.Pages = make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		p.Pages = append(p.Pages, i)
	}
	p.PrevPage = p.PageResults.Page - 1
	p.NextPage = p.PageResults.Page + 1
	p.HasPrev = p.PageResults.Page > 1
	p.HasNext = p.PageResults.Page < totalPages

	return p
}

// GetList retrieves a single list by id.
// It's permission checked by the listPerm middleware.
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
// It's permission checked by the listPerm middleware.
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
	if err := a.core.DeleteLists([]int{id}, "", true, nil); err != nil {
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
		if err := a.core.DeleteLists(ids, "", true, nil); err != nil {
			return err
		}
	} else {
		// For query deletion, get the list IDs the user has manage permission for.
		hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeManage)

		// Delete the lists from the DB with permission filtering.
		if err := a.core.DeleteLists(nil, query, hasAllPerm, permittedIDs); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}
