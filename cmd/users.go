package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleGetUsers retrieves users.
func handleGetUsers(c echo.Context) error {
	var (
		app       = c.Get("app").(*App)
		userID, _ = strconv.Atoi(c.Param("id"))
	)

	// Fetch one.
	single := false
	if userID > 0 {
		single = true
	}

	if single {
		out, err := app.core.GetUser(userID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get all users.
	out, err := app.core.GetUsers()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateUser handles user creation.
func handleCreateUser(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		u   = models.User{}
	)

	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(u.Email)

	if u.Name == "" {
		u.Name = u.Username
	}

	if !strHasLen(u.Username, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	if u.PasswordLogin {
		if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
		}
	}

	out, err := app.core.CreateUser(u)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleUpdateUser handles user modification.
func handleUpdateUser(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var u models.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	// Validate.
	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(u.Email)

	if u.Name == "" {
		u.Name = u.Username
	}

	if !strHasLen(u.Username, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	if u.PasswordLogin {
		if u.Password.String != "" {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		} else {
			// Get the existing user for password validation.
			user, err := app.core.GetUser(id)
			if err != nil {
				return err
			}

			// If password login is enabled, but there's no password in the DB and there's no incoming
			// password, throw an error.
			if !user.HasPassword {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		}
	}

	out, err := app.core.UpdateUser(id, u)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleDeleteUsers handles user deletion, either a single one (ID in the URI), or a list.
func handleDeleteUsers(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := app.core.DeleteUsers(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
