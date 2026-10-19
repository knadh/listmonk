package guard

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/labstack/echo/v5"
)

const (
	testLink  = "/link/11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333"
	testPixel = "/campaign/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333/px.png"
)

// newGuardApp returns an Echo instance with the guard on (if enabled) and stub
// handlers on the same paths as listmonk.
var testLog = log.New(io.Discard, "", 0)

func newGuardApp(t *testing.T, conf map[string]any) *echo.Echo {
	t.Helper()

	k := koanf.New(".")
	if err := k.Load(confmap.Provider(conf, "."), nil); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	g, err := New(k, testLog)
	if err != nil {
		t.Fatal(err)
	}
	if g != nil {
		e.Use(g)
	}

	ok := func(c *echo.Context) error { return c.String(http.StatusOK, "ok") }
	for _, p := range []string{"/admin", "/admin/lists", "/admin/login", "/api/lists", "/api/subscribers",
		"/api/subscribers/export", "/api/public/lists", "/subscription/form", "/public/static/a.css",
		"/admin/static/a.css", testLink, testPixel} {
		e.GET(p, ok)
		e.POST(p, ok)
	}
	e.POST("/auth/oidc", ok)
	e.POST("/api/public/subscription", ok)
	e.POST("/api/subscribers/query/delete", ok)
	e.RouteNotFound("/*", func(c *echo.Context) error { return echo.ErrNotFound })
	return e
}

type guardReq struct {
	method, target, remote, body, ctype string
	header                              map[string]string
}

func (r guardReq) do(e *echo.Echo) int {
	if r.method == "" {
		r.method = http.MethodGet
	}
	if r.remote == "" {
		r.remote = "192.0.2.1:1234"
	}
	req := httptest.NewRequest(r.method, r.target, strings.NewReader(r.body))
	req.RemoteAddr = r.remote
	if r.ctype != "" {
		req.Header.Set("Content-Type", r.ctype)
	}
	for k, v := range r.header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code
}

func get(e *echo.Echo, target, remote string) int {
	return guardReq{target: target, remote: remote}.do(e)
}

func postForm(e *echo.Echo, target, remote, body string) int {
	return guardReq{method: http.MethodPost, target: target, remote: remote, body: body,
		ctype: "application/x-www-form-urlencoded"}.do(e)
}

func want(t *testing.T, what string, got, exp int) {
	t.Helper()
	if got != exp {
		t.Errorf("%s: got %d, want %d", what, got, exp)
	}
}

func TestGuardDisabledRegistersNothing(t *testing.T) {
	for _, conf := range []map[string]any{{}, {"security.enabled": false}} {
		k := koanf.New(".")
		if err := k.Load(confmap.Provider(conf, "."), nil); err != nil {
			t.Fatal(err)
		}
		if g, err := New(k, testLog); g != nil || err != nil {
			t.Fatalf("New returned (%v, %v) while disabled", g != nil, err)
		}
	}

	// Without the middleware, an attack payload reaches the handler.
	e := newGuardApp(t, map[string]any{})
	want(t, "payload with guard off", get(e, "/api/public/lists?id="+url.QueryEscape("1' OR '1'='1' --"), ""), http.StatusOK)
}

func TestGuardBlocksAttackPayloads(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true})

	want(t, "clean request", get(e, "/api/public/lists?id=1", ""), http.StatusOK)

	// A different client per case, so an auto-ban does not mask the next one.
	for _, c := range []struct{ name, target, remote string }{
		{"sqli query", "/api/public/lists?id=" + url.QueryEscape("1' OR '1'='1' --"), "198.51.100.1:1"},
		{"xss query", "/api/public/lists?q=" + url.QueryEscape("<script>alert(1)</script>"), "198.51.100.2:1"},
		{"traversal path", "/public/static/../../../../etc/passwd", "198.51.100.3:1"},
	} {
		want(t, c.name, get(e, c.target, c.remote), http.StatusBadRequest)
	}

	want(t, "sqli header", guardReq{target: "/api/public/lists", remote: "203.0.113.5:1",
		header: map[string]string{"X-Custom": "' UNION SELECT password FROM users--"}}.do(e), http.StatusBadRequest)
}

