package babylog_test

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
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/babylog"
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

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

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
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	csvc := children.NewService(db, catalog.NewReader(catalogstore.New(db), nil, 0, quiet))
	ch := children.NewHandlers(csvc, clock.Real{})
	bsvc := babylog.NewService(db, csvc)
	ch.SetToday(bsvc)
	h := babylog.NewHandlers(bsvc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/children", locale, guard, ch.Store)
	app.Get("/api/v1/children/:id", locale, guard, ch.Show)
	p := "/api/v1/children/:id"
	app.Get(p+"/feeds", locale, guard, h.Feeds)
	app.Post(p+"/feeds", locale, guard, h.StoreFeed)
	app.Post(p+"/feeds/start", locale, guard, h.StartFeed)
	app.Put(p+"/feeds/:fid", locale, guard, h.UpdateFeed)
	app.Delete(p+"/feeds/:fid", locale, guard, h.DestroyFeed)
	app.Post(p+"/feeds/:fid/side", locale, guard, h.FeedSide)
	app.Post(p+"/feeds/:fid/stop", locale, guard, h.StopFeed)
	app.Get(p+"/sleeps", locale, guard, h.Sleeps)
	app.Post(p+"/sleeps", locale, guard, h.StoreSleep)
	app.Post(p+"/sleeps/start", locale, guard, h.StartSleep)
	app.Put(p+"/sleeps/:sid", locale, guard, h.UpdateSleep)
	app.Delete(p+"/sleeps/:sid", locale, guard, h.DestroySleep)
	app.Post(p+"/sleeps/:sid/stop", locale, guard, h.StopSleep)
	app.Get(p+"/diapers", locale, guard, h.Diapers)
	app.Post(p+"/diapers", locale, guard, h.StoreDiaper)
	app.Put(p+"/diapers/:did", locale, guard, h.UpdateDiaper)
	app.Delete(p+"/diapers/:did", locale, guard, h.DestroyDiaper)
	app.Get(p+"/baby-logs", locale, guard, h.Summary)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
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

// at is a request at 2026-10-03 hh:mm Tehran.
func at(hhmm string) string { return "2026-10-03T" + hhmm + ":00+03:30" }

func (e *env) do(t *testing.T, when, method, path, token, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept-Language", "en")
	req.Header.Set(clock.Header, when)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func (e *env) child(t *testing.T, token string) uint64 {
	t.Helper()
	r := e.do(t, at("09:00"), http.MethodPost, "/api/v1/children", token, `{"name":"Ava","birth_date":"2026-06-20","sex":"girl"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	return uint64(r.data()["id"].(float64))
}

func (e *env) feedsCount(t *testing.T, userID uint64) (string, string) {
	t.Helper()
	var v, src string
	err := e.db.QueryRow(`SELECT CAST(value_num AS UNSIGNED), source FROM health_log_entries
		WHERE user_id = ? AND log_date = '2026-10-03' AND category = 'baby' AND param = 'feeds_count'`, userID).Scan(&v, &src)
	if err == sql.ErrNoRows {
		return "", ""
	}
	require.NoError(t, err)
	return v, src
}

func TestBabyLogs_RequireAuth(t *testing.T) {
	e := setup(t)
	for _, path := range []string{"/feeds", "/sleeps", "/diapers", "/baby-logs"} {
		r := e.do(t, at("10:00"), http.MethodGet, "/api/v1/children/1"+path, "", "")
		assert.Equal(t, http.StatusUnauthorized, r.status, path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], path)
	}
}

// The artboard's feed: right 7 min, switch to left, 7:42 on the left, stop; totals, last feed, next side.
func TestBabyLogs_BreastFeedTimer(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001")
	_, err := e.db.Exec(`INSERT INTO postpartum_profiles (user_id, birth_date, created_at, updated_at) VALUES (?, '2026-06-20', NOW(), NOW())`, sara)
	require.NoError(t, err)
	id := e.child(t, tok)
	base := fmt.Sprintf("/api/v1/children/%d/feeds", id)

	r := e.do(t, at("14:00"), http.MethodPost, base+"/start", tok, `{"type":"breast","side":"right"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	fid := uint64(r.data()["id"].(float64))
	assert.Equal(t, "right", r.data()["active_side"])
	assert.Equal(t, true, r.data()["is_active"])
	v, src := e.feedsCount(t, sara)
	assert.Equal(t, "1", v, "the mother's baby.feeds_count follows the sessions")
	assert.Equal(t, "baby_log", src)

	// One running feed per child.
	r = e.do(t, at("14:01"), http.MethodPost, base+"/start", tok, `{"type":"bottle"}`)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "feed_active", r.body["error_code"])

	r = e.do(t, at("14:07"), http.MethodPost, fmt.Sprintf("%s/%d/side", base, fid), tok, `{"side":"left"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 420, r.data()["right_seconds"], 0)
	assert.Equal(t, "left", r.data()["active_side"])
	assert.Equal(t, at("14:07"), r.data()["side_started_at"])

	r = e.do(t, at("14:10"), http.MethodPost, fmt.Sprintf("%s/%d/side", base, fid), tok, `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "side must be present (null pauses)")

	r = e.do(t, at("14:14"), http.MethodGet, base, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	act := obj(r.data(), "active")
	assert.InDelta(t, 420+420, act["duration_seconds"], 0, "running segment included")

	r = e.do(t, at("14:14"), http.MethodPost, fmt.Sprintf("%s/%d/stop", base, fid), tok, `{"note":"good latch"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, false, d["is_active"])
	assert.InDelta(t, 420, d["left_seconds"], 0)
	assert.InDelta(t, 420, d["right_seconds"], 0)
	assert.InDelta(t, 840, d["duration_seconds"], 0)
	assert.Equal(t, "left", d["last_side"])
	assert.Equal(t, "good latch", d["note"])

	r = e.do(t, at("14:15"), http.MethodPost, fmt.Sprintf("%s/%d/stop", base, fid), tok, `{}`)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "session_ended", r.body["error_code"])
	r = e.do(t, at("14:15"), http.MethodPost, fmt.Sprintf("%s/%d/side", base, fid), tok, `{"side":"right"}`)
	assert.Equal(t, "session_ended", r.body["error_code"])

	// A bottle by hand; sides are breast-only.
	r = e.do(t, at("18:30"), http.MethodPost, base, tok, `{"type":"bottle","started_at":"2026-10-03 18:00","duration_minutes":10,"amount_ml":90}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	bottle := uint64(r.data()["id"].(float64))
	assert.Equal(t, at("18:10"), r.data()["ended_at"])
	r = e.do(t, at("19:00"), http.MethodPost, base+"/start", tok, `{"type":"pump"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	pump := uint64(r.data()["id"].(float64))
	r = e.do(t, at("19:01"), http.MethodPost, fmt.Sprintf("%s/%d/side", base, pump), tok, `{"side":"left"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, "not_breast_feed", r.body["error_code"])
	r = e.do(t, at("19:01"), http.MethodPut, fmt.Sprintf("%s/%d", base, pump), tok, `{"type":"pump","started_at":"2026-10-03 19:00"}`)
	assert.Equal(t, "session_running", r.body["error_code"])
	r = e.do(t, at("19:15"), http.MethodPost, fmt.Sprintf("%s/%d/stop", base, pump), tok, `{"amount_ml":120}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 900, r.data()["duration_seconds"], 0)
	assert.InDelta(t, 120, r.data()["amount_ml"], 0)

	r = e.do(t, at("20:00"), http.MethodGet, base, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Nil(t, d["active"])
	assert.Equal(t, "right", d["next_side"], "the last breast feed ended on the left (the pump after it has no side)")
	s := obj(d, "summary")
	assert.InDelta(t, 3, s["count"], 0)
	assert.InDelta(t, 90, s["bottle_ml"], 0)
	assert.InDelta(t, 120, s["pump_ml"], 0)
	assert.InDelta(t, 840+600+900, s["total_seconds"], 0)
	assert.Len(t, list(d["items"]), 3)
	v, _ = e.feedsCount(t, sara)
	assert.Equal(t, "3", v)

	// Edit and delete keep the mother's count in step.
	r = e.do(t, at("20:00"), http.MethodPut, fmt.Sprintf("%s/%d", base, bottle), tok, `{"type":"breast","started_at":"2026-10-03 18:00","left_minutes":5,"right_minutes":0}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "left", r.data()["last_side"])
	assert.InDelta(t, 300, r.data()["duration_seconds"], 0)
	for _, f := range []uint64{bottle, pump, fid} {
		r = e.do(t, at("20:00"), http.MethodDelete, fmt.Sprintf("%s/%d", base, f), tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	v, _ = e.feedsCount(t, sara)
	assert.Equal(t, "", v, "no feeds left → the sessions' slot is gone")
	r = e.do(t, at("20:00"), http.MethodDelete, fmt.Sprintf("%s/%d", base, fid), tok, "")
	assert.Equal(t, "feed_not_found", r.body["error_code"])
}

func TestBabyLogs_ManualFeedValidation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.child(t, tok)
	base := fmt.Sprintf("/api/v1/children/%d/feeds", id)
	for _, c := range []struct{ body, field string }{
		{`{"type":"cup","started_at":"2026-10-03 10:00"}`, "type"},
		{`{"type":"breast","started_at":"03/10/2026"}`, "started_at"},
		{`{"type":"breast","started_at":"2026-10-03 10:00"}`, "left_minutes"},
		{`{"type":"breast","started_at":"2026-10-03 23:00","left_minutes":5}`, "started_at"},
		{`{"type":"breast","started_at":"2026-06-19 10:00","left_minutes":5}`, "started_at"},
		{`{"type":"bottle","started_at":"2026-10-03 10:00","amount_ml":900}`, "amount_ml"},
	} {
		r := e.do(t, at("12:00"), http.MethodPost, base, tok, c.body)
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, c.body)
		assert.Contains(t, obj(r.body, "errors"), c.field, c.body)
	}
	// Without a postpartum profile the mother's log is left alone.
	r := e.do(t, at("12:00"), http.MethodPost, base, tok, `{"type":"breast","started_at":"2026-10-03 10:00","right_minutes":12}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "right", r.data()["last_side"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM health_log_entries`).Scan(&n))
	assert.Zero(t, n)
}

func TestBabyLogs_SleepAndDiapersAndChildHome(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.child(t, tok)
	sl := fmt.Sprintf("/api/v1/children/%d/sleeps", id)
	dp := fmt.Sprintf("/api/v1/children/%d/diapers", id)

	// A night sleep crossing midnight, a nap and a running sleep.
	r := e.do(t, at("08:00"), http.MethodPost, sl, tok, `{"started_at":"2026-10-02 22:00","ended_at":"2026-10-03 06:00"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, at("13:00"), http.MethodPost, sl+"/start", tok, "")
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	sid := uint64(r.data()["id"].(float64))
	r = e.do(t, at("13:01"), http.MethodPost, sl+"/start", tok, "")
	assert.Equal(t, "sleep_active", r.body["error_code"])
	r = e.do(t, at("14:30"), http.MethodPost, fmt.Sprintf("%s/%d/stop", sl, sid), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 5400, r.data()["duration_seconds"], 0)
	r = e.do(t, at("14:30"), http.MethodPost, sl, tok, `{"started_at":"2026-10-03 10:00","ended_at":"2026-10-03 09:00"}`)
	assert.Contains(t, obj(r.body, "errors"), "ended_at")
	r = e.do(t, at("14:30"), http.MethodPost, sl, tok, `{"started_at":"2026-10-01 10:00","ended_at":"2026-10-02 11:00"}`)
	assert.Contains(t, obj(r.body, "errors"), "ended_at", "over 24 hours")

	r = e.do(t, at("15:00"), http.MethodGet, sl, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 6*3600+5400, obj(r.data(), "summary")["seconds"], 0, "only today's part of the night counts")
	assert.Len(t, list(r.data()["items"]), 2)
	r = e.do(t, at("15:00"), http.MethodGet, sl+"?date=2026-10-02", tok, "")
	assert.InDelta(t, 2*3600, obj(r.data(), "summary")["seconds"], 0)
	r = e.do(t, at("15:00"), http.MethodGet, sl+"?date=yesterday", tok, "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	for _, b := range []string{`{"kind":"wet"}`, `{"kind":"both","changed_at":"2026-10-03 11:00"}`, `{"kind":"dirty","note":"green"}`} {
		r = e.do(t, at("15:00"), http.MethodPost, dp, tok, b)
		require.Equal(t, http.StatusCreated, r.status, r.raw)
	}
	did := uint64(r.data()["id"].(float64))
	r = e.do(t, at("15:00"), http.MethodPost, dp, tok, `{"kind":"soaked"}`)
	assert.Contains(t, obj(r.body, "errors"), "kind")
	r = e.do(t, at("15:05"), http.MethodPut, fmt.Sprintf("%s/%d", dp, did), tok, `{"kind":"wet","changed_at":"2026-10-03 14:50"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "wet", r.data()["kind"])
	assert.Nil(t, r.data()["note"])

	r = e.do(t, at("15:10"), http.MethodPost, fmt.Sprintf("/api/v1/children/%d/feeds/start", id), tok, `{"type":"breast","side":"left"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)

	// The child home's «امروز» card.
	r = e.do(t, at("15:20"), http.MethodGet, fmt.Sprintf("/api/v1/children/%d", id), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	today := obj(r.data(), "today")
	require.NotNil(t, today, r.raw)
	assert.Equal(t, "2026-10-03", today["date"])
	assert.InDelta(t, 1, obj(today, "feeds")["count"], 0)
	assert.InDelta(t, 600, obj(today, "feeds")["left_seconds"], 0)
	assert.Nil(t, obj(today, "feeds")["last"], "no ended feed yet")
	assert.InDelta(t, 3, obj(today, "diapers")["count"], 0)
	assert.InDelta(t, 3, obj(today, "diapers")["wet"], 0)
	assert.Equal(t, true, today["feeding_now"])
	assert.Equal(t, false, today["sleeping_now"])

	r = e.do(t, at("15:20"), http.MethodGet, fmt.Sprintf("/api/v1/children/%d/baby-logs?days=2", id), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-10-02", r.data()["from"])
	days := list(r.data()["days"])
	require.Len(t, days, 2)
	assert.InDelta(t, 2*3600, obj(days[0], "sleep")["seconds"], 0)
	assert.InDelta(t, 100, obj(r.data(), "averages")["left_percent"], 0)
	r = e.do(t, at("15:20"), http.MethodGet, fmt.Sprintf("/api/v1/children/%d/baby-logs?days=40", id), tok, "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
}

// IDOR: another user gets the uniform 404 on every route; the spouse reads a shared child and gets 403 on writes.
func TestBabyLogs_AccessMatrix(t *testing.T) {
	e := setup(t)
	sara, saraTok := e.user(t, "09120000001")
	ali, aliTok := e.user(t, "09120000002")
	_, rezaTok := e.user(t, "09120000003")
	id := e.child(t, saraTok)
	r := e.do(t, at("10:00"), http.MethodPost, fmt.Sprintf("/api/v1/children/%d/feeds/start", id), saraTok, `{"type":"breast"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	fid := uint64(r.data()["id"].(float64))
	r = e.do(t, at("10:00"), http.MethodPost, fmt.Sprintf("/api/v1/children/%d/diapers", id), saraTok, `{"kind":"wet"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	did := uint64(r.data()["id"].(float64))

	res, err := e.db.Exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at) VALUES (?, ?, 'spouse', 'active', NOW(), NOW())`, sara, ali)
	require.NoError(t, err)
	cid, _ := res.LastInsertId()
	res, err = e.db.Exec(`INSERT INTO families (owner_id, spouse_user_id, companion_id) VALUES (?, ?, ?)`, sara, ali, cid)
	require.NoError(t, err)
	fam, _ := res.LastInsertId()
	_, err = e.db.Exec(`INSERT INTO family_children (family_id, child_id) VALUES (?, ?)`, fam, id)
	require.NoError(t, err)

	reads := []string{"/feeds", "/sleeps", "/diapers", "/baby-logs"}
	writes := []struct{ method, path, body string }{
		{http.MethodPost, "/feeds/start", `{"type":"bottle"}`},
		{http.MethodPost, "/feeds", `{"type":"bottle","started_at":"2026-10-03 09:00"}`},
		{http.MethodPost, fmt.Sprintf("/feeds/%d/side", fid), `{"side":"left"}`},
		{http.MethodPost, fmt.Sprintf("/feeds/%d/stop", fid), `{}`},
		{http.MethodPut, fmt.Sprintf("/feeds/%d", fid), `{"type":"bottle","started_at":"2026-10-03 09:00"}`},
		{http.MethodDelete, fmt.Sprintf("/feeds/%d", fid), ""},
		{http.MethodPost, "/sleeps/start", ""},
		{http.MethodPost, "/diapers", `{"kind":"wet"}`},
		{http.MethodPut, fmt.Sprintf("/diapers/%d", did), `{"kind":"dirty"}`},
		{http.MethodDelete, fmt.Sprintf("/diapers/%d", did), ""},
	}
	base := fmt.Sprintf("/api/v1/children/%d", id)
	for _, p := range reads {
		r = e.do(t, at("10:05"), http.MethodGet, base+p, rezaTok, "")
		assert.Equal(t, http.StatusNotFound, r.status, p)
		assert.Equal(t, "child_not_found", r.body["error_code"], p)
		r = e.do(t, at("10:05"), http.MethodGet, base+p, aliTok, "")
		assert.Equal(t, http.StatusOK, r.status, "spouse reads "+p)
	}
	for _, w := range writes {
		r = e.do(t, at("10:05"), w.method, base+w.path, rezaTok, w.body)
		assert.Equal(t, http.StatusNotFound, r.status, w.path)
		r = e.do(t, at("10:05"), w.method, base+w.path, aliTok, w.body)
		assert.Equal(t, http.StatusForbidden, r.status, "spouse writes "+w.path)
		assert.Equal(t, "child_read_only", r.body["error_code"], w.path)
	}
	// The owner's other child's rows are not reachable through this child.
	r = e.do(t, at("10:06"), http.MethodPost, "/api/v1/children", saraTok, `{"name":"Sam","birth_date":"2025-01-01"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	sam := uint64(r.data()["id"].(float64))
	r = e.do(t, at("10:06"), http.MethodDelete, fmt.Sprintf("/api/v1/children/%d/diapers/%d", sam, did), saraTok, "")
	assert.Equal(t, "diaper_not_found", r.body["error_code"])
	r = e.do(t, at("10:06"), http.MethodPost, fmt.Sprintf("/api/v1/children/%d/feeds/%d/stop", sam, fid), saraTok, `{}`)
	assert.Equal(t, "feed_not_found", r.body["error_code"])
	var reads2 int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND section = 'children'`, sara, ali).Scan(&reads2))
	assert.Positive(t, reads2, "spouse reads are audited")
}

// Concurrent starts: exactly one running feed per child.
func TestBabyLogs_ConcurrentStart(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	id := e.child(t, tok)
	var wg sync.WaitGroup
	codes := make(chan int, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := e.do(t, at("11:00"), http.MethodPost, fmt.Sprintf("/api/v1/children/%d/feeds/start", id), tok, `{"type":"breast","side":"left"}`)
			codes <- r.status
		}()
	}
	wg.Wait()
	close(codes)
	got := map[int]int{}
	for c := range codes {
		got[c]++
	}
	assert.Equal(t, map[int]int{http.StatusCreated: 1, http.StatusConflict: 7}, got)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM baby_feeds WHERE child_id = ?`, id).Scan(&n))
	assert.Equal(t, 1, n)
}
