package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// handleGCSubscribers garbage collects (deletes) orphaned or blocklisted subscribers.
func handleGCSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		typ = c.Param("type")
	)

	var (
		n   int
		err error
	)

	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	switch typ {
	case "blocklisted":
		n, err = app.core.DeleteBlocklistedSubscribers(authID)
	case "orphan":
		n, err = app.core.DeleteOrphanSubscribers(authID)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// handleGCSubscriptions garbage collects (deletes) orphaned or blocklisted subscribers.
func handleGCSubscriptions(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)
	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	n, err := app.core.DeleteUnconfirmedSubscriptions(t, authID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// handleGCCampaignAnalytics garbage collects (deletes) campaign analytics.
func handleGCCampaignAnalytics(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		typ = c.Param("type")
	)
	authID := c.Request().Header.Get("X-Auth-ID")

	if authID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "authid is required")
	}

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	switch typ {
	case "all":
		if err := app.core.DeleteCampaignViews(t, authID); err != nil {
			return err
		}
		err = app.core.DeleteCampaignLinkClicks(t, authID)
	case "views":
		err = app.core.DeleteCampaignViews(t, authID)
	case "clicks":
		err = app.core.DeleteCampaignLinkClicks(t, authID)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
