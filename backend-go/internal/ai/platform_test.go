package ai

// B-N6-05: PII filter, pricing, budget refusal, streaming chat and document extraction through the Client with
// the fake provider (default), and the Gemini chat / extraction wire format against httptest servers only.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
)

func TestRedact(t *testing.T) {
	names := []string{"Sara Ahmadi", "  سارا  رضایی "}
	cases := []struct{ in, want string }{
		{"call me on 09121234567 please", "call me on [phone] please"},
		{"شماره\u200cام ۰۹۱۲ ۱۲۳ ۴۵۶۷ است", "شماره\u200cام [phone] است"},
		{"+98 912-123-4567", "[phone]"},
		{"00989121234567", "[phone]"},
		{"کد ملی ۰۰۱۲۳۴۵۶۷۸", "کد ملی [number]"},
		{"id 001-234567-8 ok", "id [id] ok"},
		{"card 6037991234567890", "card [number]"},
		{"write to sara.a@example.com", "write to [email]"},
		{"I am Sara Ahmadi and sara is tired", "I am [name] and [name] is tired"},
		{"من سارا هستم، رضایی هم فامیلی\u200cام", "من [name] هستم، [name] هم فامیلی\u200cام"},
		{"سارا\u200cام", "[name]\u200cام"},
		// health values and dates are kept
		{"وزنم ۵۸٫۵ کیلو، فشار ۱۲۰/۸۰، قند 92 mg/dL", "وزنم ۵۸٫۵ کیلو، فشار ۱۲۰/۸۰، قند 92 mg/dL"},
		{"TSH 2.1, date 2026-09-20, 1403/07/12", "TSH 2.1, date 2026-09-20, 1403/07/12"},
		{"Sarah is not Sara", "Sarah is not [name]"}, // whole words only
		{"912345", "912345"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, Redact(tc.in, names), tc.in)
	}
	assert.Empty(t, Redact("", names))
}

func TestPricer(t *testing.T) {
	p := NewPricer(map[string]config.AIPrice{
		"m1": {InputPerMTok: 0.30, OutputPerMTok: 2.50, AudioPerMTok: 1.00},
		"m2": {InputPerMTok: 1.25, OutputPerMTok: 10, AudioPerMTok: 1.25},
	})
	assert.Equal(t, uint64(0), p.Cost(Usage{Provider: "fake", Model: "m2", InputTokens: 1e6}), "the fake is free")
	assert.Equal(t, uint64(300+2500), p.Cost(Usage{Provider: "gemini", Model: "m1", InputTokens: 1000, OutputTokens: 1000}))
	// audio tokens are part of the input and priced at the audio rate
	assert.Equal(t, uint64(500*0.30+500*1.00), p.Cost(Usage{Provider: "gemini", Model: "m1", InputTokens: 1000, AudioTokens: 500}))
	// unknown model → the highest rates, never free
	assert.Equal(t, uint64(1250+10000), p.Cost(Usage{Provider: "gemini", Model: "unknown", InputTokens: 1000, OutputTokens: 1000}))
	assert.Equal(t, uint64(1), p.Cost(Usage{Provider: "gemini", Model: "m1", InputTokens: 1}), "rounded up")
}

type stopLimiter struct{ calls int }

func (s *stopLimiter) Allow(context.Context) error { s.calls++; return ErrBudgetExceeded }

// spyChat records what reached the provider.
type spyChat struct {
	*Fake
	got ChatRequest
}

func (s *spyChat) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	s.got = req
	return s.Fake.ChatStream(ctx, req)
}

type spyExtract struct {
	*Fake
	got   ExtractRequest
	reply *Extraction
}

func (s *spyExtract) Extract(ctx context.Context, req ExtractRequest) (Extraction, Usage, error) {
	s.got = req
	if s.reply != nil {
		return *s.reply, Usage{Provider: "fake", Model: "x"}, nil
	}
	return s.Fake.Extract(ctx, req)
}

func collect(t *testing.T, ch <-chan ChatEvent) (string, ChatEvent) {
	t.Helper()
	var sb strings.Builder
	var last ChatEvent
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return sb.String(), last
			}
			sb.WriteString(ev.Delta)
			last = ev
		case <-timeout:
			t.Fatal("stream did not end")
		}
	}
}

func userCtx() context.Context {
	return WithSubject(context.Background(), Subject{UserID: 42, Names: []string{"Sara Ahmadi"}})
}

