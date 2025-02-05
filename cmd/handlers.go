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
		I18n:                app.i18n,
	}

	h := &Handler{
		app:  app,
		echo: e,
	}

	return h, nil
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
	admin.GET("", h.handleAdminPage)
	admin.GET("/custom.css", h.serveCustomAppearance("admin.custom_css"))
	admin.GET("/custom.js", h.serveCustomAppearance("admin.custom_js"))
	admin.GET("/*", h.handleAdminPage)

	pm := h.app.auth.Perm

	// API endpoints.
	api.GET("/api/health", h.handleHealthCheck)
	api.GET("/api/config", h.handleGetServerConfig)
	api.GET("/api/lang/:lang", h.handleGetI18nLang)
	api.GET("/api/dashboard/charts", h.handleGetDashboardCharts)
	api.GET("/api/dashboard/counts", h.handleGetDashboardCounts)

	api.GET("/api/settings", h.handleGetSettings, pm("settings:get"))
	api.PUT("/api/settings", h.handleUpdateSettings, pm("settings:manage"))
	api.POST("/api/settings/smtp/test", h.handleTestSMTPSettings, pm("settings:manage"))
	api.POST("/api/admin/reload", h.handleReloadApp, pm("settings:manage"))
	api.GET("/api/logs", h.handleGetLogs, pm("settings:get"))
	api.GET("/api/events", h.handleEventStream, pm("settings:get"))
	api.GET("/api/about", h.handleGetAboutInfo)

	api.GET("/api/subscribers", h.handleQuerySubscribers, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id", h.handleGetSubscriber, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/export", h.handleExportSubscriberData, pm("subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/bounces", h.handleGetSubscriberBounces, pm("bounces:get"))
	api.DELETE("/api/subscribers/:id/bounces", h.handleDeleteSubscriberBounces, pm("bounces:manage"))
	api.POST("/api/subscribers", h.handleCreateSubscriber, pm("subscribers:manage"))
	api.PUT("/api/subscribers/:id", h.handleUpdateSubscriber, pm("subscribers:manage"))
	api.POST("/api/subscribers/:id/optin", h.handleSubscriberSendOptin, pm("subscribers:manage"))
	api.PUT("/api/subscribers/blocklist", h.handleBlocklistSubscribers, pm("subscribers:manage"))
	api.PUT("/api/subscribers/:id/blocklist", h.handleBlocklistSubscribers, pm("subscribers:manage"))
	api.PUT("/api/subscribers/lists/:id", h.handleManageSubscriberLists, pm("subscribers:manage"))
	api.PUT("/api/subscribers/lists", h.handleManageSubscriberLists, pm("subscribers:manage"))
	api.DELETE("/api/subscribers/:id", h.handleDeleteSubscribers, pm("subscribers:manage"))
	api.DELETE("/api/subscribers", h.handleDeleteSubscribers, pm("subscribers:manage"))

	api.GET("/api/bounces", h.handleGetBounces, pm("bounces:get"))
	api.GET("/api/bounces/:id", h.handleGetBounces, pm("bounces:get"))
	api.DELETE("/api/bounces", h.handleDeleteBounces, pm("bounces:manage"))
	api.DELETE("/api/bounces/:id", h.handleDeleteBounces, pm("bounces:manage"))

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	api.POST("/api/subscribers/query/delete", h.handleDeleteSubscribersByQuery, pm("subscribers:manage"))
	api.PUT("/api/subscribers/query/blocklist", h.handleBlocklistSubscribersByQuery, pm("subscribers:manage"))
	api.PUT("/api/subscribers/query/lists", h.handleManageSubscriberListsByQuery, pm("subscribers:manage"))
	api.GET("/api/subscribers/export",
		h.handleExportSubscribers,
		middleware.GzipWithConfig(middleware.GzipConfig{Level: 9}),
		pm("subscribers:get_all", "subscribers:get"),
	)

	api.GET("/api/import/subscribers", h.handleGetImportSubscribers, pm("subscribers:import"))
	api.GET("/api/import/subscribers/logs", h.handleGetImportSubscriberStats, pm("subscribers:import"))
	api.POST("/api/import/subscribers", h.handleImportSubscribers, pm("subscribers:import"))
	api.DELETE("/api/import/subscribers", h.handleStopImportSubscribers, pm("subscribers:import"))

	// Individual list permissions are applied directly within handleGetLists.
	api.GET("/api/lists", h.handleGetLists)
	api.GET("/api/lists/:id", h.handleGetList, h.listPerm())
	api.POST("/api/lists", h.handleCreateList, pm("lists:manage_all"))
	api.PUT("/api/lists/:id", h.handleUpdateList, h.listPerm())
	api.DELETE("/api/lists/:id", h.handleDeleteLists, h.listPerm())

	api.GET("/api/campaigns", h.handleGetCampaigns, pm("campaigns:get"))
	api.GET("/api/campaigns/running/stats", h.handleGetRunningCampaignStats, pm("campaigns:get"))
	api.GET("/api/campaigns/:id", h.handleGetCampaign, pm("campaigns:get"))
	api.GET("/api/campaigns/analytics/:type", h.handleGetCampaignViewAnalytics, pm("campaigns:get_analytics"))
	api.GET("/api/campaigns/:id/preview", h.handlePreviewCampaign, pm("campaigns:get"))
	api.POST("/api/campaigns/:id/preview", h.handlePreviewCampaign, pm("campaigns:get"))
	api.POST("/api/campaigns/:id/content", h.handleCampaignContent, pm("campaigns:manage"))
	api.POST("/api/campaigns/:id/text", h.handlePreviewCampaign, pm("campaigns:manage"))
	api.POST("/api/campaigns/:id/test", h.handleTestCampaign, pm("campaigns:manage"))
	api.POST("/api/campaigns", h.handleCreateCampaign, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id", h.handleUpdateCampaign, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id/status", h.handleUpdateCampaignStatus, pm("campaigns:manage"))
	api.PUT("/api/campaigns/:id/archive", h.handleUpdateCampaignArchive, pm("campaigns:manage"))
	api.DELETE("/api/campaigns/:id", h.handleDeleteCampaign, pm("campaigns:manage"))

	api.GET("/api/media", h.handleGetMedia, pm("media:get"))
	api.GET("/api/media/:id", h.handleGetMedia, pm("media:get"))
	api.POST("/api/media", h.handleUploadMedia, pm("media:manage"))
	api.DELETE("/api/media/:id", h.handleDeleteMedia, pm("media:manage"))

	api.GET("/api/templates", h.handleGetTemplates, pm("templates:get"))
	api.GET("/api/templates/:id", h.handleGetTemplates, pm("templates:get"))
	api.GET("/api/templates/:id/preview", h.handlePreviewTemplate, pm("templates:get"))
	api.POST("/api/templates/preview", h.handlePreviewTemplate, pm("templates:get"))
	api.POST("/api/templates", h.handleCreateTemplate, pm("templates:manage"))
	api.PUT("/api/templates/:id", h.handleUpdateTemplate, pm("templates:manage"))
	api.PUT("/api/templates/:id/default", h.handleTemplateSetDefault, pm("templates:manage"))
	api.DELETE("/api/templates/:id", h.handleDeleteTemplate, pm("templates:manage"))

	api.DELETE("/api/maintenance/subscribers/:type", h.handleGCSubscribers, pm("settings:maintain"))
	api.DELETE("/api/maintenance/analytics/:type", h.handleGCCampaignAnalytics, pm("settings:maintain"))
	api.DELETE("/api/maintenance/subscriptions/unconfirmed", h.handleGCSubscriptions, pm("settings:maintain"))

	api.POST("/api/tx", h.handleSendTxMessage, pm("tx:send"))

	api.GET("/api/profile", h.handleGetUserProfile)
	api.PUT("/api/profile", h.handleUpdateUserProfile)
	api.GET("/api/users", h.handleGetUsers, pm("users:get"))
	api.GET("/api/users/:id", h.handleGetUsers, pm("users:get"))
	api.POST("/api/users", h.handleCreateUser, pm("users:manage"))
	api.PUT("/api/users/:id", h.handleUpdateUser, pm("users:manage"))
	api.DELETE("/api/users", h.handleDeleteUsers, pm("users:manage"))
	api.DELETE("/api/users/:id", h.handleDeleteUsers, pm("users:manage"))
	api.POST("/api/logout", h.handleLogout)

	api.GET("/api/roles/users", h.handleGetUserRoles, pm("roles:get"))
	api.GET("/api/roles/lists", h.handleGetListRoles, pm("roles:get"))
	api.POST("/api/roles/users", h.handleCreateUserRole, pm("roles:manage"))
	api.POST("/api/roles/lists", h.handleCreateListRole, pm("roles:manage"))
	api.PUT("/api/roles/users/:id", h.handleUpdateUserRole, pm("roles:manage"))
	api.PUT("/api/roles/lists/:id", h.handleUpdateListRole, pm("roles:manage"))
	api.DELETE("/api/roles/:id", h.handleDeleteRole, pm("roles:manage"))

	if h.app.constants.BounceWebhooksEnabled {
		// Private authenticated bounce endpoint.
		api.POST("/webhooks/bounce", h.handleBounceWebhook, pm("webhooks:post_bounce"))

		// Public bounce endpoints for webservices like SES.
		p.POST("/webhooks/service/:service", h.handleBounceWebhook)
	}

	// =================================================================
	// Public API endpoints.

	// Landing page.
	p.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
	})

	// Public admin endpoints (login page, OIDC endpoints).
	p.GET(path.Join(uriAdmin, "/login"), h.handleLoginPage)
	p.POST(path.Join(uriAdmin, "/login"), h.handleLoginPage)

	if h.app.constants.Security.OIDC.Enabled {
		p.POST("/auth/oidc", h.handleOIDCLogin)
		p.GET("/auth/oidc", h.handleOIDCFinish)
	}

	// Public APIs.
	p.GET("/api/public/lists", h.handleGetPublicLists)
	p.POST("/api/public/subscription", h.handlePublicSubscription)
	if h.app.constants.EnablePublicArchive {
		p.GET("/api/public/archive", h.handleGetCampaignArchives)
	}

	// /public/static/* file server is registered in initHTTPServer().
	// Public subscriber facing views.
	p.GET("/subscription/form", h.handleSubscriptionFormPage)
	p.POST("/subscription/form", h.handleSubscriptionForm)
	p.GET("/subscription/:campUUID/:subUUID",
		h.handleSubscriptionPage,
		h.validateUUID("campUUID", "subUUID"),
		h.subscriberExists("subUUID"),
		noIndex(),
	)
	p.POST("/subscription/:campUUID/:subUUID", h.handleSubscriptionPrefs,
		h.validateUUID("campUUID", "subUUID"),
		h.subscriberExists("subUUID"),
	)
	p.GET("/subscription/optin/:subUUID", h.handleOptinPage,
		h.validateUUID("subUUID"),
		h.subscriberExists("subUUID"),
		noIndex(),
	)
	p.POST("/subscription/optin/:subUUID", h.handleOptinPage,
		h.validateUUID("subUUID"),
		h.subscriberExists("subUUID"),
	)
	p.POST("/subscription/export/:subUUID", h.handleSelfExportSubscriberData,
		h.validateUUID("subUUID"),
		h.subscriberExists("subUUID"),
	)
	p.POST("/subscription/wipe/:subUUID", h.handleWipeSubscriberData,
		h.validateUUID("subUUID"),
		h.subscriberExists("subUUID"),
	)
	p.GET("/link/:linkUUID/:campUUID/:subUUID", h.handleLinkRedirect,
		h.validateUUID("linkUUID", "campUUID", "subUUID"),
		noIndex(),
	)
	p.GET("/campaign/:campUUID/:subUUID", h.handleViewCampaignMessage,
		h.validateUUID("campUUID", "subUUID"),
		noIndex(),
	)
	p.GET("/campaign/:campUUID/:subUUID/px.png", h.handleRegisterCampaignView,
		h.validateUUID("campUUID", "subUUID"),
		noIndex(),
	)

	if h.app.constants.EnablePublicArchive {
		p.GET("/archive", h.handleCampaignArchivesPage)
		p.GET("/archive.xml", h.handleGetCampaignArchivesFeed)
		p.GET("/archive/:id", h.handleCampaignArchivePage)
		p.GET("/archive/latest", h.handleCampaignArchivePageLatest)
	}

	p.GET("/public/custom.css", h.serveCustomAppearance("public.custom_css"))
	p.GET("/public/custom.js", h.serveCustomAppearance("public.custom_js"))

	// Public health API endpoint.
	p.GET("/health", h.handleHealthCheck)

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
func (h *Handler) handleAdminPage(c echo.Context) error {

	b, err := h.app.fs.Read(path.Join(uriAdmin, "/index.html"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	b = bytes.ReplaceAll(b, []byte("asset_version"), []byte(h.app.constants.AssetVersion))

	return c.HTMLBlob(http.StatusOK, b)
}

// handleHealthCheck is a healthcheck endpoint that returns a 200 response.
func (h *Handler) handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{true})
}

