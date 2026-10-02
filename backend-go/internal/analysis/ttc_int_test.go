package analysis_test

// /api/v1/analysis/ttc and /analysis/fertility end to end on MariaDB (B-N3-11): BBT and intercourse from
// health_log_entries, LH and mucus merged from fertility_logs and health_log_entries, the caller's rows only
// (IDOR), Plus cards locked for a free user, validation, the 401 body and the request language.

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
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
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/resources/translations"
)

func setupTTC(t *testing.T) *env {
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
	locale := i18n.Middleware(languages)
	h := analysis.NewTTCHandlers(analysis.NewHandlers(cycleservice.New(db, nil), healthlog.NewService(db),
		plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet),
		i18n.NewTranslationStore(translations.FS, ""), languages, clock.Real{}), fertility.NewService(db))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/analysis/ttc", locale, guard, h.TTC)
	app.Get("/api/v1/analysis/fertility", locale, guard, h.Fertility)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) fertilityLog(t *testing.T, uid uint64, date, lh, mucus string) {
	t.Helper()
	var l, m any
	if lh != "" {
		l = lh
	}
	if mucus != "" {
		m = mucus
	}
	_, err := e.db.Exec(`INSERT INTO fertility_logs (user_id, log_date, lh_test, cervical_mucus, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())`, uid, date, l, m)
	require.NoError(t, err)
}

// seedTTC: 28-day cycles from 2026-07-20 (current from 2026-09-14, day 10 today), a full BBT chart with a
// shift after day 14 in the two completed cycles, LH positive on day 13 (fertility_logs in one cycle, the
// v2 log in the other), egg-white mucus on day 13 and unprotected intercourse on days 12 and 14.
func seedTTC(t *testing.T, e *env, uid uint64) {
	e.periods(t, uid, "2026-07-20", "2026-08-17", "2026-09-14")
	for ci, s := range []string{"2026-07-20", "2026-08-17", "2026-09-14"} {
		start := civildate.MustParse(s)
		for n := 1; n <= 28; n++ {
			d := start.AddDays(n - 1)
			if d.After(civildate.MustParse("2026-09-23")) {
				break
			}
			temp := 36.20 + float64(n%3)*0.05
			if n > 14 {
				temp = 36.70 + float64(n%2)*0.05
			}
			e.entry(t, uid, d.String(), "measurements", "bbt", "", "", fmt.Sprintf("%.2f", temp))
			switch n {
			case 12, 14:
				e.entry(t, uid, d.String(), "sex", "intercourse", "", "unprotected", "")
			case 13:
				if ci == 0 {
					e.fertilityLog(t, uid, d.String(), "positive", "egg_white")
				} else {
					e.entry(t, uid, d.String(), "measurements", "lh_test", "", "positive", "")
					e.entry(t, uid, d.String(), "discharge", "consistency", "", "egg_white", "")
				}
			}
		}
	}
}

func TestTTC_HubOwnDataAndPlus(t *testing.T) {
	e := setupTTC(t)
	a, tokA := e.user(t, "09120000011")
	b, tokB := e.user(t, "09120000012")
	seedTTC(t, e, a)
	e.periods(t, b, "2026-09-01")
	e.fertilityLog(t, b, "2026-09-10", "positive", "dry")
	e.entry(t, b, "2026-09-11", "measurements", "bbt", "", "", "37.95")

	r := e.get(t, "/api/v1/analysis/ttc", tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "2026-07-20", path(r.data(), "trying", "since"))
	assert.InDelta(t, 3, path(r.data(), "trying", "cycles"), 0)
	assert.InDelta(t, 2, path(r.data(), "trying", "confirmed_cycles"), 0)
	cycles, _ := r.data()["cycles"].([]any)
	require.Len(t, cycles, 3)
	for _, c := range cycles[:2] {
		cm := c.(map[string]any)
		assert.Equal(t, "bbt", cm["ovulation_source"])
		assert.InDelta(t, 14, cm["ovulation_day"], 0)
		assert.InDelta(t, 13, cm["lh_day"], 0, "LH read from fertility_logs and from health_log_entries")
	}
	for _, k := range []string{"timing", "mucus", "luteal"} {
		assert.Equal(t, true, path(r.data(), k, "locked"), k)
		assert.Nil(t, path(r.data(), k, "data"), k)
	}
	assert.NotContains(t, r.raw, "37.95", "B's reading never shows in A's report")

	// A trial opens the Plus cards.
	_, err := e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
		VALUES (?, '2026-09-22 10:00:00', '2026-09-29 10:00:00', NOW(), NOW())`, a)
	require.NoError(t, err)
	r = e.get(t, "/api/v1/analysis/ttc", tokA, "en")
	require.Equal(t, 200, r.status, r.raw)
	assert.InDelta(t, 14, path(r.data(), "luteal", "data", "days"), 0)
	assert.InDelta(t, 2, path(r.data(), "mucus", "data", "cycles_with_egg_white"), 0)
	assert.Contains(t, path(r.data(), "trying", "summary", "text"), "Ovulation confirmed")

	// B: own data only (one cycle, no trying history from A).
	rb := e.get(t, "/api/v1/analysis/ttc", tokB, "")
	require.Equal(t, 200, rb.status, rb.raw)
	assert.Equal(t, "2026-09-01", path(rb.data(), "trying", "since"))
	assert.NotContains(t, rb.raw, "2026-07-20")
}

func TestTTC_FertilityDetail(t *testing.T) {
	e := setupTTC(t)
	a, tok := e.user(t, "09120000013")
	seedTTC(t, e, a)

	r := e.get(t, "/api/v1/analysis/fertility", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "2026-09-14", path(r.data(), "cycle", "start"))
	assert.Equal(t, "2026-08-17", r.data()["prev_start"])

	r = e.get(t, "/api/v1/analysis/fertility?cycle=2026-07-20", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, path(r.data(), "chart", "confirmed"))
	assert.InDelta(t, 14, path(r.data(), "ovulation", "day"), 0)
	assert.Equal(t, true, path(r.data(), "luteal", "locked"))
	assert.Contains(t, path(r.data(), "explanation", "text"), "سه بالای شش")

	for _, q := range []string{"?cycle=2026-07-21", "?cycle=20-07-2026"} {
		bad := e.get(t, "/api/v1/analysis/fertility"+q, tok, "")
		assert.Equal(t, 422, bad.status, q+" "+bad.raw)
	}
}

func TestTTC_EmptyAndAuth(t *testing.T) {
	e := setupTTC(t)
	_, tok := e.user(t, "09120000014")
	r := e.get(t, "/api/v1/analysis/ttc", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Nil(t, path(r.data(), "trying", "since"))
	assert.Equal(t, false, path(r.data(), "bbt", "ready"))
	f := e.get(t, "/api/v1/analysis/fertility", tok, "")
	assert.Equal(t, 404, f.status, f.raw)

	for _, url := range []string{"/api/v1/analysis/ttc", "/api/v1/analysis/fertility"} {
		u := e.get(t, url, "", "")
		assert.Equal(t, 401, u.status)
		assert.Equal(t, "unauthenticated", u.body["error_code"])
	}
}
