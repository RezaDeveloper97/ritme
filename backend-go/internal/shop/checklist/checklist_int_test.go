package checklist_test

// CB-SHOP-04: the checklist API (template + own state + custom rows, IDOR, family share read only) and the supply
// reminder (never created when off, one care reminder when on, removed when switched off).

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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/shop/checklist"
	"github.com/ritme/backend-go/internal/shop/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee4"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fixed is the request clock: 2026-10-06 10:00 Tehran.
var fixed = time.Date(2026, 10, 6, 10, 0, 0, 0, civildate.Tehran)

type fakeDue map[uint64]civildate.Date

func (f fakeDue) DueDate(_ context.Context, userID uint64, _ civildate.Date) (civildate.Date, bool, error) {
	d, ok := f[userID]
	return d, ok, nil
}

type fakePredictor struct {
	next  map[uint64]civildate.Date
	calls int
}

func (f *fakePredictor) NextPeriodStart(_ context.Context, userID uint64, _ civildate.Date) (civildate.Date, bool, error) {
	f.calls++
	d, ok := f.next[userID]
	return d, ok, nil
}

// fakeShare: viewer → owner pairs with a pregnancy grant.
type fakeShare map[[2]uint64]bool

func (f fakeShare) CanView(_ context.Context, ownerID, viewerID uint64) (bool, error) {
	return f[[2]uint64{ownerID, viewerID}], nil
}

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	due     fakeDue
	predict *fakePredictor
	share   fakeShare
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
	e := &env{db: db, due: fakeDue{}, predict: &fakePredictor{next: map[uint64]civildate.Date{}}, share: fakeShare{}}
	svc := checklist.NewService(store.New(db), e.due, e.predict)
	h := checklist.NewHandlers(svc, nil, e.share, clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/shop"
	app.Get(p+"/checklist", locale, guard, h.Show)
	app.Post(p+"/checklist/custom", locale, guard, h.StoreCustom)
	app.Patch(p+"/checklist/custom/:id", locale, guard, h.UpdateCustom)
	app.Delete(p+"/checklist/custom/:id", locale, guard, h.DestroyCustom)
	app.Put(p+"/checklist/items/:id", locale, guard, h.SetState)
	app.Get(p+"/supply-reminder", locale, guard, h.ShowSupply)
	app.Put(p+"/supply-reminder", locale, guard, h.UpdateSupply)
	e.app, e.iss = app, passport.NewIssuer(key, q, clock.Real{}, 365)
	return e
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // positive id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // positive id
}

