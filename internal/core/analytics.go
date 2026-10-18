package core

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
	"github.com/lib/pq"
)

// GetAnalyticsItems fetches campaign analytics datapoints for items (by: lists/campaigns) in the given order.
func (c *Core) GetAnalyticsItems(resource string, ids []int, getAll bool, permittedLists []int) ([]models.AnalyticsItem, error) {
	out := []models.AnalyticsItem{}
	if len(ids) == 0 {
		return out, nil
	}

	stmt := c.q.GetListAnalyticsItems
	switch resource {
	case models.ResourceLists:
	case models.ResourceCampaigns:
		stmt = c.q.GetCampaignAnalyticsItems
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("globals.messages.invalidData"))
	}

	if err := stmt.Select(&out, pq.Array(ids), getAll, pq.Array(permittedLists)); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// getAnalyticsCounts validates the range and interval for campaign and list charts.
func (c *Core) getAnalyticsCounts(stmt *sqlx.Stmt, ids []int, fromDate, toDate, interval string) ([]models.CampaignAnalyticsCount, error) {
	if !strHasLen(fromDate, 10, 30) || !strHasLen(toDate, 10, 30) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("analytics.invalidDates"))
	}

	switch interval {
	case models.AnalyticsIntervalHour, models.AnalyticsIntervalDay, models.AnalyticsIntervalWeek, models.AnalyticsIntervalMonth:
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("globals.messages.invalidData"))
	}

	out := []models.CampaignAnalyticsCount{}
	if err := stmt.Select(&out, pq.Array(ids), fromDate, toDate, interval); err != nil {
		c.log.Printf("error fetching analytics: %v", err)

		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

func (c *Core) getAnalyticsLinks(stmt *sqlx.Stmt, ids []int, fromDate, toDate string) ([]models.CampaignAnalyticsLink, error) {
	out := []models.CampaignAnalyticsLink{}
	if err := stmt.Select(&out, pq.Array(ids), fromDate, toDate); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}
