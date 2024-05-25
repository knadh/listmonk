package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

var (
	reUsername = regexp.MustCompile("^[a-zA-Z0-9_\\-\\.]+$")
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
		out, err := app.core.GetUser(userID, "", "")
		if err != nil {
			return err
		}

		out.Password = null.String{}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get all users.
	out, err := app.core.GetUsers()
	if err != nil {
		return err
	}

	for n := range out {
		out[n].Password = null.String{}
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
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if u.Type != models.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	// Create the user in the database.
	out, err := app.core.CreateUser(u)
	if err != nil {
		return err
	}
	out.Password = null.String{}

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
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	if u.Type != models.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin && u.Password.String != "" {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}

			if u.Password.String != "" {
				if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
					return echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			} else {
				// Get the existing user for password validation.
				user, err := app.core.GetUser(id, "", "")
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

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	out, err := app.core.UpdateUser(id, u)
	if err != nil {
		return err
	}
	out.Password = null.String{}

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

// handleGetUserProfile fetches the uesr profile for the currently logged in user.
func handleGetUserProfile(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)
	)
	user.Password.String = ""
	user.Password.Valid = false

	return c.JSON(http.StatusOK, okResp{user})
}
