package main

import (
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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
	e.HTTPErrorHandler = a.handleHTTPErr
	e.Pre(middleware.RemoveTrailingSlashWithConfig(middleware.RemoveTrailingSlashConfig{
		RedirectCode: http.StatusPermanentRedirect,
		Skipper:      skipAdminTrailingSlash,
	}))

	// Configure CORS middleware if domains are configured.
	if corsOrigins := trustedURLsToCORSOrigins(a.cfg.Security.TrustedURLs); len(corsOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: corsOrigins,
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		}))
	}

	// =================================================================
	// Authenticated non /api handlers.
	{
		pm := a.auth.Perm

		// Attach a middleware to the group that checks for auth.
		g := e.Group("", a.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
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

		// Authenticated endpoints. As views are ported to the SSR admin, register
		// their routes here. Unported /admin/* paths fall through to the 404 handler.
		g.GET(path.Join(uriAdmin, "/custom.css"), serveCustomAppearance("admin.custom_css"))
		g.GET(path.Join(uriAdmin, "/custom.js"), serveCustomAppearance("admin.custom_js"))
		g.GET(uriAdmin, a.adminPage(a.ViewDashboard))
		g.GET(path.Join(uriAdmin, "/lists"), a.adminPage(a.ViewLists))
		g.GET(path.Join(uriAdmin, "/lists/analytics"), a.adminPage(a.ViewListAnalytics))
		g.GET(path.Join(uriAdmin, "/lists/new"), a.adminPage(pm(a.ViewNewList, auth.PermListManageAll)))
		g.GET(path.Join(uriAdmin, "/lists/:id"), a.adminPage(hasID(a.ViewList)))
		g.GET(path.Join(uriAdmin, "/lists/:id/edit"), a.adminPage(hasID(a.ViewList)))
		g.GET(path.Join(uriAdmin, "/lists/forms"), a.adminPage(a.ViewForms))
		g.GET(path.Join(uriAdmin, "/campaigns"), a.adminPage(pm(a.ViewCampaigns, auth.PermCampaignsGetAll, auth.PermCampaignsGet)))
		g.GET(path.Join(uriAdmin, "/campaigns/media"), a.adminPage(pm(a.ViewMedia, auth.PermMediaGet)))
		g.GET(path.Join(uriAdmin, "/campaigns/media/fragment"), a.adminPage(pm(a.ViewMediaFragment, auth.PermMediaGet)))
		g.GET(path.Join(uriAdmin, "/campaigns/analytics"), a.adminPage(pm(a.ViewCampaignAnalytics, auth.PermCampaignsGetAnalytics)))
		g.GET(path.Join(uriAdmin, "/campaigns/new"), a.adminPage(pm(a.ViewNewCampaign, auth.PermCampaignsManageAll, auth.PermCampaignsManage)))
		g.GET(path.Join(uriAdmin, "/campaigns/:id"), a.adminPage(pm(hasID(a.ViewCampaign), auth.PermCampaignsGetAll, auth.PermCampaignsGet)))
		g.GET(path.Join(uriAdmin, "/campaigns/:id/:tab"), a.adminPage(pm(hasID(a.ViewCampaign), auth.PermCampaignsGetAll, auth.PermCampaignsGet)))
		g.GET(path.Join(uriAdmin, "/templates"), a.adminPage(pm(a.ViewTemplates, auth.PermTemplatesGet)))
		g.GET(path.Join(uriAdmin, "/templates/new"), a.adminPage(pm(a.ViewNewTemplate, auth.PermTemplatesManage)))
		g.GET(path.Join(uriAdmin, "/templates/:id"), a.adminPage(pm(hasID(a.ViewTemplate), auth.PermTemplatesGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/import"), a.adminPage(pm(a.ViewImport, auth.PermSubscribersImport)))
		g.GET(path.Join(uriAdmin, "/subscribers/bounces"), a.adminPage(pm(a.ViewBounces, auth.PermBouncesGet)))
		g.GET(path.Join(uriAdmin, "/subscribers"), a.adminPage(pm(a.ViewSubscribers, auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.POST(path.Join(uriAdmin, "/subscribers"), a.adminPage(pm(a.ViewSubscribers, auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/lists/:id"), a.adminPage(pm(hasID(a.ViewSubscribers), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.POST(path.Join(uriAdmin, "/subscribers/lists/:id"), a.adminPage(pm(hasID(a.ViewSubscribers), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/new"), a.adminPage(pm(a.ViewNewSubscriber, auth.PermSubscribersManage)))
		g.GET(path.Join(uriAdmin, "/subscribers/:id/edit"), a.adminPage(pm(hasID(a.ViewSubscriber), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/:id"), a.adminPage(pm(hasID(a.ViewSubscriberInfo), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/:id/lists"), a.adminPage(pm(hasID(a.ViewSubscriber), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))
		g.GET(path.Join(uriAdmin, "/subscribers/:id/bounces"), a.adminPage(pm(pm(hasID(a.ViewSubscriberBounces), auth.PermBouncesGet), auth.PermSubscribersGetAll, auth.PermSubscribersGet)))

		// Users and roles. Static segments are registered before the :id param routes.
		g.GET(path.Join(uriAdmin, "/users"), a.adminPage(pm(a.ViewUsers, auth.PermUsersGet)))
		g.GET(path.Join(uriAdmin, "/users/new"), a.adminPage(pm(a.ViewUser, auth.PermUsersManage)))
		g.GET(path.Join(uriAdmin, "/users/roles"), a.adminPage(pm(a.ViewUserRoles, auth.PermRolesGet)))
		g.GET(path.Join(uriAdmin, "/users/roles/new"), a.adminPage(pm(a.ViewUserRole, auth.PermRolesManage)))
		g.GET(path.Join(uriAdmin, "/users/roles/lists"), a.adminPage(pm(a.ViewListRoles, auth.PermRolesGet)))
		g.GET(path.Join(uriAdmin, "/users/roles/lists/new"), a.adminPage(pm(a.ViewListRole, auth.PermRolesManage)))
		g.GET(path.Join(uriAdmin, "/users/roles/lists/:id"), a.adminPage(pm(hasID(a.ViewListRole), auth.PermRolesGet)))
		g.GET(path.Join(uriAdmin, "/users/roles/:id"), a.adminPage(pm(hasID(a.ViewUserRole), auth.PermRolesGet)))
		g.GET(path.Join(uriAdmin, "/users/:id"), a.adminPage(pm(hasID(a.ViewUser), auth.PermUsersGet)))
		g.GET(path.Join(uriAdmin, "/user/profile"), a.adminPage(a.ViewProfile))

		// Settings.
		g.GET(path.Join(uriAdmin, "/settings"), a.adminPage(pm(a.ViewSettings, auth.PermSettingsGet)))
		g.GET(path.Join(uriAdmin, "/settings/logs"), a.adminPage(pm(a.ViewLogs, auth.PermSettingsGet)))
		g.GET(path.Join(uriAdmin, "/settings/maintenance"), a.adminPage(pm(a.ViewMaintenance, auth.PermSettingsMaintain)))
	}

	// =================================================================
	// Authenticated /api/* handlers.
	{
		var (
			// Permission check middleware.
			pm = a.auth.Perm

			// Attach a middleware to the group that checks for auth.
			g = e.Group("", a.auth.Middleware, func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
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
		g.GET("/api/dashboard/campaigns", a.GetDashboardCampaigns)

		g.GET("/api/settings", pm(a.GetSettings, auth.PermSettingsGet))
		g.PUT("/api/settings", pm(a.UpdateSettings, auth.PermSettingsManage))
		g.PUT("/api/settings/:key", pm(a.UpdateSettingsByKey, auth.PermSettingsManage))
		g.POST("/api/settings/smtp/test", pm(a.TestSMTPSettings, auth.PermSettingsManage))
		g.POST("/api/admin/reload", pm(a.ReloadApp, auth.PermSettingsManage))
		g.GET("/api/logs", pm(a.GetLogs, auth.PermSettingsGet))
		g.GET("/api/events", pm(a.EventStream, auth.PermSettingsGet))
		g.GET("/api/about", a.GetAboutInfo)

		g.GET("/api/subscribers", pm(a.QuerySubscribers, auth.PermSubscribersGetAll, auth.PermSubscribersGet))
		g.GET("/api/subscribers/:id", pm(hasID(a.GetSubscriber), auth.PermSubscribersGetAll, auth.PermSubscribersGet))
		g.GET("/api/subscribers/:id/activity", pm(hasID(a.GetSubscriberActivity), auth.PermSubscribersGetAll, auth.PermSubscribersGet))
		g.DELETE("/api/subscribers/:id/activity/:type", pm(hasID(a.DeleteSubscriberActivity), auth.PermSubscribersManage))
		g.GET("/api/subscribers/:id/export", pm(hasID(a.ExportSubscriberData), auth.PermSubscribersGetAll, auth.PermSubscribersGet))
		g.GET("/api/subscribers/:id/bounces", pm(hasID(a.GetSubscriberBounces), auth.PermBouncesGet))
		g.DELETE("/api/subscribers/:id/bounces", pm(hasID(a.DeleteSubscriberBounces), auth.PermBouncesManage))
		g.POST("/api/subscribers", pm(a.CreateSubscriber, auth.PermSubscribersManage))
		g.PUT("/api/subscribers/:id", pm(hasID(a.UpdateSubscriber), auth.PermSubscribersManage))
		g.PATCH("/api/subscribers/:id", pm(hasID(a.PatchSubscriber), auth.PermSubscribersManage))
		g.POST("/api/subscribers/:id/optin", pm(hasID(a.SubscriberSendOptin), auth.PermSubscribersManage))
		g.PUT("/api/subscribers/blocklist", pm(a.BlocklistSubscribers, auth.PermSubscribersManage))
		g.PUT("/api/subscribers/:id/blocklist", pm(hasID(a.BlocklistSubscriber), auth.PermSubscribersManage))
		g.PUT("/api/subscribers/lists/:id", pm(a.ManageSubscriberLists, auth.PermSubscribersManage))
		g.PUT("/api/subscribers/lists", pm(a.ManageSubscriberLists, auth.PermSubscribersManage))
		g.DELETE("/api/subscribers/:id", pm(hasID(a.DeleteSubscriber), auth.PermSubscribersManage))
		g.DELETE("/api/subscribers", pm(a.DeleteSubscribers, auth.PermSubscribersManage))

		g.GET("/api/bounces", pm(a.GetBounces, auth.PermBouncesGet))
		g.PUT("/api/bounces/blocklist", pm(a.BlocklistBouncedSubscribers, auth.PermBouncesManage))
		g.GET("/api/bounces/:id", pm(hasID(a.GetBounce), auth.PermBouncesGet))
		g.DELETE("/api/bounces", pm(a.DeleteBounces, auth.PermBouncesManage))
		g.DELETE("/api/bounces/:id", pm(hasID(a.DeleteBounce), auth.PermBouncesManage))

		// Subscriber operations based on arbitrary SQL queries.
		// These aren't very REST-like.
		g.POST("/api/subscribers/query/delete", pm(a.DeleteSubscribersByQuery, auth.PermSubscribersManage))
		g.PUT("/api/subscribers/query/blocklist", pm(a.BlocklistSubscribersByQuery, auth.PermSubscribersManage))
		g.PUT("/api/subscribers/query/lists", pm(a.ManageSubscriberListsByQuery, auth.PermSubscribersManage))
		g.GET("/api/subscribers/export",
			pm(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(a.ExportSubscribers), auth.PermSubscribersGetAll, auth.PermSubscribersGet))

		g.GET("/api/import/subscribers", pm(a.GetImportSubscribers, auth.PermSubscribersImport))
		g.GET("/api/import/subscribers/logs", pm(a.GetImportSubscriberStats, auth.PermSubscribersImport))
		g.POST("/api/import/subscribers", pm(a.ImportSubscribers, auth.PermSubscribersImport))
		g.DELETE("/api/import/subscribers", pm(a.StopImportSubscribers, auth.PermSubscribersImport))

		// Individual list permissions are applied directly within handleGetLists.
		g.GET("/api/lists", a.GetLists)
		g.GET("/api/lists/analytics/:type", a.GetListAnalytics)
		g.GET("/api/lists/:id", hasID(a.GetList))
		g.POST("/api/lists", pm(a.CreateList, auth.PermListManageAll))
		g.PUT("/api/lists/:id", hasID(a.UpdateList))
		g.DELETE("/api/lists", a.DeleteLists)
		g.DELETE("/api/lists/:id", hasID(a.DeleteList))

		g.GET("/api/campaigns", pm(a.GetCampaigns, auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.GET("/api/campaigns/running/stats", pm(a.GetRunningCampaignStats, auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.GET("/api/campaigns/:id", pm(hasID(a.GetCampaign), auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.GET("/api/campaigns/analytics/:type", pm(a.GetCampaignViewAnalytics, auth.PermCampaignsGetAnalytics))
		g.GET("/api/campaigns/:id/preview", pm(hasID(a.PreviewCampaign), auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.POST("/api/campaigns/:id/preview/archive", pm(hasID(a.PreviewCampaignArchive), auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.POST("/api/campaigns/:id/preview", pm(hasID(a.PreviewCampaign), auth.PermCampaignsGetAll, auth.PermCampaignsGet))
		g.POST("/api/campaigns/:id/content", pm(hasID(a.CampaignContent), auth.PermCampaignsManageAll, auth.PermCampaignsManage))
		g.POST("/api/campaigns/:id/text", pm(hasID(a.PreviewCampaign), auth.PermCampaignsGet))
		g.POST("/api/campaigns/:id/test", pm(hasID(a.TestCampaign), auth.PermCampaignsManageAll, auth.PermCampaignsManage))
		g.POST("/api/campaigns", pm(a.CreateCampaign, auth.PermCampaignsManageAll, auth.PermCampaignsManage))
		g.PUT("/api/campaigns/:id", pm(hasID(a.UpdateCampaign), auth.PermCampaignsManageAll, auth.PermCampaignsManage))
		g.PUT("/api/campaigns/:id/status", pm(hasID(a.UpdateCampaignStatus), auth.PermCampaignsSend))
		g.PUT("/api/campaigns/:id/archive", pm(hasID(a.UpdateCampaignArchive), auth.PermCampaignsManageAll, auth.PermCampaignsManage))
		g.DELETE("/api/campaigns", pm(a.DeleteCampaigns, auth.PermCampaignsManage, auth.PermCampaignsManageAll))
		g.DELETE("/api/campaigns/:id", pm(hasID(a.DeleteCampaign), auth.PermCampaignsManageAll, auth.PermCampaignsManage))

		g.GET("/api/media", pm(a.GetAllMedia, auth.PermMediaGet))
		g.GET("/api/media/:id", pm(hasID(a.GetMedia), auth.PermMediaGet))
		g.POST("/api/media", pm(a.UploadMedia, auth.PermMediaManage))
		g.DELETE("/api/media/:id", pm(hasID(a.DeleteMedia), auth.PermMediaManage))

		g.GET("/api/templates", pm(a.GetTemplates, auth.PermTemplatesGet))
		g.GET("/api/templates/:id", pm(hasID(a.GetTemplate), auth.PermTemplatesGet))
		g.GET("/api/templates/:id/preview", pm(hasID(a.PreviewTemplate), auth.PermTemplatesGet))
		g.POST("/api/templates/preview", pm(a.PreviewTemplateBody, auth.PermTemplatesGet))
		g.POST("/api/templates", pm(a.CreateTemplate, auth.PermTemplatesManage))
		g.PUT("/api/templates/:id", pm(hasID(a.UpdateTemplate), auth.PermTemplatesManage))
		g.PUT("/api/templates/:id/default", pm(hasID(a.TemplateSetDefault), auth.PermTemplatesManage))
		g.DELETE("/api/templates/:id", pm(hasID(a.DeleteTemplate), auth.PermTemplatesManage))

		g.DELETE("/api/maintenance/subscribers/:type", pm(a.GCSubscribers, auth.PermSettingsMaintain))
		g.DELETE("/api/maintenance/analytics/:type", pm(a.GCCampaignAnalytics, auth.PermSettingsMaintain))
		g.GET("/api/maintenance/analytics/:type/export", pm(a.ExportCampaignAnalytics, auth.PermSettingsMaintain))
		g.DELETE("/api/maintenance/subscriptions/unconfirmed", pm(a.GCSubscriptions, auth.PermSettingsMaintain))

		g.POST("/api/tx", pm(a.SendTxMessage, auth.PermTxSend))

		g.GET("/api/profile", a.GetUserProfile)
		g.PUT("/api/profile", a.UpdateUserProfile)
		g.GET("/api/users", pm(a.GetUsers, auth.PermUsersGet))
		g.GET("/api/users/:id", pm(hasID(a.GetUser), auth.PermUsersGet))
		g.POST("/api/users", pm(a.CreateUser, auth.PermUsersManage))
		g.PUT("/api/users/:id", pm(hasID(a.UpdateUser), auth.PermUsersManage))
		g.DELETE("/api/users", pm(a.DeleteUsers, auth.PermUsersManage))
		g.DELETE("/api/users/:id", pm(hasID(a.DeleteUser), auth.PermUsersManage))
		g.POST("/api/logout", a.Logout)

		// TOTP 2FA endpoints
		g.GET("/api/users/:id/twofa/totp", hasID(a.GenerateTOTPQR))
		g.PUT("/api/users/:id/twofa", hasID(a.EnableTOTP))
		g.DELETE("/api/users/:id/twofa", hasID(a.DisableTOTP))

		g.GET("/api/roles/users", pm(a.GetUserRoles, auth.PermRolesGet))
		g.GET("/api/roles/lists", pm(a.GeListRoles, auth.PermRolesGet))
		g.POST("/api/roles/users", pm(a.CreateUserRole, auth.PermRolesManage))
		g.POST("/api/roles/lists", pm(a.CreateListRole, auth.PermRolesManage))
		g.PUT("/api/roles/users/:id", pm(hasID(a.UpdateUserRole), auth.PermRolesManage))
		g.PUT("/api/roles/lists/:id", pm(hasID(a.UpdateListRole), auth.PermRolesManage))
		g.DELETE("/api/roles/:id", pm(hasID(a.DeleteRole), auth.PermRolesManage))

		if a.cfg.BounceWebhooksEnabled {
			// Private authenticated bounce endpoint.
			g.POST("/webhooks/bounce", pm(a.BounceWebhook, auth.PermWebhooksPostBounce))
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
		g.GET("/", func(c *echo.Context) error {
			return c.Render(http.StatusOK, "home", publicTpl{Title: "listmonk"})
		})

		// Public admin endpoints (login page, OIDC endpoints, password reset).
		g.GET(path.Join(uriAdmin, "/login"), a.adminPage(a.LoginPage))
		g.POST(path.Join(uriAdmin, "/login"), a.adminPage(a.LoginPage))
		g.GET(path.Join(uriAdmin, "/login/twofa"), a.adminPage(a.TwofaPage))
		g.POST(path.Join(uriAdmin, "/login/twofa"), a.adminPage(a.TwofaPage))
		g.GET(path.Join(uriAdmin, "/forgot"), a.adminPage(a.ForgotPage))
		g.POST(path.Join(uriAdmin, "/forgot"), a.adminPage(a.ForgotPage))
		g.GET(path.Join(uriAdmin, "/reset"), a.adminPage(a.ResetPage))
		g.POST(path.Join(uriAdmin, "/reset"), a.adminPage(a.ResetPage))

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
		g.GET("/robots.txt", a.RobotsTxt)

		// 404 pages.
		g.RouteNotFound("/*", func(c *echo.Context) error {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl("404 - "+a.i18n.T("globals.terms.notFound"), "", ""))
		})
		g.RouteNotFound("/api/*", func(c *echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "404 unknown endpoint")
		})
		g.RouteNotFound("/admin/*", a.adminPage(func(c *echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "404 page not found")
		}))
	}
}

// HealthCheck is a healthcheck endpoint that returns a 200 response.
func (a *App) HealthCheck(c *echo.Context) error {
	return c.JSON(http.StatusOK, okResp{true})
}

// RobotsTxt serves the robots.txt file from the static filesystem.
func (a *App) RobotsTxt(c *echo.Context) error {
	b, err := a.fs.Read("/public/static/robots.txt")
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "robots.txt not found")
	}
	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", b)
}

// serveCustomAppearance serves the given custom CSS/JS appearance blob
// meant for customizing public and admin pages from the admin settings UI.
func serveCustomAppearance(name string) echo.HandlerFunc {
	return func(c *echo.Context) error {
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
	return func(c *echo.Context) error {
		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("globals.terms.error"), "",
					a.i18n.T("globals.messages.invalidUUID")))
			}
		}
		return next(c)
	}
}

// hasID middleware validates the :id param in the URL and sets its int value in the context.
func hasID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
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
	return func(c *echo.Context) error {
		subUUID := c.Param("subUUID")

		if _, err := a.core.GetSubscriber(0, subUUID, ""); err != nil {
			if er, ok := err.(*echo.HTTPError); ok && er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(a.i18n.T("globals.terms.notFound"), "", er.Message))
			}

			a.log.Printf("error checking subscriber existence: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(a.i18n.T("globals.terms.error"), "", a.i18n.T("public.errorProcessingRequest")))
		}

		return next(c)
	}
}

// handleHTTPErr logs errors and delegates response formatting to Echo.
func (a *App) handleHTTPErr(c *echo.Context, err error) {
	if echo.StatusCode(err) == 0 {
		a.log.Println(err.Error())
	}

	echo.DefaultHTTPErrorHandler(false)(c, err)
}

// noIndex adds the HTTP header requesting robots to not crawl the page.
func noIndex(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.Response().Header().Set("X-Robots-Tag", "noindex")
		return next(c)
	}
}

// getID returns the :id param from the URL parsed and stored as an int by the hasID middleware.
func getID(c *echo.Context) int {
	return c.Get("id").(int)
}

// trustedURLsToCORSOrigins takes a list of trusted URLs and returns a list of
// unique origin domains to be used in CORS middleware configuration, including '*' if it exists.
func trustedURLsToCORSOrigins(urls []string) []string {
	mp := map[string]struct{}{}

	for _, s := range urls {
		if s == "*" {
			mp[s] = struct{}{}
		}

		u, err := url.ParseRequestURI(s)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		s = u.Scheme + "://" + u.Host
		mp[s] = struct{}{}
	}

	out := make([]string, 0, len(mp))
	for u := range mp {
		out = append(out, u)
	}

	return out
}

// renderSandboxedHTML executes the HTML rendering handler with CSP sandboxing headers (for iframes).
func renderSandboxedHTML(c *echo.Context, body string) error {
	c.Response().Header().Add("Content-Security-Policy", "sandbox allow-scripts allow-popups allow-popups-to-escape-sandbox")
	return c.HTML(http.StatusOK, body)
}

// skipAdminTrailingSlash excludes assets and non-admin URLs from canonical redirects.
func skipAdminTrailingSlash(c *echo.Context) bool {
	p := strings.TrimRight(c.Request().URL.Path, "/")
	return (p != uriAdmin && !strings.HasPrefix(p, uriAdmin+"/")) ||
		p == uriAdmin+"/static" || strings.HasPrefix(p, uriAdmin+"/static/") ||
		p == uriAdmin+"/custom.css" || p == uriAdmin+"/custom.js"
}
