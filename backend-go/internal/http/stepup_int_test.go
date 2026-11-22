package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// newStepUpEnv is newCompanionEnv with the production OTP guard limits (companionEnv leaves them zero, which the
// send lock refuses), so send-otp / verify-otp run exactly as wired in cmd/api.
func newStepUpEnv(t *testing.T) *companionEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe48', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	Mount(app, &Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"},
			SMS: config.SMS{Provider: "log", OTP: config.OTPLimits{
				SendShortMax: 3, SendShortWindow: 15 * time.Minute, SendDailyMax: 10, SendDailyWindow: 24 * time.Hour,
				FailureMax: 10, FailureWindow: 24 * time.Hour, LockFor: time.Hour, SendLock: time.Minute,
			}},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-b4n2:"),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})
	return &companionEnv{t: t, db: db, app: app, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365), ids: map[string]uint64{}, uid: map[string]uint64{}, mr: mr}
}

// T-M12-06 (S-10/S-11, D-213): GET /profile/export and DELETE /account need a fresh OTP step-up and are throttled
// 3/hour/user; /docs refuses a client after 10 wrong passwords a minute.

// doStep is companionEnv.do with an X-Step-Up header ("" = none).
func (e *companionEnv) doStep(method, path, token, stepUp string, body any, now ...string) reply {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(e.t, err)
		rd = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if stepUp != "" {
		req.Header.Set(auth.StepUpHeader, stepUp)
	}
	at := companionNow
	if len(now) > 0 {
		at = now[0]
	}
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// stepUpStore is the env's step-up store (the same Redis and prefix as the mounted app).
func (e *companionEnv) stepUpStore() *auth.StepUp {
	rdb := redis.NewClient(&redis.Options{Addr: e.mr.Addr()})
	e.t.Cleanup(func() { _ = rdb.Close() })
	return auth.NewStepUp(cache.NewFromClient(rdb, "ritme-go-b4n2:"))
}

// mintStepUp issues a step-up token for the token's user at the request clock companionNow.
func (e *companionEnv) mintStepUp(token string) string {
	e.t.Helper()
	at, err := time.Parse(time.RFC3339, companionNow)
	require.NoError(e.t, err)
	tok, err := e.stepUpStore().Issue(context.Background(), e.uid[token], at)
	require.NoError(e.t, err)
	return tok
}

func assertStepUpRequired(t *testing.T, r reply, msg string) {
	t.Helper()
	require.Equal(t, fiber.StatusForbidden, r.Status, "%s: %s", msg, r.Raw)
	assert.Equal(t, auth.CodeStepUpRequired, r.Body["error_code"], msg)
	assert.Equal(t, false, r.Body["success"], msg)
}

// The whole flow through the real send-otp / verify-otp: the code goes to the signed-in user's own number, the
// answer is a step-up token (never a login), and the token opens the export exactly once.
func TestStepUp_OTPFlowIssuesSingleUseToken(t *testing.T) {
	e := newStepUpEnv(t)
	sara := e.user("sara", "09120000601")
	e.user("nina", "09120000602")

	// No bearer → the auth 401 (it IS an authentication failure).
	r := e.do(fiber.MethodPost, "/api/v1/auth/send-otp", "", map[string]any{"purpose": "step_up", "mobile": "09120000601"})
	require.Equal(t, fiber.StatusUnauthorized, r.Status, r.Raw)
	assert.Equal(t, auth.CodeUnauthenticated, r.Body["error_code"])

	// The body's mobile is ignored: the code is written for sara's number, nothing for nina's.
	r = e.do(fiber.MethodPost, "/api/v1/auth/send-otp", sara, map[string]any{"purpose": "step_up", "mobile": "09120000602"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = '09120000601' AND verified_at IS NULL`))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = '09120000602'`))
	var code string
	require.NoError(t, e.db.QueryRow(`SELECT code FROM otp_verifications WHERE mobile = '09120000601' AND verified_at IS NULL`).Scan(&code))
	wrong := "0000"
	if code == wrong {
		wrong = "1111"
	}

	// A wrong code is the usual 422 and counts against the number's failure budget (T-M12-01).
	r = e.do(fiber.MethodPost, "/api/v1/auth/verify-otp", sara, map[string]any{"purpose": "step_up", "code": wrong})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	// Step-up verify without a bearer → 401, and the code is not spent.
	r = e.do(fiber.MethodPost, "/api/v1/auth/verify-otp", "", map[string]any{"purpose": "step_up", "code": code})
	require.Equal(t, fiber.StatusUnauthorized, r.Status, r.Raw)

	r = e.do(fiber.MethodPost, "/api/v1/auth/verify-otp", sara, map[string]any{"purpose": "step_up", "code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	stepUp, _ := r.data()["step_up_token"].(string)
	require.NotEmpty(t, stepUp)
	assert.EqualValues(t, 300, r.data()["expires_in"])
	assert.NotContains(t, r.data(), "access_token", "a step-up never logs in")
	assert.NotContains(t, r.Raw, code)

	// The code is spent: it cannot be verified again (neither as step-up nor as a login).
	r = e.do(fiber.MethodPost, "/api/v1/auth/verify-otp", "", map[string]any{"mobile": "09120000601", "code": code})
	require.Equal(t, fiber.StatusBadRequest, r.Status, r.Raw)

	r = e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, stepUp, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assertStepUpRequired(t, e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, stepUp, nil), "single use")
}

func TestStepUp_RequiredOnExportAndDelete(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("sara", "09120000611")
	nina := e.user("nina", "09120000612")
	mona := e.user("mona", "09120000613")
	leila := e.user("leila", "09120000614")

	// Each route has a 3/hour budget per user, so the refusals are spread over accounts.
	for _, rt := range []struct{ method, path string }{
		{fiber.MethodGet, "/api/v1/profile/export"},
		{fiber.MethodDelete, "/api/v1/account"},
	} {
		assertStepUpRequired(t, e.doStep(rt.method, rt.path, sara, "", nil), rt.path+" without a step-up")
		// The login token itself (another purpose) is no step-up.
		assertStepUpRequired(t, e.doStep(rt.method, rt.path, sara, sara, nil), rt.path+" with the login token")
		assertStepUpRequired(t, e.doStep(rt.method, rt.path, sara, "not-a-token", nil), rt.path+" with garbage")
	}
	// Still no bearer → 401 first (the step-up never masks an auth failure).
	r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", "", e.mintStepUp(sara), nil)
	require.Equal(t, fiber.StatusUnauthorized, r.Status)

	// Another account's token is refused — and burnt, so its owner can't use it either.
	ninas := e.mintStepUp(nina)
	assertStepUpRequired(t, e.doStep(fiber.MethodGet, "/api/v1/profile/export", mona, ninas, nil), "another user's token")
	assertStepUpRequired(t, e.doStep(fiber.MethodGet, "/api/v1/profile/export", nina, ninas, nil), "burnt after a wrong presentation")

	// Expired: minted at 10:00, presented at 10:05:01; just inside the window it works.
	assertStepUpRequired(t, e.doStep(fiber.MethodGet, "/api/v1/profile/export", mona, e.mintStepUp(mona), nil, "2026-09-23T10:05:01+03:30"), "expired")
	r = e.doStep(fiber.MethodGet, "/api/v1/profile/export", nina, e.mintStepUp(nina), nil, "2026-09-23T10:04:59+03:30")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)

	// One token per action: the export spends it, the delete after it is refused.
	used := e.mintStepUp(leila)
	require.Equal(t, fiber.StatusOK, e.doStep(fiber.MethodGet, "/api/v1/profile/export", leila, used, nil).Status)
	assertStepUpRequired(t, e.doStep(fiber.MethodDelete, "/api/v1/account", leila, used, nil), "token spent by the export")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM users WHERE id = ?`, e.uid[leila]))

	// With a fresh step-up the delete goes through (200, as Laravel) and the session ends.
	r = e.doStep(fiber.MethodDelete, "/api/v1/account", leila, e.mintStepUp(leila), nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM users WHERE id = ?`, e.uid[leila]))
}

