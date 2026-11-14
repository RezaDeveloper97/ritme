package access_test

// B-N6-05 end to end against MariaDB with the fake provider: versioned consent endpoints, the B-N1-12 toggles
// granting the version in force, the AI gate (401, consent_required → accept → runs, ExternalOnly, daily cost
// cap fail-closed 503, the AI credit gate), the per-call usage log (no content, user scoped) and PII redaction on the way
// to the provider.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee6"
	testNow  = "2026-10-03T10:00:00+03:30"
)

// spyChat records what reached the provider; with block set, every call waits for it to close first.
type spyChat struct {
	*ai.Fake
	mu      sync.Mutex
	got     []ai.ChatRequest
	block   chan struct{}
	started chan struct{}
}

func (s *spyChat) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	s.mu.Lock()
	s.got = append(s.got, req)
	s.mu.Unlock()
	if s.block != nil {
		s.started <- struct{}{}
		<-s.block
	}
	return s.Fake.ChatStream(ctx, req)
}

func (s *spyChat) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.got)
}

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	spy     *spyChat
	credits *credits.Service
}

func setup(t *testing.T, provider string, capUSD float64) *env {
	t.Helper()
	return setupWith(t, provider, capUSD, capUSD, 0)
}

// testLimiter is a ratelimit.Limiter on the test Redis under a unique prefix (nil without TEST_REDIS_ADDR).
func testLimiter(t *testing.T) *ratelimit.Limiter {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		return nil
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), "ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })
	return ratelimit.New(c, clock.Real{})
}

