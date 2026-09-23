package http

import (
	"bytes"
	"compress/gzip"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	swaggerui "github.com/swaggest/swgui/v5/static"

	"github.com/ritme/backend-go/api"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// docsPrefix is where the API reference lives (Laravel: l5-swagger behind swagger.auth).
const docsPrefix = "/docs"

// API reference (T-M2-19): Swagger UI from an embedded bundle (no CDN) over the
// hand-maintained api/openapi.yaml, behind the same Basic auth as Laravel's
// SwaggerBasicAuth middleware.
func init() {
	Register("docs", func(r fiber.Router, d *Deps) { mountDocs(r, d.Config) })
}

func mountDocs(r fiber.Router, cfg *config.Config) {
	gate := docsBasicAuth(cfg)
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
// else; otherwise SWAGGER_USER / SWAGGER_PASSWORD must match (constant time).
func docsBasicAuth(cfg *config.Config) fiber.Handler {
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
		user, pass, _ := parseBasicAuth(c.Get(fiber.HeaderAuthorization))
		okUser := subtle.ConstantTimeCompare([]byte(wantUser), []byte(user))
		okPass := subtle.ConstantTimeCompare([]byte(wantPass), []byte(pass))
		if okUser&okPass != 1 {
			return denyDocs(c)
		}
		return c.Next()
	}
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
