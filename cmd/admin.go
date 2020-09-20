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
	RootURL       string     `json:"rootURL"`
	FromEmail     string     `json:"fromEmail"`
	Messengers    []string   `json:"messengers"`
	MediaProvider string     `json:"mediaProvider"`
	NeedsRestart  bool       `json:"needsRestart"`
	Update        *AppUpdate `json:"update"`
}

// handleGetConfigScript returns general configuration as a Javascript
// variable that can be included in an HTML page directly.
func handleGetConfigScript(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = configScript{
			RootURL:       app.constants.RootURL,
			FromEmail:     app.constants.FromEmail,
			MediaProvider: app.constants.MediaProvider,
		}
	)

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

	var (
		b = bytes.Buffer{}
		j = json.NewEncoder(&b)
	)
	b.Write([]byte(`var CONFIG = `))
	_ = j.Encode(out)
	return c.Blob(http.StatusOK, "application/javascript", b.Bytes())
}

// handleGetDashboardCharts returns chart data points to render ont he dashboard.
func handleGetDashboardCharts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetDashboardCharts.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching dashboard stats: %s", pqErrMsg(err)))
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
			fmt.Sprintf("Error fetching dashboard statsc counts: %s", pqErrMsg(err)))
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
