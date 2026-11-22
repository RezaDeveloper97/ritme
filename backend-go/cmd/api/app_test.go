package main

import (
	"io"
	"log/slog"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db"
)

func testDeps(origins ...string) *apihttp.Deps {
	if len(origins) == 0 {
		origins = config.DefaultCORSOrigins
	}
	return &apihttp.Deps{
		Config: &config.Config{CORS: config.CORS{AllowedOrigins: origins}},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
}

func do(t *testing.T, app *fiber.App, req *nethttp.Request) *nethttp.Response {
	t.Helper()
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// Two routes_<domain>.go files each call Register from their own init(); neither
// knows about the other and both end up mounted.
func TestRegistry_DomainsRegisterIndependently(t *testing.T) {
	reg := apihttp.NewRegistry()
	// routes_alpha.go
	reg.Register("alpha", func(r fiber.Router, _ *apihttp.Deps) {
		r.Get("/api/v1/alpha/static", func(c fiber.Ctx) error { return c.SendString("alpha-static") })
		r.Get("/api/v1/alpha/:id", func(c fiber.Ctx) error { return c.SendString("alpha-" + c.Params("id")) })
	})
	// routes_beta.go
	reg.Register("beta", func(r fiber.Router, _ *apihttp.Deps) {
		r.Post("/api/v1/beta", func(c fiber.Ctx) error { return c.Status(fiber.StatusCreated).SendString("beta") })
	})

	assert.Equal(t, []string{"alpha", "beta"}, reg.Domains())
	app := newApp(reg, testDeps())

	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{fiber.MethodGet, "/api/v1/alpha/static", 200, "alpha-static"},
		{fiber.MethodGet, "/api/v1/alpha/42", 200, "alpha-42"},
		{fiber.MethodHead, "/api/v1/alpha/static", 200, ""},
		{fiber.MethodPost, "/api/v1/beta", 201, "beta"},
	} {
		resp := do(t, app, httptest.NewRequest(tc.method, tc.path, nil))
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, tc.status, resp.StatusCode, tc.method+" "+tc.path)
		assert.Equal(t, tc.body, string(body), tc.method+" "+tc.path)
	}

	assert.PanicsWithValue(t, `http: domain "beta" registered twice`, func() {
		reg.Register("beta", func(fiber.Router, *apihttp.Deps) {})
	})
}

func TestDefaultRegistry_HasHealth(t *testing.T) {
	assert.Contains(t, apihttp.Domains(), "health")
}

