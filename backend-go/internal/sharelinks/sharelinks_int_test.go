package sharelinks_test

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
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/sharelinks"
	"github.com/ritme/backend-go/internal/sharelinks/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fixed is the request clock: Tuesday 2026-10-06 09:00 Tehran.
var fixed = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)

type env struct {
	db   *sql.DB
	app  *fiber.App
	iss  *passport.Issuer
	plus *plus.Service
	svc  *sharelinks.Service
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
	records := healthrecord.NewService(db, nil)
	rh := healthrecord.NewHandlers(records, clock.Fixed(fixed))
	svc := sharelinks.NewService(store.New(db), records, quiet)
	h := sharelinks.NewHandlers(svc, clock.Fixed(fixed))
	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Get("/api/v1/health-record/report", locale, guard, rh.Report)
	p := "/api/v1/health-record/share-links"
	app.Get(p, locale, guard, h.Index)
	app.Post(p, locale, guard, gate.Require(plus.PDFShare), h.Store)
	app.Delete(p+"/:id", locale, guard, h.Destroy)
	app.Get("/api/v1/shared-reports/:token", locale, h.Show)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), plus: plusSvc, svc: svc}
}

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

// user creates a user (with a running Plus trial when withPlus) and returns its id and token.
func (e *env) user(t *testing.T, mobile string, withPlus bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Maryam', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id64, err := res.LastInsertId()
	require.NoError(t, err)
	id := uint64(id64)
	if withPlus {
		require.NoError(t, e.plus.StartTrial(context.Background(), id, time.Now().In(civildate.Tehran).Truncate(time.Second)))
	}
	tok, err := e.iss.Issue(context.Background(), id, time.Now())
	require.NoError(t, err)
	return id, tok.AccessToken
}

type response struct {
	status int
	raw    string
	body   map[string]any
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, raw: string(raw), body: m}
}

// seed gives uid a profile, a medication with a private note, a checkup with a note and a blood pressure reading.
func (e *env) seed(t *testing.T, uid uint64) {
	t.Helper()
	e.exec(t, `INSERT INTO user_profiles (user_id, birthday, weight, height, created_at, updated_at)
		VALUES (?, '1993-05-01', 58.00, 164, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO reminders (user_id, type, title, subtitle, notes, recurrence, is_active, created_at, updated_at)
		VALUES (?, 'medication', 'Levothyroxine', '50 mcg', 'secret medication note', 'daily', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, created_at, updated_at)
		VALUES (?, 'bp', '2026-10-05 08:00:00', 121, 78, '2026-10-05 08:00:00', '2026-10-05 08:00:00')`, uid)
	var pap uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM checkup_types WHERE user_id IS NULL ORDER BY sort_order, id LIMIT 1").Scan(&pap))
	e.exec(t, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, note, created_at, updated_at)
		VALUES (?, ?, '2026-09-10', 'normal', 'secret checkup note', '2026-09-10 09:00:00', '2026-09-10 09:00:00')`, uid, pap)
	e.exec(t, `INSERT INTO health_record_pregnancies (user_id, outcome, ended_on, created_at, updated_at)
		VALUES (?, 'vaginal', '2024-01-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
}

var all = []string{"basics", "conditions", "medications", "allergies", "cycle", "vitals", "pregnancies", "checkups", "labs"}

func assertNoIDs(t *testing.T, raw string) {
	t.Helper()
	assert.NotContains(t, raw, `"id":`, "a share view carries no row ids")
	assert.NotContains(t, raw, "secret medication note")
	assert.NotContains(t, raw, "secret checkup note")
	assert.NotContains(t, raw, `"editable":true`)
}

func TestShareLinks_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/health-record/report?range=3m&sections=basics"},
		{http.MethodGet, "/api/v1/health-record/share-links"},
		{http.MethodPost, "/api/v1/health-record/share-links"},
		{http.MethodDelete, "/api/v1/health-record/share-links/1"},
	} {
		r := e.do(t, c.method, c.path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestReport_ShareAudienceWindowAndValidation(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	e.seed(t, uid)

	r := e.do(t, http.MethodGet, "/api/v1/health-record/report?range=3m&sections=vitals,basics,medications,checkups,pregnancies", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rng, _ := r.data()["range"].(map[string]any)
	assert.Equal(t, map[string]any{"key": "3m", "from": "2026-07-08", "to": "2026-10-06", "days": float64(91)}, rng)
	rec, _ := r.data()["record"].(map[string]any)
	assert.Equal(t, "share", rec["audience"])
	secs, _ := rec["sections"].([]any)
	keys := []string{}
	for _, s := range secs {
		keys = append(keys, s.(map[string]any)["key"].(string))
	}
	assert.Equal(t, []string{"basics", "medications", "vitals", "pregnancies", "checkups"}, keys, "screen order, only the chosen sections")
	assertNoIDs(t, r.raw)
	assert.Contains(t, r.raw, "Levothyroxine")

	r = e.do(t, http.MethodGet, "/api/v1/health-record/report?range=custom&from=2026-09-01&sections=vitals", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rng, _ = r.data()["range"].(map[string]any)
	assert.Equal(t, "2026-09-01", rng["from"])
	assert.EqualValues(t, 36, rng["days"])

	for _, q := range []string{
		"range=2y&sections=basics", "range=3m", "range=3m&sections=basics,unknown", "range=custom&sections=basics",
		"range=custom&from=2020-01-01&sections=basics", "range=custom&from=2026-10-06&sections=basics",
	} {
		r = e.do(t, http.MethodGet, "/api/v1/health-record/report?"+q, tok, nil)
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, q+" "+r.raw)
		assert.Equal(t, false, r.body["success"])
	}
}

func TestShareLinks_PlusRequired(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001", false)
	r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "3m", "sections": all})
	assert.Equal(t, http.StatusPaymentRequired, r.status, r.raw)
	assert.Equal(t, "plus_required", r.body["error_code"])
	assert.Equal(t, "plus.pdf_share", r.body["feature"])
}

func TestShareLinks_CreateOpenRevoke(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	e.seed(t, uid)
	_, otherTok := e.user(t, "09120000002", true)

	r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "9m", "sections": []string{}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok,
		map[string]any{"range": "3m", "sections": all, "question": strings.Repeat("x", 301)})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)

	r = e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok,
		map[string]any{"range": "6m", "sections": all, "question": "  Is my night  blood pressure worrying? "})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	token, _ := r.data()["token"].(string)
	require.Len(t, token, 43)
	assert.Equal(t, "active", r.data()["status"])
	assert.Equal(t, "2026-10-13T09:00:00+03:30", r.data()["expires_at"])
	id := uint64(r.data()["id"].(float64))

	// Stored: only the token hash, ciphertext without any readable value.
	var hash, payload string
	require.NoError(t, e.db.QueryRow("SELECT token_hash, payload FROM health_share_links WHERE id = ?", id).Scan(&hash, &payload))
	assert.Equal(t, sharelinks.HashToken(token), hash)
	assert.NotContains(t, payload, "Maryam")
	assert.NotContains(t, payload, "Levothyroxine")
	assert.NotContains(t, payload, token)

	// Public read: no auth, the frozen share view + the question.
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Is my night blood pressure worrying?", r.data()["question"])
	rec, _ := r.data()["record"].(map[string]any)
	person, _ := rec["person"].(map[string]any)
	assert.Equal(t, "Maryam", person["name"])
	assertNoIDs(t, r.raw)
	assert.Equal(t, "2026-10-13T09:00:00+03:30", r.data()["expires_at"])
	var views int
	require.NoError(t, e.db.QueryRow("SELECT view_count FROM health_share_links WHERE id = ?", id).Scan(&views))
	assert.Equal(t, 1, views)

	// A later edit does not change the frozen report.
	e.exec(t, "UPDATE reminders SET title = 'Changed' WHERE user_id = ?", uid)
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Contains(t, r.raw, "Levothyroxine")

	// Owner list; another user sees nothing and cannot revoke.
	r = e.do(t, http.MethodGet, "/api/v1/health-record/share-links", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	items, _ := r.data()["items"].([]any)
	require.Len(t, items, 1)
	assert.EqualValues(t, 2, items[0].(map[string]any)["views"])
	assert.NotContains(t, r.raw, token)
	assert.NotContains(t, r.raw, "Levothyroxine")
	assert.EqualValues(t, 1, r.data()["active_count"])
	r = e.do(t, http.MethodGet, "/api/v1/health-record/share-links", otherTok, nil)
	assert.Empty(t, r.data()["items"])
	r = e.do(t, http.MethodDelete, "/api/v1/health-record/share-links/"+jsonNum(id), otherTok, nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "share_link_not_found", r.body["error_code"])
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	assert.Equal(t, http.StatusOK, r.status, "still readable after a foreign revoke attempt")

	// Revoke → 410, ciphertext wiped.
	r = e.do(t, http.MethodDelete, "/api/v1/health-record/share-links/"+jsonNum(id), tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "revoked", r.data()["status"])
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	assert.Equal(t, http.StatusGone, r.status)
	assert.Equal(t, "share_link_revoked", r.body["error_code"])
	var p sql.NullString
	require.NoError(t, e.db.QueryRow("SELECT payload FROM health_share_links WHERE id = ?", id).Scan(&p))
	assert.False(t, p.Valid)
}

