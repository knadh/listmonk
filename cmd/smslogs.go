package main

import (
	"database/sql"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"net/http"
)

// handleGetSmsLogsByCampaignId retrieves lists of campaign sms
func handleGetSmsLogsByCampaignId(c echo.Context) error {
	var (
		app        = c.Get("app").(*App)
		campaignId = c.Param("campaignId")
		out        []models.CampaignSms
	)

	if err := app.queries.GetCampaignSmsLogs.Select(&out, campaignId); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		app.log.Printf("error fetching campaign sms logs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	} else if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	return c.JSON(http.StatusOK, okResp{out})
}