func TestUp(t *testing.T) {
	app := newApp(apihttp.Default(), testDeps())

	resp := do(t, app, httptest.NewRequest(fiber.MethodGet, "/up", nil))
	assert.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Vary"), "CORS only applies to api/*")

	resp = do(t, app, httptest.NewRequest(fiber.MethodHead, "/up", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func preflight(origin, method, headers string) *nethttp.Request {
	req := httptest.NewRequest(fiber.MethodOptions, "/api/v1/auth/send-otp", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", method)
	if headers != "" {
		req.Header.Set("Access-Control-Request-Headers", headers)
	}
	return req
}

// Expected values are what Laravel 12 + fruitcake/php-cors emit for backend/config/cors.php.
func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	app := newApp(apihttp.NewRegistry(), testDeps())

	resp := do(t, app, preflight("https://web.ritme.app", "post", "authorization,content-type"))
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "https://web.ritme.app", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "authorization,content-type", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "Origin, Access-Control-Request-Method, Access-Control-Request-Headers", resp.Header.Get("Vary"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
}

func TestCORS_PreflightDisallowedOrigin(t *testing.T) {
	app := newApp(apihttp.NewRegistry(), testDeps())

	resp := do(t, app, preflight("https://evil.example", "GET", ""))
	assert.Equal(t, 204, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "Origin, Access-Control-Request-Method", resp.Header.Get("Vary"))
}

func TestCORS_ActualRequest(t *testing.T) {
	reg := apihttp.NewRegistry()
	reg.Register("x", func(r fiber.Router, _ *apihttp.Deps) {
		r.Get("/api/v1/x", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })
	})
	app := newApp(reg, testDeps())

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/x", nil)
	req.Header.Set("Origin", "https://web.ritme.app")
	resp := do(t, app, req)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "https://web.ritme.app", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", resp.Header.Get("Vary"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))

	// No Origin (native app): no ACAO, but Vary: Origin all the same.
	resp = do(t, app, httptest.NewRequest(fiber.MethodGet, "/api/v1/x", nil))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", resp.Header.Get("Vary"))

	// Unknown api route still carries CORS headers (Laravel's middleware is global).
	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/missing", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp = do(t, app, req)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORS_SingleOrigin(t *testing.T) {
	app := newApp(apihttp.NewRegistry(), testDeps("https://web.ritme.app"))

	resp := do(t, app, preflight("https://evil.example", "GET", "x-a"))
	assert.Equal(t, "https://web.ritme.app", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Access-Control-Request-Method, Access-Control-Request-Headers", resp.Header.Get("Vary"))
}

func ipRegistry() *apihttp.Registry {
	reg := apihttp.NewRegistry()
	reg.Register("ip", func(r fiber.Router, _ *apihttp.Deps) {
		r.Get("/ip", func(c fiber.Ctx) error { return c.SendString(c.IP()) })
	})
	return reg
}

// serveIP runs newApp on a real 127.0.0.1 listener (app.Test's in-memory peer is 0.0.0.0, never a proxy) and returns
// what c.IP() saw for a request carrying xff.
func serveIP(t *testing.T, xff string) string {
	t.Helper()
	app := newApp(ipRegistry(), testDeps())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	req, err := nethttp.NewRequest(fiber.MethodGet, "http://"+ln.Addr().String()+"/ip", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", xff)
	resp, err := nethttp.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// T-M12-02 (S-03): behind the trusted nginx (a loopback / docker-network peer) c.IP() is the address nginx forwarded.
func TestClientIP_TrustedProxyForwardsClient(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	assert.Equal(t, "5.6.7.8", serveIP(t, "5.6.7.8"))
}

// A peer that is NOT a trusted proxy (here app.Test's 0.0.0.0 — what 0.0.0.0/0 used to trust) cannot choose its own
// rate-limit key: a forged X-Forwarded-For, single or chained, IPv4 or IPv6, leaves c.IP() on the peer.
func TestClientIP_ForgedForwardedForFromUntrustedPeerIgnored(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	app := newApp(ipRegistry(), testDeps())
	for _, xff := range []string{"5.6.7.8", "5.6.7.8, 172.18.0.5", "2001:db8::1", "127.0.0.1", ""} {
		req := httptest.NewRequest(fiber.MethodGet, "/ip", nil)
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		body, _ := io.ReadAll(do(t, app, req).Body)
		assert.Equal(t, "0.0.0.0", string(body), "X-Forwarded-For %q", xff)
	}
}

// TRUSTED_PROXIES narrows the trust to the given network: loopback is then untrusted too, so the forged header from a
// 127.0.0.1 peer is ignored (boundary of the configured list).
func TestClientIP_TrustedProxiesEnvNarrowsTrust(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.30.0.0/16")
	assert.Equal(t, "127.0.0.1", serveIP(t, "5.6.7.8"))
}

func TestTrustedProxies_Parse(t *testing.T) {
	assert.Equal(t, defaultTrustedProxies, trustedProxies("", nil))
	assert.Equal(t, []string{"172.30.0.0/16", "10.1.2.3/32", "fd00::/8"},
		trustedProxies("172.30.1.0/16, 10.1.2.3 fd00::/8", nil))
	// Never a public or catch-all range: each is dropped; nothing valid left = the defaults, not "trust all".
	for _, bad := range []string{"0.0.0.0/0", "::/0", "8.8.8.0/24", "1.2.3.4", "10.0.0.0/7", "172.0.0.0/8", "nope", "300.1.1.1"} {
		assert.Equal(t, defaultTrustedProxies, trustedProxies(bad, nil), bad)
	}
	assert.Equal(t, []string{"192.168.5.0/24"}, trustedProxies("0.0.0.0/0,192.168.5.0/24,::/0", nil))
}

func TestBodyLimit(t *testing.T) {
	reg := apihttp.NewRegistry()
	reg.Register("up", func(r fiber.Router, _ *apihttp.Deps) {
		r.Post("/upload", func(c fiber.Ctx) error { return c.SendString("ok") })
	})
	app := newApp(reg, testDeps())
	assert.Equal(t, ai.RequestBodyLimit, bodyLimit, "B-N6-05b: the AI upload limits are sized against this body limit")

	// Exactly 25 MB is accepted.
	req := httptest.NewRequest(fiber.MethodPost, "/upload", strings.NewReader(strings.Repeat("a", bodyLimit)))
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// One byte more is rejected by fasthttp while reading the request (the client
	// sees 413 Request Entity Too Large and the connection is closed).
	req = httptest.NewRequest(fiber.MethodPost, "/upload", strings.NewReader(strings.Repeat("a", bodyLimit+1)))
	_, err = app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	require.ErrorContains(t, err, "body size exceeds the given limit")
}

func TestDSN(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tehran")
	require.NoError(t, err)
	dsn := db.DSN(config.DB{Host: "mysql", Port: 3306, Database: "ritme_salamat", Username: "ritme", Password: "p@ss"}, loc)

	assert.True(t, strings.HasPrefix(dsn, "ritme:p@ss@tcp(mysql:3306)/ritme_salamat?"), dsn)
	for _, want := range []string{"parseTime=true", "loc=Asia%2FTehran", "charset=utf8mb4", "collation=utf8mb4_unicode_ci"} {
		assert.Contains(t, dsn, want)
	}
	assert.NotContains(t, dsn, "time_zone")
}
