package menomessages_test

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
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages/menomessages"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-10-01T10:00:00+03:30" // 9 Mehr 1405
	path     = "/messages/menopause"
	stamp    = "2026-09-01 09:00:00"
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
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, ?, ?)`, clientID, stamp, stamp)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	cat := catalog.NewReader(catalogstore.New(db), nil, 0, quiet)
	h := menomessages.NewHandlers(db, menopause.NewService(db, cat), clock.Real{})

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
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, ?, ?)`, mobile, stamp, stamp)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

// menopauseUser stores bloom's life profile in menopause mode with a stage ("" = not answered).
func (e *env) menopauseUser(t *testing.T, mobile, stage, lastPeriod string) (uint64, string) {
	t.Helper()
	uid, tok := e.user(t, mobile)
	var st, lp any
	if stage != "" {
		st = stage
	}
	if lastPeriod != "" {
		lp = lastPeriod
	}
	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, gender, menopause_stage, menopause_last_period, created_at, updated_at)
		VALUES (?, 'menopause', 'female', ?, ?, ?, ?)`, uid, st, lp, stamp, stamp)
	return uid, tok
}

func (e *env) spotting(t *testing.T, uid uint64, date string) {
	e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
		VALUES (?, ?, 'bleeding', 'presence', '', 'spotting', 'manual', ?, ?)`, uid, date, stamp, stamp)
}

func (e *env) score(t *testing.T, uid uint64, month string, total int) {
	e.exec(t, `INSERT INTO menopause_scores (user_id, month, answers, total, somatic, psychological, urogenital, created_at, updated_at)
		VALUES (?, ?, '{}', ?, 0, 0, 0, ?, ?)`, uid, month, total, stamp, stamp)
}

func (e *env) hrt(t *testing.T, uid uint64, name, reviewOn string) {
	e.exec(t, `INSERT INTO treatment_items (user_id, kind, name, started_on, review_on, sort_order, created_at, updated_at)
		VALUES (?, 'hrt', ?, '2026-07-01', ?, 0, ?, ?)`, uid, name, reviewOn, stamp, stamp)
}

// doneLongAgo records the checkup `key` as done on 2022-01-10 (blood sugar: every 1–3 years → overdue now).
func (e *env) doneLongAgo(t *testing.T, uid uint64, key string) {
	e.exec(t, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
		SELECT ?, id, '2022-01-10', 'normal', ?, ? FROM checkup_types WHERE `+"`key`"+` = ?`, uid, stamp, stamp, key)
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (r response) messages() []map[string]any {
	list, _ := r.data()["messages"].([]any)
	out := []map[string]any{}
	for _, v := range list {
		m, _ := v.(map[string]any)
		out = append(out, m)
	}
	return out
}

func (r response) keys() []string {
	out := []string{}
	for _, m := range r.messages() {
		k, _ := m["key"].(string)
		out = append(out, k)
	}
	return out
}

func (r response) message(key string) map[string]any {
	for _, m := range r.messages() {
		if m["key"] == key {
			return m
		}
	}
	return nil
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
	r := response{status: resp.StatusCode, body: m, raw: string(raw)}
	require.Equal(t, http.StatusOK, r.status, r.raw)
	return r
}

func TestMessages_Unauthenticated(t *testing.T) {
	e := setup(t)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, string(raw))
}

func TestMessages_NonMenopauseUsersGetNone(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001") // no life profile: cycle
	e.spotting(t, uid, "2026-09-30")
	e.hrt(t, uid, "Estradiol", "2026-10-03")
	e.score(t, uid, "2026-09-23", 20)
	e.score(t, uid, "2026-08-23", 10)
	r := e.get(t, tok, "en")
	assert.Equal(t, "cycle", r.data()["mode"])
	assert.Nil(t, r.data()["stage"])
	assert.Equal(t, []any{}, r.data()["messages"])

	// A stored menopause answer set in another mode does not count either (teen).
	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, gender, menopause_stage, created_at, updated_at)
		VALUES (?, 'teen', 'female', 'meno', ?, ?)`, uid, stamp, stamp)
	r = e.get(t, tok, "en")
	assert.Equal(t, "teen", r.data()["mode"])
	assert.Empty(t, r.keys())
}