func setupWith(t *testing.T, provider string, capUSD, userCapUSD float64, maxConcurrent int) *env {
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

	creditSvc := credits.NewService(db)
	budget := usage.NewBudget(db, capUSD, clock.Real{}, quiet).WithUserCap(userCapUSD)
	spy := &spyChat{Fake: ai.NewFake()}
	client := ai.NewClientWith(ai.Options{Provider: provider, Chatter: spy, Extractor: ai.NewFake(),
		Recorder: usage.NewRecorder(db, clock.Real{}, quiet), Limiter: budget,
		Pricer: ai.NewPricer(map[string]config.AIPrice{ai.FakeChatModel: {InputPerMTok: 1, OutputPerMTok: 1, AudioPerMTok: 1}})})
	consents := consent.NewService(db)
	g := access.NewGuard(access.Options{Client: client, Consents: consents, Budget: budget, Credits: creditSvc, Catalog: credits.NewCatalog(db),
		Limiter: testLimiter(t), Logger: quiet, MaxConcurrent: maxConcurrent})

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
			st, err := client.Chat(c.Context(), feature, ai.ChatRequest{
				Messages: []ai.ChatMessage{{Role: ai.RoleUser, Text: string(c.Body())}}, Language: i18n.Locale(c)})
			if err != nil {
				return access.Error(err, i18n.Locale(c))
			}
			defer st.Close()
			var sb strings.Builder
			for e := range st.Events {
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
	// A handler that answers an error status without returning an error (an invalid upload, …).
	app.Post("/test/assistant-invalid", locale, append(append([]any{guard}, g.Chain(ai.FeatureAssistant)...), func(c fiber.Ctx) error {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"success": false})
	})...)
	// Gate: the subscription is required, nothing is spent (assistant summaries).
	app.Post("/test/gate", locale, append(append([]any{guard}, g.Gate(ai.FeatureAssistant)...), chat(ai.FeatureAssistant))...)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), spy: spy, credits: creditSvc}
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
	// GET /consents leaves out the mandatory signup consents (T-M10-23); /consents/pending lists those.
	require.Len(t, list, len(consent.Catalog())-len(consent.Mandatory()))
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

	// B-N6-05b (M5): an AI grant through the toggles needs the version the client showed
	r = e.do(t, "PUT", "/api/v1/profile/consents", tok, "fa", `{"consents":{"ai_lab_analysis":true}}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.raw, "versions.ai_lab_analysis")
	r = e.do(t, "PUT", "/api/v1/profile/consents", tok, "fa", `{"consents":{"ai_lab_analysis":true},"versions":{"ai_lab_analysis":2}}`)
	require.Equal(t, 409, r.status, r.raw)
	assert.Equal(t, "consent_version_stale", r.body["error_code"])
	r = e.do(t, "PUT", "/api/v1/profile/consents", tok, "fa", `{"consents":{"ai_lab_analysis":true},"versions":{"ai_lab_analysis":1}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_consents WHERE user_id = ? AND consent = 'ai_lab_analysis' AND version = 1", uid))
	r = e.do(t, "POST", "/test/lab", tok, "fa", "hi")
	assert.Equal(t, 200, r.status, r.raw)
}

func TestGate_ConsentCreditsAndUsageLog(t *testing.T) {
	e := setup(t, "fake", 5)
	uid, tok := e.user(t, "09120000004", "سارا احمدی", true)

	r := e.do(t, "POST", "/test/assistant", "", "fa", "hi")
	assert.Equal(t, 401, r.status)

	r = e.do(t, "POST", "/test/assistant", tok, "fa", "hi")
	require.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "consent_required", r.body["error_code"])
	assert.Equal(t, "ai_assistant", r.body["consent"])
	assert.InDelta(t, 1, r.body["version"], 0)
	assert.Equal(t, "missing", r.body["reason"])
	assert.Zero(t, e.spy.calls(), "no provider call without consent")

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

	b := e.balance(t, uid)
	assert.Equal(t, uint32(20), b.PlanTotal, "the trial carries trial_ai_credits")
	assert.Equal(t, uint32(19), b.PlanRemaining, "one assistant use costs 1 credit")

	// Without a subscription or trial no AI feature runs: 402 before the consent question.
	_, freeTok := e.user(t, "09120000040", "Mina", false)
	r = e.do(t, "POST", "/test/lab", freeTok, "fa", "hi")
	assert.Equal(t, 402, r.status, r.raw)
	assert.Equal(t, "ai_subscription_required", r.body["error_code"])
	assert.Equal(t, "lab_analysis", r.body["feature"])

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
	_, tok := e.user(t, "09120000007", "Sara", true)
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
	calls := e.spy.calls()
	r := e.do(t, "POST", "/test/assistant", tok, "en", "hi")
	assert.Equal(t, 503, r.status, r.raw)
	assert.Equal(t, "ai_budget_exhausted", r.body["error_code"])
	assert.Equal(t, calls, e.spy.calls(), "no provider call once the cap is spent")

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

func (e *env) balance(t *testing.T, userID uint64) credits.Balance {
	t.Helper()
	b, err := e.credits.Balance(context.Background(), userID, mustNow())
	require.NoError(t, err)
	return b
}

func mustNow() time.Time {
	n, err := time.Parse(time.RFC3339, testNow)
	if err != nil {
		panic(err)
	}
	return n.In(civildate.Tehran)
}

// setTrialCredits sets the trial's credits (plus_settings) before the trial's bucket is created.
func (e *env) setTrialCredits(t *testing.T, n int) {
	t.Helper()
	_, err := e.db.Exec("UPDATE plus_settings SET value = ? WHERE `key` = 'trial_ai_credits'", n)
	require.NoError(t, err)
}

// B-N6-05b (M3) + goose 00125: the credits are reserved before the provider runs — parallel requests never
// overspend — and refunded only when the call failed.
func TestGate_ReserveFirstCredits(t *testing.T) {
	e := setupWith(t, "fake", 5, 5, 100) // no concurrency limit: the credits alone must hold
	e.setTrialCredits(t, 5)
	uid, tok := e.user(t, "09120000009", "Sara", true)
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "fa", `{"granted":true,"version":1}`).status)

	r := e.do(t, "POST", "/test/assistant", tok, "fa", "RITME-FAKE:error")
	require.Equal(t, 503, r.status, r.raw)
	assert.Equal(t, uint32(5), e.balance(t, uid).PlanRemaining, "a provider failure gives the credit back")
	r = e.do(t, "POST", "/test/assistant-invalid", tok, "fa", "hi")
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, uint32(5), e.balance(t, uid).PlanRemaining, "an error status gives the credit back")
	assert.Equal(t, 2, e.count(t, "SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND status = 'refunded'", uid))

	var wg sync.WaitGroup
	results := make(chan resp, 8)
	for range 7 { // 2 + 7 stays under the assistant burst throttle (10/min)
		wg.Go(func() { results <- e.do(t, "POST", "/test/assistant", tok, "fa", "hi") })
	}
	wg.Wait()
	close(results)
	ok, refused := 0, 0
	for res := range results {
		switch res.status {
		case 200:
			ok++
		case 402:
			refused++
			assert.Equal(t, "ai_credits_exhausted", res.body["error_code"], res.raw)
			assert.InDelta(t, 1, res.body["required"], 0)
			assert.InDelta(t, 0, res.body["balance"], 0)
			assert.Equal(t, "2026-10-08T10:00:00+03:30", res.body["period_ends_at"])
		default:
			t.Errorf("unexpected status %d: %s", res.status, res.raw)
		}
	}
	assert.Equal(t, 5, ok, "never more than the credits, however parallel")
	assert.Equal(t, 2, refused)
	assert.Zero(t, e.balance(t, uid).PlanRemaining)
	assert.Equal(t, 5, e.count(t, "SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND status = 'spent'", uid))
	assert.Equal(t, 5+1, e.spy.calls(), "refused requests never reach the provider")
}

