package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/labstack/echo"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 200

	// bodyMaxLen is the maximum allowed length for e-mail bodies.
	bodyMaxLen = 1000000

	// defaultPerPage is the default number of results returned in an GET call.
	defaultPerPage = 20

	// maxPerPage is the maximum number of allowed for paginated records.
	maxPerPage = 100
)

type okResp struct {
	Data interface{} `json:"data"`
}

// pagination represents a query's pagination (limit, offset) related values.
type pagination struct {
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
	Offset  int `json:"offset"`
	Limit   int `json:"limit"`
}

// registerHandlers registers HTTP handlers.
func registerHandlers(e *echo.Echo) {
	e.GET("/", handleIndexPage)
	e.GET("/api/config.js", handleGetConfigScript)
	e.GET("/api/dashboard/stats", handleGetDashboardStats)

	e.GET("/api/subscribers/:id", handleGetSubscriber)
	e.GET("/api/subscribers/:id/export", handleExportSubscriberData)
	e.POST("/api/subscribers", handleCreateSubscriber)
	e.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	e.PUT("/api/subscribers/blacklist", handleBlacklistSubscribers)
	e.PUT("/api/subscribers/:id/blacklist", handleBlacklistSubscribers)
	e.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists)
	e.PUT("/api/subscribers/lists", handleManageSubscriberLists)
	e.DELETE("/api/subscribers/:id", handleDeleteSubscribers)
	e.DELETE("/api/subscribers", handleDeleteSubscribers)

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	e.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery)
	e.PUT("/api/subscribers/query/blacklist", handleBlacklistSubscribersByQuery)
	e.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery)
	e.GET("/api/subscribers", handleQuerySubscribers)

	e.GET("/api/import/subscribers", handleGetImportSubscribers)
	e.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats)
	e.POST("/api/import/subscribers", handleImportSubscribers)
	e.DELETE("/api/import/subscribers", handleStopImportSubscribers)

	e.GET("/api/lists", handleGetLists)
	e.GET("/api/lists/:id", handleGetLists)
	e.POST("/api/lists", handleCreateList)
	e.PUT("/api/lists/:id", handleUpdateList)
	e.DELETE("/api/lists/:id", handleDeleteLists)

	e.GET("/api/campaigns", handleGetCampaigns)
	e.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats)
	e.GET("/api/campaigns/:id", handleGetCampaigns)
	e.GET("/api/campaigns/:id/preview", handlePreviewCampaign)
	e.POST("/api/campaigns/:id/preview", handlePreviewCampaign)
	e.POST("/api/campaigns/:id/test", handleTestCampaign)
	e.POST("/api/campaigns", handleCreateCampaign)
	e.PUT("/api/campaigns/:id", handleUpdateCampaign)
	e.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus)
	e.DELETE("/api/campaigns/:id", handleDeleteCampaign)

	e.GET("/api/media", handleGetMedia)
	e.POST("/api/media", handleUploadMedia)
	e.DELETE("/api/media/:id", handleDeleteMedia)

	e.GET("/api/templates", handleGetTemplates)
	e.GET("/api/templates/:id", handleGetTemplates)
	e.GET("/api/templates/:id/preview", handlePreviewTemplate)
	e.POST("/api/templates/preview", handlePreviewTemplate)
	e.POST("/api/templates", handleCreateTemplate)
	e.PUT("/api/templates/:id", handleUpdateTemplate)
	e.PUT("/api/templates/:id/default", handleTemplateSetDefault)
	e.DELETE("/api/templates/:id", handleDeleteTemplate)

	// Subscriber facing views.
	e.GET("/subscription/:campUUID/:subUUID", handleSubscriptionPage)
	e.POST("/subscription/:campUUID/:subUUID", handleSubscriptionPage)
	e.POST("/subscription/export/:subUUID", handleSelfExportSubscriberData)
	e.POST("/subscription/wipe/:subUUID", handleWipeSubscriberData)
	e.GET("/link/:linkUUID/:campUUID/:subUUID", handleLinkRedirect)
	e.GET("/campaign/:campUUID/:subUUID/px.png", handleRegisterCampaignView)

	// Static views.
	e.GET("/lists", handleIndexPage)
	e.GET("/subscribers", handleIndexPage)
	e.GET("/subscribers/lists/:listID", handleIndexPage)
	e.GET("/subscribers/import", handleIndexPage)
	e.GET("/campaigns", handleIndexPage)
	e.GET("/campaigns/new", handleIndexPage)
	e.GET("/campaigns/media", handleIndexPage)
	e.GET("/campaigns/templates", handleIndexPage)
	e.GET("/campaigns/:campignID", handleIndexPage)
}

// handleIndex is the root handler that renders the login page if there's no
// authenticated session, or redirects to the dashboard, if there's one.
func handleIndexPage(c echo.Context) error {
	app := c.Get("app").(*App)

	b, err := app.FS.Read("/frontend/index.html")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("Content-Type", "text/html")
	return c.String(http.StatusOK, string(b))
}

// makeAttribsBlob takes a list of keys and values and creates
// a JSON map out of them.
func makeAttribsBlob(keys []string, vals []string) ([]byte, bool) {
	attribs := make(map[string]interface{})
	for i, key := range keys {
		var (
			s   = vals[i]
			val interface{}
		)

		// Try to detect common JSON types.
		if govalidator.IsFloat(s) {
			val, _ = strconv.ParseFloat(s, 64)
		} else if govalidator.IsInt(s) {
			val, _ = strconv.ParseInt(s, 10, 64)
		} else {
			ls := strings.ToLower(s)
			if ls == "true" || ls == "false" {
				val, _ = strconv.ParseBool(ls)
			} else {
				// It's a string.
				val = s
			}
		}

		attribs[key] = val
	}

	if len(attribs) > 0 {
		j, _ := json.Marshal(attribs)
		return j, true
	}

	return nil, false
}

// getPagination takes form values and extracts pagination values from it.
func getPagination(q url.Values) pagination {
	var (
		page, _ = strconv.Atoi(q.Get("page"))
		perPage = defaultPerPage
	)

	pp := q.Get("per_page")
	if pp == "all" {
		// No limit.
		perPage = 0
	} else {
		ppi, _ := strconv.Atoi(pp)
		if ppi < 1 || ppi > maxPerPage {
			perPage = defaultPerPage
		}
	}

	if page < 1 {
		page = 0
	} else {
		page--
	}

	return pagination{
		Page:    page + 1,
		PerPage: perPage,
		Offset:  page * perPage,
		Limit:   perPage,
	}
}