func TestMessages_BleedingAlertFromCatalog(t *testing.T) {
	e := setup(t)
	uid, tok := e.menopauseUser(t, "09120000002", "meno", "2025-03-01")
	e.spotting(t, uid, "2026-09-28")

	r := e.get(t, tok, "en")
	assert.Equal(t, "menopause", r.data()["mode"])
	assert.Equal(t, "meno", r.data()["stage"])
	keys := r.keys()
	require.Equal(t, []string{"postmenopausal_bleeding", "checkup_due", "stage_meno"}, keys, "never-done checkups are due")

	b := r.messages()[0]
	assert.Equal(t, "alert", b["kind"])
	assert.Equal(t, "high", b["priority"])
	assert.Equal(t, "Bleeding after menopause", b["title"])
	assert.Contains(t, b["body"], "12 months")
	assert.Equal(t, "Tell your doctor about this", b["action"], "meta.cta")
	assert.Equal(t, "/menopause/alert", b["link"])
	assert.Equal(t, true, b["needs_review"], "seeded catalog item is flagged")
	assert.Equal(t, map[string]any{"last_on": "2026-09-28"}, b["data"])

	due := r.messages()[1]
	assert.Equal(t, "reminder", due["kind"])
	assert.Equal(t, "low", due["priority"])
	checkup := due["data"].(map[string]any)["checkup"].(map[string]any)
	assert.Equal(t, "due", checkup["status"])
	assert.Contains(t, due["body"], checkup["title"].(string), "{checkup} = the plan item's title")
	assert.Equal(t, "/checkups/"+jsonNum(checkup["id"]), due["link"])
	assert.Greater(t, due["data"].(map[string]any)["count"].(float64), float64(1))

	tip := r.messages()[2]
	assert.Equal(t, "tip", tip["kind"])
	assert.Equal(t, "Menopause", tip["title"])
	assert.Nil(t, tip["link"])
	assert.Nil(t, tip["data"])

	fa := e.get(t, tok, "fa")
	assert.Equal(t, "خونریزی بعد از یائسگی", fa.messages()[0]["title"])
	assert.Equal(t, "این مورد را به پزشک بگو", fa.messages()[0]["action"])

	// The catalog item switched off: the embedded copy still raises the alert.
	e.exec(t, "UPDATE catalog_items SET is_active = 0 WHERE `group` = 'meno_alerts' AND code = 'postmenopausal_bleeding'")
	b = e.get(t, tok, "en").message("postmenopausal_bleeding")
	require.NotNil(t, b)
	assert.Equal(t, "Bleeding after menopause", b["title"])
	assert.Contains(t, b["body"], "You logged bleeding or spotting")
	assert.Equal(t, true, b["needs_review"])
}

func TestMessages_BleedingOnlyFromMenoStage(t *testing.T) {
	e := setup(t)
	uid, tok := e.menopauseUser(t, "09120000003", "peri", "2026-06-01")
	e.spotting(t, uid, "2026-09-28")
	r := e.get(t, tok, "en")
	assert.Equal(t, "peri", r.data()["stage"])
	assert.Nil(t, r.message("postmenopausal_bleeding"), "perimenopause bleeding is not an alert")
	assert.NotNil(t, r.message("stage_peri"))

	// Older than the 30-day window: no alert in stage meno either.
	uid2, tok2 := e.menopauseUser(t, "09120000004", "meno", "2025-01-01")
	e.spotting(t, uid2, "2026-08-31")
	assert.Nil(t, e.get(t, tok2, "en").message("postmenopausal_bleeding"))
}

func TestMessages_NoStageGetsTheUnsureTip(t *testing.T) {
	e := setup(t)
	_, tok := e.menopauseUser(t, "09120000005", "", "")
	r := e.get(t, tok, "en")
	assert.Nil(t, r.data()["stage"])
	require.NotNil(t, r.message("stage_unsure"))
	assert.Equal(t, "Not sure", r.message("stage_unsure")["title"])
}

