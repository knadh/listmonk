package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
)

// analyticsData is the chart data embedded into the analytics page for the Chart.js renderer.
type analyticsData struct {
	Interval        string                          `json:"interval"`
	Items           []models.AnalyticsItem          `json:"items"`
	Subscriptions   []models.CampaignAnalyticsCount `json:"subscriptions"`
	Unsubscriptions []models.CampaignAnalyticsCount `json:"unsubscriptions"`
	Views           []models.CampaignAnalyticsCount `json:"views"`
	Clicks          []models.CampaignAnalyticsCount `json:"clicks"`
	UniqueViews     []models.CampaignAnalyticsCount `json:"unique_views,omitempty"`
	UniqueClicks    []models.CampaignAnalyticsCount `json:"unique_clicks,omitempty"`
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
	ShowUnique            bool

	Tab             string
	CanViewActivity bool
	Page            models.PageProps
	Activity        []models.CampaignActivity
	ActivityEvents  []models.CampaignActivityEvent
	Subscriber      *models.Subscriber

	HasData bool
	Counts  struct {
		Subscriptions   int
		Unsubscriptions int
		Views           int
		Clicks          int
		UniqueViews     int
		UniqueClicks    int
		Bounces         int
	}
	Data analyticsData
}

// ViewCampaignAnalytics renders the HTML view for campaign analytics.
func (a *App) ViewCampaignAnalytics(c *echo.Context) error {
	return a.viewAnalytics(c, models.ResourceCampaigns)
}

// ViewListAnalytics renders analytics for selected lists.
func (a *App) ViewListAnalytics(c *echo.Context) error {
	return a.viewAnalytics(c, models.ResourceLists)
}