func TestClient_ChatStreamsRedactsAndRecords(t *testing.T) {
	rec := &spyRecorder{}
	spy := &spyChat{Fake: NewFake()}
	c := NewClientWith(Options{Provider: "fake", Chatter: spy, Recorder: rec})
	ch, err := c.Chat(userCtx(), FeatureAssistant, ChatRequest{
		System:   "You help Sara Ahmadi (age 31).",
		Messages: []ChatMessage{{Role: RoleUser, Text: "I'm Sara, call 09121234567, my belly hurts"}},
		Language: "en",
	})
	require.NoError(t, err)
	text, last := collect(t, ch)
	assert.Equal(t, FakeChatReplies["default"]["en"], text)
	assert.True(t, last.Done)
	assert.Equal(t, FinishStop, last.Finish)
	assert.Equal(t, "You help [name] (age 31).", spy.got.System)
	assert.Equal(t, "I'm [name], call [phone], my belly hurts", spy.got.Messages[0].Text)
	assert.Equal(t, DefaultChatMaxOutputTokens, spy.got.MaxOutputTokens)
	require.Eventually(t, func() bool { return len(rec.got) == 1 }, time.Second, 5*time.Millisecond)
	u := rec.got[0]
	assert.Equal(t, OpChat, u.Op)
	assert.Equal(t, FeatureAssistant, u.Feature)
	assert.Equal(t, uint64(42), u.UserID)
	assert.True(t, u.OK)
	assert.Positive(t, u.InputTokens)
	assert.Positive(t, u.OutputTokens)
}

func TestClient_ChatFixturesAndErrors(t *testing.T) {
	rec := &spyRecorder{}
	f := NewFake()
	c := NewClientWith(Options{Provider: "fake", Chatter: f, Recorder: rec})
	msg := func(text string) ChatRequest {
		return ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: text}}, Language: "fa"}
	}
	ch, err := c.Chat(context.Background(), FeatureAssistant, msg("RITME-FAKE:emergency درد شدید"))
	require.NoError(t, err)
	text, _ := collect(t, ch)
	assert.Contains(t, text, "۱۱۵")

	_, err = c.Chat(context.Background(), FeatureAssistant, msg("RITME-FAKE:error"))
	require.ErrorIs(t, err, ErrUpstream)

	ch, err = c.Chat(context.Background(), FeatureAssistant, msg("RITME-FAKE:error_mid"))
	require.NoError(t, err)
	text, last := collect(t, ch)
	assert.NotEmpty(t, text)
	require.ErrorIs(t, last.Err, ErrUpstream)

	ch, err = c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: msg("سلام").Messages, MaxOutputTokens: 5})
	require.NoError(t, err)
	_, last = collect(t, ch)
	assert.Equal(t, FinishLength, last.Finish)

	img := []Image{{Data: []byte("\xff\xd8\xff"), MIME: "image/jpeg"}}
	ch, err = c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "ببین", Images: img}}, Language: "fa"})
	require.NoError(t, err)
	text, _ = collect(t, ch)
	assert.Equal(t, FakeChatReplies["image"]["fa"], text)

	require.Eventually(t, func() bool { return len(rec.got) == 5 }, time.Second, 5*time.Millisecond)
	okCount := 0
	for _, u := range rec.got {
		if u.OK {
			okCount++
		}
	}
	assert.Equal(t, 3, okCount, "the two failures are logged as not ok")
	assert.Equal(t, 3, rec.got[4].ImageBytes)
}

