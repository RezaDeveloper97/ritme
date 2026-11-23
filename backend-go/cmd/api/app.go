package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/valyala/fasthttp"

	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// slowRequest is the slow-request warning threshold (docs/engine-v6/README.md §6).
const slowRequest = 500 * time.Millisecond

// bodyLimit matches the upload ceiling of the PHP stack (25 MB).
const bodyLimit = 25 * 1024 * 1024

// newApp builds the Fiber app: proxy/IP handling, timeouts, access log, compression, request deadline, CORS, then
// every registered domain from reg.
func newApp(reg *apihttp.Registry, deps *apihttp.Deps) *fiber.App {
	var timeouts config.HTTPTimeouts
	if deps.Config != nil {
		timeouts = deps.Config.HTTP.Timeouts
	}
	timeouts = timeouts.WithDefaults()

	app := fiber.New(fiber.Config{
		AppName:   "ritme-backend-go",
		BodyLimit: bodyLimit,
		// T-M9-10: bounded listener timeouts (fasthttp's default is none). Write sits above the longest regular
		// response, the assistant SSE stream; the unbuffered media routes get timeouts.Media (mediaTimeouts).
		ReadTimeout:  timeouts.Read,
		WriteTimeout: timeouts.Write,
		IdleTimeout:  timeouts.Idle,
		// T-M12-02 (S-03): X-Forwarded-For is believed only from a trusted proxy — the nginx container on the docker
		// network (TRUSTED_PROXIES, default loopback + private ranges; never 0.0.0.0/0). nginx resolves the real
		// client itself (deploy/cdn-real-ip.inc trusts only the CDN edges) and forwards exactly one address, its
		// $remote_addr, so c.IP() is that address. Any other peer's X-Forwarded-For is ignored and c.IP() is the peer.
		ProxyHeader:        fiber.HeaderXForwardedFor,
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: trustedProxies(os.Getenv("TRUSTED_PROXIES"), deps.Logger)},
		EnableIPValidation: true,
		ErrorHandler:       httpx.ErrorHandler(deps.Logger),
	})
	app.Server().HeaderReceived = mediaTimeouts(timeouts.Media)

	// Observability first (T-M9-07): request id, then route metrics + the per-request query counter, so every later
	// middleware (module gate, auth) is counted and logged under the request's id.
	app.Use(httpx.RequestID())
	app.Use(metrics.HTTP())
	app.Use(accessLog(deps.Logger))
	// A panic in any handler becomes a 500 instead of killing the process (security audit of CB-REC-02); it sits
	// inside the access log and metrics so the 500 is logged and counted like any other.
	app.Use(recoverPanics(deps.Logger))
	// JSON bodies ≥ 1 KB leave brotli/gzip-compressed (D-132); streams, files and small bodies stay as they are.
	app.Use(httpx.Compress())
	// Every query made with c.Context() is cancelled after timeouts.Request, except on the AI / upload / export
	// allow-list (httpx.LongRunning).
	app.Use(httpx.RequestDeadline(timeouts.Request, httpx.LongRunning))
	// Lets the strangler smoke tests see which stack answered (deploy/switch-go-route.sh).
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Backend", "go")
		return c.Next()
	})
	// Per-request frozen clock (X-Test-Now), same gate as Laravel's TestClock.
	app.Use(clock.Middleware(clock.Real{}, clock.TestClockEnabledFromEnv()))
	app.Use(corsMiddleware(deps.Config.CORS.AllowedOrigins))
	// The user-local day (D-126): X-Timezone → stored user_profiles.timezone → Asia/Tehran, read by
	// civildate.LocalToday. A changed zone is stored after the response, never blocking it.
	if deps.DB != nil {
		app.Use(usertz.New(profilestore.New(deps.DB), deps.Logger).Handler())
	}

	// Admin kill-switches (app_modules): a disabled module's routes answer 403 module_disabled before any domain.
	if deps.DB != nil {
		app.Use(modules.Gate(modules.NewService(deps.DB, deps.Cache, deps.Logger)))
	}

	reg.Mount(app, deps)
	return app
}

// defaultTrustedProxies is where the nginx proxy reaches the API from when TRUSTED_PROXIES is unset: loopback (local
// dev, the container's own healthcheck) and the private ranges Docker allocates bridge networks from. The API port is
// never published on the host (docker-compose.{stage,beta,prod}.yml), so a peer in these ranges is a container on a
// docker network; set TRUSTED_PROXIES to the exact proxy network CIDR to narrow it further.
var defaultTrustedProxies = []string{"127.0.0.0/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"}