// serveCustomAppearance serves the given custom CSS/JS appearance blob
// meant for customizing public and admin pages from the admin settings UI.
func (h *Handler) serveCustomAppearance(name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			out []byte
			hdr string
		)

		switch name {
		case "admin.custom_css":
			out = h.app.constants.Appearance.AdminCSS
			hdr = "text/css; charset=utf-8"

		case "admin.custom_js":
			out = h.app.constants.Appearance.AdminJS
			hdr = "application/javascript; charset=utf-8"

		case "public.custom_css":
			out = h.app.constants.Appearance.PublicCSS
			hdr = "text/css; charset=utf-8"

		case "public.custom_js":
			out = h.app.constants.Appearance.PublicJS
			hdr = "application/javascript; charset=utf-8"
		}

		return c.Blob(http.StatusOK, hdr, out)
	}
}

// validateUUID middleware validates the UUID string format for a given set of params.
func (h *Handler) validateUUID(params ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, p := range params {
				if !reUUID.MatchString(c.Param(p)) {
					return c.Render(http.StatusBadRequest, tplMessage,
						makeMsgTpl(h.app.i18n.T("public.errorTitle"), "",
							h.app.i18n.T("globals.messages.invalidUUID")))
				}
			}
			return next(c)
		}
	}
}

// subscriberExists middleware checks if a subscriber exists given the UUID
// param in a request.
func (h *Handler) subscriberExists(paramName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				subUUID = c.Param(paramName)
			)

			if _, err := h.app.core.GetSubscriber(0, subUUID, ""); err != nil {
				if er, ok := err.(*echo.HTTPError); ok && er.Code == http.StatusBadRequest {
					return c.Render(http.StatusNotFound, tplMessage,
						makeMsgTpl(h.app.i18n.T("public.notFoundTitle"), "", er.Message.(string)))
				}

				h.app.log.Printf("error checking subscriber existence: %v", err)
				return c.Render(http.StatusInternalServerError, tplMessage,
					makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.errorProcessingRequest")))
			}

			return next(c)
		}
	}
}

// noIndex adds the HTTP header requesting robots to not crawl the page.
func noIndex() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Robots-Tag", "noindex")
			return next(c)
		}
	}
}
