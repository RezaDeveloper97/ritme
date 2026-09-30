package support_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/support"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	support.NewHandlers(e.DB, e.Storage, admintest.Quiet).Routes(e.Route(), e.Kit)
	// The public disk as the app serves it: a private screenshot must never be reachable there.
	e.App.Get("/storage/*", publiccontent.New(publiccontent.Deps{StoragePath: e.Storage, AppURL: "https://api.ritme.test"}).Storage)
	return e
}

func newUser(e *admintest.Env, mobile, name string) uint64 {
	e.T.Helper()
	id, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", mobile, name).LastInsertId()
	require.NoError(e.T, err)
	return uint64(id) //nolint:gosec // test ids
}

func newReport(e *admintest.Env, userID uint64, msg, shot, status, created string) string {
	e.T.Helper()
	var path any
	if shot != "" {
		path = shot
	}
	id, err := e.Exec(`INSERT INTO support_reports (user_id, message, screenshot_path, app_version, user_agent, status, created_at, updated_at)
		VALUES (?, ?, ?, '1.4.0', 'Mozilla/5.0 test', ?, ?, ?)`, userID, msg, path, status, created, created).LastInsertId()
	require.NoError(e.T, err)
	return strconv.FormatInt(id, 10)
}

// webp is a minimal RIFF/WEBP header (the stream does not decode, only sends bytes).
var webp = []byte("RIFF\x1a\x00\x00\x00WEBPVP8 \x0e\x00\x00\x00test-screenshot")

func writeShot(t *testing.T, e *admintest.Env, rel string) {
	t.Helper()
	abs := filepath.Join(e.Storage, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, webp, 0o600))
}

func TestGuards(t *testing.T) {
	e := newEnv(t)
	u := newUser(e, "09120000001", "Sara")
	id := newReport(e, u, "The calendar does not open after update", "", "open", "2026-09-30 10:00:00")

	for _, path := range []string{"/support-reports", "/support-reports/" + id, "/support-reports/" + id + "/screenshot"} {
		r := e.Anonymous().Get(path)
		assert.Equal(t, 401, r.Status, path)
		assert.Equal(t, "unauthenticated", r.Code(), path)
	}
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/support-reports").Status, "any active admin reads the inbox")

	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	r := noToken.JSON(fiber.MethodPost, "/support-reports/"+id+"/resolve", nil)
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, "open", e.String("SELECT status FROM support_reports WHERE id = ?", id))

	e.Exec("UPDATE admins SET is_active = 0 WHERE id = ?", admintest.EditorID)
	assert.Equal(t, 401, e.As(admintest.EditorID).Get("/support-reports").Status, "a deactivated admin is out")
}

func TestListFiltersAndPreview(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	u := newUser(e, "09120000001", "Sara")
	long := strings.Repeat("ب", 300)
	older := newReport(e, u, "Older open report text", "", "open", "2026-09-28 09:00:00")
	newer := newReport(e, u, long, "app/private/support-reports/ab/x.webp", "open", "2026-09-30 09:00:00")
	done := newReport(e, u, "Already handled report", "", "resolved", "2026-09-29 09:00:00")

	r := c.Get("/support-reports")
	require.Equal(t, 200, r.Status, r.Body)
	items := r.Items()
	require.Len(t, items, 2, "default filter is open")
	first := items[0].(map[string]any)
	assert.EqualValues(t, mustInt(newer), first["id"], "newest first")
	assert.EqualValues(t, mustInt(older), items[1].(map[string]any)["id"])
	assert.Len(t, []rune(first["preview"].(string)), 160, "list carries a short preview only")
	assert.NotContains(t, first, "message")
	assert.NotContains(t, first, "user_agent")
	assert.NotContains(t, first, "screenshot_path", "the storage path never leaves the API")
	assert.Equal(t, true, first["has_screenshot"])
	assert.Equal(t, map[string]any{"id": float64(u), "name": "Sara", "mobile": "09120000001"}, first["user"])
	assert.Equal(t, map[string]any{"status": "open"}, r.Data()["filters"])
	assert.Equal(t, map[string]any{"open": float64(2), "resolved": float64(1)}, r.Data()["counts"])
	assert.EqualValues(t, 2, r.Data()["meta"].(map[string]any)["total"])

	r = c.Get("/support-reports?status=resolved")
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, mustInt(done), r.Items()[0].(map[string]any)["id"])

	r = c.Get("/support-reports?status=all&per_page=2&page=2")
	require.Equal(t, 200, r.Status)
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, mustInt(older), r.Items()[0].(map[string]any)["id"], "all: newest first across statuses")

	r = c.Get("/support-reports?status=bogus")
	assert.Equal(t, "open", r.Data()["filters"].(map[string]any)["status"])
}

