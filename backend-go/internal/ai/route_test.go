package ai

// goose 00125: the OpenAI-compatible adapter (ArvanCloud AI gateway) against an httptest server, and the Client's
// model routing (primary, fallback, no route, the route's prices).

import (
	"context"
	"encoding/json"
	"errors"
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

// gateway is a fake OpenAI-compatible server: it records each request and answers with handle.
type gateway struct {
	mu     sync.Mutex
	reqs   []*http.Request
	bodies []map[string]any
	handle func(w http.ResponseWriter, r *http.Request)
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	g.mu.Lock()
	g.reqs = append(g.reqs, r)
	g.bodies = append(g.bodies, body)
	g.mu.Unlock()
	g.handle(w, r)
}

func newCompat(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) (*OpenAICompat, *gateway) {
	t.Helper()
	g := &gateway{handle: handle}
	srv := httptest.NewServer(g)
	t.Cleanup(srv.Close)
	o := NewOpenAICompat(OpenAICompatConfig{Provider: ProviderArvan, BaseURL: srv.URL + "/v1/", APIKey: "secret-key",
		AuthScheme: "apikey", Model: "default-model"}, srv.Client())
	return o, g
}

func completion(content, finish string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"message": map[string]any{"content": content}, "finish_reason": finish}},
		"usage":   map[string]any{"prompt_tokens": 120, "completion_tokens": 30},
	})
	return string(b)
}

func TestOpenAICompat_CompletionAndAuthHeader(t *testing.T) {
	o, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, completion("```json\n{\"items\":[{\"key\":\"sleep.quality\",\"value\":\"good\",\"confidence\":0.8}]}\n```", "stop"))
	})
	ctx := withRoute(context.Background(), Route{Provider: ProviderArvan, Model: "routed-model"})
	cands, u, err := o.ParseLog(ctx, LogParseRequest{Text: "slept well", Language: "en", Vocabulary: vocab()})
	require.NoError(t, err)
	assert.Equal(t, []Candidate{{Key: "sleep.quality", Value: "good", Confidence: 0.8}}, cands)
	assert.Equal(t, ProviderArvan, u.Provider)
	assert.Equal(t, "routed-model", u.Model)
	assert.Equal(t, 120, u.InputTokens)
	assert.Equal(t, 30, u.OutputTokens)

	require.Len(t, g.reqs, 1)
	r := g.reqs[0]
	assert.Equal(t, http.MethodPost, r.Method)
	assert.Equal(t, "/v1/chat/completions", r.URL.Path, "the trailing slash of the base URL is trimmed")
	assert.Equal(t, "apikey secret-key", r.Header.Get("Authorization"))
	assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
	assert.Equal(t, "routed-model", g.bodies[0]["model"], "the route picks the upstream model")
	msgs, _ := g.bodies[0]["messages"].([]any)
	require.Len(t, msgs, 2)
	assert.Equal(t, "system", msgs[0].(map[string]any)["role"])

	// Without a route the configured default model is sent; Bearer is the default scheme.
	plain := NewOpenAICompat(OpenAICompatConfig{Provider: ProviderArvan, BaseURL: "http://x", APIKey: "k"}, nil)
	assert.Equal(t, "Bearer k", plain.auth)
	_, _, err = o.ParseLog(context.Background(), LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.Equal(t, "default-model", g.bodies[1]["model"])
}

func TestOpenAICompat_UpstreamErrors(t *testing.T) {
	status := http.StatusInternalServerError
	body := `{"error":{"message":"echo of the request: secret"}}`
	o, _ := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
	_, _, err := o.ParseLog(context.Background(), LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrUpstream)
	assert.NotContains(t, err.Error(), "secret", "error bodies never reach the error")
	assert.Contains(t, err.Error(), "status 500")

	_, err = o.ChatStream(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.ErrorIs(t, err, ErrUpstream)

	_, err = o.ListModels(context.Background())
	require.ErrorIs(t, err, ErrUpstream)

	// 200 but unusable answers.
	status = http.StatusOK
	for _, b := range []string{`not json`, `{"choices":[]}`, completion("x", "content_filter"), completion("not json either", "stop")} {
		body = b
		_, _, err = o.ParseLog(context.Background(), LogParseRequest{Text: "x"})
		require.ErrorIs(t, err, ErrUpstream, b)
	}
}

func TestOpenAICompat_RefusesPDFBeforeAnyRequest(t *testing.T) {
	o, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, completion(`{"fields":[],"items":[]}`, "stop"))
	})
	_, _, err := o.Extract(context.Background(), ExtractRequest{Document: Document{Data: []byte("%PDF-1.4"), MIME: "application/pdf"}})
	require.ErrorIs(t, err, ErrUpstream, "a fallback model that reads PDFs may take the call")
	assert.Empty(t, g.reqs, "nothing was sent")

	_, _, err = o.Extract(context.Background(), ExtractRequest{Document: Document{Data: []byte("\x89PNG...."), MIME: "image/png"},
		Schema: ExtractSchema{Name: "lab"}})
	require.NoError(t, err)
	require.Len(t, g.reqs, 1)
	msgs := g.bodies[0]["messages"].([]any)
	parts := msgs[1].(map[string]any)["content"].([]any)
	require.Len(t, parts, 2)
	img := parts[1].(map[string]any)
	assert.Equal(t, "image_url", img["type"])
	assert.True(t, strings.HasPrefix(img["image_url"].(map[string]any)["url"].(string), "data:image/png;base64,"))
}

