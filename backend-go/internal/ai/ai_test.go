package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
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

// canvasVocab is a menopause-mode vocabulary with every canvas field (CB-VOICE-01) and the menopause items.
func canvasVocab() []VocabEntry {
	pain := []VocabValue{{"mild", "کم"}, {"moderate", "متوسط"}, {"severe", "شدید"}}
	sym := []VocabValue{{"yes", "دارم"}, {"no", "ندارم"}, {"mild", "کم"}, {"moderate", "متوسط"}, {"severe", "شدید"}}
	return []VocabEntry{
		{Key: "pain.location.abdomen", Type: "items", Label: "درد › محل درد › شکم", Values: pain},
		{Key: "pain.relief.painkiller", Type: "multi", Label: "درد › تسکین › مسکن"},
		{Key: "symptoms.general.hot_flashes", Type: "items", Label: "علائم › عمومی › گرگرفتگی", Values: sym},
		{Key: "symptoms.general.night_sweats", Type: "items", Label: "علائم › عمومی › تعریق شبانه", Values: sym},
		{Key: "symptoms.general.brain_fog", Type: "items", Label: "علائم › عمومی › مه مغزی", Values: sym},
		{Key: "symptoms.general.fatigue", Type: "items", Label: "علائم › عمومی › خستگی", Values: sym},
		{Key: "urogenital.symptoms.leakage", Type: "items", Label: "ادراری › علائم › نشت ادرار", Values: sym},
		{Key: "menopause.triggers.caffeine", Type: "multi", Label: "یائسگی › محرک\u200cها › کافئین"},
		{Key: "mood.moods.bored", Type: "multi", Label: "حال › حال و روحیه › بی\u200cحوصله"},
		{Key: "mood.moods.sad", Type: "multi", Label: "حال › حال و روحیه › غمگین"},
		{Key: "hot_flash.count", Type: "integer", Label: "گرگرفتگی › تعداد", Min: 1, Max: 10},
		{Key: "hot_flash.night", Type: "bool", Label: "گرگرفتگی › شبانه"},
		{Key: "pain_diary.score", Type: "integer", Label: "دفتر درد › شدت", Min: 0, Max: 10},
		{Key: "pain_diary.analgesic", Type: "text", Label: "دفتر درد › مسکن", MaxLen: 100},
		{Key: "pain_diary.analgesic_time", Type: "time", Label: "دفتر درد › ساعت مسکن"},
		{Key: "pain_diary.analgesic_effect", Type: "single", Label: "دفتر درد › اثر مسکن",
			Values: []VocabValue{{"no", "نه"}, {"a_little", "کمی"}, {"helped", "کمک کرد"}}},
		{Key: "pain_diary.missed_activity", Type: "bool", Label: "دفتر درد › نرفتن سر کار"},
		{Key: "pill.status", Type: "single", Label: "قرص › وضعیت", Values: []VocabValue{{"taken", "خورده شد"}, {"missed", "جا ماند"}}},
		{Key: "bladder.leak", Type: "single", Label: "مثانه › نشت",
			Values: []VocabValue{{"none", "نه"}, {"cough", "سرفه"}, {"urgency", "فوریت"}, {"unexplained", "بی\u200cدلیل"}}},
		{Key: "bladder.night_voids", Type: "integer", Label: "مثانه › بیدار شدن شبانه", Min: 0, Max: 20},
	}
}

