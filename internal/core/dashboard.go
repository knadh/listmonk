package core

import (
	"net/http"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo/v4"
)

// GetDashboardCharts returns chart data points to render on the dashboard.
func (c *Core) GetDashboardCharts() (types.JSONText, error) {
	_ = c.refreshCache(matDashboardCharts, false)

	var out types.JSONText
	if err := c.q.GetDashboardCharts.Get(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard charts", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardCounts returns stats counts to show on the dashboard.
func (c *Core) GetDashboardCounts(authid string, from_date string, to_date string) (types.JSONText, error) {

	_ = c.refreshCache(matDashboardCounts, false)

	if from_date == "" || to_date == "" {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

		from_date = startOfDay.Format("2006-01-02")
		to_date = endOfDay.Format("2006-01-02")
	}

	fromDate, err := time.Parse("2006-01-02", from_date)
	if err != nil {
		return nil, err
	}

	toDate, err := time.Parse("2006-01-02", to_date)
	if err != nil {
		return nil, err
	}

	toDate = toDate.Add(24 * time.Hour).Add(-time.Nanosecond)

	var out types.JSONText
	if err := c.q.GetDashboardCounts.Get(&out, authid, fromDate, toDate); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	return out, nil
}
