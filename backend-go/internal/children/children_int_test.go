package children_test

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
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-10-03T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
	svc *children.Service
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
	svc := children.NewService(db, catalog.NewReader(catalogstore.New(db), nil, 0, quiet))
	h := children.NewHandlers(svc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	p := "/api/v1/children"
	app.Get(p, locale, guard, h.Index)
	app.Post(p, locale, guard, h.Store)
	app.Get(p+"/reminders", locale, guard, h.Reminders)
	app.Get(p+"/:id", locale, guard, h.Show)
	app.Put(p+"/:id", locale, guard, h.Update)
	app.Delete(p+"/:id", locale, guard, h.Destroy)
	app.Get(p+"/:id/measurements", locale, guard, h.Measurements)
	app.Post(p+"/:id/measurements", locale, guard, h.StoreMeasurement)
	app.Put(p+"/:id/measurements/:mid", locale, guard, h.UpdateMeasurement)
	app.Delete(p+"/:id/measurements/:mid", locale, guard, h.DestroyMeasurement)
	app.Get(p+"/:id/growth", locale, guard, h.Growth)
	app.Get(p+"/:id/vaccines", locale, guard, h.Vaccines)
	app.Post(p+"/:id/vaccines/visits/:visit", locale, guard, h.MarkVisit)
	app.Put(p+"/:id/vaccines/:code", locale, guard, h.MarkDose)
	app.Delete(p+"/:id/vaccines/:code", locale, guard, h.UnmarkDose)
	app.Get(p+"/:id/milestones", locale, guard, h.Milestones)
	app.Put(p+"/:id/milestones/:code", locale, guard, h.Check)
	app.Get(p+"/:id/learn", locale, guard, h.Learn)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
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

func obj(v any, path ...string) map[string]any {
	m, _ := v.(map[string]any)
	for _, k := range path {
		m, _ = m[k].(map[string]any)
	}
	return m
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
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

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// ava is the artboard's child: a girl born 2026-06-20 (3 months and 13 days on the pinned day).
func (e *env) ava(t *testing.T, token string) uint64 {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/children", token, "fa",
		`{"name":"آوا","birth_date":"2026-06-20","sex":"girl","birth_weight_kg":3.2,"birth_length_cm":50,"delivery_type":"vaginal"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	return uint64(r.data()["id"].(float64))
}

func TestChildren_RequireAuth(t *testing.T) {
	e := setup(t)
	for _, path := range []string{"/api/v1/children", "/api/v1/children/1", "/api/v1/children/1/growth", "/api/v1/children/reminders"} {
		r := e.do(t, http.MethodGet, path, "", "fa", "")
		assert.Equal(t, http.StatusUnauthorized, r.status, path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], path)
	}
}

func TestChildren_CreateListShowUpdateDelete(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, "/api/v1/children", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, list(r.data()["children"]))
	assert.Equal(t, true, r.data()["can_add"])

	id := e.ava(t, tok)
	r = e.do(t, http.MethodGet, "/api/v1/children", tok, "fa", "")
	kids := list(r.data()["children"])
	require.Len(t, kids, 1)
	k := obj(kids[0])
	assert.Equal(t, "آوا", k["name"])
	assert.Equal(t, "آ", k["initial"])
	assert.Equal(t, "owner", k["role"])
	assert.Equal(t, true, k["can_edit"])
	assert.Equal(t, "۳ ماه و ۱۳ روز", obj(k, "age")["label"])
	assert.InDelta(t, 3.2, obj(k, "birth")["weight_kg"], 1e-9)
	v := obj(k, "vaccines")
	assert.InDelta(t, 21, v["total"], 0)
	assert.InDelta(t, 0, v["given"], 0)
	assert.Equal(t, false, v["up_to_date"])
	assert.Equal(t, "birth", obj(v, "next")["code"])
	assert.Equal(t, "normal", obj(k, "growth")["status"], "birth values 3.2 kg / 50 cm are in band")

	// Child home.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d", id), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "birth", obj(d, "latest")["source"])
	assert.InDelta(t, 3, obj(d, "milestones")["band_months"], 0)
	assert.InDelta(t, 7, obj(d, "milestones")["total"], 0)
	assert.InDelta(t, 15, obj(d, "this_week")["weeks"], 0)
	assert.Contains(t, obj(d, "this_week")["body"], "دمر")
	assert.Equal(t, "night_sleep_3m", obj(d, "learn", "featured")["code"])
	assert.Nil(t, d["today"])

	// Partial update keeps the rest.
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/children/%d", id), tok, "en", `{"name":"Ava","sex":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Ava", r.data()["name"])
	assert.Nil(t, r.data()["sex"])
	assert.Equal(t, "2026-06-20", r.data()["birth_date"])
	assert.Equal(t, "unknown", obj(r.data(), "growth")["status"], "no sex, no WHO placement")

	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/children/%d", id), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Child removed", r.body["message"])
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM children"))
}

func TestChildren_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodPost, "/api/v1/children", tok, "fa", `{"birth_date":"2026-10-04","sex":"x","birth_weight_kg":12}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	errs := obj(r.body["errors"])
	for _, f := range []string{"name", "birth_date", "sex", "birth_weight_kg"} {
		assert.Contains(t, errs, f)
	}
	r = e.do(t, http.MethodPost, "/api/v1/children", tok, "fa", `{"name":"  ","birth_date":"2000-01-01"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, obj(r.body["errors"]), "name")
	r = e.do(t, http.MethodPost, "/api/v1/children", tok, "fa", `{"name":"Old","birth_date":"2000-01-01"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, obj(r.body["errors"]), "birth_date")

	for i := range children.MaxChildren {
		r = e.do(t, http.MethodPost, "/api/v1/children", tok, "en", fmt.Sprintf(`{"name":"K%d","birth_date":"2026-01-01"}`, i))
		require.Equal(t, http.StatusCreated, r.status, r.raw)
	}
	r = e.do(t, http.MethodPost, "/api/v1/children", tok, "en", `{"name":"Extra","birth_date":"2026-01-01"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, "children_limit", r.body["error_code"])
}

func TestChildren_MeasurementsAndGrowth(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.ava(t, tok)
	base := fmt.Sprintf("/api/v1/children/%d", id)

	r := e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"measured_on":"2026-10-02","weight_kg":6.1,"length_cm":61,"head_cm":40.5}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	w := obj(r.data(), "weight")
	assert.InDelta(t, 6.1, w["value"], 1e-9)
	assert.InDelta(t, 50, w["percentile"], 2, "WHO girls weight-for-age day 104")
	assert.Equal(t, true, w["in_band"])
	assert.NotNil(t, obj(r.data(), "length")["percentile"])
	assert.NotNil(t, obj(r.data(), "head")["percentile"])
	mid := uint64(r.data()["id"].(float64))

	// Same day again merges.
	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"measured_on":"2026-10-02","weight_kg":6.2}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.InDelta(t, float64(mid), r.data()["id"], 0)
	assert.InDelta(t, 61, obj(r.data(), "length")["value"], 1e-9, "length kept")
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM child_measurements"))

	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"measured_on":"2026-06-01","weight_kg":3}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "before birth")
	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"measured_on":"2026-10-04","weight_kg":3}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "future")
	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"weight_kg":200}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	r = e.do(t, http.MethodPost, base+"/measurements", tok, "fa", `{"measured_on":"2026-08-20","weight_kg":5.0,"length_cm":57}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	other := uint64(r.data()["id"].(float64))
	r = e.do(t, http.MethodPut, fmt.Sprintf("%s/measurements/%d", base, other), tok, "fa", `{"measured_on":"2026-10-02"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "day taken")
	r = e.do(t, http.MethodPut, fmt.Sprintf("%s/measurements/%d", base, other), tok, "fa", `{"head_cm":38}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 38, obj(r.data(), "head")["value"], 1e-9)
	assert.InDelta(t, 5.0, obj(r.data(), "weight")["value"], 1e-9)

	r = e.do(t, http.MethodGet, base+"/measurements", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	ms := list(r.data()["measurements"])
	require.Len(t, ms, 3, "two measurements and the birth point")
	assert.Equal(t, "2026-10-02", obj(ms[0])["measured_on"])
	assert.Equal(t, "birth", obj(ms[2])["source"])
	assert.Equal(t, "normal", obj(r.data(), "verdict")["status"])

	r = e.do(t, http.MethodGet, base+"/growth?indicator=weight", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	g := r.data()
	assert.Equal(t, true, g["available"])
	ref := list(g["reference"])
	require.Len(t, ref, 13, "0–12 months")
	assert.InDelta(t, 3.232, obj(ref[0])["p50"], 0.001, "WHO girls median at birth")
	assert.InDelta(t, 2.44, obj(ref[0])["p3"], 0.001)
	pts := list(g["points"])
	require.Len(t, pts, 3)
	assert.Equal(t, "2026-06-20", obj(pts[0])["measured_on"], "oldest first")
	assert.Equal(t, "میانه رشد دختران", g["median_label"])

	r = e.do(t, http.MethodGet, base+"/growth?indicator=bmi", tok, "fa", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	r = e.do(t, http.MethodDelete, fmt.Sprintf("%s/measurements/%d", base, other), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodDelete, fmt.Sprintf("%s/measurements/%d", base, other), tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "measurement_not_found", r.body["error_code"])

	// Sex not told: values kept, no placement.
	e.do(t, http.MethodPut, base, tok, "fa", `{"sex":null}`)
	r = e.do(t, http.MethodGet, base+"/growth", tok, "fa", "")
	assert.Equal(t, false, r.data()["available"])
	assert.Equal(t, "sex_unknown", r.data()["reason"])
	assert.Empty(t, list(r.data()["reference"]))
	assert.Nil(t, obj(list(r.data()["points"])[0])["percentile"])
}

func TestChildren_Vaccines(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.ava(t, tok)
	base := fmt.Sprintf("/api/v1/children/%d/vaccines", id)

	r := e.do(t, http.MethodGet, base, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	visits := list(r.data()["visits"])
	require.Len(t, visits, 7)
	assert.Equal(t, "بدو تولد", obj(visits[0])["label"])
	assert.Equal(t, "overdue", obj(visits[0])["status"])
	m4 := obj(visits[2])
	assert.Equal(t, "m4", m4["code"])
	assert.Equal(t, "2026-10-20", m4["due_date"])
	assert.Equal(t, "2026-10-17", m4["remind_on"])
	assert.Equal(t, "soon", m4["status"])
	assert.InDelta(t, 17, m4["days_left"], 0)
	assert.Len(t, list(m4["doses"]), 4)
	assert.Equal(t, "y6", obj(visits[6])["code"])
	assert.Equal(t, "۶ سالگی", obj(visits[6])["label"])

	r = e.do(t, http.MethodPost, base+"/visits/birth", tok, "fa", `{"given_on":"2026-06-20"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPost, base+"/visits/m2", tok, "fa", `{"given_on":"2026-08-21","note":"تب خفیف"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	s := obj(r.data(), "summary")
	assert.InDelta(t, 7, s["given"], 0, "artboard «۷/۱۹»: 7 doses given")
	assert.InDelta(t, 2, s["completed_visits"], 0)
	assert.Equal(t, true, s["up_to_date"])
	assert.Equal(t, "m4", obj(s, "next")["code"])
	dose := obj(list(obj(list(r.data()["visits"])[1])["doses"])[0])
	assert.Equal(t, "2026-08-21", dose["given_on"])
	assert.Equal(t, "تب خفیف", dose["note"])
	assert.Equal(t, "done", dose["status"])

	r = e.do(t, http.MethodPut, base+"/penta_2", tok, "fa", `{}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 8, obj(r.data(), "summary")["given"], 0)
	r = e.do(t, http.MethodDelete, base+"/penta_2", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 7, obj(r.data(), "summary")["given"], 0)

	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, base+"/nope", tok, "fa", `{}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPost, base+"/visits/m99", tok, "fa", `{}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, base+"/bcg", tok, "fa", `{"given_on":"2026-06-01"}`).status, "before birth")
}