// goose 00125: the three credit refusals of the gate, and Gate (no cost) still needs the subscription.
func TestGate_CreditRefusals(t *testing.T) {
	e := setup(t, "fake", 5)
	uid, tok := e.user(t, "09120000041", "Sara", false)
	_, err := e.credits.AdminGrant(context.Background(), uid, 50, "gift", mustNow())
	require.NoError(t, err)

	// Extra credits without a subscription: locked, reported, nothing spent.
	for _, p := range []string{"/test/assistant", "/test/gate"} {
		r := e.do(t, "POST", p, tok, "en", "hi")
		require.Equal(t, 402, r.status, p+" "+r.raw)
		assert.Equal(t, "ai_subscription_required", r.body["error_code"])
		assert.Equal(t, "assistant", r.body["feature"])
		assert.InDelta(t, 50, r.body["extra_credits"], 0)
	}
	assert.Equal(t, uint32(50), e.balance(t, uid).Extra)
	assert.Zero(t, e.spy.calls())

	// With a running trial the extra credits are spendable; Gate spends nothing.
	_, err = e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
		VALUES (?, '2026-10-01 10:00:00', '2026-10-08 10:00:00', NOW(), NOW())`, uid)
	require.NoError(t, err)
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", tok, "en", `{"granted":true,"version":1}`).status)
	r := e.do(t, "POST", "/test/gate", tok, "en", "hi")
	require.Equal(t, 200, r.status, r.raw)
	b := e.balance(t, uid)
	assert.Equal(t, uint32(20), b.PlanRemaining, "Gate spends nothing")
	assert.Equal(t, uint32(50), b.Extra)
	assert.Zero(t, e.count(t, "SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ?", uid))

	// Not enough credits for one use: 402 ai_credits_exhausted with the balance.
	_, err = e.db.Exec("UPDATE ai_features SET credit_cost = 100 WHERE `key` = 'lab_analysis'")
	require.NoError(t, err)
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_lab_analysis", tok, "en", `{"granted":true,"version":1}`).status)
	r = e.do(t, "POST", "/test/lab", tok, "en", "hi")
	require.Equal(t, 402, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"`+access.T(access.CodeCreditsExhausted, "en")+`","error_code":"ai_credits_exhausted",
		"feature":"lab_analysis","required":100,"balance":70,"plan_remaining":20,"extra_credits":50,
		"period_ends_at":"2026-10-08T10:00:00+03:30"}`, r.raw)

	// A switched-off slot: 403 ai_feature_disabled, before the subscription and the consent.
	_, err = e.db.Exec("UPDATE ai_features SET is_active = 0 WHERE `key` = 'assistant'")
	require.NoError(t, err)
	_, freeTok := e.user(t, "09120000042", "Mina", false)
	for _, k := range []string{tok, freeTok} {
		r = e.do(t, "POST", "/test/assistant", k, "en", "hi")
		require.Equal(t, 403, r.status, r.raw)
		assert.JSONEq(t, `{"success":false,"message":"`+access.T(access.CodeFeatureDisabled, "en")+`","error_code":"ai_feature_disabled","feature":"assistant"}`, r.raw)
	}
	assert.Equal(t, 1, e.spy.calls(), "only the Gate call reached the provider, no refusal did")
	assert.Equal(t, 70, int(e.balance(t, uid).Spendable()))
}

