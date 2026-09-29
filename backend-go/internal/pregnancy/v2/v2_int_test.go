package v2_test

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
	"sync/atomic"
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
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type countingDB struct {
	*sql.DB
	n atomic.Int64
}

func (d *countingDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	d.n.Add(1)
	return d.DB.ExecContext(ctx, q, args...)
}

func (d *countingDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	d.n.Add(1)
	return d.DB.QueryContext(ctx, q, args...)
}

func (d *countingDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	d.n.Add(1)
	return d.DB.QueryRowContext(ctx, q, args...)
}

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	counter *countingDB
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
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	counter := &countingDB{DB: db}
	h := v2.NewHandlers(store.New(counter), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	const p = "/api/v1/pregnancy/v2"
	app.Get(p+"/setup-copy", locale, guard, h.SetupCopy)
	app.Post(p+"/dating-preview", locale, guard, h.DatingPreview)
	app.Get(p+"/today", locale, guard, h.Today)
	app.Get(p+"/weeks/:n", locale, guard, h.Week)
	app.Put(p+"/weeks/:n/state", locale, guard, h.WeekState)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), counter: counter}
}

// user creates a user; profileSQL (columns, values) != "" also gives them a pregnancy profile.
func (e *env) user(t *testing.T, mobile, cols, vals string, args ...any) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if cols != "" {
		_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, `+cols+`, created_at, updated_at)
			VALUES (?, `+vals+`, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, append([]any{id}, args...)...)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