func jsonNum(id uint64) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func TestShareLinks_ExpiredUnknownAndPurge(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "1y", "sections": []string{"basics"}})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	token := r.data()["token"].(string)

	e.exec(t, "UPDATE health_share_links SET expires_at = '2026-10-06 08:59:59' WHERE user_id = ?", uid)
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+token, "", nil)
	assert.Equal(t, http.StatusGone, r.status, r.raw)
	assert.Equal(t, "share_link_expired", r.body["error_code"])

	for _, bad := range []string{strings.Repeat("A", 43), "short", strings.Repeat("A", 44)} {
		r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+bad, "", nil)
		assert.Equal(t, http.StatusNotFound, r.status, bad)
		assert.Equal(t, "share_link_not_found", r.body["error_code"])
	}

	require.NoError(t, e.svc.Purge(context.Background(), fixed))
	var p sql.NullString
	require.NoError(t, e.db.QueryRow("SELECT payload FROM health_share_links WHERE user_id = ?", uid).Scan(&p))
	assert.False(t, p.Valid, "expired ciphertext wiped")
	require.NoError(t, e.svc.Purge(context.Background(), fixed.Add(31*24*time.Hour)))
	var n int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM health_share_links WHERE user_id = ?", uid).Scan(&n))
	assert.Zero(t, n, "rows deleted 30 days after expiry")
}

func TestShareLinks_ActiveLimit(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	for i := range sharelinks.MaxActive {
		e.exec(t, `INSERT INTO health_share_links (user_id, token_hash, payload, sections, range_from, range_to, expires_at, created_at, updated_at)
			VALUES (?, ?, NULL, '["basics"]', '2026-07-01', '2026-10-06', '2026-10-10 09:00:00', '2026-10-05 09:00:00', '2026-10-05 09:00:00')`,
			uid, strings.Repeat(string(rune('a'+i)), 64))
	}
	r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "3m", "sections": []string{"basics"}})
	assert.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, "share_link_limit", r.body["error_code"])
}
