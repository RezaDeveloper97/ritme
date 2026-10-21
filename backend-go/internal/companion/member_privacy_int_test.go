package companion_test

// T-M9-02 — family privacy hotfix (gap report 05 FAM-H1, M1, M2, M3, H3, L6; SPEC v6 §1.7 never-shared data) through
// the production wiring: the real route registry (internal/http.Mount) on a fresh test database, plus the pregnancy
// v1 handlers with the family-grant reset installed (pregnancy.Handlers.SetFamilyGrants).

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	"github.com/ritme/backend-go/internal/companion"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

const (
	privacyNow  = "2026-09-23T10:00:00+03:30"
	fixtureKeys = "../../contract/fixtures/keys"
)

type privacyEnv struct {
	t    *testing.T
	db   *sql.DB
	app  *fiber.App // internal/http.Mount
	preg *fiber.App // pregnancy v1 with SetFamilyGrants
	iss  *passport.Issuer
}

func newPrivacyEnv(t *testing.T) *privacyEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(fixtureKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe95', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

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
		Cache:  cache.NewFromClient(rdb, "ritme-go-m902:"),
		Logger: logger,
	})

	preg := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	preg.Use(clock.Middleware(clock.Real{}, true))
	guard := auth.NewGuardWith(keys.Public, authstore.New(db), clock.Real{}, logger).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, logger))
	h := pregnancy.NewHandlers(pregstore.New(db))
	h.SetFamilyGrants(db)
	preg.Post("/api/v1/pregnancy/activate", locale, guard, h.Activate)
	preg.Post("/api/v1/pregnancy/deactivate", locale, guard, h.Deactivate)
	preg.Post("/api/v1/pregnancy/onboarding", locale, guard, h.Onboarding)

	return &privacyEnv{t: t, db: db, app: app, preg: preg, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *privacyEnv) exec(q string, args ...any) int64 {
	e.t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	return id
}

func (e *privacyEnv) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// account creates a user with a life profile (gender male = a companion account) and returns (id, token).
func (e *privacyEnv) account(mobile, gender, mode string) (uint64, string) {
	e.t.Helper()
	id := e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, mobile)
	var lm any
	if mode != "" {
		lm = mode
	}
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, ?, ?, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`,
		id, gender, lm)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test id
	require.NoError(e.t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test id
}

// link is an active partner link owner → member with the given grants (updated_at = at).
func (e *privacyEnv) link(owner, member uint64, at string, grants companion.Grants) uint64 {
	e.t.Helper()
	cid := e.exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, display_name, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'active', 'Ali', ?, ?, ?, ?)`, owner, member, at, at, at, at)
	for s, l := range grants {
		e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, cid, string(s), string(l), at, at)
	}
	return uint64(cid) //nolint:gosec // test id
}

func (e *privacyEnv) cyclePeriod(owner uint64, lastStart string) {
	e.t.Helper()
	e.exec(`INSERT INTO user_profiles (user_id, last_period_start, period_duration, cycle_duration, created_at, updated_at)
		VALUES (?, ?, 5, 28, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, owner, lastStart)
}

type privacyReply struct {
	Status int
	Raw    string
	Header string // Cache-Control
	Body   map[string]any
}

func (r privacyReply) data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

func (e *privacyEnv) call(app *fiber.App, method, path, token string, body any) privacyReply {
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
	req.Header.Set(clock.Header, privacyNow)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := privacyReply{Status: resp.StatusCode, Raw: string(raw), Header: resp.Header.Get(fiber.HeaderCacheControl)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (e *privacyEnv) do(method, path, token string, body any) privacyReply {
	e.t.Helper()
	return e.call(e.app, method, path, token, body)
}

func section(link uint64, s string) string {
	return fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", link, s)
}

// TestMember_MoodNeverShared — FAM-H1: a member with the symptoms grant gets the owner's pain and never her mood, on
// the section read and on the companion home.
func TestMember_MoodNeverShared(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095001", "female", "cycle")
	member, tok := e.account("09120095002", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	link := e.link(owner, member, "2026-09-01 09:00:00", companion.Grants{companion.SectionSymptoms: companion.LevelView})
	for _, row := range [][]any{
		{"mood", "moods", "calm", "1"},
		{"mood", "moods", "anxious", "1"},
		{"pain", "location", "abdomen", "mild"},
	} {
		e.exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
			VALUES (?, '2026-09-22', ?, ?, ?, ?, NULL, NULL, 'manual', '2026-09-22 09:00:00', '2026-09-22 09:00:00')`, append([]any{owner}, row...)...)
	}

	for _, path := range []string{section(link, "symptoms"), "/api/v1/companion/home"} {
		r := e.do(fiber.MethodGet, path, tok, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.Contains(t, r.Raw, `"abdomen"`, path)
		for _, hidden := range []string{`"mood"`, `"moods"`, "calm", "anxious"} {
			assert.NotContains(t, r.Raw, hidden, path)
		}
	}
}

