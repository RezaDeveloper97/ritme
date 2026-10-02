package healthlog_test

// Log taxonomy v2 endpoints (/api/v1/logs, B-N3-01) end to end: taxonomy with labels, day save / read /
// delete, the two-way sync with daily_health_logs and the old /health-logs endpoints, IDOR, validation
// per category and mode.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/resources/translations"
)

const (
	logsClientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	logsNow      = "2026-09-23T10:00:00+03:30"
)

type logsEnv struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setupLogs(t *testing.T) *logsEnv {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, logsClientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)
	svc := healthlog.NewService(db)
	h := healthlog.NewLogHandlers(svc, clock.Real{}, i18n.NewTranslationStore(translations.FS, ""), languages)
	old := healthlog.NewHandlers(svc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/logs/taxonomy", locale, guard, h.Taxonomy)
	app.Get("/api/v1/logs/days", locale, guard, h.Days)
	app.Get("/api/v1/logs/days/:date", locale, guard, h.Day)
	app.Put("/api/v1/logs/days/:date", locale, guard, h.Save)
	app.Delete("/api/v1/logs/days/:date", locale, guard, h.Destroy)
	app.Get("/api/v1/logs/preferences", locale, guard, h.Preferences)
	app.Put("/api/v1/logs/preferences", locale, guard, h.SavePreferences)
	app.Delete("/api/v1/logs/preferences", locale, guard, h.ResetPreferences)
	app.Get("/api/v1/logs/custom-items", locale, guard, h.CustomItems)
	app.Post("/api/v1/logs/custom-items", locale, guard, h.AddCustomItem)
	app.Patch("/api/v1/logs/custom-items/:id", locale, guard, h.RenameCustomItem)
	app.Delete("/api/v1/logs/custom-items/:id", locale, guard, h.DeleteCustomItem)
	app.Post("/api/v1/health-logs", locale, guard, old.Store)
	app.Get("/api/v1/health-logs/:date", locale, guard, old.Show)
	app.Delete("/api/v1/health-logs/:date", locale, guard, old.Destroy)
	periodH := periods.NewHandlers(periods.NewService(db), clock.Real{}) // B-N3-14b: the period log the repro starts from
	app.Post("/api/v1/cycle/period", locale, guard, periodH.Store)
	return &logsEnv{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *logsEnv) user(t *testing.T, mobile, mode string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if mode != "" {
		_, err = e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, id, mode)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // positive id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // positive id
}

// customItem stores an active custom item of the user and returns its item code ("custom_<id>").
func (e *logsEnv) customItem(t *testing.T, userID uint64, category, param, label string) string {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO health_log_custom_items (user_id, category, param, label, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())`, userID, category, param, label)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return "custom_" + strconv.FormatInt(id, 10)
}

type logsResp struct {
	status int
	body   map[string]any
	raw    string
}

func (r logsResp) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func (r logsResp) categories() map[string]any {
	c, _ := r.data()["categories"].(map[string]any)
	return c
}

func (r logsResp) errorKeys() []string {
	m, _ := r.body["errors"].(map[string]any)
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (e *logsEnv) do(t *testing.T, method, path, token, lang, body string) logsResp {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, logsNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return logsResp{status: resp.StatusCode, body: m, raw: string(raw)}
}

func (e *logsEnv) legacy(t *testing.T, userID uint64, date, column string) sql.NullString {
	t.Helper()
	var v sql.NullString
	err := e.db.QueryRow("SELECT CAST(`"+column+"` AS CHAR) FROM daily_health_logs WHERE user_id = ? AND log_date = ?", userID, date).Scan(&v)
	if err == sql.ErrNoRows {
		return sql.NullString{}
	}
	require.NoError(t, err)
	return v
}

func (e *logsEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func TestLogs_Unauthenticated(t *testing.T) {
	e := setupLogs(t)
	for _, p := range []string{"/api/v1/logs/taxonomy", "/api/v1/logs/days?from=2026-09-01&to=2026-09-02", "/api/v1/logs/days/2026-09-01"} {
		r := e.do(t, "GET", p, "", "", "")
		assert.Equal(t, 401, r.status, p)
		assert.Equal(t, "unauthenticated", r.body["error_code"], p)
	}
	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-01", "", "", `{"categories":{}}`)
	assert.Equal(t, 401, r.status)
}

func TestLogs_Taxonomy(t *testing.T) {
	e := setupLogs(t)
	_, tok := e.user(t, "09120000001", "")
	_, postTok := e.user(t, "09120000002", "postpartum")

	r := e.do(t, "GET", "/api/v1/logs/taxonomy", tok, "fa", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "cycle", r.data()["mode"], "no stored mode → cycle")
	first, _ := r.data()["categories"].([]any)[0].(map[string]any)
	assert.Equal(t, "پریود و لکه\u200cبینی", first["label"])
	assert.NotContains(t, r.raw, `"lochia_amount"`)

	r = e.do(t, "GET", "/api/v1/logs/taxonomy", tok, "en", "")
	assert.Contains(t, r.raw, `"label":"Period & spotting"`)

	r = e.do(t, "GET", "/api/v1/logs/taxonomy", postTok, "en", "")
	assert.Equal(t, "postpartum", r.data()["mode"])
	assert.Contains(t, r.raw, `"lochia_amount"`)
	assert.NotContains(t, r.raw, `"code":"flow"`)

	r = e.do(t, "GET", "/api/v1/logs/taxonomy?mode=pregnancy", tok, "en", "")
	assert.Equal(t, "pregnancy", r.data()["mode"])
	assert.Contains(t, r.raw, `"source":"kick_counter"`)

	r = e.do(t, "GET", "/api/v1/logs/taxonomy?mode=all", tok, "en", "")
	assert.Nil(t, r.data()["mode"])
	assert.Contains(t, r.raw, `"legacy_only":true`)

	r = e.do(t, "GET", "/api/v1/logs/taxonomy?mode=martian", tok, "en", "")
	assert.Equal(t, 422, r.status)
	assert.Equal(t, []string{"mode"}, r.errorKeys())
}

func TestLogs_SaveReadDelete_WritesLegacyBack(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000001", "")
	coffee := e.customItem(t, uid, "custom", "items", "قهوه")
	body := `{"categories":{
		"bleeding":{"flow":"heavy","color":"pink","spotting":false},
		"pain":{"location":{"abdomen":{"level":"moderate","score":6}},"relief":["heat"]},
		"mood":{"moods":["calm","energetic"]},
		"measurements":{"weight":58.4},
		"note":{"text":"سلام"},
		"custom":{"items":{"coffee":"yes"}}
	}}`
	body = strings.ReplaceAll(body, `"coffee"`, `"`+coffee+`"`)
	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", body)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "2026-09-22", r.data()["date"])
	assert.Equal(t, map[string]any{"flow": "heavy", "color": "pink", "spotting": false}, r.categories()["bleeding"])

	g := e.do(t, "GET", "/api/v1/logs/days/2026-09-22", tok, "en", "")
	require.Equal(t, 200, g.status)
	assert.JSONEq(t, mustJSON(t, r.data()), mustJSON(t, g.data()))
	assert.Equal(t, map[string]any{"abdomen": map[string]any{"level": "moderate", "score": float64(6)}},
		g.categories()["pain"].(map[string]any)["location"])

	// daily_health_logs got the legacy-representable values
	assert.Equal(t, "high", e.legacy(t, uid, "2026-09-22", "bleeding_intensity").String)
	assert.False(t, e.legacy(t, uid, "2026-09-22", "blood_color").Valid, "pink has no legacy value")
	assert.Equal(t, "0", e.legacy(t, uid, "2026-09-22", "spotting").String)
	assert.Equal(t, "medium", e.legacy(t, uid, "2026-09-22", "stomach_ache_intensity").String)
	assert.Equal(t, `["calm"]`, e.legacy(t, uid, "2026-09-22", "moods").String, "energetic is v2-only")
	assert.Equal(t, "58.40", e.legacy(t, uid, "2026-09-22", "weight").String)
	assert.Equal(t, "سلام", e.legacy(t, uid, "2026-09-22", "notes").String)
	old := e.do(t, "GET", "/api/v1/health-logs/2026-09-22", tok, "en", "")
	require.Equal(t, 200, old.status)
	assert.Equal(t, "high", old.data()["bleeding_intensity"])

	// a param sent as null clears it (and its legacy column); params not sent stay
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", `{"categories":{"bleeding":{"flow":null},"mood":null}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, map[string]any{"color": "pink", "spotting": false}, r.categories()["bleeding"])
	assert.NotContains(t, r.categories(), "mood")
	assert.Contains(t, r.categories(), "note")
	assert.False(t, e.legacy(t, uid, "2026-09-22", "bleeding_intensity").Valid)
	assert.False(t, e.legacy(t, uid, "2026-09-22", "moods").Valid)
	assert.Equal(t, "58.40", e.legacy(t, uid, "2026-09-22", "weight").String)

	// a day with only v2-only values creates no legacy row
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-21", tok, "en", `{"categories":{"custom":{"items":{"`+coffee+`":"yes"}},"bleeding":{"color":"black"}}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM daily_health_logs WHERE user_id = ? AND log_date = '2026-09-21'", uid))

	// range
	d := e.do(t, "GET", "/api/v1/logs/days?from=2026-09-01&to=2026-09-30", tok, "en", "")
	require.Equal(t, 200, d.status, d.raw)
	days := d.data()["days"].([]any)
	require.Len(t, days, 2)
	assert.Equal(t, "2026-09-21", days[0].(map[string]any)["date"])

	// delete removes v2 and legacy
	x := e.do(t, "DELETE", "/api/v1/logs/days/2026-09-22", tok, "en", "")
	require.Equal(t, 200, x.status, x.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND log_date = '2026-09-22'", uid))
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM daily_health_logs WHERE user_id = ? AND log_date = '2026-09-22'", uid))
}

