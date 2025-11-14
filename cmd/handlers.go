package main

import (
	"bytes"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"

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

var (
	reUUID = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
)

// registerHandlers registers HTTP handlers.
func initHTTPHandlers(e *echo.Echo, a *App) {
	// Default error handler.
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			a.log.Println(err.Error())
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	// Configure CORS middleware if domains are configured.
	if len(a.cfg.Security.CorsOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: a.cfg.Security.CorsOrigins,
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		}))
	}

	// =================================================================
	// Authenticated non /api handlers.
	{
		// Attach a middleware to the group that checks for auth.
		g := e.Group("", a.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				u := c.Get(auth.UserHTTPCtxKey)

				// On no-auth, redirect to login page
				if _, ok := u.(*echo.HTTPError); ok {
					u, _ := url.Parse(a.urlCfg.LoginURL)
					q := url.Values{}
					q.Set("next", c.Request().RequestURI)
					u.RawQuery = q.Encode()
					return c.Redirect(http.StatusTemporaryRedirect, u.String())
				}

				return next(c)
			}
		})

		// Authenticated endpoints.
		g.GET(path.Join(uriAdmin, ""), a.AdminPage)
		g.GET(path.Join(uriAdmin, "/custom.css"), serveCustomAppearance("admin.custom_css"))
		g.GET(path.Join(uriAdmin, "/custom.js"), serveCustomAppearance("admin.custom_js"))
		g.GET(path.Join(uriAdmin, "/*"), a.AdminPage)
	}

	// =================================================================
	// Authenticated /api/* handlers.
	{
		var (
			// Permission check middleware.
			pm = a.auth.Perm

			// Attach a middleware to the group that checks for auth.
			g = e.Group("", a.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					u := c.Get(auth.UserHTTPCtxKey)

					// On no-auth, respond with a JSON error.
					if err, ok := u.(*echo.HTTPError); ok {
						return err
					}

					return next(c)
				}
			})
		)

		// API endpoints.
		g.GET("/api/health", a.HealthCheck)
		g.GET("/api/config", a.GetServerConfig)
		g.GET("/api/lang/:lang", a.GetI18nLang)
		g.GET("/api/dashboard/charts", a.GetDashboardCharts)
		g.GET("/api/dashboard/counts", a.GetDashboardCounts)

		g.GET("/api/settings", pm(a.GetSettings, "settings:get"))
		g.PUT("/api/settings", pm(a.UpdateSettings, "settings:manage"))
		g.POST("/api/settings/smtp/test", pm(a.TestSMTPSettings, "settings:manage"))
		g.POST("/api/admin/reload", pm(a.ReloadApp, "settings:manage"))
		g.GET("/api/logs", pm(a.GetLogs, "settings:get"))
		g.GET("/api/events", pm(a.EventStream, "settings:get"))
		g.GET("/api/about", a.GetAboutInfo)

		g.GET("/api/subscribers", pm(a.QuerySubscribers, "subscribers:get_all", "subscribers:get"))
		g.GET("/api/subscribers/:id", pm(hasID(a.GetSubscriber), "subscribers:get_all", "subscribers:get"))
		g.GET("/api/subscribers/:id/activity", pm(hasID(a.GetSubscriberActivity), "subscribers:get_all", "subscribers:get"))
		g.GET("/api/subscribers/:id/export", pm(hasID(a.ExportSubscriberData), "subscribers:get_all", "subscribers:get"))
		g.GET("/api/subscribers/:id/bounces", pm(hasID(a.GetSubscriberBounces), "bounces:get"))
		g.DELETE("/api/subscribers/:id/bounces", pm(hasID(a.DeleteSubscriberBounces), "bounces:manage"))
		g.POST("/api/subscribers", pm(a.CreateSubscriber, "subscribers:manage"))
		g.PUT("/api/subscribers/:id", pm(hasID(a.UpdateSubscriber), "subscribers:manage"))
		g.POST("/api/subscribers/:id/optin", pm(hasID(a.SubscriberSendOptin), "subscribers:manage"))
		g.PUT("/api/subscribers/blocklist", pm(a.BlocklistSubscribers, "subscribers:manage"))
		g.PUT("/api/subscribers/:id/blocklist", pm(hasID(a.BlocklistSubscriber), "subscribers:manage"))
		g.PUT("/api/subscribers/lists/:id", pm(a.ManageSubscriberLists, "subscribers:manage"))
		g.PUT("/api/subscribers/lists", pm(a.ManageSubscriberLists, "subscribers:manage"))
		g.DELETE("/api/subscribers/:id", pm(hasID(a.DeleteSubscriber), "subscribers:manage"))
		g.DELETE("/api/subscribers", pm(a.DeleteSubscribers, "subscribers:manage"))

		g.GET("/api/bounces", pm(a.GetBounces, "bounces:get"))
		g.PUT("/api/bounces/blocklist", pm(a.BlocklistBouncedSubscribers, "bounces:manage"))
		g.GET("/api/bounces/:id", pm(hasID(a.GetBounce), "bounces:get"))
		g.DELETE("/api/bounces", pm(a.DeleteBounces, "bounces:manage"))
		g.DELETE("/api/bounces/:id", pm(hasID(a.DeleteBounce), "bounces:manage"))

		// Subscriber operations based on arbitrary SQL queries.
		// These aren't very REST-like.
		g.POST("/api/subscribers/query/delete", pm(a.DeleteSubscribersByQuery, "subscribers:manage"))
		g.PUT("/api/subscribers/query/blocklist", pm(a.BlocklistSubscribersByQuery, "subscribers:manage"))
		g.PUT("/api/subscribers/query/lists", pm(a.ManageSubscriberListsByQuery, "subscribers:manage"))
		g.GET("/api/subscribers/export",
			pm(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(a.ExportSubscribers), "subscribers:get_all", "subscribers:get"))

		g.GET("/api/import/subscribers", pm(a.GetImportSubscribers, "subscribers:import"))
		g.GET("/api/import/subscribers/logs", pm(a.GetImportSubscriberStats, "subscribers:import"))
		g.POST("/api/import/subscribers", pm(a.ImportSubscribers, "subscribers:import"))
		g.DELETE("/api/import/subscribers", pm(a.StopImportSubscribers, "subscribers:import"))

		// Individual list permissions are applied directly within handleGetLists.
		g.GET("/api/lists", a.GetLists)
		g.GET("/api/lists/:id", hasID(a.GetList))
		g.POST("/api/lists", pm(a.CreateList, "lists:manage_all"))
		g.PUT("/api/lists/:id", hasID(a.UpdateList))
		g.DELETE("/api/lists/:id", hasID(a.DeleteLists))

		g.GET("/api/campaigns", pm(a.GetCampaigns, "campaigns:get_all", "campaigns:get"))
		g.GET("/api/campaigns/running/stats", pm(a.GetRunningCampaignStats, "campaigns:get_all", "campaigns:get"))
		g.GET("/api/campaigns/:id", pm(hasID(a.GetCampaign), "campaigns:get_all", "campaigns:get"))
		g.GET("/api/campaigns/analytics/:type", pm(a.GetCampaignViewAnalytics, "campaigns:get_analytics"))
		g.GET("/api/campaigns/:id/preview", pm(hasID(a.PreviewCampaign), "campaigns:get_all", "campaigns:get"))
		g.POST("/api/campaigns/:id/preview/archive", pm(hasID(a.PreviewCampaignArchive), "campaigns:get_all", "campaigns:get"))
		g.POST("/api/campaigns/:id/preview", pm(hasID(a.PreviewCampaign), "campaigns:get_all", "campaigns:get"))
		g.POST("/api/campaigns/:id/content", pm(hasID(a.CampaignContent), "campaigns:manage_all", "campaigns:manage"))
		g.POST("/api/campaigns/:id/text", pm(hasID(a.PreviewCampaign), "campaigns:get"))
		g.POST("/api/campaigns/:id/test", pm(hasID(a.TestCampaign), "campaigns:manage_all", "campaigns:manage"))
		g.POST("/api/campaigns", pm(a.CreateCampaign, "campaigns:manage_all", "campaigns:manage"))
		g.PUT("/api/campaigns/:id", pm(hasID(a.UpdateCampaign), "campaigns:manage_all", "campaigns:manage"))
		g.PUT("/api/campaigns/:id/status", pm(hasID(a.UpdateCampaignStatus), "campaigns:manage_all", "campaigns:manage"))
		g.PUT("/api/campaigns/:id/archive", pm(hasID(a.UpdateCampaignArchive), "campaigns:manage_all", "campaigns:manage"))
		g.DELETE("/api/campaigns/:id", pm(hasID(a.DeleteCampaign), "campaigns:manage_all", "campaigns:manage"))

		g.GET("/api/media", pm(a.GetAllMedia, "media:get"))
		g.GET("/api/media/:id", pm(hasID(a.GetMedia), "media:get"))
		g.POST("/api/media", pm(a.UploadMedia, "media:manage"))
		g.DELETE("/api/media/:id", pm(hasID(a.DeleteMedia), "media:manage"))

		g.GET("/api/templates", pm(a.GetTemplates, "templates:get"))
		g.GET("/api/templates/:id", pm(hasID(a.GetTemplate), "templates:get"))
		g.GET("/api/templates/:id/preview", pm(hasID(a.PreviewTemplate), "templates:get"))
		g.POST("/api/templates/preview", pm(a.PreviewTemplateBody, "templates:get"))
		g.POST("/api/templates", pm(a.CreateTemplate, "templates:manage"))
		g.PUT("/api/templates/:id", pm(hasID(a.UpdateTemplate), "templates:manage"))
		g.PUT("/api/templates/:id/default", pm(hasID(a.TemplateSetDefault), "templates:manage"))
		g.DELETE("/api/templates/:id", pm(hasID(a.DeleteTemplate), "templates:manage"))

		g.DELETE("/api/maintenance/subscribers/:type", pm(a.GCSubscribers, "settings:maintain"))
		g.DELETE("/api/maintenance/analytics/:type", pm(a.GCCampaignAnalytics, "settings:maintain"))
		g.DELETE("/api/maintenance/subscriptions/unconfirmed", pm(a.GCSubscriptions, "settings:maintain"))

		g.POST("/api/tx", pm(a.SendTxMessage, "tx:send"))

		g.GET("/api/profile", a.GetUserProfile)
		g.PUT("/api/profile", a.UpdateUserProfile)
		g.GET("/api/users", pm(a.GetUsers, "users:get"))
		g.GET("/api/users/:id", pm(hasID(a.GetUser), "users:get"))
		g.POST("/api/users", pm(a.CreateUser, "users:manage"))
		g.PUT("/api/users/:id", pm(hasID(a.UpdateUser), "users:manage"))
		g.DELETE("/api/users", pm(a.DeleteUsers, "users:manage"))
		g.DELETE("/api/users/:id", pm(hasID(a.DeleteUser), "users:manage"))
		g.POST("/api/logout", a.Logout)

		g.GET("/api/roles/users", pm(a.GetUserRoles, "roles:get"))
		g.GET("/api/roles/lists", pm(a.GeListRoles, "roles:get"))
		g.POST("/api/roles/users", pm(a.CreateUserRole, "roles:manage"))
		g.POST("/api/roles/lists", pm(a.CreateListRole, "roles:manage"))
		g.PUT("/api/roles/users/:id", pm(hasID(a.UpdateUserRole), "roles:manage"))
		g.PUT("/api/roles/lists/:id", pm(hasID(a.UpdateListRole), "roles:manage"))
		g.DELETE("/api/roles/:id", pm(hasID(a.DeleteRole), "roles:manage"))

		if a.cfg.BounceWebhooksEnabled {
			// Private authenticated bounce endpoint.
			g.POST("/webhooks/bounce", pm(a.BounceWebhook, "webhooks:post_bounce"))
		}
	}

	// =================================================================
	// Public API endpoints.
	{
		// Public unauthenticated endpoints.
		g := e.Group("")

		if a.cfg.BounceWebhooksEnabled {
			// Public bounce endpoints for webservices like SES.
			g.POST("/webhooks/service/:service", a.BounceWebhook)
		}

		// Landing page.
		g.GET("/", func(c echo.Context) error {
			return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
		})

		// Public admin endpoints (login page, OIDC endpoints).
		g.GET(path.Join(uriAdmin, "/login"), a.LoginPage)
		g.POST(path.Join(uriAdmin, "/login"), a.LoginPage)

		if a.cfg.Security.OIDC.Enabled {
			g.POST("/auth/oidc", a.OIDCLogin)
			g.GET("/auth/oidc", a.OIDCFinish)
		}

		// Public APIs.
		g.GET("/api/public/lists", a.GetPublicLists)
		g.POST("/api/public/subscription", a.PublicSubscription)
		g.GET("/api/public/captcha/altcha", a.AltchaChallenge)
		if a.cfg.EnablePublicArchive {
			g.GET("/api/public/archive", a.GetCampaignArchives)
		}

		// /public/static/* file server is registered in initHTTPServer().
		// Public subscriber facing views.
		g.GET("/subscription/form", a.SubscriptionFormPage)
		g.POST("/subscription/form", a.SubscriptionForm)
		g.GET("/subscription/:campUUID/:subUUID", noIndex(a.hasUUID(a.hasSub(a.SubscriptionPage), "campUUID", "subUUID")))
		g.POST("/subscription/:campUUID/:subUUID", a.hasUUID(a.hasSub(a.SubscriptionPrefs), "campUUID", "subUUID"))
		g.GET("/subscription/optin/:subUUID", noIndex(a.hasUUID(a.hasSub(a.OptinPage), "subUUID")))
		g.POST("/subscription/optin/:subUUID", a.hasUUID(a.hasSub(a.OptinPage), "subUUID"))
		g.POST("/subscription/export/:subUUID", a.hasUUID(a.hasSub(a.SelfExportSubscriberData), "subUUID"))
		g.POST("/subscription/wipe/:subUUID", a.hasUUID(a.hasSub(a.WipeSubscriberData), "subUUID"))
		g.GET("/link/:linkUUID/:campUUID/:subUUID", noIndex(a.hasUUID(a.LinkRedirect, "linkUUID", "campUUID", "subUUID")))
		g.GET("/campaign/:campUUID/:subUUID", noIndex(a.hasUUID(a.ViewCampaignMessage, "campUUID", "subUUID")))
		g.GET("/campaign/:campUUID/:subUUID/px.png", noIndex(a.hasUUID(a.RegisterCampaignView, "campUUID", "subUUID")))

		if a.cfg.EnablePublicArchive {
			g.GET("/archive", a.CampaignArchivesPage)
			g.GET("/archive.xml", a.GetCampaignArchivesFeed)
			g.GET("/archive/:id", a.CampaignArchivePage)
			g.GET("/archive/latest", a.CampaignArchivePageLatest)
		}

		g.GET("/public/custom.css", serveCustomAppearance("public.custom_css"))
		g.GET("/public/custom.js", serveCustomAppearance("public.custom_js"))

		// Public health API endpoint.
		g.GET("/health", a.HealthCheck)

		// 404 pages.
		g.RouteNotFound("/*", func(c echo.Context) error {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl("404 - "+a.i18n.T("public.notFoundTitle"), "", ""))
		})
		g.RouteNotFound("/api/*", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "404 unknown endpoint")
		})
		g.RouteNotFound("/admin/*", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "404 page not found")
		})
	}
}

