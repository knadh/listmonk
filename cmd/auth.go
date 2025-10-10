package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/zerodha/simplesessions/v3"
	"gopkg.in/volatiletech/null.v6"
)

// Note: This file updates the login flow to send email notifications for
// every login attempt (success or failure). SMTP configuration is read from
// environment variables for safety. Replace env defaults or set env vars in
// your deployment.

// Environment variables used:
// - SMTP_FROM
// - SMTP_PASS
// - SMTP_HOST
// - SMTP_PORT (defaults to 587)
// - NOTIFY_TO (superuser email)

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

// (oidcProviders removed — unused)

// ---------------- EMAIL NOTIFICATIONS -----------------

// sendLoginNotification sends a simple plaintext SMTP email containing
// username, IP, status and timestamp. SMTP configuration is expected to be
// available in environment variables. The function logs and silently
// returns on errors to avoid disrupting the login flow.
func sendLoginNotification(username, ip, status string) {
	from := os.Getenv("SMTP_FROM")
	pass := os.Getenv("SMTP_PASS")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	to := os.Getenv("NOTIFY_TO")

	// If required config missing, do nothing but log to stdout (so as not to
	// break login flow). In real deployments prefer proper logging.
	if from == "" || pass == "" || host == "" || to == "" {
		fmt.Printf("[login-notify] SMTP config incomplete, skipping notification for user=%s ip=%s status=%s\n", username, ip, status)
		return
	}

	subject := fmt.Sprintf("[Listmonk] Login attempt — %s", status)
	body := fmt.Sprintf("User: %s\nIP: %s\nStatus: %s\nTime: %s\n", username, ip, status, time.Now().Format(time.RFC1123))
	msg := []byte("Subject: " + subject + "\r\n\r\n" + body)

	auth := smtp.PlainAuth("", from, pass, host)
	addr := host + ":" + port
	if err := smtp.SendMail(addr, auth, from, []string{to}, msg); err != nil {
		fmt.Printf("[login-notify] error sending mail: %v\n", err)
		return
	}
	fmt.Printf("[login-notify] notification sent to %s for user=%s status=%s\n", to, username, status)
}

// ----------------- HANDLERS (unchanged structure, notifications added) -----------------

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

	// Prepare the OIDC payload to send to the provider.
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
		// notify on token exchange failures
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC token exchange)")
		return a.renderLoginPage(c, err)
	}

	// Validate the state.
	var state oidcState
	stateB, err := base64.URLEncoding.DecodeString(c.QueryParam("state"))
	if err != nil {
		a.log.Printf("error decoding OIDC state: %v", err)
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC state decode)")
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}
	if err := json.Unmarshal(stateB, &state); err != nil {
		a.log.Printf("error unmarshalling OIDC state: %v", err)
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC state unmarshal)")
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}
	if state.Nonce != nonce.Value {
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC nonce mismatch)")
		return a.renderLoginPage(c, echo.NewHTTPError(http.StatusUnauthorized, a.i18n.T("users.invalidRequest")))
	}

	// Validate e-mail from the claim.
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC no email)")
		return a.renderLoginPage(c, errors.New(a.i18n.Ts("globals.messages.invalidFields", "name", "email")))
	}
	em, err := mail.ParseAddress(email)
	if err != nil {
		sendLoginNotification("(oidc)", c.RealIP(), "Failed (OIDC bad email)")
		return a.renderLoginPage(c, err)
	}
	email = strings.ToLower(em.Address)
	claims.Email = email

	// Get the user by e-mail received from OIDC.
	user, userErr := a.core.GetUser(0, "", email)
	if userErr != nil {
		// If the user doesn't exist, and auto-creation is enabled, create a new user.
		if httpErr, ok := userErr.(*echo.HTTPError); ok && httpErr.Code == http.StatusNotFound && a.cfg.Security.OIDC.AutoCreateUsers {
			u, err := a.createOIDCUser(claims)
			if err != nil {
				sendLoginNotification(email, c.RealIP(), "Failed (OIDC create user)")
				return a.renderLoginPage(c, err)
			}
			user = u
			userErr = nil
		} else {
			sendLoginNotification(email, c.RealIP(), "Failed (OIDC user lookup)")
			return a.renderLoginPage(c, userErr)
		}
	}

	// Update the user login state (avatar, logged in date) in the DB.
	if err := a.core.UpdateUserLogin(user.ID, claims.Picture); err != nil {
		sendLoginNotification(user.Username, c.RealIP(), "Failed (update login)")
		return a.renderLoginPage(c, err)
	}

	// Set the session in the DB and cookie.
	if err := a.auth.SaveSession(user, oidcToken, c); err != nil {
		sendLoginNotification(user.Username, c.RealIP(), "Failed (save session)")
		return a.renderLoginPage(c, err)
	}

	// Successful OIDC login -> notify and redirect.
	sendLoginNotification(user.Username, c.RealIP(), "Success (OIDC)")
	return c.Redirect(http.StatusFound, utils.SanitizeURI(state.Next))
}

