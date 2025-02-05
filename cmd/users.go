package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

var (
	reUsername = regexp.MustCompile("^[a-zA-Z0-9_\\-\\.]+$")
)

// handleGetUsers retrieves users.
func (h *Handler) handleGetUsers(c echo.Context) error {
	userID, _ := strconv.Atoi(c.Param("id"))

	// Fetch one.
	single := false
	if userID > 0 {
		single = true
	}

	if single {
		out, err := h.app.core.GetUser(userID, "", "")
		if err != nil {
			return err
		}

		out.Password = null.String{}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get all users.
	out, err := h.app.core.GetUsers()
	if err != nil {
		return err
	}

	for n := range out {
		out[n].Password = null.String{}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateUser handles user creation.
func (h *Handler) handleCreateUser(c echo.Context) error {
	u := models.User{}

	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if u.Type != models.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	// Create the user in the database.
	user, err := h.app.core.CreateUser(u)
	if err != nil {
		return err
	}
	if user.Type != models.UserTypeAPI {
		user.Password = null.String{}
	}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// handleUpdateUser handles user modification.
func (h *Handler) handleUpdateUser(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
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
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	if u.Type != models.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin && u.Password.String != "" {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}

			if u.Password.String != "" {
				if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
					return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			} else {
				// Get the existing user for password validation.
				user, err := h.app.core.GetUser(id, "", "")
				if err != nil {
					return err
				}

				// If password login is enabled, but there's no password in the DB and there's no incoming
				// password, throw an error.
				if !user.HasPassword {
					return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	// Update the user in the DB.
	user, err := h.app.core.UpdateUser(id, u)
	if err != nil {
		return err
	}

	// Clear the pasword before sending outside.
	user.Password = null.String{}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// handleDeleteUsers handles user deletion, either a single one (ID in the URI), or a list.
func (h *Handler) handleDeleteUsers(c echo.Context) error {
	var (
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)

	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := h.app.core.DeleteUsers(ids); err != nil {
		return err
	}

	// Cache the API token for validating API queries without hitting the DB every time.
	if _, err := cacheUsers(h.app.core, h.app.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetUserProfile fetches the uesr profile for the currently logged in user.
func (h *Handler) handleGetUserProfile(c echo.Context) error {
	var (
		user = c.Get(auth.UserKey).(models.User)
	)
	user.Password.String = ""
	user.Password.Valid = false

	return c.JSON(http.StatusOK, okResp{user})
}

// handleUpdateUserProfile update's the current user's profile.
func (h *Handler) handleUpdateUserProfile(c echo.Context) error {
	user := c.Get(auth.UserKey).(models.User)

	u := models.User{}
	if err := c.Bind(&u); err != nil {
		return err
	}
	u.PasswordLogin = user.PasswordLogin
	u.Name = strings.TrimSpace(u.Name)
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if user.PasswordLogin {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		u.Email = null.String{String: email, Valid: true}
	}

	if u.PasswordLogin && u.Password.String != "" {
		if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
		}
	}

	out, err := h.app.core.UpdateUserProfile(user.ID, u)
	if err != nil {
		return err
	}
	out.Password = null.String{}

	return c.JSON(http.StatusOK, okResp{out})
}

// cacheUsers fetches (API) users and caches them in the auth module.
// It also returns a bool indicating whether there are any actual users in the DB at all,
// which if there aren't, the first time user setup needs to be run.
func cacheUsers(co *core.Core, a *auth.Auth) (bool, error) {
	allUsers, err := co.GetUsers()
	if err != nil {
		return false, err
	}

	hasUser := false
	apiUsers := make([]models.User, 0, len(allUsers))
	for _, u := range allUsers {
		if u.Type == models.UserTypeAPI && u.Status == models.UserStatusEnabled {
			apiUsers = append(apiUsers, u)
		}

		if u.Type == models.UserTypeUser {
			hasUser = true
		}
	}

	a.CacheAPIUsers(apiUsers)
	return hasUser, nil
}
