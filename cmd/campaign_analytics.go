package main

import (
	"net/http"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// analyticsCampaign is used to render campaign items in the campaign analytics page selector.
type analyticsCampaign struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// analyticsData is the chart data embedded into the analytics page for the Chart.js renderer.
type analyticsData struct {
	Campaigns []analyticsCampaign             `json:"campaigns"`
	Views     []models.CampaignAnalyticsCount `json:"views"`
	Clicks    []models.CampaignAnalyticsCount `json:"clicks"`
	Bounces   []models.CampaignAnalyticsCount `json:"bounces"`
	Links     []models.CampaignAnalyticsLink  `json:"links"`
}

// campaignAnalyticsView is the admin view for the campaign analytics page.
type campaignAnalyticsView struct {
	adminView

	Campaigns          []analyticsCampaign
	FromInput          string
	ToInput            string
	DisableTracking    bool
	IndividualTracking bool

	HasData bool
	Counts  struct {
		Views   int
		Clicks  int
		Bounces int
	}
	Data analyticsData
}

// ViewCampaignAnalytics renders the HTML view for campaign analytics.
func (a *App) ViewCampaignAnalytics(c echo.Context) error {
	var campaigns []analyticsCampaign
	if ids, err := getQueryInts("id", c.QueryParams()); err == nil {
		for _, id := range ids {
			// Skip campaigns the user doesn't have access to.
			if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
				continue
			}

			camp, err := a.core.GetCampaign(id, "", "")
			if err != nil {
				continue
			}

			campaigns = append(campaigns, analyticsCampaign{ID: camp.ID, Name: camp.Name})
		}
	}

	// Resolve the date range, defaulting to the last 7 days.
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	if t, ok := parseAnalyticsDate(c.QueryParam("from")); ok {
		from = t
	}
	if t, ok := parseAnalyticsDate(c.QueryParam("to")); ok {
		to = t
	}

	data := campaignAnalyticsView{
		adminView:          newAdminView(c, a.i18n.T("analytics.title"), "", "campaigns.analytics"),
		Campaigns:          campaigns,
		FromInput:          from.Format("2006-01-02T15:04"),
		ToInput:            to.Format("2006-01-02T15:04"),
		DisableTracking:    a.cfg.Privacy.DisableTracking,
		IndividualTracking: a.cfg.Privacy.IndividualTracking,
	}

	// Pull analytics for the selected campaigns.
	if len(campaigns) > 0 {
		var (
			fromStr = from.Format("2006-01-02 15:04:05")
			toStr   = to.Format("2006-01-02 15:04:05")
			ids     = make([]int, len(campaigns))
		)
		for i, camp := range campaigns {
			ids[i] = camp.ID
		}

		views, err := a.core.GetCampaignAnalyticsCounts(ids, "views", fromStr, toStr)
		if err != nil {
			return err
		}

		clicks, err := a.core.GetCampaignAnalyticsCounts(ids, "clicks", fromStr, toStr)
		if err != nil {
			return err
		}

		bounces, err := a.core.GetCampaignAnalyticsCounts(ids, "bounces", fromStr, toStr)
		if err != nil {
			return err
		}

		links, err := a.core.GetCampaignAnalyticsLinks(ids, "links", fromStr, toStr)
		if err != nil {
			return err
		}

		data.HasData = true
		data.Data = analyticsData{Campaigns: campaigns, Views: views, Clicks: clicks, Bounces: bounces, Links: links}
		data.Counts.Views = sumAnalyticsCounts(views)
		data.Counts.Clicks = sumAnalyticsCounts(clicks)
		data.Counts.Bounces = sumAnalyticsCounts(bounces)
	}

	return c.Render(http.StatusOK, "admin-campaign-analytics", data)
}

// GetCampaignViewAnalytics retrieves view counts for a campaign.
func (a *App) GetCampaignViewAnalytics(c echo.Context) error {
	ids, err := parseStringIDs(c.Request().URL.Query()["id"])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}

	if len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.missingFields", "name", "`id`"))
	}

	// Ensure the user has access to campaigns via lists.
	for _, id := range ids {
		if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
			return err
		}
	}

	var (
		typ  = c.Param("type")
		from = c.QueryParams().Get("from")
		to   = c.QueryParams().Get("to")
	)
	if !strHasLen(from, 10, 30) || !strHasLen(to, 10, 30) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("analytics.invalidDates"))
	}

	// Campaign link stats.
	if typ == "links" {
		out, err := a.core.GetCampaignAnalyticsLinks(ids, typ, from, to)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get the analytics numbers from the DB for the campaigns.
	out, err := a.core.GetCampaignAnalyticsCounts(ids, typ, from, to)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// parseAnalyticsDate parses a `datetime-local` form value (with or without seconds).
func parseAnalyticsDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}

	var layout string
	switch len(s) {
	case 16:
		layout = "2006-01-02T15:04"
	case 19:
		layout = "2006-01-02T15:04:05"
	default:
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(layout, s, time.Local)

	return t, err == nil
}

// sumAnalyticsCounts returns the total count across the given analytics rows.
func sumAnalyticsCounts(rows []models.CampaignAnalyticsCount) int {
	total := 0
	for _, r := range rows {
		total += r.Count
	}

	return total
}
