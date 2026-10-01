package voicelog_test

// POST /api/v1/logs/voice end to end against MariaDB with the fake AI provider: Plus gate, quota counting
// only after success, provider errors, custom items per user, no health rows written, and the follow-up
// PUT /logs/days with voice_params storing source=voice.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/voicelog"
	"github.com/ritme/backend-go/resources/translations"
)

const (
	voiceClientID = "0199c0de-0000-7000-8000-00000c0ffee5"
	voiceNow      = "2026-09-23T10:00:00+03:30"
)

type voiceEnv struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

// countingTranscriber wraps the fake and counts provider calls.
type countingTranscriber struct {
	ai.Transcriber
	calls int
}

func (c *countingTranscriber) Transcribe(ctx context.Context, r ai.TranscribeRequest) (ai.Transcript, ai.Usage, error) {
	c.calls++
	return c.Transcriber.Transcribe(ctx, r)
}

func setupVoice(t *testing.T, client func(*countingTranscriber) *ai.Client) (*voiceEnv, *countingTranscriber) {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, voiceClientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)
	bundles := i18n.NewTranslationStore(translations.FS, "")
	logs := healthlog.NewService(db)
	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})
	spy := &countingTranscriber{Transcriber: ai.NewFake()}
	h := voicelog.NewHandlers(voicelog.NewService(client(spy), logs), plusSvc, gate, bundles, languages, clock.Real{})
	lh := healthlog.NewLogHandlers(logs, clock.Real{}, bundles, languages)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/logs/voice", locale, guard, gate.Require(plus.VoiceLog), h.Voice)
	app.Put("/api/v1/logs/days/:date", locale, guard, lh.Save)
	return &voiceEnv{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}, spy
}

func fakeClient(spy *countingTranscriber) *ai.Client {
	return ai.NewClient("fake", spy, ai.NewFake(), nil)
}

func (e *voiceEnv) user(t *testing.T, mobile string, trial bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if trial {
		_, err = e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
			VALUES (?, '2026-09-22 10:00:00', '2026-09-29 10:00:00', NOW(), NOW())`, id)
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

func (e *voiceEnv) do(t *testing.T, method, path, token, lang, ct string, body io.Reader) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, voiceNow)
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp{status: res.StatusCode, body: m, raw: string(raw)}
}

func (e *voiceEnv) voice(t *testing.T, token, lang, fixture string) resp {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	require.NoError(t, w.WriteField("duration_ms", "7000"))
	fw, err := w.CreateFormFile("audio", "voice.webm")
	require.NoError(t, err)
	data := []byte("\x1a\x45\xdf\xa3\x9f\x42\x86\x81\x01webm-opus ")
	if fixture != "" {
		data = append(data, []byte(ai.FakeMarker+fixture)...)
	}
	_, _ = fw.Write(data)
	require.NoError(t, w.Close())
	return e.do(t, "POST", "/api/v1/logs/voice", token, lang, w.FormDataContentType(), &b)
}

func (e *voiceEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func (e *voiceEnv) used(t *testing.T, userID uint64) int {
	t.Helper()
	var n sql.NullInt64
	err := e.db.QueryRow(`SELECT used FROM plus_usage_counters WHERE user_id = ? AND feature = 'plus.voice_log'`, userID).Scan(&n)
	if err == sql.ErrNoRows {
		return 0
	}
	require.NoError(t, err)
	return int(n.Int64)
}

func TestVoice_Unauthenticated(t *testing.T) {
	e, spy := setupVoice(t, fakeClient)
	r := e.voice(t, "", "fa", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
	assert.Zero(t, spy.calls)
}

func TestVoice_FreeUserLocked(t *testing.T) {
	e, spy := setupVoice(t, fakeClient)
	uid, tok := e.user(t, "09120000101", false)
	r := e.voice(t, tok, "fa", "")
	require.Equal(t, 402, r.status, r.raw)
	assert.Equal(t, "plus_required", r.body["error_code"])
	assert.Equal(t, "plus.voice_log", r.body["feature"])
	assert.Equal(t, "locked", r.body["reason"])
	assert.Zero(t, spy.calls, "nothing reaches the provider for a locked user")
	assert.Zero(t, e.used(t, uid))
}

func TestVoice_TrialUser(t *testing.T) {
	e, spy := setupVoice(t, fakeClient)
	uid, tok := e.user(t, "09120000102", true)
	r := e.voice(t, tok, "fa", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, ai.FakeTranscripts["default"]["fa"], r.data()["transcript"])
	assert.Equal(t, "cycle", r.data()["mode"])
	assert.JSONEq(t, `[
		{"category":"pain","param":"location","item":"abdomen","value":"moderate","confidence":0.9,"label":"درد شکم · متوسط"},
		{"category":"symptoms","param":"digestive","item":"bloating","value":"yes","confidence":0.9,"label":"نفخ"},
		{"category":"mood","param":"moods","item":"bored","value":true,"confidence":0.9,"label":"بی\u200cحوصله"}
	]`, mustJSON(t, r.data()["suggestions"]))
	assert.Equal(t, 1, spy.calls)
	assert.Equal(t, 1, e.used(t, uid), "one use counted after success")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ?`, uid), "never auto-saves")

	en := e.voice(t, tok, "en", "weight")
	require.Equal(t, 200, en.status, en.raw)
	assert.Contains(t, mustJSON(t, en.data()), `"label":"Weight · 58.5 kg"`)
	assert.Equal(t, 2, e.used(t, uid))

	// nothing said → no suggestions, no use counted
	s := e.voice(t, tok, "fa", "silence")
	require.Equal(t, 200, s.status, s.raw)
	assert.Equal(t, "", s.data()["transcript"])
	assert.Equal(t, []any{}, s.data()["suggestions"])
	assert.Equal(t, 2, e.used(t, uid))

	// provider failure → 503 ai_failed, nothing counted
	f := e.voice(t, tok, "fa", "error")
	require.Equal(t, 503, f.status, f.raw)
	assert.Equal(t, "ai_failed", f.body["error_code"])
	assert.Equal(t, false, f.body["success"])
	assert.Equal(t, 2, e.used(t, uid))
}

