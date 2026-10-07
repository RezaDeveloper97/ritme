package telemed_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/telemed"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

// fixed is the user-side request clock: Tuesday 2030-01-08 09:00 Tehran.
var fixed = time.Date(2030, 1, 8, 9, 0, 0, 0, civildate.Tehran)

// visits is a fake VisitChecker: user id → the booking id it may review.
type visits map[uint64]uint64

func (v visits) ReviewableVisit(_ context.Context, userID, _ uint64) (uint64, bool, error) {
	id, ok := v[userID]
	return id, ok, nil
}

type env struct {
	*admintest.Env
	iss    *passport.Issuer
	visits visits
}

const appURL = "https://api.ritme.test"

func setup(t *testing.T) *env {
	t.Helper()
	e := admintest.New(t)
	e.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(e.DB)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, admintest.Quiet).RequireUser
	locale := i18n.Middleware(e.Registry)
	reader := catalog.NewReader(catalogstore.New(e.DB), nil, 0, admintest.Quiet)
	vs := visits{}
	svc := telemed.NewService(e.DB, reader, nil, vs)
	h := telemed.NewHandlers(svc, clock.Fixed(fixed), appURL)

	p := "/api/v1/telemed/doctors"
	e.App.Get(p, locale, guard, h.List)
	e.App.Get(p+"/filters", locale, guard, h.Filters)
	e.App.Get(p+"/:id", locale, guard, h.Show)
	e.App.Get(p+"/:id/slots", locale, guard, h.Slots)
	e.App.Get(p+"/:id/reviews", locale, guard, h.Reviews)
	e.App.Post(p+"/:id/reviews", locale, guard, h.StoreReview)
	e.App.Delete(p+"/:id/reviews/:review", locale, guard, h.DestroyReview)

	telemed.NewAdmin(svc, media.NewDisk(e.Storage), media.DefaultOptimizer, appURL, admintest.Quiet).Routes(e.Route(), e.Kit)
	return &env{Env: e, iss: passport.NewIssuer(key, q, clock.Real{}, 365), visits: vs}
}

func (e *env) user(t *testing.T, name, mobile string) (uint64, string) {
	t.Helper()
	res := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2029-09-01 09:00:00', '2029-09-01 09:00:00')`, name, mobile)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

type response struct {
	status int
	body   map[string]any
}

func (r response) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func (r response) items() []any { d, _ := r.data()["items"].([]any); return d }

func (r response) obj(key string) map[string]any { d, _ := r.data()[key].(map[string]any); return d }

func (e *env) do(t *testing.T, method, path, token, lang string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, rd)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	res, err := e.App.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, body: m}
}

func id(v any) uint64 { return uint64(v.(float64)) }

// doctor creates a doctor through the admin API and returns its id.
func (e *env) doctor(t *testing.T, body map[string]any) uint64 {
	t.Helper()
	r := e.As(admintest.EditorID).JSON(http.MethodPost, "/telemed/doctors", body)
	require.Equal(t, http.StatusCreated, r.Status, r.Body)
	return id(r.Obj("doctor")["id"])
}

func (e *env) put(t *testing.T, path string, body any) admintest.Resp {
	t.Helper()
	r := e.As(admintest.EditorID).JSON(http.MethodPut, path, body)
	require.Equal(t, http.StatusOK, r.Status, r.Body)
	return r
}