func TestClient_ChatLimitsAndBudget(t *testing.T) {
	f := NewFake()
	lim := &stopLimiter{}
	c := NewClientWith(Options{Provider: "fake", Chatter: f, Limiter: lim})
	ok := []ChatMessage{{Role: RoleUser, Text: "hi"}}
	_, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: ok})
	require.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Equal(t, 1, lim.calls)

	c = NewClientWith(Options{Provider: "fake", Chatter: f})
	bad := []ChatRequest{
		{},
		{Messages: []ChatMessage{{Role: "system", Text: "x"}}},
		{Messages: []ChatMessage{{Role: RoleUser, Text: strings.Repeat("a", MaxChatTextRunes+1)}}},
		{Messages: []ChatMessage{{Role: RoleAssistant, Text: "x", Images: []Image{{Data: []byte("x"), MIME: "image/png"}}}}},
		{Messages: []ChatMessage{{Role: RoleUser, Images: []Image{{Data: []byte("x"), MIME: "image/gif"}}}}},
		{Messages: []ChatMessage{{Role: RoleUser, Images: make([]Image, MaxChatImages+1)}}},
	}
	for i, req := range bad {
		_, err := c.Chat(context.Background(), FeatureAssistant, req)
		require.ErrorIs(t, err, ErrInvalidRequest, i)
	}
	var nilClient *Client
	_, err = nilClient.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: ok})
	require.ErrorIs(t, err, ErrUnavailable)

	// every capability checks the budget first
	c = NewClientWith(Options{Provider: "fake", Transcriber: f, LogParser: f, Extractor: f, Limiter: lim})
	_, err = c.Transcribe(context.Background(), FeatureVoiceLog, TranscribeRequest{Audio: audio("")})
	require.ErrorIs(t, err, ErrBudgetExceeded)
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrBudgetExceeded)
	_, err = c.Extract(context.Background(), FeatureLabAnalysis, ExtractRequest{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: labSchema()})
	require.ErrorIs(t, err, ErrBudgetExceeded)
}

func TestClient_ChatCancelStopsGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	c := NewClientWith(Options{Provider: "fake", Chatter: NewFake()})
	for range 20 {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := c.Chat(ctx, FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}, Language: "en"})
		require.NoError(t, err)
		<-ch // one word, then the client goes away
		cancel()
	}
	require.Eventually(t, func() bool { return runtime.NumGoroutine() <= before+2 }, 2*time.Second, 10*time.Millisecond)
}

func labSchema() ExtractSchema {
	return ExtractSchema{
		Name:   "lab_panel",
		Fields: []FieldSpec{{Key: "date", Type: FieldDate, Description: "sample date"}},
		Items: []FieldSpec{
			{Key: "name", Type: FieldString, Description: "marker name as printed"},
			{Key: "value", Type: FieldNumber, Description: "result"},
			{Key: "unit", Type: FieldString, Description: "unit"},
			{Key: "ref_low", Type: FieldNumber, Description: "lower reference"},
			{Key: "ref_high", Type: FieldNumber, Description: "upper reference"},
		},
	}
}

func TestClient_ExtractFakeFixtures(t *testing.T) {
	rec := &spyRecorder{}
	f := NewFake()
	c := NewClientWith(Options{Provider: "fake", Extractor: f, Recorder: rec})
	doc := Document{Data: []byte("%PDF-1.7 ..."), MIME: "application/pdf"}
	out, err := c.Extract(userCtx(), FeatureLabAnalysis, ExtractRequest{Document: doc, Schema: labSchema(), Language: "fa"})
	require.NoError(t, err)
	require.Len(t, out.Fields, 1, "lab_name is not in the schema")
	assert.Equal(t, "2026-09-20", out.Fields[0].Value)
	require.Len(t, out.Items, 5)
	assert.Equal(t, []string{"name", "value", "unit", "ref_low", "ref_high"}, fieldKeys(out.Items[0]), "ref_text dropped, schema order")
	assert.InDelta(t, 11.4, out.Items[0][1].Value, 1e-9)
	require.Len(t, rec.got, 1)
	assert.Equal(t, OpExtract, rec.got[0].Op)
	assert.Equal(t, len(doc.Data), rec.got[0].ImageBytes)
	assert.Equal(t, uint64(42), rec.got[0].UserID)

	blurry := Document{Data: []byte("\x89PNG RITME-FAKE:blurry"), MIME: "image/png"}
	out, err = c.Extract(context.Background(), FeatureLabAnalysis, ExtractRequest{Document: blurry, Schema: labSchema()})
	require.NoError(t, err)
	assert.InDelta(t, 0.3, out.Items[0][0].Confidence, 1e-9)

	_, err = c.Extract(context.Background(), FeatureLabAnalysis, ExtractRequest{Document: Document{Data: []byte("RITME-FAKE:error"), MIME: "image/png"}, Schema: labSchema()})
	require.ErrorIs(t, err, ErrUpstream)

	imaging := ExtractSchema{Name: "imaging", Fields: []FieldSpec{
		{Key: "ga_weeks", Type: FieldNumber}, {Key: "edd", Type: FieldDate},
		{Key: "kind", Type: FieldEnum, Values: []string{"ultrasound", "nt_scan"}}, {Key: "findings", Type: FieldString},
	}}
	out, err = c.Extract(context.Background(), FeatureDocExtract, ExtractRequest{Document: doc, Schema: imaging})
	require.NoError(t, err)
	assert.Equal(t, []string{"ga_weeks", "edd", "kind", "findings"}, fieldKeys(out.Fields))
	assert.Empty(t, out.Items)

	out, err = c.Extract(context.Background(), FeatureDocExtract, ExtractRequest{Document: doc, Schema: ExtractSchema{Name: "other", Fields: imaging.Fields}})
	require.NoError(t, err)
	assert.Empty(t, out.Fields, "unknown schema → blank fixture")
}

