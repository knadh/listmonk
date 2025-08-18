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

var oidcProviders = map[string]struct{}{
	"google.com":          {},
	"microsoftonline.com": {},
	"auth0.com":           {},
	"github.com":          {},
}

// LoginPage renders the login page and handles the login form.
func (a *App) LoginPage(c echo.Context) error {
	// Has the user been setup?
	a.Lock()
	needsUserSetup := a.needsUserSetup
	a.Unlock()

	if needsUserSetup {
		return a.LoginSetupPage(c)
	}

	// Process POST login request.
	var loginErr error
	if c.Request().Method == http.MethodPost {
		loginErr = a.doLogin(c)
		if loginErr == nil {
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	// Render the page, with or without POST.
	return a.renderLoginPage(c, loginErr)
}

// LoginSetupPage renders the first time user login page and handles the login form.
func (a *App) LoginSetupPage(c echo.Context) error {
	// Process POST login request.
	var loginErr error
	if c.Request().Method == http.MethodPost {
		loginErr = a.doFirstTimeSetup(c)
		if loginErr == nil {
			a.Lock()
			a.needsUserSetup = false
			a.Unlock()
			return c.Redirect(http.StatusFound, utils.SanitizeURI(c.FormValue("next")))
		}
	}

	// Render the page, with or without POST.
	return a.renderLoginSetupPage(c, loginErr)
}

// Logout logs a user out.
func (a *App) Logout(c echo.Context) error {
	// Delete the session from the DB and cookie.
	sess := c.Get(auth.SessionKey).(*simplesessions.Session)
	_ = sess.Destroy()

	return c.JSON(http.StatusOK, okResp{true})
}

// OIDCLogin initializes an OIDC request and redirects to the OIDC provider for login.
func (a *App) OIDCLogin(c echo.Context) error {
	// Verify that the request came from the login page (CSRF).
	nonce, err := c.Cookie("nonce")
	if err != nil || nonce.Value == "" || nonce.Value != c.FormValue("nonce") {
		return echo.NewHTTPError(http.StatusUnauthorized, a.i18n.T("users.invalidRequest"))
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
		a.log.Printf("error marshalling OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Redirect to the external OIDC provider.
	return c.Redirect(http.StatusFound, a.auth.GetOIDCAuthURL(base64.URLEncoding.EncodeToString(b), nonce.Value))
}

// OIDCFinish receives the redirect callback from the OIDC provider and completes the handshake.
func (a *App) OIDCFinish(c echo.Context) error {
	// Verify that the request actually originated from the login request (which sets the nonce value).
	nonce, err := c.Cookie("nonce")
	if err != nil || nonce.Value == "" {
		return a.renderLoginPage(c, echo.NewHTTPError(http.StatusUnauthorized, a.i18n.T("users.invalidRequest")))
	}

	// Validate the OIDC token.
	oidcToken, claims, err := a.auth.ExchangeOIDCToken(c.Request().URL.Query().Get("code"), nonce.Value)
	if err != nil {
		return a.renderLoginPage(c, err)
	}

	// Validate the state.
	var state oidcState
	stateB, err := base64.URLEncoding.DecodeString(c.QueryParam("state"))
	if err != nil {
		a.log.Printf("error decoding OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}
	if err := json.Unmarshal(stateB, &state); err != nil {
		a.log.Printf("error unmarshalling OIDC state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}
	if state.Nonce != nonce.Value {
		return a.renderLoginPage(c, echo.NewHTTPError(http.StatusUnauthorized, a.i18n.T("users.invalidRequest")))
	}

	// Validate e-mail from the claim.
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		return a.renderLoginPage(c, errors.New(a.i18n.Ts("globals.messages.invalidFields", "name", "email")))
	}
	em, err := mail.ParseAddress(email)
	if err != nil {
		return a.renderLoginPage(c, err)
	}
	email = strings.ToLower(em.Address)
	claims.Email = email

	// Get the user by e-mail received from OIDC.
	user, userErr := a.core.GetUser(0, "", email)
	if userErr != nil {
		// If the user doesn't exist, and auto-creation is enabled, create a new user.
		if httpErr, ok := userErr.(*echo.HTTPError); ok && httpErr.Code == http.StatusNotFound && a.cfg.Security.OIDC.AutoCreateUsers {
			u, err := a.createOIDCUser(claims, c)
			if err != nil {
				return a.renderLoginPage(c, err)
			}
			user = u
			userErr = nil
		} else {
			return a.renderLoginPage(c, userErr)
		}
	}

	// Update the user login state (avatar, logged in date) in the DB.
	if err := a.core.UpdateUserLogin(user.ID, claims.Picture); err != nil {
		return a.renderLoginPage(c, err)
	}

	// Set the session in the DB and cookie.
	if err := a.auth.SaveSession(user, oidcToken, c); err != nil {
		return a.renderLoginPage(c, err)
	}

	// Redirect to the next page.
	return c.Redirect(http.StatusFound, utils.SanitizeURI(state.Next))
}

// renderLoginPage renders the login page and handles the login form.
func (a *App) renderLoginPage(c echo.Context, loginErr error) error {
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	var (
		oidcProviderName = ""
		oidcLogo         = ""
	)
	if a.cfg.Security.OIDC.Enabled {
		// Defaults.
		oidcProviderName = a.cfg.Security.OIDC.ProviderName
		oidcLogo = "oidc.png"

		u, err := url.Parse(a.cfg.Security.OIDC.ProviderURL)
		if err == nil {
			h := strings.Split(u.Hostname(), ".")

			// Get the last two h for the root domain
			prov := ""
			if len(h) >= 2 {
				prov = h[len(h)-2] + "." + h[len(h)-1]
			} else {
				prov = u.Hostname()
			}

			if oidcProviderName == "" {
				oidcProviderName = prov
			}

			// Lookup the logo in the known providers map.
			if _, ok := oidcProviders[prov]; ok {
				oidcLogo = prov + ".png"
			}
		}
	}

	out := loginTpl{
		Title:            a.i18n.T("users.login"),
		PasswordEnabled:  true,
		OIDCProvider:     oidcProviderName,
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
		a.log.Printf("error generating OIDC nonce: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.internalError"))
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
func (a *App) renderLoginSetupPage(c echo.Context, loginErr error) error {
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	out := loginTpl{
		Title:           a.i18n.T("users.login"),
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

// createOIDCUser creates a new user in the DB with the OIDC claims.
func (a *App) createOIDCUser(claims auth.OIDCclaim, c echo.Context) (auth.User, error) {
	name := claims.Name
	if name == "" {
		name = strings.TrimSpace(claims.PreferredUsername)
	}
	if name == "" {
		name = strings.Split(claims.Email, "@")[0]
	}

	var listRoleID *int
	if a.cfg.Security.OIDC.DefaultListRoleID > 0 {
		listRoleID = &a.cfg.Security.OIDC.DefaultListRoleID
	}

	user, err := a.core.CreateUser(auth.User{
		Type:          auth.UserTypeUser,
		HasPassword:   false,
		PasswordLogin: false,
		Username:      claims.Email,
		Name:          name,
		Email:         null.NewString(claims.Email, true),
		UserRoleID:    a.cfg.Security.OIDC.DefaultUserRoleID,
		ListRoleID:    listRoleID,
		Status:        auth.UserStatusEnabled,
	})

	return user, err
}

// doLogin logs a user in with a username and password.
func (a *App) doLogin(c echo.Context) error {
	var (
		username = strings.TrimSpace(c.FormValue("username"))
		password = strings.TrimSpace(c.FormValue("password"))
	)

	if !strHasLen(username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}

	// Log the user in by fetching and verifying credentials from the DB.
	user, err := a.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Resist potential constant-time-comparison attacks with a min response time.
	if ms := time.Since(time.Now()).Milliseconds(); ms < 100 {
		time.Sleep(time.Duration(ms))
	}

	// Set the session in the DB and cookie.
	if err := a.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}

// doFirstTimeSetup sets a user up for the first time.
func (a *App) doFirstTimeSetup(c echo.Context) error {
	var (
		email     = strings.TrimSpace(c.FormValue("email"))
		username  = strings.TrimSpace(c.FormValue("username"))
		password  = strings.TrimSpace(c.FormValue("password"))
		password2 = strings.TrimSpace(c.FormValue("password2"))
	)
	if !utils.ValidateEmail(email) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
	}
	if !strHasLen(username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}
	if password != password2 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.passwordMismatch"))
	}

	// Create the default "Super Admin" with all permissions if it doesn't exist.
	if _, err := a.core.GetRole(auth.SuperAdminRoleID); err != nil {
		r := auth.Role{
			Type: auth.RoleTypeUser,
			Name: null.NewString("Super Admin", true),
		}
		for p := range a.cfg.Permissions {
			r.Permissions = append(r.Permissions, p)
		}

		// Create the role in the DB.
		if _, err := a.core.CreateRole(r); err != nil {
			return err
		}
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
		UserRoleID:    auth.SuperAdminRoleID,
		Status:        auth.UserStatusEnabled,
	}
	if _, err := a.core.CreateUser(u); err != nil {
		return err
	}

	// Log the user in directly.
	user, err := a.core.LoginUser(username, password)
	if err != nil {
		return err
	}

	// Set the session in the DB and cookie.
	if err := a.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return nil
}