func sse(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, l := range lines {
		_, _ = io.WriteString(w, l+"\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func drainChunks(t *testing.T, ch <-chan ChatChunk) (string, ChatChunk) {
	t.Helper()
	var sb strings.Builder
	var last ChatChunk
	for c := range ch {
		sb.WriteString(c.Delta)
		last = c
	}
	return sb.String(), last
}

func TestOpenAICompat_ChatStream(t *testing.T) {
	o, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		sse(w,
			`: keep-alive`,
			`data: {"choices":[{"delta":{"content":"سلام"}}]}`,
			`data: {"choices":[{"delta":{"content":" دنیا"},"finish_reason":"stop"}]}`,
			`data: {"choices":[],"usage":{"prompt_tokens":42,"completion_tokens":7}}`,
			`data: [DONE]`,
			`data: {"choices":[{"delta":{"content":"after done"}}]}`,
		)
	})
	ch, err := o.ChatStream(context.Background(), ChatRequest{System: "be kind", Language: "fa", MaxOutputTokens: 100,
		Messages: []ChatMessage{
			{Role: RoleUser, Text: "hi", Images: []Image{{Data: []byte("\x89PNG"), MIME: "image/png"}}},
			{Role: RoleAssistant, Text: ""},
		}})
	require.NoError(t, err)
	text, last := drainChunks(t, ch)
	assert.Equal(t, "سلام دنیا", text, "nothing after [DONE]")
	assert.True(t, last.Done)
	assert.Equal(t, FinishStop, last.Finish)
	assert.Equal(t, 42, last.Usage.InputTokens, "usage arrives in the last chunk")
	assert.Equal(t, 7, last.Usage.OutputTokens)
	assert.Equal(t, "default-model", last.Usage.Model)

	require.Len(t, g.reqs, 1)
	assert.Equal(t, "text/event-stream", g.reqs[0].Header.Get("Accept"))
	assert.Equal(t, "apikey secret-key", g.reqs[0].Header.Get("Authorization"))
	b := g.bodies[0]
	assert.Equal(t, true, b["stream"])
	assert.Equal(t, map[string]any{"include_usage": true}, b["stream_options"])
	assert.InDelta(t, 100, b["max_tokens"], 0)
	msgs := b["messages"].([]any)
	require.Len(t, msgs, 3)
	assert.Contains(t, msgs[0].(map[string]any)["content"], "be kind")
	parts := msgs[1].(map[string]any)["content"].([]any)
	assert.Equal(t, "text", parts[0].(map[string]any)["type"])
	assert.Equal(t, "image_url", parts[1].(map[string]any)["type"])
	assert.Equal(t, "assistant", msgs[2].(map[string]any)["role"])
	assert.Equal(t, " ", msgs[2].(map[string]any)["content"], "an empty turn is sent as a space")
}