// seed builds the directory of the main test: A (gynaecology, Tehran, insurer, video + in person on Tuesday evenings,
// time off 17–18 on the fixed Tuesday), B (midwife, phone on Wednesday mornings), C (inactive) and D (no visit type).
func (e *env) seed(t *testing.T) (a, b, c, d uint64) {
	t.Helper()
	e.Exec(`INSERT INTO catalog_items (` + "`group`" + `, code, sort_order, is_active, title, needs_review, created_at, updated_at) VALUES
		('telemed_cities', 'test_city', 1, 1, '{"fa":"شهر آزمون","en":"Test city"}', 0, NOW(), NOW()),
		('telemed_insurers', 'test_insurer', 1, 1, '{"fa":"بیمه آزمون","en":"Test insurer"}', 0, NOW(), NOW())`)
	a = e.doctor(t, map[string]any{
		"kind": "doctor", "name": map[string]any{"fa": "آزمون الف", "en": "Test Doctor A"},
		"headline": map[string]any{"fa": "سرتیتر آزمون"}, "bio": map[string]any{"fa": "متن آزمون"},
		"specialty": "gynecology", "city": "test_city", "licence_no": "T-0001", "experience_years": 9,
		"response_minutes": 60, "insurers": []any{"test_insurer"},
	})
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/visit-types", a), map[string]any{"visit_types": []any{
		map[string]any{"mode": "video", "duration_minutes": 20, "price_rials": 2_000_000, "note": map[string]any{"fa": "یادداشت آزمون"}},
		map[string]any{"mode": "in_person", "duration_minutes": 30, "price_rials": 3_000_000, "address": map[string]any{"fa": "نشانی آزمون"}},
	}})
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/availability", a), map[string]any{"rules": []any{
		map[string]any{"weekday": 2, "start_time": "16:00", "end_time": "20:00", "slot_minutes": 30},
	}})
	r := e.As(admintest.EditorID).JSON(http.MethodPost, fmt.Sprintf("/telemed/doctors/%d/time-off", a),
		map[string]any{"starts_at": "2030-01-08 17:00", "ends_at": "2030-01-08 18:00", "note": "test"})
	require.Equal(t, http.StatusCreated, r.Status, r.Body)

	b = e.doctor(t, map[string]any{
		"kind": "midwife", "name": map[string]any{"fa": "آزمون ب", "en": "Test Midwife B"},
		"specialty": "midwifery", "licence_no": "T-0002",
	})
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/visit-types", b), map[string]any{"visit_types": []any{
		map[string]any{"mode": "phone", "duration_minutes": 15, "price_rials": 1_000_000},
	}})
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/availability", b), map[string]any{"rules": []any{
		map[string]any{"weekday": 3, "start_time": "10:00", "end_time": "12:00", "slot_minutes": 30, "modes": []any{"phone"}},
	}})

	c = e.doctor(t, map[string]any{"kind": "doctor", "name": map[string]any{"fa": "آزمون ج"}, "specialty": "general",
		"licence_no": "T-0003", "is_active": false})
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/visit-types", c), map[string]any{"visit_types": []any{
		map[string]any{"mode": "video", "duration_minutes": 20, "price_rials": 1},
	}})
	d = e.doctor(t, map[string]any{"kind": "doctor", "name": map[string]any{"fa": "آزمون د"}, "specialty": "general",
		"licence_no": "T-0004"})
	return a, b, c, d
}

func names(r response) []string {
	out := []string{}
	for _, it := range r.items() {
		out = append(out, it.(map[string]any)["name"].(string))
	}
	return out
}

func TestTelemed_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/telemed/doctors"},
		{http.MethodGet, "/api/v1/telemed/doctors/filters"},
		{http.MethodGet, "/api/v1/telemed/doctors/1"},
		{http.MethodGet, "/api/v1/telemed/doctors/1/slots"},
		{http.MethodGet, "/api/v1/telemed/doctors/1/reviews"},
		{http.MethodPost, "/api/v1/telemed/doctors/1/reviews"},
		{http.MethodDelete, "/api/v1/telemed/doctors/1/reviews/1"},
	} {
		r := e.do(t, tc.method, tc.path, "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], tc.path)
	}
	assert.Equal(t, http.StatusUnauthorized, e.Anonymous().Get("/telemed/doctors").Status)
}

