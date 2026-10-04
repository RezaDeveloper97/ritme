package ai

// B-N6-05b security fixes: stream cost is never lost (cumulative usage, conservative estimate), the Client owns
// a bounded, cancellable stream context, Gemini thinking tokens are billed and turned off for task calls, the
// per-user budget, magic-byte sniffing and limits, identity keys in schemas, and the hardened PII filter.
// Gemini runs against httptest servers only; nothing here touches the network.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
)

// lockedRecorder is a goroutine-safe spy recorder.
type lockedRecorder struct {
	mu  sync.Mutex
	got []Usage
}

func (r *lockedRecorder) Record(_ context.Context, u Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, u)
}

func (r *lockedRecorder) all() []Usage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Usage(nil), r.got...)
}

var testPrices = map[string]config.AIPrice{"test-model": {InputPerMTok: 1, OutputPerMTok: 10, AudioPerMTok: 1}}

// hangingSSE streams the given events, then keeps the connection open without a byte until the client leaves.
func hangingSSE(t *testing.T, events []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for _, e := range events {
			_, _ = fmt.Fprintf(w, "data: %s\r\n\r\n", e)
			w.(http.Flusher).Flush()
		}
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
}

// H1: a stream cancelled mid-way records the last cumulative usage the provider reported, never zero.
func TestChat_CancelMidStreamRecordsReportedUsage(t *testing.T) {
	srv := hangingSSE(t, []string{sseChunk("Hello ", "", 120, 7)})
	defer srv.Close()
	rec := &lockedRecorder{}
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	c := NewClientWith(Options{Provider: "gemini", Chatter: g, Recorder: rec, Pricer: NewPricer(testPrices)})
	st, err := c.Chat(userCtx(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	ev := <-st.Events
	assert.Equal(t, "Hello ", ev.Delta)
	st.Close() // the browser went away
	st.Wait()
	got := rec.all()
	require.Len(t, got, 1)
	assert.False(t, got[0].OK)
	assert.Equal(t, 120, got[0].InputTokens, "the last cumulative usageMetadata is kept")
	assert.Equal(t, 7, got[0].OutputTokens)
	assert.Equal(t, uint64(120+70), got[0].CostMicros)
	assert.Equal(t, uint64(42), got[0].UserID)
}

// H1: without any usageMetadata, an interrupted stream is recorded with a conservative estimate (never 0).
func TestChat_MissingUsageIsEstimated(t *testing.T) {
	noUsage := func(text string) string {
		b, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": text}}}}}})
		return string(b)
	}
	srv := hangingSSE(t, []string{noUsage("abcdefghij")})
	defer srv.Close()
	rec := &lockedRecorder{}
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	c := NewClientWith(Options{Provider: "gemini", Chatter: g, Recorder: rec, Pricer: NewPricer(testPrices)})
	req := ChatRequest{System: strings.Repeat("s", 100), Messages: []ChatMessage{{Role: RoleUser, Text: strings.Repeat("q", 100)}}}
	st, err := c.Chat(context.Background(), FeatureAssistant, req)
	require.NoError(t, err)
	<-st.Events
	st.Close()
	st.Wait()
	got := rec.all()
	require.Len(t, got, 1)
	assert.True(t, got[0].Estimated)
	assert.GreaterOrEqual(t, got[0].InputTokens, 100, "≥ one token per two characters of the request")
	assert.Equal(t, 5, got[0].OutputTokens, "the delivered text, one token per two characters")
	assert.Positive(t, got[0].CostMicros)

	// cancelled before the first chunk: the input estimate alone, still never free
	srv2 := hangingSSE(t, nil)
	defer srv2.Close()
	rec2 := &lockedRecorder{}
	g2 := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv2.URL}, srv2.Client())
	c2 := NewClientWith(Options{Provider: "gemini", Chatter: g2, Recorder: rec2, Pricer: NewPricer(testPrices)})
	st, err = c2.Chat(context.Background(), FeatureAssistant, req)
	require.NoError(t, err)
	st.Close()
	st.Wait()
	got = rec2.all()
	require.Len(t, got, 1)
	assert.Positive(t, got[0].InputTokens)
	assert.Positive(t, got[0].CostMicros)
}

