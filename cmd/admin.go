package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

// handleGetServerConfig returns general server config.
func (h *Handler) handleGetServerConfig(c echo.Context) error {
	out := serverConfig{
		RootURL:       h.app.constants.RootURL,
		FromEmail:     h.app.constants.FromEmail,
		Lang:          h.app.constants.Lang,
		Permissions:   h.app.constants.PermissionsRaw,
		HasLegacyUser: h.app.constants.HasLegacyUser,
	}

	// Language list.
	langList, err := getI18nLangList(h.app.constants.Lang, h.app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	// Sort messenger names with `email` always as the first item.
	var names []string
	for name := range h.app.messengers {
		if name == emailMsgr {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	out.Messengers = append(out.Messengers, emailMsgr)
	out.Messengers = append(out.Messengers, names...)

	h.app.RLock()
	out.NeedsRestart = h.app.needsRestart
	out.Update = h.app.update
	h.app.RUnlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCharts returns chart data points to render ont he dashboard.
func (h *Handler) handleGetDashboardCharts(c echo.Context) error {
	out, err := h.app.core.GetDashboardCharts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func (h *Handler) handleGetDashboardCounts(c echo.Context) error {
	out, err := h.app.core.GetDashboardCounts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleReloadApp restarts the app.
func (h *Handler) handleReloadApp(c echo.Context) error {
	go func() {
		<-time.After(time.Millisecond * 500)
		h.app.chReload <- syscall.SIGHUP
	}()
	return c.JSON(http.StatusOK, okResp{true})
}