func TestChildren_Reminders(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	// Born 2026-06-06: the 4-month visit is due 2026-10-06, its reminder opened on 10-03 (today).
	r := e.do(t, http.MethodPost, "/api/v1/children", tok, "fa", `{"name":"سام","birth_date":"2026-06-06","sex":"boy"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := uint64(r.data()["id"].(float64))
	e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/children/%d/vaccines/visits/birth", id), tok, "fa", `{"given_on":"2026-06-06"}`)
	e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/children/%d/vaccines/visits/m2", id), tok, "fa", `{"given_on":"2026-08-06"}`)

	r = e.do(t, http.MethodGet, "/api/v1/children/reminders", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rem := list(r.data()["reminders"])
	require.Len(t, rem, 1)
	assert.Equal(t, "m4", obj(rem[0], "visit")["code"])
	assert.InDelta(t, 3, obj(rem[0])["days_left"], 0)
	assert.Equal(t, "سام", obj(rem[0], "child")["name"])
	assert.InDelta(t, 3, r.data()["days_before"], 0)
}

func TestChildren_Milestones(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.ava(t, tok)
	base := fmt.Sprintf("/api/v1/children/%d/milestones", id)

	r := e.do(t, http.MethodGet, base, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	b := obj(r.data(), "band")
	assert.InDelta(t, 3, b["months"], 0)
	assert.Equal(t, "۳ ماهگی", b["label"])
	assert.Len(t, list(b["items"]), 7)
	assert.Len(t, list(b["activities"]), 2)
	assert.NotEmpty(t, b["doctor_note"])
	bands := list(r.data()["bands"])
	assert.Len(t, bands, 11)
	assert.Equal(t, true, obj(bands[1])["current"])

	r = e.do(t, http.MethodPut, base+"/m3_social_smile", tok, "fa", `{"checked":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 1, obj(r.data(), "band")["checked"], 0)
	item := obj(list(obj(r.data(), "band")["items"])[0])
	assert.Equal(t, true, item["checked"])
	assert.Equal(t, "2026-10-03", item["checked_on"])
	r = e.do(t, http.MethodPut, base+"/m6_rolls_over", tok, "fa", `{"checked":true,"checked_on":"2026-09-30"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 6, obj(r.data(), "band")["months"], 0, "answered with the item's band")
	r = e.do(t, http.MethodPut, base+"/m3_social_smile", tok, "fa", `{"checked":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 0, obj(r.data(), "band")["checked"], 0)

	r = e.do(t, http.MethodGet, base+"?month=12", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 12, obj(r.data(), "band")["months"], 0)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodGet, base+"?month=5", tok, "fa", "").status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, base+"/nope", tok, "fa", `{"checked":true}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, base+"/m3_laughs", tok, "fa", `{}`).status)
}

func TestChildren_Learn(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.ava(t, tok)
	e.db.Exec(`INSERT INTO articles (slug, title, is_published, created_at, updated_at) VALUES ('sleep-3m', '{"fa":"خواب"}', 1, NOW(), NOW())`) //nolint:errcheck // fixture
	_, err := e.db.Exec("UPDATE catalog_items SET meta = JSON_SET(meta, '$.article_slug', 'sleep-3m') WHERE `group` = 'child_learn' AND code = 'night_sleep_3m'")
	require.NoError(t, err)

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d/learn", id), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	f := obj(r.data(), "featured")
	assert.Equal(t, "night_sleep_3m", f["code"])
	assert.Equal(t, "خواب شبانه از ۳ ماهگی", f["title"])
	assert.Equal(t, "sleep-3m", obj(f, "article")["slug"])
	assert.Len(t, list(r.data()["topics"]), 6)
	for _, tip := range list(r.data()["tips"]) {
		m := obj(tip)
		assert.LessOrEqual(t, m["from_months"].(float64), 3.0)
		assert.GreaterOrEqual(t, m["to_months"].(float64), 3.0)
	}

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d/learn?topic=feeding", id), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "feeding", r.data()["topic"])
	for _, tip := range list(r.data()["tips"]) {
		assert.Equal(t, "feeding", obj(tip)["topic"])
	}
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d/learn?topic=x", id), tok, "en", "").status)
}

// IDOR: another account gets the same 404 on every child route, reads and writes alike.
func TestChildren_OtherUsersChildIsNotFound(t *testing.T) {
	e := setup(t)
	_, sara := e.user(t, "09120000001")
	_, mina := e.user(t, "09120000002")
	id := e.ava(t, sara)
	r := e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/children/%d/measurements", id), sara, "fa", `{"weight_kg":6}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	mid := uint64(r.data()["id"].(float64))

	base := fmt.Sprintf("/api/v1/children/%d", id)
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, base, ""},
		{http.MethodPut, base, `{"name":"X"}`},
		{http.MethodDelete, base, ""},
		{http.MethodGet, base + "/measurements", ""},
		{http.MethodPost, base + "/measurements", `{"weight_kg":6}`},
		{http.MethodPut, fmt.Sprintf("%s/measurements/%d", base, mid), `{"weight_kg":7}`},
		{http.MethodDelete, fmt.Sprintf("%s/measurements/%d", base, mid), ""},
		{http.MethodGet, base + "/growth", ""},
		{http.MethodGet, base + "/vaccines", ""},
		{http.MethodPost, base + "/vaccines/visits/birth", `{}`},
		{http.MethodPut, base + "/vaccines/bcg", `{}`},
		{http.MethodDelete, base + "/vaccines/bcg", ""},
		{http.MethodGet, base + "/milestones", ""},
		{http.MethodPut, base + "/milestones/m3_social_smile", `{"checked":true}`},
		{http.MethodGet, base + "/learn", ""},
		{http.MethodGet, "/api/v1/children/999999", ""},
		{http.MethodGet, "/api/v1/children/abc", ""},
	} {
		r := e.do(t, c.method, c.path, mina, "en", c.body)
		assert.Equal(t, http.StatusNotFound, r.status, "%s %s", c.method, c.path)
		assert.Equal(t, "child_not_found", r.body["error_code"], "%s %s", c.method, c.path)
	}
	assert.Empty(t, list(e.do(t, http.MethodGet, "/api/v1/children", mina, "en", "").data()["children"]))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM children"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM child_measurements"))

	// A measurement id of another child is not reachable through one's own child.
	own := e.ava(t, mina)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/children/%d/measurements/%d", own, mid), mina, "en", `{"weight_kg":7}`)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "measurement_not_found", r.body["error_code"])
}

