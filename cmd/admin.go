package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"syscall"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

type messengerConfig struct {
	Name             string `json:"name"`
	DefaultFromEmail string `json:"from_email"`
}

type serverConfig struct {
	RootURL            string `json:"root_url"`
	FromEmail          string `json:"from_email"`
	PublicSubscription struct {
		Enabled        bool        `json:"enabled"`
		CaptchaEnabled bool        `json:"captcha_enabled"`
		CaptchaKey     null.String `json:"captcha_key"`
	} `json:"public_subscription"`
	Messengers    []messengerConfig `json:"messengers"`
	Langs         []i18nLang        `json:"langs"`
	Lang          string            `json:"lang"`
	Permissions   json.RawMessage   `json:"permissions"`
	Update        *AppUpdate        `json:"update"`
	NeedsRestart  bool              `json:"needs_restart"`
	HasLegacyUser bool              `json:"has_legacy_user"`
	Version       string            `json:"version"`
}

// GetServerConfig returns general server config.
func (a *App) GetServerConfig(c echo.Context) error {
	out := serverConfig{
		RootURL:       a.urlCfg.RootURL,
		FromEmail:     a.cfg.FromEmail,
		Lang:          a.cfg.Lang,
		Permissions:   a.cfg.PermissionsRaw,
		HasLegacyUser: a.cfg.HasLegacyUser,
		Messengers:    []messengerConfig{{Name: emailMsgr, DefaultFromEmail: a.cfg.FromEmail}},
	}
	out.PublicSubscription.Enabled = a.cfg.EnablePublicSubPage
	if a.cfg.Security.EnableCaptcha {
		out.PublicSubscription.CaptchaEnabled = true
		out.PublicSubscription.CaptchaKey = null.StringFrom(a.cfg.Security.CaptchaKey)
	}

	// Language list.
	langList, err := getI18nLangList(a.fs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	// List messengers user has access to.
	if u, ok := c.Get(auth.UserHTTPCtxKey).(auth.User); ok {
		hasPerm := u.HasPerm(auth.PermMessengersGetAll)
		if hasPerm || len(u.Messengers) > 0 {
			out.Messengers = make([]messengerConfig, 0, len(a.messengers))
			for _, m := range a.messengers {
				if hasPerm || slices.Contains(u.Messengers, m.Name()) {
					msgr := messengerConfig{Name: m.Name()}
					if msgr.Name == emailMsgr {
						msgr.DefaultFromEmail = a.cfg.FromEmail
					} else if em, ok := m.(*email.Emailer); ok {
						msgr.DefaultFromEmail = em.DefaultFromEmail
					}
					out.Messengers = append(out.Messengers, msgr)
				}
			}
		}
	}

	a.Lock()
	out.NeedsRestart = a.needsRestart
	out.Update = a.update
	a.Unlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCharts returns chart data points to render on the dashboard.
func (a *App) GetDashboardCharts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := a.core.GetDashboardCharts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCounts returns stats counts to show on the dashboard.
func (a *App) GetDashboardCounts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := a.core.GetDashboardCounts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ReloadApp sends a reload signal to the app, causing a full restart.
func (a *App) ReloadApp(c echo.Context) error {
	go func() {
		<-time.After(time.Millisecond * 500)

		// Send the reload signal to trigger the wait loop in main.
		a.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}