// TestMember_MedsAppointments_NoFreeText — FAM-M2: nothing the owner typed for herself (notes, topic, result note,
// prep, specialty, free-text subtitle / dose / unit) reaches a member endpoint.
func TestMember_MedsAppointments_NoFreeText(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095011", "female", "cycle")
	member, tok := e.account("09120095012", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	link := e.link(owner, member, "2026-09-01 09:00:00", companion.Grants{
		companion.SectionMeds: companion.LevelView, companion.SectionAppointments: companion.LevelView})
	e.exec(`INSERT INTO reminders (user_id, type, title, subtitle, notes, recurrence, recurrence_time, is_active, meta, created_at, updated_at)
		VALUES (?, 'medication', 'Iron', 'SENTINEL_SUBTITLE', 'SENTINEL_MED_NOTE', 'daily', '21:00:00', 1, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		owner, `{"v":1,"dose":"SENTINEL_DOSE","unit":"SENTINEL_UNIT","form":"SENTINEL_FORM","times":["21:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true}`)
	e.exec(`INSERT INTO reminders (user_id, type, title, subtitle, notes, scheduled_at, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Checkup', 'SENTINEL_APPT_SUBTITLE', 'SENTINEL_APPT_NOTE', '2026-09-30 10:00:00', 'none', 1, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		owner, `{"v":1,"kind":"in_person","with":"Dr. Rahimi","specialty":"SENTINEL_SPECIALTY","topic":"ultrasound","location":"Clinic A","remind_before":"1d","add_to_calendar":false,"prep":[{"id":"p1","text":"SENTINEL_PREP","done":false}],"status":"scheduled","stage":"result","result_note":"SENTINEL_RESULT"}`)

	sentinels := []string{"SENTINEL", `"notes"`, `"topic"`, "ultrasound", `"result_note"`, `"prep"`, `"specialty"`, `"stage"`, `"subtitle"`}
	for _, path := range []string{section(link, "meds"), section(link, "appointments"), "/api/v1/companion/home"} {
		r := e.do(fiber.MethodGet, path, tok, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		for _, s := range sentinels {
			assert.NotContains(t, r.Raw, s, path)
		}
	}
	// What the member card needs is still there.
	r := e.do(fiber.MethodGet, section(link, "meds"), tok, nil)
	assert.Contains(t, r.Raw, `"title":"Iron"`)
	assert.Contains(t, r.Raw, `"times":["21:00"]`)
	r = e.do(fiber.MethodGet, section(link, "appointments"), tok, nil)
	assert.Contains(t, r.Raw, `"title":"Checkup"`)
	assert.Contains(t, r.Raw, `"with":"Dr. Rahimi"`)
	assert.Contains(t, r.Raw, `"location":"Clinic A"`)
	assert.Contains(t, r.Raw, `"scheduled_at":"2026-09-30 10:00:00"`)
}

// TestMember_PostpartumIsNotPregnancy — FAM-M3: the pregnancy grant does not cover postpartum.
func TestMember_PostpartumIsNotPregnancy(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095021", "female", "postpartum")
	member, tok := e.account("09120095022", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 0, 'lmp', '2025-12-01', '2025-12-10 09:00:00', '2026-09-01 09:00:00')`, owner)
	link := e.link(owner, member, "2026-09-20 09:00:00", companion.Grants{
		companion.SectionPregnancy: companion.LevelView, companion.SectionCycle: companion.LevelView})

	r := e.do(fiber.MethodGet, section(link, "pregnancy"), tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, map[string]any{"is_active": false}, r.data()["data"])
	r = e.do(fiber.MethodGet, section(link, "cycle"), tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cyc, _ := r.data()["data"].(map[string]any)
	assert.Equal(t, false, cyc["has_data"], "postpartum: the neutral cycle view")
	r = e.do(fiber.MethodGet, "/api/v1/companion/home", tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, `"phase":"pregnancy"`)
	assert.NotContains(t, r.Raw, `"is_active":true`)
}

// TestMember_CycleView_NoLateness — FAM-M1: an owner 9 days late shows no days_late and no lateness-derived number.
func TestMember_CycleView_NoLateness(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095031", "female", "cycle")
	member, tok := e.account("09120095032", "male", "")
	e.cyclePeriod(owner, "2026-08-17") // 28-day cycle → expected 2026-09-14, 9 days late on 2026-09-23
	link := e.link(owner, member, "2026-09-01 09:00:00", companion.Grants{companion.SectionCycle: companion.LevelView})

	r := e.do(fiber.MethodGet, section(link, "cycle"), tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cyc, _ := r.data()["data"].(map[string]any)
	assert.Equal(t, true, cyc["has_data"])
	_, hasLate := cyc["days_late"]
	assert.False(t, hasLate, "no days_late key")
	assert.Nil(t, cyc["cycle_day"], "a cycle day past the length counts the lateness")
	assert.Nil(t, cyc["days_to_period"])
	assert.Nil(t, cyc["predicted_next_period_start"], "a past prediction counts the lateness")
	r = e.do(fiber.MethodGet, "/api/v1/companion/home", tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "days_late")
	assert.NotContains(t, r.Raw, `"cycle_day":37`)
}

// resetChecks asserts the pregnancy grant of link is gone (other grants kept), with one grants_changed audit row on the
// pregnancy section and one owner notice carrying the diff.
func (e *privacyEnv) resetChecks(owner, link uint64, reason companion.ResetReason, old string) {
	e.t.Helper()
	assert.Zero(e.t, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, link), "grant → none")
	assert.Equal(e.t, 1, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'cycle'`, link), "other grants kept")
	assert.Equal(e.t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND companion_id = ? AND section = 'pregnancy'
		AND action = 'grants_changed' AND actor_id = ?`, owner, link, owner), "audit row")
	var data, title string
	require.NoError(e.t, e.db.QueryRow(`SELECT data, title FROM user_notifications WHERE user_id = ? AND type = 'companion'
		AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = ?`, owner, companion.EventPregnancySharingReset).Scan(&data, &title))
	var d struct {
		CompanionID uint64              `json:"companion_id"`
		Reason      string              `json:"reason"`
		Diff        map[string][]string `json:"diff"`
	}
	require.NoError(e.t, json.Unmarshal([]byte(data), &d))
	assert.Equal(e.t, link, d.CompanionID)
	assert.Equal(e.t, string(reason), d.Reason)
	assert.Equal(e.t, map[string][]string{"pregnancy": {old, "none"}}, d.Diff)
	assert.Contains(e.t, title, "اشتراک بارداری با همراهت متوقف شد؛ اگر بخواهی، دوباره روشنش کن.")
}

func (e *privacyEnv) pregnancyLink(owner, member uint64, level companion.Level) uint64 {
	return e.link(owner, member, "2026-03-01 09:00:00", companion.Grants{companion.SectionPregnancy: level, companion.SectionCycle: companion.LevelView})
}

func TestPregnancyGrant_ResetOnActivate(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095041", "female", "cycle")
	member, mtok := e.account("09120095042", "male", "")
	link := e.pregnancyLink(owner, member, companion.LevelView)

	r := e.call(e.preg, fiber.MethodPost, "/api/v1/pregnancy/activate", otok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	e.resetChecks(owner, link, companion.ResetPregnancyActivated, "view")
	r = e.do(fiber.MethodGet, section(link, "pregnancy"), mtok, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, "the new pregnancy is not shared from its first day")

	// A grant the owner gives during this pregnancy survives a repeated activate / the setup (onboarding) that follows.
	e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'pregnancy', 'view', NOW(), NOW())`, link)
	require.Equal(t, fiber.StatusOK, e.call(e.preg, fiber.MethodPost, "/api/v1/pregnancy/activate", otok, nil).Status)
	r = e.call(e.preg, fiber.MethodPost, "/api/v1/pregnancy/onboarding", otok, map[string]any{"age_source": "lmp", "lmp_date": "2026-08-01"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, link))
}

func TestPregnancyGrant_ResetOnSetup(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095046", "female", "cycle")
	member, _ := e.account("09120095047", "male", "")
	link := e.pregnancyLink(owner, member, companion.LevelEdit)
	r := e.call(e.preg, fiber.MethodPost, "/api/v1/pregnancy/onboarding", otok, map[string]any{"age_source": "lmp", "lmp_date": "2026-08-01"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	e.resetChecks(owner, link, companion.ResetPregnancyActivated, "edit")
}

func TestPregnancyGrant_ResetOnDeactivate(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095051", "female", "cycle")
	member, _ := e.account("09120095052", "male", "")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', '2026-08-20 09:00:00', '2026-08-20 09:00:00')`, owner)
	link := e.pregnancyLink(owner, member, companion.LevelView)

	r := e.call(e.preg, fiber.MethodPost, "/api/v1/pregnancy/deactivate", otok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	e.resetChecks(owner, link, companion.ResetPregnancyDeactivated, "view")
}

func TestPregnancyGrant_ResetOnLoss(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095061", "female", "pregnancy")
	member, mtok := e.account("09120095062", "male", "")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', '2026-08-20 09:00:00', '2026-08-20 09:00:00')`, owner)
	link := e.pregnancyLink(owner, member, companion.LevelView)
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, section(link, "pregnancy"), mtok, nil).Status)

	r := e.do(fiber.MethodPost, "/api/v1/loss", otok, map[string]any{"type": "unspecified", "notify_companion": true})
	require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated}, r.Status, r.Raw)
	e.resetChecks(owner, link, companion.ResetPregnancyLoss, "view")
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, section(link, "pregnancy"), mtok, nil).Status)
}

