package analysis_test

// GET /api/v1/analysis/postpartum end to end on MariaDB (T-M10-27): a birth 6 weeks ago with two EPDS checks, lochia,
// feeds, sleep, weights and a measured child → every card ready; a free user gets the EPDS card and locked Plus cards;
// one EPDS score → no trend; the caller's own rows only (IDOR); 409 outside postpartum; 403 module_disabled; the 401.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/children"
	childrenstore "github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/postpartum"
	postpartumstore "github.com/ritme/backend-go/internal/postpartum/store"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

const ppURL = "/api/v1/analysis/postpartum"

func setupPostpartum(t *testing.T) (*env, *modules.Service) {
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
	mods := modules.NewService(db, nil, quiet)
	kids := children.NewService(db, nil)
	h := analysis.NewPostpartumHandlers(analysis.PostpartumDeps{
		Postpartum: postpartum.NewService(db), PostpartumStore: postpartumstore.New(db), Pregnancy: pregstore.New(db),
		Logs: healthlog.NewService(db), Plus: plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet), Modules: mods,
		ChildrenStore: childrenstore.New(db), Children: kids, BabyLog: babylog.NewService(db, kids),
	}, clock.Real{})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get(ppURL, i18n.Middleware(languages), guard, h.Postpartum)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}, mods
}

var ppNow = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func activatePostpartum(t *testing.T, e *env, uid uint64, birth string) {
	t.Helper()
	require.NoError(t, postpartum.NewService(e.db).Activate(context.Background(), uid,
		postpartum.ActivateInput{BirthDate: civildate.MustParse(birth), BabyCount: 1}, ppNow))
}

// epds stores a full check of total (items 1… scored 1 each, item 10 always 0) taken on day.
func epds(t *testing.T, e *env, uid uint64, day string, total int) {
	t.Helper()
	answers := map[string]int{}
	for i, it := range postpartum.Items {
		if i < total && it.Code != postpartum.SelfHarmItem {
			answers[it.Code] = 1
		}
	}
	r := postpartum.Score(postpartum.KindFull, answers)
	require.Equal(t, total, r.Total)
	_, err := postpartum.NewService(e.db).SaveCheck(context.Background(), uid, r, civildate.MustParse(day), ppNow)
	require.NoError(t, err)
}

func ppCardOf(r response, key string) map[string]any {
	c, _ := path(r.data(), "cards", key).(map[string]any)
	return c
}

