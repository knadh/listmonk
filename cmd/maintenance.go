package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// GCSubscribers garbage collects (deletes) orphaned or blocklisted subscribers.
func (a *App) GCSubscribers(c echo.Context) error {
	var (
		typ = c.Param("type")

		n   int
		err error
	)

	switch typ {
	case "blocklisted":
		n, err = a.core.DeleteBlocklistedSubscribers()
	case "orphan":
		n, err = a.core.DeleteOrphanSubscribers()
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCSubscriptions garbage collects (deletes) orphaned or blocklisted subscribers.
func (a *App) GCSubscriptions(c echo.Context) error {
	// Validate the date.
	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	// Delete unconfirmed subscriptions from the DB in bulk.
	n, err := a.core.DeleteUnconfirmedSubscriptions(t)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{n}})
}

// GCCampaignAnalytics garbage collects (deletes) campaign analytics.
func (a *App) GCCampaignAnalytics(c echo.Context) error {

	t, err := time.Parse(time.RFC3339, c.FormValue("before_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	switch c.Param("type") {
	case "all":
		if err := a.core.DeleteCampaignViews(t); err != nil {
			return err
		}
		err = a.core.DeleteCampaignLinkClicks(t)
	case "views":
		err = a.core.DeleteCampaignViews(t)
	case "clicks":
		err = a.core.DeleteCampaignLinkClicks(t)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ExportCampaignAnalytics streams campaign analytics (views or link clicks) as a CSV file.
func (a *App) ExportCampaignAnalytics(c echo.Context) error {
	since, err := time.Parse(time.RFC3339, c.QueryParam("since"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	typ := c.Param("type")
	if typ != "views" && typ != "clicks" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	var (
		hdr = c.Response().Header()
		wr  = csv.NewWriter(c.Response())
	)
	hdr.Set(echo.HeaderContentType, "text/csv")
	hdr.Set(echo.HeaderContentDisposition, "attachment; filename=campaign_"+typ+".csv")
	hdr.Set("Cache-Control", "no-cache")

	switch typ {
	case "views":
		wr.Write([]string{"campaign_id", "campaign_uuid", "campaign_name", "subscriber_id", "subscriber_uuid", "email", "subscriber_name", "created_at"})
		next := a.core.ExportCampaignViews(since, a.cfg.DBBatchSize)
		for {
			rows, err := next()
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				break
			}
			for _, r := range rows {
				if err := wr.Write([]string{
					strconv.Itoa(r.CampaignID), r.CampaignUUID, r.CampaignName,
					strconv.Itoa(r.SubscriberID), r.SubscriberUUID, r.Email, r.SubscriberName,
					r.CreatedAt.Format(time.RFC3339),
				}); err != nil {
					a.log.Printf("error streaming CSV: %v", err)
					return nil
				}
			}
			wr.Flush()
		}

	case "clicks":
		wr.Write([]string{"campaign_id", "campaign_uuid", "campaign_name", "subscriber_id", "subscriber_uuid", "email", "subscriber_name", "url", "created_at"})
		next := a.core.ExportCampaignLinkClicks(since, a.cfg.DBBatchSize)
		for {
			rows, err := next()
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				break
			}
			for _, r := range rows {
				if err := wr.Write([]string{
					strconv.Itoa(r.CampaignID), r.CampaignUUID, r.CampaignName,
					strconv.Itoa(r.SubscriberID), r.SubscriberUUID, r.Email, r.SubscriberName, r.URL,
					r.CreatedAt.Format(time.RFC3339),
				}); err != nil {
					a.log.Printf("error streaming CSV: %v", err)
					return nil
				}
			}
			wr.Flush()
		}
	}

	return nil
}

// RunDBVacuum runs a full VACUUM on the PostgreSQL database.
// VACUUM reclaims storage occupied by dead tuples and updates planner statistics.
func RunDBVacuum(db *sqlx.DB, lo *log.Logger) {
	lo.Println("running database VACUUM ANALYZE")
	if _, err := db.Exec("VACUUM ANALYZE"); err != nil {
		lo.Printf("error running VACUUM ANALYZE: %v", err)
		return
	}
	lo.Println("finished database VACUUM ANALYZE")
}