func TestPregnancyGrant_ResetOnPostpartumStart(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095071", "female", "pregnancy")
	member, mtok := e.account("09120095072", "male", "")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2025-12-20', '2026-01-20 09:00:00', '2026-01-20 09:00:00')`, owner)
	link := e.pregnancyLink(owner, member, companion.LevelView)

	r := e.do(fiber.MethodPost, "/api/v1/postpartum/activate", otok, map[string]any{"birth_date": "2026-09-20"})
	require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated}, r.Status, r.Raw)
	e.resetChecks(owner, link, companion.ResetPostpartumStarted, "view")
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, section(link, "pregnancy"), mtok, nil).Status)
}

// ResetModeScopedGrants runs inside the caller's transaction: a rollback keeps the grant (and writes nothing).
func TestResetModeScopedGrants_InCallerTransaction(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095081", "female", "pregnancy")
	member, _ := e.account("09120095082", "male", "")
	link := e.pregnancyLink(owner, member, companion.LevelView)
	ctx := context.Background()
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)

	tx, err := e.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	n, err := companion.ResetModeScopedGrants(ctx, tx, owner, companion.ResetPregnancyLoss, at)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.NoError(t, tx.Rollback())
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, link))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, owner))

	n, err = companion.ResetModeScopedGrants(ctx, e.db, owner, companion.ResetPregnancyLoss, at)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = companion.ResetModeScopedGrants(ctx, e.db, owner, companion.ResetPregnancyLoss, at)
	require.NoError(t, err)
	assert.Zero(t, n, "idempotent: nothing left to reset, no second notice")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, owner))
}

