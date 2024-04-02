package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
)

type Config struct {
	ProviderURL  string
	RedirectURL  string
	ClientID     string
	ClientSecret string

	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper
}

type OIDC struct {
	cfg      oauth2.Config
	verifier *oidc.IDTokenVerifier
	skipper  middleware.Skipper
}

func New(cfg Config) *OIDC {
	provider, err := oidc.NewProvider(context.Background(), cfg.ProviderURL)
	if err != nil {
		panic(err)
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	oidcConfig := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &OIDC{
		verifier: verifier,
		cfg:      oidcConfig,
		skipper:  cfg.Skipper,
	}
}

// HandleCallback is the HTTP handler that handles the post-OIDC provider redirect callback.
func (o *OIDC) HandleCallback(c echo.Context) error {
	tk, err := o.cfg.Exchange(c.Request().Context(), c.Request().URL.Query().Get("code"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	idTk, err := o.verifier.Verify(c.Request().Context(), rawIDTk)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error verifying ID token: %v", err))
	}

	nonce, err := c.Cookie("nonce")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("nonce cookie not found: %v", err))
	}

	if idTk.Nonce != nonce.Value {
		return echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
	}

	c.SetCookie(&http.Cookie{
		Name:     "id_token",
		Value:    rawIDTk,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	return c.Redirect(http.StatusTemporaryRedirect, c.Request().URL.Query().Get("state"))
}

func (o *OIDC) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if o.skipper != nil && o.skipper(c) {
			return next(c)
		}

		rawIDTk, err := c.Cookie("id_token")
		if err == nil {
			// Verify the token.
			_, err = o.verifier.Verify(c.Request().Context(), rawIDTk.Value)
			if err == nil {
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
			SameSite: http.SameSiteStrictMode,
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
