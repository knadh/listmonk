package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// handleGCSubscribers garbage collects (deletes) orphaned or blocklisted subscribers.
func (h *Handler) handleGCSubscribers(c echo.Context) error {
	typ := c.Param("type")

	var (
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

// handleGCSubscriptions garbage collects (deletes) orphaned or blocklisted subscribers.
func (h *Handler) handleGCSubscriptions(c echo.Context) error {
	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	n, err := h.app.core.DeleteUnconfirmedSubscriptions(t)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// handleGCCampaignAnalytics garbage collects (deletes) campaign analytics.
func (h *Handler) handleGCCampaignAnalytics(c echo.Context) error {
	typ := c.Param("type")

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
	}

	switch typ {
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