func TestAnalysisPostpartum_EveryCardReadyPlusAndStates(t *testing.T) {
	e, mods := setupPostpartum(t)
	a, tokA := e.user(t, "09120000201")
	b, tokB := e.user(t, "09120000202")
	_, tokC := e.user(t, "09120000203") // not postpartum

	// A: birth 2026-08-12 (6 weeks before 2026-09-23), two full EPDS checks, lochia decreasing, sleep, weights, a
	// child with two measurements and a week of feeds and sleep. Plus through a trial.
	activatePostpartum(t, e, a, "2026-08-12")
	epds(t, e, a, "2026-09-07", 9)
	epds(t, e, a, "2026-09-21", 6)
	for i, amount := range []string{"medium", "medium", "light", "light", "spotting", "none"} {
		e.entry(t, a, civildate.MustParse("2026-09-08").AddDays(i*3).String(), "bleeding", "lochia_amount", "", amount, "")
	}
	e.entry(t, a, "2025-10-20", "measurements", "weight", "", "", "61.00") // before the pregnancy
	e.entry(t, a, "2026-08-20", "measurements", "weight", "", "", "70.40")
	e.entry(t, a, "2026-09-20", "measurements", "weight", "", "", "67.10")
	res, err := e.db.Exec(`INSERT INTO children (owner_id, name, birth_date, sex, birth_weight_kg, birth_length_cm, created_at, updated_at)
		VALUES (?, 'Sara', '2026-08-12', 'girl', 3.300, 50.0, NOW(), NOW())`, a)
	require.NoError(t, err)
	child, err := res.LastInsertId()
	require.NoError(t, err)
	e.exec(t, `INSERT INTO child_measurements (child_id, measured_on, weight_kg, length_cm, created_at, updated_at)
		VALUES (?, '2026-09-20', 4.600, 55.5, NOW(), NOW())`, child)
	for i := range 7 {
		d := civildate.MustParse("2026-09-17").AddDays(i)
		e.entry(t, a, d.String(), "sleep", "hours", "", "", "5.50")
		at := d.TehranMidnight().Add(8 * time.Hour)
		for f := range 8 {
			s := at.Add(time.Duration(f) * time.Hour)
			e.exec(t, `INSERT INTO baby_feeds (child_id, type, started_at, ended_at, duration_seconds, amount_ml, created_at, updated_at)
				VALUES (?, 'bottle', ?, ?, 900, 90, NOW(), NOW())`, child, s, s.Add(15*time.Minute))
		}
		e.exec(t, `INSERT INTO baby_sleeps (child_id, started_at, ended_at, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
			child, at.Add(-6*time.Hour), at)
	}
	e.exec(t, `INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
		VALUES (?, '2026-09-22 10:00:00', '2026-09-29 10:00:00', NOW(), NOW())`, a)

	// B: postpartum too, one EPDS score of 12 and a heavy lochia day; free.
	activatePostpartum(t, e, b, "2026-09-01")
	epds(t, e, b, "2026-09-20", 7)
	e.entry(t, b, "2026-09-22", "bleeding", "lochia_amount", "", "heavy", "")

	r := e.get(t, ppURL, tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.InDelta(t, 42, r.data()["days_since_birth"], 0)
	for _, k := range []string{"epds", "bleeding", "breastfeeding", "sleep", "growth", "weight"} {
		c := ppCardOf(r, k)
		assert.Equal(t, true, c["ready"], "%s: %v", k, c)
		assert.Equal(t, false, c["locked"], k)
		assert.InDelta(t, 0, c["days_needed"], 0, k)
	}
	assert.Equal(t, "down", path(ppCardOf(r, "epds"), "data", "trend", "direction"))
	assert.InDelta(t, 6, path(ppCardOf(r, "epds"), "data", "latest", "total"), 0)
	assert.Equal(t, "decreasing", path(ppCardOf(r, "bleeding"), "data", "trend"))
	assert.Equal(t, "baby_log", path(ppCardOf(r, "breastfeeding"), "data", "source"))
	assert.InDelta(t, 8, path(ppCardOf(r, "breastfeeding"), "data", "feeds_per_day"), 0)
	assert.InDelta(t, 5.5, path(ppCardOf(r, "sleep"), "data", "mother", "avg_hours"), 0)
	assert.Equal(t, true, path(ppCardOf(r, "sleep"), "data", "baby", "ready"))
	assert.Equal(t, "Sara", path(ppCardOf(r, "growth"), "data", "child", "name"))
	assert.NotNil(t, path(ppCardOf(r, "growth"), "data", "latest", "weight", "percentile"))
	assert.InDelta(t, 61, path(ppCardOf(r, "weight"), "data", "pre_pregnancy", "kg"), 0)
	assert.InDelta(t, 6.1, path(ppCardOf(r, "weight"), "data", "delta_vs_pre_pregnancy"), 0.001)
	assert.NotContains(t, r.raw, "answers")
	assert.NotContains(t, r.raw, "self_harm", "item 10 never leaves the EPDS store (D-54)")

	// B: one score → no trend, Plus cards locked, nothing of A's.
	rb := e.get(t, ppURL, tokB, "")
	require.Equal(t, 200, rb.status, rb.raw)
	assert.Equal(t, true, ppCardOf(rb, "epds")["ready"])
	assert.Equal(t, false, ppCardOf(rb, "epds")["plus"])
	assert.Equal(t, false, path(ppCardOf(rb, "epds"), "data", "trend", "ready"))
	for _, k := range []string{"sleep", "weight"} {
		assert.Equal(t, true, ppCardOf(rb, k)["locked"], k)
		assert.Nil(t, ppCardOf(rb, k)["data"], k)
	}
	assert.Equal(t, false, ppCardOf(rb, "bleeding")["ready"])
	assert.InDelta(t, 3, ppCardOf(rb, "bleeding")["days_needed"], 0)
	assert.Nil(t, path(ppCardOf(rb, "growth"), "data", "child"))
	assert.NotContains(t, rb.raw, "Sara")

	// Not postpartum → 409; no token → the 401 body.
	rc := e.get(t, ppURL, tokC, "en")
	assert.Equal(t, 409, rc.status, rc.raw)
	assert.Equal(t, "not_postpartum", rc.body["error_code"])
	assert.Equal(t, false, rc.body["success"])
	r0 := e.get(t, ppURL, "", "")
	assert.Equal(t, 401, r0.status)
	assert.Equal(t, "unauthenticated", r0.body["error_code"])

	// Children module off → no child cards; postpartum module off → 403 module_disabled.
	ctx := context.Background()
	t.Cleanup(func() {
		_ = mods.Set(ctx, map[string]bool{modules.Children: true, modules.Postpartum: true}, time.Now())
	})
	require.NoError(t, mods.Set(ctx, map[string]bool{modules.Children: false}, time.Now()))
	r = e.get(t, ppURL, tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, false, path(ppCardOf(r, "growth"), "data", "available"))
	assert.Equal(t, "recovery", path(ppCardOf(r, "breastfeeding"), "data", "source"))
	require.NoError(t, mods.Set(ctx, map[string]bool{modules.Postpartum: false}, time.Now()))
	r = e.get(t, ppURL, tokA, "")
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "module_disabled", r.body["error_code"])
	assert.Equal(t, "postpartum", r.body["module"])
}
