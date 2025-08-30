package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/zerodha/simplesessions/stores/postgres/v3"
	"github.com/zerodha/simplesessions/v3"
	"golang.org/x/oauth2"
)

type OIDCclaim struct {
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Sub               string `json:"sub"`
	Picture           string `json:"picture"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

type OIDCConfig struct {
	Enabled           bool   `json:"enabled"`
	ProviderURL       string `json:"provider_url"`
	RedirectURL       string `json:"redirect_url"`
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	AutoCreateUsers   bool   `json:"auto_create_users"`
	DefaultUserRoleID int    `json:"default_user_role_id"`
	DefaultListRoleID int    `json:"default_list_role_id"`
}

type BasicAuthConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	OIDC      OIDCConfig
	BasicAuth BasicAuthConfig
}

// Callbacks takes two callback functions required by simplesessions.
type Callbacks struct {
	SetCookie func(cookie *http.Cookie, w any) error
	GetCookie func(name string, r any) (*http.Cookie, error)
	GetUser   func(id int) (User, error)
}

type Auth struct {
	apiUsers map[string]User
	sync.RWMutex

	cfg       Config
	oauthCfg  oauth2.Config
	verifier  *oidc.IDTokenVerifier
	provider  *oidc.Provider
	sess      *simplesessions.Manager
	sessStore *postgres.Store
	cb        *Callbacks
	log       *log.Logger
}

var sessPruneInterval = time.Hour * 12

// New returns an initialize Auth instance.
func New(cfg Config, db *sql.DB, cb *Callbacks, lo *log.Logger) (*Auth, error) {
	a := &Auth{
		cfg: cfg,
		cb:  cb,
		log: lo,

		apiUsers: map[string]User{},
	}


	// Initialize session manager.
	a.sess = simplesessions.New(simplesessions.Options{
		EnableAutoCreate: false,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			IsHTTPOnly: true,
			MaxAge:     time.Hour * 24 * 7,
		},
	})
	st, err := postgres.New(postgres.Opt{}, db)
	if err != nil {
		return nil, err
	}
	a.sessStore = st
	a.sess.UseStore(st)
	a.sess.SetCookieHooks(cb.GetCookie, cb.SetCookie)

	// Prune dead sessions from the DB periodically.
	go func() {
		if err := st.Prune(); err != nil {
			lo.Printf("error pruning login sessions: %v", err)
		}
		time.Sleep(sessPruneInterval)
	}()

	return a, nil
}

// CacheAPIUsers caches API users for authenticating requests. It wipes
// the existing cache every time and is meant for syncing all API users
// in the database in one shot.
func (o *Auth) CacheAPIUsers(users []User) {
	o.Lock()
	defer o.Unlock()

	o.apiUsers = map[string]User{}
	for _, u := range users {
		o.apiUsers[u.Username] = u
	}
}

// CacheAPIUser caches an API user for authenticating requests.
func (o *Auth) CacheAPIUser(u User) {
	o.Lock()
	o.apiUsers[u.Username] = u
	o.Unlock()
}

// GetAPIToken validates an API user+token.
func (o *Auth) GetAPIToken(user string, token string) (User, bool) {
	o.RLock()
	t, ok := o.apiUsers[user]
	o.RUnlock()

	if !ok || subtle.ConstantTimeCompare([]byte(t.Password.String), []byte(token)) != 1 {
		return User{}, false
	}

	return t, true
}

// initOIDC initializes the OIDC provider, verifier, and OAuth config.
func (o *Auth) initOIDC() error {
	if !o.cfg.OIDC.Enabled {
		return fmt.Errorf("OIDC is not enabled")
	}

	provider, err := oidc.NewProvider(context.Background(), o.cfg.OIDC.ProviderURL)
	if err != nil {
		return fmt.Errorf("error initializing OIDC OAuth provider: %v", err)
	}

	o.verifier = provider.Verifier(&oidc.Config{
		ClientID: o.cfg.OIDC.ClientID,
	})

	o.oauthCfg = oauth2.Config{
		ClientID:     o.cfg.OIDC.ClientID,
		ClientSecret: o.cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  o.cfg.OIDC.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	o.provider = provider

	return nil
}

// getProvider returns the OIDC provider, initializing it if necessary.
func (o *Auth) getProvider() (*oidc.Provider, error) {
	o.Lock()
	defer o.Unlock()

	if o.provider == nil {
		if err := o.initOIDC(); err != nil {
			return nil, err
		}
	}
	return o.provider, nil
}

// getVerifier returns the OIDC verifier, initializing it if necessary.
func (o *Auth) getVerifier() (*oidc.IDTokenVerifier, error) {
	o.Lock()
	defer o.Unlock()

	if o.verifier == nil {
		if err := o.initOIDC(); err != nil {
			return nil, err
		}
	}
	return o.verifier, nil
}

// getOAuthConfig returns the OAuth config, initializing it if necessary.
func (o *Auth) getOAuthConfig() (*oauth2.Config, error) {
	o.Lock()
	defer o.Unlock()

	if o.oauthCfg.ClientID == "" {
		if err := o.initOIDC(); err != nil {
			return nil, err
		}
	}
	return &o.oauthCfg, nil
}

// GetOIDCAuthURL returns the OIDC provider's auth URL to redirect to.
func (o *Auth) GetOIDCAuthURL(state, nonce string) string {
	cfg, err := o.getOAuthConfig()
	if err != nil {
		o.log.Printf("error getting OAuth config: %v", err)
		return ""
	}
	return cfg.AuthCodeURL(state, oidc.Nonce(nonce))
}

// ExchangeOIDCToken takes an OIDC authorization code (recieved via redirect from the OIDC provider),
// validates it, and returns an OIDC token for subsequent auth.
func (o *Auth) ExchangeOIDCToken(code, nonce string) (string, OIDCclaim, error) {
	cfg, err := o.getOAuthConfig()
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error getting OAuth config: %v", err))
	}

	tk, err := cfg.Exchange(context.TODO(), code)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	verifier, err := o.getVerifier()
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error getting verifier: %v", err))
	}

	idTk, err := verifier.Verify(context.TODO(), rawIDTk)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error verifying ID token: %v", err))
	}

	if idTk.Nonce != nonce {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
	}

	var claims OIDCclaim
	if err := idTk.Claims(&claims); err != nil {
		return "", OIDCclaim{}, errors.New("error getting user from OIDC")
	}

	// If claims doesn't have the e-mail, attempt to fetch it from the userinfo endpoint.
	if claims.Email == "" {
		provider, err := o.getProvider()
		if err != nil {
			return "", OIDCclaim{}, fmt.Errorf("error getting provider: %v", err)
		}

		userInfo, err := provider.UserInfo(context.TODO(), oauth2.StaticTokenSource(tk))
		if err != nil {
			return "", OIDCclaim{}, errors.New("error fetching user info from OIDC")
		}

		// Parse the UserInfo claims into the claims struct
		if err := userInfo.Claims(&claims); err != nil {
			return "", OIDCclaim{}, errors.New("error parsing user info claims")
		}
	}

	return rawIDTk, claims, nil
}

// Middleware is the HTTP middleware used for wrapping HTTP handlers registered on the echo router.
// It authorizes token (BasicAuth/token) based and cookie based sessions and on successful auth,
// sets the authenticated User{} on the echo context on the key UserKey. On failure, it sets an Error{}
// instead on the same key.
func (o *Auth) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// It's an `Authorization` header request.
		hdr := strings.TrimSpace(c.Request().Header.Get("Authorization"))

		// If cookie is set, ignore BasicAuth. This is to preserve backwards compatibility
		// in v3 -> v4 upgrade where the user browser sessions would still have old
		// BasicAuth credentials, which no longer work in the new system which expects
		// session cookies instead, which causes a redirect loop despite loggin in and session
		// cookies being set.
		//
		// TODO: This should be removed in a future version.
		if c := strings.TrimSpace(c.Request().Header.Get("Cookie")); strings.Contains(c, "session=") {
			hdr = ""
		}

		if len(hdr) > 0 {
			key, token, err := parseAuthHeader(hdr)
			if err != nil {
				c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, err.Error()))
				return next(c)
			}

			// Validate the token.
			user, ok := o.GetAPIToken(key, token)
			if !ok {
				c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid API credentials"))
				return next(c)
			}

			// Set the user details on the handler context.
			c.Set(UserHTTPCtxKey, user)
			return next(c)
		}

		// Is it a cookie based session?
		sess, user, err := o.validateSession(c)
		if err != nil {
			c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// Set the user details on the handler context.
		c.Set(UserHTTPCtxKey, user)
		c.Set(SessionKey, sess)
		return next(c)
	}
}

// Perm is an HTTP handler middleware that checks if the authenticated user has the required permissions.
func (o *Auth) Perm(next echo.HandlerFunc, perms ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		u, ok := c.Get(UserHTTPCtxKey).(User)
		if !ok {
			c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// If the current user is a Super Admin user, do no checks.
		if u.UserRole.ID == SuperAdminRoleID {
			return next(c)
		}

		// Check if the current handler's permission is in the user's permission map.
		var (
			has  = false
			perm = ""
		)
		for _, perm = range perms {
			if _, ok := u.PermissionsMap[perm]; ok {
				has = true
				break
			}
		}

		if !has {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("permission denied: %s", perm))
		}

		return next(c)
	}
}

// SaveSession creates and sets a session (post successful login/auth).
func (o *Auth) SaveSession(u User, oidcToken string, c echo.Context) error {
	sess, err := o.sess.NewSession(c, c)
	if err != nil {
		o.log.Printf("error creating login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	if err := sess.SetMulti(map[string]any{"user_id": u.ID, "oidc_token": oidcToken}); err != nil {
		o.log.Printf("error setting login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	return nil
}

// validateSession checks if the cookie session is valid (in the DB) and returns the session and user details.
func (o *Auth) validateSession(c echo.Context) (*simplesessions.Session, User, error) {
	// Cookie session.
	sess, err := o.sess.Acquire(context.TODO(), c, c)
	if err != nil {
		return nil, User{}, echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	// Get the session variables.
	vars, err := sess.GetMulti("user_id", "oidc_token")
	if err != nil {
		return nil, User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Validate the user ID in the session.
	userID, err := o.sessStore.Int(vars["user_id"], nil)
	if err != nil || userID < 1 {
		o.log.Printf("error fetching session user ID: %v", err)
		return nil, User{}, echo.NewHTTPError(http.StatusInternalServerError, "invalid session.")
	}

	// Fetch user details from the database.
	user, err := o.cb.GetUser(userID)
	if err != nil {
		o.log.Printf("error fetching session user: %v", err)
	}

	return sess, user, err
}

// GetUser retrieves and returns the User object from an authenticated
// HTTP handler request.
func GetUser(c echo.Context) User {
	return c.Get(UserHTTPCtxKey).(User)
}

// parseAuthHeader parses the Authorization header and returns the api_key and access_token.
func parseAuthHeader(h string) (string, string, error) {
	const authBasic = "Basic"
	const authToken = "token"

	var (
		pair  []string
		delim = ":"
	)

	if strings.HasPrefix(h, authToken) {
		// token api_key:access_token.
		pair = strings.SplitN(strings.Trim(h[len(authToken):], " "), delim, 2)
	} else if strings.HasPrefix(h, authBasic) {
		// HTTP BasicAuth. This is supported for backwards compatibility.
		payload, err := base64.StdEncoding.DecodeString(string(strings.Trim(h[len(authBasic):], " ")))
		if err != nil {
			return "", "", echo.NewHTTPError(http.StatusBadRequest, "invalid Base64 value in Basic Authorization header")
		}
		pair = strings.SplitN(string(payload), delim, 2)
	} else {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "unknown Authorization scheme")
	}

	if len(pair) < 2 {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "api_key:token missing")
	}

	if len(pair[0]) == 0 || len(pair[1]) == 0 {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "empty `api_key` or `token`")
	}

	return pair[0], pair[1], nil
}
