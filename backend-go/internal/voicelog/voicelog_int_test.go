package voicelog_test

// POST /api/v1/logs/voice end to end against MariaDB with the fake AI provider: Plus gate, quota counting
// only after success, provider errors, custom items per user, no health rows written, and the follow-up
// PUT /logs/days with voice_params storing source=voice. CB-VOICE-01: canvas suggestions per eligibility and
// POST /logs/voice/commit writing hot flashes, the pain diary, the pill and the bladder diary through their
// services, user-scoped.

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
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/pelvic"
	pelvicstore "github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
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
	db    *sql.DB
	app   *fiber.App
	iss   *passport.Issuer
	conds *conditions.Service
	contr *contraception.Service
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
	cat := catalog.NewReader(catalogstore.New(db), nil, 0, quiet)
	conds, contr := conditions.NewService(db, cat), contraception.NewService(db)
	svc := voicelog.NewService(client(spy), logs).WithWriters(voicelog.Writers{
		Flashes: menopause.NewService(db, cat), Pain: conds, Pills: contr, Bladder: pelvic.NewService(pelvicstore.New(db), cat),
	})
	h := voicelog.NewHandlers(svc, plusSvc, gate, bundles, languages, clock.Real{})
	lh := healthlog.NewLogHandlers(logs, clock.Real{}, bundles, languages)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/logs/voice", locale, guard, gate.Require(plus.VoiceLog), h.Voice)
	app.Put("/api/v1/logs/days/:date", locale, guard, lh.Save)
	app.Post("/api/v1/logs/voice/commit", locale, guard, h.Commit)
	return &voiceEnv{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), conds: conds, contr: contr}, spy
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
		{"target":"log","category":"pain","param":"location","item":"abdomen","value":"moderate","confidence":0.9,"label":"درد شکم · متوسط","options":[]},
		{"target":"log","category":"symptoms","param":"digestive","item":"bloating","value":"yes","confidence":0.9,"label":"نفخ","options":[]},
		{"target":"log","category":"mood","param":"moods","item":"bored","value":true,"confidence":0.9,"label":"بی\u200cحوصله","options":[
			{"category":"mood","param":"moods","item":"sad","value":true,"label":"غمگین"},
			{"category":"symptoms","param":"general","item":"fatigue","value":"yes","label":"خستگی"}]}
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

func (e *voiceEnv) commit(t *testing.T, token, lang, body string) resp {
	t.Helper()
	return e.do(t, "POST", "/api/v1/logs/voice/commit", token, lang, "application/json", strings.NewReader(body))
}

func suggestionKeys(t *testing.T, r resp) map[string]any {
	t.Helper()
	out := map[string]any{}
	list, _ := r.data()["suggestions"].([]any)
	for _, x := range list {
		m := x.(map[string]any)
		item, _ := m["item"].(string)
		out[m["target"].(string)+":"+m["category"].(string)+"."+m["param"].(string)+"."+item] = m["value"]
	}
	return out
}

