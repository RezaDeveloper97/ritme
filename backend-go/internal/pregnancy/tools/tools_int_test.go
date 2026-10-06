package tools_test

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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
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
	h := tools.NewHandlers(tools.NewService(db, pregnancyalerts.New(store.New(db))), clock.Real{})
	v1 := pregnancy.NewHandlers(store.New(db))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/pregnancy/fetal-movement", locale, guard, v1.FetalIndex)
	app.Post("/api/v1/pregnancy/fetal-movement", locale, guard, v1.FetalStore)
	k := "/api/v1/pregnancy/kick-sessions"
	app.Get(k, locale, guard, h.Kicks)
	app.Post(k, locale, guard, h.StartKicks)
	app.Post(k+"/:id/kicks", locale, guard, h.Kick)
	app.Delete(k+"/:id/kicks", locale, guard, h.UndoKick)
	app.Post(k+"/:id/stop", locale, guard, h.StopKicks)
	app.Delete(k+"/:id", locale, guard, h.DestroyKicks)
	c := "/api/v1/pregnancy/contractions"
	app.Get(c, locale, guard, h.Contractions)
	app.Post(c+"/start", locale, guard, h.StartContraction)
	app.Post(c+"/stop", locale, guard, h.StopContraction)
	app.Get(c+"/sessions/:id", locale, guard, h.Timing)
	app.Post(c+"/sessions/:id/finish", locale, guard, h.FinishTiming)
	app.Delete(c+"/sessions/:id", locale, guard, h.DestroyTiming)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