// TestShopChecklist_StaleGrantDenied — the layette share uses the same pregnancy check: a reset grant, or an owner who is
// not pregnant now (postpartum), shows no due date or weeks left to a member.
func TestShopChecklist_StaleGrantDenied(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095091", "female", "pregnancy")
	member, mtok := e.account("09120095092", "male", "")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', '2026-08-20 09:00:00', '2026-08-20 09:00:00')`, owner)
	link := e.pregnancyLink(owner, member, companion.LevelView)
	path := fmt.Sprintf("/api/v1/shop/checklist?for_user_id=%d", owner)

	r := e.do(fiber.MethodGet, path, mtok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotNil(t, r.data()["due_date"], "a live grant during the pregnancy shows the due date")

	// The pregnancy ends in a loss: the grant is reset, the share is closed.
	require.Contains(t, []int{fiber.StatusOK, fiber.StatusCreated},
		e.do(fiber.MethodPost, "/api/v1/loss", otok, map[string]any{"type": "unspecified"}).Status)
	r = e.do(fiber.MethodGet, path, mtok, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "due_date")
	assert.NotContains(t, r.Raw, "weeks_left")

	// A pregnancy grant that still exists on an owner who is not pregnant (postpartum) opens nothing either.
	e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'pregnancy', 'view', NOW(), NOW())`, link)
	_, err := e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'postpartum' WHERE user_id = ?`, owner)
	require.NoError(t, err)
	r = e.do(fiber.MethodGet, path, mtok, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "due_date")
}

