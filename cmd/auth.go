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

// LoginPage renders the login page and handles the login form.
func (h *Handlers) LoginPage(c echo.Context) error {
	// Has the user been setup?
	h.app.Lock()
	needsUserSetup := h.app.needsUserSetup
	h.app.Unlock()

	if needsUserSetup {
		return h.LoginSetupPage(c)
	}

	// Process POST login request.
	var loginErr error
	if c.Request().Method == http.MethodPost {
		loginErr = h.doLogin(c)
		if loginErr == nil {
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	// Render the page, with or without POST.
	return h.renderLoginPage(c, loginErr)
}

// LoginSetupPage renders the first time user login page and handles the login form.
func (h *Handlers) LoginSetupPage(c echo.Context) error {
	// Process POST login request.
	var loginErr error
	if c.Request().Method == http.MethodPost {
		loginErr = h.doFirstTimeSetup(c)
		if loginErr == nil {
			h.app.Lock()
			h.app.needsUserSetup = false
			h.app.Unlock()
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	// Render the page, with or without POST.
	return h.renderLoginSetupPage(c, loginErr)
}

// Logout logs a user out.
func (h *Handlers) Logout(c echo.Context) error {
	// Delete the session from the DB and cookie.
	sess := c.Get(auth.SessionKey).(*simplesessions.Session)
	_ = sess.Destroy()

	return c.JSON(http.StatusOK, okResp{true})
}

// OIDCLogin initializes an OIDC request and redirects to the OIDC provider for login.
func (h *Handlers) OIDCLogin(c echo.Context) error {
	// Verify that the request came from the login page (CSRF).
	nonce, err := c.Cookie("nonce")
	if err != nil || nonce.Value == "" || nonce.Value != c.FormValue("nonce") {
		return echo.NewHTTPError(http.StatusUnauthorized, h.app.i18n.T("users.invalidRequest"))
	}

	// Sanitize the URL and make it relative.
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	// Preparethe OIDC payload to send to the provider.
	state := oidcState{Nonce: nonce.Value, Next: next}

	b, err := json.Marshal(state)
	if err != nil {
		h.app.log.Printf("error marshalling OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, h.app.i18n.T("globals.messages.internalError"))
	}

	// Redirect to the external OIDC provider.
	return c.Redirect(http.StatusFound, h.app.auth.GetOIDCAuthURL(base64.URLEncoding.EncodeToString(b), nonce.Value))
}

// OIDCFinish receives the redirect callback from the OIDC provider and completes the handshake.
func (h *Handlers) OIDCFinish(c echo.Context) error {
	// Verify that the request actually originated from the login request (which sets the nonce value).
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

	// Update the user login state (avatar, logged in date) in the DB.
	if err := h.app.core.UpdateUserLogin(user.ID, claims.Picture); err != nil {
		return h.renderLoginPage(c, err)
	}

	// Set the session in the DB and cookie.
	if err := h.app.auth.SaveSession(user, oidcToken, c); err != nil {
		return h.renderLoginPage(c, err)
	}

	// Redirect to the next page.
	return c.Redirect(http.StatusFound, utils.SanitizeURI(state.Next))
}

// renderLoginPage renders the login page and handles the login form.
func (h *Handlers) renderLoginPage(c echo.Context, loginErr error) error {
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	var (
		oidcProvider = ""
		oidcLogo     = ""
	)
	if h.app.constants.Security.OIDC.Enabled {
		oidcLogo = "oidc.png"
		u, err := url.Parse(h.app.constants.Security.OIDC.Provider)
		if err == nil {
			h := strings.Split(u.Hostname(), ".")

			// Get the last two h for the root domain
			if len(h) >= 2 {
				oidcProvider = h[len(h)-2] + "." + h[len(h)-1]
			} else {
				oidcProvider = u.Hostname()
			}

			// Lookup the logo in the known providers map.
			if _, ok := oidcProviders[oidcProvider]; ok {
				oidcLogo = oidcProvider + ".png"
			}
		}
	}

	out := loginTpl{
		Title:            h.app.i18n.T("users.login"),
		PasswordEnabled:  true,
		OIDCProvider:     oidcProvider,
		OIDCProviderLogo: oidcLogo,
		NextURI:          next,
	}

	// If there was an error in the previous state (POST reqest), set it to render in the template.
	if loginErr != nil {
		if e, ok := loginErr.(*echo.HTTPError); ok {
			out.Error = e.Message.(string)
		} else {
			out.Error = loginErr.Error()
		}
	}

	// Generate and set a nonce for preventing CSRF requests that will be valided in the subsequent requests.
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

	// Render the login page.
	return c.Render(http.StatusOK, "admin-login", out)
}

// renderLoginSetupPage renders the first time user setup page.
func (h *Handlers) renderLoginSetupPage(c echo.Context, loginErr error) error {
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	out := loginTpl{
		Title:           h.app.i18n.T("users.login"),
		PasswordEnabled: true,
		NextURI:         next,
	}

	// If there was an error in the previous state (POST reqest), set it to render in the template.
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
func (h *Handlers) doLogin(c echo.Context) error {
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

	// Log the user in by fetching and verifying credentials from the DB.
	user, err := h.app.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Resist potential constant-time-comparison attacks with a min response time.
	if ms := time.Since(time.Now()).Milliseconds(); ms < 100 {
		time.Sleep(time.Duration(ms))
	}

	// Set the session in the DB and cookie.
	if err := h.app.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}

// doFirstTimeSetup sets a user up for the first time.
func (h *Handlers) doFirstTimeSetup(c echo.Context) error {
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

	// Create the default "Super Admin" with all permission.
	r := auth.Role{
		Type: auth.RoleTypeUser,
		Name: null.NewString("Super Admin", true),
	}
	for p := range h.app.constants.Permissions {
		r.Permissions = append(r.Permissions, p)
	}

	// Create the role in the DB.
	role, err := h.app.core.CreateRole(r)
	if err != nil {
		return err
	}

	// Create the super admin user in the DB.
	u := auth.User{
		Type:          auth.UserTypeUser,
		HasPassword:   true,
		PasswordLogin: true,
		Username:      username,
		Name:          username,
		Password:      null.NewString(password, true),
		Email:         null.NewString(email, true),
		UserRoleID:    role.ID,
		Status:        auth.UserStatusEnabled,
	}
	if _, err := h.app.core.CreateUser(u); err != nil {
		return err
	}

	// Log the user in directly.
	user, err := h.app.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Set the session in the DB and cookie.
	if err := h.app.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}