func TestGuardHoneypot(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true})
	bot, human := "192.0.2.10:1", "192.0.2.11:1"

	want(t, "empty nonce", postForm(e, "/subscription/form", human, "email=a%40b.com&nonce="), http.StatusOK)
	want(t, "no nonce", postForm(e, "/subscription/form", human, "email=a%40b.com"), http.StatusOK)

	want(t, "filled nonce", postForm(e, "/subscription/form", bot, "email=a%40b.com&nonce=x"), http.StatusForbidden)
	want(t, "next GET from the banned client", get(e, "/public/static/a.css", bot), http.StatusForbidden)
	want(t, "next GET from another client", get(e, "/api/public/lists", human), http.StatusOK)

	// Multipart bodies are parsed too.
	mp := "--b\r\nContent-Disposition: form-data; name=\"nonce\"\r\n\r\nx\r\n--b--\r\n"
	want(t, "multipart nonce", guardReq{method: http.MethodPost, target: "/subscription/form",
		remote: "192.0.2.12:1", body: mp, ctype: "multipart/form-data; boundary=b"}.do(e), http.StatusForbidden)

	// An unparseable body is not a hit.
	want(t, "broken multipart", guardReq{method: http.MethodPost, target: "/subscription/form",
		remote: "192.0.2.13:1", body: "garbage nonce=x", ctype: "multipart/form-data; boundary=b"}.do(e), http.StatusOK)
}

func TestGuardHoneypotOtherRoutes(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true})
	c := "192.0.2.20:1"

	// The OIDC login legitimately posts a field named nonce.
	want(t, "oidc nonce", postForm(e, "/auth/oidc", c, "nonce=abc"), http.StatusOK)
	want(t, "oidc nonce did not ban", get(e, "/api/public/lists", c), http.StatusOK)

	want(t, "api subscription nonce", postForm(e, "/api/public/subscription", c, "nonce=abc"), http.StatusOK)
	want(t, "api subscription nonce did not ban", get(e, "/api/public/lists", c), http.StatusOK)
}

func TestGuardHoneypotDisabledByEmptyField(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true, "security.honeypot_field": ""})
	want(t, "nonce with honeypot off", postForm(e, "/subscription/form", "192.0.2.30:1", "nonce=x"), http.StatusOK)
}

func TestGuardPassiveModeDoesNotBlockOrBan(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true, "security.passive_mode": true})
	c := "192.0.2.40:1"

	want(t, "honeypot hit", postForm(e, "/subscription/form", c, "nonce=x"), http.StatusOK)
	want(t, "next GET", get(e, "/api/public/lists", c), http.StatusOK)
	want(t, "sqli", get(e, "/api/public/lists?id="+url.QueryEscape("1' OR '1'='1' --"), c), http.StatusOK)
}

func TestGuardAdminAllowlist(t *testing.T) {
	e := newGuardApp(t, map[string]any{
		"security.enabled":         true,
		"security.admin_allowlist": []string{"10.0.0.0/8"},
	})
	out, in := "192.0.2.50:1", "10.1.2.3:1"

	for _, p := range []string{"/admin", "/admin/lists", "/api/lists", "/api/subscribers"} {
		want(t, p+" outside", get(e, p, out), http.StatusForbidden)
		want(t, p+" inside", get(e, p, in), http.StatusOK)
	}
	want(t, "login outside", postForm(e, "/admin/login", out, "username=a"), http.StatusForbidden)
	want(t, "login inside", postForm(e, "/admin/login", in, "username=a"), http.StatusOK)

	// Public endpoints stay open.
	want(t, "subscription form", postForm(e, "/subscription/form", out, "email=a%40b.com"), http.StatusOK)
	want(t, "link", get(e, testLink, out), http.StatusOK)
	want(t, "pixel", get(e, testPixel, out), http.StatusOK)
	want(t, "public lists", get(e, "/api/public/lists", out), http.StatusOK)
	want(t, "admin static", get(e, "/admin/static/a.css", out), http.StatusOK)
	want(t, "login page outside", get(e, "/admin/login", out), http.StatusForbidden)
}

