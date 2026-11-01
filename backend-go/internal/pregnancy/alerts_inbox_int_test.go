package pregnancy

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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

const (
	inboxClientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	inboxNow      = "2026-09-23T10:00:00+03:30"
)

type inboxEnv struct {
	db  *sql.DB
	q   *store.Queries
	app *fiber.App
	iss *passport.Issuer
}

func setupInboxAlerts(t *testing.T) *inboxEnv {
	t.Helper()
	if !testdb.Enabled() {
		t.Skip("TEST_DB_DSN not set")
	}
	db := testdb.New(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, inboxClientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	aq := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, aq, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	inboxalerts.Install(db, quiet)
	t.Cleanup(func() { pregnancyalerts.InstallBackend(nil) })
	q := store.New(db)
	h := NewHandlers(q)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	g := app.Group("/api/v1/pregnancy", locale, guard)
	g.Post("/symptoms", h.SymptomStore)
	g.Get("/alerts", h.AlertIndex)
	g.Get("/alerts/summary", h.AlertSummary)
	g.Post("/alerts/read-all", h.AlertReadAll)
	g.Get("/alerts/:id", h.AlertShow)
	g.Post("/alerts/:id/read", h.AlertRead)
	g.Post("/alerts/:id/dismiss", h.AlertDismiss)
	return &inboxEnv{db: db, q: q, app: app, iss: passport.NewIssuer(key, aq, clock.Real{}, 365)}
}

// user is pregnant by LMP 2026-07-01 (12w0d on 2026-09-23).
func (e *inboxEnv) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id)
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test id
}

func (e *inboxEnv) do(t *testing.T, method, path, token, body string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, inboxNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	data, _ := out["data"].(map[string]any)
	return resp.StatusCode, data, string(raw)
}

func (e *inboxEnv) state(t *testing.T, id uint64) string {
	t.Helper()
	var s string
	require.NoError(t, e.db.QueryRow(`SELECT state FROM message_inbox WHERE id = ?`, id).Scan(&s))
	return s
}

func alertList(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	list, _ := data["alerts"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, a.(map[string]any))
	}
	return out
}

// v1Keys is the v1 PregnancyAlert key order (alertModel), which the inbox rows keep.
var v1Keys = []string{"id", "user_id", "alert_level", "alert_type", "title", "message", "pregnancy_week", "trigger_symptoms",
	"medical_history_flags", "is_read", "is_dismissed", "read_at", "dismissed_at", "recommended_actions", "created_at",
	"updated_at"}

// alertKeys are the keys of the first alert object after marker in raw, in order.
func alertKeys(t *testing.T, raw, marker string) []string {
	t.Helper()
	i := strings.Index(raw, marker)
	require.GreaterOrEqual(t, i, 0, raw)
	dec := json.NewDecoder(strings.NewReader(raw[i+len(marker)-1:]))
	_, err := dec.Token() // {
	require.NoError(t, err)
	var keys []string
	for dec.More() {
		k, err := dec.Token()
		require.NoError(t, err)
		keys = append(keys, k.(string))
		var skip json.RawMessage
		require.NoError(t, dec.Decode(&skip))
	}
	return keys
}

