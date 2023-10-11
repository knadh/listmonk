package main

import (
	"crypto/subtle"
	"net/http"
	"path"
	"regexp"

	"github.com/knadh/paginator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 2000

	sortAsc  = "asc"
	sortDesc = "desc"
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

var (
	reUUID     = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
	reLangCode = regexp.MustCompile("[^a-zA-Z_0-9\\-]")

	paginate = paginator.New(paginator.Opt{
		DefaultPerPage: 20,
		MaxPerPage:     50,
		NumPageNums:    10,
		PageParam:      "page",
		PerPageParam:   "per_page",
	})
)

// registerHandlers registers HTTP handlers.
func initHTTPHandlers(e *echo.Echo, app *App) {
	// Group of private handlers with BasicAuth.
	var g *echo.Group

	if len(app.constants.AdminUsername) == 0 ||
		len(app.constants.AdminPassword) == 0 {
		g = e.Group("")
	} else {
		g = e.Group("", middleware.BasicAuth(basicAuth))
	}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			app.log.Println(err.Error())
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	// Admin JS app views.
	// /admin/static/* file server is registered in initHTTPServer().
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
	})
	g.GET(path.Join(adminRoot, ""), handleAdminPage)
	g.GET(path.Join(adminRoot, "/custom.css"), serveCustomApperance("admin.custom_css"))
	g.GET(path.Join(adminRoot, "/custom.js"), serveCustomApperance("admin.custom_js"))
	g.GET(path.Join(adminRoot, "/*"), handleAdminPage)

	// API endpoints.
	g.GET("/api/health", handleHealthCheck)
	g.GET("/api/config", handleGetServerConfig)
	g.GET("/api/lang/:lang", handleGetI18nLang)
	g.GET("/api/dashboard/charts", handleGetDashboardCharts)
	g.GET("/api/dashboard/counts", handleGetDashboardCounts)

	g.GET("/api/settings", handleGetSettings)
	g.PUT("/api/settings", handleUpdateSettings)
	g.POST("/api/settings/smtp/test", handleTestSMTPSettings)
	g.POST("/api/admin/reload", handleReloadApp)
	g.GET("/api/logs", handleGetLogs)
	g.GET("/api/about", handleGetAboutInfo)

	g.GET("/api/subscribers/:id", handleGetSubscriber)
	g.GET("/api/subscribers/:id/export", handleExportSubscriberData)
	g.GET("/api/subscribers/:id/bounces", handleGetSubscriberBounces)
	g.DELETE("/api/subscribers/:id/bounces", handleDeleteSubscriberBounces)
	g.POST("/api/subscribers", handleCreateSubscriber)
	g.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	g.POST("/api/subscribers/:id/optin", handleSubscriberSendOptin)
	g.PUT("/api/subscribers/blocklist", handleBlocklistSubscribers)
	g.PUT("/api/subscribers/:id/blocklist", handleBlocklistSubscribers)
	g.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists)
	g.PUT("/api/subscribers/lists", handleManageSubscriberLists)
	g.DELETE("/api/subscribers/:id", handleDeleteSubscribers)
	g.DELETE("/api/subscribers", handleDeleteSubscribers)

	g.GET("/api/bounces", handleGetBounces)
	g.GET("/api/bounces/:id", handleGetBounces)
	g.DELETE("/api/bounces", handleDeleteBounces)
	g.DELETE("/api/bounces/:id", handleDeleteBounces)

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	g.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery)
	g.PUT("/api/subscribers/query/blocklist", handleBlocklistSubscribersByQuery)
	g.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery)
	g.GET("/api/subscribers", handleQuerySubscribers)
	g.GET("/api/subscribers/export",
		middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(handleExportSubscribers))

	g.GET("/api/import/subscribers", handleGetImportSubscribers)
	g.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats)
	g.POST("/api/import/subscribers", handleImportSubscribers)
	g.DELETE("/api/import/subscribers", handleStopImportSubscribers)

	g.GET("/api/lists", handleGetLists)
	g.GET("/api/lists/:id", handleGetLists)
	g.POST("/api/lists", handleCreateList)
	g.PUT("/api/lists/:id", handleUpdateList)
	g.DELETE("/api/lists/:id", handleDeleteLists)

	g.GET("/api/campaigns", handleGetCampaigns)
	g.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats)
	g.GET("/api/campaigns/:id", handleGetCampaign)
	g.GET("/api/campaigns/analytics/:type", handleGetCampaignViewAnalytics)
	g.GET("/api/campaigns/:id/preview", handlePreviewCampaign)
	g.POST("/api/campaigns/:id/preview", handlePreviewCampaign)
	g.POST("/api/campaigns/:id/content", handleCampaignContent)
	g.POST("/api/campaigns/:id/text", handlePreviewCampaign)
	g.POST("/api/campaigns/:id/test", handleTestCampaign)
	g.POST("/api/campaigns", handleCreateCampaign)
	g.PUT("/api/campaigns/:id", handleUpdateCampaign)
	g.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus)
	g.PUT("/api/campaigns/:id/archive", handleUpdateCampaignArchive)
	g.DELETE("/api/campaigns/:id", handleDeleteCampaign)

	g.GET("/api/media", handleGetMedia)
	g.GET("/api/media/:id", handleGetMedia)
	g.POST("/api/media", handleUploadMedia)
	g.DELETE("/api/media/:id", handleDeleteMedia)

	g.GET("/api/templates", handleGetTemplates)
	g.GET("/api/templates/:id", handleGetTemplates)
	g.GET("/api/templates/:id/preview", handlePreviewTemplate)
	g.POST("/api/templates/preview", handlePreviewTemplate)
	g.POST("/api/templates", handleCreateTemplate)
	g.PUT("/api/templates/:id", handleUpdateTemplate)
	g.PUT("/api/templates/:id/default", handleTemplateSetDefault)
	g.DELETE("/api/templates/:id", handleDeleteTemplate)

	g.DELETE("/api/maintenance/subscribers/:type", handleGCSubscribers)
	g.DELETE("/api/maintenance/analytics/:type", handleGCCampaignAnalytics)
	g.DELETE("/api/maintenance/subscriptions/unconfirmed", handleGCSubscriptions)

	g.POST("/api/tx", handleSendTxMessage)

	g.GET("/api/events", handleEventStream)

	if app.constants.BounceWebhooksEnabled {
		// Private authenticated bounce endpoint.
		g.POST("/webhooks/bounce", handleBounceWebhook)

		// Public bounce endpoints for webservices like SES.
		e.POST("/webhooks/service/:service", handleBounceWebhook)
	}

	// Public API endpoints.
	e.GET("/api/public/lists", handleGetPublicLists)
	e.POST("/api/public/subscription", handlePublicSubscription)

	if app.constants.EnablePublicArchive {
		e.GET("/api/public/archive", handleGetCampaignArchives)
	}

	// /public/static/* file server is registered in initHTTPServer().
	// Public subscriber facing views.
	e.GET("/subscription/form", handleSubscriptionFormPage)
	e.POST("/subscription/form", handleSubscriptionForm)
	e.GET("/subscription/:campUUID/:subUUID", noIndex(validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID")))
	e.POST("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(handleSubscriptionPrefs),
		"campUUID", "subUUID"))
	e.GET("/subscription/optin/:subUUID", noIndex(validateUUID(subscriberExists(handleOptinPage), "subUUID")))
	e.POST("/subscription/optin/:subUUID", validateUUID(subscriberExists(handleOptinPage), "subUUID"))
	e.POST("/subscription/export/:subUUID", validateUUID(subscriberExists(handleSelfExportSubscriberData),
		"subUUID"))
	e.POST("/subscription/wipe/:subUUID", validateUUID(subscriberExists(handleWipeSubscriberData),
		"subUUID"))
	e.GET("/link/:linkUUID/:campUUID/:subUUID", noIndex(validateUUID(handleLinkRedirect,
		"linkUUID", "campUUID", "subUUID")))
	e.GET("/campaign/:campUUID/:subUUID", noIndex(validateUUID(handleViewCampaignMessage,
		"campUUID", "subUUID")))
	e.GET("/campaign/:campUUID/:subUUID/px.png", noIndex(validateUUID(handleRegisterCampaignView,
		"campUUID", "subUUID")))

	if app.constants.EnablePublicArchive {
		e.GET("/archive", handleCampaignArchivesPage)
		e.GET("/archive.xml", handleGetCampaignArchivesFeed)
		e.GET("/archive/:uuid", handleCampaignArchivePage)
		e.GET("/archive/latest", handleCampaignArchivePageLatest)
	}

	e.GET("/public/custom.css", serveCustomApperance("public.custom_css"))
	e.GET("/public/custom.js", serveCustomApperance("public.custom_js"))

	// Public health API endpoint.
	e.GET("/health", handleHealthCheck)
}

