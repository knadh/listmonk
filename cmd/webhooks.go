package main

import (
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetWebhookEvents returns the list of available webhook events.
func (a *App) GetWebhookEvents(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{models.AllWebhookEvents()})
}