func TestFake_ParseLog_CanvasFixtures(t *testing.T) {
	f := NewFake()
	parse := func(text string, v []VocabEntry) map[string]any {
		cs, _, err := f.ParseLog(context.Background(), LogParseRequest{Text: text, Vocabulary: v})
		require.NoError(t, err)
		return keys(cs)
	}
	for _, lang := range []string{"fa", "en"} {
		t.Run(lang, func(t *testing.T) {
			assert.Equal(t, map[string]any{
				"hot_flash.count":               3.0,
				"hot_flash.night":               true,
				"symptoms.general.night_sweats": "yes",
				"symptoms.general.brain_fog":    "yes",
				"menopause.triggers.caffeine":   true,
			}, parse(FakeTranscripts["menopause"][lang], canvasVocab()), "hot flashes go to the diary, not the symptom item")

			assert.Equal(t, map[string]any{
				"pain.location.abdomen":       "moderate", // 6 of 10 → moderate, whatever «کمی» says
				"pain.relief.painkiller":      true,
				"pain_diary.score":            6.0,
				"pain_diary.analgesic":        map[string]string{"fa": "ایبوپروفن", "en": "ibuprofen"}[lang],
				"pain_diary.analgesic_time":   "10:00",
				"pain_diary.analgesic_effect": "a_little",
				"pain_diary.missed_activity":  true,
			}, parse(FakeTranscripts["pain_diary"][lang], canvasVocab()))

			assert.Equal(t, map[string]any{"pill.status": "taken"}, parse(FakeTranscripts["pill"][lang], canvasVocab()))

			assert.Equal(t, map[string]any{"bladder.leak": "cough", "bladder.night_voids": 2.0},
				parse(FakeTranscripts["pelvic"][lang], canvasVocab()), "a leak goes to the bladder diary, not the symptom item")
		})
	}
	// missed pill; an iron pill is not the contraceptive pill
	assert.Equal(t, map[string]any{"pill.status": "missed"}, parse("دیروز قرصم یادم رفت", canvasVocab()))
	assert.Equal(t, map[string]any{"pill.status": "missed"}, parse("I forgot my pill", canvasVocab()))
	assert.Empty(t, parse("ساعت ۹ قرص آهنم رو خوردم", canvasVocab()))
	// urgency leak; no canvas field in the vocabulary → the log symptom / nothing
	assert.Equal(t, map[string]any{"bladder.leak": "urgency"}, parse("یهو ادرارم نشت کرد، فوری بود", canvasVocab()))
	assert.Equal(t, map[string]any{"symptoms.general.hot_flashes": "yes"},
		parse(FakeTranscripts["menopause"]["fa"], canvasVocab()[2:3]), "cycle mode: the symptom item")
	assert.Empty(t, parse(FakeTranscripts["pill"]["fa"], vocab()))
}

func TestFake_ParseLog_Alternatives(t *testing.T) {
	cs, _, err := NewFake().ParseLog(context.Background(), LogParseRequest{Text: FakeTranscripts["default"]["fa"], Vocabulary: canvasVocab()})
	require.NoError(t, err)
	var bored Candidate
	for _, c := range cs {
		if c.Key == "mood.moods.bored" {
			bored = c
		}
	}
	assert.Equal(t, []string{"mood.moods.sad", "symptoms.general.fatigue"}, bored.Alternatives)
	cs, _, _ = NewFake().ParseLog(context.Background(), LogParseRequest{Text: FakeTranscripts["default"]["fa"], Vocabulary: vocab()})
	for _, c := range cs {
		assert.Empty(t, c.Alternatives, "alternatives outside the vocabulary are dropped")
	}
}

