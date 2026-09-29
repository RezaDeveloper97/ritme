package checkups_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
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
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	// Wednesday 2026-09-23 = 1 Mehr 1405.
	now = "2026-09-23T10:00:00+03:30"
)

// Seeded catalog ids (00003_checkups.sql).
const (
	selfExam      = 1
	clinicalExam  = 2
	papSmear      = 3
	bloodTest     = 4
	dentist       = 5
	mammography   = 6
	catalogLength = 6
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	queries *countingDB
}

// countingDB counts the statements the checkups handlers send (the /checkups/home budget).
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
	h := checkups.NewHandlers(counter, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/checkups", locale, guard, h.List)
	app.Get("/api/v1/checkups/home", locale, guard, h.Home)
	app.Get("/api/v1/checkups/preview-next", locale, guard, h.PreviewNext)
	app.Get("/api/v1/checkups/records", locale, guard, h.ListRecords)
	app.Put("/api/v1/checkups/records/:recordId", locale, guard, h.UpdateRecord)
	app.Delete("/api/v1/checkups/records/:recordId", locale, guard, h.DestroyRecord)
	app.Post("/api/v1/checkups/custom", locale, guard, h.StoreCustom)
	app.Put("/api/v1/checkups/custom/:id", locale, guard, h.UpdateCustom)
	app.Delete("/api/v1/checkups/custom/:id", locale, guard, h.DestroyCustom)
	app.Get("/api/v1/checkups/:id", locale, guard, h.Show)
	app.Post("/api/v1/checkups/:id/records", locale, guard, h.StoreRecord)
	app.Put("/api/v1/checkups/:id/settings", locale, guard, h.UpdateSettings)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), queries: counter}
}

// user creates a user; birthday != "" also gives her a profile.
func (e *env) user(t *testing.T, mobile, birthday string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if birthday != "" {
		_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, birthday, created_at, updated_at)
			VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, birthday)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

