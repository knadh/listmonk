package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo"
)

type configScript struct {
	RootURL             string          `json:"rootURL"`
	FromEmail           string          `json:"fromEmail"`
	Messengers          []string        `json:"messengers"`
	MediaProvider       string          `json:"mediaProvider"`
	NeedsRestart        bool            `json:"needsRestart"`
	Update              *AppUpdate      `json:"update"`
	Langs               []i18nLang      `json:"langs"`
	EnablePublicSubPage bool            `json:"enablePublicSubscriptionPage"`
	Lang                json.RawMessage `json:"lang"`
}

// handleGetConfigScript returns general configuration as a Javascript
// variable that can be included in an HTML page directly.
func handleGetConfigScript(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = configScript{
			RootURL:             app.constants.RootURL,
			FromEmail:           app.constants.FromEmail,
			MediaProvider:       app.constants.MediaProvider,
			EnablePublicSubPage: app.constants.EnablePublicSubPage,
		}
	)

	// Language list.
	langList, err := geti18nLangList(app.constants.Lang, app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	// Current language.
	out.Lang = json.RawMessage(app.i18n.JSON())

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

	// Write the Javascript variable opening;
	b := bytes.Buffer{}
	b.Write([]byte(`var CONFIG = `))

	// Encode the config payload as JSON and write as the variable's value assignment.
	j := json.NewEncoder(&b)
	if err := j.Encode(out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("admin.errorMarshallingConfig", "error", err.Error()))
	}
	return c.Blob(http.StatusOK, "application/javascript; charset=utf-8", b.Bytes())
}

// handleGetDashboardCharts returns chart data points to render ont he dashboard.
func handleGetDashboardCharts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetDashboardCharts.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching", "name", "dashboard charts", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func handleGetDashboardCounts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetDashboardCounts.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleReloadApp restarts the app.
func handleReloadApp(c echo.Context) error {
	app := c.Get("app").(*App)
	go func() {
		<-time.After(time.Millisecond * 500)
		app.sigChan <- syscall.SIGHUP
	}()
	return c.JSON(http.StatusOK, okResp{true})
}
