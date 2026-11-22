package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	swaggerui "github.com/swaggest/swgui/v5/static"

	"github.com/ritme/backend-go/api"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// docsPrefix is where the API reference lives (Laravel: l5-swagger behind swagger.auth).
const docsPrefix = "/docs"

// API reference (T-M2-19): Swagger UI from an embedded bundle (no CDN) over the
// hand-maintained api/openapi.yaml, behind the same Basic auth as Laravel's
// SwaggerBasicAuth middleware.
func init() {
	Register("docs", func(r fiber.Router, d *Deps) { mountDocs(r, d.Config, d.Cache) })
}

func mountDocs(r fiber.Router, cfg *config.Config, c *cache.Client) {
	gate := docsBasicAuth(cfg, newDocsFailures(c))
	r.Get(docsPrefix, gate, serveDocsIndex)
	r.Get(docsPrefix+"/openapi.yaml", gate, func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml; charset=utf-8")
		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.Send(api.OpenAPI)
	})
	r.Get(docsPrefix+"/assets/:file", gate, serveDocsAsset)
}

// docsBasicAuth ports backend/app/Http/Middleware/SwaggerBasicAuth.php: with no
// SWAGGER_PASSWORD the docs are open only in local/testing and closed (401) everywhere
// else; otherwise SWAGGER_USER / SWAGGER_PASSWORD must match (constant time). T-M12-06 (S-11): a client network
// (IPv6 /64) with DocsFailuresPerMinute wrong passwords in a minute gets 429 on every attempt — the right password
// too, so the lock is no oracle — until the window ends (fails is nil without Redis: no limit).
func docsBasicAuth(cfg *config.Config, fails *docsFailures) fiber.Handler {
	var env, wantUser, wantPass string
	if cfg != nil {
		env, wantUser, wantPass = cfg.App.Env, cfg.Swagger.User, cfg.Swagger.Password
	}
	return func(c fiber.Ctx) error {
		if wantPass == "" {
			if env == "local" || env == "testing" {
				return c.Next()
			}
			return denyDocs(c)
		}
		now := clock.FromContext(c, clock.Real{}).Now()
		ip := ratelimit.ClientNet(c.IP())
		if retry, err := fails.locked(c.Context(), ip, now); err != nil {
			return err
		} else if retry > 0 {
			return httpx.TooManyRequests(retry, DocsFailuresPerMinute, 0, now.Unix()+int64(retry))
		}
		user, pass, _ := parseBasicAuth(c.Get(fiber.HeaderAuthorization))
		okUser := subtle.ConstantTimeCompare([]byte(wantUser), []byte(user))
		okPass := subtle.ConstantTimeCompare([]byte(wantPass), []byte(pass))
		if okUser&okPass != 1 {
			if err := fails.record(c.Context(), ip, now); err != nil {
				return err
			}
			return denyDocs(c)
		}
		return c.Next()
	}
}

// DocsFailuresPerMinute is the /docs wrong-password budget per client network and minute (T-M12-06).
const DocsFailuresPerMinute = 10

// docsFailures counts wrong /docs passwords per client network in a fixed one-minute window that starts at the first
// failure. Only failures count; the window end is compared against the request clock (the key TTL only collects
// garbage). Keys hold sha256(network), never the address.
type docsFailures struct{ cache *cache.Client }

func newDocsFailures(c *cache.Client) *docsFailures {
	if c == nil {
		return nil
	}
	return &docsFailures{cache: c}
}

func (f *docsFailures) key(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return f.cache.Key("docs_auth_fail:" + hex.EncodeToString(sum[:]))
}

// docsLockedScript: KEYS[1] hash {n, end}; ARGV now ms, max → ms until the window ends when it holds max failures, else 0.
var docsLockedScript = redis.NewScript(`
local n = tonumber(redis.call('HGET', KEYS[1], 'n') or '0')
local e = tonumber(redis.call('HGET', KEYS[1], 'end') or '0')
local now = tonumber(ARGV[1])
if n >= tonumber(ARGV[2]) and e > now then return e - now end
return 0
`)