func TestGemini_ParseLog_CanvasSchema(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := geminiServer(t, 200, `{"items":[{"key":"mood.moods.bored","value":true,"confidence":0.7,"alternatives":["mood.moods.sad"]},{"key":"pain_diary.analgesic_time","value":"10:00","confidence":0.8}]}`, &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	cs, _, err := g.ParseLog(context.Background(), LogParseRequest{Text: "x", Language: "fa", Vocabulary: canvasVocab()})
	require.NoError(t, err)
	require.Len(t, cs, 2)
	assert.Equal(t, []string{"mood.moods.sad"}, cs[0].Alternatives)
	assert.Equal(t, "10:00", cs[1].Value)
	body := mustJSON(t, seen)
	assert.Contains(t, body, "pain_diary.analgesic | text | دفتر درد › مسکن | ≤ 100 characters")
	assert.Contains(t, body, "pain_diary.analgesic_time | time | دفتر درد › ساعت مسکن | HH:MM")
	assert.Contains(t, body, "alternatives")
}

// gapVocab adds the CB-VOICE-03b slots (bleeding, cravings «ترش», sex «خشکی», vaginal dryness) to the canvas one.
func gapVocab() []VocabEntry {
	sym := []VocabValue{{"yes", "دارم"}, {"no", "ندارم"}, {"mild", "کم"}, {"moderate", "متوسط"}, {"severe", "شدید"}}
	return append(canvasVocab(),
		VocabEntry{Key: "bleeding.flow", Type: "single", Label: "خونریزی › میزان",
			Values: []VocabValue{{"light", "کم"}, {"medium", "متوسط"}, {"heavy", "زیاد"}, {"very_heavy", "خیلی زیاد"}}},
		VocabEntry{Key: "bleeding.presence", Type: "single", Label: "خونریزی › وضعیت",
			Values: []VocabValue{{"none", "نداشتم"}, {"spotting", "لکه\u200cبینی"}, {"bleeding", "خونریزی"}}},
		VocabEntry{Key: "pain.location.back", Type: "items", Label: "درد › محل درد › کمر", Values: sym[2:]},
		VocabEntry{Key: "pain.location.joints", Type: "items", Label: "درد › محل درد › مفاصل", Values: sym[2:]},
		VocabEntry{Key: "appetite_energy.cravings.sour", Type: "multi", Label: "اشتها › هوس › ترش"},
		VocabEntry{Key: "sex.symptoms.dryness", Type: "multi", Label: "رابطه › علائم › خشکی"},
		VocabEntry{Key: "urogenital.symptoms.vaginal_dryness", Type: "items", Label: "ادراری › علائم › خشکی واژن", Values: sym},
		VocabEntry{Key: "menopause.triggers.hot_drink", Type: "multi", Label: "یائسگی › محرک\u200cها › نوشیدنی داغ"},
	)
}

// TestFake_ParseLog_QAGaps pins the CB-VOICE-03b fixes (gaps found by the CB-VOICE-03 accuracy fixtures).
func TestFake_ParseLog_QAGaps(t *testing.T) {
	parse := func(text string, v []VocabEntry) map[string]any {
		cs, _, err := NewFake().ParseLog(context.Background(), LogParseRequest{Text: text, Vocabulary: v})
		require.NoError(t, err)
		return keys(cs)
	}
	without := func(v []VocabEntry, drop ...string) []VocabEntry {
		var out []VocabEntry
		for _, e := range v {
			if !slices.Contains(drop, e.Key) {
				out = append(out, e)
			}
		}
		return out
	}
	cycle := without(gapVocab(), "bleeding.presence")
	meno := without(gapVocab(), "bleeding.flow")

	t.Run("bleeding", func(t *testing.T) {
		assert.Equal(t, "heavy", parse("خونریزیم زیاده", cycle)["bleeding.flow"])
		assert.Equal(t, "very_heavy", parse("پریودم خیلی زیاده", cycle)["bleeding.flow"])
		assert.Equal(t, "light", parse("خونریزیم کمه", cycle)["bleeding.flow"])
		assert.Equal(t, "medium", parse("امروز پریودم شروع شد", cycle)["bleeding.flow"], "amount not said → medium")
		assert.Equal(t, "heavy", parse("I have a heavy period today", cycle)["bleeding.flow"])
		assert.Equal(t, "light", parse("my bleeding is light", cycle)["bleeding.flow"])
		assert.Empty(t, parse("پریودم دیر کرده", cycle), "a late period is no bleeding")
		assert.Empty(t, parse("خونریزی ندارم", cycle))
		assert.Equal(t, map[string]any{"bleeding.presence": "bleeding"}, parse("امروز خونریزی داشتم", meno))
		assert.Equal(t, map[string]any{"bleeding.presence": "none"}, parse("خونریزی نداشتم", meno))
		assert.Equal(t, map[string]any{"bleeding.presence": "spotting"}, parse("یه کم لکه بینی داشتم", meno))
		assert.Empty(t, parse("از خواب پریدم", cycle), "«پریدم» is not «پریود»")
	})

	t.Run("adverb inside a pain phrase", func(t *testing.T) {
		assert.Equal(t, map[string]any{"pain.location.abdomen": "severe"}, parse("دلم خیلی درد می\u200cکنه", cycle))
		assert.Equal(t, map[string]any{"pain.location.back": "mild"}, parse("کمرم یه کم درد میکنه", cycle))
		assert.Equal(t, map[string]any{"pain.location.joints": "severe"}, parse("زانوهام هم خیلی درد میکنه", meno))
		assert.Equal(t, map[string]any{"pain.location.back": "moderate"}, parse("my back really hurts", cycle), "«really» is not a level word")
	})

	t.Run("N times with a hot flash", func(t *testing.T) {
		got := parse("دیشب دو بار با گرگرفتگی از خواب پریدم", meno)
		assert.Equal(t, 2.0, got["hot_flash.count"])
		assert.Equal(t, true, got["hot_flash.night"])
		assert.Equal(t, 4.0, parse("امروز چهار بار گر گرفتم", meno)["hot_flash.count"])
		assert.Equal(t, 1.0, parse("دو بار رفتم دستشویی و یه گرگرفتگی داشتم", meno)["hot_flash.count"],
			"a count of something else is not the flash count")
	})

	t.Run("brain fog", func(t *testing.T) {
		for _, s := range []string{"حواسم پرته", "تمرکز ندارم", "نمی\u200cتونم تمرکز کنم", "I can't concentrate"} {
			assert.Equal(t, map[string]any{"symptoms.general.brain_fog": "yes"}, parse(s, meno), s)
		}
	})

	t.Run("clock without am/pm", func(t *testing.T) {
		at := func(text string) any { return parse(text, cycle)["pain_diary.analgesic_time"] }
		assert.Equal(t, "14:00", at("ساعت دو یه ژلوفن خوردم"), "bare 1–6 → afternoon")
		assert.Equal(t, "18:30", at("ساعت 6:30 یه ژلوفن خوردم"))
		assert.Equal(t, "10:00", at("ساعت ده یه ژلوفن خوردم"), "bare 7–12 → as said")
		assert.Equal(t, "12:00", at("ساعت دوازده یه ژلوفن خوردم"))
		assert.Equal(t, "02:00", at("ساعت دو شب یه ژلوفن خوردم"), "night 1–5 → after midnight")
		assert.Equal(t, "22:00", at("ساعت ده شب یه ژلوفن خوردم"))
		assert.Equal(t, "05:00", at("صبح ساعت پنج یه ژلوفن خوردم"))
		assert.Equal(t, "15:00", at("ساعت سه بعدازظهر یه ژلوفن خوردم"))
		assert.Equal(t, "02:00", at("I took an ibuprofen at 2 am"))
		assert.Equal(t, "20:00", at("I took an ibuprofen at 8 pm"))
		assert.Equal(t, "16:00", at("I took an ibuprofen at four"))
		assert.Equal(t, "17:00", at("ساعت 17 یه ژلوفن خوردم"))
		assert.Nil(t, at("ساعت 30 یه ژلوفن خوردم"))
	})

	t.Run("whole-word labels", func(t *testing.T) {
		assert.Equal(t, map[string]any{"hot_flash.count": 5.0, "menopause.triggers.hot_drink": true},
			parse("امروز پنج بار گرگرفتگی داشتم، بیشترش بعد از چای داغ", meno), "no «ترش» inside «بیشترش»")
		assert.Equal(t, map[string]any{"appetite_energy.cravings.sour": true}, parse("هوس ترشی کردم", meno), "suffix allowed")
		assert.Equal(t, map[string]any{"urogenital.symptoms.vaginal_dryness": "yes"},
			parse("خشکی واژن اذیتم میکنه", meno), "the lexicon's words are not read again as «رابطه › خشکی»")
		assert.Equal(t, map[string]any{"sex.symptoms.dryness": true},
			parse("موقع رابطه خشکی داشتم", meno), "the label alone still matches")
		assert.Equal(t, map[string]any{"sex.symptoms.dryness": true},
			parse("خشکی واژن دارم", without(cycle, "urogenital.symptoms.vaginal_dryness")),
			"cycle mode: no vaginal-dryness slot, so the sex item is the reading")
	})
}

func TestClockHour(t *testing.T) {
	for _, c := range []struct {
		h    int
		part string
		want int
	}{
		{2, "", 14}, {6, "", 18}, {7, "", 7}, {12, "", 12}, {0, "", 0}, {13, "", 13}, {24, "", -1},
		{12, "am", 0}, {9, "صبح", 9}, {12, "شب", 0}, {3, "شب", 3}, {9, "شب", 21}, {1, "ظهر", 13}, {12, "ظهر", 12},
		{5, "عصر", 17}, {11, "pm", 23},
	} {
		assert.Equal(t, c.want, clockHour(c.h, c.part), "%d %s", c.h, c.part)
	}
}
