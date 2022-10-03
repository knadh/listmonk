package core

import (
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo/v4"
)

// GetDashboardCharts returns chart data points to render on the dashboard.
func (c *Core) GetDashboardCharts() (types.JSONText, error) {
	var out types.JSONText
	if err := c.q.GetDashboardCharts.Get(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard charts", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardCounts returns stats counts to show on the dashboard.
func (c *Core) GetDashboardCounts() (types.JSONText, error) {
	var out types.JSONText
	if err := c.q.GetDashboardCounts.Get(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardSubscribersCount returns subscriber count chart data points to render on the dashboard.
func (c *Core) GetDashboardSubscribersCount(list_id string, months string) (types.JSONText, error) {
	nrMonths, err := strconv.Atoi(months)
	interval := "2 months"
	if err == nil {
		interval = strconv.Itoa(nrMonths) + " months"
	}

	var out types.JSONText
	if err := c.q.GetDashboardSubscribersCount.Get(&out, list_id, interval); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard subscriber count", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardDomainsCount returns subscriber e-mail domains chart data points to render on the dashboard.
func (c *Core) GetDashboardDomainsCount(list_id string) (types.JSONText, error) {
	var out types.JSONText
	if list_id != "" {
		if err := c.q.GetDashboardDomainsByList.Get(&out, list_id); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard domain stats", "error", pqErrMsg(err)))
		}
	} else {
		if err := c.q.GetDashboardDomains.Get(&out); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard domain stats", "error", pqErrMsg(err)))
		}
	}

	return out, nil
}

// handleGetDashboardCountries returns subscriber country stats counts to show on the dashboard.
func (c *Core) GetDashboardCountries(list_id string) (types.JSONText, error) {
	var out types.JSONText
	if list_id != "" {
		if err := c.q.GetDashboardCountryStatsByList.Get(&out, list_id); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard country stats", "error", pqErrMsg(err)))
		}
	} else {
		if err := c.q.GetDashboardCountryStats.Get(&out); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard country stats", "error", pqErrMsg(err)))
		}
	}

	return out, nil
}