// docsFailScript: KEYS[1] hash {n, end}; ARGV now ms, window ms → records one failure (a new window when none is open).
var docsFailScript = redis.NewScript(`
local now, win = tonumber(ARGV[1]), tonumber(ARGV[2])
local e = tonumber(redis.call('HGET', KEYS[1], 'end') or '0')
if e <= now then
  redis.call('HSET', KEYS[1], 'n', 1, 'end', now + win)
else
  redis.call('HINCRBY', KEYS[1], 'n', 1)
end
redis.call('PEXPIRE', KEYS[1], win)
return 1
`)

// locked returns the seconds until the network may try again (0 = it may).
func (f *docsFailures) locked(ctx context.Context, ip string, now time.Time) (int, error) {
	if f == nil {
		return 0, nil
	}
	left, err := docsLockedScript.Run(ctx, f.cache.Redis(), []string{f.key(ip)}, now.UnixMilli(), DocsFailuresPerMinute).Int64()
	if err != nil {
		return 0, fmt.Errorf("docs: failure budget: %w", err)
	}
	if left <= 0 {
		return 0, nil
	}
	return int((left + 999) / 1000), nil
}

func (f *docsFailures) record(ctx context.Context, ip string, now time.Time) error {
	if f == nil {
		return nil
	}
	if err := docsFailScript.Run(ctx, f.cache.Redis(), []string{f.key(ip)}, now.UnixMilli(), time.Minute.Milliseconds()).Err(); err != nil {
		return fmt.Errorf("docs: failure budget: %w", err)
	}
	return nil
}

func denyDocs(c fiber.Ctx) error {
	c.Set(fiber.HeaderWWWAuthenticate, `Basic realm="Ritme API Docs"`)
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.Status(fiber.StatusUnauthorized).SendString("Authentication required.")
}

// parseBasicAuth is net/http's Request.BasicAuth for a raw header value.
func parseBasicAuth(header string) (user, pass string, ok bool) {
	const prefix = "Basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}
	user, pass, ok = strings.Cut(string(raw), ":")
	return user, pass, ok
}

var docsIndex = template.Must(template.New("docs").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>Ritme API</title>
<link rel="icon" type="image/png" href="{{.}}/assets/favicon-32x32.png">
<link rel="stylesheet" href="{{.}}/assets/swagger-ui.css">
<style>body{margin:0;background:#fafafa}</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="{{.}}/assets/swagger-ui-bundle.js"></script>
<script src="{{.}}/assets/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function () {
  window.ui = SwaggerUIBundle({
    // Absolute URL without userinfo: fetch() refuses URLs that carry credentials.
    url: window.location.origin + "{{.}}/openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    persistAuthorization: false,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout"
  });
};
</script>
</body>
</html>
`))

func serveDocsIndex(c fiber.Ctx) error {
	var buf bytes.Buffer
	if err := docsIndex.Execute(&buf, docsPrefix); err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	c.Set(fiber.HeaderCacheControl, "no-cache")
	return c.Send(buf.Bytes())
}

// serveDocsAsset serves the Swagger UI bundle from github.com/swaggest/swgui, which
// ships the JS/CSS pre-gzipped: sent as-is to gzip clients, inflated for the rest.
func serveDocsAsset(c fiber.Ctx) error {
	name := c.Params("file")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return httpx.NotFound()
	}
	ctype := map[string]string{
		".js": "text/javascript; charset=utf-8", ".css": "text/css; charset=utf-8", ".png": "image/png",
	}[path.Ext(name)]
	if ctype == "" {
		return httpx.NotFound()
	}
	c.Set(fiber.HeaderContentType, ctype)
	c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
	if data, err := fs.ReadFile(swaggerui.FS, name); err == nil {
		return c.Send(data)
	}
	gz, err := fs.ReadFile(swaggerui.FS, name+".gz")
	if errors.Is(err, fs.ErrNotExist) {
		return httpx.NotFound()
	} else if err != nil {
		return err
	}
	c.Vary(fiber.HeaderAcceptEncoding)
	if strings.Contains(c.Get(fiber.HeaderAcceptEncoding), "gzip") {
		c.Set(fiber.HeaderContentEncoding, "gzip")
		return c.Send(gz)
	}
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return err
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		return err
	}
	return c.Send(plain)
}