func TestVoice_CanvasMenopauseHotFlashes(t *testing.T) {
	e, _ := setupVoice(t, fakeClient)
	a, tokA := e.user(t, "09120000201", true)
	b, tokB := e.user(t, "09120000202", true)
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, gender, created_at, updated_at) VALUES (?, 'menopause', 'female', NOW(), NOW())`, a)
	require.NoError(t, err)

	r := e.voice(t, tokA, "fa", "menopause")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "menopause", r.data()["mode"])
	assert.Equal(t, map[string]any{
		"hot_flash:hot_flash.count.":        3.0,
		"hot_flash:hot_flash.night.":        true,
		"log:symptoms.general.night_sweats": "yes",
		"log:symptoms.general.brain_fog":    "yes",
		"log:menopause.triggers.caffeine":   true,
	}, suggestionKeys(t, r))
	// cycle mode: the same words are the hot-flash symptom of the log, nothing for the diary
	rb := e.voice(t, tokB, "fa", "menopause")
	require.Equal(t, 200, rb.status, rb.raw)
	assert.Equal(t, map[string]any{"log:symptoms.general.hot_flashes": "yes"}, suggestionKeys(t, rb))

	c := e.commit(t, tokA, "fa", `{"date":"2026-09-23","items":[{"category":"hot_flash","param":"count","value":3},{"category":"hot_flash","param":"night","value":true}]}`)
	require.Equal(t, 200, c.status, c.raw)
	assert.Equal(t, "2 مورد ثبت شد.", c.body["message"])
	assert.JSONEq(t, `{"date":"2026-09-23","saved":[
		{"target":"hot_flash","category":"hot_flash","param":"count","item":null,"value":3,"label":"گرگرفتگی · ۳ بار"},
		{"target":"hot_flash","category":"hot_flash","param":"night","item":null,"value":true,"label":"گرگرفتگی شبانه"}]}`, mustJSON(t, c.data()))
	assert.Equal(t, 3, e.count(t, `SELECT COUNT(*) FROM hot_flashes WHERE user_id = ? AND night = 1 AND duration_s = ?
		AND started_at BETWEEN '2026-09-23 02:50:00' AND '2026-09-23 03:00:00'`, a, voicelog.DefaultFlashSeconds))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM hot_flashes WHERE user_id = ?`, b), "user-scoped")

	// B is not in menopause mode: the same commit is refused and writes nothing
	cb := e.commit(t, tokB, "en", `{"date":"2026-09-23","items":[{"category":"hot_flash","param":"count","value":3}]}`)
	require.Equal(t, 422, cb.status, cb.raw)
	assert.Equal(t, "Hot flashes are logged in menopause mode only.", cb.body["message"])
	assert.Contains(t, cb.body["errors"], "items.0.category")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM hot_flashes WHERE user_id = ?`, b))
	assert.Equal(t, 3, e.count(t, `SELECT COUNT(*) FROM hot_flashes WHERE user_id = ?`, a))
}

func TestVoice_CanvasPainPillBladder(t *testing.T) {
	e, _ := setupVoice(t, fakeClient)
	a, tok := e.user(t, "09120000203", true)
	other, _ := e.user(t, "09120000204", true)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

	// not enrolled, no pill method: only the bladder diary is offered
	r := e.voice(t, tok, "en", "pain_diary")
	require.Equal(t, 200, r.status, r.raw)
	assert.NotContains(t, r.raw, `"target":"pain_diary"`)
	assert.NotContains(t, e.voice(t, tok, "en", "pill").raw, `"target":"pill"`)

	require.NoError(t, e.conds.Enrol(ctx, a, conditions.ProgramEndo, civildate.MustParse("2026-09-01"), now))
	require.NoError(t, e.contr.SaveMethod(ctx, a, contraception.Input{Method: contraception.MethodCombinedPill,
		PackType: "21_7", PackStartedOn: civildate.MustParse("2026-09-16")}, now, "en"))

	r = e.voice(t, tok, "en", "pain_diary")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, map[string]any{
		"log:pain.location.abdomen":               "moderate",
		"log:pain.relief.painkiller":              true,
		"pain_diary:pain_diary.score.":            6.0,
		"pain_diary:pain_diary.analgesic.":        "ibuprofen",
		"pain_diary:pain_diary.analgesic_time.":   "10:00",
		"pain_diary:pain_diary.analgesic_effect.": "a_little",
		"pain_diary:pain_diary.missed_activity.":  true,
	}, suggestionKeys(t, r))
	assert.Equal(t, map[string]any{"pill:pill.status.": "taken"}, suggestionKeys(t, e.voice(t, tok, "fa", "pill")))
	assert.Equal(t, map[string]any{"bladder:bladder.leak.": "cough", "bladder:bladder.night_voids.": 2.0},
		suggestionKeys(t, e.voice(t, tok, "fa", "pelvic")))

	// a pain score needs a location that day: refused before anything is written
	items := `[{"category":"pain_diary","param":"score","value":6},{"category":"pain_diary","param":"analgesic","value":"ibuprofen"},
		{"category":"pain_diary","param":"analgesic_time","value":"10:00"},{"category":"pain_diary","param":"analgesic_effect","value":"a_little"},
		{"category":"pain_diary","param":"missed_activity","value":true},{"category":"pill","param":"status","value":"taken"},
		{"category":"bladder","param":"leak","value":"cough"},{"category":"bladder","param":"night_voids","value":2}]`
	c := e.commit(t, tok, "en", `{"date":"2026-09-23","items":`+items+`}`)
	require.Equal(t, 422, c.status, c.raw)
	assert.Contains(t, c.body["errors"], "items.0.value")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM contraception_pill_logs WHERE user_id = ?`, a), "nothing half-saved")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM pelvic_bladder_logs WHERE user_id = ?`, a))

	// the client saves the log suggestions first (PUT /logs/days), then commits the canvas ones
	put := e.do(t, "PUT", "/api/v1/logs/days/2026-09-23", tok, "en", "application/json",
		strings.NewReader(`{"categories":{"pain":{"location":{"abdomen":"moderate"}}},"voice_params":["pain.location"]}`))
	require.Equal(t, 200, put.status, put.raw)
	c = e.commit(t, tok, "en", `{"date":"2026-09-23","items":`+items+`}`)
	require.Equal(t, 200, c.status, c.raw)
	assert.Equal(t, "8 items saved.", c.body["message"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM condition_pain_entries WHERE user_id = ? AND entry_date = '2026-09-23'
		AND analgesic = 'ibuprofen' AND analgesic_time = '10:00' AND analgesic_effect = 'a_little' AND missed_activity = 1`, a))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND category = 'pain' AND param = 'location'
		AND item = 'abdomen' AND value_num = 6`, a), "the score is written to the log through the pain diary")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM contraception_pill_logs WHERE user_id = ? AND log_date = '2026-09-23' AND status = 'taken'`, a))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM pelvic_bladder_logs WHERE user_id = ? AND log_date = '2026-09-23' AND leak = 'cough' AND night_voids = 2`, a))
	for _, table := range []string{"condition_pain_entries", "contraception_pill_logs", "pelvic_bladder_logs", "health_log_entries"} {
		assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, other), table)
	}
}

func TestVoice_CommitValidation(t *testing.T) {
	e, _ := setupVoice(t, fakeClient)
	_, tok := e.user(t, "09120000205", false) // free user: committing runs no AI and is not Plus-gated

	anon := e.commit(t, "", "en", `{}`)
	assert.Equal(t, 401, anon.status)
	assert.Equal(t, "unauthenticated", anon.body["error_code"])

	r := e.commit(t, tok, "en", `{"date":"2026-09-24","items":[]}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "date")
	assert.Contains(t, r.body["errors"], "items")
	r = e.commit(t, tok, "en", `{"date":"2026-09-01","items":[{"category":"bladder"}]}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "date", "older than the 7-day window")
	assert.Contains(t, r.body["errors"], "items.0.param")
	r = e.commit(t, tok, "en", `{"date":"2026-09-23","items":[{"category":"pain","param":"location","value":"mild"}]}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, "This item can't be logged by voice.", r.body["message"])
	r = e.commit(t, tok, "fa", `{"date":"2026-09-23","items":[{"category":"bladder","param":"leak","value":["cough"]}]}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, "مقدار این مورد معتبر نیست.", r.body["message"])

	ok := e.commit(t, tok, "en", `{"date":"2026-09-22","items":[{"category":"bladder","param":"leak","value":"urgency"}]}`)
	require.Equal(t, 200, ok.status, ok.raw)
	assert.Contains(t, mustJSON(t, ok.data()), `"label":"Leak · With urgency"`)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