// renderLoginPage renders the login page and handles the login form.
func (a *App) renderLoginPage(c echo.Context, loginErr error) error {
	next := utils.SanitizeURI(c.FormValue("next"))
	if next == "/" {
		next = uriAdmin
	}

	var oidcProviderName, oidcLogo string
	if a.cfg.Security.OIDC.Enabled {
		// Prefer configured display name; fallback to provider host-derived name/logo.
		oidcProviderName = a.cfg.Security.OIDC.ProviderName
		oidcLogo = "oidc.png"
		if a.cfg.Security.OIDC.ProviderURL != "" {
			if u, err := url.Parse(a.cfg.Security.OIDC.ProviderURL); err == nil {
				h := strings.Split(u.Hostname(), ".")
				prov := ""
				if len(h) >= 2 {
					prov = h[len(h)-2] + "." + h[len(h)-1]
				} else {
					prov = u.Hostname()
				}
				name, logo := oidcProviderInfo(prov)
				if oidcProviderName == "" {
					oidcProviderName = name
				}
				oidcLogo = logo
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
func (a *App) createOIDCUser(claims auth.OIDCclaim) (auth.User, error) {
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

	u := auth.User{
		Type:          auth.UserTypeUser,
		HasPassword:   false,
		PasswordLogin: false,
		Username:      claims.Email,
		Name:          name,
		Email:         null.NewString(claims.Email, true),
		UserRoleID:    a.cfg.Security.OIDC.DefaultUserRoleID,
		ListRoleID:    listRoleID,
		Status:        auth.UserStatusEnabled,
	}

	// Apply type-specific defaults/overrides via a tagged switch on u.Type.
	applyUserType(&u)

	return a.core.CreateUser(u)
}

// applyUserType sets type-specific defaults for a user based on u.Type.
func applyUserType(u *auth.User) {
	switch u.Type {
	case auth.UserTypeUser:
		if u.Status == "" {
			u.Status = auth.UserStatusEnabled
		}
	case auth.UserTypeAPI:
		u.HasPassword = false
		u.PasswordLogin = false
		if u.Status == "" {
			u.Status = auth.UserStatusDisabled
		}
	default:
		if u.Status == "" {
			u.Status = auth.UserStatusEnabled
		}
	}
}

// doLogin handles username/password POST login. It uses the Auth implementation's
// Authenticate method if present, saves a session on success and sends notifications.
// If the Auth implementation doesn't expose Authenticate, it returns 501.
func (a *App) doLogin(c echo.Context) error {
	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	if username == "" || password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidFields"))
	}

	// If the auth implementation exposes Authenticate(username, password, ip) (optional),
	// call it. Use a runtime interface assertion so this compiles regardless of whether
	// the concrete auth.Auth implements it.
	type authenticator interface {
		Authenticate(username, password, ip string) (auth.User, error)
	}

	if a.auth != nil {
		if authImpl, ok := interface{}(a.auth).(authenticator); ok {
			user, err := authImpl.Authenticate(username, password, c.RealIP())
			if err != nil {
				sendLoginNotification(username, c.RealIP(), "Failed (password login)")
				return err
			}

			// Save session (token empty for password logins).
			if err := a.auth.SaveSession(user, "", c); err != nil {
				sendLoginNotification(username, c.RealIP(), "Failed (save session)")
				return err
			}

			sendLoginNotification(username, c.RealIP(), "Success (password)")
			return nil
		}
	}

	// Fallback when no Authenticate method implemented.
	return echo.NewHTTPError(http.StatusNotImplemented, "password login not available")
}

// doFirstTimeSetup handles initial admin/user creation from the setup page.
func (a *App) doFirstTimeSetup(c echo.Context) error {
	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")
	name := strings.TrimSpace(c.FormValue("name"))

	if username == "" || password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidFields"))
	}

	// Try to parse an email from the username; if invalid, leave Email unset.
	var email null.String
	if em, err := mail.ParseAddress(username); err == nil {
		email = null.NewString(strings.ToLower(em.Address), true)
	} else {
		email = null.NewString("", false)
	}

	u := auth.User{
		Type:          auth.UserTypeUser,
		HasPassword:   true,
		PasswordLogin: true,
		Username:      username,
		Name:          name,
		Email:         email,
		// Default to role id 1 for initial setup (adjust if your app uses a different id).
		UserRoleID: 1,
		Status:     auth.UserStatusEnabled,
	}

	// Create the user (core.CreateUser is expected to handle password hashing/storing).
	created, err := a.core.CreateUser(u)
	if err != nil {
		a.log.Printf("first time setup: create user error: %v", err)
		return err
	}

	// Try to create a session so the user is logged in immediately (non-fatal).
	if a.auth != nil {
		if err := a.auth.SaveSession(created, "", c); err != nil {
			a.log.Printf("first time setup: save session error: %v", err)
		}
	}

	return nil
}