// user creates a user; pregnant ones have an LMP of 2026-02-20 (week 33 on 2026-10-03).
func (e *env) user(t *testing.T, mobile string, pregnant bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if pregnant {
		_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
			VALUES (?, 1, 'lmp', '2026-02-20', NOW(), NOW())`, id)
		require.NoError(t, err)
	}
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

var t0 = time.Date(2026, 10, 3, 20, 0, 0, 0, time.FixedZone("IRST", 12600))

func stamp(t time.Time) string { return t.Format(time.RFC3339) }

func (e *env) do(t *testing.T, when time.Time, method, path, token, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept-Language", "en")
	req.Header.Set(clock.Header, stamp(when))
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func TestTools_RequireAuthAndPregnancy(t *testing.T) {
	e := setup(t)
	for _, path := range []string{"/api/v1/pregnancy/kick-sessions", "/api/v1/pregnancy/contractions"} {
		r := e.do(t, t0, http.MethodGet, path, "", "")
		assert.Equal(t, http.StatusUnauthorized, r.status, path)
		assert.Equal(t, "unauthenticated", r.body["error_code"], path)
	}
	_, tok := e.user(t, "09120000001", false)
	r := e.do(t, t0, http.MethodGet, "/api/v1/pregnancy/kick-sessions", tok, "")
	assert.Equal(t, http.StatusOK, r.status, "history stays readable")
	r = e.do(t, t0, http.MethodPost, "/api/v1/pregnancy/kick-sessions", tok, "")
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "pregnancy_not_active", r.body["error_code"])
	r = e.do(t, t0, http.MethodPost, "/api/v1/pregnancy/contractions/start", tok, "")
	assert.Equal(t, "pregnancy_not_active", r.body["error_code"])
	r = e.do(t, t0, http.MethodGet, "/api/v1/pregnancy/contractions", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"interval_max_minutes": 5.0, "duration_min_seconds": 45.0, "run_minutes": 60.0}, r.data()["params"],
		"the seeded contractions_511 row")
}

// The artboard's count: 10 movements in 24:10, an undo, stop → the day's fetal movement log.
func TestTools_KickCounter(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true)
	base := "/api/v1/pregnancy/kick-sessions"
	r := e.do(t, t0, http.MethodPost, base, tok, "")
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := uint64(r.data()["id"].(float64))
	assert.InDelta(t, 33, r.data()["pregnancy_week"], 0)
	r = e.do(t, t0, http.MethodPost, base, tok, "")
	assert.Equal(t, "kick_session_active", r.body["error_code"])

	kick := fmt.Sprintf("%s/%d/kicks", base, id)
	for i := 1; i <= 11; i++ {
		r = e.do(t, t0.Add(time.Duration(i)*145*time.Second), http.MethodPost, kick, tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	assert.InDelta(t, 11, r.data()["kicks"], 0)
	assert.InDelta(t, 1450, r.data()["time_to_target_seconds"], 0, "the 10th at 24:10")
	r = e.do(t, t0.Add(27*time.Minute), http.MethodDelete, kick, tok, "")
	assert.InDelta(t, 10, r.data()["kicks"], 0)
	assert.InDelta(t, 1450, r.data()["time_to_target_seconds"], 0)
	r = e.do(t, t0.Add(27*time.Minute), http.MethodDelete, kick, tok, "")
	assert.InDelta(t, 9, r.data()["kicks"], 0)
	assert.Nil(t, r.data()["time_to_target_seconds"], "under 10 again")
	r = e.do(t, t0.Add(28*time.Minute), http.MethodPost, kick, tok, "")
	assert.InDelta(t, 1680, r.data()["time_to_target_seconds"], 0)

	r = e.do(t, t0.Add(30*time.Minute), http.MethodGet, base, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 10, obj(r.data(), "active")["kicks"], 0)
	assert.Equal(t, map[string]any{"date": "2026-10-03", "kicks": 10.0, "sessions": 1.0}, r.data()["today"])

	r = e.do(t, t0.Add(30*time.Minute), http.MethodPost, fmt.Sprintf("%s/%d/stop", base, id), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["is_active"])
	assert.Equal(t, true, r.data()["reached_target"])
	assert.InDelta(t, 1800, r.data()["elapsed_seconds"], 0)
	r = e.do(t, t0.Add(31*time.Minute), http.MethodPost, kick, tok, "")
	assert.Equal(t, "session_ended", r.body["error_code"])

	r = e.do(t, t0.Add(31*time.Minute), http.MethodGet, "/api/v1/pregnancy/fetal-movement", tok, "")
	logs := list(r.data()["logs"])
	require.Len(t, logs, 1, r.raw)
	fm := obj(logs[0])
	assert.Equal(t, "normal", fm["movement_status"])
	assert.InDelta(t, 10, fm["movement_count"], 0)
	assert.InDelta(t, 33, fm["pregnancy_week"], 0)
	assert.Equal(t, "20:00:00", fm["first_movement_time"])
	assert.Equal(t, "20:28:00", fm["last_movement_time"])
	var felt bool
	require.NoError(t, e.db.QueryRow(`SELECT fetal_movement_felt FROM pregnancy_profiles WHERE user_id = ?`, sara).Scan(&felt))
	assert.True(t, felt)

	// A second, short session adds to the day; deleting it rewrites the total.
	r = e.do(t, t0.Add(2*time.Hour), http.MethodPost, base, tok, "")
	id2 := uint64(r.data()["id"].(float64))
	for i := 1; i <= 3; i++ {
		e.do(t, t0.Add(2*time.Hour+time.Duration(i)*time.Minute), http.MethodPost, fmt.Sprintf("%s/%d/kicks", base, id2), tok, "")
	}
	r = e.do(t, t0.Add(3*time.Hour), http.MethodPost, fmt.Sprintf("%s/%d/stop", base, id2), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, t0.Add(3*time.Hour), http.MethodGet, "/api/v1/pregnancy/fetal-movement", tok, "")
	assert.InDelta(t, 13, obj(list(r.data()["logs"])[0])["movement_count"], 0)
	r = e.do(t, t0.Add(3*time.Hour), http.MethodDelete, fmt.Sprintf("%s/%d", base, id2), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, t0.Add(3*time.Hour), http.MethodGet, "/api/v1/pregnancy/fetal-movement", tok, "")
	assert.InDelta(t, 10, obj(list(r.data()["logs"])[0])["movement_count"], 0)
	r = e.do(t, t0.Add(3*time.Hour), http.MethodGet, base, tok, "")
	assert.Len(t, list(r.data()["history"]), 1)
}

// A day the user logged by hand keeps her status (here «reduced», which raised its own alert); the kick total
// replaces the count only.
func TestTools_KickKeepsManualStatus(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001", true)
	r := e.do(t, t0, http.MethodPost, "/api/v1/pregnancy/fetal-movement", tok,
		`{"log_date":"2026-10-03","pregnancy_week":33,"movement_status":"reduced","movement_count":3,"notes":"quiet day"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, t0.Add(time.Hour), http.MethodPost, "/api/v1/pregnancy/kick-sessions", tok, "")
	id := uint64(r.data()["id"].(float64))
	for i := 1; i <= 4; i++ {
		e.do(t, t0.Add(time.Hour+time.Duration(i)*time.Minute), http.MethodPost, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d/kicks", id), tok, "")
	}
	r = e.do(t, t0.Add(4*time.Hour), http.MethodPost, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d/stop", id), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["low_count"], "4 in 3 hours: the call guidance")
	r = e.do(t, t0.Add(4*time.Hour), http.MethodGet, "/api/v1/pregnancy/fetal-movement", tok, "")
	fm := obj(list(r.data()["logs"])[0])
	assert.Equal(t, "reduced", fm["movement_status"])
	assert.InDelta(t, 4, fm["movement_count"], 0)
	assert.Equal(t, "quiet day", fm["notes"])
}

// 13 contractions every 5 minutes, 60 s each: the 13th stop meets 5-1-1 (61 min) and raises the urgent alert once.
func TestTools_ContractionTimer511(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001", true)
	c := "/api/v1/pregnancy/contractions"
	r := e.do(t, t0, http.MethodPost, c+"/stop", tok, "")
	assert.Equal(t, "no_contraction_running", r.body["error_code"])

	var sessionID uint64
	for i := range 13 {
		start := t0.Add(time.Duration(i) * 5 * time.Minute)
		r = e.do(t, start, http.MethodPost, c+"/start", tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		sessionID = uint64(r.data()["id"].(float64))
		if i == 0 {
			assert.InDelta(t, 0, r.data()["elapsed_seconds"], 0)
			r = e.do(t, start.Add(10*time.Second), http.MethodPost, c+"/start", tok, "")
			assert.Equal(t, "contraction_running", r.body["error_code"])
		}
		r = e.do(t, start.Add(time.Minute), http.MethodPost, c+"/stop", tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		alerts := list(r.data()["alerts"])
		if i < 12 {
			assert.Empty(t, alerts, "not yet an hour (contraction %d)", i+1)
			assert.Equal(t, false, obj(r.data(), "session", "five_one_one")["met"])
			continue
		}
		require.Len(t, alerts, 1, r.raw)
		a := obj(alerts[0])
		assert.Equal(t, "contractions_511", a["rule_key"])
		assert.Equal(t, "urgent", a["level"])
		assert.Equal(t, "You timed 13 contractions in the last 61 minutes, on average 5:00 apart and 1:00 long", a["what_we_saw"])
		assert.Contains(t, a["contact"], "115")
		assert.Equal(t, "call", obj(list(a["actions"])[0])["key"])
		s := obj(r.data(), "session")
		assert.Equal(t, true, obj(s, "five_one_one")["met"])
		assert.Equal(t, stamp(start.Add(time.Minute)), s["alert_at"])
		assert.InDelta(t, 60, s["avg_duration_seconds"], 0)
		assert.InDelta(t, 300, s["avg_interval_seconds"], 0)
		assert.Len(t, list(s["contractions"]), 13)
	}

	// The next one does not raise it again (once per session).
	r = e.do(t, t0.Add(65*time.Minute), http.MethodPost, c+"/start", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, t0.Add(66*time.Minute), http.MethodPost, c+"/stop", tok, "")
	assert.Empty(t, list(r.data()["alerts"]))
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:contractions_511'`, sara).Scan(&n))
	assert.Equal(t, 1, n)

	// Finish with one in progress: it ends now.
	e.do(t, t0.Add(70*time.Minute), http.MethodPost, c+"/start", tok, "")
	r = e.do(t, t0.Add(70*time.Minute+30*time.Second), http.MethodPost, fmt.Sprintf("%s/sessions/%d/finish", c, sessionID), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["is_active"])
	assert.Nil(t, r.data()["running"])
	assert.InDelta(t, 15, r.data()["count"], 0)
	r = e.do(t, t0.Add(71*time.Minute), http.MethodPost, fmt.Sprintf("%s/sessions/%d/finish", c, sessionID), tok, "")
	assert.Equal(t, "session_ended", r.body["error_code"])

	r = e.do(t, t0.Add(72*time.Minute), http.MethodGet, c, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["active"])
	hist := list(r.data()["history"])
	require.Len(t, hist, 1)
	assert.NotContains(t, obj(hist[0]), "contractions", "the history is summaries")
	assert.Equal(t, true, obj(hist[0], "five_one_one")["met"])

	// A new tap opens a new session.
	r = e.do(t, t0.Add(80*time.Minute), http.MethodPost, c+"/start", tok, "")
	assert.NotEqual(t, float64(sessionID), r.data()["id"])
}

// B-N5-09: with the admin-set contact_phone param the urgent alert's contact is {text, phone}.
func TestTools_ContractionTimer511ContactPhone(t *testing.T) {
	e := setup(t)
	_, err := e.db.Exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.params.contact_phone', '021 6612 3456')
		WHERE ` + "`group`" + ` = 'pregnancy_alert' AND item_key = 'contractions_511'`)
	require.NoError(t, err)
	_, tok := e.user(t, "09120000001", true)
	c := "/api/v1/pregnancy/contractions"
	var r response
	for i := range 13 {
		start := t0.Add(time.Duration(i) * 5 * time.Minute)
		r = e.do(t, start, http.MethodPost, c+"/start", tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		r = e.do(t, start.Add(time.Minute), http.MethodPost, c+"/stop", tok, "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	alerts := list(r.data()["alerts"])
	require.Len(t, alerts, 1, r.raw)
	contact := obj(obj(alerts[0]), "contact")
	assert.Equal(t, "021 6612 3456", contact["phone"])
	assert.Contains(t, contact["text"], "115")
}

// IDOR: another user's sessions are a uniform 404.
func TestTools_OtherUsersSessions(t *testing.T) {
	e := setup(t)
	_, sara := e.user(t, "09120000001", true)
	_, reza := e.user(t, "09120000002", true)
	r := e.do(t, t0, http.MethodPost, "/api/v1/pregnancy/kick-sessions", sara, "")
	kid := uint64(r.data()["id"].(float64))
	r = e.do(t, t0, http.MethodPost, "/api/v1/pregnancy/contractions/start", sara, "")
	cid := uint64(r.data()["id"].(float64))

	for _, c := range []struct{ method, path string }{
		{http.MethodPost, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d/kicks", kid)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d/kicks", kid)},
		{http.MethodPost, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d/stop", kid)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/pregnancy/kick-sessions/%d", kid)},
		{http.MethodGet, fmt.Sprintf("/api/v1/pregnancy/contractions/sessions/%d", cid)},
		{http.MethodPost, fmt.Sprintf("/api/v1/pregnancy/contractions/sessions/%d/finish", cid)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/pregnancy/contractions/sessions/%d", cid)},
		{http.MethodGet, "/api/v1/pregnancy/contractions/sessions/abc"},
	} {
		r = e.do(t, t0.Add(time.Minute), c.method, c.path, reza, "")
		assert.Equal(t, http.StatusNotFound, r.status, c.path)
		assert.Equal(t, "session_not_found", r.body["error_code"], c.path)
	}
	// Reza's own stop does not touch Sara's running contraction.
	r = e.do(t, t0.Add(time.Minute), http.MethodPost, "/api/v1/pregnancy/contractions/stop", reza, "")
	assert.Equal(t, "no_contraction_running", r.body["error_code"])
	r = e.do(t, t0.Add(time.Minute), http.MethodGet, "/api/v1/pregnancy/kick-sessions", reza, "")
	assert.Nil(t, r.data()["active"])
	r = e.do(t, t0.Add(time.Minute), http.MethodGet, fmt.Sprintf("/api/v1/pregnancy/contractions/sessions/%d", cid), sara, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.NotNil(t, r.data()["running"])
}

// Concurrent starts: one kick session; concurrent first taps: one contraction session with one contraction.
func TestTools_ConcurrentStarts(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	for _, path := range []string{"/api/v1/pregnancy/kick-sessions", "/api/v1/pregnancy/contractions/start"} {
		var wg sync.WaitGroup
		codes := make(chan int, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes <- e.do(t, t0, http.MethodPost, path, tok, "").status
			}()
		}
		wg.Wait()
		close(codes)
		ok := 0
		for c := range codes {
			if c == http.StatusOK || c == http.StatusCreated {
				ok++
			} else {
				assert.Equal(t, http.StatusConflict, c, path)
			}
		}
		assert.Equal(t, 1, ok, path)
	}
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_kick_sessions WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 1, n)
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_contraction_sessions WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 1, n)
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_contractions WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 1, n)
}
