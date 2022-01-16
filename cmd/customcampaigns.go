package main

import (
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"net/http"
	"strconv"
	"strings"
)

// handleGetCampaigns handles retrieval of campaigns.
func handleGetCampaignsByUserId(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 20)
		out campsWrap

		id, _     = strconv.Atoi(c.Param("id"))
		status    = c.QueryParams()["status"]
		query     = strings.TrimSpace(c.FormValue("query"))
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		noBody, _ = strconv.ParseBool(c.QueryParam("no_body"))
		userId    = c.Param("userid")
	)

	// Fetch one campaign.
	single := false
	if id > 0 {
		single = true
	}

	queryStr, stmt := makeSearchQuery(query, orderBy, order, app.queries.QueryCampaignsByUserId)

	// Unsafe to ignore scanning fields not present in models.Campaigns.
	if err := db.Select(&out.Results, stmt, id, pq.StringArray(status), queryStr, pg.Offset, pg.Limit, userId); err != nil {
		app.log.Printf("error fetching campaigns: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	if single && len(out.Results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("campaigns.notFound", "name", "{globals.terms.campaign}"))
	}
	if len(out.Results) == 0 {
		out.Results = []models.Campaign{}
		return c.JSON(http.StatusOK, okResp{out})
	}

	for i := 0; i < len(out.Results); i++ {
		// Replace null tags.
		if out.Results[i].Tags == nil {
			out.Results[i].Tags = make(pq.StringArray, 0)
		}

		if noBody {
			out.Results[i].Body = ""
		}
	}

	// Lazy load stats.
	if err := out.Results.LoadStats(app.queries.GetCampaignStats); err != nil {
		app.log.Printf("error fetching campaign stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out.Results[0]})
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}
