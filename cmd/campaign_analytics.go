package main

import (
	"net/http"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// analyticsData is the chart data embedded into the analytics page for the Chart.js renderer.
type analyticsData struct {
	Items           []models.AnalyticsItem          `json:"items"`
	Subscriptions   []models.CampaignAnalyticsCount `json:"subscriptions"`
	Unsubscriptions []models.CampaignAnalyticsCount `json:"unsubscriptions"`
	Views           []models.CampaignAnalyticsCount `json:"views"`
	Clicks          []models.CampaignAnalyticsCount `json:"clicks"`
	Bounces         []models.CampaignAnalyticsCount `json:"bounces"`
	Links           []models.CampaignAnalyticsLink  `json:"links"`
}

// analyticsView is the shared admin view for campaign and list analytics.
type analyticsView struct {
	adminView

	Items                 []models.AnalyticsItem
	Resource              string
	CanViewCampaignCharts bool
	FromInput             string
	ToInput               string
	Interval              string
	DisableTracking       bool
	IndividualTracking    bool

	HasData bool
	Counts  struct {
		Subscriptions   int
		Unsubscriptions int
		Views           int
		Clicks          int
		Bounces         int
	}
	Data analyticsData
}

// ViewCampaignAnalytics renders the HTML view for campaign analytics.
func (a *App) ViewCampaignAnalytics(c echo.Context) error {
	return a.viewAnalytics(c, "campaigns")
}

// ViewListAnalytics renders analytics for selected lists.
func (a *App) ViewListAnalytics(c echo.Context) error {
	return a.viewAnalytics(c, "lists")
}

func (a *App) viewAnalytics(c echo.Context, resource string) error {
	user := auth.GetUser(c)
	ids, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}

	getAll, permittedLists := user.GetPermittedLists(auth.PermTypeGet)
	if resource == "campaigns" {
		getAll, permittedLists = user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)
		getAll = getAll || user.HasPerm(auth.PermCampaignsGetAll)
	}

	items, err := a.core.GetAnalyticsItems(resource, ids, getAll, permittedLists)
	if err != nil {
		return err
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

	data := analyticsView{
		adminView:          newAdminView(c, a.i18n.T("analytics.title"), "", resource+".analytics"),
		Items:              items,
		Resource:           resource,
		FromInput:          from.Format("2006-01-02T15:04"),
		ToInput:            to.Format("2006-01-02T15:04"),
		Interval:           c.QueryParam("interval"),
		DisableTracking:    a.cfg.Privacy.DisableTracking,
		IndividualTracking: a.cfg.Privacy.IndividualTracking,
	}

	data.CanViewCampaignCharts = resource == "campaigns" || (data.Can("campaigns:get", "campaigns:get_all") && data.Can("campaigns:get_analytics"))

	if data.Interval == "" {
		data.Interval = "day"
	}

	// Pull analytics for the selected campaigns or lists.
	if len(items) > 0 {
		var (
			fromStr = from.Format("2006-01-02 15:04:05")
			toStr   = to.Format("2006-01-02 15:04:05")
			ids     = make([]int, len(items))
		)
		for i, item := range items {
			ids[i] = item.ID
		}

		data.HasData = true
		data.Data.Items = items
		getCounts := a.core.GetCampaignAnalyticsCounts
		getLinks := a.core.GetCampaignAnalyticsLinks

		if resource == "lists" {
			getCounts = a.core.GetListAnalyticsCounts
			getLinks = a.core.GetListAnalyticsLinks
			rows, err := getCounts(ids, "subscriptions", fromStr, toStr, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Subscriptions = rows
			data.Counts.Subscriptions = sumAnalyticsCounts(rows)
			rows, err = getCounts(ids, "unsubscriptions", fromStr, toStr, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Unsubscriptions = rows
			data.Counts.Unsubscriptions = sumAnalyticsCounts(rows)
		}
		if data.CanViewCampaignCharts {
			var err error
			data.Data.Views, err = getCounts(ids, "views", fromStr, toStr, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Clicks, err = getCounts(ids, "clicks", fromStr, toStr, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Bounces, err = getCounts(ids, "bounces", fromStr, toStr, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Links, err = getLinks(ids, fromStr, toStr)
			if err != nil {
				return err
			}

			data.Counts.Views = sumAnalyticsCounts(data.Data.Views)
			data.Counts.Clicks = sumAnalyticsCounts(data.Data.Clicks)
			data.Counts.Bounces = sumAnalyticsCounts(data.Data.Bounces)
		}
	}

	return c.Render(http.StatusOK, "admin-analytics", data)
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
		out, err := a.core.GetCampaignAnalyticsLinks(ids, from, to)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	interval := c.QueryParam("interval")
	if interval == "" {
		interval = "day"
	}

	// Get the analytics numbers from the DB for the campaigns.
	out, err := a.core.GetCampaignAnalyticsCounts(ids, typ, from, to, interval)
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