func (e *env) record(t *testing.T, userID uint64, typeID int, doneOn string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
		VALUES (?, ?, ?, 'normal', NOW(), NOW())`, userID, typeID, doneOn)
	require.NoError(t, err)
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

func (r response) errors() map[string]any {
	d, _ := r.body["errors"].(map[string]any)
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

// home reads GET /checkups/home and asserts the query budget.
func (e *env) home(t *testing.T, token, lang string) response {
	t.Helper()
	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, "/api/v1/checkups/home", token, lang, "")
	if r.status == http.StatusOK {
		assert.Equal(t, int64(3), e.queries.n.Load(), "query budget: profile, cycle history, plan")
	}
	return r
}

func itemsByID(t *testing.T, list any) map[int]map[string]any {
	t.Helper()
	out := map[int]map[string]any{}
	for _, x := range list.([]any) {
		m := x.(map[string]any)
		out[int(m["id"].(float64))] = m
	}
	return out
}

// fixture: 34 years old; blood test in February, dentist in January (overdue), nothing else.
func (e *env) fixture(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	id, tok := e.user(t, mobile, "1992-06-15")
	e.record(t, id, bloodTest, "2026-02-01")
	e.record(t, id, dentist, "2026-01-10")
	return id, tok
}

func TestList_Fixture(t *testing.T) {
	e := setup(t)
	_, tok := e.fixture(t, "09120000001")

	r := e.do(t, http.MethodGet, "/api/v1/checkups", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 34, d["age"])
	assert.Equal(t, map[string]any{"total": 6.0, "up_to_date": 2.0, "due": 3.0, "overdue": 1.0}, d["summary"])
	items := itemsByID(t, d["items"])
	require.Len(t, items, catalogLength)

	blood := items[bloodTest]
	assert.Equal(t, "up_to_date", blood["status"])
	assert.Equal(t, "annual", blood["section"])
	assert.Equal(t, "هر سال", blood["interval_label"])
	assert.Equal(t, "2026-02-01", blood["last_done_on"])
	assert.Equal(t, "2027-02-01", blood["next_due_on"])
	assert.Equal(t, "بهمن ۱۴۰۵", blood["next_due_label"])
	assert.Equal(t, "blood_test", blood["key"])
	assert.Equal(t, false, blood["is_custom"])
	assert.Nil(t, blood["timing_label"])

	dent := items[dentist]
	assert.Equal(t, "overdue", dent["status"])
	assert.Equal(t, "overdue", dent["section"])
	assert.Equal(t, "هر ۶ ماه", dent["interval_label"])
	assert.Equal(t, "عقب\u200cافتاده از تیر", dent["next_due_label"])

	mammo := items[mammography]
	assert.Equal(t, "not_yet", mammo["status"])
	assert.Equal(t, "2032-06-15", mammo["next_due_on"])
	assert.Equal(t, "از ۱۴۱۱ (۴۰ سالگی)", mammo["next_due_label"])
	assert.Equal(t, "هر ۱ تا ۲ سال", mammo["interval_label"])

	self := items[selfExam]
	assert.Equal(t, "due", self["status"])
	assert.Equal(t, "هر ماه", self["interval_label"])
	assert.Equal(t, "روز ۷ تا ۱۰ سیکل", self["timing_label"])
	assert.Equal(t, "زمانش رسیده", self["next_due_label"], "never recorded, no cycle data")
	assert.Equal(t, "this_month", self["section"])

	pap := items[papSmear]
	assert.Equal(t, "due", pap["status"])
	assert.Equal(t, "هر ۳ سال", pap["interval_label"])
	assert.Equal(t, "روز ۱۰ تا ۲۰ سیکل", pap["timing_label"])

	r = e.do(t, http.MethodGet, "/api/v1/checkups", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	items = itemsByID(t, r.data()["items"])
	assert.Equal(t, "Overdue since July", items[dentist]["next_due_label"])
	assert.Equal(t, "February 2027", items[bloodTest]["next_due_label"])
	assert.Equal(t, "From 2032 (age 40)", items[mammography]["next_due_label"])
	assert.Equal(t, "Every 1–2 years", items[mammography]["interval_label"])
	assert.Equal(t, "Cycle day 7–10", items[selfExam]["timing_label"])
	assert.Equal(t, "Every 6 months", items[dentist]["interval_label"])

	r = e.do(t, http.MethodGet, "/api/v1/checkups?filter=action", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Len(t, r.data()["items"], 4, "due ×3 + overdue")
	assert.EqualValues(t, 6, r.data()["summary"].(map[string]any)["total"], "the summary ignores the filter")

	r = e.do(t, http.MethodGet, "/api/v1/checkups?filter=done", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	done := itemsByID(t, r.data()["items"])
	assert.Len(t, done, 1)
	assert.Contains(t, done, bloodTest)

	r = e.do(t, http.MethodGet, "/api/v1/checkups?filter=soon", tok, "fa", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Contains(t, r.errors(), "filter")
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, path := range []string{"/api/v1/checkups", "/api/v1/checkups/home", "/api/v1/checkups/1", "/api/v1/checkups/records"} {
		r := e.do(t, http.MethodGet, path, "", "fa", "")
		assert.Equal(t, http.StatusUnauthorized, r.status, path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], path)
	}
}

func TestHome(t *testing.T) {
	e := setup(t)
	id, tok := e.fixture(t, "09120000002")

	r := e.home(t, tok, "fa")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, map[string]any{"total": 6.0, "up_to_date": 2.0, "due": 3.0, "overdue": 1.0}, d["summary"])
	hl := d["highlights"].([]any)
	require.Len(t, hl, 2)
	assert.EqualValues(t, selfExam, hl[0].(map[string]any)["id"], "cycle-timed first")
	assert.EqualValues(t, papSmear, hl[1].(map[string]any)["id"])

	// Pregnant: the self-exam and mammography leave the plan; the overdue dentist comes up.
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, id)
	require.NoError(t, err)
	r = e.home(t, tok, "fa")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 4, r.data()["summary"].(map[string]any)["total"])
	hl = r.data()["highlights"].([]any)
	assert.EqualValues(t, papSmear, hl[0].(map[string]any)["id"])
	assert.EqualValues(t, dentist, hl[1].(map[string]any)["id"], "overdue before due")

	// Everything switched off → nothing applies → data null.
	for typeID := 1; typeID <= catalogLength; typeID++ {
		r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", typeID), tok, "fa", `{"enabled":false}`)
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	r = e.home(t, tok, "fa")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"data":null}`, r.raw)

	// A brand-new user without a profile still gets the whole catalog.
	_, tok2 := e.user(t, "09120000003", "")
	r = e.home(t, tok2, "en")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 6, r.data()["summary"].(map[string]any)["total"])
}

