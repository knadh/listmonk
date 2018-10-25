package main

import (
	"net/http"

	"github.com/labstack/echo"
)

// handleGetStats returns a collection of general statistics.
func handleGetStats(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.JSON(http.StatusOK, okResp{app.Runner.GetMessengerNames()})
}