// Family sharing: a child the owner shares with her active spouse link is visible to that spouse, read-only and
// audited; a partner link, an unshared child or a revoked link sees nothing.
func TestChildren_SharedWithSpouseReadOnly(t *testing.T) {
	e := setup(t)
	sara, saraTok := e.user(t, "09120000001")
	ali, aliTok := e.user(t, "09120000002")
	reza, rezaTok := e.user(t, "09120000003")
	ava := e.ava(t, saraTok)
	r := e.do(t, http.MethodPost, "/api/v1/children", saraTok, "fa", `{"name":"سام","birth_date":"2022-08-01","sex":"boy"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	sam := uint64(r.data()["id"].(float64))

	link := func(companionUser uint64, typ string) (uint64, uint64) {
		res, err := e.db.Exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at) VALUES (?, ?, ?, 'active', NOW(), NOW())`,
			sara, companionUser, typ)
		require.NoError(t, err)
		cid, _ := res.LastInsertId()
		res, err = e.db.Exec(`INSERT INTO families (owner_id, spouse_user_id, companion_id) VALUES (?, ?, ?)`, sara, companionUser, cid)
		require.NoError(t, err)
		fid, _ := res.LastInsertId()
		return uint64(cid), uint64(fid)
	}
	cid, fid := link(ali, "spouse")
	_, err := e.db.Exec(`INSERT INTO family_children (family_id, child_id) VALUES (?, ?)`, fid, ava)
	require.NoError(t, err)

	ok, err := e.svc.OwnsChildren(context.Background(), sara, []uint64{ava, sam})
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = e.svc.OwnsChildren(context.Background(), ali, []uint64{ava})
	require.NoError(t, err)
	assert.False(t, ok, "the spouse does not own a shared child")

	r = e.do(t, http.MethodGet, "/api/v1/children", aliTok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	kids := list(r.data()["children"])
	require.Len(t, kids, 1, "only the shared child")
	k := obj(kids[0])
	assert.Equal(t, "shared", k["role"])
	assert.Equal(t, false, k["can_edit"])
	assert.Equal(t, "Sara", k["owner_name"])
	assert.InDelta(t, 0, r.data()["owned_count"], 0)

	for _, path := range []string{"", "/measurements", "/growth", "/vaccines", "/milestones", "/learn"} {
		r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d%s", ava, path), aliTok, "fa", "")
		assert.Equal(t, http.StatusOK, r.status, path)
	}
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d", sam), aliTok, "fa", "").status, "not shared")
	for _, c := range []struct{ method, path, body string }{
		{http.MethodPut, "", `{"name":"X"}`},
		{http.MethodDelete, "", ""},
		{http.MethodPost, "/measurements", `{"weight_kg":6}`},
		{http.MethodPut, "/vaccines/bcg", `{}`},
		{http.MethodPut, "/milestones/m3_social_smile", `{"checked":true}`},
	} {
		r = e.do(t, c.method, fmt.Sprintf("/api/v1/children/%d%s", ava, c.path), aliTok, "en", c.body)
		assert.Equal(t, http.StatusForbidden, r.status, "%s %s", c.method, c.path)
		assert.Equal(t, "child_read_only", r.body["error_code"])
	}
	assert.Positive(t, e.count(t, `SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND companion_id = ?
		AND section = 'children' AND action = 'read'`, sara, ali, cid), "spouse reads are audited")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM companion_audit_logs WHERE actor_id = ?`, sara), "the owner's own reads are not")

	card, err := e.svc.CompanionCard(context.Background(), ali, sara, cid, "fa", time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	b, _ := json.Marshal(card)
	assert.Contains(t, string(b), `"name":"آوا"`)
	assert.Contains(t, string(b), `"next_vaccine":{"code":"birth"`)
	none, err := e.svc.CompanionCard(context.Background(), reza, sara, cid, "fa", time.Now())
	require.NoError(t, err)
	assert.Nil(t, none)

	// A partner-type family row (never created by the companion service) still needs spouse_user_id = the viewer and
	// an active link; a revoked link hides everything.
	_, pfid := link(reza, "partner")
	_, err = e.db.Exec(`INSERT INTO family_children (family_id, child_id) VALUES (?, ?)`, pfid, sam)
	require.NoError(t, err)
	_, err = e.db.Exec(`UPDATE companions SET status = 'revoked' WHERE companion_user_id = ?`, reza)
	require.NoError(t, err)
	assert.Empty(t, list(e.do(t, http.MethodGet, "/api/v1/children", rezaTok, "fa", "").data()["children"]))
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d", sam), rezaTok, "fa", "").status)

	_, err = e.db.Exec(`UPDATE companions SET status = 'revoked' WHERE id = ?`, cid)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/children/%d", ava), aliTok, "fa", "").status)
	assert.Empty(t, list(e.do(t, http.MethodGet, "/api/v1/children", aliTok, "fa", "").data()["children"]))
}
