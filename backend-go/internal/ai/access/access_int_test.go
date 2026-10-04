package access_test

// B-N6-05 end to end against MariaDB with the fake provider: versioned consent endpoints, the B-N1-12 toggles
// granting the version in force, the AI gate (401, consent_required → accept → runs, ExternalOnly, daily cost
// cap fail-closed 503, Plus gate), the per-call usage log (no content, user scoped) and PII redaction on the way
// to the provider.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee6"
	testNow  = "2026-10-03T10:00:00+03:30"
)

// spyChat records what reached the provider.
type spyChat struct {
	*ai.Fake
	got []ai.ChatRequest
}

func (s *spyChat) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	s.got = append(s.got, req)
	return s.Fake.ChatStream(ctx, req)
}

type env struct {
	db   *sql.DB
	app  *fiber.App
	iss  *passport.Issuer
	spy  *spyChat
	plus *plus.Service
}

func setup(t *testing.T, provider string, capUSD float64) *env {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))

	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})
	budget := usage.NewBudget(db, capUSD, clock.Real{}, quiet)
	spy := &spyChat{Fake: ai.NewFake()}
	client := ai.NewClientWith(ai.Options{Provider: provider, Chatter: spy, Extractor: ai.NewFake(),
		Recorder: usage.NewRecorder(db, clock.Real{}, quiet), Limiter: budget,
		Pricer: ai.NewPricer(map[string]config.AIPrice{ai.FakeChatModel: {InputPerMTok: 1, OutputPerMTok: 1, AudioPerMTok: 1}})})
	consents := consent.NewService(db)
	g := access.NewGuard(access.Options{Client: client, Consents: consents, Budget: budget, Gate: gate})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	ch := consent.NewHandlers(consents, clock.Real{})
	app.Get("/api/v1/consents", locale, guard, ch.List)
	app.Get("/api/v1/consents/:code", locale, guard, ch.Show)
	app.Put("/api/v1/consents/:code", locale, guard, ch.Update)
	ph := profile.NewPrivacyHandlers(profile.PrivacyOptions{Store: profilestore.New(db), Logger: quiet})
	app.Put("/api/v1/profile/consents", locale, guard, ph.UpdateConsents)

	// A minimal feature route: the gate, then a chat through the client with the request context.
	chat := func(feature ai.Feature) fiber.Handler {
		return func(c fiber.Ctx) error {
			ev, err := client.Chat(c.Context(), feature, ai.ChatRequest{
				Messages: []ai.ChatMessage{{Role: ai.RoleUser, Text: string(c.Body())}}, Language: i18n.Locale(c)})
			if err != nil {
				return access.Error(err, i18n.Locale(c))
			}
			var sb strings.Builder
			for e := range ev {
				if e.Err != nil {
					return access.Error(e.Err, i18n.Locale(c))
				}
				sb.WriteString(e.Delta)
			}
			return httpx.OK(c, map[string]any{"text": sb.String()})
		}
	}
	app.Post("/test/assistant", locale, append(append([]any{guard}, g.Chain(ai.FeatureAssistant)...), chat(ai.FeatureAssistant))...)
	app.Post("/test/voice", locale, append(append([]any{guard}, g.Chain(ai.FeatureVoiceLog)...), chat(ai.FeatureVoiceLog))...)
	app.Post("/test/lab", locale, append(append([]any{guard}, g.Chain(ai.FeatureLabAnalysis)...), chat(ai.FeatureLabAnalysis))...)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), spy: spy, plus: plusSvc}
}