func TestGuardIgnoresSpoofedForwardedForFromUntrustedPeer(t *testing.T) {
	e := newGuardApp(t, map[string]any{
		"security.enabled":         true,
		"security.admin_allowlist": []string{"10.0.0.0/8"},
	})

	// The TCP peer is outside the allowlist, the header claims to be inside.
	want(t, "spoofed allowlist", guardReq{target: "/admin", remote: "192.0.2.60:1",
		header: map[string]string{"X-Forwarded-For": "10.1.2.3"}}.do(e), http.StatusForbidden)

	// A honeypot hit with a spoofed header bans the peer, not the victim.
	want(t, "honeypot", guardReq{method: http.MethodPost, target: "/subscription/form", remote: "192.0.2.61:1",
		body: "nonce=x", ctype: "application/x-www-form-urlencoded",
		header: map[string]string{"X-Forwarded-For": "198.51.100.7"}}.do(e), http.StatusForbidden)
	want(t, "peer is banned", get(e, "/api/public/lists", "192.0.2.61:1"), http.StatusForbidden)
	want(t, "victim is not banned", get(e, "/api/public/lists", "198.51.100.7:1"), http.StatusOK)
}

func TestGuardUsesForwardedForFromTrustedProxy(t *testing.T) {
	e := newGuardApp(t, map[string]any{
		"security.enabled":         true,
		"security.trusted_proxies": []string{"203.0.113.1"},
		"security.admin_allowlist": []string{"10.0.0.0/8"},
	})
	proxy := "203.0.113.1:1"
	xff := func(ip string) map[string]string { return map[string]string{"X-Forwarded-For": ip} }

	want(t, "forwarded client inside", guardReq{target: "/admin", remote: proxy, header: xff("10.9.9.9")}.do(e), http.StatusOK)
	want(t, "forwarded client outside", guardReq{target: "/admin", remote: proxy, header: xff("192.0.2.70")}.do(e), http.StatusForbidden)

	// The custom validator sees the forwarded client IP, so the ban lands on it.
	want(t, "honeypot", guardReq{method: http.MethodPost, target: "/subscription/form", remote: proxy, body: "nonce=x",
		ctype: "application/x-www-form-urlencoded", header: xff("198.51.100.9")}.do(e), http.StatusForbidden)
	want(t, "forwarded client is banned", guardReq{target: "/api/public/lists", remote: proxy, header: xff("198.51.100.9")}.do(e), http.StatusForbidden)
	want(t, "other forwarded client is not", guardReq{target: "/api/public/lists", remote: proxy, header: xff("198.51.100.10")}.do(e), http.StatusOK)
}

func TestGuardAdminQueryParamIsNotScanned(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true})
	expr := url.QueryEscape("subscribers.name LIKE '%a%' OR subscribers.id IN (SELECT 1 UNION SELECT 2)")

	want(t, "admin query param", get(e, "/api/subscribers?query="+expr, "192.0.2.80:1"), http.StatusOK)
	want(t, "admin export query param", get(e, "/api/subscribers/export?query="+expr, "192.0.2.80:1"), http.StatusOK)
	want(t, "admin JSON body", guardReq{method: http.MethodPost, target: "/api/subscribers/query/delete", remote: "192.0.2.80:1",
		body: `{"query":"subscribers.name LIKE '%a%' OR 1=1 --"}`, ctype: "application/json"}.do(e), http.StatusOK)

	want(t, "same value on a public route", get(e, "/api/public/lists?query="+expr, "192.0.2.81:1"), http.StatusBadRequest)

	// Other admin params are still screened.
	want(t, "other admin param", get(e, "/api/subscribers?q="+expr, "192.0.2.82:1"), http.StatusBadRequest)
}

func TestGuardLocalhostRefererIsNotSSRF(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true})

	for _, ref := range []string{"http://localhost:9000/admin", "http://127.0.0.1:9000/admin", "http://192.168.1.10:9000/admin"} {
		want(t, ref, guardReq{target: "/admin/lists", header: map[string]string{"Referer": ref}}.do(e), http.StatusOK)
	}
}