func TestShow_AndAdminEditVisible(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000004", "1992-06-15")
	e.record(t, id, selfExam, "2026-06-01")
	e.record(t, id, selfExam, "2026-07-01")
	e.record(t, id, selfExam, "2026-08-01")

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", selfExam), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "breast_self_exam", d["key"])
	assert.NotEmpty(t, d["why"])
	assert.Equal(t, "self", d["performed_by"])
	assert.EqualValues(t, 7, d["cycle_day_from"])
	assert.EqualValues(t, 10, d["cycle_day_to"])
	assert.NotEmpty(t, d["prep_steps"])
	assert.IsType(t, "", d["prep_steps"].([]any)[0], "localized strings")
	guide := d["guide_steps"].([]any)
	require.NotEmpty(t, guide)
	assert.IsType(t, "", guide[0].(map[string]any)["title"])
	opts := d["finding_options"].([]any)
	require.NotEmpty(t, opts)
	assert.Equal(t, map[string]any{"key": "none", "label": opts[0].(map[string]any)["label"], "exclusive": true}, opts[0])
	recs := d["records"].([]any)
	require.Len(t, recs, 2, "latest two")
	assert.Equal(t, "2026-08-01", recs[0].(map[string]any)["done_on"])
	assert.Equal(t, "2026-07-01", recs[1].(map[string]any)["done_on"])
	assert.Equal(t, map[string]any{"enabled": true, "remind": true}, d["settings"])

	// An admin edit (T-M4-03's UpdateAdminCheckupType) shows on the next read.
	q := store.New(e.db)
	row, err := q.GetAdminCheckupType(context.Background(), bloodTest)
	require.NoError(t, err)
	require.NoError(t, q.UpdateAdminCheckupType(context.Background(), store.UpdateAdminCheckupTypeParams{
		Category: row.Category, Title: json.RawMessage(`{"fa":"آزمایش خون کامل","en":"Full blood panel"}`),
		Subtitle: row.Subtitle, Why: row.Why, PerformedBy: row.PerformedBy, Icon: row.Icon, Tone: row.Tone,
		IntervalMonths: 24, IntervalMonthsMax: row.IntervalMonthsMax, AgeMin: row.AgeMin, AgeMax: row.AgeMax,
		CycleDayFrom: row.CycleDayFrom, CycleDayTo: row.CycleDayTo, RemindLeadDays: row.RemindLeadDays,
		PrepSteps: row.PrepSteps, GuideSteps: row.GuideSteps, FindingOptions: row.FindingOptions,
		HideInPregnancy: row.HideInPregnancy, IsActive: row.IsActive, SortOrder: row.SortOrder,
		SourceNote: row.SourceNote, Now: sql.NullTime{Time: time.Now(), Valid: true}, ID: bloodTest,
	}))
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", bloodTest), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Full blood panel", r.data()["title"])
	assert.Equal(t, "Every 2 years", r.data()["interval_label"])
	assert.EqualValues(t, 24, r.data()["interval_months"])

	// Not found: unknown, malformed, another user's custom checkup, an inactive type.
	_, bobTok := e.user(t, "09120000005", "")
	r = e.do(t, http.MethodPost, "/api/v1/checkups/custom", bobTok, "fa", `{"title":"فیزیوتراپی","interval_months":6,"performed_by":"doctor"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	bobCustom := int(r.data()["id"].(float64))
	_, err = e.db.Exec(`UPDATE checkup_types SET is_active = 0 WHERE id = ?`, dentist)
	require.NoError(t, err)
	for _, path := range []string{"/api/v1/checkups/9999", "/api/v1/checkups/abc", fmt.Sprintf("/api/v1/checkups/%d", bobCustom), fmt.Sprintf("/api/v1/checkups/%d", dentist)} {
		r = e.do(t, http.MethodGet, path, tok, "fa", "")
		assert.Equal(t, http.StatusNotFound, r.status, path)
		assert.Equal(t, "چکاپ پیدا نشد", r.body["message"], path)
	}
}

func TestRecords_CRUDAndValidation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000006", "1992-06-15")

	// Findings: only the type's own option keys.
	r := e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", selfExam), tok, "fa",
		`{"done_on":"2026-09-24","result":"great","findings":["lump","nope"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "اطلاعات واردشده نامعتبر است", r.body["message"])
	errs := r.errors()
	assert.Equal(t, []any{"تاریخ نمی\u200cتواند در آینده باشد."}, errs["done_on"])
	assert.Contains(t, errs, "result")
	assert.Equal(t, []any{"این مورد جزو گزینه\u200cهای این چکاپ نیست."}, errs["findings.1"])
	assert.NotContains(t, errs, "findings.0")

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", bloodTest), tok, "en",
		`{"done_on":"2026-09-01","result":"normal","findings":["lump"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "findings.0", "a type without options accepts no findings")

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", selfExam), tok, "fa",
		`{"done_on":"2026-09-20","result":"follow_up","findings":["lump","lump"],"note":" ","has_attachment":true,"ignored":1}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "انجام چکاپ ثبت شد", r.body["message"])
	rec := r.data()["record"].(map[string]any)
	recID := int(rec["id"].(float64))
	assert.EqualValues(t, selfExam, rec["checkup_type_id"])
	assert.Equal(t, "2026-09-20", rec["done_on"])
	assert.Equal(t, "follow_up", rec["result"])
	assert.Equal(t, []any{"lump"}, rec["findings"])
	assert.Nil(t, rec["note"])
	assert.Equal(t, true, rec["has_attachment"])
	assert.Nil(t, rec["next_due_on"])
	assert.NotEmpty(t, rec["checkup_title"])
	item := r.data()["item"].(map[string]any)
	assert.Equal(t, "2026-09-20", item["last_done_on"])
	assert.Equal(t, "soon", item["status"], "next due a month later without cycle data")
	assert.Equal(t, "2026-10-20", item["next_due_on"])

	// Partial update: only has_attachment changes (the attachment rollback).
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", `{"has_attachment":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rec = r.data()["record"].(map[string]any)
	assert.Equal(t, false, rec["has_attachment"])
	assert.Equal(t, []any{"lump"}, rec["findings"])
	assert.Equal(t, "follow_up", rec["result"])

	// The user's next-due override wins.
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", `{"next_due_on":"2026-11-05","note":"پیگیری"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-11-05", r.data()["item"].(map[string]any)["next_due_on"])
	assert.Equal(t, "پیگیری", r.data()["record"].(map[string]any)["note"])
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", `{"next_due_on":"2026-09-01"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "next_due_on", "must be after done_on")

	// Another user can neither edit nor delete it, nor record on her custom type.
	_, bobTok := e.user(t, "09120000007", "")
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), bobTok, "fa", `{"result":"normal"}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Equal(t, "سابقه چکاپ پیدا نشد", r.body["message"])
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/records/%d", recID), bobTok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/checkups/custom", tok, "fa", `{"title":"فیزیوتراپی","interval_months":6,"performed_by":"doctor"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	custom := int(r.data()["id"].(float64))
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", custom), bobTok, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)

	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"message":"سابقه چکاپ حذف شد","data":null}`, r.raw)
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
}

func TestRecords_List(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000008", "")
	e.record(t, id, dentist, "2025-12-01")   // Azar 1404: last Jalali year, this Gregorian year? no — 2025
	e.record(t, id, dentist, "2026-03-25")   // 5 Farvardin 1405: this Jalali year
	e.record(t, id, bloodTest, "2026-02-01") // Bahman 1404: 2026 but not 1405
	for i := 1; i <= 21; i++ {               // 21 self-exams → two pages
		e.record(t, id, selfExam, fmt.Sprintf("2024-%02d-%02d", (i-1)/2+1, 1+(i%2)*10))
	}
	_, err := e.db.Exec(`UPDATE checkup_records SET has_attachment = 1 WHERE user_id = ? AND done_on = '2026-02-01'`, id)
	require.NoError(t, err)
	bob, _ := e.user(t, "09120000009", "")
	e.record(t, bob, dentist, "2026-09-01")

	r := e.do(t, http.MethodGet, "/api/v1/checkups/records", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	items := d["items"].([]any)
	require.Len(t, items, 20)
	first := items[0].(map[string]any)
	assert.Equal(t, "2026-03-25", first["done_on"], "newest first")
	assert.Equal(t, "dentist", first["checkup_key"])
	assert.Equal(t, "tooth", first["checkup_icon"])
	assert.Equal(t, "green", first["checkup_tone"], "00007 catalog tone")
	assert.NotEmpty(t, first["checkup_title"])
	assert.Equal(t, map[string]any{"current_page": 1.0, "last_page": 2.0, "per_page": 20.0, "total": 24.0}, d["meta"])

	r = e.do(t, http.MethodGet, "/api/v1/checkups/records?page=2", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Len(t, r.data()["items"], 4)

	dates := func(r response) []any {
		out := []any{}
		for _, x := range r.data()["items"].([]any) {
			out = append(out, x.(map[string]any)["done_on"])
		}
		return out
	}
	r = e.do(t, http.MethodGet, "/api/v1/checkups/records?filter=this_year", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"2026-03-25"}, dates(r), "Jalali year 1405")
	r = e.do(t, http.MethodGet, "/api/v1/checkups/records?filter=this_year", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"2026-03-25", "2026-02-01"}, dates(r), "Gregorian 2026")
	r = e.do(t, http.MethodGet, "/api/v1/checkups/records?filter=with_attachment", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"2026-02-01"}, dates(r))
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/records?type=%d", dentist), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"2026-03-25", "2025-12-01"}, dates(r), "bob's dentist record is not listed")

	r = e.do(t, http.MethodGet, "/api/v1/checkups/records?filter=bad&type=x", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "filter")
	assert.Contains(t, r.errors(), "type")
}

func TestPreviewNext(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000010", "1992-06-15")

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/preview-next?type=%d&done_on=2026-09-20", papSmear), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"data":{"next_due_on":"2029-09-20","due_by":"2029-09-20",`+
		`"next_due_label":"۳۰ شهریور ۱۴۰۸","interval_label":"هر ۳ سال","timing_label":"روز ۱۰ تا ۲۰ سیکل",`+
		`"reminder_label":"۳۰ روز قبل یادآوری می\u200cکنیم","cycle_timed":false}}`, r.raw)

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/preview-next?type=%d&done_on=2026-01-31", mammography), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2027-01-31", d["next_due_on"])
	assert.Equal(t, "2028-01-31", d["due_by"])
	assert.Equal(t, "January 31, 2027", d["next_due_label"])

	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", dentist), tok, "fa", `{"remind":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/preview-next?type=%d&done_on=2026-09-01", dentist), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Reminders are off", r.data()["reminder_label"])
	assert.Equal(t, "2027-03-01", r.data()["next_due_on"])

	r = e.do(t, http.MethodGet, "/api/v1/checkups/preview-next?type=999&done_on=2026-09-01", tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/checkups/preview-next?done_on=2026-12-01", tok, "fa", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "type")
	assert.Contains(t, r.errors(), "done_on")
}

