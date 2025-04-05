package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

type serverConfig struct {
	RootURL       string          `json:"root_url"`
	FromEmail     string          `json:"from_email"`
	Messengers    []string        `json:"messengers"`
	Langs         []i18nLang      `json:"langs"`
	Lang          string          `json:"lang"`
	Permissions   json.RawMessage `json:"permissions"`
	Update        *AppUpdate      `json:"update"`
	NeedsRestart  bool            `json:"needs_restart"`
	HasLegacyUser bool            `json:"has_legacy_user"`
	Version       string          `json:"version"`
}

// GetServerConfig returns general server config.
func (h *Handlers) GetServerConfig(c echo.Context) error {
	out := serverConfig{
		RootURL:       h.app.constants.RootURL,
		FromEmail:     h.app.constants.FromEmail,
		Lang:          h.app.constants.Lang,
		Permissions:   h.app.constants.PermissionsRaw,
		HasLegacyUser: h.app.constants.HasLegacyUser,
	}

	// Language list.
	langList, err := getI18nLangList(h.app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	out.Messengers = make([]string, 0, len(h.app.messengers))
	for _, m := range h.app.messengers {
		out.Messengers = append(out.Messengers, m.Name())
	}

	h.app.Lock()
	out.NeedsRestart = h.app.needsRestart
	out.Update = h.app.update
	h.app.Unlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCharts returns chart data points to render ont he dashboard.
func (h *Handlers) GetDashboardCharts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := h.app.core.GetDashboardCharts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCounts returns stats counts to show on the dashboard.
func (h *Handlers) GetDashboardCounts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := h.app.core.GetDashboardCounts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ReloadApp sends a reload signal to the app, causing a full restart.
func (h *Handlers) ReloadApp(c echo.Context) error {
	go func() {
		<-time.After(time.Millisecond * 500)

		// Send the reload signal to trigger the wait loop in main.
		h.app.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}
