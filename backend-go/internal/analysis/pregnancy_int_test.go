package analysis_test

// GET /api/v1/analysis/pregnancy end to end on MariaDB (B-N3-12): the caller's own pregnancy rows only
// (IDOR), the taxonomy v2 day log and the weekly log merged, Plus sections locked for a free user and open
// during a trial, 409 outside pregnancy mode, the 401 body and the request language.

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/healthlog"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/resources/translations"
)

func setupPregnancy(t *testing.T) *env {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	h := analysis.NewPregnancyHandlers(store.New(db), healthlogstore.New(db), healthlog.NewService(db),
		plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet),
		i18n.NewTranslationStore(translations.FS, ""), languages, clock.Real{})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/analysis/pregnancy", i18n.Middleware(languages), guard, h.Pregnancy)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

func TestAnalysisPregnancy_OwnDataPlusAndStates(t *testing.T) {
	e := setupPregnancy(t)
	a, tokA := e.user(t, "09120000101")
	b, tokB := e.user(t, "09120000102")
	_, tokC := e.user(t, "09120000103") // not pregnant

	// A: LMP 2026-02-20 (week 31 on 2026-09-23), 165 cm, 60 kg logged before the pregnancy.
	e.exec(t, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-02-20', NOW(), NOW())`, a)
	e.exec(t, `INSERT INTO user_profiles (user_id, height, weight, created_at, updated_at) VALUES (?, 165, 61.00, NOW(), NOW())`, a)
	e.entry(t, a, "2026-02-01", "measurements", "weight", "", "", "60.00")
	e.exec(t, `INSERT INTO pregnancy_weekly_logs (user_id, log_date, pregnancy_week, weight, systolic_pressure, diastolic_pressure,
		fasting_blood_sugar, post_meal_blood_sugar, created_at, updated_at)
		VALUES (?, '2026-08-20', 26, 66.50, 118, 76, 88.00, 132.00, NOW(), NOW())`, a)
	e.entry(t, a, "2026-09-20", "measurements", "weight", "", "", "68.40")
	e.entry(t, a, "2026-09-20", "measurements", "bp_systolic", "", "", "121")
	e.entry(t, a, "2026-09-20", "measurements", "bp_diastolic", "", "", "79")
	e.entry(t, a, "2026-09-20", "symptoms", "general", "swelling", "moderate", "")
	e.exec(t, `INSERT INTO pregnancy_symptom_logs (user_id, log_date, has_nausea, nausea_severity, created_at, updated_at)
		VALUES (?, '2026-04-10', 1, 'moderate', NOW(), NOW())`, a)
	e.exec(t, `INSERT INTO pregnancy_fetal_movements (user_id, log_date, pregnancy_week, movement_status, movement_count,
		first_movement_time, last_movement_time, created_at, updated_at)
		VALUES (?, '2026-09-22', 31, 'normal', 10, '20:10', '20:48', NOW(), NOW())`, a)
	e.exec(t, `INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Ultrasound', '2026-10-03 10:00:00', 'none', 1, '{}', NOW(), NOW())`, a)

	// B: pregnant too, with a high reading and a heavy weight that must never leak into A's report.
	e.exec(t, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-05-01', NOW(), NOW())`, b)
	e.entry(t, b, "2026-09-21", "measurements", "weight", "", "", "93.30")
	e.entry(t, b, "2026-09-21", "measurements", "bp_systolic", "", "", "151")
	e.entry(t, b, "2026-09-21", "measurements", "bp_diastolic", "", "", "97")

	r := e.get(t, "/api/v1/analysis/pregnancy", tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.InDelta(t, 31, path(r.data(), "pregnancy", "week"), 0)
	w := path(r.data(), "sections", "weight_gain").(map[string]any)
	assert.Equal(t, true, w["ready"], r.raw)
	assert.InDelta(t, 8.4, path(w, "data", "current", "gain"), 0.001)
	assert.Equal(t, "before_pregnancy", path(w, "data", "baseline", "source"))
	assert.Equal(t, "normal", path(w, "data", "bmi_category"))
	points, _ := path(w, "data", "points").([]any)
	assert.Len(t, points, 2, "the weekly log and the day log both feed the curve")
	assert.Equal(t, "below_threshold", path(r.data(), "sections", "blood_pressure", "data", "status"))
	assert.InDelta(t, 2, path(r.data(), "sections", "blood_pressure", "data", "readings_count"), 0)
	days, _ := path(r.data(), "sections", "kicks", "data", "days").([]any)
	require.Len(t, days, 7)
	assert.InDelta(t, 38, days[5].(map[string]any)["minutes_to_target"], 0, "20:10 → 20:48")
	assert.Equal(t, true, path(r.data(), "sections", "glucose", "locked"))
	assert.Nil(t, path(r.data(), "sections", "glucose", "data"))
	assert.Equal(t, true, path(r.data(), "sections", "symptoms", "locked"))
	assert.Equal(t, "Ultrasound", path(r.data(), "sections", "visits", "data", "next", "title"))
	assert.NotContains(t, r.raw, "93.3", "B's weight never shows in A's report")
	assert.NotContains(t, r.raw, "151")

	// A trial opens the Plus sections.
	e.exec(t, `INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
		VALUES (?, '2026-09-22 10:00:00', '2026-09-29 10:00:00', NOW(), NOW())`, a)
	r = e.get(t, "/api/v1/analysis/pregnancy", tokA, "en")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, false, path(r.data(), "sections", "glucose", "locked"))
	slots, _ := path(r.data(), "sections", "glucose", "data", "slots").([]any)
	require.Len(t, slots, 3)
	assert.InDelta(t, 88, slots[0].(map[string]any)["avg"], 0)
	assert.InDelta(t, 132, slots[1].(map[string]any)["avg"], 0)
	tris, _ := path(r.data(), "sections", "symptoms", "data", "trimesters").([]any)
	require.Len(t, tris, 3)
	first := tris[0].(map[string]any)["items"].([]any)
	require.Len(t, first, 1)
	assert.Equal(t, "Nausea", first[0].(map[string]any)["label"])
	third := tris[2].(map[string]any)["items"].([]any)
	require.Len(t, third, 1)
	assert.Equal(t, "symptoms.general.swelling", third[0].(map[string]any)["key"])

	// B sees B's high reading.
	rb := e.get(t, "/api/v1/analysis/pregnancy", tokB, "")
	require.Equal(t, 200, rb.status, rb.raw)
	assert.Equal(t, "high", path(rb.data(), "sections", "blood_pressure", "data", "status"))
	assert.Equal(t, "baseline", path(rb.data(), "sections", "weight_gain", "data", "missing"))

	// Not pregnant → 409; no token → the 401 body.
	rc := e.get(t, "/api/v1/analysis/pregnancy", tokC, "")
	assert.Equal(t, 409, rc.status, rc.raw)
	assert.Equal(t, "pregnancy_not_active", rc.body["error_code"])
	assert.Equal(t, false, rc.body["success"])
	r0 := e.get(t, "/api/v1/analysis/pregnancy", "", "")
	assert.Equal(t, 401, r0.status)
	assert.Equal(t, "unauthenticated", r0.body["error_code"])
}
