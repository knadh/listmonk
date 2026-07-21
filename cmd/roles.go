package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// roleRow is the normalized shape of a user/list role for the roles table view.
type roleRow struct {
	ID        int
	Name      string
	CreatedAt null.Time
	UpdatedAt null.Time
}

// rolesView is the admin page view listing user or list roles.
type rolesView struct {
	adminView

	Type  string
	Roles []roleRow
}

// permGroup is a group of granular permissions as defined in permissions.json.
type permGroup struct {
	Group       string   `json:"group"`
	Permissions []string `json:"permissions"`
}

// roleFormView is the admin page view for a single role's add/edit form.
type roleFormView struct {
	adminView

	IsEditing  bool
	Role       any
	PermGroups []permGroup
	AllLists   []models.List
}

// ViewUserRoles renders the HTML view listing user roles.
func (a *App) ViewUserRoles(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermRolesGet) {
		return auth.ErrPermDenied
	}

	roles, err := a.core.GetRoles()
	if err != nil {
		return err
	}

	rows := make([]roleRow, 0, len(roles))
	for _, r := range roles {
		rows = append(rows, roleRow{ID: r.ID, Name: r.Name.String, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}

	data := rolesView{
		adminView: newAdminView(c, a.i18n.T("users.userRoles"), ""),
		Type:      "user",
		Roles:     rows,
	}

	return c.Render(http.StatusOK, "admin-roles", data)
}

// ViewListRoles renders the HTML view listing list roles.
func (a *App) ViewListRoles(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermRolesGet) {
		return auth.ErrPermDenied
	}

	roles, err := a.core.GetListRoles()
	if err != nil {
		return err
	}

	rows := make([]roleRow, 0, len(roles))
	for _, r := range roles {
		rows = append(rows, roleRow{ID: r.ID, Name: r.Name.String, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}

	data := rolesView{
		adminView: newAdminView(c, a.i18n.T("users.listRoles"), ""),
		Type:      "list",
		Roles:     rows,
	}

	return c.Render(http.StatusOK, "admin-roles", data)
}

// ViewUserRole renders the HTML add/edit form for a user role.
func (a *App) ViewUserRole(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermRolesGet) {
		return auth.ErrPermDenied
	}

	var (
		role      auth.Role
		isEditing bool
	)
	if c.Get("id") != nil {
		r, err := a.core.GetRole(getID(c))
		if err != nil {
			return err
		}
		role = r
		isEditing = true
	}

	groups, err := a.parsePermGroups()
	if err != nil {
		return err
	}

	title := a.i18n.T("users.newUserRole")
	if isEditing {
		title = role.Name.String
	}

	data := roleFormView{
		adminView:  newAdminView(c, title, ""),
		IsEditing:  isEditing,
		Role:       role,
		PermGroups: groups,
	}

	return c.Render(http.StatusOK, "admin-user-role", data)
}

// ViewListRole renders the HTML add/edit form for a list role.
func (a *App) ViewListRole(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermRolesGet) {
		return auth.ErrPermDenied
	}

	var (
		role      auth.ListRole
		isEditing bool
	)
	if c.Get("id") != nil {
		// There's no core getter for a single list role, so fetch all and pick the one.
		id := getID(c)
		roles, err := a.core.GetListRoles()
		if err != nil {
			return err
		}
		found := false
		for _, r := range roles {
			if r.ID == id {
				role = r
				found = true
				break
			}
		}
		if !found {
			return echo.NewHTTPError(http.StatusNotFound, a.i18n.Ts("globals.messages.notFound", "name", "{users.listRole}"))
		}
		isEditing = true
	}

	// All lists for the list-permission selector.
	lists, err := a.core.GetLists("", models.ListStatusActive, true, nil)
	if err != nil {
		return err
	}

	title := a.i18n.T("users.newListRole")
	if isEditing {
		title = role.Name.String
	}

	data := roleFormView{
		adminView: newAdminView(c, title, ""),
		IsEditing: isEditing,
		Role:      role,
		AllLists:  lists,
	}

	return c.Render(http.StatusOK, "admin-list-role", data)
}

// parsePermGroups parses the raw permissions.json config into permission groups.
func (a *App) parsePermGroups() ([]permGroup, error) {
	var groups []permGroup
	if err := json.Unmarshal(a.cfg.PermissionsRaw, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// GetUserRoles retrieves roles.
func (a *App) GetUserRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GeListRoles retrieves roles.
func (a *App) GeListRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetListRoles()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUserRole handles role creation.
func (a *App) CreateUserRole(c echo.Context) error {
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateListRole handles role creation.
func (a *App) CreateListRole(c echo.Context) error {
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateListRole(r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateUserRole handles role modification.
func (a *App) UpdateUserRole(c echo.Context) error {
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateUserRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateListRole handles role modification.
func (a *App) UpdateListRole(c echo.Context) error {
	// Get the role ID.
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateListRole(id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteRole handles (user|list) role deletion.
func (a *App) DeleteRole(c echo.Context) error {
	// Get the role ID.
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the role from the DB.
	if err := a.core.DeleteRole(int(id)); err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateUserRole(r auth.Role) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := a.cfg.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("permission: %s", p)))
		}
	}

	return nil
}

func (a *App) validateListRole(r auth.ListRole) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
