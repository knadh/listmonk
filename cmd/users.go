package main

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"gopkg.in/volatiletech/null.v6"
)

var (
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]+$`)
)

// usersView is the admin page view for the users list page.
type usersView struct {
	adminView

	Users []auth.User
}

// userView is the admin page view for a single user's add/edit form.
type userView struct {
	adminView

	User      auth.User
	IsEditing bool
	UserRoles []auth.Role
	ListRoles []auth.ListRole
}

// ViewUsers renders the HTML view listing all users.
func (a *App) ViewUsers(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermUsersGet) {
		return auth.ErrPermDenied
	}

	users, err := a.core.GetUsers()
	if err != nil {
		return err
	}

	// Blank out the password hashes.
	for n := range users {
		users[n].Password = null.String{}
	}

	data := usersView{
		adminView: newAdminView(c, a.i18n.T("globals.terms.users"), ""),
		Users:     users,
	}

	return c.Render(http.StatusOK, "admin-users", data)
}

// ViewUser renders the HTML add/edit form for a user. It handles both the "new user"
// (no ID) and the "edit user" (ID in the URI) cases.
func (a *App) ViewUser(c echo.Context) error {
	if !can(auth.GetUser(c), auth.PermUsersGet) {
		return auth.ErrPermDenied
	}

	var (
		user      auth.User
		isEditing bool
	)
	if c.Get("id") != nil {
		out, err := a.core.GetUser(getID(c), "", "")
		if err != nil {
			return err
		}
		out.Password = null.String{}
		user = out
		isEditing = true
	}

	// Roles for the role selectors.
	userRoles, listRoles, err := a.getRoles()
	if err != nil {
		return err
	}

	title := a.i18n.T("users.newUser")
	if isEditing {
		title = user.Name
	}

	data := userView{
		adminView: newAdminView(c, title, ""),
		User:      user,
		IsEditing: isEditing,
		UserRoles: userRoles,
		ListRoles: listRoles,
	}

	return c.Render(http.StatusOK, "admin-user", data)
}

// getRoles fetches user and list roles for the user-form role selectors.
func (a *App) getRoles() ([]auth.Role, []auth.ListRole, error) {
	userRoles, err := a.core.GetRoles()
	if err != nil {
		return nil, nil, err
	}
	listRoles, err := a.core.GetListRoles()
	if err != nil {
		return nil, nil, err
	}
	return userRoles, listRoles, nil
}

// GetUser retrieves a single user by ID.
func (a *App) GetUser(c echo.Context) error {
	// Get the user from the DB.
	id := getID(c)
	out, err := a.core.GetUser(id, "", "")
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	out.Password = null.String{}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetUsers retrieves all users.
func (a *App) GetUsers(c echo.Context) error {
	// Get all users from the DB.
	out, err := a.core.GetUsers()
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	for n := range out {
		out[n].Password = null.String{}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUser handles user creation.
func (a *App) CreateUser(c echo.Context) error {
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if u.Type != auth.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	// Create the user in the DB.
	user, err := a.core.CreateUser(u)
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	if user.Type != auth.UserTypeAPI {
		user.Password = null.String{}
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// UpdateUser handles user modification.
func (a *App) UpdateUser(c echo.Context) error {
	// Incoming params.
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	// Get the user ID.
	id := getID(c)
	if u.Type != auth.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}

		// Validate password if password login is enabled.
		if u.PasswordLogin {
			if u.Password.String != "" {
				// If a password is sent, validate it before updating in the DB. If it's not set, leave the password in the DB untouched.
				if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
					return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			} else {
				// Get the user from the DB.
				user, err := a.core.GetUser(id, "", "")
				if err != nil {
					return err
				}

				// If password login is enabled, but there's no password in the DB and there's no incoming
				// password, throw an error.
				if !user.HasPassword {
					return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	// Default the name to username if not set.
	if u.Name == "" {
		u.Name = u.Username
	}

	// Update the user in the DB.
	user, err := a.core.UpdateUser(id, u)
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	user.Password = null.String{}

	// If password was changed by admin, destroy all sessions for the given user.
	if u.Password.String != "" {
		if err := a.core.DeleteUserSessions(id, ""); err != nil {
			a.log.Printf("error destroying sessions on admin password change for user_id=%d: %v", id, err)
		}
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// DeleteUser handles the deletion of a single user by ID.
func (a *App) DeleteUser(c echo.Context) error {
	// Delete the user(s) from the DB.
	id := getID(c)
	if err := a.core.DeleteUsers([]int{id}); err != nil {
		return err
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteUsers handles user deletion, either a single one (ID in the URI), or a list.
func (a *App) DeleteUsers(c echo.Context) error {
	ids, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the user(s) from the DB.
	if err := a.core.DeleteUsers(ids); err != nil {
		return err
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetUserProfile fetches the uesr profile for the currently logged in user.
func (a *App) GetUserProfile(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Blank out the password hash in the response.
	user.Password.String = ""
	user.Password.Valid = false

	return c.JSON(http.StatusOK, okResp{user})
}

// UpdateUserProfile update's the current user's profile.
func (a *App) UpdateUserProfile(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Incoming params.
	u := auth.User{}
	if err := c.Bind(&u); err != nil {
		return err
	}
	u.PasswordLogin = user.PasswordLogin
	u.Name = strings.TrimSpace(u.Name)
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if user.PasswordLogin {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		u.Email = null.String{String: email, Valid: true}
	}

	if u.PasswordLogin && u.Password.String != "" {
		if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
		}
	}

	// Update the user in the DB.
	out, err := a.core.UpdateUserProfile(user.ID, u)
	if err != nil {
		return err
	}

	// If password was changed, destroy all existing sessions for the user except for the current one.
	if u.Password.String != "" {
		if err := a.core.DeleteUserSessions(user.ID, auth.GetSessionID(c)); err != nil {
			a.log.Printf("error destroying sessions after profile password change for user_id=%d: %v", user.ID, err)
		}
	}

	// Blank out the password hash in the response.
	out.Password = null.String{}

	return c.JSON(http.StatusOK, okResp{out})
}

// EnableTOTP enables TOTP 2FA for a user after verifying the code.
func (a *App) EnableTOTP(c echo.Context) error {
	var (
		u      = c.Get(auth.UserHTTPCtxKey).(auth.User)
		secret = strings.TrimSpace(c.FormValue("secret"))
		code   = strings.TrimSpace(c.FormValue("code"))
	)

	if secret == "" || code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidFields"))
	}

	// If password login is disabled, can't enable TOTP.
	if !u.PasswordLogin {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.invalidFeature"))
	}

	// If TOTP is already enabled, don't allow re-enabling.
	if u.TwofaType == models.TwofaTypeTOTP {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.twoFAAlreadyEnabled"))
	}

	// Verify the TOTP code.
	valid := totp.Validate(code, secret)
	if !valid {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.invalidTOTPCode"))
	}

	// Enable TOTP in the DB.
	if err := a.core.SetTwoFA(u.ID, models.TwofaTypeTOTP, secret); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DisableTOTP disables TOTP 2FA for a user after verifying the password.
func (a *App) DisableTOTP(c echo.Context) error {
	var (
		u        = c.Get(auth.UserHTTPCtxKey).(auth.User)
		password = c.FormValue("password")
	)

	// TOTP isn't enabled.
	if u.TwofaType != models.TwofaTypeTOTP {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.twoFANotEnabled"))
	}

	// Validate password.
	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}

	// Verify the password.
	if _, err := a.core.LoginUser(u.Username, password); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("users.invalidPassword"))
	}

	// Disable TOTP in the DB.
	if err := a.core.SetTwoFA(u.ID, models.TwofaTypeNone, ""); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// cacheUsers fetches (API) users and caches them in the auth module.
// It also returns a bool indicating whether there are any actual users in the DB at all,
// which if there aren't, the first time user setup needs to be run.
func cacheUsers(co *core.Core, a *auth.Auth) (bool, error) {
	users, err := co.GetUsers()
	if err != nil {
		return false, err
	}

	hasUser := false
	apiUsers := make([]auth.User, 0, len(users))
	for _, u := range users {
		if u.Type == auth.UserTypeAPI && u.Status == auth.UserStatusEnabled {
			apiUsers = append(apiUsers, u)
		}

		if u.Type == auth.UserTypeUser {
			hasUser = true
		}
	}

	a.CacheAPIUsers(apiUsers)
	return hasUser, nil
}