func (e *env) template(t *testing.T, group, fa, en string, sort int, essential bool) uint64 {
	t.Helper()
	title, _ := json.Marshal(map[string]string{"fa": fa, "en": en})
	res, err := e.db.Exec(`INSERT INTO shop_layette_items (group_key, title, suggested_qty, is_essential, is_first_days, category_id, sort_order, is_active, created_at, updated_at)
		VALUES (?, ?, 3, ?, 0, (SELECT id FROM shop_categories WHERE slug = 'layette'), ?, 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		group, string(title), essential, sort)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // positive id
}

type response struct {
	status int
	body   map[string]any
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token, lang string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
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
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, body: m}
}

func (e *env) careReminders(t *testing.T, userID uint64) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT IFNULL(DATE_FORMAT(scheduled_at, '%Y-%m-%d %H:%i'), '') FROM reminders WHERE user_id = ? ORDER BY id`, userID)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

func id(n uint64) string { return strconv.FormatUint(n, 10) }

func TestChecklist_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/shop/checklist"},
		{http.MethodPost, "/api/v1/shop/checklist/custom"},
		{http.MethodPut, "/api/v1/shop/supply-reminder"},
	} {
		r := e.do(t, tc.method, tc.path, "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestChecklist_TemplateStatesCustomAndIDOR(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000401")
	_, tokB := e.user(t, "09120000402")
	body := e.template(t, checklist.GroupClothing, "بادی", "Bodysuit", 1, true)
	socks := e.template(t, checklist.GroupClothing, "جوراب", "Socks", 2, false)
	crib := e.template(t, checklist.GroupSleep, "تخت", "Crib", 1, true)
	e.due[a] = civildate.MustParse("2026-12-11")

	r := e.do(t, http.MethodGet, "/api/v1/shop/checklist", tokA, "en", nil)
	require.Equal(t, 200, r.status, r.body)
	d := r.data()
	assert.Equal(t, "2026-12-11", d["due_date"])
	assert.EqualValues(t, 9, d["weeks_left"])
	assert.Equal(t, false, d["read_only"])
	assert.Equal(t, map[string]any{"ready": 0.0, "total": 3.0, "not_needed": 0.0, "percent": 0.0}, d["summary"])
	groups := d["groups"].([]any)
	require.Len(t, groups, 2)
	g0 := groups[0].(map[string]any)
	assert.Equal(t, "clothing", g0["key"])
	assert.Equal(t, "Clothing", g0["title"])
	it := g0["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "Bodysuit", it["title"])
	assert.Equal(t, "todo", it["state"])
	assert.Equal(t, "layette", it["category_slug"])
	assert.Nil(t, it["id"])

	// States.
	r = e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/"+id(body), tokA, "en", map[string]any{"state": "have", "quantity": 6})
	require.Equal(t, 200, r.status, r.body)
	r = e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/"+id(socks), tokA, "en", map[string]any{"state": "not_needed"})
	require.Equal(t, 200, r.status, r.body)
	r = e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/"+id(crib), tokA, "en", map[string]any{"state": "bogus"})
	assert.Equal(t, 422, r.status)
	assert.Contains(t, r.body["errors"], "state")
	r = e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/999999", tokA, "en", map[string]any{"state": "have"})
	assert.Equal(t, 404, r.status)

	// Custom rows.
	r = e.do(t, http.MethodPost, "/api/v1/shop/checklist/custom", tokA, "en", map[string]any{"title": "Swaddle", "group": "sleep", "state": "in_cart"})
	require.Equal(t, 201, r.status, r.body)
	custom := uint64(r.data()["item"].(map[string]any)["id"].(float64))
	assert.Equal(t, true, r.data()["item"].(map[string]any)["is_custom"])
	r = e.do(t, http.MethodPost, "/api/v1/shop/checklist/custom", tokA, "en", map[string]any{"title": "   ", "group": "nope"})
	assert.Equal(t, 422, r.status)

	r = e.do(t, http.MethodGet, "/api/v1/shop/checklist", tokA, "en", nil)
	d = r.data()
	assert.Equal(t, map[string]any{"ready": 1.0, "total": 3.0, "not_needed": 1.0, "percent": 33.0}, d["summary"])

	// User B cannot touch A's custom row, and sees only the template.
	assert.Equal(t, 404, e.do(t, http.MethodPatch, "/api/v1/shop/checklist/custom/"+id(custom), tokB, "en", map[string]any{"state": "have"}).status)
	assert.Equal(t, 404, e.do(t, http.MethodDelete, "/api/v1/shop/checklist/custom/"+id(custom), tokB, "en", nil).status)
	r = e.do(t, http.MethodGet, "/api/v1/shop/checklist", tokB, "en", nil)
	assert.Equal(t, map[string]any{"ready": 0.0, "total": 3.0, "not_needed": 0.0, "percent": 0.0}, r.data()["summary"])
	assert.Nil(t, r.data()["due_date"])
	assert.Nil(t, r.data()["weeks_left"])

	// Owner edits and deletes her row.
	r = e.do(t, http.MethodPatch, "/api/v1/shop/checklist/custom/"+id(custom), tokA, "en", map[string]any{"state": "have"})
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, "Swaddle", r.data()["item"].(map[string]any)["title"])
	assert.Equal(t, 200, e.do(t, http.MethodDelete, "/api/v1/shop/checklist/custom/"+id(custom), tokA, "en", nil).status)
	assert.Equal(t, 404, e.do(t, http.MethodDelete, "/api/v1/shop/checklist/custom/"+id(custom), tokA, "en", nil).status)
}

func TestChecklist_FamilyShareReadOnly(t *testing.T) {
	e := setup(t)
	owner, tokOwner := e.user(t, "09120000411")
	viewer, tokViewer := e.user(t, "09120000412")
	_, tokStranger := e.user(t, "09120000413")
	item := e.template(t, checklist.GroupFeeding, "شیشه شیر", "Bottle", 1, true)
	require.Equal(t, 200, e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/"+id(item), tokOwner, "en", map[string]any{"state": "in_cart"}).status)
	e.share[[2]uint64{owner, viewer}] = true

	r := e.do(t, http.MethodGet, "/api/v1/shop/checklist?for_user_id="+id(owner), tokViewer, "en", nil)
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, true, r.data()["read_only"])
	g := r.data()["groups"].([]any)[0].(map[string]any)
	assert.Equal(t, "in_cart", g["items"].([]any)[0].(map[string]any)["state"])

	r = e.do(t, http.MethodGet, "/api/v1/shop/checklist?for_user_id="+id(owner), tokStranger, "en", nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, checklist.ErrorCodeCompanionForbidden, r.body["error_code"])
	assert.Equal(t, 422, e.do(t, http.MethodGet, "/api/v1/shop/checklist?for_user_id=abc", tokViewer, "en", nil).status)

	// Writes never act on the owner's list.
	r = e.do(t, http.MethodPut, "/api/v1/shop/checklist/items/"+id(item), tokViewer, "en", map[string]any{"state": "have", "for_user_id": owner})
	assert.Equal(t, 403, r.status)
	r = e.do(t, http.MethodPost, "/api/v1/shop/checklist/custom", tokViewer, "en", map[string]any{"title": "x", "group": "bath", "for_user_id": owner})
	assert.Equal(t, 403, r.status)
}