func TestChatUsage(t *testing.T) {
	reported := Usage{InputTokens: 50, OutputTokens: 3}
	assert.Equal(t, reported, chatUsage(reported, 999, 999, true), "a finished stream keeps the provider's numbers")
	u := chatUsage(reported, 999, 40, false)
	assert.Equal(t, 50, u.InputTokens)
	assert.Equal(t, 20, u.OutputTokens, "interrupted: at least what was delivered")
	assert.True(t, u.Estimated)
	u = chatUsage(Usage{}, 30, 0, true)
	assert.Equal(t, 30, u.InputTokens, "done without metadata → estimate")
	assert.True(t, u.Estimated)
	assert.Equal(t, 1, estimateChatInput(ChatRequest{Messages: []ChatMessage{{Role: RoleUser}}}))
	assert.Equal(t, 2+258, estimateChatInput(ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "abc", Images: []Image{{}}}}}))
}

// H2: the stream ends by itself after the Client's max duration even when nobody cancels and the consumer stops
// reading; the provider request is aborted and the usage recorded.
func TestChat_MaxDurationEndsStream(t *testing.T) {
	srv := hangingSSE(t, []string{sseChunk("a", "", 10, 1)})
	defer srv.Close()
	rec := &lockedRecorder{}
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	c := NewClientWith(Options{Provider: "gemini", Chatter: g, Recorder: rec, ChatMaxDuration: 150 * time.Millisecond})
	st, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "x"}}})
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { st.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the stream outlived ChatMaxDuration")
	}
	require.Len(t, rec.all(), 1)
	for range st.Events { //nolint:revive // drained: the channel is closed
	}
	st.Close() // idempotent after the end
	st.Close()
}

// H2: a silent upstream (no line for longer than the idle timeout) fails the stream with ErrUpstream.
func TestGemini_StreamIdleTimeout(t *testing.T) {
	srv := hangingSSE(t, []string{sseChunk("a", "", 10, 1)})
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client()).WithStreamIdleTimeout(100 * time.Millisecond)
	c := NewClientWith(Options{Provider: "gemini", Chatter: g})
	st, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "x"}}})
	require.NoError(t, err)
	text, last := collect(t, st)
	assert.Equal(t, "a", text)
	require.ErrorIs(t, last.Err, ErrUpstream)
}

// H2: the provider HTTP client has transport timeouts but no total timeout (that would cut every stream).
func TestProviderHTTPClient(t *testing.T) {
	hc := ProviderHTTPClient(7 * time.Second)
	assert.Zero(t, hc.Timeout)
	tr, ok := hc.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Equal(t, 7*time.Second, tr.ResponseHeaderTimeout)
	assert.Positive(t, tr.TLSHandshakeTimeout)
	assert.Positive(t, tr.IdleConnTimeout)
	require.Error(t, hc.CheckRedirect(nil, nil), "redirects are never followed")
}

// H2: a non-streaming call is bounded by its own timeout.
func TestGemini_CallTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body) // then the server notices the client leaving
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client()).WithCallTimeout(100 * time.Millisecond)
	start := time.Now()
	_, _, err := g.ParseLog(context.Background(), LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrUpstream)
	assert.Less(t, time.Since(start), 3*time.Second)
}

// M1: thinking and tool-use prompt tokens are billed; whatever totalTokenCount holds beyond the parts is output.
func TestGeminiUsage_ThoughtsAndToolUse(t *testing.T) {
	var m gUsage
	require.NoError(t, json.Unmarshal([]byte(`{"promptTokenCount":100,"candidatesTokenCount":10,"thoughtsTokenCount":900,
		"toolUsePromptTokenCount":20,"totalTokenCount":1030}`), &m))
	var u Usage
	m.apply(&u)
	assert.Equal(t, 120, u.InputTokens)
	assert.Equal(t, 910, u.OutputTokens)
	var m2 gUsage
	require.NoError(t, json.Unmarshal([]byte(`{"promptTokenCount":100,"candidatesTokenCount":10,"totalTokenCount":200}`), &m2))
	m2.apply(&u)
	assert.Equal(t, 100, u.OutputTokens, "unknown remainder billed at the output rate")
}