func TestGuardLoginRateLimit(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true, "security.login_rate_limit": 3})
	c := "192.0.2.90:1"

	for i := 0; i < 3; i++ {
		want(t, "login within the limit", postForm(e, "/admin/login", c, "username=a"), http.StatusOK)
	}
	want(t, "login over the limit", postForm(e, "/admin/login", c, "username=a"), http.StatusTooManyRequests)
	want(t, "other clients are unaffected", postForm(e, "/admin/login", "192.0.2.91:1", "username=a"), http.StatusOK)

	// No global limit by default: plenty of ordinary requests pass.
	for i := 0; i < 50; i++ {
		want(t, "admin request", get(e, "/admin/lists", "192.0.2.92:1"), http.StatusOK)
	}
}

func TestGuardTrackingUsageRule(t *testing.T) {
	// The default action is log, so a client behind a shared mail proxy is never blocked.
	e := newGuardApp(t, map[string]any{"security.enabled": true, "security.tracking_usage_limit": 2})
	for i := 0; i < 6; i++ {
		want(t, "pixel with log action", get(e, testPixel, "192.0.2.100:1"), http.StatusOK)
	}

	e = newGuardApp(t, map[string]any{"security.enabled": true,
		"security.tracking_usage_limit": 2, "security.tracking_usage_action": "ban"})
	c := "192.0.2.101:1"
	for i := 0; i < 3; i++ {
		get(e, testPixel, c)
	}
	want(t, "request after the ban rule tripped", get(e, "/api/public/lists", c), http.StatusForbidden)
}

func TestGuardScanner404Ban(t *testing.T) {
	e := newGuardApp(t, map[string]any{"security.enabled": true, "security.scanner_404_limit": 3})
	c := "192.0.2.110:1"

	// The rule trips on the 4th hit (count above the limit), across different paths.
	for _, p := range []string{"/login-a", "/login-b", "/login-c", "/login-d"} {
		want(t, p, get(e, p, c), http.StatusNotFound)
	}
	want(t, "after the scanner ban", get(e, "/api/public/lists", c), http.StatusForbidden)

	// 404s on link and pixel tracking (a deleted campaign) never ban the shared mail proxies.
	proxy := "192.0.2.111:1"
	for i := 0; i < 10; i++ {
		want(t, "missing pixel", get(e, "/campaign/x/y/px.png", proxy), http.StatusNotFound)
		want(t, "missing link", get(e, "/link/x/y/z", proxy), http.StatusNotFound)
	}
	want(t, "proxy after tracking 404s", get(e, testPixel, proxy), http.StatusOK)
}

func TestGuardPolicyMapping(t *testing.T) {
	for _, c := range []struct{ method, path, want string }{
		{"GET", "/admin", policyAdmin},
		{"GET", "/admin/campaigns/1", policyAdmin},
		{"GET", "/admin/unknown/page", policyAdmin},
		{"GET", "/api/lists", policyAdmin},
		{"GET", "/api/public/lists", policyDefault},
		{"GET", "/admin/login", policyAdmin},
		{"GET", "/admin/static/a.js", policyDefault},
		{"POST", "/admin/login", policyLogin},
		{"POST", "/admin/forgot", policyLogin},
		{"POST", "/auth/oidc", policyLogin},
		{"GET", "/auth/oidc", policyDefault},
		{"POST", "/subscription/form", policySubscription},
		{"POST", "/api/public/subscription", policySubscription},
		{"GET", "/subscription/form", policyDefault},
		{"GET", testLink, policyTracking},
		{"GET", testPixel, policyTracking},
		{"GET", "/campaign/22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333", policyDefault},
		{"GET", "/", policyDefault},
	} {
		if got := guardPolicy(c.method, c.path); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

func TestGuardLogsBlocks(t *testing.T) {
	var buf bytes.Buffer
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(map[string]any{"security.enabled": true}, "."), nil); err != nil {
		t.Fatal(err)
	}
	g, err := New(k, log.New(&buf, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.Use(g)
	e.GET("/api/public/lists", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

	want(t, "sqli", get(e, "/api/public/lists?id="+url.QueryEscape("1' OR '1'='1' --"), "192.0.2.120:1"), http.StatusBadRequest)
	if out := buf.String(); !strings.Contains(out, "security: blocked GET /api/public/lists from 192.0.2.120: suspicious_activity") {
		t.Errorf("block not logged: %q", out)
	}
}
