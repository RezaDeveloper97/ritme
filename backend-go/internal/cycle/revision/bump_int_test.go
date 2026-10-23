package revision_test

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
	"github.com/ritme/backend-go/internal/cycle/revision"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// T-M9-30: every write request to an engine input moves user_revisions.engine_rev by exactly 1, through the
// production wiring (internal/http.Mount); a refused write (validation, 4xx) and a rolled-back transaction do not.

const (
	keysDir = "../../../contract/fixtures/keys"
	testNow = "2026-09-23T10:00:00+03:30"
)

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
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe48', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))

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
		Cache:  cache.NewFromClient(rdb, "ritme-go-t-m9-30:"),
		Logger: quiet,
	})
	return &env{t: t, db: db, app: app, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

// user creates a woman's account and returns its id and bearer token.
func (e *env) user() (uint64, string) {
	e.t.Helper()
	e.n++
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`,
		"0912300"+strconv.Itoa(1000+e.n))
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(e.t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

type reply struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (r reply) data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

func (e *env) do(method, path, token string, body any) reply {
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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, testNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// rev is the user's engine_rev (0 without a user_revisions row).
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

// bumps runs one request and asserts its status and that it moved engine_rev by want (1 for a write, 0 for a
// refused one). It returns the reply.
func (e *env) bumps(want int, uid uint64, tok string, status int, method, path string, body any) reply {
	e.t.Helper()
	before := e.rev(uid)
	r := e.do(method, path, tok, body)
	require.Equal(e.t, status, r.Status, "%s %s: %s", method, path, r.Raw)
	assert.Equal(e.t, want, e.rev(uid)-before, "%s %s moves engine_rev by %d", method, path, want)
	return r
}

func (e *env) onboarded(tok string, uid uint64) {
	e.t.Helper()
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/onboarding/steps/gender", map[string]any{"gender": "female"})
}

func TestBump_Periods(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2026-09-01"})
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/cycle/period/end", map[string]any{"date": "2026-09-05"})
	r := e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/cycle/period", map[string]any{"start_date": "2026-08-03", "end_date": "2026-08-07"})
	id := int(r.data()["id"].(float64))
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/cycle/period/"+strconv.Itoa(id), map[string]any{"start_date": "2026-08-04", "end_date": "2026-08-08"})
	e.bumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/cycle/period/"+strconv.Itoa(id), nil)

	// Refused writes do not bump: a validation error, an overlap refusal (rolled back inside the transaction), a 404.
	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/cycle/period", map[string]any{"start_date": "not-a-date"})
	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/cycle/period", map[string]any{"start_date": "2026-09-02", "end_date": "2026-09-03"})
	e.bumps(0, uid, tok, 404, fiber.MethodDelete, "/api/v1/cycle/period/999999", nil)
}

func TestBump_HealthLogV1(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	// A log before any period (gap 07 B7: calculation_version only moved with an LMP) still bumps.
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/health-logs", map[string]any{"log_date": "2026-09-10", "mood": "happy"})
	// Flow days: the bleeding-log bridge creates, then grows its period — still one bump per request.
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/health-logs", map[string]any{"log_date": "2026-09-20", "bleeding_intensity": "medium"})
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/health-logs", map[string]any{"log_date": "2026-09-21", "bleeding_intensity": "low"})
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM cycle_histories WHERE user_id = ? AND source = 'bleeding_log'`, uid))
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/health-logs", map[string]any{"log_date": "2026-09-20", "bleeding_intensity": "high"})
	e.bumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/health-logs/2026-09-21", nil)

	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/health-logs", map[string]any{"log_date": "2026-09-20", "bleeding_intensity": "rivers"})
	e.bumps(0, uid, tok, 404, fiber.MethodDelete, "/api/v1/health-logs/2026-01-01", nil)
}

