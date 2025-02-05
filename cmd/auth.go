package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/zerodha/simplesessions/v3"
	"gopkg.in/volatiletech/null.v6"
)

type loginTpl struct {
	Title       string
	Description string

	NextURI          string
	Nonce            string
	PasswordEnabled  bool
	OIDCProvider     string
	OIDCProviderLogo string
	Error            string
}

type oidcState struct {
	Nonce string `json:"nonce"`
	Next  string `json:"next"`
}

var oidcProviders = map[string]bool{
	"google.com":          true,
	"microsoftonline.com": true,
	"auth0.com":           true,
	"github.com":          true,
}

// handleLoginPage renders the login page and handles the login form.
func (h *Handler) handleLoginPage(c echo.Context) error {
	// Has the user been setup?
	h.app.RLock()
	needsUserSetup := h.app.needsUserSetup
	h.app.RUnlock()

	if needsUserSetup {
		return h.handleLoginSetupPage(c)
	}

	// Process POST login request.
	var loginErr error
	if c.Request().Method == http.MethodPost {
		loginErr = h.doLogin(c)
		if loginErr == nil {
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	return h.renderLoginPage(c, loginErr)
}

// handleLoginSetupPage renders the first time user login page and handles the login form.
func (h *Handler) handleLoginSetupPage(c echo.Context) error {
	// Process POST login request.
	var loginErr error

	if c.Request().Method == http.MethodPost {
		loginErr = h.doLoginSetup(c)
		if loginErr == nil {
			h.app.Lock()
			h.app.needsUserSetup = false
			h.app.Unlock()
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	return h.renderLoginSetupPage(c, loginErr)
}

// handleLogout logs a user out.
func (h *Handler) handleLogout(c echo.Context) error {
	var (
		sess = c.Get(auth.SessionKey).(*simplesessions.Session)
	)

	// Clear the session.
	_ = sess.Destroy()

	return c.JSON(http.StatusOK, okResp{true})
}

// handleOIDCLogin initializes an OIDC request and redirects to the OIDC provider for login.
func (h *Handler) handleOIDCLogin(c echo.Context) error {
	// Verify that the request came from the login page (CSRF).
	nonce, err := c.Cookie("nonce")
	if err != nil || nonce.Value == "" || nonce.Value != c.FormValue("nonce") {
		return echo.NewHTTPError(http.StatusUnauthorized, h.app.i18n.T("users.invalidRequest"))
	}

	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	state := oidcState{
		Nonce: nonce.Value,
		Next:  next,
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		h.app.log.Printf("error marshalling OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("globals.messages.internalError"))
	}

	return c.Redirect(http.StatusFound, h.app.auth.GetOIDCAuthURL(base64.URLEncoding.EncodeToString(stateJSON), nonce.Value))
}

// handleOIDCFinish receives the redirect callback from the OIDC provider and completes the handshake.
func (h *Handler) handleOIDCFinish(c echo.Context) error {
	nonce, err := c.Cookie("nonce")
	if err != nil || nonce.Value == "" {
		return h.renderLoginPage(c, echo.NewHTTPError(http.StatusUnauthorized, h.app.i18n.T("users.invalidRequest")))
	}

	// Validate the OIDC token.
	oidcToken, claims, err := h.app.auth.ExchangeOIDCToken(c.Request().URL.Query().Get("code"), nonce.Value)
	if err != nil {
		return h.renderLoginPage(c, err)
	}

	// Validate the state.
	var state oidcState
	stateB, err := base64.URLEncoding.DecodeString(c.QueryParam("state"))
	if err != nil {
		h.app.log.Printf("error decoding OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("globals.messages.internalError"))
	}
	if err := json.Unmarshal(stateB, &state); err != nil {
		h.app.log.Printf("error unmarshalling OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("globals.messages.internalError"))
	}
	if state.Nonce != nonce.Value {
		return h.renderLoginPage(c, echo.NewHTTPError(http.StatusUnauthorized, h.app.i18n.T("users.invalidRequest")))
	}

	// Validate e-mail from the claim.
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		return h.renderLoginPage(c, errors.New(h.app.i18n.Ts("globals.messages.invalidFields", "name", "email")))
	}

	em, err := mail.ParseAddress(email)
	if err != nil {
		return h.renderLoginPage(c, err)
	}
	email = strings.ToLower(em.Address)

	// Get the user by e-mail received from OIDC.
	user, err := h.app.core.GetUser(0, "", email)
	if err != nil {
		return h.renderLoginPage(c, err)
	}

	// Update user login.
	if err := h.app.core.UpdateUserLogin(user.ID, claims.Picture); err != nil {
		return h.renderLoginPage(c, err)
	}

	// Set the session.
	if err := h.app.auth.SaveSession(user, oidcToken, c); err != nil {
		return h.renderLoginPage(c, err)
	}

	return c.Redirect(http.StatusFound, utils.SanitizeURI(state.Next))
}

// renderLoginPage renders the login page and handles the login form.
func (h *Handler) renderLoginPage(c echo.Context, loginErr error) error {
	var (
		next = utils.SanitizeURI(c.FormValue("next"))
	)

	if next == "/" {
		next = uriAdmin
	}

	oidcProvider := ""
	oidcProviderLogo := ""
	if h.app.constants.Security.OIDC.Enabled {
		oidcProviderLogo = "oidc.png"
		u, err := url.Parse(h.app.constants.Security.OIDC.Provider)
		if err == nil {
			h := strings.Split(u.Hostname(), ".")

			// Get the last two h for the root domain
			if len(h) >= 2 {
				oidcProvider = h[len(h)-2] + "." + h[len(h)-1]
			} else {
				oidcProvider = u.Hostname()
			}

			if _, ok := oidcProviders[oidcProvider]; ok {
				oidcProviderLogo = oidcProvider + ".png"
			}
		}
	}

	out := loginTpl{
		Title:            h.app.i18n.T("users.login"),
		PasswordEnabled:  true,
		OIDCProvider:     oidcProvider,
		OIDCProviderLogo: oidcProviderLogo,
		NextURI:          next,
	}

	if loginErr != nil {
		if e, ok := loginErr.(*echo.HTTPError); ok {
			out.Error = e.Message.(string)
		} else {
			out.Error = loginErr.Error()
		}
	}

	// Generate and set a nonce for preventing CSRF requests.
	nonce, err := utils.GenerateRandomString(16)
	if err != nil {
		h.app.log.Printf("error generating OIDC nonce: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.internalError"))
	}
	c.SetCookie(&http.Cookie{
		Name:     "nonce",
		Value:    nonce,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	out.Nonce = nonce

	return c.Render(http.StatusOK, "admin-login", out)
}

// renderLoginSetupPage renders the first time user setup page.
func (h *Handler) renderLoginSetupPage(c echo.Context, loginErr error) error {
	var (
		next = utils.SanitizeURI(c.FormValue("next"))
	)

	if next == "/" {
		next = uriAdmin
	}

	out := loginTpl{
		Title:           h.app.i18n.T("users.login"),
		PasswordEnabled: true,
		NextURI:         next,
	}

	if loginErr != nil {
		if e, ok := loginErr.(*echo.HTTPError); ok {
			out.Error = e.Message.(string)
		} else {
			out.Error = loginErr.Error()
		}
	}

	return c.Render(http.StatusOK, "admin-login-setup", out)
}

// doLogin logs a user in with a username and password.
func (h *Handler) doLogin(c echo.Context) error {
	var (
		username = strings.TrimSpace(c.FormValue("username"))
		password = strings.TrimSpace(c.FormValue("password"))
	)

	if !strHasLen(username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}

	start := time.Now()

	user, err := h.app.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Resist potential constant-time-comparison attacks with a min response time.
	if ms := time.Now().Sub(start).Milliseconds(); ms < 100 {
		time.Sleep(time.Duration(ms))
	}

	// Set the session.
	if err := h.app.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}

// doLoginSetup sets a user up for the first time.
func (h *Handler) doLoginSetup(c echo.Context) error {
	var (
		email     = strings.TrimSpace(c.FormValue("email"))
		username  = strings.TrimSpace(c.FormValue("username"))
		password  = strings.TrimSpace(c.FormValue("password"))
		password2 = strings.TrimSpace(c.FormValue("password2"))
	)

	if !utils.ValidateEmail(email) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "email"))
	}
	if !strHasLen(username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}
	if password != password2 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("users.passwordMismatch"))
	}

	// Create the default "Super Admin".
	r := models.Role{
		Type: models.RoleTypeUser,
		Name: null.NewString("Super Admin", true),
	}
	for p := range h.app.constants.Permissions {
		r.Permissions = append(r.Permissions, p)
	}
	role, err := h.app.core.CreateRole(r)
	if err != nil {
		return err
	}

	// Create the super admin user.
	u := models.User{
		Type:          models.UserTypeUser,
		HasPassword:   true,
		PasswordLogin: true,
		Username:      username,
		Name:          username,
		Password:      null.NewString(password, true),
		Email:         null.NewString(email, true),
		UserRoleID:    role.ID,
		Status:        models.UserStatusEnabled,
	}
	if _, err := h.app.core.CreateUser(u); err != nil {
		return err
	}

	// Log the user in.
	user, err := h.app.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Set the session.
	if err := h.app.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}
