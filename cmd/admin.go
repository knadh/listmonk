package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/knadh/listmonk/internal/captcha"
	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

type serverConfig struct {
	RootURL            string `json:"root_url"`
	FromEmail          string `json:"from_email"`
	PublicSubscription struct {
		Enabled          bool        `json:"enabled"`
		CaptchaEnabled   bool        `json:"captcha_enabled"`
		CaptchaProvider  null.String `json:"captcha_provider"`
		CaptchaKey       null.String `json:"captcha_key"`
		AltchaComplexity int         `json:"altcha_complexity"`
		RedirectURLs     []string    `json:"redirect_urls"`
	} `json:"public_subscription"`
	Privacy struct {
		DisableTracking    bool `json:"disable_tracking"`
		IndividualTracking bool `json:"individual_tracking"`
	} `json:"privacy"`
	MediaProvider string          `json:"media_provider"`
	Messengers    []string        `json:"messengers"`
	Langs         []i18nLang      `json:"langs"`
	Lang          string          `json:"lang"`
	Permissions   json.RawMessage `json:"permissions"`
	Update        *AppUpdate      `json:"update"`
	NeedsRestart  bool            `json:"needs_restart"`
	HasLegacyUser bool            `json:"has_legacy_user"`
	Version       string          `json:"version"`
}

var adminJSI18nKeys = []string{
	"globals.buttons.back",
	"globals.buttons.cancel",
	"globals.buttons.ok",
	"globals.messages.copied",
	"globals.messages.confirm",
	"globals.messages.confirmDelete",
	"globals.messages.created",
	"globals.messages.deleted",
	"globals.messages.deletedCount",
	"globals.messages.numSelected",
	"globals.messages.selectAll",
	"globals.messages.updated",
	"globals.terms.list",
	"globals.terms.second",
	"globals.terms.minute",
	"globals.terms.minute",
	"globals.terms.hour",
	"globals.terms.hour",
	"globals.terms.day",
	"globals.terms.day",
	"globals.terms.month",
	"globals.terms.month",
	"globals.terms.year",
	"globals.terms.year",

	"lists.confirmDelete",
	"lists.confirmSub",
	"lists.optinTo",
	"lists.newList",

	"public.sub",
	"public.subName",
	"subscribers.email",
}

func (a *App) makeServerConfig() (serverConfig, error) {
	out := serverConfig{
		RootURL:       a.urlCfg.RootURL,
		FromEmail:     a.cfg.FromEmail,
		Lang:          a.cfg.Lang,
		Permissions:   a.cfg.PermissionsRaw,
		HasLegacyUser: a.cfg.HasLegacyUser,
		Privacy: struct {
			DisableTracking    bool `json:"disable_tracking"`
			IndividualTracking bool `json:"individual_tracking"`
		}{
			DisableTracking:    a.cfg.Privacy.DisableTracking,
			IndividualTracking: a.cfg.Privacy.IndividualTracking,
		},
	}
	out.PublicSubscription.Enabled = a.cfg.EnablePublicSubPage
	for _, d := range a.cfg.Security.TrustedURLs {
		if d == "*" {
			continue
		}
		out.PublicSubscription.RedirectURLs = append(out.PublicSubscription.RedirectURLs, d)
	}

	// CAPTCHA.
	if a.cfg.Security.Captcha.Altcha.Enabled {
		out.PublicSubscription.CaptchaEnabled = true
		out.PublicSubscription.CaptchaProvider = null.StringFrom(captcha.ProviderAltcha)
		out.PublicSubscription.AltchaComplexity = a.cfg.Security.Captcha.Altcha.Complexity
	} else if a.cfg.Security.Captcha.HCaptcha.Enabled {
		out.PublicSubscription.CaptchaEnabled = true
		out.PublicSubscription.CaptchaProvider = null.StringFrom(captcha.ProviderHCaptcha)
		out.PublicSubscription.CaptchaKey = null.StringFrom(a.cfg.Security.Captcha.HCaptcha.Key)
	}

	out.MediaProvider = a.cfg.MediaUpload.Provider

	// Language list.
	langList, err := getI18nLangList(a.fs)
	if err != nil {
		return out, fmt.Errorf("error loading language list: %v", err)
	}
	out.Langs = langList

	out.Messengers = make([]string, 0, len(a.messengers))
	for _, m := range a.messengers {
		out.Messengers = append(out.Messengers, m.Name())
	}

	a.Lock()
	out.NeedsRestart = a.needsRestart
	out.Update = a.update
	a.Unlock()
	out.Version = versionString

	return out, nil
}

func (a *App) makeAdminJSI18n() map[string]string {
	var lang map[string]string
	if err := json.Unmarshal(a.i18n.JSON(), &lang); err != nil {
		return map[string]string{}
	}

	out := make(map[string]string, len(adminJSI18nKeys))
	for _, key := range adminJSI18nKeys {
		if val, ok := lang[key]; ok {
			out[key] = val
		}
	}
	return out
}

// GetServerConfig returns general server config.
func (a *App) GetServerConfig(c echo.Context) error {
	out, err := a.makeServerConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading server config: %v", err))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCharts returns chart data points to render ont he dashboard.
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
