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
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	storage string // STORAGE_PATH of the handlers (support-report screenshots)
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
	storage := t.TempDir()
	h := profile.NewHandlers(profile.Options{DB: db, Logger: quiet, StoragePath: storage})
	ph := profile.NewPrivacyHandlers(profile.PrivacyOptions{Store: profilestore.New(db), StoragePath: storage, Logger: quiet})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/profile", guard.RequireUser, h.Show)
	app.Post("/api/v1/profile", guard.RequireUser, h.Store)
	app.Get("/api/v1/profile/export", guard.RequireUser, h.Export)
	app.Delete("/api/v1/account", guard.RequireUser, h.DestroyAccount)
	app.Get("/api/v1/profile/consents", guard.RequireUser, ph.Consents)
	app.Put("/api/v1/profile/consents", guard.RequireUser, ph.UpdateConsents)
	app.Post("/api/v1/support/reports", guard.RequireUser, ph.CreateSupportReport)
	cs := profile.NewCycleSettingsHandlers(db, clock.Real{}) // B-N1-09
	app.Get("/api/v1/profile/cycle-settings", guard.RequireUser, cs.Show)
	app.Put("/api/v1/profile/cycle-settings", guard.RequireUser, cs.Update)
	ob := profile.NewOnboardingHandlers(db, clock.Real{}) // B-N2-01
	app.Get("/api/v1/onboarding", guard.RequireUser, ob.Show)
	app.Post("/api/v1/onboarding/complete", guard.RequireUser, ob.Complete)
	app.Put("/api/v1/onboarding/steps/:step", guard.RequireUser, ob.UpdateStep)
	app.Get("/api/v1/profile/life-stage", guard.RequireUser, ob.ShowLifeStage)
	app.Put("/api/v1/profile/life-stage", guard.RequireUser, ob.UpdateLifeStage)
	app.Get("/api/v1/profile/life-stage/loss-copy", guard.RequireUser, ob.ShowLossCopy) // B-N2-03
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), storage: storage}
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
	// A refresh token of the user's token (Passport keeps them in a separate table, no FK), and
	// another user's token that must survive.
	var tokID string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM oauth_access_tokens WHERE user_id=?`, id).Scan(&tokID))
	_, err := e.db.Exec(`INSERT INTO oauth_refresh_tokens (id, access_token_id, revoked, expires_at) VALUES ('rt-del', ?, 0, NULL)`, tokID)
	require.NoError(t, err)
	otherID, otherTok := e.user(t, "09120000005")

	r := e.do(t, http.MethodDelete, "/api/v1/account", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
		return n
	}
	assert.Zero(t, count(`SELECT COUNT(*) FROM users WHERE id=?`, id))
	// D-25 (T-M2-34): the tokens are deleted with the user, not left behind revoked.
	assert.Zero(t, count(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id=?`, id))
	assert.Zero(t, count(`SELECT COUNT(*) FROM oauth_refresh_tokens WHERE access_token_id=?`, tokID))
	r = e.do(t, http.MethodGet, "/api/v1/profile", tok, "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "token_revoked", r.body["error_code"])

	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id=? AND revoked=0`, otherID))
	assert.Equal(t, 200, e.do(t, http.MethodGet, "/api/v1/profile", otherTok, "").status)
}

// Review #2 (T-M2-34, D-24): a profile save without last_period_start no longer fabricates a
// confirmed period dated today — neither on a new profile nor on an edit of one without it.
func TestStore_NoLastPeriodStartSeedsNothing(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000006")
	cycles := func() int {
		var n int
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cycle_histories WHERE user_id=?`, id).Scan(&n))
		return n
	}
	lmp := func() sql.NullString {
		var v sql.NullString
		require.NoError(t, e.db.QueryRow(`SELECT last_period_start FROM user_profiles WHERE user_id=?`, id).Scan(&v))
		return v
	}

	r := e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"weight":60}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Nil(t, r.body["data"].(map[string]any)["profile"].(map[string]any)["last_period_start"])
	assert.False(t, lmp().Valid, "new profile: no invented LMP")
	assert.Zero(t, cycles(), "new profile: no cycle history row")

	r = e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"pregnancy_intention":"unsure","height":165}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.False(t, lmp().Valid, "edit: still no LMP")
	assert.Zero(t, cycles())

	r = e.do(t, http.MethodGet, "/api/v1/profile", tok, "")
	require.Equal(t, 200, r.status, r.raw)

	// Logging the period later still seeds the onboarding row.
	r = e.do(t, http.MethodPost, "/api/v1/profile", tok, `{"last_period_start":"2026-09-10"}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "2026-09-10", lmp().String[:10])
	assert.Equal(t, 1, cycles())
}
