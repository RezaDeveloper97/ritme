package consent_test

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
	"github.com/ritme/backend-go/internal/consent"
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
	svc *consent.Service
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
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet)
	svc := consent.NewService(db)
	h := consent.NewHandlers(svc, clock.Real{})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/consents/pending", guard.RequireUser, h.Pending)
	app.Get("/api/v1/consents/history", guard.RequireUser, h.History)
	app.Get("/api/v1/consents/:code", guard.RequireUser, h.Show)
	app.Put("/api/v1/consents/:code", guard.RequireUser, h.Update)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (e *env) do(t *testing.T, method, path, token, body, now string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if now != "" {
		req.Header.Set("X-Test-Now", now)
	}
	res, err := e.app.Test(req)
	require.NoError(t, err)
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: res.StatusCode, body: m, raw: string(raw)}
}

func (e *env) events(t *testing.T, userID uint64) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT CONCAT(consent, ':', version, ':', action, ':', source) FROM consent_events WHERE user_id = ? ORDER BY id`, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

// T-M10-23: grant → revoke → grant is three events and the projection ends granted at the latest version; a repeat
// of the same grant adds nothing; an outdated grant still answers 403 outdated (Require unchanged).
func TestLedger_GrantRevokeGrant(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, tok := e.user(t, "09120000101")

	r := e.do(t, http.MethodPut, "/api/v1/consents/ai_assistant", tok, `{"granted":true,"version":1}`, "2026-10-22T10:00:00+03:30")
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/consents/ai_assistant", tok, `{"granted":true,"version":1}`, "2026-10-22T10:00:30+03:30")
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/consents/ai_assistant", tok, `{"granted":false}`, "2026-10-22T10:01:00+03:30")
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/consents/ai_assistant", tok, `{"granted":true,"version":1}`, "2026-10-22T10:02:00+03:30")
	require.Equal(t, 200, r.status, r.raw)

	assert.Equal(t, []string{"ai_assistant:1:grant:feature", "ai_assistant:1:revoke:feature", "ai_assistant:1:grant:feature"}, e.events(t, uid))
	require.NoError(t, e.svc.Require(ctx, uid, consent.AIAssistant))
	var version int
	var granted bool
	require.NoError(t, e.db.QueryRow(`SELECT granted, version FROM user_consents WHERE user_id = ? AND consent = 'ai_assistant'`, uid).Scan(&granted, &version))
	assert.True(t, granted)
	assert.Equal(t, 1, version)

	// A grant of an older text (before versioning) is outdated: 403 reason outdated, unchanged by the ledger.
	_, err := e.db.Exec(`INSERT INTO user_consents (user_id, consent, granted, version, granted_at, created_at, updated_at) VALUES (?, 'ai_lab_analysis', 1, NULL, '2026-09-01 10:00:00', '2026-09-01 10:00:00', '2026-09-01 10:00:00')`, uid)
	require.NoError(t, err)
	var req *consent.RequiredError
	require.ErrorAs(t, e.svc.Require(ctx, uid, consent.AILabAnalysis), &req)
	assert.Equal(t, consent.ReasonOutdated, req.Reason)
	var fail *httpx.FailError
	require.ErrorAs(t, consent.Required(req, "en"), &fail)
	assert.Equal(t, 403, fail.Status)

	// A stale version writes neither the projection nor an event.
	r = e.do(t, http.MethodPut, "/api/v1/consents/ai_voice_log", tok, `{"granted":true,"version":9}`, "")
	require.Equal(t, 409, r.status, r.raw)
	assert.Len(t, e.events(t, uid), 3)
}

// The ledger refuses UPDATE and DELETE; deleting the user removes her events (an FK cascade fires no trigger).
func TestLedger_AppendOnlyAndCascade(t *testing.T) {
	e := setup(t)
	uid, _ := e.user(t, "09120000102")
	require.NoError(t, e.svc.AcceptFrom(context.Background(), uid, consent.Terms, 1, consent.SourceSignup, time.Now()))

	_, err := e.db.Exec(`UPDATE consent_events SET action = 'revoke' WHERE user_id = ?`, uid)
	require.ErrorContains(t, err, "append-only")
	_, err = e.db.Exec(`DELETE FROM consent_events WHERE user_id = ?`, uid)
	require.ErrorContains(t, err, "append-only")
	assert.Equal(t, []string{"terms:1:grant:signup"}, e.events(t, uid))

	_, err = e.db.Exec(`DELETE FROM users WHERE id = ?`, uid)
	require.NoError(t, err, "account deletion still works")
	assert.Empty(t, e.events(t, uid))
}

// An existing user without the signup consents gets them as pending (each with its text), accepts them from the
// sheet (PUT /consents/{code}, source settings) and the list empties; history lists only her own events, newest first.
func TestLedger_PendingAndHistory(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000103")
	other, otherTok := e.user(t, "09120000104")
	require.NoError(t, e.svc.AcceptFrom(context.Background(), other, consent.Terms, 1, consent.SourceSignup, time.Now()))

	r := e.do(t, http.MethodGet, "/api/v1/consents/pending", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	list := r.body["data"].(map[string]any)["consents"].([]any)
	require.Len(t, list, 2)
	first := list[0].(map[string]any)
	assert.Equal(t, "terms", first["code"])
	assert.InDelta(t, 1, first["version"], 0)
	assert.Equal(t, "missing", first["reason"])
	assert.NotEmpty(t, first["title"])
	assert.NotEmpty(t, first["points"])
	assert.Equal(t, "health_data_processing", list[1].(map[string]any)["code"])

	for _, code := range []string{"terms", "health_data_processing"} {
		r = e.do(t, http.MethodPut, "/api/v1/consents/"+code, tok, `{"granted":true,"version":1}`, "2026-10-22T11:00:00+03:30")
		require.Equal(t, 200, r.status, r.raw)
	}
	r = e.do(t, http.MethodGet, "/api/v1/consents/pending", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Empty(t, r.body["data"].(map[string]any)["consents"])

	r = e.do(t, http.MethodGet, "/api/v1/consents/history", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	events := r.body["data"].(map[string]any)["events"].([]any)
	require.Len(t, events, 2, "only her own")
	newest := events[0].(map[string]any)
	assert.Equal(t, "health_data_processing", newest["consent"])
	assert.Equal(t, "grant", newest["action"])
	assert.Equal(t, "settings", newest["source"])
	assert.Equal(t, "2026-10-22T11:00:00+03:30", newest["created_at"])
	assert.Nil(t, newest["scope"])
	assert.NotEmpty(t, newest["title"])

	r = e.do(t, http.MethodGet, "/api/v1/consents/history", otherTok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Len(t, r.body["data"].(map[string]any)["events"], 1)

	r = e.do(t, http.MethodGet, "/api/v1/consents/history", "", "", "")
	assert.Equal(t, 401, r.status, "unauthenticated is the only 401")
}

// Scoped evidence: AppendScoped writes the scope as JSON with the evidence version; an unknown code or source is
// refused; History returns the scope.
func TestLedger_ScopedEvidence(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000105")
	ctx := context.Background()
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.Local)
	require.NoError(t, consent.AppendScoped(ctx, e.db, consent.Scoped{UserID: uid, Consent: consent.ShareLink,
		Action: consent.ActionGrant, Source: consent.SourceShareLink, Scope: map[string]any{"link_id": 7}, At: now}))
	require.ErrorIs(t, consent.AppendScoped(ctx, e.db, consent.Scoped{UserID: uid, Consent: consent.Terms,
		Action: consent.ActionGrant, Source: consent.SourceSettings, At: now}), consent.ErrInvalidEvent, "catalog codes go through Accept")
	require.ErrorIs(t, consent.AppendScoped(ctx, e.db, consent.Scoped{UserID: uid, Consent: consent.ShareLink,
		Action: "maybe", Source: consent.SourceShareLink, At: now}), consent.ErrInvalidEvent)

	r := e.do(t, http.MethodGet, "/api/v1/consents/history", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	ev := r.body["data"].(map[string]any)["events"].([]any)[0].(map[string]any)
	assert.Equal(t, "share_link", ev["consent"])
	assert.Equal(t, map[string]any{"link_id": float64(7)}, ev["scope"])
	assert.NotEmpty(t, ev["title"])
}
