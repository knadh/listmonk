package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/labstack/echo"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 200

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

var reUUID = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// registerHandlers registers HTTP handlers.
func registerHTTPHandlers(e *echo.Echo) {
	e.GET("/", handleIndexPage)
	e.GET("/api/config.js", handleGetConfigScript)
	e.GET("/api/dashboard/stats", handleGetDashboardStats)

	e.GET("/api/subscribers/:id", handleGetSubscriber)
	e.GET("/api/subscribers/:id/export", handleExportSubscriberData)
	e.POST("/api/subscribers", handleCreateSubscriber)
	e.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	e.POST("/api/subscribers/:id/optin", handleGetSubscriberSendOptin)
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
	e.POST("/subscription/form", handleSubscriptionForm)
	e.GET("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID"))
	e.POST("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID"))
	e.GET("/subscription/optin/:subUUID", validateUUID(subscriberExists(handleOptinPage), "subUUID"))
	e.POST("/subscription/optin/:subUUID", validateUUID(subscriberExists(handleOptinPage), "subUUID"))
	e.POST("/subscription/export/:subUUID", validateUUID(subscriberExists(handleSelfExportSubscriberData),
		"subUUID"))
	e.POST("/subscription/wipe/:subUUID", validateUUID(subscriberExists(handleWipeSubscriberData),
		"subUUID"))
	e.GET("/link/:linkUUID/:campUUID/:subUUID", validateUUID(handleLinkRedirect,
		"linkUUID", "campUUID", "subUUID"))
	e.GET("/campaign/:campUUID/:subUUID", validateUUID(handleViewCampaignMessage,
		"campUUID", "subUUID"))
	e.GET("/campaign/:campUUID/:subUUID/px.png", validateUUID(handleRegisterCampaignView,
		"campUUID", "subUUID"))

	// Static views.
	e.GET("/lists", handleIndexPage)
	e.GET("/lists/forms", handleIndexPage)
	e.GET("/subscribers", handleIndexPage)
	e.GET("/subscribers/lists/:listID", handleIndexPage)
	e.GET("/subscribers/import", handleIndexPage)
	e.GET("/campaigns", handleIndexPage)
	e.GET("/campaigns/new", handleIndexPage)
	e.GET("/campaigns/media", handleIndexPage)
	e.GET("/campaigns/templates", handleIndexPage)
	e.GET("/campaigns/:campignID", handleIndexPage)
}

// handleIndex is the root handler that renders the Javascript frontend.
func handleIndexPage(c echo.Context) error {
	app := c.Get("app").(*App)

	b, err := app.fs.Read("/frontend/index.html")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("Content-Type", "text/html")
	return c.String(http.StatusOK, string(b))
}

// validateUUID middleware validates the UUID string format for a given set of params.
func validateUUID(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage,
					makeMsgTpl("Invalid request", "",
						`One or more UUIDs in the request are invalid.`))
			}
		}
		return next(c)
	}
}

// subscriberExists middleware checks if a subscriber exists given the UUID
// param in a request.
func subscriberExists(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			app     = c.Get("app").(*App)
			subUUID = c.Param("subUUID")
		)

		var exists bool
		if err := app.queries.SubscriberExists.Get(&exists, 0, subUUID); err != nil {
			app.log.Printf("error checking subscriber existence: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl("Error", "",
					`Error processing request. Please retry.`))
		}

		if !exists {
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl("Not found", "",
					`Subscription not found.`))
		}
		return next(c)
	}
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
		if ppi > 0 && ppi <= maxPerPage {
			perPage = ppi
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
