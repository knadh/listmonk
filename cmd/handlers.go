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
	Data any `json:"data"`
}

type Handlers struct {
	app *App
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
				u := c.Get(auth.UserHTTPCtxKey)

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
				u := c.Get(auth.UserHTTPCtxKey)

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

		h = &Handlers{
			app: app,
		}
	)

	// Authenticated endpoints.
	a.GET(path.Join(uriAdmin, ""), h.AdminPage)
	a.GET(path.Join(uriAdmin, "/custom.css"), serveCustomAppearance("admin.custom_css"))
	a.GET(path.Join(uriAdmin, "/custom.js"), serveCustomAppearance("admin.custom_js"))
	a.GET(path.Join(uriAdmin, "/*"), h.AdminPage)

	pm := app.auth.Perm

	// API endpoints.
	api.GET("/api/health", h.HealthCheck)
	api.GET("/api/config", h.GetServerConfig)
	api.GET("/api/lang/:lang", h.GetI18nLang)
	api.GET("/api/dashboard/charts", h.GetDashboardCharts)
	api.GET("/api/dashboard/counts", h.GetDashboardCounts)

	api.GET("/api/settings", pm(h.GetSettings, "settings:get"))
	api.PUT("/api/settings", pm(h.UpdateSettings, "settings:manage"))
	api.POST("/api/settings/smtp/test", pm(h.TestSMTPSettings, "settings:manage"))
	api.POST("/api/admin/reload", pm(h.ReloadApp, "settings:manage"))
	api.GET("/api/logs", pm(h.GetLogs, "settings:get"))
	api.GET("/api/events", pm(h.EventStream, "settings:get"))
	api.GET("/api/about", h.GetAboutInfo)

	api.GET("/api/subscribers", pm(h.QuerySubscribers, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id", pm(h.GetSubscriber, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/export", pm(h.ExportSubscriberData, "subscribers:get_all", "subscribers:get"))
	api.GET("/api/subscribers/:id/bounces", pm(h.GetSubscriberBounces, "bounces:get"))
	api.DELETE("/api/subscribers/:id/bounces", pm(h.DeleteSubscriberBounces, "bounces:manage"))
	api.POST("/api/subscribers", pm(h.CreateSubscriber, "subscribers:manage"))
	api.PUT("/api/subscribers/:id", pm(h.UpdateSubscriber, "subscribers:manage"))
	api.POST("/api/subscribers/:id/optin", pm(h.SubscriberSendOptin, "subscribers:manage"))
	api.PUT("/api/subscribers/blocklist", pm(h.BlocklistSubscribers, "subscribers:manage"))
	api.PUT("/api/subscribers/:id/blocklist", pm(h.BlocklistSubscribers, "subscribers:manage"))
	api.PUT("/api/subscribers/lists/:id", pm(h.ManageSubscriberLists, "subscribers:manage"))
	api.PUT("/api/subscribers/lists", pm(h.ManageSubscriberLists, "subscribers:manage"))
	api.DELETE("/api/subscribers/:id", pm(h.DeleteSubscribers, "subscribers:manage"))
	api.DELETE("/api/subscribers", pm(h.DeleteSubscribers, "subscribers:manage"))

	api.GET("/api/bounces", pm(h.GetBounces, "bounces:get"))
	api.GET("/api/bounces/:id", pm(h.GetBounces, "bounces:get"))
	api.DELETE("/api/bounces", pm(h.DeleteBounces, "bounces:manage"))
	api.DELETE("/api/bounces/:id", pm(h.DeleteBounces, "bounces:manage"))

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	api.POST("/api/subscribers/query/delete", pm(h.DeleteSubscribersByQuery, "subscribers:manage"))
	api.PUT("/api/subscribers/query/blocklist", pm(h.BlocklistSubscribersByQuery, "subscribers:manage"))
	api.PUT("/api/subscribers/query/lists", pm(h.ManageSubscriberListsByQuery, "subscribers:manage"))
	api.GET("/api/subscribers/export",
		pm(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(h.ExportSubscribers), "subscribers:get_all", "subscribers:get"))

	api.GET("/api/import/subscribers", pm(h.GetImportSubscribers, "subscribers:import"))
	api.GET("/api/import/subscribers/logs", pm(h.GetImportSubscriberStats, "subscribers:import"))
	api.POST("/api/import/subscribers", pm(h.ImportSubscribers, "subscribers:import"))
	api.DELETE("/api/import/subscribers", pm(h.StopImportSubscribers, "subscribers:import"))

	// Individual list permissions are applied directly within handleGetLists.
	api.GET("/api/lists", h.GetLists)
	api.GET("/api/lists/:id", h.GetList)
	api.POST("/api/lists", pm(h.CreateList, "lists:manage_all"))
	api.PUT("/api/lists/:id", h.UpdateList)
	api.DELETE("/api/lists/:id", h.DeleteLists)

	api.GET("/api/campaigns", pm(h.GetCampaigns, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/running/stats", pm(h.GetRunningCampaignStats, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/:id", pm(h.GetCampaign, "campaigns:get_all", "campaigns:get"))
	api.GET("/api/campaigns/analytics/:type", pm(h.GetCampaignViewAnalytics, "campaigns:get_analytics"))
	api.GET("/api/campaigns/:id/preview", pm(h.PreviewCampaign, "campaigns:get_all", "campaigns:get"))
	api.POST("/api/campaigns/:id/preview", pm(h.PreviewCampaign, "campaigns:get_all", "campaigns:get"))
	api.POST("/api/campaigns/:id/content", pm(h.CampaignContent, "campaigns:manage_all", "campaigns:manage"))
	api.POST("/api/campaigns/:id/text", pm(h.PreviewCampaign, "campaigns:get"))
	api.POST("/api/campaigns/:id/test", pm(h.TestCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.POST("/api/campaigns", pm(h.CreateCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id", pm(h.UpdateCampaign, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id/status", pm(h.UpdateCampaignStatus, "campaigns:manage_all", "campaigns:manage"))
	api.PUT("/api/campaigns/:id/archive", pm(h.UpdateCampaignArchive, "campaigns:manage_all", "campaigns:manage"))
	api.DELETE("/api/campaigns/:id", pm(h.DeleteCampaign, "campaigns:manage_all", "campaigns:manage"))

	api.GET("/api/media", pm(h.GetMedia, "media:get"))
	api.GET("/api/media/:id", pm(h.GetMedia, "media:get"))
	api.POST("/api/media", pm(h.UploadMedia, "media:manage"))
	api.DELETE("/api/media/:id", pm(h.DeleteMedia, "media:manage"))

	api.GET("/api/templates", pm(h.GetTemplates, "templates:get"))
	api.GET("/api/templates/:id", pm(h.GetTemplates, "templates:get"))
	api.GET("/api/templates/:id/preview", pm(h.PreviewTemplate, "templates:get"))
	api.POST("/api/templates/preview", pm(h.PreviewTemplate, "templates:get"))
	api.POST("/api/templates", pm(h.CreateTemplate, "templates:manage"))
	api.PUT("/api/templates/:id", pm(h.UpdateTemplate, "templates:manage"))
	api.PUT("/api/templates/:id/default", pm(h.TemplateSetDefault, "templates:manage"))
	api.DELETE("/api/templates/:id", pm(h.DeleteTemplate, "templates:manage"))

	api.DELETE("/api/maintenance/subscribers/:type", pm(h.GCSubscribers, "settings:maintain"))
	api.DELETE("/api/maintenance/analytics/:type", pm(h.GCCampaignAnalytics, "settings:maintain"))
	api.DELETE("/api/maintenance/subscriptions/unconfirmed", pm(h.GCSubscriptions, "settings:maintain"))

	api.POST("/api/tx", pm(h.SendTxMessage, "tx:send"))

	api.GET("/api/profile", h.GetUserProfile)
	api.PUT("/api/profile", h.UpdateUserProfile)
	api.GET("/api/users", pm(h.GetUsers, "users:get"))
	api.GET("/api/users/:id", pm(h.GetUsers, "users:get"))
	api.POST("/api/users", pm(h.CreateUser, "users:manage"))
	api.PUT("/api/users/:id", pm(h.UpdateUser, "users:manage"))
	api.DELETE("/api/users", pm(h.DeleteUsers, "users:manage"))
	api.DELETE("/api/users/:id", pm(h.DeleteUsers, "users:manage"))
	api.POST("/api/logout", h.Logout)

	api.GET("/api/roles/users", pm(h.GetUserRoles, "roles:get"))
	api.GET("/api/roles/lists", pm(h.GeListRoles, "roles:get"))
	api.POST("/api/roles/users", pm(h.CreateUserRole, "roles:manage"))
	api.POST("/api/roles/lists", pm(h.CreateListRole, "roles:manage"))
	api.PUT("/api/roles/users/:id", pm(h.UpdateUserRole, "roles:manage"))
	api.PUT("/api/roles/lists/:id", pm(h.UpdateListRole, "roles:manage"))
	api.DELETE("/api/roles/:id", pm(h.DeleteRole, "roles:manage"))

	if app.constants.BounceWebhooksEnabled {
		// Private authenticated bounce endpoint.
		api.POST("/webhooks/bounce", pm(h.BounceWebhook, "webhooks:post_bounce"))

		// Public bounce endpoints for webservices like SES.
		p.POST("/webhooks/service/:service", h.BounceWebhook)
	}

	// =================================================================
	// Public API endpoints.

	// Landing page.
	p.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
	})

	// Public admin endpoints (login page, OIDC endpoints).
	p.GET(path.Join(uriAdmin, "/login"), h.LoginPage)
	p.POST(path.Join(uriAdmin, "/login"), h.LoginPage)

	if app.constants.Security.OIDC.Enabled {
		p.POST("/auth/oidc", h.OIDCLogin)
		p.GET("/auth/oidc", h.OIDCFinish)
	}

	// Public APIs.
	p.GET("/api/public/lists", h.GetPublicLists)
	p.POST("/api/public/subscription", h.PublicSubscription)
	if app.constants.EnablePublicArchive {
		p.GET("/api/public/archive", h.GetCampaignArchives)
	}

	// /public/static/* file server is registered in initHTTPServer().
	// Public subscriber facing views.
	p.GET("/subscription/form", h.SubscriptionFormPage)
	p.POST("/subscription/form", h.SubscriptionForm)
	p.GET("/subscription/:campUUID/:subUUID", noIndex(validateUUID(subscriberExists(h.SubscriptionPage),
		"campUUID", "subUUID")))
	p.POST("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(h.SubscriptionPrefs),
		"campUUID", "subUUID"))
	p.GET("/subscription/optin/:subUUID", noIndex(validateUUID(subscriberExists(h.OptinPage), "subUUID")))
	p.POST("/subscription/optin/:subUUID", validateUUID(subscriberExists(h.OptinPage), "subUUID"))
	p.POST("/subscription/export/:subUUID", validateUUID(subscriberExists(h.SelfExportSubscriberData),
		"subUUID"))
	p.POST("/subscription/wipe/:subUUID", validateUUID(subscriberExists(h.WipeSubscriberData),
		"subUUID"))
	p.GET("/link/:linkUUID/:campUUID/:subUUID", noIndex(validateUUID(h.LinkRedirect,
		"linkUUID", "campUUID", "subUUID")))
	p.GET("/campaign/:campUUID/:subUUID", noIndex(validateUUID(h.ViewCampaignMessage,
		"campUUID", "subUUID")))
	p.GET("/campaign/:campUUID/:subUUID/px.png", noIndex(validateUUID(h.RegisterCampaignView,
		"campUUID", "subUUID")))

	if app.constants.EnablePublicArchive {
		p.GET("/archive", h.CampaignArchivesPage)
		p.GET("/archive.xml", h.GetCampaignArchivesFeed)
		p.GET("/archive/:id", h.CampaignArchivePage)
		p.GET("/archive/latest", h.CampaignArchivePageLatest)
	}

	p.GET("/public/custom.css", serveCustomAppearance("public.custom_css"))
	p.GET("/public/custom.js", serveCustomAppearance("public.custom_js"))

	// Public health API endpoint.
	p.GET("/health", h.HealthCheck)

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

// AdminPage is the root handler that renders the Javascript admin frontend.
func (h *Handlers) AdminPage(c echo.Context) error {
	b, err := h.app.fs.Read(path.Join(uriAdmin, "/index.html"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	b = bytes.ReplaceAll(b, []byte("asset_version"), []byte(h.app.constants.AssetVersion))

	return c.HTMLBlob(http.StatusOK, b)
}

// HealthCheck is a healthcheck endpoint that returns a 200 response.
func (h *Handlers) HealthCheck(c echo.Context) error {
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