// AdminPage is the root handler that renders the Javascript admin frontend.
func (a *App) AdminPage(c echo.Context) error {
	b, err := a.fs.Read(path.Join(uriAdmin, "/index.html"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	b = bytes.ReplaceAll(b, []byte("asset_version"), []byte(a.cfg.AssetVersion))

	return c.HTMLBlob(http.StatusOK, b)
}

// HealthCheck is a healthcheck endpoint that returns a 200 response.
func (a *App) HealthCheck(c echo.Context) error {
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
			out = app.cfg.Appearance.AdminCSS
			hdr = "text/css; charset=utf-8"

		case "admin.custom_js":
			out = app.cfg.Appearance.AdminJS
			hdr = "application/javascript; charset=utf-8"

		case "public.custom_css":
			out = app.cfg.Appearance.PublicCSS
			hdr = "text/css; charset=utf-8"

		case "public.custom_js":
			out = app.cfg.Appearance.PublicJS
			hdr = "application/javascript; charset=utf-8"
		}

		return c.Blob(http.StatusOK, hdr, out)
	}
}

// hasUUID middleware validates the UUID string format for a given set of params.
func (a *App) hasUUID(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("public.errorTitle"), "",
					a.i18n.T("globals.messages.invalidUUID")))
			}
		}
		return next(c)
	}
}

// hasID middleware validates the :id param in the URL and sets its int value in the context.
func hasID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, _ := strconv.Atoi(c.Param("id"))
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
		}

		c.Set("id", id)
		return next(c)
	}
}

// hasSub middleware checks if a subscriber exists given the UUID
// param in a request.
func (a *App) hasSub(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		subUUID := c.Param("subUUID")

		if _, err := a.core.GetSubscriber(0, subUUID, ""); err != nil {
			if er, ok := err.(*echo.HTTPError); ok && er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(a.i18n.T("public.notFoundTitle"), "", er.Message.(string)))
			}

			a.log.Printf("error checking subscriber existence: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.errorProcessingRequest")))
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

// getID returns the :id param from the URL parsed and stored as an int by the hasID middleware.
func getID(c echo.Context) int {
	return c.Get("id").(int)
}
