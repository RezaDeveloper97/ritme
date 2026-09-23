package content

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

func storageApp(t *testing.T) (*fiber.App, string) {
	t.Helper()
	storage := t.TempDir()
	public := filepath.Join(storage, "app", "public")
	require.NoError(t, os.MkdirAll(filepath.Join(public, "banners"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(public, "banners", "a.webp"), []byte("webp-bytes"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(public, ".gitignore"), []byte("*"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(storage, "oauth-private.key"), []byte("secret"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(storage, "oauth-private.key"), filepath.Join(public, "escape.key")))

	h := New(Deps{StoragePath: storage})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Get("/storage/*", h.Storage)
	return app, public
}

func get(t *testing.T, app *fiber.App, target string, headers ...string) (int, string, map[string]string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "http://localhost/", nil)
	req.URL.Opaque = target // keep the raw (possibly encoded) path
	req.RequestURI = target
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	h := map[string]string{}
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	return resp.StatusCode, string(body), h
}

func TestStorageServesPublicFiles(t *testing.T) {
	app, _ := storageApp(t)
	status, body, h := get(t, app, "/storage/banners/a.webp")
	assert.Equal(t, 200, status)
	assert.Equal(t, "webp-bytes", body)
	assert.Equal(t, "image/webp", h["Content-Type"])
	assert.Equal(t, "bytes", h["Accept-Ranges"])
	assert.NotEmpty(t, h["Last-Modified"])
	require.NotEmpty(t, h["Etag"])

	status, body, _ = get(t, app, "/storage/banners/a.webp", "If-None-Match", h["Etag"])
	assert.Equal(t, 304, status)
	assert.Empty(t, body)
	status, _, _ = get(t, app, "/storage/banners/a.webp", "If-Modified-Since", h["Last-Modified"])
	assert.Equal(t, 304, status)
}

func TestStorageRejectsTraversalAndListings(t *testing.T) {
	app, _ := storageApp(t)
	for _, target := range []string{
		"/storage/",
		"/storage/banners",
		"/storage/banners/",
		"/storage/missing.png",
		"/storage/.gitignore",
		"/storage/escape.key",
		"/storage/../oauth-private.key",
		"/storage/..%2f..%2foauth-private.key",
		"/storage/..%2F..%2Foauth-private.key",
		"/storage/%2e%2e/%2e%2e/oauth-private.key",
		"/storage/banners/..%2f..%2f..%2foauth-private.key",
		"/storage/banners%2f..%2f..%2f..%2foauth-private.key",
		"/storage/..%5c..%5coauth-private.key",
		"/storage/banners/a.webp%00.png",
	} {
		status, body, _ := get(t, app, target)
		assert.Equal(t, 404, status, target)
		assert.NotContains(t, body, "secret", target)
	}
}

func TestPublicURL(t *testing.T) {
	assert.Equal(t, "https://api.ritme.app/storage/banners/a.webp", PublicURL("https://api.ritme.app", "banners/a.webp"))
	assert.Equal(t, "https://api.ritme.app/storage/banners/a.webp", PublicURL("https://api.ritme.app", "/banners/a.webp"))
	// env('APP_URL').'/storage' is not normalised by Laravel either.
	assert.Equal(t, "https://x//storage/a", PublicURL("https://x/", "a"))
}

func TestStorageDecodesNames(t *testing.T) {
	app, public := storageApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(public, "banners", "a b.png"), []byte("png"), 0o644))
	status, body, h := get(t, app, "/storage/banners/a%20b.png")
	assert.Equal(t, 200, status)
	assert.Equal(t, "png", body)
	assert.Equal(t, "image/png", h["Content-Type"])
}

func TestStorageRanges(t *testing.T) {
	app, _ := storageApp(t) // a.webp = "webp-bytes" (10 bytes)
	status, body, h := get(t, app, "/storage/banners/a.webp", "Range", "bytes=0-3")
	assert.Equal(t, 206, status)
	assert.Equal(t, "webp", body)
	assert.Equal(t, "bytes 0-3/10", h["Content-Range"])
	status, body, _ = get(t, app, "/storage/banners/a.webp", "Range", "bytes=-5")
	assert.Equal(t, 206, status)
	assert.Equal(t, "bytes", body)
	status, body, _ = get(t, app, "/storage/banners/a.webp", "Range", "bytes=5-")
	assert.Equal(t, 206, status)
	assert.Equal(t, "bytes", body)
	status, _, h = get(t, app, "/storage/banners/a.webp", "Range", "bytes=20-")
	assert.Equal(t, 416, status)
	assert.Equal(t, "bytes */10", h["Content-Range"])
	status, body, _ = get(t, app, "/storage/banners/a.webp", "Range", "bytes=0-1,4-5")
	assert.Equal(t, 200, status)
	assert.Equal(t, "webp-bytes", body)
}
