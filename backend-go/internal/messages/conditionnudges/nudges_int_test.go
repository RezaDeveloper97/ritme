package conditionnudges_test

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
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-20T10:00:00+03:30"
	path     = "/nudges"
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
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	h := conditionnudges.NewHandlers(db, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get(path, locale, guard, h.Index)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

func (e *env) period(t *testing.T, uid uint64, start string) {
	e.exec(t, `INSERT INTO cycle_histories (user_id, period_start_date, bleeding_length, is_confirmed, created_at, updated_at)
		VALUES (?, ?, 5, 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid, start)
}

func (e *env) painDay(t *testing.T, uid uint64, date string, score string) {
	e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, source, created_at, updated_at)
		VALUES (?, ?, 'pain', 'location', 'abdomen', 'severe', ?, 'manual', NOW(), NOW())`, uid, date, score)
}

func (e *env) flowDay(t *testing.T, uid uint64, date, code string) {
	e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
		VALUES (?, ?, 'bleeding', 'flow', '', ?, 'legacy', NOW(), NOW())`, uid, date, code)
}

func (e *env) enrol(t *testing.T, uid uint64, program string) {
	e.exec(t, `INSERT INTO condition_enrolments (user_id, program, enrolled_on, created_at, updated_at)
		VALUES (?, ?, '2026-09-01', NOW(), NOW())`, uid, program)
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) nudges() []map[string]any {
	data, _ := r.body["data"].(map[string]any)
	list, _ := data["nudges"].([]any)
	out := []map[string]any{}
	for _, v := range list {
		m, _ := v.(map[string]any)
		out = append(out, m)
	}
	return out
}

func (r response) keys() []string {
	out := []string{}
	for _, n := range r.nudges() {
		k, _ := n["key"].(string)
		out = append(out, k)
	}
	return out
}

func (e *env) get(t *testing.T, token, lang string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func TestNudges_Unauthenticated(t *testing.T) {
	e := setup(t)
	r := e.get(t, "", "en")
	assert.Equal(t, http.StatusUnauthorized, r.status, r.raw)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestNudges_HeavyPainAndBleeding(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")
	e.period(t, uid, "2026-09-08")
	e.painDay(t, uid, "2026-09-05", "9") // previous cycle: not counted
	e.painDay(t, uid, "2026-09-09", "7")
	e.painDay(t, uid, "2026-09-11", "8.5")
	e.flowDay(t, uid, "2026-09-08", "very_heavy")
	e.flowDay(t, uid, "2026-09-09", "heavy")

	r := e.get(t, tok, "en")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, "2026-09-08", data["from"])
	assert.Equal(t, "2026-09-20", data["to"])
	require.Equal(t, []string{"heavy_pain", "heavy_bleeding"}, r.keys())

	pain := r.nudges()[0]
	assert.Equal(t, "endo", pain["program"])
	assert.Equal(t, "/programs/pain", pain["link"])
	assert.Nil(t, pain["doctor"], "no doctors directory yet: no doctor link")
	assert.EqualValues(t, 2, pain["days"])
	assert.Equal(t, []any{"2026-09-09", "2026-09-11"}, pain["dates"])
	assert.Contains(t, pain["body"], "2 days")
	assert.Equal(t, true, pain["needs_review"])

	bleed := r.nudges()[1]
	assert.Equal(t, "heavy_bleeding", bleed["program"])
	assert.Equal(t, "/programs/bleeding", bleed["link"])

	fa := e.get(t, tok, "fa")
	require.Equal(t, http.StatusOK, fa.status, fa.raw)
	assert.Contains(t, fa.nudges()[0]["title"], "درد")
	assert.Contains(t, fa.nudges()[0]["body"], "۲ روز", "{days} in Persian digits for fa")
	assert.NotContains(t, fa.nudges()[0]["body"], "2")
}

func TestNudges_BelowThresholdNothing(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000002")
	e.period(t, uid, "2026-09-08")
	e.painDay(t, uid, "2026-09-09", "6")
	e.painDay(t, uid, "2026-09-10", "9")
	e.flowDay(t, uid, "2026-09-09", "heavy")
	e.flowDay(t, uid, "2026-09-10", "medium")

	r := e.get(t, tok, "en")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, r.nudges())
}

func TestNudges_EnrolledUsersNotNudged(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000003")
	e.period(t, uid, "2026-09-08")
	for _, day := range []string{"2026-09-09", "2026-09-10"} {
		e.painDay(t, uid, day, "8")
		e.flowDay(t, uid, day, "heavy")
	}
	e.enrol(t, uid, "endo")
	assert.Equal(t, []string{"heavy_bleeding"}, e.get(t, tok, "en").keys())

	e.enrol(t, uid, "heavy_bleeding")
	assert.Empty(t, e.get(t, tok, "en").keys())
}

func TestNudges_UserIsolation(t *testing.T) {
	e := setup(t)
	a, _ := e.user(t, "09120000004")
	b, tokB := e.user(t, "09120000005")
	e.period(t, a, "2026-09-08")
	e.period(t, b, "2026-09-08")
	for _, day := range []string{"2026-09-09", "2026-09-10"} {
		e.painDay(t, a, day, "9")
		e.flowDay(t, a, day, "heavy")
	}
	e.flowDay(t, b, "2026-09-09", "heavy") // one day only

	// B sees nothing of A's log …
	assert.Empty(t, e.get(t, tokB, "en").keys())
	// … and A's enrolment does not silence B.
	e.enrol(t, a, "heavy_bleeding")
	e.flowDay(t, b, "2026-09-10", "very_heavy")
	assert.Equal(t, []string{"heavy_bleeding"}, e.get(t, tokB, "en").keys())
}

func TestNudges_AdminEditedTexts(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000006")
	e.period(t, uid, "2026-09-08")
	e.painDay(t, uid, "2026-09-09", "8")
	e.painDay(t, uid, "2026-09-10", "8")
	e.exec(t, `INSERT INTO message_contents (`+"`group`"+`, item_key, locale, payload, is_active, is_approved, created_at, updated_at)
		VALUES ('condition_nudge', 'heavy_pain', 'en', '{"title":"Edited title","body":"Pain on {days} days"}', 1, 1, NOW(), NOW())`)

	n := e.get(t, tok, "en").nudges()
	require.Len(t, n, 1)
	assert.Equal(t, "Edited title", n[0]["title"])
	assert.Equal(t, "Pain on 2 days", n[0]["body"])
	assert.Equal(t, "Open the pain diary", n[0]["action"], "a missing text falls back to the embedded copy")
	assert.Equal(t, true, n[0]["needs_review"])

	// An unapproved row is not live.
	e.exec(t, `UPDATE message_contents SET is_approved = 0 WHERE `+"`group`"+` = 'condition_nudge'`)
	assert.Equal(t, "Strong pain on several days", e.get(t, tok, "en").nudges()[0]["title"])
}

func TestEngine_NoPeriodUsesProfileCycleLength(t *testing.T) {
	e := setup(t)
	uid, _ := e.user(t, "09120000007")
	e.exec(t, `INSERT INTO user_profiles (user_id, cycle_duration, created_at, updated_at) VALUES (?, 30, NOW(), NOW())`, uid)
	e.flowDay(t, uid, "2026-08-21", "heavy") // 31 days ago: outside
	e.flowDay(t, uid, "2026-08-22", "heavy")
	today, err := civildate.Parse("2026-09-20")
	require.NoError(t, err)

	res, err := conditionnudges.NewEngine(e.db, nil).Evaluate(context.Background(), uid, today, "en")
	require.NoError(t, err)
	assert.Equal(t, "2026-08-22", res.From.String())
	assert.Empty(t, res.Nudges)

	e.flowDay(t, uid, "2026-09-19", "heavy")
	res, err = conditionnudges.NewEngine(e.db, nil).Evaluate(context.Background(), uid, today, "en")
	require.NoError(t, err)
	require.Len(t, res.Nudges, 1)
	assert.Equal(t, conditionnudges.RuleHeavyBleeding, res.Nudges[0].Rule)
}

type fakeDoctors struct{}

func (fakeDoctors) DoctorLink(context.Context, string) (string, bool, error) {
	return "/services/doctors?specialty=gynaecology", true, nil
}

func TestEngine_DoctorLinkOnlyWithDirectory(t *testing.T) {
	e := setup(t)
	uid, _ := e.user(t, "09120000008")
	e.period(t, uid, "2026-09-08")
	e.painDay(t, uid, "2026-09-09", "8")
	e.painDay(t, uid, "2026-09-10", "8")
	today, err := civildate.Parse("2026-09-20")
	require.NoError(t, err)

	res, err := conditionnudges.NewEngine(e.db, fakeDoctors{}).Evaluate(context.Background(), uid, today, "en")
	require.NoError(t, err)
	require.Len(t, res.Nudges, 1)
	require.NotNil(t, res.Nudges[0].DoctorLink)
	raw, err := json.Marshal(res.JSON())
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"doctor":{"action":"Find a gynaecologist","link":"/services/doctors?specialty=gynaecology"}`)
}
