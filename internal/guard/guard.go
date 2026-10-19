package guard

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/labstack/echo/v5"
	guardecho "github.com/rennf93/echo-guard/v2"
	guardcore "github.com/rennf93/guard-core-go/v4/guardcore"
)

// section is the config.toml section (and env prefix) of the optional security middleware.
const section = "security"

// Policy names, used as the guard's route IDs.
const (
	policyAdmin        = "admin"
	policyLogin        = "login"
	policySubscription = "subscription"
	policyTracking     = "tracking"
	policyDefault      = "default"
)

// New returns the echo-guard security middleware, or nil if [security] enabled is false.
func New(ko *koanf.Koanf, lo *log.Logger) (echo.MiddlewareFunc, error) {
	k := func(key string) string { return section + "." + key }
	if !ko.Bool(k("enabled")) {
		return nil, nil
	}

	// num returns the int config value or def if it is unset (or 0).
	num := func(key string, def int) int {
		if v := ko.Int(k(key)); v > 0 {
			return v
		}
		return def
	}

	cfg := guardcore.DefaultSecurityConfig()
	cfg.EnableRedis = false
	cfg.TrustedProxies = ko.Strings(k("trusted_proxies"))
	cfg.Whitelist = ko.Strings(k("allowlist"))
	cfg.Blacklist = ko.Strings(k("blocklist"))
	cfg.BlockedCountries = ko.Strings(k("blocked_countries"))
	cfg.WhitelistCountries = ko.Strings(k("allowed_countries"))
	cfg.GeoIPDBPath = ko.String(k("geoip_db_path"))
	cfg.IPInfoToken = ko.String(k("ipinfo_token"))
	cfg.BlockedUserAgents = ko.Strings(k("blocked_user_agents"))
	cfg.EmergencyMode = ko.Bool(k("emergency_mode"))
	cfg.EmergencyWhitelist = ko.Strings(k("emergency_allowlist"))
	cfg.PassiveMode = ko.Bool(k("passive_mode"))
	cfg.AutoBanThreshold = num("auto_ban_threshold", 5)
	cfg.AutoBanDuration = num("auto_ban_duration", 3600)
	cfg.ExcludePaths = []string{"/public/static", "/admin/static"}

	// Log every block (and every would-be block in passive mode) to listmonk's log.
	cfg.OnBlock = func(_ guardcore.Request, p map[string]any) {
		verb := "blocked"
		if passive, _ := p["passive_mode"].(bool); passive {
			verb = "would block (passive mode)"
		}
		lo.Printf("security: %s %v %v from %v: %v: %v %v", verb, p["method"], p["path"], p["client_ip"],
			p["check_name"], p["reason"], p["trigger_info"])
	}

	// Engine gap: a localhost or LAN Referer is flagged as ssrf, which breaks the admin UI there.
	cfg.EnabledDetectionCategories = slices.DeleteFunc(slices.Clone(guardcore.AllDetectionCategories),
		func(c string) bool { return c == "ssrf" })

	// Route limits (login, subscription) only run while rate limiting is on. With no global
	// limit set, the global tier is made unreachable instead.
	cfg.EnableRateLimiting = true
	cfg.EnableRateLimitAutoBan = true
	cfg.RateLimit = num("rate_limit", math.MaxInt32)
	cfg.RateLimitWindow = num("rate_limit_window", 60)

	// The engine's default headers include Cross-Origin-Resource-Policy: same-origin and
	// Cross-Origin-Embedder-Policy: require-corp, which stop mail clients from loading the
	// tracking pixel and images served from listmonk's uploads.
	cfg.SecurityHeaders = nil

	scanner404 := guardcore.BehaviorRuleConfig{
		RuleType:  "return_pattern",
		Pattern:   "status:404",
		Threshold: num("scanner_404_limit", 30),
		Window:    num("scanner_404_window", 300),
		Action:    "ban",
	}
	if err := guardcore.ValidateBehaviorRuleConfig(&scanner404); err != nil {
		return nil, fmt.Errorf("invalid %s config: %v", section, err)
	}
	// Set on every policy except tracking: a deleted campaign's pixel and links 404 for
	// the shared mail proxies too.
	scannerRules := []guardcore.BehaviorRuleConfig{scanner404}

	engine, err := guardcore.NewEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("error initializing %s: %v", section, err)
	}
	if err := engine.Initialize(); err != nil {
		return nil, fmt.Errorf("error initializing %s: %v", section, err)
	}

	// Mail clients fetch pixels and links through shared proxies (Gmail's image proxy,
	// Microsoft Safe Links), so tracking has no cloud blocking and the default action is log, not ban.
	var trackingRules []guardcore.BehaviorRuleConfig
	if limit := ko.Int(k("tracking_usage_limit")); limit > 0 {
		rule := guardcore.BehaviorRuleConfig{
			RuleType:  "usage",
			Threshold: limit,
			Window:    num("tracking_usage_window", 3600),
			Action:    ko.String(k("tracking_usage_action")),
		}
		if rule.Action == "" {
			rule.Action = "log"
		}
		if err := guardcore.ValidateBehaviorRuleConfig(&rule); err != nil {
			return nil, fmt.Errorf("invalid %s tracking rule: %v", section, err)
		}
		trackingRules = append(trackingRules, rule)
	}

	adminAllow := ko.Strings(k("admin_allowlist"))
	cloud := ko.Strings(k("block_cloud_providers"))

	engine.Routes.Register(policyAdmin, func(rc *guardcore.RouteConfig) {
		rc.IPWhitelist = adminAllow
		rc.BlockCloudProviders = cloud

		// The subscriber search takes raw SQL in `query`, and admin bodies (campaigns, templates)
		// are full of HTML and JS.
		rc.ExcludedDetectionParams = map[string]bool{"query": true}
		off := false
		rc.DetectionScanBody = &off
		rc.BehaviorRules = scannerRules
	})

	engine.Routes.Register(policyLogin, func(rc *guardcore.RouteConfig) {
		rc.IPWhitelist = adminAllow
		rc.BlockCloudProviders = cloud
		rc.RateLimit = num("login_rate_limit", 10)
		rc.RateLimitWindow = num("login_rate_window", 60)
		rc.BehaviorRules = scannerRules
	})

	honeypot := ko.String(k("honeypot_field"))
	if !ko.Exists(k("honeypot_field")) {
		honeypot = "nonce"
	}
	engine.Routes.Register(policySubscription, func(rc *guardcore.RouteConfig) {
		rc.BlockCloudProviders = cloud
		rc.RateLimit = num("subscription_rate_limit", 10)
		rc.RateLimitWindow = num("subscription_rate_window", 60)
		rc.BehaviorRules = scannerRules
		if honeypot != "" {
			rc.CustomValidators = []func(guardcore.Request) *guardcore.Response{honeypotValidator(engine, honeypot, lo)}
		}
	})

	// The 404 rule counts per route ID, so unmapped paths share one ID to be counted together.
	engine.Routes.Register(policyDefault, func(rc *guardcore.RouteConfig) {
		rc.BehaviorRules = scannerRules
	})

	engine.Routes.Register(policyTracking, func(rc *guardcore.RouteConfig) {
		rc.BehaviorRules = trackingRules
	})

	guard, err := guardecho.New(engine, guardecho.WithLogger(lo))
	if err != nil {
		return nil, fmt.Errorf("error initializing %s: %v", section, err)
	}

	// The route ID middleware has to run before the guard.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		g := guard(next)
		return func(c *echo.Context) error {
			id := guardPolicy(c.Request().Method, c.Request().URL.Path)
			c.SetRequest(c.Request().WithContext(guardecho.WithRouteID(c.Request().Context(), id)))
			return g(c)
		}
	}, nil
}