func TestTelemed_DirectoryAndFilters(t *testing.T) {
	e := setup(t)
	a, b, _, _ := e.seed(t)
	_, tok := e.user(t, "Nava", "09120000001")

	r := e.do(t, http.MethodGet, "/api/v1/telemed/doctors", tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, []string{"آزمون الف", "آزمون ب"}, names(r), "inactive C and D without visit types are not listed")
	assert.InDelta(t, 2, r.obj("meta")["total"], 0)
	first := r.items()[0].(map[string]any)
	assert.InDelta(t, float64(a), first["id"], 0)
	assert.Equal(t, "doctor", first["kind"])
	assert.Equal(t, "آ", first["initial"])
	assert.Equal(t, "سرتیتر آزمون", first["headline"])
	assert.Equal(t, map[string]any{"code": "gynecology", "title": "زنان و زایمان"}, first["specialty"])
	assert.Equal(t, map[string]any{"code": "test_city", "title": "شهر آزمون"}, first["city"])
	assert.Equal(t, []any{"video", "in_person"}, first["modes"])
	assert.InDelta(t, 2_000_000, first["price_from_rials"], 0)
	assert.Nil(t, first["rating"])
	next := first["next_slot"].(map[string]any)
	assert.Equal(t, "2030-01-08T16:00:00+03:30", next["starts_at"])
	assert.Equal(t, "video", next["mode"])
	second := r.items()[1].(map[string]any)
	assert.Nil(t, second["city"])
	assert.Equal(t, "2030-01-09T10:00:00+03:30", second["next_slot"].(map[string]any)["starts_at"])

	r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors", tok, "en", nil)
	assert.Equal(t, []string{"Test Doctor A", "Test Midwife B"}, names(r))
	assert.Equal(t, "Obstetrics & gynaecology", r.items()[0].(map[string]any)["specialty"].(map[string]any)["title"])

	for q, want := range map[string][]string{
		"mode=phone":                {"آزمون ب"},
		"mode=in_person":            {"آزمون الف"},
		"today=1":                   {"آزمون الف"},
		"today=1&mode=phone":        {},
		"specialty=midwifery":       {"آزمون ب"},
		"specialty=unknown":         {},
		"city=test_city":            {"آزمون الف"},
		"insurance=any":             {"آزمون الف"},
		"insurance=test_insurer":    {"آزمون الف"},
		"insurance=other":           {},
		"kind=midwife":              {"آزمون ب"},
		"q=Midwife":                 {"آزمون ب"},
		"q=" + "%D8%A7%D9%84%D9%81": {"آزمون الف"}, // «الف»
	} {
		r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors?"+q, tok, "fa", nil)
		require.Equal(t, http.StatusOK, r.status, q)
		assert.Equal(t, want, names(r), q)
	}
	r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors?mode=video", tok, "fa", nil)
	assert.InDelta(t, 2_000_000, r.items()[0].(map[string]any)["price_from_rials"], 0)
	r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors?mode=in_person", tok, "fa", nil)
	assert.InDelta(t, 3_000_000, r.items()[0].(map[string]any)["price_from_rials"], 0, "the filtered mode's price")
	assert.Equal(t, "in_person", r.items()[0].(map[string]any)["next_slot"].(map[string]any)["mode"])

	r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors?mode=fax&kind=nurse", tok, "en", nil)
	require.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, false, r.body["success"])
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "mode")
	assert.Contains(t, errs, "kind")

	r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors/filters", tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.InDelta(t, 2, r.data()["total"], 0)
	assert.Equal(t, []any{"video", "phone", "in_person"}, r.data()["modes"])
	specs := r.data()["specialties"].([]any)
	require.Len(t, specs, 6, "the seeded catalog, in catalog order")
	assert.Equal(t, map[string]any{"code": "gynecology", "title": "زنان و زایمان", "count": float64(1)}, specs[0])
	assert.Equal(t, map[string]any{"code": "midwifery", "title": "ماما", "count": float64(1)}, specs[1])
	assert.Equal(t, []any{map[string]any{"code": "test_city", "title": "شهر آزمون", "count": float64(1)}}, r.data()["cities"])
	assert.Equal(t, []any{map[string]any{"code": "test_insurer", "title": "بیمه آزمون", "count": float64(1)}}, r.data()["insurers"])
	_ = b
}