func TestOpenAICompat_ChatStreamFailures(t *testing.T) {
	answer := []string{`data: [DONE]`}
	o, _ := newCompat(t, func(w http.ResponseWriter, _ *http.Request) { sse(w, answer...) })
	ch, err := o.ChatStream(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	_, last := drainChunks(t, ch)
	require.ErrorIs(t, last.Err, ErrUpstream, "an empty stream is a failure")

	answer = []string{`data: {"choices":[{"delta":{"content":"x"}}]}`, `data: {broken`}
	ch, err = o.ChatStream(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	_, last = drainChunks(t, ch)
	require.ErrorIs(t, last.Err, ErrUpstream)

	// No finish reason but text: a normal end.
	answer = []string{`data: {"choices":[{"delta":{"content":"x"}}]}`}
	ch, err = o.ChatStream(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	_, last = drainChunks(t, ch)
	assert.True(t, last.Done)
	assert.Equal(t, FinishStop, last.Finish)

	answer = []string{`data: {"choices":[{"delta":{"content":"x"},"finish_reason":"content_filter"}]}`}
	ch, err = o.ChatStream(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	_, last = drainChunks(t, ch)
	assert.Equal(t, FinishSafety, last.Finish)
	assert.Equal(t, FinishLength, oaFinish("length"))
}

func TestOpenAICompat_ListModels(t *testing.T) {
	o, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"zeta"},{"id":""},{"id":"alpha"}]}`)
	})
	ids, err := o.ListModels(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "zeta"}, ids)
	require.Len(t, g.reqs, 1)
	assert.Equal(t, http.MethodGet, g.reqs[0].Method)
	assert.Equal(t, "/v1/models", g.reqs[0].URL.Path)
	assert.Equal(t, "apikey secret-key", g.reqs[0].Header.Get("Authorization"))

	// The Client lists only through a configured adapter that can.
	c := NewClientWith(Options{Provider: ProviderArvan, Adapters: map[string]Adapter{ProviderArvan: {Lister: o}, ProviderGemini: {}}})
	ids, err = c.ProviderModels(context.Background(), ProviderArvan)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	_, err = c.ProviderModels(context.Background(), ProviderGemini)
	require.ErrorIs(t, err, ErrUnavailable)
	assert.Equal(t, []string{ProviderArvan, ProviderGemini}, c.ConfiguredProviders())
}

// stubParser is a provider adapter that records the model each call was routed to.
type stubParser struct {
	name  string
	err   error
	mu    sync.Mutex
	calls []string
}

func (s *stubParser) ParseLog(ctx context.Context, _ LogParseRequest) ([]Candidate, Usage, error) {
	s.mu.Lock()
	s.calls = append(s.calls, routeModel(ctx, "default"))
	s.mu.Unlock()
	if s.err != nil {
		return nil, Usage{}, s.err
	}
	return []Candidate{{Key: s.name}}, Usage{InputTokens: 1000, OutputTokens: 1000}, nil
}

type stubRouter struct {
	routes []Route
	err    error
}

func (r stubRouter) Routes(context.Context, Feature) ([]Route, error) { return r.routes, r.err }

type usageLog struct {
	mu sync.Mutex
	u  []Usage
}

func (l *usageLog) Record(_ context.Context, u Usage) {
	l.mu.Lock()
	l.u = append(l.u, u)
	l.mu.Unlock()
}

func routedClient(router Router, adapters map[string]Adapter, rec Recorder) *Client {
	return NewClientWith(Options{Provider: ProviderGemini, Router: router, Adapters: adapters, Recorder: rec,
		Pricer: NewPricer(map[string]config.AIPrice{})})
}

func TestClient_RoutingPrimaryAndFallback(t *testing.T) {
	primary := Route{Provider: ProviderArvan, Model: "arvan-m", Priced: true, InputPriceMilli: 300, OutputPriceMilli: 2500}
	fallback := Route{Provider: ProviderGemini, Model: "gemini-m", Priced: true, InputPriceMilli: 100, OutputPriceMilli: 400}
	arvan, gemini := &stubParser{name: "arvan"}, &stubParser{name: "gemini"}
	rec := &usageLog{}
	c := routedClient(stubRouter{routes: []Route{primary, fallback}},
		map[string]Adapter{ProviderArvan: {LogParser: arvan}, ProviderGemini: {LogParser: gemini}}, rec)

	// The primary route serves the call; its model and prices are used.
	out, err := c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.Equal(t, "arvan", out[0].Key)
	assert.Equal(t, []string{"arvan-m"}, arvan.calls)
	assert.Empty(t, gemini.calls)
	require.Len(t, rec.u, 1)
	assert.Equal(t, ProviderArvan, rec.u[0].Provider)
	assert.Equal(t, "arvan-m", rec.u[0].Model)
	assert.Equal(t, uint64(2800), rec.u[0].CostMicros, "1000 × 300 + 1000 × 2500 milli-USD per 1M tokens")
	assert.True(t, rec.u[0].OK)

	// The primary fails upstream: the fallback answers, both calls are logged.
	arvan.err = fmt.Errorf("%w: status 502", ErrUpstream)
	out, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.Equal(t, "gemini", out[0].Key)
	assert.Equal(t, []string{"gemini-m"}, gemini.calls)
	require.Len(t, rec.u, 3)
	assert.False(t, rec.u[1].OK)
	assert.Equal(t, ProviderGemini, rec.u[2].Provider)
	assert.Equal(t, uint64(500), rec.u[2].CostMicros, "the fallback's own prices")

	// Not an upstream failure (an invalid request would fail there too): no fallback.
	arvan.err = ErrInvalidRequest
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Len(t, gemini.calls, 1)

	// Both fail: the last error is returned.
	arvan.err, gemini.err = fmt.Errorf("%w: a", ErrUpstream), fmt.Errorf("%w: b", ErrUpstream)
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, ErrUpstream)
	assert.Contains(t, err.Error(), "b")
}

func TestClient_RoutingMissingAdapterAndNoRoutes(t *testing.T) {
	gemini := &stubParser{name: "gemini"}
	routes := []Route{{Provider: ProviderArvan, Model: "arvan-m"}, {Provider: ProviderGemini, Model: "gemini-m"}}
	// The primary's provider has no key on this server: the fallback serves.
	c := routedClient(stubRouter{routes: routes}, map[string]Adapter{ProviderGemini: {LogParser: gemini}}, nil)
	out, err := c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.Equal(t, "gemini", out[0].Key)

	// An adapter without the capability is skipped too.
	c = routedClient(stubRouter{routes: routes}, map[string]Adapter{ProviderArvan: {}, ProviderGemini: {LogParser: gemini}}, nil)
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.NoError(t, err)
	assert.Len(t, gemini.calls, 2)

	// No route (feature off / no model), or none with a configured adapter: ErrUnavailable, nothing called.
	for _, r := range []stubRouter{{}, {routes: []Route{{Provider: ProviderArvan, Model: "m"}}}} {
		c = routedClient(r, map[string]Adapter{ProviderGemini: {LogParser: gemini}}, nil)
		_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
		require.ErrorIs(t, err, ErrUnavailable)
	}
	assert.Len(t, gemini.calls, 2)

	// A router error passes through.
	boom := errors.New("catalogue down")
	c = routedClient(stubRouter{err: boom}, map[string]Adapter{ProviderGemini: {LogParser: gemini}}, nil)
	_, err = c.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x"})
	require.ErrorIs(t, err, boom)

	// The fake bypasses routing: fixtures answer for every feature.
	f := NewFake()
	fake := NewClientWith(Options{Provider: config.AIProviderFake, LogParser: f, Router: stubRouter{}})
	_, err = fake.ParseLog(context.Background(), FeatureVoiceLog, LogParseRequest{Text: "x", Vocabulary: vocab()})
	require.NoError(t, err)
}

func TestClient_ChatFallsBackBeforeStreaming(t *testing.T) {
	failing, _ := newCompat(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	working, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		sse(w, `data: {"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`, `data: [DONE]`)
	})
	rec := &usageLog{}
	c := NewClientWith(Options{Provider: ProviderGemini, Recorder: rec, Pricer: NewPricer(map[string]config.AIPrice{}),
		Router: stubRouter{routes: []Route{
			{Provider: ProviderGemini, Model: "primary", Priced: true},
			{Provider: ProviderArvan, Model: "fallback", Priced: true, InputPriceMilli: 1000, OutputPriceMilli: 1000},
		}},
		Adapters: map[string]Adapter{ProviderGemini: {Chatter: failing}, ProviderArvan: {Chatter: working}}})
	st, err := c.Chat(context.Background(), FeatureAssistant, ChatRequest{Messages: []ChatMessage{{Role: RoleUser, Text: "hi"}}})
	require.NoError(t, err)
	var text strings.Builder
	for ev := range st.Events {
		require.NoError(t, ev.Err)
		text.WriteString(ev.Delta)
	}
	assert.Equal(t, "ok", text.String())
	assert.Equal(t, "fallback", g.bodies[0]["model"])
	require.Eventually(t, func() bool { rec.mu.Lock(); defer rec.mu.Unlock(); return len(rec.u) == 2 }, time.Second, 10*time.Millisecond)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, "primary", rec.u[0].Model)
	assert.False(t, rec.u[0].OK)
	assert.Equal(t, "fallback", rec.u[1].Model)
	assert.Equal(t, ProviderArvan, rec.u[1].Provider)
	assert.True(t, rec.u[1].OK)
}
