package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var (
	quiet   = slog.New(slog.NewTextHandler(io.Discard, nil))
	testKey = func() *rsa.PrivateKey {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		return k
	}()
)

type recorder struct {
	mu   sync.Mutex
	jobs []auth.SendOTPJob
}

func (r *recorder) Dispatch(_ context.Context, _ string, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, payload.(auth.SendOTPJob))
	return nil
}

type env struct {
	db   *sql.DB
	app  *fiber.App
	jobs *recorder
}

// setup builds the auth routes on a fresh DB. tokenClock drives JWT time validation and
// issuing (the "real" clock in production); requests carry X-Test-Now for the Carbon clock.
func setup(t *testing.T, tokenClock clock.Clock) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	q := store.New(db)
	rec := &recorder{}
	issuer := passport.NewIssuer(testKey, q, tokenClock, 365)
	h := auth.NewHandlers(q, issuer, rec, clock.Real{}, 30, quiet)
	guard := auth.NewGuardWith(&testKey.PublicKey, q, tokenClock, quiet)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/auth/send-otp", h.SendOTP)
	app.Post("/api/v1/auth/verify-otp", h.VerifyOTP)
	app.Post("/api/v1/auth/logout", guard.RequireUser, h.Logout)
	app.Get("/api/v1/auth/user", guard.RequireUser, h.User)
	app.Post("/api/v1/auth/refresh-session", guard.RequireUser, h.RefreshSession)
	return &env{db: db, app: app, jobs: rec}
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (e *env) do(t *testing.T, method, path, token, body, now string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if now != "" {
		req.Header.Set(clock.Header, now)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func (e *env) otp(t *testing.T, mobile string) string {
	t.Helper()
	var code string
	require.NoError(t, e.db.QueryRow("SELECT code FROM otp_verifications WHERE mobile=? ORDER BY id DESC LIMIT 1", mobile).Scan(&code))
	return code
}

// login sends and verifies an OTP at request time now ("" = real time).
func (e *env) login(t *testing.T, mobile, now string) string {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"`+mobile+`"}`, now)
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"`+mobile+`","code":"`+e.otp(t, mobile)+`"}`, now)
	require.Equal(t, 200, r.status, r.raw)
	return r.body["data"].(map[string]any)["access_token"].(string)
}

const (
	t0 = "2026-09-23T10:00:00+03:30"
	t1 = "2026-09-23T10:01:01+03:30" // after the 60 s resend window
)

func TestOTP_ParallelVerifyCannotExceedAttemptCap(t *testing.T) {
	e := setup(t, clock.Real{})
	r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121234567"}`, t0)
	require.Equal(t, 200, r.status, r.raw)
	code := e.otp(t, "09121234567")
	wrong := "0000"
	if code == wrong {
		wrong = "0001"
	}

	const n = 30
	statuses := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "",
				`{"mobile":"09121234567","code":"`+wrong+`"}`, t0).status
		}()
	}
	wg.Wait()
	close(statuses)
	count := map[int]int{}
	for s := range statuses {
		count[s]++
	}
	assert.Equal(t, map[int]int{422: auth.OTPMaxAttempts, 429: n - auth.OTPMaxAttempts}, count)

	var attempts int
	require.NoError(t, e.db.QueryRow("SELECT attempts FROM otp_verifications WHERE mobile='09121234567'").Scan(&attempts))
	assert.Equal(t, auth.OTPMaxAttempts, attempts)

	// Even the right code is refused once the cap is reached.
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"09121234567","code":"`+code+`"}`, t0)
	assert.Equal(t, 429, r.status)
	assert.Equal(t, "Too many attempts. Please request a new OTP.", r.body["message"])
}

func TestOTP_SendResendWindowAndJob(t *testing.T) {
	e := setup(t, clock.Real{})
	r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121234567"}`, t0)
	require.Equal(t, 200, r.status)
	assert.JSONEq(t, `{"success":true,"message":"OTP sent successfully","data":{"expires_in":120}}`, r.raw)
	require.Len(t, e.jobs.jobs, 1)
	assert.Equal(t, auth.SendOTPJob{Mobile: "09121234567", Code: e.otp(t, "09121234567"), Template: "login_otp"}, e.jobs.jobs[0])
	assert.Regexp(t, `^[1-9][0-9]{3}$`, e.jobs.jobs[0].Code)

	r = e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121234567"}`, "2026-09-23T10:00:20.5+03:30")
	assert.Equal(t, 429, r.status)
	assert.Equal(t, `{"success":false,"message":"Please wait before requesting a new OTP","data":{"retry_after":80.5}}`, r.raw)

	r = e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121234567"}`, "2026-09-23T10:01:01+03:30")
	assert.Equal(t, 200, r.status)
	var rows int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM otp_verifications WHERE mobile='09121234567'").Scan(&rows))
	assert.Equal(t, 1, rows, "old OTPs are deleted")
}