func TestTelemed_ProfileAndSlots(t *testing.T) {
	e := setup(t)
	a, b, c, d := e.seed(t)
	_, tok := e.user(t, "Nava", "09120000001")

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", a), tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	doc := r.obj("doctor")
	assert.Equal(t, "T-0001", doc["licence_no"])
	assert.Equal(t, "متن آزمون", doc["bio"])
	assert.Equal(t, map[string]any{"experience_years": float64(9), "visits_count": float64(0), "response_minutes": float64(60),
		"satisfaction_percent": nil}, doc["stats"])
	vts := doc["visit_types"].([]any)
	require.Len(t, vts, 2)
	assert.Equal(t, map[string]any{"mode": "video", "duration_minutes": float64(20), "price_rials": float64(2_000_000),
		"note": "یادداشت آزمون", "address": nil}, vts[0])
	assert.Equal(t, "نشانی آزمون", vts[1].(map[string]any)["address"])
	assert.Equal(t, []any{map[string]any{"code": "test_insurer", "title": "بیمه آزمون"}}, doc["insurers"])
	assert.Equal(t, map[string]any{"count": float64(0), "items": []any{}}, doc["reviews"])
	assert.Equal(t, false, doc["can_review"])

	for _, missing := range []string{fmt.Sprint(c), fmt.Sprint(d), "999999", "abc", "01"} {
		r = e.do(t, http.MethodGet, "/api/v1/telemed/doctors/"+missing, tok, "en", nil)
		assert.Equal(t, http.StatusNotFound, r.status, missing)
		assert.Equal(t, "doctor_not_found", r.body["error_code"], missing)
	}

	// Slots: the Tuesday grid minus the 17–18 time off; nothing the next six days; the next Tuesday is full.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots?mode=video&days=8", a), tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "2030-01-08", r.data()["from"])
	assert.Equal(t, "video", r.obj("visit_type")["mode"])
	days := r.data()["days"].([]any)
	require.Len(t, days, 8)
	day0 := days[0].(map[string]any)
	assert.InDelta(t, 2, day0["weekday"], 0)
	got := []string{}
	for _, s := range day0["slots"].([]any) {
		got = append(got, s.(map[string]any)["time"].(string))
	}
	assert.Equal(t, []string{"16:00", "16:30", "18:00", "18:30", "19:00", "19:30"}, got)
	assert.Equal(t, "2030-01-08T16:20:00+03:30", day0["slots"].([]any)[0].(map[string]any)["ends_at"])
	assert.InDelta(t, 0, days[1].(map[string]any)["count"], 0)
	assert.InDelta(t, 8, days[7].(map[string]any)["count"], 0)
	assert.Equal(t, "2030-01-08T16:00:00+03:30", r.obj("next_slot")["starts_at"])

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots?mode=phone", a), tok, "en", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.body["errors"], "mode")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots?days=40", a), tok, "en", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots", c), tok, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)

	// No mode: the first offered one (B offers phone only); Wednesday 10:00–12:00 on a 30-minute grid.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots?from=2030-01-09&days=1", b), tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "phone", r.obj("visit_type")["mode"])
	assert.InDelta(t, 4, r.data()["days"].([]any)[0].(map[string]any)["count"], 0)

	// The admin preview answers for the inactive doctor too.
	pr := e.As(admintest.EditorID).Get(fmt.Sprintf("/telemed/doctors/%d/slots?from=2030-01-15&days=1", c))
	require.Equal(t, http.StatusOK, pr.Status, pr.Body)
}

