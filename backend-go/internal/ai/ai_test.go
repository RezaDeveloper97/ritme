package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
)

func vocab() []VocabEntry {
	pain := []VocabValue{{"mild", "کم"}, {"moderate", "متوسط"}, {"severe", "شدید"}}
	sym := []VocabValue{{"yes", "دارم"}, {"no", "ندارم"}, {"mild", "کم"}, {"moderate", "متوسط"}, {"severe", "شدید"}}
	return []VocabEntry{
		{Key: "pain.location.abdomen", Type: "items", Label: "درد › محل درد › شکم", Values: pain},
		{Key: "pain.location.head", Type: "items", Label: "درد › محل درد › سر", Values: pain},
		{Key: "symptoms.digestive.bloating", Type: "items", Label: "علائم › گوارشی › نفخ", Values: sym},
		{Key: "symptoms.digestive.nausea", Type: "items", Label: "علائم › گوارشی › تهوع", Values: sym},
		{Key: "mood.moods.bored", Type: "multi", Label: "حال › حال و روحیه › بی\u200cحوصله"},
		{Key: "mood.moods.custom_7", Type: "multi", Label: "حال › حال و روحیه › دلتنگ"},
		{Key: "sleep.quality", Type: "single", Label: "خواب › کیفیت خواب", Values: []VocabValue{{"good", "خوب"}, {"poor", "بد"}}},
		{Key: "measurements.weight", Type: "number", Label: "اندازه\u200cها › وزن", Min: 20, Max: 300, Unit: "kg"},
	}
}

func audio(marker string) Audio {
	return Audio{Data: []byte("\x1aE\xdf\xa3....." + marker + "...."), MIME: "audio/webm"}
}

func TestFake_TranscribeFixtures(t *testing.T) {
	f := NewFake()
	tr, u, err := f.Transcribe(context.Background(), TranscribeRequest{Audio: audio(""), Language: "fa"})
	require.NoError(t, err)
	assert.Equal(t, FakeTranscripts["default"]["fa"], tr.Text)
	assert.Equal(t, "fake", u.Provider)

	tr, _, err = f.Transcribe(context.Background(), TranscribeRequest{Audio: audio("RITME-FAKE:weight"), Language: "en"})
	require.NoError(t, err)
	assert.Equal(t, FakeTranscripts["weight"]["en"], tr.Text)

	// unknown language → en; unknown key → default
	tr, _, err = f.Transcribe(context.Background(), TranscribeRequest{Audio: audio("RITME-FAKE:nope"), Language: "de"})
	require.NoError(t, err)
	assert.Equal(t, "en", tr.Language)
	assert.Equal(t, FakeTranscripts["default"]["en"], tr.Text)

	tr, _, err = f.Transcribe(context.Background(), TranscribeRequest{Audio: audio("RITME-FAKE:silence"), Language: "fa"})
	require.NoError(t, err)
	assert.Empty(t, tr.Text)

	_, _, err = f.Transcribe(context.Background(), TranscribeRequest{Audio: audio("RITME-FAKE:error"), Language: "fa"})
	require.ErrorIs(t, err, ErrUpstream)
}

func keys(cs []Candidate) map[string]any {
	out := map[string]any{}
	for _, c := range cs {
		out[c.Key] = c.Value
	}
	return out
}

func TestFake_ParseLog(t *testing.T) {
	f := NewFake()
	cs, _, err := f.ParseLog(context.Background(), LogParseRequest{Text: FakeTranscripts["default"]["fa"], Language: "fa", Vocabulary: vocab()})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"pain.location.abdomen":       "moderate",
		"symptoms.digestive.bloating": "yes",
		"mood.moods.bored":            true,
	}, keys(cs))

	cs, _, _ = f.ParseLog(context.Background(), LogParseRequest{Text: FakeTranscripts["weight"]["fa"], Vocabulary: vocab()})
	assert.Equal(t, map[string]any{"measurements.weight": 58.5, "sleep.quality": "good"}, keys(cs))

	cs, _, _ = f.ParseLog(context.Background(), LogParseRequest{Text: FakeTranscripts["headache"]["en"], Vocabulary: vocab()})
	assert.Equal(t, map[string]any{"pain.location.head": "severe", "symptoms.digestive.nausea": "yes"}, keys(cs))

	// custom item said verbatim; a slot outside the vocabulary never comes back
	cs, _, _ = f.ParseLog(context.Background(), LogParseRequest{Text: "امروز دلتنگم", Vocabulary: vocab()})
	assert.Equal(t, map[string]any{"mood.moods.custom_7": true}, keys(cs))
	cs, _, _ = f.ParseLog(context.Background(), LogParseRequest{Text: "سرم درد میکنه", Vocabulary: vocab()[2:]})
	assert.Empty(t, cs)
}

func TestNormalizeText(t *testing.T) {
	assert.Equal(t, "بیحوصلهام 58.5 کیلو", NormalizeText("  بی\u200cحوصله\u200cام   ۵۸٫۵ کيلو "))
}

func TestAudio_Wipe(t *testing.T) {
	a := audio("x")
	buf := a.Data
	a.Wipe()
	assert.Nil(t, a.Data)
	for _, b := range buf {
		require.Zero(t, b)
	}
}

type spyRecorder struct{ got []Usage }

func (s *spyRecorder) Record(_ context.Context, u Usage) { s.got = append(s.got, u) }

