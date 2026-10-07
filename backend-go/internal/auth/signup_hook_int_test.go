package auth_test

import (
	"context"
	"errors"
	"net/http"
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

type hookCall struct {
	userID uint64
	mobile string
}

// setupWithHooks is setup() plus signup hooks (B-N8-01).
func setupWithHooks(t *testing.T, hooks ...auth.SignupHook) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	q := store.New(db)
	rec := &recorder{}
	h := auth.NewHandlers(q, passport.NewIssuer(testKey, q, clock.Real{}, 365), rec, clock.Real{}, 30, quiet)
	for _, hk := range hooks {
		h.OnSignup(hk)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/auth/send-otp", h.SendOTP)
	app.Post("/api/v1/auth/verify-otp", h.VerifyOTP)
	return &env{db: db, app: app, jobs: rec}
}

func TestSignupHook_RunsOnceForNewAccountsOnly(t *testing.T) {
	var mu sync.Mutex
	var calls []hookCall
	e := setupWithHooks(t, func(_ context.Context, userID uint64, mobile string, _ time.Time) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, hookCall{userID, mobile})
		return nil
	})
	e.login(t, "09121230001", t0)
	var id uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM users WHERE mobile = '09121230001'").Scan(&id))
	require.Equal(t, []hookCall{{id, "09121230001"}}, calls)

	e.login(t, "09121230001", t1) // returning user: no hook
	assert.Len(t, calls, 1)
}

func TestSignupHook_FailureOrPanicNeverBreaksLogin(t *testing.T) {
	ran := 0
	e := setupWithHooks(t,
		func(context.Context, uint64, string, time.Time) error { ran++; return errors.New("boom") },
		func(context.Context, uint64, string, time.Time) error { ran++; panic("hook panic") },
		func(ctx context.Context, _ uint64, _ string, _ time.Time) error {
			ran++
			_, ok := ctx.Deadline()
			assert.True(t, ok, "hooks run under a deadline")
			return nil
		},
	)
	r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09121230002"}`, t0)
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"09121230002","code":"`+e.otp(t, "09121230002")+`"}`, t0)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, r.body["data"].(map[string]any)["new_user"])
	assert.NotEmpty(t, r.body["data"].(map[string]any)["access_token"])
	assert.Equal(t, 3, ran)
}
