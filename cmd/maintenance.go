package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// GCSubscribers garbage collects (deletes) orphaned or blocklisted subscribers.
func (h *Handlers) GCSubscribers(c echo.Context) error {
	var (
		typ = c.Param("type")

		n   int
		err error
	)

	switch typ {
	case "blocklisted":
		n, err = h.app.core.DeleteBlocklistedSubscribers()
	case "orphan":
		n, err = h.app.core.DeleteOrphanSubscribers()
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCSubscriptions garbage collects (deletes) orphaned or blocklisted subscribers.
func (h *Handlers) GCSubscriptions(c echo.Context) error {
	// Validate the date.
	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	// Delete unconfirmed subscriptions from the DB in bulk.
	n, err := h.app.core.DeleteUnconfirmedSubscriptions(t)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCCampaignAnalytics garbage collects (deletes) campaign analytics.
func (h *Handlers) GCCampaignAnalytics(c echo.Context) error {

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	switch c.Param("type") {
	case "all":
		if err := h.app.core.DeleteCampaignViews(t); err != nil {
			return err
		}
		err = h.app.core.DeleteCampaignLinkClicks(t)
	case "views":
		err = h.app.core.DeleteCampaignViews(t)
	case "clicks":
		err = h.app.core.DeleteCampaignLinkClicks(t)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}