// TestMemberResponses_NoStore — FAM-L6: every member-facing companion response is `Cache-Control: private, no-store`,
// errors included.
func TestMemberResponses_NoStore(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, _ := e.account("09120095101", "female", "pregnancy")
	member, mtok := e.account("09120095102", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', '2026-08-20 09:00:00', '2026-08-20 09:00:00')`, owner)
	all := companion.Grants{}
	for _, s := range companion.Sections {
		all[s] = companion.LevelEdit
	}
	link := e.link(owner, member, "2026-09-01 09:00:00", all)
	med := e.exec(`INSERT INTO reminders (user_id, type, title, recurrence, recurrence_time, is_active, created_at, updated_at)
		VALUES (?, 'medication', 'Iron', 'daily', '21:00:00', 1, NOW(), NOW())`, owner)

	for _, c := range []struct{ method, path string }{
		{fiber.MethodGet, "/api/v1/companions/links"},
		{fiber.MethodGet, "/api/v1/companion/home"},
		{fiber.MethodGet, section(link, "cycle")},
		{fiber.MethodGet, section(link, "symptoms")},
		{fiber.MethodGet, section(link, "meds")},
		{fiber.MethodGet, section(link, "appointments")},
		{fiber.MethodGet, section(link, "pregnancy")},
		{fiber.MethodGet, section(link+999, "cycle")}, // 404 — errors carry it too
		{fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", med, owner)},
		{fiber.MethodGet, fmt.Sprintf("/api/v1/shop/checklist?for_user_id=%d", owner)},
		{fiber.MethodPost, "/api/v1/companions/accept"}, // 422 invalid code
		{fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/links/%d", link)},
	} {
		var body any
		if c.method == fiber.MethodPost {
			body = map[string]any{"code": "ZZZZZZ"}
		}
		r := e.do(c.method, c.path, mtok, body)
		assert.Less(t, r.Status, 500, "%s %s: %s", c.method, c.path, r.Raw)
		assert.Equal(t, companion.CacheControlNoStore, r.Header, "%s %s (%d)", c.method, c.path, r.Status)
	}
}