func fieldKeys(row []ExtractedField) []string {
	out := make([]string, len(row))
	for i, f := range row {
		out[i] = f.Key
	}
	return out
}

func TestClient_ExtractSanitizesProviderAnswer(t *testing.T) {
	reply := Extraction{
		Fields: []ExtractedField{
			{Key: "date", Value: "۲۰۲۶-۰۹-۲۰", Confidence: 1.7},
			{Key: "patient_name", Value: "Sara Ahmadi", Confidence: 0.9}, // not asked → dropped
		},
		Items: [][]ExtractedField{
			{{Key: "name", Value: "Hb for Sara Ahmadi 09121234567", Confidence: -1}, {Key: "value", Value: "۱۱٫۴", Confidence: 0.9}},
			{{Key: "value", Value: "high"}}, // not a number → row empty → dropped
			{{Key: "unit", Value: strings.Repeat("x", 900)}},
		},
	}
	spy := &spyExtract{Fake: NewFake(), reply: &reply}
	c := NewClientWith(Options{Provider: "fake", Extractor: spy})
	out, err := c.Extract(userCtx(), FeatureLabAnalysis, ExtractRequest{
		Document: Document{Data: []byte("x"), MIME: "image/webp"}, Schema: labSchema(), Hint: "for Sara Ahmadi",
	})
	require.NoError(t, err)
	assert.Equal(t, "for [name]", spy.got.Hint)
	require.Len(t, out.Fields, 1)
	assert.Equal(t, "2026-09-20", out.Fields[0].Value)
	assert.InDelta(t, 1.0, out.Fields[0].Confidence, 1e-9)
	require.Len(t, out.Items, 2)
	assert.Equal(t, "Hb for [name] [phone]", out.Items[0][0].Value)
	assert.InDelta(t, 0.0, out.Items[0][0].Confidence, 1e-9)
	assert.InDelta(t, 11.4, out.Items[0][1].Value, 1e-9)
	assert.Len(t, []rune(out.Items[1][0].Value.(string)), MaxExtractedRunes)

	bad := []ExtractRequest{
		{Document: Document{MIME: "image/png"}, Schema: labSchema()},
		{Document: Document{Data: []byte("x"), MIME: "text/html"}, Schema: labSchema()},
		{Document: Document{Data: make([]byte, MaxDocumentBytes+1), MIME: "application/pdf"}, Schema: labSchema()},
		{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: ExtractSchema{}},
		{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: ExtractSchema{Fields: []FieldSpec{{Key: "Bad Key", Type: FieldString}}}},
		{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: ExtractSchema{Fields: []FieldSpec{{Key: "k", Type: FieldEnum}}}},
		{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: ExtractSchema{Fields: []FieldSpec{{Key: "k", Type: "blob"}}}},
		{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: ExtractSchema{Fields: []FieldSpec{{Key: "k", Type: FieldString}, {Key: "k", Type: FieldString}}}},
	}
	for i, req := range bad {
		_, err := c.Extract(context.Background(), FeatureLabAnalysis, req)
		require.ErrorIs(t, err, ErrInvalidRequest, i)
	}
}

func TestDocument_Wipe(t *testing.T) {
	d := Document{Data: []byte("secret lab"), MIME: "application/pdf"}
	buf := d.Data
	d.Wipe()
	assert.Nil(t, d.Data)
	assert.Equal(t, make([]byte, len(buf)), buf)
}

func TestNew_WiresAllCapabilities(t *testing.T) {
	lim := &stopLimiter{}
	c := New(Deps{App: config.App{Env: "staging"}, Limiter: lim, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	require.NotNil(t, c)
	assert.False(t, c.External())
	_, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "x"}}})
	require.ErrorIs(t, err, ErrBudgetExceeded, "New passes the limiter through")
	g := New(Deps{App: config.App{Env: "production"}, Config: config.AI{Provider: "gemini", Gemini: config.Gemini{APIKey: "k"}}})
	assert.True(t, g.External())
}