func TestClient_RecordsUsage(t *testing.T) {
	rec := &spyRecorder{}
	f := NewFake()
	c := NewClient("fake", f, f, rec)
	_, err := c.Transcribe(context.Background(), FeatureVoiceLog, TranscribeRequest{Audio: audio(""), Language: "fa"})
	require.NoError(t, err)
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "نفخ", Vocabulary: vocab()})
	require.NoError(t, err)
	require.Len(t, rec.got, 2)
	assert.Equal(t, "transcribe", rec.got[0].Op)
	assert.Equal(t, FeatureVoiceLog, rec.got[0].Feature)
	assert.Positive(t, rec.got[0].AudioBytes)
	assert.True(t, rec.got[0].OK)
	assert.Equal(t, "parse_log", rec.got[1].Op)

	var nilClient *Client
	_, err = nilClient.Transcribe(context.Background(), FeatureVoiceLog, TranscribeRequest{})
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestLogRecorder_NoPayload(t *testing.T) {
	var sb strings.Builder
	r := LogRecorder{Logger: slog.New(slog.NewJSONHandler(&sb, nil))}
	r.Record(context.Background(), Usage{Provider: "gemini", Model: "m", Feature: FeatureVoiceLog, Op: "transcribe", AudioBytes: 10, OK: true})
	assert.Contains(t, sb.String(), `"op":"transcribe"`)
	assert.NotContains(t, sb.String(), "user")
}

func TestNew_Registry(t *testing.T) {
	staging := config.App{Env: "staging"}
	prod := config.App{Env: "production"}
	assert.Equal(t, "fake", New(Deps{App: staging}).Provider())
	assert.Nil(t, New(Deps{App: prod}))                                           // unset in production → none
	assert.Nil(t, New(Deps{App: prod, Config: config.AI{Provider: "fake"}}))      // fake never in production
	assert.Nil(t, New(Deps{App: staging, Config: config.AI{Provider: "gemini"}})) // no key
	assert.Nil(t, New(Deps{App: staging, Config: config.AI{Provider: "none"}}))   // explicit none
	g := New(Deps{App: prod, Config: config.AI{Provider: "gemini", Gemini: config.Gemini{APIKey: "k"}}})
	require.NotNil(t, g)
	assert.Equal(t, "gemini", g.Provider())
}

// geminiServer answers like generateContent and records the last request.
func geminiServer(t *testing.T, status int, answer string, seen *map[string]any, header *http.Header) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta/models/test-model:generateContent", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery, "the key never rides in the URL")
		*header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, seen)
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":{"message":"bad things with secret-echo"}}`))
			return
		}
		body, _ := json.Marshal(map[string]any{
			"candidates":    []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": answer}}}}},
			"usageMetadata": map[string]any{"promptTokenCount": 120, "candidatesTokenCount": 30},
		})
		_, _ = w.Write(body)
	}))
}

func TestGemini_Transcribe(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := geminiServer(t, 200, "«از صبح دلم درد می\u200cکنه»", &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	tr, u, err := g.Transcribe(context.Background(), TranscribeRequest{Audio: Audio{Data: []byte("abc"), MIME: "audio/webm"}, Language: "fa"})
	require.NoError(t, err)
	assert.Equal(t, "از صبح دلم درد می\u200cکنه", tr.Text)
	assert.Equal(t, "test-key", h.Get("x-goog-api-key"))
	assert.Equal(t, 120, u.InputTokens)
	assert.Equal(t, 30, u.OutputTokens)
	parts := seen["contents"].([]any)[0].(map[string]any)["parts"].([]any)
	inline := parts[1].(map[string]any)["inline_data"].(map[string]any)
	assert.Equal(t, "audio/webm", inline["mime_type"])
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("abc")), inline["data"])
	assert.NotContains(t, mustJSON(t, seen), "mobile")
}

func TestGemini_ParseLog(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := geminiServer(t, 200, "```json\n{\"items\":[{\"key\":\"pain.location.abdomen\",\"value\":\"moderate\",\"confidence\":0.92}]}\n```", &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	cs, _, err := g.ParseLog(context.Background(), LogParseRequest{Text: "دلم درد میکنه", Language: "fa", Vocabulary: vocab()})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, Candidate{Key: "pain.location.abdomen", Value: "moderate", Confidence: 0.92}, cs[0])
	body := mustJSON(t, seen)
	assert.Contains(t, body, "pain.location.abdomen | items")
	assert.Contains(t, body, "application/json")
}

func TestGemini_Errors(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := geminiServer(t, 500, "", &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	_, _, err := g.Transcribe(context.Background(), TranscribeRequest{Audio: Audio{Data: []byte("a"), MIME: "audio/ogg"}})
	require.ErrorIs(t, err, ErrUpstream)
	assert.NotContains(t, err.Error(), "secret-echo", "the provider's error body is never surfaced")

	bad := geminiServer(t, 200, "not json", &seen, &h)
	defer bad.Close()
	g = NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: bad.URL}, bad.Client())
	_, _, err = g.ParseLog(context.Background(), LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrUpstream)

	// unreachable host
	g = NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: "http://127.0.0.1:1"}, http.DefaultClient)
	_, _, err = g.Transcribe(context.Background(), TranscribeRequest{})
	require.ErrorIs(t, err, ErrUpstream)
	assert.NotContains(t, err.Error(), "127.0.0.1")
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