func TestLogs_OldEndpointKeepsV2InStep(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000001", "")
	coffee := e.customItem(t, uid, "custom", "items", "قهوه")
	r := e.do(t, "POST", "/api/v1/health-logs", tok, "en",
		`{"log_date":"2026-09-20","headache_intensity":"high","moods":["sad"],"weight":60,"spotting":true,"vaginal_burning":true}`)
	require.Equal(t, 201, r.status, r.raw)

	g := e.do(t, "GET", "/api/v1/logs/days/2026-09-20", tok, "en", "")
	c := g.categories()
	assert.Equal(t, map[string]any{"head": map[string]any{"level": "severe", "score": nil}}, c["pain"].(map[string]any)["location"])
	assert.Equal(t, []any{"sad"}, c["mood"].(map[string]any)["moods"])
	assert.Equal(t, float64(60), c["measurements"].(map[string]any)["weight"])
	assert.Equal(t, true, c["bleeding"].(map[string]any)["spotting"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND source = 'legacy' AND item = 'vaginal_burning' AND value_code = 'yes'", uid))

	// the intensity of a merged pair wins once it is written
	r = e.do(t, "POST", "/api/v1/health-logs", tok, "en", `{"log_date":"2026-09-20","vaginal_burning_intensity":"low"}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND item = 'vaginal_burning' AND value_code = 'mild'", uid))

	// v2-only values survive old-endpoint writes and deletes
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-20", tok, "en", `{"categories":{"custom":{"items":{"`+coffee+`":"yes"}},"pain":{"location":{"head":{"level":"severe","score":9}}}}}`)
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, "POST", "/api/v1/health-logs", tok, "en", `{"log_date":"2026-09-20","moods":["calm"],"headache_intensity":null}`)
	require.Equal(t, 200, r.status, r.raw)
	c = e.do(t, "GET", "/api/v1/logs/days/2026-09-20", tok, "en", "").categories()
	assert.Equal(t, []any{"calm"}, c["mood"].(map[string]any)["moods"])
	assert.NotContains(t, c, "pain", "headache cleared through the old endpoint")
	assert.Contains(t, c, "custom")

	r = e.do(t, "DELETE", "/api/v1/health-logs/2026-09-20", tok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	c = e.do(t, "GET", "/api/v1/logs/days/2026-09-20", tok, "en", "").categories()
	assert.Equal(t, []string{"custom"}, keysOf(c), "only v2-only entries remain")
}

func TestLogs_IDOR(t *testing.T) {
	e := setupLogs(t)
	a, tokA := e.user(t, "09120000001", "")
	_, tokB := e.user(t, "09120000002", "")
	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tokA, "en", `{"categories":{"bleeding":{"flow":"heavy"},"note":{"text":"A"}}}`)
	require.Equal(t, 200, r.status, r.raw)
	before := e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ?", a)

	g := e.do(t, "GET", "/api/v1/logs/days/2026-09-22", tokB, "en", "")
	require.Equal(t, 200, g.status)
	assert.Empty(t, g.categories(), "B never sees A's day")
	d := e.do(t, "GET", "/api/v1/logs/days?from=2026-09-01&to=2026-09-30", tokB, "en", "")
	assert.Empty(t, d.data()["days"])

	assert.Equal(t, 200, e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tokB, "en", `{"categories":{"bleeding":null,"note":{"text":"B"}}}`).status)
	assert.Equal(t, 200, e.do(t, "DELETE", "/api/v1/logs/days/2026-09-22", tokB, "en", "").status)
	assert.Equal(t, before, e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ?", a), "A's entries untouched")
	ga := e.do(t, "GET", "/api/v1/logs/days/2026-09-22", tokA, "en", "")
	assert.Equal(t, map[string]any{"text": "A"}, ga.categories()["note"])
	assert.Equal(t, "high", e.legacy(t, a, "2026-09-22", "bleeding_intensity").String)
}

func TestLogs_Validation(t *testing.T) {
	e := setupLogs(t)
	_, tok := e.user(t, "09120000001", "")
	_, postTok := e.user(t, "09120000002", "postpartum")

	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-24", tok, "en", `{"categories":{"note":{"text":"x"}}}`)
	assert.Equal(t, 422, r.status, "future day")
	assert.Equal(t, []string{"date"}, r.errorKeys())
	r = e.do(t, "GET", "/api/v1/logs/days/2026-02-30", tok, "en", "")
	assert.Equal(t, 422, r.status, "not a real day")

	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "fa", `{"categories":{"measurements":{"weight":500},"pain":{"location":{"abdomen":{"level":"moderate","score":0}}}}}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.ElementsMatch(t, []string{"categories.measurements.weight", "categories.pain.location.abdomen.score"}, r.errorKeys())
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM health_log_entries"), "nothing is written on a 422")

	lochia := `{"categories":{"bleeding":{"lochia_amount":"heavy"}}}`
	assert.Equal(t, 422, e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", lochia).status, "lochia is postpartum only")
	assert.Equal(t, 200, e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", postTok, "en", lochia).status)

	r = e.do(t, "GET", "/api/v1/logs/days?from=2025-01-01&to=2026-09-01", tok, "en", "")
	assert.Equal(t, 422, r.status, "range over 366 days")
	r = e.do(t, "GET", "/api/v1/logs/days?from=2026-09-10&to=2026-09-01", tok, "en", "")
	assert.Equal(t, 422, r.status)
	r = e.do(t, "GET", "/api/v1/logs/days", tok, "en", "")
	assert.ElementsMatch(t, []string{"from", "to"}, r.errorKeys())
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