func TestShowResolveReopen(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	u := newUser(e, "09120000001", "")
	id := newReport(e, u, "The calendar does not open after update", "", "open", "2026-09-30 10:00:00")

	r := c.Get("/support-reports/" + id)
	require.Equal(t, 200, r.Status, r.Body)
	rep := r.Obj("support_report")
	assert.Equal(t, "The calendar does not open after update", rep["message"])
	assert.Equal(t, false, rep["has_screenshot"])
	assert.Equal(t, "1.4.0", rep["app_version"])
	assert.Equal(t, "Mozilla/5.0 test", rep["user_agent"])
	assert.Equal(t, "open", rep["status"])
	assert.Equal(t, "2026-09-30T10:00:00+03:30", rep["created_at"])
	assert.NotContains(t, rep, "screenshot_path")

	r = c.JSON(fiber.MethodPost, "/support-reports/"+id+"/resolve", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "resolved", r.Obj("support_report")["status"])
	assert.Equal(t, "Report resolved.", r.Body["message"])
	assert.Equal(t, "resolved", e.String("SELECT status FROM support_reports WHERE id = ?", id))

	r = c.JSON(fiber.MethodPost, "/support-reports/"+id+"/resolve", nil)
	require.Equal(t, 200, r.Status, "resolving twice is idempotent")

	r = c.JSON(fiber.MethodPost, "/support-reports/"+id+"/reopen", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "open", r.Obj("support_report")["status"])

	for _, path := range []string{"/support-reports/999999", "/support-reports/abc", "/support-reports/0"} {
		r = c.Get(path)
		assert.Equal(t, 404, r.Status, path)
	}
	r = c.JSON(fiber.MethodPost, "/support-reports/999999/resolve", nil)
	assert.Equal(t, 404, r.Status)
}

func TestScreenshotIsPrivate(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	u := newUser(e, "09120000001", "Sara")
	const rel = "app/private/support-reports/ab/shot.webp"
	writeShot(t, e, rel)
	withShot := newReport(e, u, "Look at this screenshot please", rel, "open", "2026-09-30 10:00:00")
	noShot := newReport(e, u, "No screenshot on this one", "", "open", "2026-09-30 11:00:00")
	gone := newReport(e, u, "File was removed from disk", "app/private/support-reports/ab/missing.webp", "open", "2026-09-30 12:00:00")
	// A tampered row must not turn the stream into a file reader.
	writeShot(t, e, "app/public/banners/leak.webp")
	escape := newReport(e, u, "Row pointing outside the private dir", "app/public/banners/leak.webp", "open", "2026-09-30 13:00:00")
	dots := newReport(e, u, "Row with a dot-dot path segment", "app/private/support-reports/../../public/banners/leak.webp", "open", "2026-09-30 14:00:00")

	r := c.Get("/support-reports/" + withShot + "/screenshot")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, webp, r.Raw)

	res := raw(t, e, "/api/admin/v1/support-reports/"+withShot+"/screenshot", true)
	assert.Equal(t, "image/webp", res.Header.Get("Content-Type"))
	assert.Equal(t, "private, no-store", res.Header.Get("Cache-Control"))
	assert.Equal(t, "nosniff", res.Header.Get("X-Content-Type-Options"))

	for _, id := range []string{noShot, gone, escape, dots} {
		assert.Equal(t, 404, c.Get("/support-reports/"+id+"/screenshot").Status, id)
	}
	assert.Equal(t, 401, e.Anonymous().Get("/support-reports/"+withShot+"/screenshot").Status)

	// Never through the public disk, whatever the path.
	for _, p := range []string{"/storage/" + rel, "/storage/private/support-reports/ab/shot.webp", "/storage/support-reports/ab/shot.webp"} {
		res := raw(t, e, p, false)
		assert.NotEqual(t, 200, res.StatusCode, p)
	}
}

// raw sends a GET on the admin host (signed in as the editor when signedIn) and returns the raw response.
func raw(t *testing.T, e *admintest.Env, path string, signedIn bool) *http.Response {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Host = admintest.Host
	if signedIn {
		sess, err := e.Kit.Sessions().Create(t.Context(), admintest.EditorID, false, time.Now())
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: e.Kit.SessionCookieName(), Value: sess.ID()}) //nolint:gosec // G124: test request cookie
	}
	res, err := e.App.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func mustInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