// ---------------------------------------------------------------------------- Gemini over httptest

func sseServer(t *testing.T, status int, events []string, seen *map[string]any, header *http.Header) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta/models/test-model:streamGenerateContent", r.URL.Path)
		assert.Equal(t, "alt=sse", r.URL.RawQuery, "the key never rides in the URL")
		*header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, seen)
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":{"message":"bad things with secret-echo"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			_, _ = fmt.Fprintf(w, "data: %s\r\n\r\n", e)
			w.(http.Flusher).Flush()
		}
	}))
}

func sseChunk(text, finish string, in, out int) string {
	cand := map[string]any{"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": text}}}}
	if finish != "" {
		cand["finishReason"] = finish
	}
	b, _ := json.Marshal(map[string]any{"candidates": []any{cand}, "usageMetadata": map[string]any{"promptTokenCount": in, "candidatesTokenCount": out}})
	return string(b)
}

func TestGemini_ChatStream(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := sseServer(t, 200, []string{sseChunk("Hello ", "", 50, 1), sseChunk("there.", "STOP", 50, 3)}, &seen, &h)
	defer srv.Close()
	rec := &spyRecorder{}
	g := NewGemini(config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	c := NewClientWith(Options{Provider: "gemini", Chatter: g, Recorder: rec,
		Pricer: NewPricer(map[string]config.AIPrice{"test-model": {InputPerMTok: 1, OutputPerMTok: 10, AudioPerMTok: 1}})})
	ch, err := c.Chat(userCtx(), FeatureAssistant, ChatRequest{
		System: "Be kind.",
		Messages: []ChatMessage{
			{Role: RoleUser, Text: "hi, I'm Sara"},
			{Role: RoleAssistant, Text: "Hello!"},
			{Role: RoleUser, Text: "look", Images: []Image{{Data: []byte("img"), MIME: "image/jpeg"}}},
		},
		Language: "fa", MaxOutputTokens: 300,
	})
	require.NoError(t, err)
	text, last := collect(t, ch)
	assert.Equal(t, "Hello there.", text)
	assert.Equal(t, FinishStop, last.Finish)
	assert.Equal(t, "test-key", h.Get("x-goog-api-key"))
	contents := seen["contents"].([]any)
	require.Len(t, contents, 3)
	assert.Equal(t, "model", contents[1].(map[string]any)["role"])
	assert.Equal(t, "hi, I'm [name]", contents[0].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"])
	inline := contents[2].(map[string]any)["parts"].([]any)[1].(map[string]any)["inline_data"].(map[string]any)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("img")), inline["data"])
	assert.Contains(t, mustJSON(t, seen["system_instruction"]), `\"fa\"`)
	assert.InDelta(t, 300, seen["generationConfig"].(map[string]any)["maxOutputTokens"], 0)
	require.Eventually(t, func() bool { return len(rec.got) == 1 }, time.Second, 5*time.Millisecond)
	assert.Equal(t, 50, rec.got[0].InputTokens)
	assert.Equal(t, 3, rec.got[0].OutputTokens)
	assert.Equal(t, uint64(50+30), rec.got[0].CostMicros)
	assert.Equal(t, 3, rec.got[0].ImageBytes)
}

func TestGemini_ChatStreamFinishAndErrors(t *testing.T) {
	var seen map[string]any
	var h http.Header
	run := func(status int, events []string) (string, ChatEvent, error) {
		srv := sseServer(t, status, events, &seen, &h)
		defer srv.Close()
		g := NewGemini(config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv.URL}, srv.Client())
		c := NewClientWith(Options{Provider: "gemini", Chatter: g})
		evs, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "x"}}})
		if err != nil {
			return "", ChatEvent{}, err
		}
		text, last := collect(t, evs)
		return text, last, nil
	}
	_, last, err := run(200, []string{sseChunk("partial", "MAX_TOKENS", 5, 5)})
	require.NoError(t, err)
	assert.Equal(t, FinishLength, last.Finish)
	_, last, err = run(200, []string{`{"promptFeedback":{"blockReason":"SAFETY"}}`})
	require.NoError(t, err)
	assert.Equal(t, FinishSafety, last.Finish)
	_, last, err = run(200, []string{sseChunk("a", "", 1, 1), `{not json`})
	require.NoError(t, err)
	require.ErrorIs(t, last.Err, ErrUpstream)
	_, last, err = run(200, nil)
	require.NoError(t, err)
	require.ErrorIs(t, last.Err, ErrUpstream, "an empty stream is unusable")
	_, _, err = run(500, nil)
	require.ErrorIs(t, err, ErrUpstream)
	assert.NotContains(t, err.Error(), "secret-echo", "error bodies are never echoed")
	assert.NotContains(t, err.Error(), "test-key")
}