func TestMessages_OverdueScoreAndHRT(t *testing.T) {
	e := setup(t)
	uid, tok := e.menopauseUser(t, "09120000006", "post", "2020-01-01")
	e.doneLongAgo(t, uid, "meno_blood_sugar")
	e.score(t, uid, "2026-08-23", 9)  // Shahrivar
	e.score(t, uid, "2026-09-23", 14) // Mehr: +5
	e.hrt(t, uid, "Estradiol gel", "2026-10-06")
	e.hrt(t, uid, "Progesterone", "2026-11-30") // too far

	r := e.get(t, tok, "en")
	require.Equal(t, []string{"checkup_overdue", "score_worsened", "hrt_review", "stage_post"}, r.keys())

	ov := r.message("checkup_overdue")
	assert.Equal(t, "medium", ov["priority"])
	assert.Equal(t, "Fasting blood sugar or HbA1c is past its due time. Booking it soon keeps your health plan on track.", ov["body"])
	assert.EqualValues(t, 1, ov["data"].(map[string]any)["count"])
	assert.Equal(t, "overdue", ov["data"].(map[string]any)["checkup"].(map[string]any)["status"])

	sc := r.message("score_worsened")
	assert.Equal(t, "alert", sc["kind"])
	assert.Equal(t, "Your latest score is 14, 5 points higher than the time before (9). Note what changed, and talk to your doctor if it stays high.", sc["body"])
	assert.Equal(t, "/menopause/score", sc["link"])
	assert.Equal(t, true, sc["needs_review"])
	assert.Equal(t, map[string]any{"month": "2026-09-23", "total": float64(14), "previous_month": "2026-08-23",
		"previous": float64(9), "delta": float64(5)}, sc["data"])

	hr := r.message("hrt_review")
	assert.Contains(t, hr["body"], "Estradiol gel")
	assert.Contains(t, hr["body"], "In 5 days: the review of Estradiol gel")
	d := hr["data"].(map[string]any)
	assert.Equal(t, "2026-10-06", d["review_on"])
	assert.EqualValues(t, 5, d["days_left"])

	fa := e.get(t, tok, "fa")
	assert.Contains(t, fa.message("score_worsened")["body"], "۱۴")
	assert.Contains(t, fa.message("score_worsened")["body"], "۵ واحد")
	assert.Contains(t, fa.message("hrt_review")["body"], "۱۴ مهر ۱۴۰۵")

	// Filled again with a lower total: no longer worse.
	e.exec(t, `UPDATE menopause_scores SET total = 12 WHERE user_id = ? AND month = '2026-09-23'`, uid)
	assert.Nil(t, e.get(t, tok, "en").message("score_worsened"))
}

func TestMessages_AdminEditedTexts(t *testing.T) {
	e := setup(t)
	uid, tok := e.menopauseUser(t, "09120000007", "meno", "2025-01-01")
	e.score(t, uid, "2026-08-23", 5)
	e.score(t, uid, "2026-09-23", 15)
	e.exec(t, "INSERT INTO message_contents (`group`, item_key, locale, payload, is_active, is_approved, created_at, updated_at)"+
		` VALUES ('menopause_message', 'score_worsened', 'en', '{"title":"Edited","body":"Up by {delta} to {total}"}', 1, 1, NOW(), NOW())`)

	m := e.get(t, tok, "en").message("score_worsened")
	assert.Equal(t, "Edited", m["title"])
	assert.Equal(t, "Up by 10 to 15", m["body"])
	assert.Equal(t, "See my score", m["action"], "a missing text falls back to the embedded copy")
	assert.Equal(t, true, m["needs_review"])

	e.exec(t, "UPDATE message_contents SET is_approved = 0 WHERE `group` = 'menopause_message'")
	assert.Equal(t, "Your symptom score went up", e.get(t, tok, "en").message("score_worsened")["title"])
}

func TestMessages_UserIsolation(t *testing.T) {
	e := setup(t)
	a, _ := e.menopauseUser(t, "09120000008", "meno", "2025-01-01")
	_, tokB := e.menopauseUser(t, "09120000009", "meno", "2025-01-01")
	e.spotting(t, a, "2026-09-30")
	e.score(t, a, "2026-08-23", 5)
	e.score(t, a, "2026-09-23", 20)
	e.hrt(t, a, "Estradiol", "2026-10-02")
	e.doneLongAgo(t, a, "meno_blood_sugar")

	r := e.get(t, tokB, "en")
	assert.Equal(t, []string{"checkup_due", "stage_meno"}, r.keys(), "B sees nothing of A's logs, scores, treatment or records")
}

func jsonNum(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
