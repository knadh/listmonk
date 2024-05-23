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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vividvilla/simplesessions/stores/postgres"
	"github.com/vividvilla/simplesessions/v2"
	"golang.org/x/oauth2"
)

// UserKey is the key on which the User profile is set on echo handlers.
const UserKey = "auth_user"

const (
	sessTypeNative = "native"
	sessTypeOIDC   = "oidc"
)

type OIDCConfig struct {
	Enabled      bool   `json:"enabled"`
	ProviderURL  string `json:"provider_url"`
	RedirectURL  string `json:"redirect_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper
}

type BasicAuthConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	OIDC      OIDCConfig
	BasicAuth BasicAuthConfig
	LoginURL  string
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
	skipper   middleware.Skipper
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
			panic(err)
		}

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

		a.skipper = cfg.OIDC.Skipper
	}

	// Initialize session manager.
	a.sess = simplesessions.New(simplesessions.Options{
		IsHTTPOnlyCookie: true,
		CookieLifetime:   time.Hour * 24 * 7,
	})
	st, err := postgres.New(postgres.Opt{}, db)
	if err != nil {
		return nil, err
	}
	a.sessStore = st
	a.sess.UseStore(st)
	a.sess.RegisterGetCookie(cb.GetCookie)
	a.sess.RegisterSetCookie(cb.SetCookie)

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
func (o *Auth) ExchangeOIDCToken(code, nonce string) (string, models.User, error) {
	var user models.User

	tk, err := o.oauthCfg.Exchange(context.TODO(), code)
	if err != nil {
		return "", user, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", user, echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	idTk, err := o.verifier.Verify(context.TODO(), rawIDTk)
	if err != nil {
		return "", user, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error verifying ID token: %v", err))
	}

	if idTk.Nonce != nonce {
		return "", user, echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
	}

	if err := idTk.Claims(&user); err != nil {
		return "", user, errors.New("error getting user from OIDC")
	}

	return rawIDTk, user, nil
}

// Middleware is the HTTP middleware used for wrapping HTTP handlers registered on the echo router.
// It authorizes token (BasicAuth/token) based and cookie based sessions.
func (o *Auth) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// It's an `Authorization` header request.
		hdr := c.Response().Header().Get("Authorization")
		if len(hdr) > 0 {
			key, token, err := parseAuthHeader(hdr)
			if err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}

			// Validate the token.
			user, ok := o.GetToken(key, token)
			if !ok {
				return echo.NewHTTPError(http.StatusForbidden, "invalid token:secret")
			}

			// Set the user details on the handler context.
			c.Set(UserKey, user)
			return next(c)
		}

		// It's a cookie based session.
		user, err := o.validateSession(c)
		if err != nil {
			u, _ := url.Parse(o.cfg.LoginURL)
			q := url.Values{}
			q.Set("next", c.Request().RequestURI)
			u.RawQuery = q.Encode()

			return c.Redirect(http.StatusTemporaryRedirect, u.String())
		}

		// Set the user details on the handler context.
		c.Set(UserKey, user)
		return next(c)
	}
}

// SetSession creates and sets a session (post successful login/auth).
func (o *Auth) SetSession(u models.User, oidcToken string, c echo.Context) error {
	sess, err := o.sess.Acquire(c, c, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	// sess, err := simplesessions.NewSession(o.sess, c, c)
	// if err != nil {
	// 	o.log.Printf("error creating login session: %v", err)
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	// }

	if err := sess.SetMulti(map[string]interface{}{"user_id": u.ID, "oidc_token": oidcToken}); err != nil {
		o.log.Printf("error setting login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}
	if err := sess.Commit(); err != nil {
		o.log.Printf("error committing login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	return nil
}

func (o *Auth) validateSession(c echo.Context) (models.User, error) {
	// Cookie session.
	sess, err := o.sess.Acquire(c, c, nil)
	if err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	// Get the session variables.
	vars, err := sess.GetMulti("user_id", "oidc_token")
	if err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Validate the user ID in the session.
	userID, err := o.sessStore.Int(vars["user_id"], nil)
	if err != nil || userID < 1 {
		o.log.Printf("error fetching session user ID: %v", err)
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// If it's an OIDC session, validate the claim.
	if vars["oidc_token"] != "" {
		if !o.cfg.OIDC.Enabled {
			return models.User{}, echo.NewHTTPError(http.StatusForbidden, "OIDC aut his not enabled.")
		}
		if _, err := o.verifyOIDC(vars["oidc_token"].(string), c); err != nil {
			return models.User{}, err
		}
	}

	// Fetch user details from the database.
	user, err := o.cb.GetUser(userID)
	return user, err
}

func (o *Auth) verifyOIDC(token string, c echo.Context) (models.User, error) {
	idTk, err := o.verifier.Verify(c.Request().Context(), token)
	if err != nil {
		return models.User{}, err
	}

	var user models.User
	if err := idTk.Claims(&user); err != nil {
		return user, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("error verifying OIDC claim: %v", user))
	}

	if user.ID < 1 {
		return user, echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("invalid user ID in OIDC: %v", user))
	}

	return user, err
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
