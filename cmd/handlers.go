package main

import (
	"bytes"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 2000

	// URIs.
	uriAdmin = "/admin"
)

type okResp struct {
	Data interface{} `json:"data"`
}

var (
	reUUID = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
)

// registerHandlers registers HTTP handlers.
func initHTTPHandlers(e *echo.Echo, app *App) {
	// Default error handler.
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			app.log.Println(err.Error())
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	var (
		// Authenticated /api/* handlers.
		api = e.Group("", app.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				u := c.Get(auth.UserKey)

				// On no-auth, respond with a JSON error.
				if err, ok := u.(*echo.HTTPError); ok {
					return err
				}

				return next(c)
			}
		})

		// Authenticated non /api handlers.
		a = e.Group("", app.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				u := c.Get(auth.UserKey)
				// On no-auth, redirect to login page
				if _, ok := u.(*echo.HTTPError); ok {
					u, _ := url.Parse(app.constants.LoginURL)
					q := url.Values{}
					q.Set("next", c.Request().RequestURI)
					u.RawQuery = q.Encode()
					return c.Redirect(http.StatusTemporaryRedirect, u.String())
				}

				return next(c)
			}
		})

		// Public unauthenticated endpoints.
		p = e.Group("")
	)

	// Authenticated endpoints.
	a.GET(path.Join(uriAdmin, ""), handleAdminPage)
	a.GET(path.Join(uriAdmin, "/custom.css"), serveCustomAppearance("admin.custom_css"))
	a.GET(path.Join(uriAdmin, "/custom.js"), serveCustomAppearance("admin.custom_js"))
	a.GET(path.Join(uriAdmin, "/*"), handleAdminPage)

	pm := app.auth.Perm

	// API endpoints.
	api.GET("/api/health", handleHealthCheck)
	api.GET("/api/config", handleGetServerConfig)
	api.GET("/api/lang/:lang", handleGetI18nLang)
	api.GET("/api/dashboard/charts", handleGetDashboardCharts)
	api.GET("/api/dashboard/counts", handleGetDashboardCounts)

	api.GET("/api/settings", pm(handleGetSettings, "settings:get"))
	api.PUT("/api/settings", pm(handleUpdateSettings, "settings:manage"))
	api.POST("/api/settings/smtp/test", pm(handleTestSMTPSettings, "settings:manage"))
	api.POST("/api/admin/reload", pm(handleReloadApp, "settings:manage"))
	api.GET("/api/logs", pm(handleGetLogs, "settings:get"))
	api.GET("/api/events", pm(handleEventStream, "settings:get"))
	api.GET("/api/about", handleGetAboutInfo)

	api.GET("/api/subscribers", pm(handleQuerySubscribers, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id", pm(handleGetSubscriber, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/export", pm(handleExportSubscriberData, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/bounces", pm(handleGetSubscriberBounces, "bounces:get"))
	api.DELETE("/api/subscribers/:id/bounces", pm(handleDeleteSubscriberBounces, "bounces:manage"))
	api.POST("/api/subscribers", pm(handleCreateSubscriber, "subscribers:manage"))
	api.PUT("/api/subscribers/:id", pm(handleUpdateSubscriber, "subscribers:manage"))
	api.POST("/api/subscribers/:id/optin", pm(handleSubscriberSendOptin, "subscribers:manage"))
	api.PUT("/api/subscribers/blocklist", pm(handleBlocklistSubscribers, "subscribers:manage"))
	api.PUT("/api/subscribers/:id/blocklist", pm(handleBlocklistSubscribers, "subscribers:manage"))
	api.PUT("/api/subscribers/lists/:id", pm(handleManageSubscriberLists, "subscribers:manage"))
	api.PUT("/api/subscribers/lists", pm(handleManageSubscriberLists, "subscribers:manage"))
	api.DELETE("/api/subscribers/:id", pm(handleDeleteSubscribers, "subscribers:manage"))
	api.DELETE("/api/subscribers", pm(handleDeleteSubscribers, "subscribers:manage"))

	api.GET("/api/bounces", pm(handleGetBounces, "bounces:get"))
	api.GET("/api/bounces/:id", pm(handleGetBounces, "bounces:get"))
	api.DELETE("/api/bounces", pm(handleDeleteBounces, "bounces:manage"))
	api.DELETE("/api/bounces/:id", pm(handleDeleteBounces, "bounces:manage"))

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	api.POST("/api/subscribers/query/delete", pm(handleDeleteSubscribersByQuery, "subscribers:manage"))
	api.PUT("/api/subscribers/query/blocklist", pm(handleBlocklistSubscribersByQuery, "subscribers:manage"))
	api.PUT("/api/subscribers/query/lists", pm(handleManageSubscriberListsByQuery, "subscribers:manage"))
	api.GET("/api/subscribers/export",
		pm(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(handleExportSubscribers), "subscribers:get_all", "subscribers:get"))

	api.GET("/api/import/subscribers", pm(handleGetImportSubscribers, "subscribers:import"))
	api.GET("/api/import/subscribers/logs", pm(handleGetImportSubscriberStats, "subscribers:import"))
	api.POST("/api/import/subscribers", pm(handleImportSubscribers, "subscribers:import"))
	api.DELETE("/api/import/subscribers", pm(handleStopImportSubscribers, "subscribers:import"))

	// Individual list permissions are applied directly within handleGetLists.
	api.GET("/api/lists", handleGetLists)
	api.GET("/api/lists/:id", listPerm(handleGetList))
	api.POST("/api/lists", pm(handleCreateList, "lists:manage_all"))
	api.PUT("/api/lists/:id", listPerm(handleUpdateList))
	api.DELETE("/api/lists/:id", listPerm(handleDeleteLists))

	api.GET("/api/campaigns", pm(handleGetCampaigns, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/running/stats", pm(handleGetRunningCampaignStats, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/:id", pm(handleGetCampaign, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/analytics/:type", pm(handleGetCampaignViewAnalytics, "campaigns:get_analytics"))
	api.GET("/api/campaigns/:id/preview", pm(handlePreviewCampaign, "campaigns:get_all", "campaigns:get"))
	api.POST("/api/campaigns/:id/preview", pm(handlePreviewCampaign, "campaigns:get_all", "campaigns:get"))
	api.POST("/api/campaigns/:id/content", pm(handleCampaignContent, "campaigns:manage_all", "campaigns:manage"))
	api.POST("/api/campaigns/:id/text", pm(handlePreviewCampaign, "campaigns:get"))
	api.POST("/api/campaigns/:id/test", pm(handleTestCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.POST("/api/campaigns", pm(handleCreateCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id", pm(handleUpdateCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id/status", pm(handleUpdateCampaignStatus, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id/archive", pm(handleUpdateCampaignArchive, "campaigns:manage_all", "campaigns:manage"))
	api.DELETE("/api/campaigns/:id", pm(handleDeleteCampaign, "campaigns:manage_all", "campaigns:manage"))

	api.GET("/api/media", pm(handleGetMedia, "media:get"))
	api.GET("/api/media/:id", pm(handleGetMedia, "media:get"))
	api.POST("/api/media", pm(handleUploadMedia, "media:manage"))
	api.DELETE("/api/media/:id", pm(handleDeleteMedia, "media:manage"))

	api.GET("/api/templates", pm(handleGetTemplates, "templates:get"))
	api.GET("/api/templates/:id", pm(handleGetTemplates, "templates:get"))
	api.GET("/api/templates/:id/preview", pm(handlePreviewTemplate, "templates:get"))
	api.POST("/api/templates/preview", pm(handlePreviewTemplate, "templates:get"))
	api.POST("/api/templates", pm(handleCreateTemplate, "templates:manage"))
	api.PUT("/api/templates/:id", pm(handleUpdateTemplate, "templates:manage"))
	api.PUT("/api/templates/:id/default", pm(handleTemplateSetDefault, "templates:manage"))
	api.DELETE("/api/templates/:id", pm(handleDeleteTemplate, "templates:manage"))

	api.DELETE("/api/maintenance/subscribers/:type", pm(handleGCSubscribers, "settings:maintain"))
	api.DELETE("/api/maintenance/analytics/:type", pm(handleGCCampaignAnalytics, "settings:maintain"))
	api.DELETE("/api/maintenance/subscriptions/unconfirmed", pm(handleGCSubscriptions, "settings:maintain"))

	api.POST("/api/tx", pm(handleSendTxMessage, "tx:send"))

	api.GET("/api/profile", handleGetUserProfile)
	api.PUT("/api/profile", handleUpdateUserProfile)
	api.GET("/api/users", pm(handleGetUsers, "users:get"))
	api.GET("/api/users/:id", pm(handleGetUsers, "users:get"))
	api.POST("/api/users", pm(handleCreateUser, "users:manage"))
	api.PUT("/api/users/:id", pm(handleUpdateUser, "users:manage"))
	api.DELETE("/api/users", pm(handleDeleteUsers, "users:manage"))
	api.DELETE("/api/users/:id", pm(handleDeleteUsers, "users:manage"))
	api.POST("/api/logout", handleLogout)

	api.GET("/api/roles/users", pm(handleGetUserRoles, "roles:get"))
	api.GET("/api/roles/lists", pm(handleGeListRoles, "roles:get"))
	api.POST("/api/roles/users", pm(handleCreateUserRole, "roles:manage"))
	api.POST("/api/roles/lists", pm(handleCreateListRole, "roles:manage"))
	api.PUT("/api/roles/users/:id", pm(handleUpdateUserRole, "roles:manage"))
	api.PUT("/api/roles/lists/:id", pm(handleUpdateListRole, "roles:manage"))
	api.DELETE("/api/roles/:id", pm(handleDeleteRole, "roles:manage"))

	if app.constants.BounceWebhooksEnabled {
		// Private authenticated bounce endpoint.
		api.POST("/webhooks/bounce", pm(handleBounceWebhook, "webhooks:post_bounce"))

		// Public bounce endpoints for webservices like SES.
		p.POST("/webhooks/service/:service", handleBounceWebhook)
	}

	// =================================================================
	// Public API endpoints.

	// Landing page.
	p.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
	})

	// Public admin endpoints (login page, OIDC endpoints).
	p.GET(path.Join(uriAdmin, "/login"), handleLoginPage)
	p.POST(path.Join(uriAdmin, "/login"), handleLoginPage)

	if app.constants.Security.OIDC.Enabled {
		p.POST("/auth/oidc", handleOIDCLogin)
		p.GET("/auth/oidc", handleOIDCFinish)
	}

	// Public APIs.
	p.GET("/api/public/lists", handleGetPublicLists)
	p.POST("/api/public/subscription", handlePublicSubscription)
	if app.constants.EnablePublicArchive {
		p.GET("/api/public/archive", handleGetCampaignArchives)
	}

	// /public/static/* file server is registered in initHTTPServer().
	// Public subscriber facing views.
	p.GET("/subscription/form", handleSubscriptionFormPage)
	p.POST("/subscription/form", handleSubscriptionForm)
	p.GET("/subscription/:campUUID/:subUUID", noIndex(validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID")))
	p.POST("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(handleSubscriptionPrefs),
		"campUUID", "subUUID"))
	p.GET("/subscription/optin/:subUUID", noIndex(validateUUID(subscriberExists(handleOptinPage), "subUUID")))
	p.POST("/subscription/optin/:subUUID", validateUUID(subscriberExists(handleOptinPage), "subUUID"))
	p.POST("/subscription/export/:subUUID", validateUUID(subscriberExists(handleSelfExportSubscriberData),
		"subUUID"))
	p.POST("/subscription/wipe/:subUUID", validateUUID(subscriberExists(handleWipeSubscriberData),
		"subUUID"))
	p.GET("/link/:linkUUID/:campUUID/:subUUID", noIndex(validateUUID(handleLinkRedirect,
		"linkUUID", "campUUID", "subUUID")))
	p.GET("/campaign/:campUUID/:subUUID", noIndex(validateUUID(handleViewCampaignMessage,
		"campUUID", "subUUID")))
	p.GET("/campaign/:campUUID/:subUUID/px.png", noIndex(validateUUID(handleRegisterCampaignView,
		"campUUID", "subUUID")))

	if app.constants.EnablePublicArchive {
		p.GET("/archive", handleCampaignArchivesPage)
		p.GET("/archive.xml", handleGetCampaignArchivesFeed)
		p.GET("/archive/:id", handleCampaignArchivePage)
		p.GET("/archive/latest", handleCampaignArchivePageLatest)
	}

	p.GET("/public/custom.css", serveCustomAppearance("public.custom_css"))
	p.GET("/public/custom.js", serveCustomAppearance("public.custom_js"))

	// Public health API endpoint.
	p.GET("/health", handleHealthCheck)

	// 404 pages.
	p.RouteNotFound("/*", func(c echo.Context) error {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl("404 - "+app.i18n.T("public.notFoundTitle"), "", ""))
	})
	p.RouteNotFound("/api/*", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "404 unknown endpoint")
	})
	p.RouteNotFound("/admin/*", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "404 page not found")
	})
}

// handleAdminPage is the root handler that renders the Javascript admin frontend.
func handleAdminPage(c echo.Context) error {
	app := c.Get("app").(*App)

	b, err := app.fs.Read(path.Join(uriAdmin, "/index.html"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	b = bytes.ReplaceAll(b, []byte("asset_version"), []byte(app.constants.AssetVersion))

	return c.HTMLBlob(http.StatusOK, b)
}

// handleHealthCheck is a healthcheck endpoint that returns a 200 response.
func handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{true})
}

// serveCustomAppearance serves the given custom CSS/JS appearance blob
// meant for customizing public and admin pages from the admin settings UI.
func serveCustomAppearance(name string) echo.HandlerFunc {
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

// validateUUID middleware validates the UUID string format for a given set of params.
func validateUUID(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		app := c.Get("app").(*App)

		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(app.i18n.T("public.errorTitle"), "",
					app.i18n.T("globals.messages.invalidUUID")))
			}
		}
		return next(c)
	}
}

// subscriberExists middleware checks if a subscriber exists given the UUID
// param in a request.
func subscriberExists(next echo.HandlerFunc) echo.HandlerFunc {
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
func noIndex(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("X-Robots-Tag", "noindex")
		return next(c)
	}
}
