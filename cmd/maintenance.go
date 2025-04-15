package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// GCSubscribers garbage collects (deletes) orphaned or blocklisted subscribers.
func (a *App) GCSubscribers(c echo.Context) error {
	var (
		typ = c.Param("type")

		n   int
		err error
	)

	switch typ {
	case "blocklisted":
		n, err = a.core.DeleteBlocklistedSubscribers()
	case "orphan":
		n, err = a.core.DeleteOrphanSubscribers()
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCSubscriptions garbage collects (deletes) orphaned or blocklisted subscribers.
func (a *App) GCSubscriptions(c echo.Context) error {
	// Validate the date.
	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	// Delete unconfirmed subscriptions from the DB in bulk.
	n, err := a.core.DeleteUnconfirmedSubscriptions(t)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCCampaignAnalytics garbage collects (deletes) campaign analytics.
func (a *App) GCCampaignAnalytics(c echo.Context) error {

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	switch c.Param("type") {
	case "all":
		if err := a.core.DeleteCampaignViews(t); err != nil {
			return err
		}
		err = a.core.DeleteCampaignLinkClicks(t)
	case "views":
		err = a.core.DeleteCampaignViews(t)
	case "clicks":
		err = a.core.DeleteCampaignLinkClicks(t)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
