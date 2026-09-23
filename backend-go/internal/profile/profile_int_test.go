package profile_test

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
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/profile"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet)
	h := profile.NewHandlers(profile.Options{DB: db, Logger: quiet})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/profile", guard.RequireUser, h.Show)
	app.Post("/api/v1/profile", guard.RequireUser, h.Store)
	app.Get("/api/v1/profile/export", guard.RequireUser, h.Export)
	app.Delete("/api/v1/account", guard.RequireUser, h.DestroyAccount)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (e *env) do(t *testing.T, method, path, token, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func TestStore_ConcurrentUpdatesIncrementVersionTwice(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"cycle_duration":28,"period_duration":5,"last_period_start":"2026-09-10"}`)
	require.Equal(t, 200, r.status, r.raw)
	var before int
	require.NoError(t, e.db.QueryRow(`SELECT calculation_version FROM user_profiles WHERE user_id=?`, id).Scan(&before))
	require.Equal(t, 1, before)

	var wg sync.WaitGroup
	for _, body := range []string{`{"cycle_duration":30}`, `{"cycle_duration":31}`} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := e.do(t, http.MethodPost, "/api/v1/profile", tok, body)
			assert.Equal(t, 200, r.status, r.raw)
		}()
	}
	wg.Wait()

	var after int
	var status string
	require.NoError(t, e.db.QueryRow(`SELECT calculation_version, calculation_status FROM user_profiles WHERE user_id=?`, id).Scan(&after, &status))
	assert.Equal(t, before+2, after)
	assert.Equal(t, "completed", status)
}

func TestStore_OnboardingSeedsAndResyncsPeriod(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000002")
	r := e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"period_duration":5,"last_period_start":"2026-09-10"}`)
	require.Equal(t, 200, r.status, r.raw)

	row := func() (start, end string, bleeding sql.NullInt64, confirmed, estimated bool, source string, n int) {
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cycle_histories WHERE user_id=?`, id).Scan(&n))
		var endN sql.NullString
		require.NoError(t, e.db.QueryRow(`SELECT DATE_FORMAT(period_start_date,'%Y-%m-%d'), DATE_FORMAT(period_end_date,'%Y-%m-%d'),
			bleeding_length, is_confirmed, is_estimated, source FROM cycle_histories WHERE user_id=?`, id).
			Scan(&start, &endN, &bleeding, &confirmed, &estimated, &source))
		return start, endN.String, bleeding, confirmed, estimated, source, n
	}
	start, end, bleeding, confirmed, estimated, source, n := row()
	assert.Equal(t, 1, n)
	assert.Equal(t, "2026-09-10", start)
	assert.Equal(t, "2026-09-14", end)
	assert.Equal(t, int64(5), bleeding.Int64)
	assert.True(t, confirmed)
	assert.False(t, estimated)
	assert.Equal(t, "user_profile_confirmed", source)

	// Editing the declared LMP re-syncs the same row; an end in the future leaves it open.
	r = e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"last_period_start":"2026-09-21"}`)
	require.Equal(t, 200, r.status, r.raw)
	start, end, bleeding, _, _, _, n = row()
	assert.Equal(t, 1, n)
	assert.Equal(t, "2026-09-21", start)
	assert.Empty(t, end)
	assert.False(t, bleeding.Valid)
}

func TestStore_NullUserGoalIsA500WithNoRow(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000003")
	r := e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"user_goal":null}`)
	require.Equal(t, 500, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_profiles WHERE user_id=?`, id).Scan(&n))
	assert.Zero(t, n, "a rejected INSERT leaves no profile")
}

func TestDestroyAccount_RevokesTokensAndDeletes(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000004")
	require.Equal(t, 200, e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"weight":60}`).status)
	r := e.do(t, http.MethodDelete, "/api/v1/account", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	var users, live int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=?`, id).Scan(&users))
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id=? AND revoked=0`, id).Scan(&live))
	assert.Zero(t, users)
	assert.Zero(t, live)
	assert.Equal(t, 401, e.do(t, http.MethodGet, "/api/v1/profile", tok, "").status)
}
