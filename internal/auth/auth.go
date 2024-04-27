package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
)

// UserKey is the key on which the User profile is set on echo handlers.
const UserKey = "auth_user"

// User struct holds the email and name of the authenticatd user.
// It's attached to the echo handler.
type User struct {
	Email   string `json:"name"`
	Name    string `json:"email"`
	Picture string `json:"picture"`
}

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
}

type Auth struct {
	tokens map[string]struct{}
	mut    sync.RWMutex

	cfg      oauth2.Config
	verifier *oidc.IDTokenVerifier
	skipper  middleware.Skipper
}

func New(cfg Config) *Auth {
	provider, err := oidc.NewProvider(context.Background(), cfg.OIDC.ProviderURL)
	if err != nil {
		panic(err)
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDC.ClientID,
	})

	oidcConfig := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDC.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &Auth{
		verifier: verifier,
		cfg:      oidcConfig,
		skipper:  cfg.OIDC.Skipper,
	}
}

// SetTokens remembers a list of string API tokens that are used for authenticating
// API queries.
func (o *Auth) SetTokens(tokens []string) {
	o.mut.Lock()
	defer o.mut.Unlock()

	o.tokens = make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		o.tokens[t] = struct{}{}
	}
}

// CheckToken validates an API token.
func (o *Auth) CheckToken(token string) bool {
	_, ok := o.tokens[token]
	return ok
}

// HandleOIDCCallback is the HTTP handler that handles the post-OIDC provider redirect callback.
func (o *Auth) HandleOIDCCallback(c echo.Context) error {
	tk, err := o.cfg.Exchange(c.Request().Context(), c.Request().URL.Query().Get("code"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	// idTk, err := o.verifier.Verify(c.Request().Context(), rawIDTk)
	// if err != nil {
	// 	return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error verifying ID token: %v", err))
	// }

	// nonce, err := c.Cookie("nonce")
	// if err != nil {
	// 	return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("nonce cookie not found: %v", err))
	// }

	// if idTk.Nonce != nonce.Value {
	// 	return echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
	// }

	c.SetCookie(&http.Cookie{
		Name:     "id_token",
		Value:    rawIDTk,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	return c.Redirect(http.StatusTemporaryRedirect, c.Request().URL.Query().Get("state"))
}

func (o *Auth) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if o.skipper != nil && o.skipper(c) {
			return next(c)
		}

		rawIDTk, err := c.Cookie("id_token")
		if err == nil {
			// Verify the token.
			idTk, err := o.verifier.Verify(c.Request().Context(), rawIDTk.Value)
			if err == nil {
				var user User
				if err := idTk.Claims(&user); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError,
						fmt.Sprintf("error verifying OIDC claim: %v", user))
				}
				fmt.Println(user)
				c.Set(UserKey, user)

				return next(c)
			}
		} else if err != http.ErrNoCookie {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// If the verification failed, redirect to the provider for auth.
		nonce, err := randString(16)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		c.SetCookie(&http.Cookie{
			Name:     "nonce",
			Value:    nonce,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
		return c.Redirect(http.StatusTemporaryRedirect, o.cfg.AuthCodeURL(c.Request().URL.RequestURI(), oidc.Nonce(nonce)))
	}
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