// B-N6-05b (M3): one user may run at most MaxConcurrent AI requests at once.
func TestGate_ConcurrencyPerUser(t *testing.T) {
	e := setupWith(t, "fake", 5, 5, 1)
	e.spy.block, e.spy.started = make(chan struct{}), make(chan struct{}, 4)
	_, tok := e.user(t, "09120000010", "Sara", true)
	_, tok2 := e.user(t, "09120000011", "Mina", true)
	for _, k := range []string{tok, tok2} {
		require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", k, "fa", `{"granted":true,"version":1}`).status)
	}
	first := make(chan resp, 1)
	go func() { first <- e.do(t, "POST", "/test/assistant", tok, "fa", "hi") }()
	<-e.spy.started // the first request is inside the provider
	r := e.do(t, "POST", "/test/assistant", tok, "fa", "hi")
	assert.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, "ai_busy", r.body["error_code"])
	other := make(chan resp, 1)
	go func() { other <- e.do(t, "POST", "/test/assistant", tok2, "fa", "hi") }()
	<-e.spy.started // another user is not blocked
	close(e.spy.block)
	assert.Equal(t, 200, (<-first).status)
	assert.Equal(t, 200, (<-other).status)
	assert.Equal(t, 200, e.do(t, "POST", "/test/assistant", tok, "fa", "hi").status, "the slot is released")
}

// B-N6-05b (M2): the per-user daily cost cap answers 429 for that user only.
func TestGate_UserDailyCostCap(t *testing.T) {
	e := setupWith(t, "gemini", 5, 0.01, 0) // 10 000 micro-USD per user
	uid, tok := e.user(t, "09120000012", "Sara", true)
	_, tok2 := e.user(t, "09120000013", "Mina", true)
	for _, k := range []string{tok, tok2} {
		require.Equal(t, 200, e.do(t, "PUT", "/api/v1/consents/ai_assistant", k, "en", `{"granted":true,"version":1}`).status)
	}
	require.Equal(t, 200, e.do(t, "POST", "/test/assistant", tok, "en", "hi").status)
	_, err := e.db.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, cost_micros, ok, created_at)
		VALUES (?, 'assistant', 'chat', 'gemini', 'm', 10000, 1, NOW())`, uid)
	require.NoError(t, err)
	calls := e.spy.calls()
	r := e.do(t, "POST", "/test/assistant", tok, "en", "hi")
	assert.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, "ai_user_budget_exhausted", r.body["error_code"])
	assert.Equal(t, calls, e.spy.calls())
	assert.Equal(t, 200, e.do(t, "POST", "/test/assistant", tok2, "en", "hi").status, "other users keep going")
}

// B-N6-05b (L2 / M2): the feature throttles run right after auth — before the Plus check and the consent.
func TestGate_ThrottleBeforeGate(t *testing.T) {
	if os.Getenv("TEST_REDIS_ADDR") == "" {
		t.Skip("TEST_REDIS_ADDR not set (run `make test-int`)")
	}
	e := setup(t, "fake", 5)
	_, tok := e.user(t, "09120000014", "Sara", false)
	for range 4 { // ai-lab-burst
		r := e.do(t, "POST", "/test/lab", tok, "fa", "hi")
		require.Equal(t, 402, r.status, r.raw)
	}
	r := e.do(t, "POST", "/test/lab", tok, "fa", "hi")
	assert.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, "too_many_requests", r.body["error_code"])
}
