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
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/zerodha/simplesessions/stores/postgres/v3"
	"github.com/zerodha/simplesessions/v3"
	"golang.org/x/oauth2"
)

const (
	// UserKey is the key on which the User profile is set on echo handlers.
	UserKey    = "auth_user"
	SessionKey = "auth_session"

	SuperAdminRole = 1
)

const (
	sessTypeNative = "native"
	sessTypeOIDC   = "oidc"
)

type OIDCclaim struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Sub           string `json:"sub"`
	Picture       string `json:"picture"`
}

type OIDCConfig struct {
	Enabled      bool   `json:"enabled"`
	ProviderURL  string `json:"provider_url"`
	RedirectURL  string `json:"redirect_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
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
	SetCookie func(cookie *http.Cookie, w interface{}) error
	GetCookie func(name string, r interface{}) (*http.Cookie, error)
	GetUser   func(id int) (models.User, error)
}

type Auth struct {
	tokens map[string]models.User
	sync.RWMutex

	cfg       Config
	oauthCfg  oauth2.Config
	verifier  *oidc.IDTokenVerifier
	sess      *simplesessions.Manager
	sessStore *postgres.Store
	cb        *Callbacks
	log       *log.Logger
}

func New(cfg Config, db *sql.DB, cb *Callbacks, lo *log.Logger) (*Auth, error) {
	a := &Auth{
		cfg: cfg,
		cb:  cb,
		log: lo,
	}

	// Initialize OIDC.
	if cfg.OIDC.Enabled {
		provider, err := oidc.NewProvider(context.Background(), cfg.OIDC.ProviderURL)
		if err != nil {
			lo.Printf("error initializing OIDC OAuth provider: %v", err)
		} else {

			a.verifier = provider.Verifier(&oidc.Config{
				ClientID: cfg.OIDC.ClientID,
			})

			a.oauthCfg = oauth2.Config{
				ClientID:     cfg.OIDC.ClientID,
				ClientSecret: cfg.OIDC.ClientSecret,
				Endpoint:     provider.Endpoint(),
				RedirectURL:  cfg.OIDC.RedirectURL,
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
			}
		}
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
		time.Sleep(time.Hour * 12)
	}()

	return a, nil
}

// SetTokens caches tokens for authenticating API client calls.
func (o *Auth) SetTokens(tokens map[string]models.User) {
	o.Lock()
	defer o.Unlock()

	o.tokens = make(map[string]models.User, len(tokens))
	for userID, u := range tokens {
		o.tokens[userID] = u
	}
}

// GetToken validates an API user+token.
func (o *Auth) GetToken(user string, token string) (models.User, bool) {
	o.RLock()
	t, ok := o.tokens[user]
	o.RUnlock()

	if !ok || subtle.ConstantTimeCompare([]byte(t.Password.String), []byte(token)) != 1 {
		return models.User{}, false
	}

	return t, true
}

// GetOIDCAuthURL returns the OIDC provider's auth URL to redirect to.
func (o *Auth) GetOIDCAuthURL(state, nonce string) string {
	return o.oauthCfg.AuthCodeURL(state, oidc.Nonce(nonce))
}

// ExchangeOIDCToken takes an OIDC authorization code (recieved via redirect from the OIDC provider),
// validates it, and returns an OIDC token for subsequent auth.
func (o *Auth) ExchangeOIDCToken(code, nonce string) (string, OIDCclaim, error) {
	tk, err := o.oauthCfg.Exchange(context.TODO(), code)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	idTk, err := o.verifier.Verify(context.TODO(), rawIDTk)
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
				c.Set(UserKey, echo.NewHTTPError(http.StatusForbidden, err.Error()))
				return next(c)
			}

			// Validate the token.
			user, ok := o.GetToken(key, token)
			if !ok {
				c.Set(UserKey, echo.NewHTTPError(http.StatusForbidden, "invalid API credentials"))
				return next(c)
			}

			// Set the user details on the handler context.
			c.Set(UserKey, user)
			return next(c)
		}

		// It's a cookie based session.
		sess, user, err := o.validateSession(c)
		if err != nil {
			c.Set(UserKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// Set the user details on the handler context.
		c.Set(UserKey, user)
		c.Set(SessionKey, sess)
		return next(c)
	}
}

func (o *Auth) Perm(next echo.HandlerFunc, perm string) echo.HandlerFunc {
	return func(c echo.Context) error {
		u, ok := c.Get(UserKey).(models.User)
		if !ok {
			c.Set(UserKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// If there's no permission set on the handler or if the current user is a super admin, do no checks.
		if perm == "" || u.RoleID == SuperAdminRole {
			return next(c)
		}

		// Check if the current handler's permission is in the user's permission map.
		if _, ok := u.PermissionsMap[perm]; !ok {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("permission denied (%s)", perm))
		}

		return next(c)
	}
}

// SetSession creates and sets a session (post successful login/auth).
func (o *Auth) SetSession(u models.User, oidcToken string, c echo.Context) error {
	sess, err := o.sess.NewSession(c, c)
	if err != nil {
		o.log.Printf("error creating login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	if err := sess.SetMulti(map[string]interface{}{"user_id": u.ID, "oidc_token": oidcToken}); err != nil {
		o.log.Printf("error setting login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	return nil
}

func (o *Auth) validateSession(c echo.Context) (*simplesessions.Session, models.User, error) {
	// Cookie session.
	sess, err := o.sess.Acquire(nil, c, c)
	if err != nil {
		return nil, models.User{}, echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	// Get the session variables.
	vars, err := sess.GetMulti("user_id", "oidc_token")
	if err != nil {
		return nil, models.User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Validate the user ID in the session.
	userID, err := o.sessStore.Int(vars["user_id"], nil)
	if err != nil || userID < 1 {
		o.log.Printf("error fetching session user ID: %v", err)
		return nil, models.User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Fetch user details from the database.
	user, err := o.cb.GetUser(userID)
	if err != nil {
		o.log.Printf("error fetching session user: %v", err)
	}

	return sess, user, err
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