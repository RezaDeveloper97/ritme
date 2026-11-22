// Package testenv boots the real API (every registered domain, internal/http.Mount) on the integration test database
// and Redis for end-to-end flow tests (T-M12-15 and the flows after it): a frozen test clock per request
// (X-Test-Now), OTP sign-up with the code read from otp_verifications (SMS_PROVIDER=log) and a small JSON client.
//
// The middleware chain mirrors cmd/api's newApp where it changes behaviour (test clock → user zone → module gate →
// domains); request ids, metrics, the access log, compression and CORS are left out. The configuration comes from
// config.LoadFrom, so every default (OTP caps, engine flags, …) is the production default unless a test overrides it.
//
// Redis: TEST_REDIS_ADDR when set (make test-int), under a fresh key prefix per Env that is deleted afterwards;
// otherwise an in-process miniredis. Tests that use an Env must not run in parallel (testdb shares one database per
// package) and the package needs `func TestMain(m *testing.M) { testdb.Main(m) }`.
package testenv

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// DefaultNow is the clock of a new Env (the contract clock).
const DefaultNow = "2026-09-23T10:00:00+03:30"

// personalClientID is the Passport personal access client the issuer needs (as in the other http int tests).
const personalClientID = "0199c0de-0000-7000-8000-00000c0ffe48"

// Env is one booted API over a clean test database.
type Env struct {
	T     *testing.T
	DB    *sql.DB
	App   *fiber.App
	Cache *cache.Client
	// Now is the X-Test-Now of every request that does not name its own (SetNow / Request.Now).
	Now string
	// Lang is the Accept-Language of every request ("en" by default).
	Lang string

	iss *passport.Issuer
}

// Option changes the environment variables the configuration is loaded from.
type Option func(vars map[string]string)

// WithEnv sets one configuration variable (e.g. WithEnv("CYCLE_ENGINE", "v1")).
func WithEnv(key, value string) Option { return func(v map[string]string) { v[key] = value } }

// New boots the API. Defaults: APP_ENV=testing, SMS_PROVIDER=log, CYCLE_ENGINE=v6 (what stage serves), queue sync.
func New(t *testing.T, opts ...Option) *Env {
	t.Helper()
	db := testdb.New(t)
	keys := keysDir(t)
	vars := map[string]string{
		"APP_ENV": "testing", "APP_URL": "http://localhost",
		// Never dialled: Deps carries the test pool and Redis client.
		"DB_HOST": "testdb", "DB_DATABASE": "testdb", "DB_USERNAME": "testdb", "REDIS_HOST": "testredis",
		"STORAGE_PATH": keys, "SMS_PROVIDER": "log", "CYCLE_ENGINE": "v6", "QUEUE_CONNECTION": "sync",
	}
	for _, o := range opts {
		o(vars)
	}
	cfg, err := config.LoadFrom(func(k string) (string, bool) { v, ok := vars[k]; return v, ok })
	require.NoError(t, err, "testenv: config")

	pk, err := passport.LoadKeys(keys)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), `INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-01-01 00:00:00', '2026-01-01 00:00:00')`, personalClientID)
	require.NoError(t, err)

	rc := newCache(t)
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	deps := &apihttp.Deps{Config: cfg, DB: db, Cache: rc, Logger: quiet}

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	tz := usertz.New(profilestore.New(db), quiet)
	t.Cleanup(tz.Wait) // the zone is stored after the response
	app.Use(tz.Handler())
	app.Use(modules.Gate(modules.NewService(db, rc, quiet)))
	apihttp.Mount(app, deps)

	return &Env{T: t, DB: db, App: app, Cache: rc, Now: DefaultNow, Lang: "en",
		iss: passport.NewIssuer(pk.Private, authstore.New(db), clock.Real{}, cfg.Passport.TokenLifetimeDays)}
}

// keysDir is backend-go/contract/fixtures/keys (the contract Passport key pair), found from this file.
func keysDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "..", "contract", "fixtures", "keys"))
	require.NoError(t, err)
	return dir
}

// newCache is the real test Redis under a fresh prefix (deleted afterwards), or miniredis without TEST_REDIS_ADDR.
func newCache(t *testing.T) *cache.Client {
	t.Helper()
	b := make([]byte, 6)
	_, err := rand.Read(b)
	require.NoError(t, err)
	prefix := "ritme-go-testenv-" + hex.EncodeToString(b) + ":"
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = miniredis.RunT(t).Addr()
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	require.NoError(t, rdb.Ping(context.Background()).Err(), "testenv: redis at %s", addr)
	t.Cleanup(func() {
		ctx := context.Background()
		iter := rdb.Scan(ctx, 0, prefix+"*", 500).Iterator()
		for iter.Next(ctx) {
			_ = rdb.Del(ctx, iter.Val()).Err()
		}
		_ = rdb.Close()
	})
	return cache.NewFromClient(rdb, prefix)
}

// SetNow moves the test clock of the following requests (any form X-Test-Now accepts).
func (e *Env) SetNow(now string) { e.Now = now }

// Reply is one answer: status, headers, raw body and the decoded JSON object (nil when the body is not one).
type Reply struct {
	Status int
	Header nethttp.Header
	Raw    string
	Body   map[string]any
}

// Data is body.data as an object (nil otherwise).
func (r Reply) Data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

// List is body.data as a list (nil otherwise).
func (r Reply) List() []any {
	d, _ := r.Body["data"].([]any)
	return d
}

// ETag is the ETag response header.
func (r Reply) ETag() string { return r.Header.Get(fiber.HeaderETag) }