func TestSupplyReminder_OffByDefaultNeverCreated(t *testing.T) {
	e := setup(t)
	u, tok := e.user(t, "09120000421")
	e.predict.next[u] = civildate.MustParse("2026-10-20")

	r := e.do(t, http.MethodGet, "/api/v1/shop/supply-reminder", tok, "en", nil)
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, map[string]any{"enabled": false, "lead_days": 3.0, "product_id": nil, "remind_on": nil}, r.data())

	// Saving with the switch off (even with other fields) never creates a reminder nor reads the prediction.
	r = e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"enabled": false, "lead_days": 5})
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, false, r.data()["enabled"])
	assert.Nil(t, r.data()["remind_on"])
	assert.Empty(t, e.careReminders(t, u))
	assert.Equal(t, 0, e.predict.calls)

	r = e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"lead_days": 9})
	assert.Equal(t, 422, r.status)
	assert.Contains(t, r.body["errors"], "enabled")
	assert.Contains(t, r.body["errors"], "lead_days")
	r = e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"enabled": true, "product_id": 999999})
	assert.Equal(t, 422, r.status)
	assert.Empty(t, e.careReminders(t, u))
}

func TestSupplyReminder_OnCreatesOneDiscreetCareReminderOffRemovesIt(t *testing.T) {
	e := setup(t)
	u, tok := e.user(t, "09120000431")
	other, _ := e.user(t, "09120000432")
	e.predict.next[u] = civildate.MustParse("2026-10-20")

	r := e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "fa", map[string]any{"enabled": true})
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, "2026-10-17", r.data()["remind_on"])
	assert.Equal(t, []string{"2026-10-17 09:00"}, e.careReminders(t, u))
	var title, typ, meta string
	require.NoError(t, e.db.QueryRow(`SELECT title, type, meta FROM reminders WHERE user_id = ?`, u).Scan(&title, &typ, &meta))
	assert.Equal(t, "custom", typ)
	assert.JSONEq(t, `{"source":"shop_supply"}`, meta)
	for _, w := range []string{"پریود", "قاعدگی", "نوار", "period", "pad"} {
		assert.NotContains(t, strings.ToLower(title), w, "the care reminder title is discreet")
	}

	// Saving again / reading keeps exactly one row; a new prediction moves it.
	require.Equal(t, 200, e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"enabled": true, "lead_days": 2}).status)
	assert.Equal(t, []string{"2026-10-18 09:00"}, e.careReminders(t, u))
	e.predict.next[u] = civildate.MustParse("2026-11-17")
	r = e.do(t, http.MethodGet, "/api/v1/shop/supply-reminder", tok, "en", nil)
	assert.Equal(t, "2026-11-15", r.data()["remind_on"])
	assert.Equal(t, []string{"2026-11-15 09:00"}, e.careReminders(t, u))

	// A prediction due within the lead days reminds today; a due / late period removes it.
	e.predict.next[u] = civildate.MustParse("2026-10-07")
	r = e.do(t, http.MethodGet, "/api/v1/shop/supply-reminder", tok, "en", nil)
	assert.Equal(t, "2026-10-06", r.data()["remind_on"])
	e.predict.next[u] = civildate.MustParse("2026-10-06")
	r = e.do(t, http.MethodGet, "/api/v1/shop/supply-reminder", tok, "en", nil)
	assert.Nil(t, r.data()["remind_on"])
	assert.Empty(t, e.careReminders(t, u))

	// Off removes it; other users' reminders are untouched.
	_, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'custom', 'x', 'none', 1, '{"source":"shop_supply"}', NOW(), NOW())`, other)
	require.NoError(t, err)
	e.predict.next[u] = civildate.MustParse("2026-11-17")
	require.Equal(t, 200, e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"enabled": true}).status)
	require.Len(t, e.careReminders(t, u), 1)
	r = e.do(t, http.MethodPut, "/api/v1/shop/supply-reminder", tok, "en", map[string]any{"enabled": false})
	require.Equal(t, 200, r.status, r.body)
	assert.Empty(t, e.careReminders(t, u))
	assert.Len(t, e.careReminders(t, other), 1)
}
