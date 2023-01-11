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

	switch typ {
	case "blocklisted":
		n, err = app.core.DeleteBlocklistedSubscribers(c.Request().Context())
	case "orphan":
		n, err = app.core.DeleteOrphanSubscribers(c.Request().Context())
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

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	n, err := app.core.DeleteUnconfirmedSubscriptions(c.Request().Context(), t)
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

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	switch typ {
	case "all":
		if err := app.core.DeleteCampaignViews(c.Request().Context(), t); err != nil {
			return err
		}
		err = app.core.DeleteCampaignLinkClicks(c.Request().Context(), t)
	case "views":
		err = app.core.DeleteCampaignViews(c.Request().Context(), t)
	case "clicks":
		err = app.core.DeleteCampaignLinkClicks(c.Request().Context(), t)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