func TestBump_HealthLogV2(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/logs/days/2026-09-20", map[string]any{"categories": map[string]any{
		"bleeding": map[string]any{"flow": "medium"}, "measurements": map[string]any{"lh_test": "positive"},
	}})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/logs/days/2026-09-21", map[string]any{"categories": map[string]any{
		"mood": map[string]any{"moods": []string{"calm"}},
	}})
	e.bumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/logs/days/2026-09-20", nil)
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM cycle_histories WHERE user_id = ?`, uid), "the deleted flow day took its period with it")

	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/logs/days/2026-09-21", map[string]any{"categories": map[string]any{
		"bleeding": map[string]any{"flow": "rivers"},
	}})
}

func TestBump_FertilityDay(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/fertility/days/2026-09-22", map[string]any{"lh": "faint"})
	// fertility_logs and the day log in one transaction: still one bump.
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/fertility/days/2026-09-22", map[string]any{"lh": "positive", "bbt": 36.7})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/fertility/days/2026-09-22", map[string]any{"lh": nil})
	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/fertility/days/2026-09-22", map[string]any{"lh": "glowing"})
}

func TestBump_ProfileSave(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()

	// The first save inserts the profile and stores the declared LMP (SetOnboardingLastPeriodStart): one bump.
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/profile", map[string]any{"last_period_start": "2026-09-01", "cycle_duration": 29, "period_duration": 5})
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/profile", map[string]any{"birthday": "1995-04-01"})
	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/profile", map[string]any{"cycle_duration": "long"})
}

func TestBump_CycleSettings(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	// The switch and the manual lengths (POST /profile's Save) in one transaction: one bump.
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/profile/cycle-settings", map[string]any{"lengths_auto": false, "cycle_length": 31, "period_length": 6})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/profile/cycle-settings", map[string]any{"lengths_auto": true})
	// Reminder settings are not an engine input.
	e.bumps(0, uid, tok, 200, fiber.MethodPut, "/api/v1/profile/cycle-settings", map[string]any{"reminders": map[string]any{"pms": map[string]any{"enabled": false}}})
	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/profile/cycle-settings", map[string]any{"cycle_length": 3})
}

func TestBump_LifeModeSwitch(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	// The goal step writes user_goal (Save) and the life profile: one transaction, one bump.
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/onboarding/steps/goal", map[string]any{"goal": "cycle"})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/onboarding/steps/cycle", map[string]any{"last_period_start": "2026-09-01", "period_duration": 5, "cycle_duration": 28})
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/onboarding/complete", nil)
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/profile/life-stage", map[string]any{"mode": "ttc"})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/profile/life-stage", map[string]any{"mode": "cycle", "track_contraception": true})
	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/profile/life-stage", map[string]any{"mode": "astronaut"})
}

func TestBump_Pregnancy(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/pregnancy/activate", nil)
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/pregnancy/deactivate", nil)
	// v1 onboarding: the profile and its due date in one transaction.
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "lmp", "lmp_date": "2026-08-02"})
	// The v2 Setup screen posts the same endpoint with a scan.
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{
		"age_source": "ultrasound", "ultrasound_date": "2026-09-20", "ultrasound_weeks": 7, "ultrasound_days": 2,
	})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/pregnancy/profile", map[string]any{"rh_factor": "negative"})
	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "guess"})
}

func TestBump_Postpartum(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)
	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "lmp", "lmp_date": "2026-01-02"})

	// Closing the pregnancy, the postpartum profile, the life mode and the legacy goal: one transaction, one bump.
	e.bumps(1, uid, tok, 200, fiber.MethodPost, "/api/v1/postpartum/activate", map[string]any{"birth_date": "2026-09-10"})
	e.bumps(1, uid, tok, 200, fiber.MethodPatch, "/api/v1/postpartum", map[string]any{"breastfeeding": true})
	e.bumps(1, uid, tok, 200, fiber.MethodPatch, "/api/v1/postpartum", map[string]any{"breastfeeding": nil})
	e.bumps(0, uid, tok, 422, fiber.MethodPatch, "/api/v1/postpartum", map[string]any{"breastfeeding": "sometimes"})

	// Without a postpartum profile the breastfeeding update writes nothing and rolls back: no bump.
	other, otherTok := e.user()
	e.onboarded(otherTok, other)
	before := e.rev(other)
	r := e.do(fiber.MethodPatch, "/api/v1/postpartum", otherTok, map[string]any{"breastfeeding": true})
	assert.GreaterOrEqual(t, r.Status, 400, r.Raw)
	assert.Equal(t, before, e.rev(other))
}

func TestBump_Contraception(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/contraception/method", map[string]any{
		"method": "combined_pill", "pack_type": "21_7", "pack_started_on": "2026-09-16", "packs_left": 1,
	})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/contraception/method", map[string]any{"method": "copper_iud", "inserted_on": "2026-01-10", "iud_lifetime_years": 10})
	e.bumps(1, uid, tok, 200, fiber.MethodDelete, "/api/v1/contraception/method", nil)
	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/contraception/method", map[string]any{"method": "wishful_thinking"})
}

func TestBump_IVFAndMenopause(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/ivf/cycles", map[string]any{
		"protocol": "antagonist", "started_on": "2026-09-10", "stage": "stim", "stim_started_on": "2026-09-17",
	})
	e.bumps(0, uid, tok, 422, fiber.MethodPost, "/api/v1/ivf/cycles", map[string]any{"protocol": "nope"})

	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/menopause/profile", map[string]any{"stage": "unsure", "last_period": "2025-08-01", "surgical": false, "hrt": true})
	e.bumps(0, uid, tok, 422, fiber.MethodPut, "/api/v1/menopause/profile", map[string]any{"stage": "teenager"})
}

// A rolled-back transaction leaves the revision where it was: the bump rolls back with the write.
func TestBump_RolledBackTransactionDoesNotBump(t *testing.T) {
	e := newEnv(t)
	uid, _ := e.user()
	ctx := context.Background()

	tx, err := e.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, revision.Bump(ctx, tx, uid))
	require.NoError(t, tx.Rollback())
	assert.Equal(t, 0, e.rev(uid))

	// The owner helper of the profile writes: an error after a write and a bump rolls both back.
	svc := &profile.Service{DB: e.db, Q: profilestore.New(e.db)}
	boom := errors.New("boom")
	err = svc.EngineTx(ctx, uid, func(t2 *profile.Service) error {
		at := sql.NullTime{Time: time.Now(), Valid: true}
		if _, err := t2.Q.InsertProfile(ctx, profilestore.InsertProfileParams{UserID: uid, CreatedAt: at, UpdatedAt: at}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, 0, e.rev(uid))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`, uid))

	// Committed: exactly +1 per transaction, + analysis-only leaves engine_rev alone.
	require.NoError(t, svc.EngineTx(ctx, uid, func(*profile.Service) error { return nil }))
	assert.Equal(t, 1, e.rev(uid))
	tx, err = e.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, revision.BumpAnalysis(ctx, tx, uid))
	require.NoError(t, tx.Commit())
	var engine, analysis int
	require.NoError(t, e.db.QueryRow(`SELECT engine_rev, analysis_rev FROM user_revisions WHERE user_id = ?`, uid).Scan(&engine, &analysis))
	assert.Equal(t, [2]int{1, 2}, [2]int{engine, analysis})
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}
