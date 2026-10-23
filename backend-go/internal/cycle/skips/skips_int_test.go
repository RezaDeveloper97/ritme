package skips_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/store"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const keysDir = "../../../contract/fixtures/keys"

type env struct {
	t   *testing.T
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
	n   int
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe29', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-t-m9-29:"),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})
	return &env{t: t, db: db, app: app, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

// user creates an account with three logged 28-day cycles (last start 08-26 → next predicted 09-23).
func (e *env) user() (uint64, string) {
	e.t.Helper()
	e.n++
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`,
		"0912290"+strconv.Itoa(1000+e.n))
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(e.t, err)
	for _, p := range [][2]string{{"2026-07-01", "2026-07-05"}, {"2026-07-29", "2026-08-02"}, {"2026-08-26", "2026-08-30"}} {
		r := e.do(fiber.MethodPost, "/api/v1/cycle/period", tok.AccessToken, "2026-09-01", map[string]any{"start_date": p[0], "end_date": p[1]})
		require.Less(e.t, r.Status, 300, r.Raw)
	}
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

type reply struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (e *env) do(method, path, token, today string, body any) reply {
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
	req.Header.Set(clock.Header, today+"T10:00:00+03:30")
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (e *env) skip(tok, today, expected string) reply {
	return e.do(fiber.MethodPost, "/api/v1/cycle/skips", tok, today, map[string]any{"expected_start": expected})
}

func (e *env) rev(uid uint64) int {
	e.t.Helper()
	var n int
	err := e.db.QueryRow(`SELECT engine_rev FROM user_revisions WHERE user_id = ?`, uid).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	require.NoError(e.t, err)
	return n
}

// status is the v6 engine status of uid for today, read from the database like the endpoint does.
func (e *env) status(uid uint64, today string) engine.Status {
	e.t.Helper()
	d := civildate.MustParse(today)
	in, err := inputs.Load(context.Background(), store.New(e.db), uid, d)
	require.NoError(e.t, err)
	return engine.Prepare(in, engineparams.For(d)).Resolve(engine.DayOf(d), engine.DayOf(d))
}

func (e *env) refused(r reply, code string) {
	e.t.Helper()
	require.Equal(e.t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(e.t, false, r.Body["success"])
	assert.Equal(e.t, code, r.Body["code"], r.Raw)
	assert.NotEmpty(e.t, r.Body["message"])
}

// Manifest test for cycle.InsertCycleSkip / cycle.DeleteCycleSkip (T-M9-30): each successful write bumps engine_rev
// by exactly 1; a repeat, a refusal and an unknown delete bump nothing.
func TestBump_CycleSkips(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	rev := e.rev(uid)

	r := e.skip(tok, "2026-09-23", "2026-09-23")
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.JSONEq(t, `{"success":true,"data":{"expected_start":"2026-09-23"}}`, r.Raw)
	assert.Equal(t, rev+1, e.rev(uid))

	r = e.skip(tok, "2026-09-23", "2026-09-23")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.JSONEq(t, `{"success":true,"data":{"expected_start":"2026-09-23"}}`, r.Raw)
	assert.Equal(t, rev+1, e.rev(uid), "an idempotent repeat does not bump")

	e.refused(e.skip(tok, "2026-09-23", "2026-09-22"), "prediction_mismatch")
	assert.Equal(t, rev+1, e.rev(uid), "a refusal does not bump")

	r = e.do(fiber.MethodDelete, "/api/v1/cycle/skips/2026-09-23", tok, "2026-09-23", nil)
	require.Equal(t, fiber.StatusNoContent, r.Status, r.Raw)
	assert.Empty(t, r.Raw)
	assert.Equal(t, rev+2, e.rev(uid))

	r = e.do(fiber.MethodDelete, "/api/v1/cycle/skips/2026-09-23", tok, "2026-09-23", nil)
	require.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
	assert.Equal(t, rev+2, e.rev(uid), "an unknown delete does not bump")
}

func TestSkips_AcceptAndRefuse(t *testing.T) {
	e := newEnv(t)

	t.Run("15 days late, then delete restores the late output", func(t *testing.T) {
		uid, tok := e.user()
		before := e.status(uid, "2026-10-08")
		require.Equal(t, 15, before.DaysLate.Or(-1))

		r := e.skip(tok, "2026-10-08", "2026-09-23")
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		after := e.status(uid, "2026-10-08")
		assert.True(t, after.Warnings.Has(engine.WarnSkippedCycleRecorded))
		assert.Equal(t, engine.DayOf(civildate.MustParse("2026-10-21")), after.Anchors.PredictedNextPeriodStart)
		assert.Equal(t, 0, after.DaysLate.Or(-1))

		r = e.do(fiber.MethodDelete, "/api/v1/cycle/skips/2026-09-23", tok, "2026-10-08", nil)
		require.Equal(t, fiber.StatusNoContent, r.Status, r.Raw)
		assert.Equal(t, before, e.status(uid, "2026-10-08"))
	})

	t.Run("before the expected day", func(t *testing.T) {
		_, tok := e.user()
		e.refused(e.skip(tok, "2026-09-20", "2026-09-23"), "not_expected_yet")
	})

	t.Run("after a real start", func(t *testing.T) {
		_, tok := e.user()
		r := e.do(fiber.MethodPost, "/api/v1/cycle/period/start", tok, "2026-09-26", map[string]any{"date": "2026-09-25"})
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		e.refused(e.skip(tok, "2026-09-26", "2026-09-23"), "period_started")
	})

	t.Run("wrong date", func(t *testing.T) {
		_, tok := e.user()
		e.refused(e.skip(tok, "2026-09-25", "2026-09-24"), "prediction_mismatch")
	})

	t.Run("validation is a 422, never a 401", func(t *testing.T) {
		_, tok := e.user()
		for _, body := range []map[string]any{{}, {"expected_start": "23-09-2026"}, {"expected_start": 5}} {
			r := e.do(fiber.MethodPost, "/api/v1/cycle/skips", tok, "2026-09-23", body)
			require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
			assert.Contains(t, r.Body["errors"], "expected_start", r.Raw)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := e.skip("", "2026-09-23", "2026-09-23")
		require.Equal(t, fiber.StatusUnauthorized, r.Status, r.Raw)
		assert.Equal(t, "unauthenticated", r.Body["error_code"])
		r = e.do(fiber.MethodDelete, "/api/v1/cycle/skips/2026-09-23", "", "2026-09-23", nil)
		require.Equal(t, fiber.StatusUnauthorized, r.Status, r.Raw)
	})

	t.Run("another user's skip is a 404 and stays", func(t *testing.T) {
		ownerID, owner := e.user()
		_, other := e.user()
		require.Equal(t, fiber.StatusCreated, e.skip(owner, "2026-09-24", "2026-09-23").Status)
		rev := e.rev(ownerID)

		r := e.do(fiber.MethodDelete, "/api/v1/cycle/skips/2026-09-23", other, "2026-09-24", nil)
		require.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
		assert.Equal(t, false, r.Body["success"])
		assert.Equal(t, rev, e.rev(ownerID))
		assert.True(t, e.status(ownerID, "2026-09-24").Warnings.Has(engine.WarnSkippedCycleRecorded))
	})

	t.Run("not a date is a 404", func(t *testing.T) {
		_, tok := e.user()
		r := e.do(fiber.MethodDelete, "/api/v1/cycle/skips/yesterday", tok, "2026-09-23", nil)
		require.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
	})
}

// Concurrent taps of the same skip: one row, one bump, every reply 200/201.
func TestSkips_ConcurrentSameSkip(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	rev := e.rev(uid)
	const n = 8
	codes := make(chan int, n)
	for range n {
		go func() { codes <- e.skip(tok, "2026-09-23", "2026-09-23").Status }()
	}
	created := 0
	for range n {
		c := <-codes
		require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated}, c)
		if c == fiber.StatusCreated {
			created++
		}
	}
	assert.Equal(t, 1, created)
	assert.Equal(t, rev+1, e.rev(uid))
	var rows int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cycle_skips WHERE user_id = ?`, uid).Scan(&rows))
	assert.Equal(t, 1, rows)
}
