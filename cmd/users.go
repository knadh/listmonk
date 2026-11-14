package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/internal/tmptokens"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
	"github.com/pquerna/otp/totp"
	"gopkg.in/volatiletech/null.v6"
)

var (
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]+$`)

	// usersQueryDefaults is the allow list of filter query params.
	usersQueryDefaults = map[string]string{
		"type":         "",
		"status":       "",
		"user_role_id": "",
		"list_role_id": "",
		"order_by":     models.FieldCreatedAt,
		"order":        models.OrderAsc,
	}
)

// emailChange binds a verification link to an account and its current e-mail.
type emailChange struct {
	UserID   int
	OldEmail string
	Email    string
}

// usersView is the admin page view for the users list page.
type usersView struct {
	adminView

	Users []auth.User
	Page  models.PageProps

	// Filters.
	Type         string
	Status       string
	UserRoleID   int
	UserRoleName string
	ListRoleID   int
	ListRoleName string
}

// userView is the admin page view for a single user's add/edit form.
type userView struct {
	adminView

	User      auth.User
	IsNew     bool
	UserRoles []auth.Role
	ListRoles []auth.ListRole
}

// ViewUsers renders the HTML view listing all users.
func (a *App) ViewUsers(c *echo.Context) error {
	// Whitelisted filter params.
	q := makeQuery(c.Request().URL.Query(), usersQueryDefaults)
	var (
		fType        = q.Get("type")
		fStatus      = q.Get("status")
		fUserRole, _ = strconv.Atoi(q.Get("user_role_id"))
		fListRole, _ = strconv.Atoi(q.Get("list_role_id"))
	)

	// Filtering is done in SQL.
	users, err := a.core.GetUsers(fType, fStatus, fUserRole, fListRole, q.Get("order_by"), q.Get("order"))
	if err != nil {
		return err
	}

	data := usersView{
		adminView:  newAdminView(c, a.i18n.T("globals.terms.users"), "", "users.users"),
		Type:       fType,
		Status:     fStatus,
		UserRoleID: fUserRole,
		ListRoleID: fListRole,
	}

	for n := range users {
		// Empty the password hash.
		users[n].Password = null.String{}

		// Capture the role names for rendering in the filter badges on the admin UI.
		if fUserRole > 0 {
			data.UserRoleName = users[n].UserRole.Name
		}
		if fListRole > 0 && users[n].ListRole != nil {
			data.ListRoleName = users[n].ListRole.Name
		}
	}

	data.Users = users
	data.Page = models.NewPageProps(q, len(users), 1, 0)

	return c.Render(http.StatusOK, "admin-users", data)
}

// ViewUser renders the HTML add/edit form for a user. It handles both the "new user"
// (no ID) and the "edit user" (ID in the URI) cases.
func (a *App) ViewUser(c *echo.Context) error {
	var (
		user  auth.User
		isNew = true
	)
	if c.Get("id") != nil {
		out, err := a.core.GetUser(getID(c), "", "")
		if err != nil {
			return err
		}

		out.Password = null.String{}
		user = out
		isNew = false
	}

	// Roles for the role selectors.
	var (
		userRoles []auth.Role
		listRoles []auth.ListRole
	)
	if can(auth.GetUser(c), auth.PermRolesGet) {
		var err error
		userRoles, listRoles, err = a.getRoles()
		if err != nil {
			return err
		}
	}

	title := a.i18n.T("users.newUser")
	if !isNew {
		title = user.Name
	}
	title = fmt.Sprintf("%s / %s", title, a.i18n.T("globals.terms.users"))

	data := userView{
		adminView: newAdminView(c, title, "", "users.users"),
		User:      user,
		IsNew:     isNew,
		UserRoles: userRoles,
		ListRoles: listRoles,
	}

	return c.Render(http.StatusOK, "admin-user", data)
}

// profileView is the admin page view for the logged in user's profile.
type profileView struct {
	adminView

	User auth.User
}

// ViewProfile renders the HTML view for the logged in user's profile.
func (a *App) ViewProfile(c *echo.Context) error {
	user := auth.GetUser(c)

	// Blank out the password hash before sending (obviously!).
	user.Password = null.String{}

	data := profileView{
		adminView: newAdminView(c, a.i18n.T("globals.terms.profile"), "", "globals.terms.profile"),
		User:      user,
	}

	return c.Render(http.StatusOK, "admin-profile", data)
}

// getRoles fetches user and list roles for the user-form role selectors.
func (a *App) getRoles() ([]auth.Role, []auth.ListRole, error) {
	userRoles, err := a.core.GetRoles("", "", "")
	if err != nil {
		return nil, nil, err
	}
	listRoles, err := a.core.GetListRoles("", "", "")
	if err != nil {
		return nil, nil, err
	}
	return userRoles, listRoles, nil
}

// GetUser retrieves a single user by ID.
func (a *App) GetUser(c *echo.Context) error {
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
func (a *App) GetUsers(c *echo.Context) error {
	// Get all users from the DB.
	out, err := a.core.GetUsers("", "", 0, 0, c.QueryParam("order_by"), c.QueryParam("order"))
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
func (a *App) CreateUser(c *echo.Context) error {
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if u.Type != auth.UserTypeUser && u.Type != auth.UserTypeAPI {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}
	if u.Status != auth.UserStatusEnabled && u.Status != auth.UserStatusDisabled {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "status"))
	}
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
func (a *App) UpdateUser(c *echo.Context) error {
	// Incoming params.
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if u.Type != "" && u.Type != auth.UserTypeUser && u.Type != auth.UserTypeAPI {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}
	if u.Status != "" && u.Status != auth.UserStatusEnabled && u.Status != auth.UserStatusDisabled {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "status"))
	}
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
func (a *App) DeleteUser(c *echo.Context) error {
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
func (a *App) DeleteUsers(c *echo.Context) error {
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
func (a *App) GetUserProfile(c *echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Blank out the password hash in the response.
	user.Password.String = ""
	user.Password.Valid = false

	return c.JSON(http.StatusOK, okResp{user})
}

// UpdateUserProfile updates the current user's profile.
func (a *App) UpdateUserProfile(c *echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Incoming params.
	u := auth.User{}
	if err := c.Bind(&u); err != nil {
		return err
	}
	u.PasswordLogin = user.PasswordLogin
	u.Name = strings.TrimSpace(u.Name)

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

// RequestEmailChange sends a verification link to the proposed e-mail address.
func (a *App) RequestEmailChange(c *echo.Context) error {
	user := auth.GetUser(c)
	if !user.PasswordLogin || user.Type == auth.UserTypeAPI {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("public.invalidFeature"))
	}

	// Validate e-mail.
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !utils.ValidateEmail(req.Email) || strings.EqualFold(req.Email, user.Email.String) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
	}

	// Reject addresses already assigned to another account before sending mail.
	if _, err := a.core.GetUser(0, "", req.Email); err == nil {
		return echo.NewHTTPError(http.StatusConflict, a.i18n.T("subscribers.emailExists"))
	} else if httpErr, ok := err.(*echo.HTTPError); !ok || httpErr.Code != http.StatusNotFound {
		return err
	}

	// Set up a temp request ID to e-mail.
	token, err := generateRandomString(tmpAuthTokenLen)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Allow one verification e-mail per minute per user.
	rateKey := fmt.Sprintf("email-change-rate:%d", user.ID)
	if tmptokens.SetNX(rateKey, time.Minute, token).(string) != token {
		return echo.NewHTTPError(http.StatusTooManyRequests, a.i18n.T("users.emailChangeRateLimit"))
	}

	// Set the token.
	key := "email-change:" + token
	tmptokens.Set(key, 30*time.Minute, emailChange{UserID: user.ID, OldEmail: user.Email.String, Email: req.Email})

	// Send the verification e-mail.
	data := struct {
		VerifyURL string
		OldEmail  string
		Email     string
		IP        string
		UserAgent string
	}{
		OldEmail:  user.Email.String,
		Email:     req.Email,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		VerifyURL: fmt.Sprintf("%s/admin/verify-email?token=%s", a.urlCfg.RootURL, token),
	}
	if err := notifs.Notify([]string{req.Email}, a.i18n.T("email.emailChange.subject"), notifs.TplVerifyEmail, data, nil); err != nil {
		tmptokens.Delete(key)
		a.log.Printf("error sending e-mail verification: %v", err)

		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// VerifyEmail renders the verification page and consumes the link on POST.
func (a *App) VerifyEmail(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Referrer-Policy", "no-referrer")
	if c.Request().Method == http.MethodGet {
		return c.Render(http.StatusOK, "verify-email", struct {
			publicTpl
			Token string
		}{publicTpl: publicTpl{Title: a.i18n.T("users.verifyEmail")}, Token: c.QueryParam("token")})
	}

	// Verify the token.
	data, err := tmptokens.Get("email-change:" + c.FormValue("token"))
	if err != nil {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.verifyEmail"), "", a.i18n.T("users.invalidRequest")))
	}

	change, ok := data.(emailChange)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.invalidRequest"))
	}

	// Verified. Update the user's e-mail.
	if err := a.core.UpdateUserEmail(change.UserID, change.OldEmail, change.Email); err != nil {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.verifyEmail"), "", a.i18n.T("users.invalidRequest")))
	}

	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("users.verifyEmail"), "", a.i18n.T("users.emailChanged")))
}

// EnableTOTP enables TOTP 2FA for a user after verifying the code.
func (a *App) EnableTOTP(c *echo.Context) error {
	u := c.Get(auth.UserHTTPCtxKey).(auth.User)

	// Incoming request.
	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	var (
		secret = strings.TrimSpace(req.Secret)
		code   = strings.TrimSpace(req.Code)
	)

	if secret == "" || code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "{users.totpCode}"))
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
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "{users.totpCode}"))
	}

	// Enable TOTP in the DB.
	if err := a.core.SetTwoFA(u.ID, models.TwofaTypeTOTP, secret); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DisableTOTP disables TOTP 2FA for a user after verifying the password.
func (a *App) DisableTOTP(c *echo.Context) error {
	u := c.Get(auth.UserHTTPCtxKey).(auth.User)

	// Incoming request.
	var req struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	password := req.Password

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
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("globals.messages.invalidPassword"))
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
	users, err := co.GetUsers("", "", 0, 0, "", "")
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