// trustedProxies parses TRUSTED_PROXIES (comma- or space-separated IPs / CIDRs) into Fiber's proxy allow-list. An
// entry that does not parse, or that is not a private / loopback range (0.0.0.0/0, ::/0, any public network), is
// dropped with a warning: trusting a public peer would let it forge the client IP every per-IP limit keys on. Empty
// (or nothing valid left) means defaultTrustedProxies.
func trustedProxies(raw string, logger *slog.Logger) []string {
	var out []string
	for _, entry := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			addr, aerr := netip.ParseAddr(entry)
			if aerr != nil {
				warnProxy(logger, entry, "not an IP or CIDR")
				continue
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		if !internalPrefix(prefix) {
			warnProxy(logger, entry, "not a private or loopback range")
			continue
		}
		out = append(out, prefix.Masked().String())
	}
	if len(out) == 0 {
		return append([]string(nil), defaultTrustedProxies...)
	}
	return out
}

// internalPrefix reports whether every address of p is loopback or private (RFC 1918 / RFC 4193): the network's first
// address must be one, and the prefix may not be wider than the range it sits in (10/8, 172.16/12, 192.168/16, fc00::/7,
// 127/8, ::1/128).
func internalPrefix(p netip.Prefix) bool {
	a := p.Masked().Addr().Unmap()
	if !a.IsLoopback() && !a.IsPrivate() {
		return false
	}
	for _, r := range defaultTrustedProxies {
		rp := netip.MustParsePrefix(r)
		if rp.Contains(a) && p.Bits() >= rp.Bits() {
			return true
		}
	}
	return false
}

func warnProxy(logger *slog.Logger, entry, why string) {
	if logger != nil {
		logger.Warn("TRUSTED_PROXIES entry ignored", slog.String("entry", entry), slog.String("reason", why))
	}
}

// mediaTimeouts is the fasthttp HeaderReceived hook: the media upload chunks and the playback stream, which nginx
// does not buffer (proxy_request_buffering / proxy_buffering off), read and write at the client's speed, so they get
// d instead of the listener's ReadTimeout / WriteTimeout. Every other request keeps the listener's.
func mediaTimeouts(d time.Duration) func(*fasthttp.RequestHeader) fasthttp.RequestConfig {
	return func(h *fasthttp.RequestHeader) fasthttp.RequestConfig {
		if !httpx.MediaTransfer(string(h.RequestURI())) {
			return fasthttp.RequestConfig{}
		}
		return fasthttp.RequestConfig{ReadTimeout: d, WriteTimeout: d}
	}
}

// recoverPanics turns a handler panic into the framework 500 (httpx.ServerError through the error handler). The log
// line carries the method, the masked path (httpx.LogPath) and the panic's Go type only — never its value, the body or
// a stack, which could hold health data.
func recoverPanics(logger *slog.Logger) fiber.Handler {
	return fiberrecover.New(fiberrecover.Config{PanicHandler: func(c fiber.Ctx, r any) error {
		logger.LogAttrs(c.Context(), slog.LevelError, "panic recovered",
			slog.String("method", c.Method()),
			slog.String("path", httpx.LogPath(c.Path())),
			slog.String("panic_type", fmt.Sprintf("%T", r)),
		)
		return httpx.ServerError()
	}})
}

// accessLog writes one JSON line per request (with its request_id and DB query count) and a "slow request" warning
// above slowRequest.
func accessLog(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		elapsed := time.Since(start)
		status := c.Response().StatusCode()
		if err != nil {
			status = httpx.StatusOf(err)
		}
		queries := metrics.RequestQueries(c)
		if elapsed > slowRequest {
			logger.LogAttrs(c.Context(), slog.LevelWarn, "slow request",
				slog.String("method", c.Method()),
				slog.String("route", metrics.RouteLabel(c, status)),
				slog.Int("status", status),
				slog.Int64("queries", queries),
				slog.Float64("duration_ms", float64(elapsed.Microseconds())/1000),
			)
		}
		level := slog.LevelInfo
		if status >= fiber.StatusInternalServerError {
			level = slog.LevelError
		} else if p := c.Path(); p == "/up" || p == "/ready" {
			return err // Docker probes /up every 10s and deploys poll /ready; don't flood the log.
		}
		attrs := []slog.Attr{
			slog.String("request_id", httpx.RequestIDFrom(c.Context())),
			slog.String("method", c.Method()),
			slog.String("path", httpx.LogPath(c.Path())),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(elapsed.Microseconds())/1000),
			slog.Int64("queries", queries),
		}
		if !httpx.Sensitive(c.Path()) { // CB-REC-03: no viewer IP next to a public share / emergency read
			attrs = append(attrs, slog.String("ip", c.IP()))
		}
		if err != nil && status >= fiber.StatusInternalServerError {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
		logger.LogAttrs(c.Context(), level, "request", attrs...)
		return err
	}
}
