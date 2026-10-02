package voicelog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/resources/translations"
)

// webm is the EBML magic of a Chrome MediaRecorder file followed by filler.
func webm(extra string) []byte {
	return append([]byte("\x1a\x45\xdf\xa3\x9f\x42\x86\x81\x01webm-opus-data "), []byte(extra)...)
}

func ns(t *testing.T, locale string) any {
	t.Helper()
	return i18n.NewTranslationStore(translations.FS, "").NamespaceMessages(locale, "log-taxonomy", "fa")
}

func TestSniff(t *testing.T) {
	assert.Equal(t, "audio/webm", Sniff(webm("")))
	assert.Equal(t, "audio/ogg", Sniff([]byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00")))
	assert.Equal(t, "audio/mp4", Sniff([]byte("\x00\x00\x00\x1cftypM4A \x00\x00\x00\x00isomM4A ")))
	assert.Equal(t, "audio/wav", Sniff([]byte("RIFF\x24\x00\x00\x00WAVEfmt ")))
	assert.Equal(t, "audio/mpeg", Sniff([]byte("ID3\x03\x00\x00\x00\x00\x00\x00")))
	assert.Equal(t, "audio/mp4", Sniff([]byte("\x00\x00\x00\x18ftypisom\x00\x00\x02\x00")))
	assert.Equal(t, "audio/mp4", Sniff([]byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00")))
	assert.Empty(t, Sniff([]byte("\x00\x00\x00\x14ftypqt  \x00\x00\x00\x00")), "QuickTime / video brands are refused")
	assert.Empty(t, Sniff([]byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00")), "an image in an ISO BMFF box")
	assert.Empty(t, Sniff([]byte("%PDF-1.7 hello")))
	assert.Empty(t, Sniff([]byte("\x89PNG\r\n\x1a\n0000")))
	assert.Empty(t, Sniff([]byte("<html><script>alert(1)</script>")))
}

func vocabFor(t *testing.T, mode, locale string, custom ...store.HealthLogCustomItem) *vocabulary {
	t.Helper()
	return buildVocabulary(mode, ns(t, locale), custom)
}

func TestVocabulary_ModeAndTypes(t *testing.T) {
	v := vocabFor(t, "cycle", "fa", store.HealthLogCustomItem{ID: 7, Category: "mood", Param: "moods", Label: "دلتنگ"})
	assert.Contains(t, v.slots, "pain.location.abdomen")
	assert.Contains(t, v.slots, "mood.moods.custom_7")
	assert.Contains(t, v.slots, "measurements.weight")
	assert.NotContains(t, v.slots, "pain.location.leg", "pregnancy-only item")
	assert.NotContains(t, v.slots, "bleeding.lochia_amount", "postpartum-only param")
	assert.NotContains(t, v.slots, "note.text", "free text stays manual")
	assert.NotContains(t, v.slots, "meds.taken", "care reminder ids stay manual")
	for _, e := range v.entries {
		assert.NotContains(t, e.Label, "09", "no PII in the vocabulary")
	}
	var abd ai.VocabEntry
	for _, e := range v.entries {
		if e.Key == "pain.location.abdomen" {
			abd = e
		}
	}
	assert.Equal(t, "درد › محل درد › شکم", abd.Label)
	assert.Len(t, abd.Values, 3)

	assert.Contains(t, vocabFor(t, "pregnancy", "fa").slots, "pain.location.leg")
}

func TestValidate(t *testing.T) {
	v := vocabFor(t, "cycle", "fa", store.HealthLogCustomItem{ID: 7, Category: "mood", Param: "moods", Label: "دلتنگ"})
	got := v.validate([]ai.Candidate{
		{Key: "pain.location.abdomen", Value: "moderate", Confidence: 0.93},
		{Key: "symptoms.digestive.bloating", Value: "yes", Confidence: 0.9},
		{Key: "mood.moods.bored", Value: true, Confidence: 1.4},
		{Key: "mood.moods.custom_7", Value: true, Confidence: 0.8},
		{Key: "measurements.weight", Value: 58.456, Confidence: 0.8},
		{Key: "sleep.quality", Value: "good", Confidence: 0.8},
		// dropped:
		{Key: "pain.location.abdomen", Value: "severe", Confidence: 0.9}, // duplicate
		{Key: "pain.location.leg", Value: "mild", Confidence: 0.9},       // not in cycle mode
		{Key: "pain.location.head", Value: "yes", Confidence: 0.9},       // not a pain level
		{Key: "symptoms.digestive.nausea", Value: "no", Confidence: 0.9}, // "no" is not a suggestion
		{Key: "measurements.weight", Value: 900, Confidence: 0.9},        // dup anyway
		{Key: "measurements.bbt", Value: 50, Confidence: 0.9},            // out of range
		{Key: "sleep.duration", Value: "forever", Confidence: 0.9},       // unknown option
		{Key: "mood.moods.custom_99", Value: true, Confidence: 0.9},      // someone else's / unknown custom item
		{Key: "drop.table", Value: true, Confidence: 0.9},
		{Key: "sex.desire", Value: "higher", Confidence: 0.3}, // unsure
		{Key: "activity.duration", Value: "abc", Confidence: 0.9},
	}, "cycle", "fa")
	labels := map[string]string{}
	values := map[string]any{}
	for _, s := range got {
		k := s.Category + "." + s.Param + "." + s.Item
		labels[k], values[k] = s.Label, s.Value
	}
	assert.Equal(t, map[string]string{
		"pain.location.abdomen":       "درد شکم · متوسط",
		"symptoms.digestive.bloating": "نفخ",
		"mood.moods.bored":            "بی\u200cحوصله",
		"mood.moods.custom_7":         "دلتنگ",
		"measurements.weight.":        "وزن · 58.46 کیلوگرم",
		"sleep.quality.":              "خواب · کیفیت: خوب",
	}, labels)
	assert.InDelta(t, 58.46, values["measurements.weight."], 0.0001)
	assert.InDelta(t, 1.0, got[2].Confidence, 0.0001, "confidence clamped")

	en := vocabFor(t, "cycle", "en").validate([]ai.Candidate{{Key: "pain.location.abdomen", Value: "severe", Confidence: 0.9}}, "cycle", "en")
	require.Len(t, en, 1)
	assert.Equal(t, "Abdomen pain · Severe", en[0].Label)

	raw, err := json.Marshal(got[0].JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"target":"log","category":"pain","param":"location","item":"abdomen","value":"moderate","confidence":0.93,"label":"درد شکم · متوسط","options":[]}`, string(raw))
	raw, _ = json.Marshal(got[4].JSON())
	assert.Contains(t, string(raw), `"item":null,"value":58.46`)
}

func TestValidate_Cap(t *testing.T) {
	v := vocabFor(t, "cycle", "fa")
	var cands []ai.Candidate
	for k := range v.slots {
		cands = append(cands, ai.Candidate{Key: k, Value: true, Confidence: 0.9})
		cands = append(cands, ai.Candidate{Key: k, Value: "yes", Confidence: 0.9})
	}
	assert.LessOrEqual(t, len(v.validate(cands, "cycle", "fa")), MaxSuggestions)
}

// stubLogs is the health log for unit tests.
type stubLogs struct{ mode string }

func (s stubLogs) LifeMode(context.Context, uint64) (string, error) { return s.mode, nil }
func (s stubLogs) CustomItems(context.Context, uint64, bool) ([]store.HealthLogCustomItem, error) {
	return nil, nil
}

// spyTranscriber keeps the audio buffer it was handed, to check it was wiped afterwards.
type spyTranscriber struct {
	seen []byte
	text string
	err  error
}

func (s *spyTranscriber) Transcribe(_ context.Context, req ai.TranscribeRequest) (ai.Transcript, ai.Usage, error) {
	s.seen = req.Audio.Data[:cap(req.Audio.Data)]
	return ai.Transcript{Text: s.text, Language: "fa"}, ai.Usage{}, s.err
}

func allZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func TestProcess_AudioWipedAfterTranscription(t *testing.T) {
	for name, spy := range map[string]*spyTranscriber{
		"success": {text: ai.FakeTranscripts["default"]["fa"]},
		"empty":   {text: ""},
		"failure": {err: ai.ErrUpstream},
	} {
		t.Run(name, func(t *testing.T) {
			svc := NewService(ai.NewClient("spy", spy, ai.NewFake(), nil), stubLogs{mode: "cycle"})
			buf := make([]byte, 64, 128)
			copy(buf, webm(""))
			audio := &ai.Audio{Data: buf, MIME: "audio/webm"}
			res, err := svc.Process(context.Background(), 1, "fa", ns(t, "fa"), audio, time.Now())
			require.NotEmpty(t, spy.seen)
			assert.True(t, allZero(spy.seen), "the recording buffer is zeroed right after transcription")
			assert.True(t, allZero(buf[:cap(buf)]))
			assert.Nil(t, audio.Data)
			if spy.err != nil {
				require.ErrorIs(t, err, ai.ErrUpstream)
				return
			}
			require.NoError(t, err)
			if spy.text == "" {
				assert.Empty(t, res.Suggestions)
				return
			}
			assert.Len(t, res.Suggestions, 3)
		})
	}
}

func TestProcess_Unavailable(t *testing.T) {
	audio := &ai.Audio{Data: webm("")}
	_, err := NewService(nil, stubLogs{mode: "cycle"}).Process(context.Background(), 1, "fa", nil, audio, time.Now())
	require.ErrorIs(t, err, ai.ErrUnavailable)
	assert.Nil(t, audio.Data)
}

// uploadApp runs readUpload behind a Fiber route and reports what it saw.
func uploadApp(t *testing.T, got **ai.Audio, raw *[]byte) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Post("/u", func(c fiber.Ctx) error {
		body := c.Request().Body()
		*raw = body
		defer clear(body)
		a, err := readUpload(c, body, "en")
		if err != nil {
			return err
		}
		*got = a
		return c.SendStatus(204)
	})
	return app
}

func multipartBody(t *testing.T, fields map[string]string, file []byte, fileName string) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	if file != nil {
		fw, err := w.CreateFormFile("audio", fileName)
		require.NoError(t, err)
		_, _ = fw.Write(file)
	}
	require.NoError(t, w.Close())
	return &b, w.FormDataContentType()
}

func post(t *testing.T, app *fiber.App, body io.Reader, ct string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/u", body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m
}

func TestReadUpload(t *testing.T) {
	var got *ai.Audio
	var raw []byte
	app := uploadApp(t, &got, &raw)

	b, ct := multipartBody(t, map[string]string{"duration_ms": "7000"}, webm("x"), "rec.webm")
	status, _ := post(t, app, b, ct)
	require.Equal(t, 204, status)
	require.NotNil(t, got)
	assert.Equal(t, "audio/webm", got.MIME, "sniffed, not the client's type")
	assert.Equal(t, webm("x"), got.Data)
	assert.True(t, allZero(raw), "the raw request body is zeroed")

	// missing file, not a file, empty file
	b, ct = multipartBody(t, map[string]string{"duration_ms": "1"}, nil, "")
	status, m := post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Equal(t, "The audio file field is required.", m["message"])
	b, ct = multipartBody(t, map[string]string{"audio": "not a file"}, nil, "")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Equal(t, "The audio file field must be a file.", m["message"])
	b, ct = multipartBody(t, nil, []byte{}, "rec.webm")
	status, _ = post(t, app, b, ct)
	assert.Equal(t, 422, status)

	// JSON body instead of multipart
	status, _ = post(t, app, bytes.NewBufferString(`{"audio":"AAAA"}`), "application/json")
	assert.Equal(t, 422, status)

	// not audio (an HTML / PDF renamed .webm)
	b, ct = multipartBody(t, nil, []byte("<html><body>hi</body></html>"), "rec.webm")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Equal(t, "The audio file field must be a file of type: "+AcceptedFormats+".", m["message"])

	// too long, not a number
	b, ct = multipartBody(t, map[string]string{"duration_ms": "95000"}, webm(""), "rec.webm")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Contains(t, m["errors"], "duration_ms")
	b, ct = multipartBody(t, map[string]string{"duration_ms": "abc"}, webm(""), "rec.webm")
	status, _ = post(t, app, b, ct)
	assert.Equal(t, 422, status)

	// negative length, an unknown field, too many parts
	b, ct = multipartBody(t, map[string]string{"duration_ms": "-5"}, webm(""), "rec.webm")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Contains(t, m["errors"], "duration_ms")
	b, ct = multipartBody(t, map[string]string{"user_name": "x"}, webm(""), "rec.webm")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Equal(t, "The audio file field is required.", m["message"])
	var many bytes.Buffer
	mw := multipart.NewWriter(&many)
	for range 5 {
		require.NoError(t, mw.WriteField("duration_ms", "1000"))
	}
	fw, err := mw.CreateFormFile("audio", "rec.webm")
	require.NoError(t, err)
	_, _ = fw.Write(webm(""))
	require.NoError(t, mw.Close())
	status, _ = post(t, app, &many, mw.FormDataContentType())
	assert.Equal(t, 422, status, "more than MaxParts parts")

	// over 2 MB
	big := append(webm(""), make([]byte, MaxAudioBytes)...)
	b, ct = multipartBody(t, nil, big, "rec.webm")
	status, m = post(t, app, b, ct)
	assert.Equal(t, 422, status)
	assert.Equal(t, "The audio file field must not be greater than 2048 kilobytes.", m["message"])
}

func TestLang_SameKeys(t *testing.T) {
	read := func(code string) map[string]any {
		b, err := langFS.ReadFile("lang/" + code + "/voicelog.json")
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		return m
	}
	fa, en := read("fa"), read("en")
	for group, v := range fa {
		require.Contains(t, en, group)
		for k := range v.(map[string]any) {
			assert.Contains(t, en[group], k, group+"."+k)
		}
	}
	assert.Equal(t, "Voice logging isn't available right now. Try again later or log by hand.", T("messages.ai_unavailable", "de", nil))
}
