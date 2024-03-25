package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

func OIDCAuth(config Config) echo.MiddlewareFunc {
	provider, err := oidc.NewProvider(context.Background(), config.ProviderURL)
	if err != nil {
		panic(err)
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	oidcConfig := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	pathURL, err := url.Parse(config.RedirectURL)
	if err != nil {
		panic(err)
	}

	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			if c.Request().URL.Path == pathURL.Path {
				oauth2Token, err := oidcConfig.Exchange(c.Request().Context(), c.Request().URL.Query().Get("code"))
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to exchange token: %v", err))
				}

				rawIDToken, ok := oauth2Token.Extra("id_token").(string)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "No id_token field in oauth2 token")
				}

				idToken, err := verifier.Verify(c.Request().Context(), rawIDToken)
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to verify ID Token: %v", err))
				}

				nonce, err := c.Cookie("nonce")
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("nonce cookie not found: %v", err))
				}

				if idToken.Nonce != nonce.Value {
					return echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
				}

				c.SetCookie(&http.Cookie{
					Name:     "id_token",
					Value:    rawIDToken,
					Secure:   true,
					SameSite: http.SameSiteLaxMode,
					Path:     "/",
				})

				// Login success - redirect back to the intended page
				return c.Redirect(302, c.Request().URL.Query().Get("state"))
			}

			// check if request is authenticated
			rawIDToken, err := c.Cookie("id_token")
			if err == nil { // cookie found
				_, err = verifier.Verify(c.Request().Context(), rawIDToken.Value)
				if err == nil {
					return next(c)
				}
			} else if err != http.ErrNoCookie {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}

			// Redirect to login
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
			return c.Redirect(302, oidcConfig.AuthCodeURL(c.Request().URL.RequestURI(), oidc.Nonce(nonce)))
		}
	}
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