func TestTelemed_ReviewsHookIDORAndModeration(t *testing.T) {
	e := setup(t)
	a, _, c, _ := e.seed(t)
	u1, tok1 := e.user(t, "Nava", "09120000001")
	u2, tok2 := e.user(t, "Roya", "09120000002")
	_ = u1
	reviews := fmt.Sprintf("/api/v1/telemed/doctors/%d/reviews", a)

	// No completed visit → 403 (never 401).
	r := e.do(t, http.MethodPost, reviews, tok1, "fa", map[string]any{"rating": 5})
	assert.Equal(t, http.StatusForbidden, r.status)
	assert.Equal(t, "review_not_allowed", r.body["error_code"])

	e.visits[u2] = 9001
	r = e.do(t, http.MethodPost, reviews, tok2, "en", map[string]any{"rating": 9, "body": strings.Repeat("x", 1001)})
	require.Equal(t, http.StatusUnprocessableEntity, r.status)
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "rating")
	assert.Contains(t, errs, "body")

	r = e.do(t, http.MethodPost, reviews, tok2, "fa", map[string]any{"rating": 4, "body": "خوب بود"})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "ممنون از نظرت", r.body["message"])
	rev := r.obj("review")
	revID := id(rev["id"])
	assert.Equal(t, "R", rev["author_initial"])
	assert.Equal(t, true, rev["mine"])
	assert.NotContains(t, rev, "user_id")

	r = e.do(t, http.MethodPost, reviews, tok2, "fa", map[string]any{"rating": 5})
	assert.Equal(t, http.StatusConflict, r.status, "one review per visit")
	assert.Equal(t, "review_exists", r.body["error_code"])

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/telemed/doctors/%d/reviews", c), tok2, "fa", map[string]any{"rating": 5})
	assert.Equal(t, http.StatusNotFound, r.status, "an unlisted doctor cannot be reviewed")

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", a), tok1, "fa", nil)
	doc := r.obj("doctor")
	assert.InDelta(t, 4, doc["rating"], 0)
	assert.InDelta(t, 1, doc["reviews_count"], 0)
	assert.InDelta(t, 100, doc["stats"].(map[string]any)["satisfaction_percent"], 0)
	items := doc["reviews"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, false, items[0].(map[string]any)["mine"])
	assert.Equal(t, false, doc["can_review"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", a), tok2, "fa", nil)
	assert.Equal(t, true, r.obj("doctor")["can_review"], "the fake hook still offers a visit")

	r = e.do(t, http.MethodGet, reviews+"?page=1", tok1, "fa", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.InDelta(t, 1, r.obj("meta")["total"], 0)
	assert.Equal(t, "خوب بود", r.items()[0].(map[string]any)["body"])

	// IDOR: user 1 cannot delete user 2's review; wrong doctor id is the same 404.
	r = e.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", reviews, revID), tok1, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "review_not_found", r.body["error_code"])
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/telemed/doctors/%d/reviews/%d", c, revID), tok2, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM telemed_reviews WHERE id = ?`, revID))

	// Moderation: a hidden review leaves the list and the rating.
	mr := e.As(admintest.EditorID).JSON(http.MethodPut, fmt.Sprintf("/telemed/reviews/%d", revID), map[string]any{"is_visible": false})
	require.Equal(t, http.StatusOK, mr.Status, mr.Body)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", a), tok1, "fa", nil)
	assert.Nil(t, r.obj("doctor")["rating"])
	assert.InDelta(t, 0, r.obj("doctor")["reviews_count"], 0)
	ar := e.As(admintest.EditorID).Get(fmt.Sprintf("/telemed/reviews?doctor_id=%d&visibility=hidden", a))
	require.Equal(t, http.StatusOK, ar.Status)
	require.Len(t, ar.Items(), 1)
	e.As(admintest.EditorID).JSON(http.MethodPut, fmt.Sprintf("/telemed/reviews/%d", revID), map[string]any{"is_visible": true})

	r = e.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", reviews, revID), tok2, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", a), tok1, "fa", nil)
	assert.Nil(t, r.obj("doctor")["rating"])
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255}) //nolint:gosec // G115: test pattern
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestTelemed_AdminCRUD(t *testing.T) {
	e := setup(t)
	ed := e.As(admintest.EditorID)

	// Validation: unknown catalog codes, missing name, bad kind / licence.
	r := ed.JSON(http.MethodPost, "/telemed/doctors", map[string]any{
		"kind": "nurse", "name": map[string]any{"en": "x"}, "specialty": "astrology", "city": "atlantis",
		"insurers": []any{"nobody"}, "experience_years": 99,
	})
	require.Equal(t, http.StatusUnprocessableEntity, r.Status, r.Body)
	for _, f := range []string{"kind", "specialty", "city", "insurers.0", "licence_no", "experience_years"} {
		assert.Contains(t, r.Errors(), f)
	}
	hasName := false
	for k := range r.Errors() {
		hasName = hasName || strings.HasPrefix(k, "name.")
	}
	assert.True(t, hasName, "the default-language name is required: %v", r.Errors())

	docID := e.doctor(t, map[string]any{"kind": "doctor", "name": map[string]any{"fa": "آزمون"}, "specialty": "general",
		"licence_no": "T-1"})
	r = ed.Get(fmt.Sprintf("/telemed/doctors/%d", docID))
	require.Equal(t, http.StatusOK, r.Status)
	doc := r.Obj("doctor")
	assert.Equal(t, map[string]any{"fa": "آزمون"}, doc["name"])
	assert.Equal(t, true, doc["is_active"])
	assert.Equal(t, []any{}, doc["insurers"])
	assert.Equal(t, []any{}, doc["visit_types"])

	// Visit types: duplicate modes and bad values are refused; leaving a mode out removes it.
	r = ed.JSON(http.MethodPut, fmt.Sprintf("/telemed/doctors/%d/visit-types", docID), map[string]any{"visit_types": []any{
		map[string]any{"mode": "video", "duration_minutes": 20, "price_rials": 1},
		map[string]any{"mode": "video", "duration_minutes": 2, "price_rials": -1},
	}})
	require.Equal(t, http.StatusUnprocessableEntity, r.Status)
	for _, f := range []string{"visit_types.1.mode", "visit_types.1.duration_minutes", "visit_types.1.price_rials"} {
		assert.Contains(t, r.Errors(), f)
	}
	e.put(t, fmt.Sprintf("/telemed/doctors/%d/visit-types", docID), map[string]any{"visit_types": []any{
		map[string]any{"mode": "video", "duration_minutes": 20, "price_rials": 10},
		map[string]any{"mode": "phone", "duration_minutes": 15, "price_rials": 5, "is_active": false},
	}})
	r = e.put(t, fmt.Sprintf("/telemed/doctors/%d/visit-types", docID), map[string]any{"visit_types": []any{
		map[string]any{"mode": "phone", "duration_minutes": 15, "price_rials": 7},
	}})
	vts := r.Obj("doctor")["visit_types"].([]any)
	require.Len(t, vts, 1)
	assert.Equal(t, "phone", vts[0].(map[string]any)["mode"])
	assert.Equal(t, true, vts[0].(map[string]any)["is_active"])

	// Availability: end after start, step within the window, known modes.
	r = ed.JSON(http.MethodPut, fmt.Sprintf("/telemed/doctors/%d/availability", docID), map[string]any{"rules": []any{
		map[string]any{"weekday": 7, "start_time": "18:00", "end_time": "17:00", "slot_minutes": 30},
		map[string]any{"weekday": 1, "start_time": "10:00", "end_time": "10:20", "slot_minutes": 30, "modes": []any{"fax"}},
		map[string]any{"weekday": 1, "start_time": "25:00", "end_time": "24:00", "slot_minutes": 30},
	}})
	require.Equal(t, http.StatusUnprocessableEntity, r.Status)
	for _, f := range []string{"rules.0.weekday", "rules.0.end_time", "rules.1.slot_minutes", "rules.1.modes.0", "rules.2.start_time"} {
		assert.Contains(t, r.Errors(), f)
	}
	r = e.put(t, fmt.Sprintf("/telemed/doctors/%d/availability", docID), map[string]any{"rules": []any{
		map[string]any{"weekday": 6, "start_time": "21:00", "end_time": "24:00", "slot_minutes": 60, "modes": []any{"phone", "phone"}},
	}})
	rules := r.Obj("doctor")["availability"].([]any)
	require.Len(t, rules, 1)
	assert.Equal(t, map[string]any{"id": rules[0].(map[string]any)["id"], "weekday": float64(6), "start_time": "21:00",
		"end_time": "24:00", "slot_minutes": float64(60), "modes": []any{"phone"}}, rules[0])

	// Time off.
	r = ed.JSON(http.MethodPost, fmt.Sprintf("/telemed/doctors/%d/time-off", docID), map[string]any{
		"starts_at": "2030-02-01 10:00", "ends_at": "2030-02-01 09:00"})
	require.Equal(t, http.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Errors(), "ends_at")
	r = ed.JSON(http.MethodPost, fmt.Sprintf("/telemed/doctors/%d/time-off", docID), map[string]any{
		"starts_at": "2030-02-01 10:00", "ends_at": "2030-02-03 10:00"})
	require.Equal(t, http.StatusCreated, r.Status, r.Body)
	offs := r.Obj("doctor")["time_off"].([]any)
	require.Len(t, offs, 1)
	off := offs[0].(map[string]any)
	assert.Equal(t, "2030-02-01T10:00:00+03:30", off["starts_at"])
	r = ed.JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d/time-off/%d", docID+1000, id(off["id"])), nil)
	assert.Equal(t, http.StatusNotFound, r.Status)
	r = ed.JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d/time-off/%d", docID, id(off["id"])), nil)
	require.Equal(t, http.StatusOK, r.Status)
	assert.Empty(t, r.Obj("doctor")["time_off"])

	// Update keeps absent optional fields, null clears them.
	e.put(t, fmt.Sprintf("/telemed/doctors/%d", docID), map[string]any{"kind": "doctor", "name": map[string]any{"fa": "آزمون"},
		"specialty": "general", "licence_no": "T-1", "experience_years": 5, "headline": map[string]any{"fa": "سرتیتر"}})
	r = e.put(t, fmt.Sprintf("/telemed/doctors/%d", docID), map[string]any{"kind": "midwife", "name": map[string]any{"fa": "آزمون ۲"},
		"specialty": "midwifery", "licence_no": "T-2", "headline": nil})
	doc = r.Obj("doctor")
	assert.Equal(t, "midwife", doc["kind"])
	assert.InDelta(t, 5, doc["experience_years"], 0)
	assert.Nil(t, doc["headline"])

	// Photo.
	r = ed.Multipart(http.MethodPost, fmt.Sprintf("/telemed/doctors/%d/photo", docID), nil,
		admintest.File{Field: "photo", Name: "p.png", ContentType: "image/png", Data: testPNG(t, 50, 50)})
	require.Equal(t, http.StatusUnprocessableEntity, r.Status, "too small")
	r = ed.Multipart(http.MethodPost, fmt.Sprintf("/telemed/doctors/%d/photo", docID), nil,
		admintest.File{Field: "photo", Name: "p.png", ContentType: "image/png", Data: testPNG(t, 300, 300)})
	require.Equal(t, http.StatusOK, r.Status, r.Body)
	path, _ := r.Obj("doctor")["photo_path"].(string)
	assert.True(t, strings.HasPrefix(path, "doctors/"), path)
	assert.Equal(t, appURL+"/storage/"+path, r.Obj("doctor")["photo_url"])
	r = ed.JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d/photo", docID), nil)
	require.Equal(t, http.StatusOK, r.Status)
	assert.Nil(t, r.Obj("doctor")["photo_path"])

	// List + search; delete is super only.
	r = ed.Get("/telemed/doctors?q=T-2&status=active")
	require.Equal(t, http.StatusOK, r.Status)
	require.Len(t, r.Items(), 1)
	assert.Equal(t, []any{"phone"}, r.Items()[0].(map[string]any)["modes"])
	r = ed.JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d", docID), nil)
	assert.Equal(t, http.StatusForbidden, r.Status)
	r = e.As(admintest.SuperID).JSON(http.MethodDelete, fmt.Sprintf("/telemed/doctors/%d", docID), nil)
	require.Equal(t, http.StatusOK, r.Status)
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM telemed_visit_types WHERE doctor_id = ?`, docID), "cascade")
	assert.Equal(t, http.StatusNotFound, ed.Get(fmt.Sprintf("/telemed/doctors/%d", docID)).Status)

	r = ed.Get("/telemed/doctors/options")
	require.Equal(t, http.StatusOK, r.Status)
	assert.Equal(t, []any{"doctor", "midwife"}, r.Data()["kinds"])
}