// guardPolicy maps a request to a policy name. policyDefault means the global settings only.
// It reads the URL path as Echo's RouteNotFound routes match unknown /admin/* paths.
func guardPolicy(method, p string) string {
	switch {
	case method == http.MethodPost && slices.Contains([]string{
		"/admin/login", "/admin/login/twofa", "/admin/forgot", "/admin/reset", "/auth/oidc"}, p):
		return policyLogin
	case method == http.MethodPost && (p == "/subscription/form" || p == "/api/public/subscription"):
		return policySubscription
	case method == http.MethodGet && (strings.HasPrefix(p, "/link/") ||
		strings.HasPrefix(p, "/campaign/") && strings.HasSuffix(p, "/px.png")):
		return policyTracking
	case strings.HasPrefix(p, "/api/public/"), strings.HasPrefix(p, "/admin/static/"):
		return policyDefault
	case p == "/admin", strings.HasPrefix(p, "/admin/"), strings.HasPrefix(p, "/api/"):
		return policyAdmin
	}
	return policyDefault
}

// honeypotValidator bans the client on the first request that fills the hidden form field.
func honeypotValidator(engine *guardcore.Engine, field string, lo *log.Logger) func(guardcore.Request) *guardcore.Response {
	return func(req guardcore.Request) *guardcore.Response {
		// The subscription policy also covers POST /api/public/subscription, which has no honeypot.
		if req.URLPath() != "/subscription/form" {
			return nil
		}
		body, err := req.Body()
		if err != nil {
			return nil
		}
		ct, _ := req.Headers().Get("Content-Type")
		r := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Type": {ct}}, Body: io.NopCloser(bytes.NewReader(body))}

		// PostFormValue parses urlencoded and multipart bodies; an unparseable body gives "".
		if r.PostFormValue(field) == "" {
			return nil
		}

		cfg := engine.Config
		if ip := req.State().ClientIP; ip != "" && !cfg.PassiveMode && cfg.EnableIPBanning {
			if _, err := engine.Ban.Ban(ip, cfg.AutoBanDuration, "honeypot"); err != nil {
				lo.Printf("security: error banning %s: %v", ip, err)
			}
		}
		return engine.CreateErrorResponse(http.StatusForbidden, "Forbidden")
	}
}