// handleAdminPage is the root handler that renders the Javascript admin frontend.
func handleAdminPage(c echo.Context) error {
	app := c.Get("app").(*App)

	b, err := app.fs.Read(path.Join(adminRoot, "/index.html"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTMLBlob(http.StatusOK, b)
}

// handleHealthCheck is a healthcheck endpoint that returns a 200 response.
func handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{true})
}

// serveCustomApperance serves the given custom CSS/JS appearance blob
// meant for customizing public and admin pages from the admin settings UI.
func serveCustomApperance(name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			app = c.Get("app").(*App)

			out []byte
			hdr string
		)

		switch name {
		case "admin.custom_css":
			out = app.constants.Appearance.AdminCSS
			hdr = "text/css; charset=utf-8"

		case "admin.custom_js":
			out = app.constants.Appearance.AdminJS
			hdr = "application/javascript; charset=utf-8"

		case "public.custom_css":
			out = app.constants.Appearance.PublicCSS
			hdr = "text/css; charset=utf-8"

		case "public.custom_js":
			out = app.constants.Appearance.PublicJS
			hdr = "application/javascript; charset=utf-8"
		}

		return c.Blob(http.StatusOK, hdr, out)
	}
}

// basicAuth middleware does an HTTP BasicAuth authentication for admin handlers.
func basicAuth(username, password string, c echo.Context) (bool, error) {
	app := c.Get("app").(*App)

	// Auth is disabled.
	if len(app.constants.AdminUsername) == 0 &&
		len(app.constants.AdminPassword) == 0 {
		return true, nil
	}

	if subtle.ConstantTimeCompare([]byte(username), app.constants.AdminUsername) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), app.constants.AdminPassword) == 1 {
		return true, nil
	}
	return false, nil
}

// validateUUID middleware validates the UUID string format for a given set of params.
func validateUUID(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		app := c.Get("app").(*App)

		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage,
					makeMsgTpl(app.i18n.T("public.errorTitle"), "",
						app.i18n.T("globals.messages.invalidUUID")))
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

		if _, err := app.core.GetSubscriber(0, subUUID, ""); err != nil {
			if er, ok := err.(*echo.HTTPError); ok && er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(app.i18n.T("public.notFoundTitle"), "", er.Message.(string)))
			}

			app.log.Printf("error checking subscriber existence: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.errorProcessingRequest")))
		}

		return next(c)
	}
}

// noIndex adds the HTTP header requesting robots to not crawl the page.
func noIndex(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("X-Robots-Tag", "noindex")
		return next(c)
	}
}