func (a *App) viewAnalytics(c *echo.Context, resource string) error {
	user := auth.GetUser(c)
	ids, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}

	getAll, permittedLists := user.GetPermittedLists(auth.PermTypeGet)
	if resource == models.ResourceCampaigns {
		getAll, permittedLists = user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)
		getAll = getAll || user.HasPerm(auth.PermCampaignsGetAll)
	}

	items, err := a.core.GetAnalyticsItems(resource, ids, getAll, permittedLists)
	if err != nil {
		return err
	}

	// Resolve the date range, defaulting to the last 7 days.
	var (
		to   = time.Now()
		from = to.AddDate(0, 0, -7)
	)
	if t, ok := parseDateTime(c.QueryParam("from")); ok {
		from = t
	}
	if t, ok := parseDateTime(c.QueryParam("to")); ok {
		to = t
	}

	data := analyticsView{
		adminView:          newAdminView(c, a.i18n.T("globals.terms.analytics"), "", resource+".analytics"),
		Items:              items,
		HasData:            len(items) > 0,
		Resource:           resource,
		FromInput:          from.Format(time.RFC3339),
		ToInput:            to.Format(time.RFC3339),
		Interval:           c.QueryParam("interval"),
		DisableTracking:    a.cfg.Privacy.DisableTracking,
		IndividualTracking: a.cfg.Privacy.IndividualTracking,
		ShowUnique:         a.cfg.Privacy.IndividualTracking && c.QueryParam("unique") == "true",
	}

	// Filtered object IDs.
	ids = make([]int, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}

	if data.Interval == "" {
		data.Interval = models.AnalyticsIntervalDay
	}
	data.Data.Interval = data.Interval

	// Set the query params for the paginator and the activity tab.
	q := c.QueryParams()
	q.Set("from", data.FromInput)
	q.Set("to", data.ToInput)

	data.Page = models.NewPageProps(q, 0, 1, 20)
	data.CanViewActivity = resource == models.ResourceCampaigns && data.Can(auth.PermSubscribersGet, auth.PermSubscribersGetAll)
	if c.QueryParam("tab") == "activity" {
		if !data.CanViewActivity {
			return auth.ErrPermDenied
		}

		data.Tab = "activity"
		if from.After(to) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("analytics.invalidDates"))
		}

		// Load activity for the selected campaigns.
		if err := a.loadCampaignActivity(c, &data, ids); err != nil {
			return err
		}

		return c.Render(http.StatusOK, "admin-analytics", data)
	}

	data.CanViewCampaignCharts = resource == models.ResourceCampaigns ||
		(data.Can(auth.PermCampaignsGet, auth.PermCampaignsGetAll) && data.Can(auth.PermCampaignsGetAnalytics))

	// Pull analytics for the selected campaigns or lists.
	if len(items) > 0 {
		data.Data.Items = items
		getCounts := a.core.GetCampaignAnalyticsCounts
		getLinks := a.core.GetCampaignAnalyticsLinks

		if resource == models.ResourceLists {
			getCounts = a.core.GetListAnalyticsCounts
			getLinks = a.core.GetListAnalyticsLinks
			rows, err := getCounts(ids, models.AnalyticsTypeSubscriptions, data.FromInput, data.ToInput, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Subscriptions = rows
			data.Counts.Subscriptions = sumAnalyticsCounts(rows)
			rows, err = getCounts(ids, models.AnalyticsTypeUnsubscriptions, data.FromInput, data.ToInput, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Unsubscriptions = rows
			data.Counts.Unsubscriptions = sumAnalyticsCounts(rows)
		}

		if data.CanViewCampaignCharts {
			var err error
			data.Data.Views, err = getCounts(ids, models.AnalyticsTypeTotalViews, data.FromInput, data.ToInput, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Clicks, err = getCounts(ids, models.AnalyticsTypeTotalClicks, data.FromInput, data.ToInput, data.Interval)
			if err != nil {
				return err
			}

			if data.ShowUnique {
				data.Data.UniqueViews, err = getCounts(ids, models.AnalyticsTypeViews, data.FromInput, data.ToInput, data.Interval)
				if err != nil {
					return err
				}

				data.Data.UniqueClicks, err = getCounts(ids, models.AnalyticsTypeClicks, data.FromInput, data.ToInput, data.Interval)
				if err != nil {
					return err
				}

				data.Counts.UniqueViews = sumAnalyticsCounts(data.Data.UniqueViews)
				data.Counts.UniqueClicks = sumAnalyticsCounts(data.Data.UniqueClicks)
			}

			data.Data.Bounces, err = getCounts(ids, models.AnalyticsTypeBounces, data.FromInput, data.ToInput, data.Interval)
			if err != nil {
				return err
			}

			data.Data.Links, err = getLinks(ids, data.FromInput, data.ToInput)
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

// GetCampaignViewAnalytics retrieves analytics for campaigns.
func (a *App) GetCampaignViewAnalytics(c *echo.Context) error {
	return a.getAnalytics(c, models.ResourceCampaigns)
}

// GetListAnalytics retrieves analytics for lists.
func (a *App) GetListAnalytics(c *echo.Context) error {
	return a.getAnalytics(c, models.ResourceLists)
}

func (a *App) getAnalytics(c *echo.Context, resource string) error {
	ids, err := parseStringIDs(c.Request().URL.Query()["id"])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}

	if len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.missingFields", "name", "`id`"))
	}

	getCounts := a.core.GetCampaignAnalyticsCounts
	getLinks := a.core.GetCampaignAnalyticsLinks
	if resource == models.ResourceLists {
		user := auth.GetUser(c)
		if err := user.HasListPerm(auth.PermTypeGet, ids...); err != nil {
			return err
		}
		// Match the list page: campaign metrics require campaign analytics access.
		if typ := c.Param("type"); typ != models.AnalyticsTypeSubscriptions && typ != models.AnalyticsTypeUnsubscriptions {
			if !(user.HasPerm(auth.PermCampaignsGet) || user.HasPerm(auth.PermCampaignsGetAll)) || !user.HasPerm(auth.PermCampaignsGetAnalytics) {
				return auth.ErrPermDenied
			}
		}
		getCounts = a.core.GetListAnalyticsCounts
		getLinks = a.core.GetListAnalyticsLinks
	} else {
		// Ensure the user has access to campaigns via lists.
		for _, id := range ids {
			if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
				return err
			}
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

	// Link stats.
	if typ == models.AnalyticsTypeLinks {
		out, err := getLinks(ids, from, to)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	interval := c.QueryParam("interval")
	if interval == "" {
		interval = models.AnalyticsIntervalDay
	}

	// Views and clicks default to totals, with unique counts available when tracking is on.
	if c.QueryParam("unique") != "true" || !a.cfg.Privacy.IndividualTracking {
		switch typ {
		case models.AnalyticsTypeViews:
			typ = models.AnalyticsTypeTotalViews
		case models.AnalyticsTypeClicks:
			typ = models.AnalyticsTypeTotalClicks
		}
	}

	// Get the analytics numbers from the DB.
	out, err := getCounts(ids, typ, from, to, interval)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// sumAnalyticsCounts returns the combined list totals when available or sums up the rows.
func sumAnalyticsCounts(rows []models.CampaignAnalyticsCount) int {
	if len(rows) > 0 && rows[0].Total != nil {
		return *rows[0].Total
	}

	total := 0
	for _, r := range rows {
		total += r.Count
	}

	return total
}

// loadCampaignActivity fetches the selected tab results.
func (a *App) loadCampaignActivity(c *echo.Context, data *analyticsView, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	q := data.Page.QueryParams
	q.Set("per_page", "20")

	var (
		pg    = a.pg.NewFromURL(q)
		user  = auth.GetUser(c)
		total = 0
	)

	// Individual subscriber activity.
	if subID := q.Get("subscriber_id"); subID != "" {
		id, err := strconv.Atoi(subID)
		if err != nil || id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

		// Check if the user has permission to view this subscriber's activity.
		if !user.HasPerm(auth.PermSubscribersGetAll) {
			if err := a.hasSubPerm(user, auth.PermTypeGet, []int{id}); err != nil {
				return err
			}
		}

		sub, err := a.core.GetSubscriber(id, "", "")
		if err != nil {
			return err
		}

		data.Subscriber = &sub
		data.ActivityEvents, err = a.core.GetCampaignActivityEvents(ids, data.FromInput, data.ToInput, id, pg.Offset, pg.Limit)
		if err != nil {
			return err
		}

		if len(data.ActivityEvents) > 0 {
			total = data.ActivityEvents[0].Total
		}
	} else {
		all, lists := user.GetPermittedLists(auth.PermTypeGet)
		rows, err := a.core.GetCampaignActivity(ids, data.FromInput, data.ToInput, all || user.HasPerm(auth.PermSubscribersGetAll), lists, pg.Offset, pg.Limit)
		if err != nil {
			return err
		}

		data.Activity = rows
		if len(rows) > 0 {
			total = rows[0].Total
		}
	}

	data.Page = models.NewPageProps(q, total, pg.Page, pg.PerPage)
	return nil
}