func (e *env) user(t *testing.T, mobile, name string, trial bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, name, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if trial {
		_, err = e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
			VALUES (?, '2026-10-01 10:00:00', '2026-10-08 10:00:00', NOW(), NOW())`, id)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // positive id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // positive id
}

type resp struct {
	status int
	body   map[string]any
	raw    string
}

func (r resp) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func (e *env) do(t *testing.T, method, path, token, lang, body string) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if strings.HasPrefix(body, "{") {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "text/plain")
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, testNow)
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp{status: res.StatusCode, body: m, raw: string(raw)}
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func TestConsentEndpoints(t *testing.T) {
	e := setup(t, "fake", 5)
	uid, tok := e.user(t, "09120000001", "Sara Ahmadi", false)
	_, other := e.user(t, "09120000002", "Mina", false)

	assert.Equal(t, 401, e.do(t, "GET", "/api/v1/consents", "", "fa", "").status)
	assert.Equal(t, 401, e.do(t, "PUT", "/api/v1/consents/ai_assistant", "", "fa", `{"granted":true,"version":1}`).status)

	r := e.do(t, "GET", "/api/v1/consents", tok, "fa", "")
	require.Equal(t, 200, r.status, r.raw)
	list := r.data()["consents"].([]any)
	require.Len(t, list, len(consent.Catalog()))
	first := list[0].(map[string]any)
	assert.Equal(t, "ai_lab_analysis", first["code"])
	assert.InDelta(t, 1, first["version"], 0)
	assert.Equal(t, "تحلیل آزمایش با هوش مصنوعی", first["title"])
	assert.NotEmpty(t, first["points"])
	assert.Equal(t, false, first["granted"])
	assert.Equal(t, true, first["needs_consent"])
	assert.Equal(t, "missing", first["reason"])

	r = e.do(t, "GET", "/api/v1/consents/nope", tok, "en", "")
	assert.Equal(t, 404, r.status)
	assert.Equal(t, "consent_not_found", r.body["error_code"])

	r = e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":true}`)
	assert.Equal(t, 422, r.status, "granting needs the version shown")
	assert.Contains(t, r.raw, "version")
	r = e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":"maybe"}`)
	assert.Equal(t, 422, r.status)
	r = e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":true,"version":2}`)
	assert.Equal(t, 409, r.status)
	assert.Equal(t, "consent_version_stale", r.body["error_code"])
	assert.InDelta(t, 1, r.body["version"], 0)

	r = e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":true,"version":1}`)
	require.Equal(t, 200, r.status, r.raw)
	c := r.data()["consent"].(map[string]any)
	assert.Equal(t, true, c["granted"])
	assert.InDelta(t, 1, c["accepted_version"], 0)
	assert.Equal(t, "2026-10-03T10:00:00+03:30", c["granted_at"])
	assert.Nil(t, c["reason"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_consents WHERE user_id = ? AND consent = 'ai_assistant' AND granted = 1 AND version = 1", uid))

	// other users are untouched (IDOR: user_id comes from the token only)
	r = e.do(t, "GET", "/api/v1/consents/ai_assistant", other, "en", "")
	assert.Equal(t, false, r.data()["consent"].(map[string]any)["granted"])

	r = e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":false}`)
	require.Equal(t, 200, r.status, r.raw)
	c = r.data()["consent"].(map[string]any)
	assert.Equal(t, false, c["granted"])
	assert.InDelta(t, 1, c["accepted_version"], 0, "what was last accepted is kept")
	assert.NotNil(t, c["revoked_at"])
}

func TestProfileToggleGrantsVersionInForce(t *testing.T) {
	e := setup(t, "fake", 5)
	uid, tok := e.user(t, "09120000003", "Sara", true)
	// a B-N1-12 row granted before versions existed is outdated for the AI gate
	_, err := e.db.Exec(`INSERT INTO user_consents (user_id, consent, granted, granted_at, created_at, updated_at)
		VALUES (?, 'ai_lab_analysis', 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00', '2026-09-01 10:00:00')`, uid)
	require.NoError(t, err)
	r := e.do(t, "POST", "/test/lab", tok, "fa", "hi")
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "outdated", r.body["reason"])

	r = e.do(t, "PUT", "/api/v1/profile/consents", tok, "fa", `{"consents":{"ai_lab_analysis":true}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_consents WHERE user_id = ? AND consent = 'ai_lab_analysis' AND version = 1", uid))
	r = e.do(t, "POST", "/test/lab", tok, "fa", "hi")
	assert.Equal(t, 200, r.status, r.raw)
}

func TestGate_ConsentPlusAndUsageLog(t *testing.T) {
	e := setup(t, "fake", 5)
	uid, tok := e.user(t, "09120000004", "سارا احمدی", false)

	r := e.do(t, "POST", "/test/assistant", "", "fa", "hi")
	assert.Equal(t, 401, r.status)

	r = e.do(t, "POST", "/test/assistant", tok, "fa", "hi")
	require.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "consent_required", r.body["error_code"])
	assert.Equal(t, "ai_assistant", r.body["consent"])
	assert.InDelta(t, 1, r.body["version"], 0)
	assert.Equal(t, "missing", r.body["reason"])
	assert.Empty(t, e.spy.got, "no provider call without consent")

	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "fa", `{"granted":true,"version":1}`).status)
	r = e.do(t, "POST", "/test/assistant", tok, "fa", "من سارا هستم، شماره\u200cام 09121234567 است و دلم درد می\u200cکند")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, ai.FakeChatReplies["default"]["fa"], r.data()["text"])
	require.Len(t, e.spy.got, 1)
	assert.Equal(t, "من [name] هستم، شماره\u200cام [phone] است و دلم درد می\u200cکند", e.spy.got[0].Messages[0].Text)

	var feature, op, provider, model string
	var userID sql.NullInt64
	var in, out, cost int64
	var ok bool
	require.NoError(t, e.db.QueryRow(`SELECT user_id, feature, op, provider, model, input_tokens, output_tokens, cost_micros, ok
		FROM ai_usage_logs`).Scan(&userID, &feature, &op, &provider, &model, &in, &out, &cost, &ok))
	assert.Equal(t, int64(uid), userID.Int64) //nolint:gosec // test id
	assert.Equal(t, "assistant", feature)
	assert.Equal(t, "chat", op)
	assert.Equal(t, "fake", provider)
	assert.True(t, ok)
	assert.Positive(t, in)
	assert.Zero(t, cost, "the fake is free")

	// Lab analysis is Plus-only: a free user gets 402 before the consent question.
	r = e.do(t, "POST", "/test/lab", tok, "fa", "hi")
	assert.Equal(t, 402, r.status, r.raw)
	assert.Equal(t, "plus_required", r.body["error_code"])

	// Account deletion keeps the cost history without the user.
	_, err := e.db.Exec("DELETE FROM users WHERE id = ?", uid)
	require.NoError(t, err)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM ai_usage_logs WHERE user_id IS NULL"))
}

func TestGate_ExternalOnly(t *testing.T) {
	// The fake keeps voice logging in-process: no consent needed (the voice UI has no consent sheet yet).
	e := setup(t, "fake", 5)
	_, tok := e.user(t, "09120000005", "Sara", true)
	r := e.do(t, "POST", "/test/voice", tok, "fa", "hi")
	assert.Equal(t, 200, r.status, r.raw)

	// A real provider: the consent is required.
	ext := setup(t, "gemini", 5)
	_, tok = ext.user(t, "09120000006", "Sara", true)
	r = ext.do(t, "POST", "/test/voice", tok, "fa", "hi")
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "ai_voice_log", r.body["consent"])
	require.Equal(t, 200, ext.do(t, "PUT", "/api/v1/consents/ai_voice_log", tok, "fa", `{"granted":true,"version":1}`).status)
	assert.Equal(t, 200, ext.do(t, "POST", "/test/voice", tok, "fa", "hi").status)
}

func TestGate_DailyCostCapFailsClosed(t *testing.T) {
	e := setup(t, "gemini", 0.01) // 10 000 micro-USD
	_, tok := e.user(t, "09120000007", "Sara", false)
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":true,"version":1}`).status)
	require.Equal(t, 200, e.do(t, "POST", "/test/assistant", tok, "en", "hi").status)

	// Yesterday's spend does not count; today's does.
	_, err := e.db.Exec(`INSERT INTO ai_usage_logs (feature, op, provider, model, cost_micros, ok, created_at)
		VALUES ('assistant', 'chat', 'gemini', 'm', 50000, 1, ?)`, time.Now().Add(-48*time.Hour).Format("2006-01-02 15:04:05"))
	require.NoError(t, err)
	require.Equal(t, 200, e.do(t, "POST", "/test/assistant", tok, "en", "hi").status)
	_, err = e.db.Exec(`INSERT INTO ai_usage_logs (feature, op, provider, model, cost_micros, ok, created_at)
		VALUES ('lab_analysis', 'extract', 'gemini', 'm', 10000, 1, NOW())`)
	require.NoError(t, err)
	calls := len(e.spy.got)
	r := e.do(t, "POST", "/test/assistant", tok, "en", "hi")
	assert.Equal(t, 503, r.status, r.raw)
	assert.Equal(t, "ai_budget_exhausted", r.body["error_code"])
	assert.Len(t, e.spy.got, calls, "no provider call once the cap is spent")

	// cap 0 refuses everything
	z := setup(t, "fake", 0)
	_, tok = z.user(t, "09120000008", "Sara", true)
	assert.Equal(t, 503, z.do(t, "POST", "/test/voice", tok, "fa", "hi").status)
}

func TestUsageRecorder_NoContentColumns(t *testing.T) {
	e := setup(t, "fake", 5)
	rows, err := e.db.Query("SHOW COLUMNS FROM ai_usage_logs")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var cols []string
	for rows.Next() {
		var field, typ, null, key string
		var def, extra sql.NullString
		require.NoError(t, rows.Scan(&field, &typ, &null, &key, &def, &extra))
		cols = append(cols, field)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"id", "user_id", "feature", "op", "provider", "model", "input_tokens", "output_tokens",
		"audio_bytes", "image_bytes", "cost_micros", "latency_ms", "ok", "created_at"}, cols)
}
