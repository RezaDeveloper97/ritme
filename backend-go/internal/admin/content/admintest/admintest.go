// Package admintest is the integration-test harness of the admin API packages (T-M2-21):
// a Fiber app with the production error handler on a fresh test database and a miniredis,
// signed-in admin clients (super / editor) that keep cookies and echo the CSRF token, and
// JSON / multipart request helpers. Only test code imports it.
package admintest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Host is the admin host the test app answers on.
const Host = "adpanell.ritme.test"

// Admin ids seeded by New.
const (
	SuperID  uint64 = 1
	EditorID uint64 = 2
)

// Quiet is a discarding logger.
var Quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// Env is one test's app, database, Redis and storage directory.
type Env struct {
	T        *testing.T
	DB       *sql.DB
	App      *fiber.App
	Cache    *cache.Client
	Redis    *miniredis.Miniredis
	Kit      *httpadmin.Kit
	Registry *i18n.Registry
	Storage  string // STORAGE_PATH
}

// New builds the environment (skips without TEST_DB_DSN). Mount routes with Route.
func New(t *testing.T) *Env {
	t.Helper()
	db := testdb.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := cache.NewFromClient(rdb, "ritme-go:")
	reg := i18n.NewRegistry(i18nstore.New(db), c, Quiet)
	kit := httpadmin.NewKit(httpadmin.Options{Hosts: []string{Host}, CookieSecure: true},
		httpadmin.NewSessions(c), store.New(db), i18n.DefaultMiddleware(reg), Quiet)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(Quiet), BodyLimit: 25 << 20})
	e := &Env{T: t, DB: db, App: app, Cache: c, Redis: mr, Kit: kit, Registry: reg, Storage: t.TempDir()}
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(1, 'Root', 'root@ritme.test', ?, 'super', 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00'),
		(2, 'Editor', 'editor@ritme.test', ?, 'editor', 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00')`, pw, pw)
	return e
}

// Route is the registrar the packages' Routes methods take.
func (e *Env) Route() func(method, path string, chain httpadmin.Chain) {
	return func(method, path string, chain httpadmin.Chain) {
		httpadmin.Handle(e.App, method, httpadmin.Prefix+path, chain)
	}
}

// Exec runs a statement.
func (e *Env) Exec(query string, args ...any) sql.Result {
	e.T.Helper()
	res, err := e.DB.ExecContext(e.T.Context(), query, args...)
	require.NoError(e.T, err)
	return res
}

// Int runs a scalar integer query.
func (e *Env) Int(query string, args ...any) int {
	e.T.Helper()
	var n int
	require.NoError(e.T, e.DB.QueryRowContext(e.T.Context(), query, args...).Scan(&n))
	return n
}

// String runs a scalar string query ("" for NULL).
func (e *Env) String(query string, args ...any) string {
	e.T.Helper()
	var s sql.NullString
	require.NoError(e.T, e.DB.QueryRowContext(e.T.Context(), query, args...).Scan(&s))
	return s.String
}

// Client is a signed-in browser: it keeps the cookies and sends the CSRF token.
type Client struct {
	e       *Env
	cookies map[string]string
	CSRF    string
}

// As signs in the admin (a real session in Redis, as after POST /auth/login).
func (e *Env) As(adminID uint64) *Client {
	e.T.Helper()
	sess, err := e.Kit.Sessions().Create(e.T.Context(), adminID, false, time.Now())
	require.NoError(e.T, err)
	return &Client{e: e, CSRF: sess.CSRF, cookies: map[string]string{e.Kit.SessionCookieName(): sess.ID()}}
}

// Anonymous is a client without a session.
func (e *Env) Anonymous() *Client { return &Client{e: e, cookies: map[string]string{}} }

// Resp is a decoded response.
type Resp struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

// Data is body.data as an object.
func (r Resp) Data() map[string]any { d, _ := r.Body["data"].(map[string]any); return d }

// Obj is body.data[key] as an object.
func (r Resp) Obj(key string) map[string]any { d, _ := r.Data()[key].(map[string]any); return d }

// Items is body.data.items.
func (r Resp) Items() []any { d, _ := r.Data()["items"].([]any); return d }

// Errors is the 422 error bag.
func (r Resp) Errors() map[string]any { d, _ := r.Body["errors"].(map[string]any); return d }

// Code is body.error_code.
func (r Resp) Code() string { s, _ := r.Body["error_code"].(string); return s }

func (c *Client) send(req *http.Request) Resp {
	c.e.T.Helper()
	req.Host = Host
	req.Header.Set("Accept", "application/json")
	for k, v := range c.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v}) //nolint:gosec // G124: request cookie in a test client
	}
	if c.CSRF != "" {
		req.Header.Set(httpadmin.CSRFHeader, c.CSRF)
	}
	res, err := c.e.App.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(c.e.T, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	out := Resp{Status: res.StatusCode, Raw: raw}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// JSON sends body (nil = none) as application/json.
func (c *Client) JSON(method, path string, body any) Resp {
	c.e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.e.T, err)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequestWithContext(c.e.T.Context(), method, httpadmin.Prefix+path, rd)
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.send(req)
}

// Get is JSON(GET, path, nil).
func (c *Client) Get(path string) Resp { c.e.T.Helper(); return c.JSON(fiber.MethodGet, path, nil) }

// File is one multipart file part.
type File struct {
	Field, Name, ContentType string
	Data                     []byte
}

// Multipart sends form fields (PHP bracket names allowed: "title[fa]") and files.
func (c *Client) Multipart(method, path string, fields map[string]string, files ...File) Resp {
	c.e.T.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(c.e.T, w.WriteField(k, v))
	}
	for _, f := range files {
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="` + f.Field + `"; filename="` +
			strings.ReplaceAll(f.Name, `"`, "") + `"`}
		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h["Content-Type"] = []string{ct}
		part, err := w.CreatePart(h)
		require.NoError(c.e.T, err)
		_, err = part.Write(f.Data)
		require.NoError(c.e.T, err)
	}
	require.NoError(c.e.T, w.Close())
	req := httptest.NewRequestWithContext(c.e.T.Context(), method, httpadmin.Prefix+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.send(req)
}
