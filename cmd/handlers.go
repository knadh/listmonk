package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/paginator"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 2000

	sortAsc  = "asc"
	sortDesc = "desc"

	basicAuthd = "basicauthd"

	// URIs.
	uriAdmin = "/admin"
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

// Handler is the main HTTP handler for the app.
type Handler struct {
	app  *App
	echo *echo.Echo
}

// newHandler returns a new Handler instance.
func newHandler(app *App) (*Handler, error) {
	e := echo.New()
	e.HideBanner = true

	tpl, err := stuffbin.ParseTemplatesGlob(initTplFuncs(app.i18n, app.constants), app.fs, "/public/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing public templates: %w", err)
	}
	e.Renderer = &tplRenderer{
		templates:           tpl,
		SiteName:            app.constants.SiteName,
		RootURL:             app.constants.RootURL,
		LogoURL:             app.constants.LogoURL,
		FaviconURL:          app.constants.FaviconURL,
		AssetVersion:        app.constants.AssetVersion,
		EnablePublicSubPage: app.constants.EnablePublicSubPage,
		EnablePublicArchive: app.constants.EnablePublicArchive,
		IndividualTracking:  app.constants.Privacy.IndividualTracking,
	}
}

func (h *Handler) register() {
	// Initialize the static file server.
	fSrv := h.app.fs.FileServer()

	// Public (subscriber) facing static files.
	h.echo.GET("/public/static/*", echo.WrapHandler(fSrv))

	// Admin (frontend) facing static files.
	h.echo.GET("/admin/static/*", echo.WrapHandler(fSrv))

	// Public (subscriber) facing media upload files.
	if ko.String("upload.provider") == "filesystem" && ko.String("upload.filesystem.upload_uri") != "" {
		h.echo.Static(ko.String("upload.filesystem.upload_uri"), ko.String("upload.filesystem.upload_path"))
	}
}