// T-M10-08b (D-165): the v1 alert routes read the alerts inbox. A critical symptom save is an emergency in the v1 list
// and the summary; read acknowledges, dismiss archives, read-all acknowledges every active row; legacy v1 rows stay
// listed next to them (v2 legacy rows were migrated by 00113 and are not listed twice); nothing writes a new
// pregnancy_alerts row.
func TestV1Alerts_ServedFromTheInbox(t *testing.T) {
	e := setupInboxAlerts(t)
	uid, tok := e.user(t, "09120000801")
	_, otherTok := e.user(t, "09120000802")

	st, data, raw := e.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok,
		`{"log_date":"2026-09-23","has_bleeding":true,"bleeding_severity":"severe"}`)
	require.Equal(t, http.StatusCreated, st, raw)
	saved := alertList(t, data)
	require.Len(t, saved, 1, raw)
	assert.Equal(t, "critical_symptom", saved[0]["alert_type"])
	assert.Equal(t, "emergency", saved[0]["alert_level"])
	var inboxID uint64
	require.NoError(t, e.db.QueryRow(`SELECT id FROM message_inbox WHERE user_id = ? AND rule_key = 'preg.critical_symptom'`, uid).Scan(&inboxID))
	assert.EqualValues(t, inboxID, saved[0]["id"])

	// Legacy rows: a v1 warning (listed) and a v2 row (moved to the inbox by 00113 — not listed).
	_, err := e.db.Exec(`INSERT INTO pregnancy_alerts (user_id, alert_level, alert_type, title, message, created_at, updated_at) VALUES
		(?, 'warning', 'symptom_based', 'Legacy', 'm', '2026-09-20 09:00:00', '2026-09-20 09:00:00'),
		(?, 'emergency', 'v2:critical_symptom', 'v2', 'm', '2026-09-21 09:00:00', '2026-09-21 09:00:00')`, uid, uid)
	require.NoError(t, err)
	legacyCount := func() int {
		var n int
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid).Scan(&n))
		return n
	}

	// The save also opened the day's calendar rows (week entered, weight missing): the open preg.* rows.
	var open int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM message_inbox WHERE user_id = ? AND rule_key LIKE 'preg.%'`, uid).Scan(&open))
	require.GreaterOrEqual(t, open, 1)

	st, data, raw = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts", tok, "")
	require.Equal(t, http.StatusOK, st, raw)
	list := alertList(t, data)
	require.Len(t, list, open+1, raw)
	var a map[string]any
	for _, x := range list[:open] { // the inbox rows are newer than the legacy row
		if x["alert_type"] == "critical_symptom" {
			a = x
		}
	}
	require.NotNil(t, a, raw)
	assert.EqualValues(t, inboxID, a["id"])
	assert.EqualValues(t, 13, a["pregnancy_week"], "the week of the first hit")
	assert.Equal(t, "critical_symptom", a["alert_type"])
	assert.Equal(t, "emergency", a["alert_level"])
	assert.Equal(t, false, a["is_read"])
	assert.Equal(t, false, a["is_dismissed"])
	assert.NotEmpty(t, a["title"])
	assert.NotEmpty(t, a["message"])
	assert.Equal(t, map[string]any{"bleeding": true}, a["trigger_symptoms"])
	assert.NotEmpty(t, a["recommended_actions"])
	assert.Equal(t, v1Keys, alertKeys(t, raw, `"alerts":[{`))
	assert.Equal(t, "Legacy", list[open]["title"])
	counts := data["counts"].(map[string]any)
	assert.EqualValues(t, open+1, counts["total"])
	assert.EqualValues(t, open+1, counts["unread"])
	assert.EqualValues(t, 1, counts["emergency"])
	assert.EqualValues(t, 1, counts["warning"])
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts?level=emergency", tok, "")
	require.Len(t, alertList(t, data), 1)
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts?level=warning", tok, "")
	require.Len(t, alertList(t, data), 1)

	st, data, raw = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/summary", tok, "")
	require.Equal(t, http.StatusOK, st, raw)
	assert.Equal(t, true, data["has_emergency"], raw)
	assert.Equal(t, true, data["has_unread"])
	latest := data["latest_emergency"].(map[string]any)
	assert.EqualValues(t, inboxID, latest["id"])
	assert.Equal(t, v1Keys, alertKeys(t, raw, `"latest_emergency":{`))

	// The v2 Today badge counts the active preg.* inbox rows.
	extras := func() int64 {
		ex, err := e.q.GetV2TodayExtras(context.Background(), store.GetV2TodayExtrasParams{UserID: uid,
			AlertsSince: sql.NullTime{Time: time.Date(2026, 9, 17, 0, 0, 0, 0, time.Local), Valid: true},
			Now:         sql.NullTime{Time: time.Now(), Valid: true}, CurrentWeek: 13})
		require.NoError(t, err)
		return ex.UnreadAlerts
	}
	assert.EqualValues(t, open, extras())

	// Another user cannot reach the row.
	st, _, _ = e.do(t, http.MethodPost, "/api/v1/pregnancy/alerts/"+itoa(inboxID)+"/read", otherTok, "")
	assert.Equal(t, http.StatusNotFound, st)
	assert.Equal(t, "active", e.state(t, inboxID))

	// read = acknowledged.
	st, _, raw = e.do(t, http.MethodPost, "/api/v1/pregnancy/alerts/"+itoa(inboxID)+"/read", tok, "")
	require.Equal(t, http.StatusOK, st, raw)
	assert.Equal(t, "acknowledged", e.state(t, inboxID))
	assert.EqualValues(t, open-1, extras())
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/summary", tok, "")
	assert.Equal(t, false, data["has_emergency"])
	assert.Equal(t, true, data["has_unread"], "the legacy warning is unread")
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/"+itoa(inboxID), tok, "")
	shown := data["alert"].(map[string]any)
	assert.Equal(t, true, shown["is_read"])
	assert.NotNil(t, shown["read_at"])

	// read-all: the legacy row and the other open inbox rows.
	_, data, _ = e.do(t, http.MethodPost, "/api/v1/pregnancy/alerts/read-all", tok, "")
	assert.EqualValues(t, 1+open-1, data["count"])
	assert.EqualValues(t, 0, extras())
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/summary", tok, "")
	assert.Equal(t, false, data["has_unread"])

	// dismiss = archived (and acknowledged): gone from the list, still shown by id.
	st, _, raw = e.do(t, http.MethodPost, "/api/v1/pregnancy/alerts/"+itoa(inboxID)+"/dismiss", tok, "")
	require.Equal(t, http.StatusOK, st, raw)
	assert.Equal(t, "archived", e.state(t, inboxID))
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts", tok, "")
	require.Len(t, alertList(t, data), open)
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/"+itoa(inboxID), tok, "")
	assert.Equal(t, true, data["alert"].(map[string]any)["is_dismissed"])

	assert.Equal(t, 2, legacyCount(), "no new pregnancy_alerts row")
}

// read-all acknowledges every active preg.* row; an emergency read again by a new hit is unread again (M-16).
func TestV1Alerts_ReadAllAcknowledgesInboxRows(t *testing.T) {
	e := setupInboxAlerts(t)
	uid, tok := e.user(t, "09120000803")
	st, _, raw := e.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok,
		`{"log_date":"2026-09-23","has_fluid_leakage":true,"fluid_leakage_severity":"moderate"}`)
	require.Equal(t, http.StatusCreated, st, raw)
	var active int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM message_inbox WHERE user_id = ? AND rule_key LIKE 'preg.%' AND state = 'active'`, uid).Scan(&active))
	require.GreaterOrEqual(t, active, 1)
	_, data, _ := e.do(t, http.MethodPost, "/api/v1/pregnancy/alerts/read-all", tok, "")
	assert.EqualValues(t, active, data["count"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM message_inbox WHERE user_id = ? AND rule_key LIKE 'preg.%' AND state = 'active'`, uid).Scan(&n))
	assert.Zero(t, n)
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/summary", tok, "")
	assert.Equal(t, false, data["has_emergency"])

	// A new critical symptom (new evidence) makes the urgent row active again: the emergency is unread again.
	st, _, raw = e.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok,
		`{"log_date":"2026-09-23","has_fluid_leakage":true,"fluid_leakage_severity":"moderate","has_severe_sudden_pain":true,"severe_sudden_pain_severity":"severe"}`)
	require.Equal(t, http.StatusCreated, st, raw)
	_, data, _ = e.do(t, http.MethodGet, "/api/v1/pregnancy/alerts/summary", tok, "")
	assert.Equal(t, true, data["has_emergency"])
}

func itoa(n uint64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
