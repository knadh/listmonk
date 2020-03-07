package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo"
)

type configScript struct {
	RootURL    string   `json:"rootURL"`
	UploadURI  string   `json:"uploadURI"`
	FromEmail  string   `json:"fromEmail"`
	Messengers []string `json:"messengers"`
}

type dashboardStats struct {
	Stats types.JSONText `db:"stats"`
}

// handleGetConfigScript returns general configuration as a Javascript
// variable that can be included in an HTML page directly.
func handleGetConfigScript(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = configScript{
			RootURL:    app.constants.RootURL,
			FromEmail:  app.constants.FromEmail,
			Messengers: app.manager.GetMessengerNames(),
		}

		b = bytes.Buffer{}
		j = json.NewEncoder(&b)
	)

	b.Write([]byte(`var CONFIG = `))
	_ = j.Encode(out)
	return c.Blob(http.StatusOK, "application/javascript", b.Bytes())
}

// handleGetDashboardStats returns general states for the dashboard.
func handleGetDashboardStats(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out dashboardStats
	)

	if err := app.queries.GetDashboardStats.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching dashboard stats: %s", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out.Stats})
}