// Request is a request with every knob; Do covers the common case.
type Request struct {
	Method, Path string
	Body         any // JSON-encoded unless nil
	// Raw is sent verbatim instead of Body (a multipart upload, a form post) with ContentType.
	Raw         []byte
	ContentType string
	Token       string
	Now         string            // X-Test-Now; "" = Env.Now
	Header      map[string]string // extra headers (If-None-Match, X-Timezone, …)
}

// Do sends method path with a JSON body (nil = none) and a bearer token ("" = anonymous) at Env.Now.
func (e *Env) Do(method, path string, body any, token string) Reply {
	e.T.Helper()
	return e.Send(Request{Method: method, Path: path, Body: body, Token: token})
}

// Send sends r.
func (e *Env) Send(r Request) Reply {
	e.T.Helper()
	var rd io.Reader
	if r.Body != nil {
		raw, err := json.Marshal(r.Body)
		require.NoError(e.T, err)
		rd = strings.NewReader(string(raw))
	}
	ct := "application/json"
	if r.Raw != nil {
		rd, ct = bytes.NewReader(r.Raw), r.ContentType
	}
	req := httptest.NewRequestWithContext(context.Background(), r.Method, r.Path, rd)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", e.Lang)
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
	now := r.Now
	if now == "" {
		now = e.Now
	}
	req.Header.Set(clock.Header, now)
	for k, v := range r.Header {
		req.Header.Set(k, v)
	}
	resp, err := e.App.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.T, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.T, err)
	_ = resp.Body.Close()
	out := Reply{Status: resp.StatusCode, Header: resp.Header, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// MustOK sends the request and fails the test unless the status is want.
func (e *Env) MustOK(want int, method, path string, body any, token string) Reply {
	e.T.Helper()
	r := e.Do(method, path, body, token)
	require.Equal(e.T, want, r.Status, "%s %s: %s", method, path, r.Raw)
	return r
}

// mobiles numbers every Mobile() call of the process: each sign-up gets a number no other test used, so the
// per-number OTP send caps and failure lockout (T-M12-01) never carry over.
var mobiles atomic.Uint32

// Mobile is a fresh valid Iranian mobile number (09 + 9 digits).
func Mobile() string { return fmt.Sprintf("0935%07d", 1_000_000+mobiles.Add(1)) }

// OTP is the newest unverified code sent to mobile (SMS_PROVIDER=log keeps it in otp_verifications only).
func (e *Env) OTP(mobile string) string {
	e.T.Helper()
	var code string
	err := e.DB.QueryRowContext(context.Background(), `SELECT code FROM otp_verifications WHERE mobile = ? AND verified_at IS NULL ORDER BY id DESC LIMIT 1`, mobile).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		e.T.Fatalf("testenv: no OTP for the number")
	}
	require.NoError(e.T, err)
	return code
}

// SignupConsents asks verify-otp which mandatory consents a new account needs (its 422 consent_required answer
// names them with their versions): mobile must hold a fresh code. The code is not spent by that answer.
func (e *Env) SignupConsents(mobile, code string) map[string]any {
	e.T.Helper()
	r := e.Do(fiber.MethodPost, "/api/v1/auth/verify-otp", map[string]any{"mobile": mobile, "code": code}, "")
	require.Equal(e.T, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	require.Equal(e.T, "consent_required", r.Body["error_code"], r.Raw)
	c, ok := r.Body["consents"].(map[string]any)
	require.True(e.T, ok, r.Raw)
	return maps.Clone(c)
}

// Account is a signed-in user.
type Account struct {
	ID     uint64
	Mobile string
	Token  string
}

// SignUp creates an account through the real OTP routes: send-otp → code from the database → verify-otp with the
// mandatory consents (terms, health_data_processing, at the versions the server asks for).
func (e *Env) SignUp() Account {
	e.T.Helper()
	mobile := Mobile()
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/auth/send-otp", map[string]any{"mobile": mobile}, "")
	code := e.OTP(mobile)
	consents := e.SignupConsents(mobile, code)
	r := e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/auth/verify-otp",
		map[string]any{"mobile": mobile, "code": code, "consents": consents}, "")
	d := r.Data()
	require.Equal(e.T, true, d["new_user"], r.Raw)
	tok, _ := d["access_token"].(string)
	require.NotEmpty(e.T, tok, r.Raw)
	user, _ := d["user"].(map[string]any)
	id, _ := user["id"].(float64)
	require.Positive(e.T, id, r.Raw)
	return Account{ID: uint64(id), Mobile: mobile, Token: tok}
}

// User creates an account straight in the database with a token (no OTP, no consents): for flows that do not test
// sign-up.
func (e *Env) User() Account {
	e.T.Helper()
	mobile := Mobile()
	res, err := e.DB.ExecContext(context.Background(), `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-01-01 00:00:00', '2026-01-01 00:00:00')`, mobile)
	require.NoError(e.T, err)
	id, err := res.LastInsertId()
	require.NoError(e.T, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // G115: test ids are positive
	require.NoError(e.T, err)
	return Account{ID: uint64(id), Mobile: mobile, Token: tok.AccessToken} //nolint:gosec // G115: test ids are positive
}

// Count runs a COUNT(*) query.
func (e *Env) Count(query string, args ...any) int {
	e.T.Helper()
	var n int
	require.NoError(e.T, e.DB.QueryRowContext(context.Background(), query, args...).Scan(&n))
	return n
}

// Path digs into decoded JSON: Path(m, "anchors", "current_period_start"). A missing key or non-object is nil.
func Path(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}