// initHTTPHandlers registers HTTP handlers.
func (h *Handler) initHTTPHandlers() {
	// Default error handler.
	h.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			h.app.log.Println(err.Error())
		}
		h.echo.DefaultHTTPErrorHandler(err, c)
	}

	var (
		// Authenticated /api/* handlers.
		api = h.echo.Group("", h.app.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				u := c.Get(auth.UserKey)

				// On no-auth, respond with a JSON error.
				if err, ok := u.(*echo.HTTPError); ok {
					return err
				}

				return next(c)
			}
		})

		// Authenticated /admin/* handlers.
		admin = h.echo.Group("/admin", h.app.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				u := c.Get(auth.UserKey)
				// On no-auth, redirect to login page
				if _, ok := u.(*echo.HTTPError); ok {
					u, _ := url.Parse(h.app.constants.LoginURL)
					q := url.Values{}
					q.Set("next", c.Request().RequestURI)
					u.RawQuery = q.Encode()
					return c.Redirect(http.StatusTemporaryRedirect, u.String())
				}

				return next(c)
			}
		})

		// Public unauthenticated endpoints.
		p = h.echo.Group("")
	)

	// Authenticated endpoints.
	admin.GET("", handleAdminPage)
	admin.GET("/custom.css", serveCustomAppearance("admin.custom_css"))
	admin.GET("/custom.js", serveCustomAppearance("admin.custom_js"))
	admin.GET("/*", handleAdminPage)

	pm := h.app.auth.Perm

	// API endpoints.
	api.GET("/api/health", handleHealthCheck)
	api.GET("/api/config", handleGetServerConfig)
	api.GET("/api/lang/:lang", handleGetI18nLang)
	api.GET("/api/dashboard/charts", handleGetDashboardCharts)
	api.GET("/api/dashboard/counts", handleGetDashboardCounts)

	api.GET("/api/settings", handleGetSettings, pm("settings:get"))
	api.PUT("/api/settings", handleUpdateSettings, pm("settings:manage"))
	api.POST("/api/settings/smtp/test", handleTestSMTPSettings, pm("settings:manage"))
	api.POST("/api/admin/reload", handleReloadApp, pm("settings:manage"))
	api.GET("/api/logs", handleGetLogs, pm("settings:get"))
	api.GET("/api/events", handleEventStream, pm("settings:get"))
	api.GET("/api/about", handleGetAboutInfo)

	api.GET("/api/subscribers", handleQuerySubscribers, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id", handleGetSubscriber, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/export", handleExportSubscriberData, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/bounces", handleGetSubscriberBounces, pm("bounces:get"))
	api.DELETE("/api/subscribers/:id/bounces", handleDeleteSubscriberBounces, pm("bounces:manage"))
	api.POST("/api/subscribers", handleCreateSubscriber, pm("subscribers:manage"))
	api.PUT("/api/subscribers/:id", handleUpdateSubscriber, pm("subscribers:manage"))
	api.POST("/api/subscribers/:id/optin", handleSubscriberSendOptin, pm("subscribers:manage"))
	api.PUT("/api/subscribers/blocklist", handleBlocklistSubscribers, pm("subscribers:manage"))
	api.PUT("/api/subscribers/:id/blocklist", handleBlocklistSubscribers, pm("subscribers:manage"))
	api.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists, pm("subscribers:manage"))
	api.PUT("/api/subscribers/lists", handleManageSubscriberLists, pm("subscribers:manage"))
	api.DELETE("/api/subscribers/:id", handleDeleteSubscribers, pm("subscribers:manage"))
	api.DELETE("/api/subscribers", handleDeleteSubscribers, pm("subscribers:manage"))

	api.GET("/api/bounces", handleGetBounces, pm("bounces:get"))
	api.GET("/api/bounces/:id", handleGetBounces, pm("bounces:get"))
	api.DELETE("/api/bounces", handleDeleteBounces, pm("bounces:manage"))
	api.DELETE("/api/bounces/:id", handleDeleteBounces, pm("bounces:manage"))

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	api.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery, pm("subscribers:manage"))
	api.PUT("/api/subscribers/query/blocklist", handleBlocklistSubscribersByQuery, pm("subscribers:manage"))
	api.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery, pm("subscribers:manage"))
	api.GET("/api/subscribers/export",
		handleExportSubscribers,
		middleware.GzipWithConfig(middleware.GzipConfig{Level: 9}),
		pm("subscribers:get_all", "subscribers:get"),
	)

	api.GET("/api/import/subscribers", handleGetImportSubscribers, pm("subscribers:import"))
	api.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats, pm("subscribers:import"))
	api.POST("/api/import/subscribers", handleImportSubscribers, pm("subscribers:import"))
	api.DELETE("/api/import/subscribers", handleStopImportSubscribers, pm("subscribers:import"))

	// Individual list permissions are applied directly within handleGetLists.
	api.GET("/api/lists", handleGetLists)
	api.GET("/api/lists/:id", listPerm(handleGetList))
	api.POST("/api/lists", handleCreateList, pm("lists:manage_all"))
	api.PUT("/api/lists/:id", listPerm(handleUpdateList))
	api.DELETE("/api/lists/:id", listPerm(handleDeleteLists))

	api.GET("/api/campaigns", handleGetCampaigns, pm("campaigns:get"))
	api.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats, pm("campaigns:get"))
	api.GET("/api/campaigns/:id", handleGetCampaign, pm("campaigns:get"))
	api.GET("/api/campaigns/analytics/:type", handleGetCampaignViewAnalytics, pm("campaigns:get_analytics"))
	api.GET("/api/campaigns/:id/preview", handlePreviewCampaign, pm("campaigns:get"))
	api.POST("/api/campaigns/:id/preview", handlePreviewCampaign, pm("campaigns:get"))
	api.POST("/api/campaigns/:id/content", handleCampaignContent, pm("campaigns:manage"))
	api.POST("/api/campaigns/:id/text", handlePreviewCampaign, pm("campaigns:manage"))
	api.POST("/api/campaigns/:id/test", handleTestCampaign, pm("campaigns:manage"))
	api.POST("/api/campaigns", handleCreateCampaign, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id", handleUpdateCampaign, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id/archive", handleUpdateCampaignArchive, pm("campaigns:manage"))
	api.DELETE("/api/campaigns/:id", handleDeleteCampaign, pm("campaigns:manage"))

	api.GET("/api/media", handleGetMedia, pm("media:get"))
	api.GET("/api/media/:id", handleGetMedia, pm("media:get"))
	api.POST("/api/media", handleUploadMedia, pm("media:manage"))
	api.DELETE("/api/media/:id", handleDeleteMedia, pm("media:manage"))

	api.GET("/api/templates", handleGetTemplates, pm("templates:get"))
	api.GET("/api/templates/:id", handleGetTemplates, pm("templates:get"))
	api.GET("/api/templates/:id/preview", handlePreviewTemplate, pm("templates:get"))
	api.POST("/api/templates/preview", handlePreviewTemplate, pm("templates:get"))
	api.POST("/api/templates", handleCreateTemplate, pm("templates:manage"))
	api.PUT("/api/templates/:id", handleUpdateTemplate, pm("templates:manage"))
	api.PUT("/api/templates/:id/default", handleTemplateSetDefault, pm("templates:manage"))
	api.DELETE("/api/templates/:id", handleDeleteTemplate, pm("templates:manage"))

	api.DELETE("/api/maintenance/subscribers/:type", handleGCSubscribers, pm("settings:maintain"))
	api.DELETE("/api/maintenance/analytics/:type", handleGCCampaignAnalytics, pm("settings:maintain"))
	api.DELETE("/api/maintenance/subscriptions/unconfirmed", handleGCSubscriptions, pm("settings:maintain"))

	api.POST("/api/tx", handleSendTxMessage, pm("tx:send"))

	api.GET("/api/profile", handleGetUserProfile)
	api.PUT("/api/profile", handleUpdateUserProfile)
	api.GET("/api/users", handleGetUsers, pm("users:get"))
	api.GET("/api/users/:id", handleGetUsers, pm("users:get"))
	api.POST("/api/users", handleCreateUser, pm("users:manage"))
	api.PUT("/api/users/:id", handleUpdateUser, pm("users:manage"))
	api.DELETE("/api/users", handleDeleteUsers, pm("users:manage"))
	api.DELETE("/api/users/:id", handleDeleteUsers, pm("users:manage"))
	api.POST("/api/logout", handleLogout)

	api.GET("/api/roles/users", handleGetUserRoles, pm("roles:get"))
	api.GET("/api/roles/lists", handleGeListRoles, pm("roles:get"))
	api.POST("/api/roles/users", handleCreateUserRole, pm("roles:manage"))
	api.POST("/api/roles/lists", handleCreateListRole, pm("roles:manage"))
	api.PUT("/api/roles/users/:id", handleUpdateUserRole, pm("roles:manage"))
	api.PUT("/api/roles/lists/:id", handleUpdateListRole, pm("roles:manage"))
	api.DELETE("/api/roles/:id", handleDeleteRole, pm("roles:manage"))

	if h.app.constants.BounceWebhooksEnabled {
		// Private authenticated bounce endpoint.
		api.POST("/webhooks/bounce", handleBounceWebhook, pm("webhooks:post_bounce"))

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

	if h.app.constants.Security.OIDC.Enabled {
		p.POST("/auth/oidc", handleOIDCLogin)
		p.GET("/auth/oidc", handleOIDCFinish)
	}

	// Public APIs.
	p.GET("/api/public/lists", handleGetPublicLists)
	p.POST("/api/public/subscription", handlePublicSubscription)
	if h.app.constants.EnablePublicArchive {
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

	if h.app.constants.EnablePublicArchive {
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
			makeMsgTpl("404 - "+h.app.i18n.T("public.notFoundTitle"), "", ""))
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