// M1: parse / extract / transcribe turn thinking off (pro: its minimum) and cap the output.
func TestGemini_TaskCallsDisableThinking(t *testing.T) {
	var seen map[string]any
	var h http.Header
	srv := geminiServer(t, 200, `{"items":[]}`, &seen, &h)
	defer srv.Close()
	g := NewGemini(config.Gemini{APIKey: "k", Model: "test-model", BaseURL: srv.URL}, srv.Client())
	gc := func() map[string]any { return seen["generationConfig"].(map[string]any) }

	_, _, err := g.ParseLog(context.Background(), LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.InDelta(t, 0, gc()["thinkingConfig"].(map[string]any)["thinkingBudget"], 0)
	assert.InDelta(t, GeminiParseMaxOutputTokens, gc()["maxOutputTokens"], 0)

	_, _, err = g.Transcribe(context.Background(), TranscribeRequest{Audio: audio(""), Language: "fa"})
	require.NoError(t, err)
	assert.InDelta(t, 0, gc()["thinkingConfig"].(map[string]any)["thinkingBudget"], 0)
	assert.InDelta(t, GeminiTranscribeMaxOutputTokens, gc()["maxOutputTokens"], 0)

	_, _, _ = g.Extract(context.Background(), ExtractRequest{Document: Document{Data: []byte(pngMagic), MIME: "image/png"}, Schema: labSchema()})
	assert.InDelta(t, 0, gc()["thinkingConfig"].(map[string]any)["thinkingBudget"], 0)
	assert.InDelta(t, GeminiExtractMaxOutputTokens, gc()["maxOutputTokens"], 0)

	pro := NewGemini(config.Gemini{Model: "gemini-2.5-pro"}, nil)
	assert.Equal(t, 128, pro.thinkingBudget(), "pro models cannot turn thinking off")
}

// userLimiter is a Limiter + UserLimiter spy.
type userLimiter struct{ blocked map[uint64]bool }

func (userLimiter) Allow(context.Context) error { return nil }
func (u userLimiter) AllowUser(_ context.Context, id uint64) error {
	if u.blocked[id] {
		return ErrUserBudgetExceeded
	}
	return nil
}

// M2: the Client checks the user's own cap when the context names her.
func TestClient_UserBudget(t *testing.T) {
	f := NewFake()
	c := NewClientWith(Options{Provider: "fake", Transcriber: f, LogParser: f, Chatter: f, Extractor: f,
		Limiter: userLimiter{blocked: map[uint64]bool{42: true}}})
	_, err := c.ParseLog(userCtx(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrUserBudgetExceeded)
	_, err = c.Chat(userCtx(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "x"}}})
	require.ErrorIs(t, err, ErrUserBudgetExceeded)
	other := WithSubject(context.Background(), Subject{UserID: 7})
	_, err = c.ParseLog(other, FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.NoError(t, err)
}

// L3: magic-byte sniffing and limits that fit the 25 MB request body.
func TestSniffAndLimits(t *testing.T) {
	assert.Equal(t, "image/jpeg", SniffImage([]byte("\xff\xd8\xff\xe0JFIF")))
	assert.Equal(t, "image/png", SniffImage([]byte(pngMagic+"IHDR")))
	assert.Equal(t, "image/webp", SniffImage([]byte("RIFF\x10\x00\x00\x00WEBPVP8 ")))
	assert.Empty(t, SniffImage([]byte("RIFF\x10\x00\x00\x00WAVEfmt ")), "a WAV is not an image")
	assert.Empty(t, SniffImage([]byte("<svg onload=alert(1)>")))
	assert.Empty(t, SniffImage([]byte("GIF89a")))
	assert.Empty(t, SniffImage(nil))
	assert.Equal(t, "application/pdf", SniffDocument([]byte("%PDF-1.7\n")))
	assert.Equal(t, "application/pdf", SniffDocument([]byte("\xef\xbb\xbf%PDF-1.4")))
	assert.Equal(t, "image/png", SniffDocument([]byte(pngMagic)))
	assert.Empty(t, SniffDocument([]byte("PK\x03\x04 docx")))
	assert.Empty(t, SniffDocument([]byte("<html>%PDF-")[:6]))

	const multipartSlack = 1 << 20
	assert.Less(t, MaxDocumentBytes+multipartSlack, RequestBodyLimit)
	assert.LessOrEqual(t, MaxChatImageBytes, MaxChatImages*MaxImageBytes)
	assert.Less(t, MaxChatImageBytes+4*MaxChatTextRunes+multipartSlack, RequestBodyLimit, "photos + text of one chat request")
}

// M4: schemas may not ask for the person.
func TestIdentityKey(t *testing.T) {
	for _, k := range []string{"name", "name_fa", "patient", "patient_name", "patient_id", "first_name", "full_name",
		"family_name", "surname", "national_id", "nid", "phone", "mobile_number", "address", "email", "insurance_number",
		"father_name", "dob", "birthdate"} {
		assert.True(t, identityKey(k), k)
	}
	for _, k := range []string{"marker", "marker_name", "test_name", "lab_name", "centre", "doctor", "value", "unit",
		"date", "ga_weeks", "findings", "drug_name"} {
		assert.False(t, identityKey(k), k)
	}
}

// M4: normalisation, separator-tolerant long numbers, Arabic letter forms and Latin ↔ Persian names.
func TestRedact_Hardened(t *testing.T) {
	names := []string{"Fatemeh Hosseini", "مریم کریمی"}
	cases := []struct{ in, want string }{
		// separators inside long numbers (cards, ids, postal codes, landlines)
		{"card 6037 9912 3456 7890 ok", "card [number] ok"},
		{"کارت ۶۰۳۷-۹۹۱۲-۳۴۵۶-۷۸۹۰", "کارت [number]"},
		{"کد پستی ۱۲۳۴۵_۶۷۸۹۰", "کد پستی [number]"},
		{"tel 021 8888 7777", "tel [number]"},
		{"mixed ۰۹۱۲1234567", "mixed [phone]"},
		{"zwnj ۰۹۱۲\u200c۱۲۳\u200c۴۵۶۷", "zwnj [phone]"},
		{"zwsp 0012\u200b345678", "zwsp [number]"},
		{"rlm 6037\u200f9912\u200f3456\u200f7890", "rlm [number]"},
		{"id ۰۰۱-۲۳۴۵۶۷-۸", "id [id]"},
		// kept: health values, dates (also next to another number), short runs
		{"BP 120 80, pulse 72", "BP 120 80, pulse 72"},
		{"دمای ۳۶.۵ و ۳۶.۷ و ۳۶.۶", "دمای ۳۶.۵ و ۳۶.۷ و ۳۶.۶"},
		{"2026-09-20 1403/07/12", "2026-09-20 1403/07/12"},
		{"۱۴۰۳/۰۷/۱۲ ساعت ۱۰:۳۰", "۱۴۰۳/۰۷/۱۲ ساعت ۱۰:۳۰"},
		// names: Arabic letter forms, ZWNJ / spaces between the words
		{"اسمم مريم كريمي است", "اسمم [name] است"},
		{"مریم\u200cکریمی هستم", "[name] هستم"},
		{"FATEMEH hosseini", "[name]"},
		// names across scripts
		{"من فاطمه حسینی هستم", "من [name] [name] هستم"},
		{"فاطمه\u200cام", "[name]\u200cام"},
		{"I'm Maryam Karimi", "I'm [name] [name]"},
		{"Fatema called", "Fatema called"}, // same-script spelling variants are not matched (only literal + cross-script)
		// other words are kept
		{"درد شکم و سر درد دارم", "درد شکم و سر درد دارم"},
		{"mild cramps today", "mild cramps today"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, Redact(tc.in, names), tc.in)
	}
	// short names are matched literally only (their skeletons collide with common words)
	assert.Equal(t, "سر درد دارم، [name] هستم", Redact("سر درد دارم، Sara هستم", []string{"Sara"}))
}
