package main

import (
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// CreateAPIUser handles API user creation.
func (a *App) CreateAPIUser(c echo.Context) error {
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	// Set API user type
	u.Type = auth.UserTypeAPI

	if u.Name == "" {
		u.Name = u.Username
	}

	// Create the API user in the DB.
	user, err := a.core.CreateUser(u)
	if err != nil {
		return err
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// AssignRoleToUser handles assigning a role to a user.
func (a *App) AssignRoleToUser(c echo.Context) error {
	// Get the user ID.
	id := getID(c)

	// Incoming params.
	var data struct {
		UserRoleID int  `json:"user_role_id"`
		ListRoleID *int `json:"list_role_id"`
	}
	if err := c.Bind(&data); err != nil {
		return err
	}

	// Validate role IDs.
	if data.UserRoleID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "user_role_id"))
	}

	// Get the user from the DB.
	user, err := a.core.GetUser(id, "", "")
	if err != nil {
		return err
	}

	// Update the user's roles.
	user.UserRoleID = data.UserRoleID
	user.ListRoleID = data.ListRoleID

	// Update the user in the DB.
	updatedUser, err := a.core.UpdateUser(id, user)
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	updatedUser.Password = null.String{}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{updatedUser})
}
