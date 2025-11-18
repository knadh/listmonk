package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/notifs"
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

type resetEntry struct {
	Token     string
	Email     string
	Timestamp time.Time
}

type forgotPasswordTpl struct {
	Title       string
	Description string
	Error       string
}

type resetPasswordTpl struct {
	Title       string
	Description string
	Token       string
	Email       string
	Error       string
}

var (
	oidcProviders = map[string]struct{}{
		"google.com":          {},
		"microsoftonline.com": {},
		"auth0.com":           {},
		"github.com":          {},
	}

	// In-memory map to store password reset tokens. This is a good-enough, simple implementation
	// that's a fair tradeoff compared to full fledged database state management for reset tokens.
	pwdResetTokens = make(map[string]resetEntry)
	resetMu        sync.Mutex
)

const passwordExpiryTTL = 30 * time.Minute

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
		startTime = time.Now()
		username  = strings.TrimSpace(c.FormValue("username"))
		password  = strings.TrimSpace(c.FormValue("password"))
	)

	// Ensure timing mitigation is applied regardless of early returns
	defer func() {
		if elapsed := time.Since(startTime).Milliseconds(); elapsed < 100 {
			time.Sleep(time.Duration(100-elapsed) * time.Millisecond)
		}
	}()

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

// ForgotPage renders the forgot password page and handles the forgot password form.
func (a *App) ForgotPage(c echo.Context) error {
	// Process the forgot password request.
	if c.Request().Method == http.MethodPost {
		return a.doForgotPassword(c)
	}

	// Render the forgot page.
	out := forgotPasswordTpl{Title: a.i18n.T("users.forgotPassword")}
	return c.Render(http.StatusOK, "admin-forgot-password", out)
}

// ResetPage renders the reset password page and handles the reset password form.
func (a *App) ResetPage(c echo.Context) error {
	var (
		token = strings.TrimSpace(c.QueryParam("token"))
		email = strings.ToLower(strings.TrimSpace(c.QueryParam("email")))
	)

	// Validate token and email.
	resetMu.Lock()
	entry, exists := pwdResetTokens[email]
	resetMu.Unlock()

	if !exists || entry.Token != token {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	// Check if the token has expired.
	if time.Since(entry.Timestamp) > passwordExpiryTTL {
		resetMu.Lock()
		delete(pwdResetTokens, email)
		resetMu.Unlock()

		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	// Validate that the user exists.
	_, err := a.core.GetUser(0, "", email)
	if err != nil {
		resetMu.Lock()
		delete(pwdResetTokens, email)
		resetMu.Unlock()
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	// Process the reset password request form with the new passwords.
	if c.Request().Method == http.MethodPost {
		return a.doResetPassword(c, entry)
	}

	// Render the reset password form for GET request.
	return a.renderResetPasswordPage(c, token, email, "")
}

// renderResetPasswordPage renders the reset password page.
func (a *App) renderResetPasswordPage(c echo.Context, token, email, errMsg string) error {
	out := resetPasswordTpl{
		Title: a.i18n.T("users.resetPassword"),
		Token: token,
		Email: email,
		Error: errMsg,
	}
	return c.Render(http.StatusOK, "admin-reset-password", out)
}

// doForgotPassword handles the forgot password form submission.
func (a *App) doForgotPassword(c echo.Context) error {
	var (
		email = strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	)

	// Validate email format.
	if !utils.ValidateEmail(email) {
		return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.resetLinkSent")))
	}

	// Get the user by email.
	user, err := a.core.GetUser(0, "", email)
	if err != nil {
		return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.resetLinkSent")))
	}

	// If the password login is disabled, do not proceed, but show success message to prevent email enumeration.
	if !user.PasswordLogin {
		return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.resetLinkSent")))
	}

	// Generate a random token.
	token, err := generateRandomString(64)
	if err != nil {
		a.log.Printf("error generating reset token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Store the reset entry in the map.
	resetMu.Lock()
	pwdResetTokens[email] = resetEntry{
		Token:     token,
		Email:     email,
		Timestamp: time.Now(),
	}
	resetMu.Unlock()

	// Prepare the reset URL.
	resetURL := fmt.Sprintf("%s/admin/reset?token=%s&email=%s", a.urlCfg.RootURL, token, url.QueryEscape(email))

	// Prepare the email.
	var msg bytes.Buffer
	data := struct {
		ResetURL string
		L        *i18n.I18n
	}{
		ResetURL: resetURL,
		L:        a.i18n,
	}

	// Render the email template.
	if err := notifs.Tpls.ExecuteTemplate(&msg, notifs.TplForgotPassword, data); err != nil {
		a.log.Printf("error compiling notification template '%s': %v", notifs.TplForgotPassword, err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	subject, body := notifs.GetTplSubject(a.i18n.T("email.forgotPassword.subject"), msg.Bytes())

	// Send the email.
	if err := a.emailMsgr.Push(models.Message{
		From:    a.cfg.FromEmail,
		To:      []string{email},
		Subject: subject,
		Body:    body,
	}); err != nil {
		a.log.Printf("error sending reset email: %s", err)
	}

	// Show the success e-mail nonetheless to prevent e-mail enumeration.
	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.resetLinkSent")))
}

// doResetPassword handles the reset password form submission.
func (a *App) doResetPassword(c echo.Context, r resetEntry) error {
	var (
		password  = c.FormValue("password")
		password2 = c.FormValue("password2")
	)

	// Validate password.
	if !strHasLen(password, 8, stdInputMaxLen) {
		return a.renderResetPasswordPage(c, r.Token, r.Email, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}
	if password != password2 {
		return a.renderResetPasswordPage(c, r.Token, r.Email, a.i18n.T("users.passwordMismatch"))
	}

	// Get the user.
	user, err := a.core.GetUser(0, "", r.Email)
	if err != nil {
		resetMu.Lock()
		delete(pwdResetTokens, r.Email)
		resetMu.Unlock()

		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	// Password login is disabled for the user.
	if !user.PasswordLogin {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("public.invalidFeature")))
	}

	user.Password = null.NewString(password, true)
	if _, err := a.core.UpdateUserProfile(user.ID, user); err != nil {
		a.log.Printf("error updating user password: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Clean up the reset token.
	resetMu.Lock()
	delete(pwdResetTokens, r.Email)
	resetMu.Unlock()

	// Log the user in directly without forcing a manual login right after password change.
	if err := a.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	// Redirect to the admin page.
	return c.Redirect(http.StatusFound, uriAdmin)
}
