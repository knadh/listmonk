package main

import (
	"fmt"
	"net/http"
	"sort"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

type serverConfig struct {
	Messengers   []string   `json:"messengers"`
	Langs        []i18nLang `json:"langs"`
	Lang         string     `json:"lang"`
	Update       *AppUpdate `json:"update"`
	NeedsRestart bool       `json:"needs_restart"`
	Version      string     `json:"version"`
}

// handleGetServerConfig returns general server config.
func handleGetServerConfig(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = serverConfig{}
	)

	// Language list.
	langList, err := getI18nLangList(app.constants.Lang, app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList
	out.Lang = app.constants.Lang

	// Sort messenger names with `email` always as the first item.
	var names []string
	for name := range app.messengers {
		if name == emailMsgr {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	out.Messengers = append(out.Messengers, emailMsgr)
	out.Messengers = append(out.Messengers, names...)

	app.Lock()
	out.NeedsRestart = app.needsRestart
	out.Update = app.update
	app.Unlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCharts returns chart data points to render on the dashboard.
func handleGetDashboardCharts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	out, err := app.core.GetDashboardCharts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardSubscribersCount returns subscriber count chart data points to render on the dashboard.
func handleGetDashboardSubscribersCount(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		list_id = c.Param("list_id")
		months  = c.QueryParam("months")
	)

	out, err := app.core.GetDashboardSubscribersCount(list_id, months)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardDomainsCount returns subscriber e-mail domains chart data points to render on the dashboard.
func handleGetDashboardDomainsCount(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		list_id = c.Param("list_id")
	)

	out, err := app.core.GetDashboardDomainsCount(list_id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func handleGetDashboardCounts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	out, err := app.core.GetDashboardCounts()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCountries returns subscriber country stats counts to show on the dashboard.
func handleGetDashboardCountries(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		list_id = c.Param("list_id")
	)

	out, err := app.core.GetDashboardCountries(list_id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleReloadApp restarts the app.
func handleReloadApp(c echo.Context) error {
	app := c.Get("app").(*App)
	go func() {
		<-time.After(time.Millisecond * 500)
		app.chReload <- syscall.SIGHUP
	}()
	return c.JSON(http.StatusOK, okResp{true})
}