func TestGemini_Extract(t *testing.T) {
	var seen map[string]any
	var h http.Header
	answer := `{"fields":[{"key":"date","value":"2026-09-20","confidence":0.9}],"items":[[{"key":"name","value":"TSH","confidence":0.8},{"key":"value","value":2.1,"confidence":0.8}]]}`
	srv := geminiServer(t, 200, answer, &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	c := NewClientWith(Options{Provider: "gemini", Extractor: g})
	out, err := c.Extract(userCtx(), FeatureLabAnalysis, ExtractRequest{
		Document: Document{Data: []byte("%PDF"), MIME: "application/pdf"}, Schema: labSchema(), Language: "fa", Hint: "Sara Ahmadi's CBC",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-09-20", out.Fields[0].Value)
	require.Len(t, out.Items, 1)
	assert.Equal(t, "TSH", out.Items[0][0].Value)
	parts := seen["contents"].([]any)[0].(map[string]any)["parts"].([]any)
	prompt := parts[0].(map[string]any)["text"].(string)
	assert.Contains(t, prompt, "value | number | result")
	assert.Contains(t, prompt, "[name]'s CBC")
	assert.NotContains(t, prompt, "Sara")
	inline := parts[1].(map[string]any)["inline_data"].(map[string]any)
	assert.Equal(t, "application/pdf", inline["mime_type"])
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("%PDF")), inline["data"])
	assert.Equal(t, "application/json", seen["generationConfig"].(map[string]any)["responseMimeType"])

	srv2 := geminiServer(t, 200, "not json", &seen, &h)
	defer srv2.Close()
	g2 := NewGemini(config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv2.URL}, srv2.Client())
	_, _, err = g2.Extract(context.Background(), ExtractRequest{Document: Document{Data: []byte("x"), MIME: "image/png"}, Schema: labSchema()})
	require.ErrorIs(t, err, ErrUpstream)
}

func TestGeminiUsage_AudioTokens(t *testing.T) {
	var m gUsage
	require.NoError(t, json.Unmarshal([]byte(`{"promptTokenCount":900,"candidatesTokenCount":10,
		"promptTokensDetails":[{"modality":"TEXT","tokenCount":100},{"modality":"AUDIO","tokenCount":800}]}`), &m))
	var u Usage
	m.apply(&u)
	assert.Equal(t, 900, u.InputTokens)
	assert.Equal(t, 800, u.AudioTokens)
}

// TestRecorders_FanOut: the slog line and the durable recorder both see the usage; a nil entry is skipped.
func TestRecorders_FanOut(t *testing.T) {
	a, b := &spyRecorder{}, &spyRecorder{}
	Recorders{a, nil, b}.Record(context.Background(), Usage{Op: OpChat})
	assert.Len(t, a.got, 1)
	assert.Len(t, b.got, 1)
}

// TestLogRecorder_NoKeyNoContent: a Gemini failure never puts the key or the request text into logs.
func TestLogRecorder_NoKeyNoContent(t *testing.T) {
	var sb strings.Builder
	var mu sync.Mutex
	logger := slog.New(slog.NewJSONHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return sb.Write(p)
	}), nil))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"API key test-key invalid; text: my belly hurts"}`))
	}))
	defer srv.Close()
	c := New(Deps{App: config.App{Env: "staging"}, Logger: logger, HTTPClient: srv.Client(),
		Config: config.AI{Provider: "gemini", Gemini: config.Gemini{APIKey: "test-key", Model: "test-model", BaseURL: srv.URL}}})
	_, err := c.Extract(context.Background(), FeatureLabAnalysis, ExtractRequest{Document: Document{Data: []byte("my belly hurts"), MIME: "image/png"}, Schema: labSchema(), Hint: "my belly hurts"})
	require.ErrorIs(t, err, ErrUpstream)
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, sb.String(), `"op":"extract"`)
	assert.NotContains(t, sb.String(), "test-key")
	assert.NotContains(t, sb.String(), "belly")
	assert.False(t, errors.Is(err, ErrUnavailable))
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
