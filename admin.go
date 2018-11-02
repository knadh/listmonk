package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo"
)

type configScript struct {
	RootURL    string   `json:"rootURL"`
	UploadURI  string   `json:"uploadURL"`
	FromEmail  string   `json:"fromEmail"`
	Messengers []string `json:"messengers"`
}

// handleGetStats returns a collection of general statistics.
func handleGetStats(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{app.Runner.GetMessengerNames()})
}

// handleGetConfigScript returns general configuration as a Javascript
// variable that can be included in an HTML page directly.
func handleGetConfigScript(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = configScript{
			RootURL:    app.Constants.RootURL,
			UploadURI:  app.Constants.UploadURI,
			FromEmail:  app.Constants.FromEmail,
			Messengers: app.Runner.GetMessengerNames(),
		}

		b = bytes.Buffer{}
		j = json.NewEncoder(&b)
	)

	b.Write([]byte(`var CONFIG = `))
	j.Encode(out)
	return c.Blob(http.StatusOK, "application/javascript", b.Bytes())
}