func (e *env) lmpUser(t *testing.T, mobile, lmp string) (uint64, string) {
	return e.user(t, mobile, "pregnancy_mode, age_source, lmp_date", "1, 'lmp', ?", lmp)
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

func (e *env) do(t *testing.T, method, path, token, lang, body string) response {
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
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

const base = "/api/v1/pregnancy/v2"

func TestPreview_LMPvsUltrasound(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000701", "", "")

	r := e.do(t, http.MethodPost, base+"/dating-preview", tok, "fa", `{"source":"lmp","lmp_date":"2026-07-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 12, d["weeks"])
	assert.EqualValues(t, 0, d["days"])
	assert.Equal(t, "2027-04-07", d["due_date"])
	assert.Equal(t, map[string]any{"level": "medium", "label": "متوسط"}, d["confidence"])
	assert.EqualValues(t, 3, d["uncertainty_days"])
	assert.Equal(t, map[string]any{"from": "2027-03-21", "to": "2027-04-24"}, d["range"])
	assert.Contains(t, d["basis"], "۱۰ تیر ۱۴۰۵")

	// 8w2d on 2026-09-09 → 10w2d today, high confidence.
	r = e.do(t, http.MethodPost, base+"/dating-preview", tok, "en",
		`{"source":"ultrasound","ultrasound_date":"2026-09-09","ultrasound_weeks":8,"ultrasound_days":2}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.EqualValues(t, 10, d["weeks"])
	assert.EqualValues(t, 2, d["days"])
	assert.Equal(t, "high", d["confidence"].(map[string]any)["level"])
	assert.Equal(t, "This is based on your ultrasound of September 9, 2026.", d["basis"])

	// Nothing written.
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_profiles`).Scan(&n))
	assert.Zero(t, n)
}

func TestPreview_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000702", "", "")
	r := e.do(t, http.MethodPost, base+"/dating-preview", tok, "en", `{"source":"ultrasound"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	errs, _ := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "ultrasound_date")

	r = e.do(t, http.MethodPost, base+"/dating-preview", tok, "en", `{"source":"lmp","lmp_date":"2025-01-01"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	errs, _ = r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "lmp_date")
}

func TestToday(t *testing.T) {
	e := setup(t)
	uid, tok := e.lmpUser(t, "09120000703", "2026-07-01") // 12w0d → week 13
	_, err := e.db.Exec(`INSERT INTO pregnancy_week_user_state (user_id, week, bookmarked, done_task_keys, created_at, updated_at)
		SELECT ?, 13, 0, JSON_ARRAY(JSON_UNQUOTE(JSON_EXTRACT(tasks, '$[0].key'))), NOW(), NOW() FROM pregnancy_week_details WHERE week_number = 13`, uid)
	require.NoError(t, err)

	e.counter.n.Store(0)
	r := e.do(t, http.MethodGet, base+"/today", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.LessOrEqual(t, e.counter.n.Load(), int64(5), "≤ 5 queries")
	d := r.data()
	assert.Equal(t, "2026-09-23", d["date"])
	assert.EqualValues(t, 12, d["weeks"])
	assert.EqualValues(t, 13, d["week"])
	carousel, _ := d["carousel"].([]any)
	require.Len(t, carousel, 3)
	assert.Equal(t, "past", carousel[0].(map[string]any)["relation"])
	cur := carousel[1].(map[string]any)
	assert.EqualValues(t, 13, cur["week"])
	assert.Equal(t, "current", cur["relation"])
	assert.NotEmpty(t, cur["title"])
	due := d["due"].(map[string]any)
	assert.Equal(t, "2027-04-07", due["date"])
	assert.EqualValues(t, 196, due["days_left"])
	progress := d["progress"].(map[string]any)
	assert.EqualValues(t, 30, progress["percent"])
	assert.Len(t, progress["trimesters"], 3)
	assert.Equal(t, "2026-09-30", progress["trimesters"].([]any)[1].(map[string]any)["start_date"])
	tip, _ := d["tip"].(map[string]any)
	require.NotNil(t, tip)
	assert.EqualValues(t, 13, tip["week"])
	tasks, _ := d["tasks"].([]any)
	require.NotEmpty(t, tasks)
	assert.Equal(t, true, tasks[0].(map[string]any)["done"])
	visit, _ := d["next_visit"].(map[string]any)
	require.NotNil(t, visit, "care-plan fallback")
	assert.Equal(t, "nt_scan", visit["care_item_key"])
	assert.EqualValues(t, 0, d["unread_alerts"])

	// An appointment wins over the care plan.
	_, err = e.db.Exec(`INSERT INTO reminders (user_id, type, title, recurrence, scheduled_at, meta, is_active, created_at, updated_at)
		VALUES (?, 'appointment', 'NT', 'once', '2026-09-28 09:30:00', '{"care_item_key":"nt_scan","stage":"booked"}', 1, NOW(), NOW())`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, base+"/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	visit = r.data()["next_visit"].(map[string]any)
	assert.Equal(t, "2026-09-28", visit["date"])
	assert.Equal(t, "09:30", visit["time"])
	assert.Equal(t, "booked", visit["stage"])
	assert.EqualValues(t, 5, visit["days_until"])

	// Review #1 (T-M2-34): the reminder bell (is_active) off keeps the visit; only cancel drops it.
	_, err = e.db.Exec(`UPDATE reminders SET is_active = 0 WHERE user_id = ? AND type = 'appointment'`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, base+"/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-09-28", r.data()["next_visit"].(map[string]any)["date"])
	_, err = e.db.Exec(`UPDATE reminders SET meta = JSON_SET(meta, '$.status', 'cancelled') WHERE user_id = ? AND type = 'appointment'`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, base+"/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["next_visit"].(map[string]any)["appointment_id"], "cancelled → care-plan fallback")
}

func TestToday_Overdue(t *testing.T) {
	e := setup(t)
	_, tok := e.lmpUser(t, "09120000704", "2025-12-10") // 41w0d
	r := e.do(t, http.MethodGet, base+"/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 42, d["week"])
	due := d["due"].(map[string]any)
	assert.EqualValues(t, 0, due["days_left"])
	assert.EqualValues(t, 7, due["overdue_days"])
	assert.Len(t, d["carousel"], 2, "no week 43")
}

func TestNotActive(t *testing.T) {
	e := setup(t)
	_, none := e.user(t, "09120000705", "", "")
	_, off := e.user(t, "09120000706", "pregnancy_mode, age_source, lmp_date", "0, 'lmp', '2026-07-01'")
	for _, tok := range []string{none, off} {
		for _, path := range []string{"/today", "/weeks/10"} {
			r := e.do(t, http.MethodGet, base+path, tok, "en", "")
			require.Equal(t, http.StatusConflict, r.status, r.raw)
			assert.Equal(t, "pregnancy_not_active", r.body["error_code"])
			assert.Equal(t, false, r.body["success"])
		}
	}
	r := e.do(t, http.MethodGet, base+"/today", "", "en", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestWeekAndState(t *testing.T) {
	e := setup(t)
	_, tok := e.lmpUser(t, "09120000707", "2026-07-01")
	_, other := e.lmpUser(t, "09120000708", "2026-07-01")

	for _, n := range []string{"0", "43", "x"} {
		r := e.do(t, http.MethodGet, base+"/weeks/"+n, tok, "en", "")
		assert.Equal(t, http.StatusNotFound, r.status, n)
	}

	r := e.do(t, http.MethodGet, base+"/weeks/20", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 20, d["week"])
	assert.Equal(t, "future", d["relation"])
	assert.Equal(t, map[string]any{"from": "2026-11-11", "to": "2026-11-17"}, d["range"])
	details := d["details"].(map[string]any)
	assert.NotNil(t, details["headline"])
	assert.NotEmpty(t, details["body_symptoms"])
	tasks := d["tasks"].([]any)
	require.NotEmpty(t, tasks)
	key := tasks[0].(map[string]any)["key"].(string)

	r = e.do(t, http.MethodPut, base+"/weeks/20/state", tok, "fa", `{"bookmarked":true,"done_task_keys":["`+key+`","nope","`+key+`"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "وضعیت هفته ذخیره شد", r.body["message"])
	assert.Equal(t, map[string]any{"week": float64(20), "bookmarked": true, "done_task_keys": []any{key}}, r.data())

	// Partial: bookmark only, tasks kept.
	r = e.do(t, http.MethodPut, base+"/weeks/20/state", tok, "en", `{"bookmarked":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{key}, r.data()["done_task_keys"])

	r = e.do(t, http.MethodGet, base+"/weeks/20", tok, "en", "")
	assert.Equal(t, true, r.data()["tasks"].([]any)[0].(map[string]any)["done"])

	// IDOR: the other user sees their own (empty) state.
	r = e.do(t, http.MethodGet, base+"/weeks/20", other, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["bookmarked"])
	assert.Equal(t, []any{}, r.data()["done_task_keys"])

	r = e.do(t, http.MethodPut, base+"/weeks/20/state", tok, "en", `{"bookmarked":"maybe"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

// GET /setup-copy (T-M7-20, design audit A1): the admin-edited pregnancy_setup texts in the request
// locale, the default language's row where the locale has none, null / [] where no row has a text;
// no pregnancy needed; one query; 401 without a session.
func TestSetupCopy(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000720", "", "") // no pregnancy profile: Setup runs before activation

	assert.Equal(t, http.StatusUnauthorized, e.do(t, http.MethodGet, base+"/setup-copy", "", "en", "").status)

	e.counter.n.Store(0)
	r := e.do(t, http.MethodGet, base+"/setup-copy", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, int64(1), e.counter.n.Load(), "one query")
	d := r.data()
	for _, k := range []string{"welcome", "dating", "source_lmp", "source_ultrasound", "source_manual", "history", "result"} {
		assert.Contains(t, d, k)
	}
	welcome := d["welcome"].(map[string]any)
	assert.Equal(t, "به حالت بارداری خوش اومدی", welcome["title"])
	benefits, _ := welcome["benefits"].([]any)
	require.Len(t, benefits, 3)
	assert.Contains(t, benefits[0], "ابزارهای بارداری همیشه رایگان")
	assert.Equal(t, "فعلاً نه", welcome["secondary"])
	assert.Contains(t, d["result"].(map[string]any)["range"], "{range_from}", "templates are returned unfilled")
	assert.NotContains(t, d, "calendar_note")

	// An admin edit shows at once; an item without an en row falls back to the default language (fa);
	// a text no row has is null and a missing list [].
	_, err := e.db.Exec("UPDATE message_contents SET payload = JSON_SET(payload, '$.title', 'Hello, pregnancy mode') " +
		"WHERE `group` = 'pregnancy_setup' AND item_key = 'welcome' AND locale = 'en'")
	require.NoError(t, err)
	_, err = e.db.Exec("DELETE FROM message_contents WHERE `group` = 'pregnancy_setup' AND item_key = 'dating' AND locale = 'en'")
	require.NoError(t, err)
	_, err = e.db.Exec("DELETE FROM message_contents WHERE `group` = 'pregnancy_setup' AND item_key = 'history'")
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, base+"/setup-copy", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "Hello, pregnancy mode", d["welcome"].(map[string]any)["title"])
	assert.Equal(t, "Not now", d["welcome"].(map[string]any)["secondary"])
	assert.Contains(t, d["dating"].(map[string]any)["title"], "تنظیم کنیم", "fa fallback")
	assert.Equal(t, map[string]any{"title": nil, "body": nil, "disclaimer": nil, "skip": nil}, d["history"])
	assert.Equal(t, "Ultrasound", d["source_ultrasound"].(map[string]any)["label"])
}