func TestCustom_CRUD(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000011", "1992-06-15")

	r := e.do(t, http.MethodPost, "/api/v1/checkups/custom", tok, "fa", `{"title":"","interval_months":0,"performed_by":"nurse","last_done_on":"2026-10-01"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	for _, k := range []string{"title", "interval_months", "performed_by", "last_done_on"} {
		assert.Contains(t, r.errors(), k)
	}

	r = e.do(t, http.MethodPost, "/api/v1/checkups/custom", tok, "fa",
		`{"title":"چشم\u200cپزشکی","interval_months":24,"performed_by":"doctor","note":"عینک","last_done_on":"2026-03-01"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "چکاپ سفارشی اضافه شد", r.body["message"])
	d := r.data()
	id := int(d["id"].(float64))
	assert.Equal(t, "چشم\u200cپزشکی", d["title"])
	assert.Equal(t, "عینک", d["subtitle"])
	assert.Equal(t, "custom", d["category"])
	assert.Equal(t, "custom", d["section"])
	assert.Equal(t, true, d["is_custom"])
	assert.Nil(t, d["key"])
	assert.Equal(t, "stethoscope", d["icon"])
	assert.Equal(t, "neutral", d["tone"])
	assert.Equal(t, "هر ۲ سال", d["interval_label"])
	assert.Equal(t, "2026-03-01", d["last_done_on"], "last_done_on seeds a record")
	assert.Equal(t, "2028-03-01", d["next_due_on"])
	assert.Equal(t, "up_to_date", d["status"])

	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/custom/%d", id), tok, "fa", `{"interval_months":12,"note":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "چشم\u200cپزشکی", d["title"], "untouched")
	assert.Nil(t, d["subtitle"])
	assert.Equal(t, "هر سال", d["interval_label"])
	assert.Equal(t, "2027-03-01", d["next_due_on"])

	// Only her: another user's custom id is 404 on every verb, and not in her list.
	_, bobTok := e.user(t, "09120000012", "")
	for _, m := range []string{http.MethodPut, http.MethodDelete} {
		r = e.do(t, m, fmt.Sprintf("/api/v1/checkups/custom/%d", id), bobTok, "fa", `{"title":"x"}`)
		assert.Equal(t, http.StatusNotFound, r.status, m)
	}
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", id), bobTok, "fa", `{"enabled":false}`)
	assert.Equal(t, http.StatusNotFound, r.status)
	r = e.do(t, http.MethodGet, "/api/v1/checkups", bobTok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.NotContains(t, itemsByID(t, r.data()["items"]), id)
	// A catalog type is not "custom".
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/custom/%d", dentist), tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status)

	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/custom/%d", id), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM checkup_records WHERE checkup_type_id = ?`, id).Scan(&n))
	assert.Zero(t, n, "records cascade")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", id), tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status)
}

func TestSettings(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000013", "1992-06-15")

	r := e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", dentist), tok, "fa", `{"enabled":"yes"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "enabled")

	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", dentist), tok, "fa", `{"enabled":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"enabled": false, "remind": true}, r.data()["settings"])
	assert.Equal(t, "disabled", r.data()["item"].(map[string]any)["status"])
	assert.Equal(t, "غیرفعال", r.data()["item"].(map[string]any)["next_due_label"])

	r = e.do(t, http.MethodGet, "/api/v1/checkups", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 5, r.data()["summary"].(map[string]any)["total"], "a disabled type leaves the summary")
	assert.Len(t, r.data()["items"], 6, "but stays listed")

	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", dentist), tok, "fa", `{"remind":0}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"enabled": false, "remind": false}, r.data()["settings"], "partial")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", dentist), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"enabled": false, "remind": false}, r.data()["settings"])
}

// With a cycle, the never-recorded self-exam is placed on the next predicted window.
func TestList_CycleTimedSelfExam(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000014", "1992-06-15")
	_, err := e.db.Exec(`UPDATE user_profiles SET last_period_start = '2026-09-10', cycle_duration = 28, period_duration = 5 WHERE user_id = ?`, id)
	require.NoError(t, err)

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", selfExam), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "soon", d["status"], "never recorded, but the window is weeks away (audit 5d)")
	assert.Equal(t, "2026-10-14", d["next_due_on"], "cycle day 7 of the cycle starting 2026-10-08")
	assert.Equal(t, "۲۱ روز دیگر", d["next_due_label"])
	assert.Equal(t, "this_month", d["section"], "inside Mehr 1405")
}