func TestLoginFlow_UserLogoutRevoked(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234567", t0)

	r := e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	user := r.body["data"].(map[string]any)["user"].(map[string]any)
	assert.Equal(t, "09121234567", user["mobile"])
	assert.Nil(t, user["mobile_verified_at"], "not fillable in Laravel → never set")
	assert.NotContains(t, r.raw, "password")
	assert.Equal(t, false, r.body["data"].(map[string]any)["profile_completed"])

	// Second login: existing user.
	tok2 := e.login(t, "09121234567", t1)
	assert.NotEqual(t, tok, tok2)

	r = e.do(t, http.MethodPost, "/api/v1/auth/logout", tok, "", "")
	assert.Equal(t, 200, r.status)
	assert.Equal(t, `{"success":true,"message":"Successfully logged out"}`, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, `{"message":"Unauthenticated.","error_code":"token_revoked"}`, r.raw)
	// The other session is untouched.
	assert.Equal(t, 200, e.do(t, http.MethodGet, "/api/v1/auth/user", tok2, "", "").status)
}

func TestExpiredToken_TestClock(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234567", t0)

	// Same token, verified by a guard whose clock is 365 days later.
	later := auth.NewGuardWith(&testKey.PublicKey, store.New(e.db), clock.At(time.Now().AddDate(0, 0, 366)), quiet)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Get("/u", later.RequireUser, func(c fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest(http.MethodGet, "/u", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, `{"message":"Unauthenticated.","error_code":"token_expired"}`, string(raw))
}

func TestBlockedUser_403AndRevokeAll(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234567", t0)
	var id uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM users WHERE mobile='09121234567'").Scan(&id))

	// Admin block = blocked_at + revoke every token.
	_, err := e.db.Exec("UPDATE users SET blocked_at='2026-09-23 09:00:00' WHERE id=?", id)
	require.NoError(t, err)
	n, err := auth.RevokeUserTokens(context.Background(), store.New(e.db), id, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	r := e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "")
	assert.Equal(t, "token_revoked", r.body["error_code"])

	e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121234567"}`, t1)
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"09121234567","code":"`+e.otp(t, "09121234567")+`"}`, t1)
	assert.Equal(t, 403, r.status)
	// json_encode escapes non-ASCII by default, like Laravel's response()->json().
	assert.Equal(t, `{"success":false,"message":"\u062d\u0633\u0627\u0628 \u06a9\u0627\u0631\u0628\u0631\u06cc `+
		`\u0634\u0645\u0627 \u0645\u0633\u062f\u0648\u062f \u0634\u062f\u0647 \u0627\u0633\u062a."}`, r.raw)
}

func TestRefreshSession_WindowAndIssueThenRevoke(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234567", t0)

	// 365 days left → nothing happens.
	r := e.do(t, http.MethodPost, "/api/v1/auth/refresh-session", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, false, data["refreshed"])
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+03:30$`, data["expires_at"])
	assert.Equal(t, 200, e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "").status)

	// 20 days before expiry (request clock) → a new 365-day token, the old one revoked.
	var exp time.Time
	require.NoError(t, e.db.QueryRow("SELECT expires_at FROM oauth_access_tokens ORDER BY created_at DESC LIMIT 1").Scan(&exp))
	soon := exp.AddDate(0, 0, -20).Format(time.RFC3339)
	r = e.do(t, http.MethodPost, "/api/v1/auth/refresh-session", tok, "", soon)
	require.Equal(t, 200, r.status, r.raw)
	data = r.body["data"].(map[string]any)
	assert.Equal(t, true, data["refreshed"])
	assert.Equal(t, "Bearer", data["token_type"])
	newTok := data["access_token"].(string)

	assert.Equal(t, "token_revoked", e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "").body["error_code"])
	assert.Equal(t, 200, e.do(t, http.MethodGet, "/api/v1/auth/user", newTok, "", "").status)

	var newExp time.Time
	var name, scopes string
	require.NoError(t, e.db.QueryRow("SELECT expires_at, name, scopes FROM oauth_access_tokens WHERE revoked=0").Scan(&newExp, &name, &scopes))
	assert.Equal(t, "auth_token", name)
	assert.Equal(t, "[]", scopes)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 365), newExp, 5*time.Second)
}