func TestVoice_Unavailable(t *testing.T) {
	e, _ := setupVoice(t, func(*countingTranscriber) *ai.Client { return nil })
	_, tok := e.user(t, "09120000103", true)
	r := e.voice(t, tok, "fa", "")
	require.Equal(t, 503, r.status, r.raw)
	assert.Equal(t, "ai_unavailable", r.body["error_code"])
}

func TestVoice_CustomItemsArePerUser(t *testing.T) {
	e, _ := setupVoice(t, fakeClient)
	a, tokA := e.user(t, "09120000104", true)
	_, tokB := e.user(t, "09120000105", true)
	res, err := e.db.Exec(`INSERT INTO health_log_custom_items (user_id, category, param, label, created_at, updated_at)
		VALUES (?, 'mood', 'moods', 'دلتنگ', NOW(), NOW())`, a)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	code := "custom_" + strconv.FormatInt(id, 10)

	ra := e.voice(t, tokA, "fa", "custom")
	require.Equal(t, 200, ra.status, ra.raw)
	assert.Contains(t, ra.raw, `"item":"`+code+`"`)
	assert.Contains(t, mustJSON(t, ra.data()), `"label":"دلتنگ"`)

	rb := e.voice(t, tokB, "fa", "custom")
	require.Equal(t, 200, rb.status, rb.raw)
	assert.NotContains(t, rb.raw, code, "B never gets A's custom item")
	assert.Contains(t, rb.raw, `"item":"fatigue"`)
}

func TestVoice_ValidationAndSave(t *testing.T) {
	e, spy := setupVoice(t, fakeClient)
	uid, tok := e.user(t, "09120000106", true)

	r := e.do(t, "POST", "/api/v1/logs/voice", tok, "en", "application/json", strings.NewReader(`{"audio":"x"}`))
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, "The audio file field is required.", r.body["message"])
	assert.Zero(t, spy.calls)
	assert.Zero(t, e.used(t, uid))

	// The client merges the suggestions and saves; voice_params marks them source=voice.
	put := e.do(t, "PUT", "/api/v1/logs/days/2026-09-23", tok, "fa", "application/json", strings.NewReader(
		`{"categories":{"pain":{"location":{"abdomen":"moderate"}},"sleep":{"quality":"good"},"note":{"text":"x"}},"voice_params":["pain.location","note.text","nope.x",3]}`))
	require.Equal(t, 200, put.status, put.raw)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND source = 'voice' AND category = 'pain'`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND source = 'manual' AND category = 'sleep'`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND source = 'manual' AND category = 'note'`, uid),
		"free text never carries source=voice")
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