func TestStepUp_ThrottleExportAndDelete(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("sara", "09120000621")
	nina := e.user("nina", "09120000622")

	for i := 1; i <= 3; i++ {
		r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, e.mintStepUp(sara), nil)
		require.Equal(t, fiber.StatusOK, r.Status, "export %d: %s", i, r.Raw)
	}
	// The 4th export within the hour is 429 even with a valid step-up — and that token is not spent.
	fourth := e.mintStepUp(sara)
	r := e.doStep(fiber.MethodGet, "/api/v1/profile/export", sara, fourth, nil)
	require.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	// Per user: nina's export is unaffected.
	require.Equal(t, fiber.StatusOK, e.doStep(fiber.MethodGet, "/api/v1/profile/export", nina, e.mintStepUp(nina), nil).Status)
	// Delete has its own budget: 3 attempts (here without a step-up → 403), the 4th → 429.
	for i := 1; i <= 3; i++ {
		assertStepUpRequired(t, e.doStep(fiber.MethodDelete, "/api/v1/account", sara, "", nil), "delete attempt")
	}
	r = e.doStep(fiber.MethodDelete, "/api/v1/account", sara, fourth, nil)
	require.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM users WHERE id = ?`, e.uid[sara]))
	// Neither throttled request spent the step-up token (the throttle runs first).
	at, err := time.Parse(time.RFC3339, companionNow)
	require.NoError(t, err)
	ok, err := e.stepUpStore().Consume(context.Background(), e.uid[sara], fourth, at)
	require.NoError(t, err)
	assert.True(t, ok, "a throttled request must not spend the step-up token")
}

func TestDocs_WrongPasswordBudget(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)))})
	app.Use(clock.Middleware(clock.Real{}, true))
	mountDocs(app, &config.Config{App: config.App{Env: "production"}, Swagger: config.Swagger{User: "ritme", Password: "s3cret"}},
		cache.NewFromClient(rdb, "ritme-go-docs:"))

	get := func(pass, at string) int {
		req := httptest.NewRequest(fiber.MethodGet, docsPrefix+"/openapi.yaml", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Basic "+base64.StdEncoding.EncodeToString([]byte("ritme:"+pass)))
		req.Header.Set(clock.Header, at)
		resp, err := app.Test(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	const t0 = "2026-09-23T10:00:00+03:30"
	// The right password does not count.
	require.Equal(t, fiber.StatusOK, get("s3cret", t0))
	for i := 1; i <= DocsFailuresPerMinute; i++ {
		require.Equal(t, fiber.StatusUnauthorized, get("wrong", t0), "failure %d", i)
	}
	assert.Equal(t, fiber.StatusTooManyRequests, get("wrong", t0), "11th failure")
	assert.Equal(t, fiber.StatusTooManyRequests, get("s3cret", "2026-09-23T10:00:59+03:30"), "locked: even the right password (no oracle)")
	assert.Equal(t, fiber.StatusOK, get("s3cret", "2026-09-23T10:01:01+03:30"), "the window is over")
}

// The export semaphore refuses at once when every slot is busy (429 export_busy, Retry-After) and frees the slot after.
func TestExportSlots_RefuseWhenFull(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	entered, release := make(chan struct{}), make(chan struct{})
	app.Get("/x", exportSlots(1), func(c fiber.Ctx) error {
		entered <- struct{}{}
		<-release
		return c.SendStatus(fiber.StatusOK)
	})
	get := func() *nethttpResponse {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/x", nil), fiber.TestConfig{Timeout: 10 * time.Second})
		require.NoError(t, err)
		_ = resp.Body.Close()
		return &nethttpResponse{status: resp.StatusCode, retry: resp.Header.Get(fiber.HeaderRetryAfter)}
	}
	first := make(chan *nethttpResponse, 1)
	go func() { first <- get() }()
	<-entered
	busy := get()
	assert.Equal(t, fiber.StatusTooManyRequests, busy.status)
	assert.Equal(t, "30", busy.retry)
	close(release)
	assert.Equal(t, fiber.StatusOK, (<-first).status)
	go func() { <-entered }()
	assert.Equal(t, fiber.StatusOK, get().status, "the slot is free again")
}

type nethttpResponse struct {
	status int
	retry  string
}
