package ai

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/platform/config"
)

// Recorder receives the Usage of every call: the slog line (LogRecorder) and the ai_usage_logs row
// (internal/ai/usage.Recorder). Record must not fail the call; it gets a context that outlives the request.
type Recorder interface {
	Record(ctx context.Context, u Usage)
}

// Limiter is the global spend guard (internal/ai/usage.Budget): Allow returns ErrBudgetExceeded once the daily
// cost cap is spent — and also when the spend cannot be read (fail closed).
type Limiter interface {
	Allow(ctx context.Context) error
}

// UserLimiter is the per-user spend guard (B-N6-05b, internal/ai/usage.Budget): AllowUser returns
// ErrUserBudgetExceeded once the user's daily cost cap is spent — and also when her spend cannot be read (fail
// closed). The Client checks it when its Limiter implements it and the context carries a Subject.
type UserLimiter interface {
	AllowUser(ctx context.Context, userID uint64) error
}

// LogRecorder writes one structured slog line per call: provider, model, feature, op, token counts, sizes,
// cost, latency and outcome. Never the payload, never who asked.
type LogRecorder struct{ Logger *slog.Logger }

// Record implements Recorder.
func (r LogRecorder) Record(ctx context.Context, u Usage) {
	if r.Logger == nil {
		return
	}
	r.Logger.LogAttrs(ctx, slog.LevelInfo, "ai: call",
		slog.String("provider", u.Provider), slog.String("model", u.Model), slog.String("feature", string(u.Feature)),
		slog.String("op", u.Op), slog.Int("input_tokens", u.InputTokens), slog.Int("output_tokens", u.OutputTokens),
		slog.Int("audio_bytes", u.AudioBytes), slog.Int("image_bytes", u.ImageBytes), slog.Uint64("cost_micros", u.CostMicros),
		slog.Int64("latency_ms", u.Latency.Milliseconds()), slog.Bool("ok", u.OK))
}

// Recorders fans a usage out to several recorders.
type Recorders []Recorder

// Record implements Recorder.
func (rs Recorders) Record(ctx context.Context, u Usage) {
	for _, r := range rs {
		if r != nil {
			r.Record(ctx, u)
		}
	}
}

// Client is the AI layer a domain holds: the configured provider's capabilities plus budget, PII filter and
// usage metering.
type Client struct {
	provider string
	// def is the default adapter: every call without a Router goes to it (tests, tools, the fake).
	def adapter
	// adapters are the real providers by id (ProviderGemini, ProviderArvan); the Router picks among them.
	adapters map[string]adapter
	router   Router
	recorder Recorder
	limiter  Limiter
	pricer   Pricer
	chatMax  time.Duration
	now      func() time.Time
}

// adapter is one provider's capabilities (nil = it cannot do that).
type adapter struct {
	transcriber Transcriber
	logParser   LogParser
	chatter     Chatter
	extractor   Extractor
	lister      ModelLister
}

// ModelLister is a provider that can say which models it serves (the admin catalogue sync).
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// ConfiguredProviders are the real adapters this process has a key for, sorted.
func (c *Client) ConfiguredProviders() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.adapters))
	for id := range c.adapters {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// ProviderModels asks provider which models it serves. ErrUnavailable when the adapter is not configured or
// cannot list its models.
func (c *Client) ProviderModels(ctx context.Context, provider string) ([]string, error) {
	if c == nil {
		return nil, ErrUnavailable
	}
	a, ok := c.adapters[provider]
	if !ok || a.lister == nil {
		return nil, ErrUnavailable
	}
	return a.lister.ListModels(ctx)
}

// candidate is one adapter to try for a call, with the context that carries its route.
type candidate struct {
	ctx context.Context //nolint:containedctx // the route travels to the adapter on the context
	a   adapter
}

// candidates are the adapters to try for feature, best first: the default adapter without a Router (or while the
// provider is the fake — fixtures replace every model), else the feature's routes whose adapter is configured.
func (c *Client) candidates(ctx context.Context, feature Feature) ([]candidate, error) {
	if c.router == nil || c.provider == config.AIProviderFake {
		return []candidate{{ctx: ctx, a: c.def}}, nil
	}
	routes, err := c.router.Routes(ctx, feature)
	if err != nil {
		return nil, err
	}
	out := make([]candidate, 0, len(routes))
	for _, r := range routes {
		if a, ok := c.adapters[r.Provider]; ok {
			out = append(out, candidate{ctx: withRoute(ctx, r), a: a})
		}
	}
	if len(out) == 0 {
		return nil, ErrUnavailable
	}
	return out, nil
}

// tryNext reports whether a failed call may go on to the next candidate: only a provider failure does — an
// invalid request or a spent budget would fail there too.
func tryNext(err error, i, n int) bool {
	return errors.Is(err, ErrUpstream) && i < n-1
}

// Options wires a Client from capabilities (tests and New). A nil capability answers ErrUnavailable.
type Options struct {
	Provider    string
	Transcriber Transcriber
	LogParser   LogParser
	Chatter     Chatter
	Extractor   Extractor
	Recorder    Recorder
	Limiter     Limiter // nil = no cap (tests); New always sets the DB budget when one is given in Deps
	Pricer      Pricer
	// Router picks the model of each feature (route.go); nil = the default adapter with its configured model.
	Router Router
	// Adapters are the real providers the Router may name, by provider id.
	Adapters map[string]Adapter
	// ChatMaxDuration overrides MaxChatDuration (tests).
	ChatMaxDuration time.Duration
}

// Adapter is a real provider's capabilities as Options take them (nil = it cannot do that).
type Adapter struct {
	Transcriber Transcriber
	LogParser   LogParser
	Chatter     Chatter
	Extractor   Extractor
	Lister      ModelLister
}

// NewClientWith builds a client from o.
func NewClientWith(o Options) *Client {
	c := &Client{provider: o.Provider, recorder: o.Recorder, limiter: o.Limiter, pricer: o.Pricer,
		chatMax: o.ChatMaxDuration, router: o.Router, adapters: map[string]adapter{}, now: time.Now,
		def: adapter{transcriber: o.Transcriber, logParser: o.LogParser, chatter: o.Chatter, extractor: o.Extractor}}
	for id, a := range o.Adapters {
		c.adapters[id] = adapter{transcriber: a.Transcriber, logParser: a.LogParser, chatter: a.Chatter, extractor: a.Extractor,
			lister: a.Lister}
	}
	return c
}

// NewClient wires a speech + parse client (B-N3-05 tests; production code uses New).
func NewClient(provider string, t Transcriber, p LogParser, rec Recorder) *Client {
	return NewClientWith(Options{Provider: provider, Transcriber: t, LogParser: p, Recorder: rec})
}

// Provider is the configured provider id ("fake", "gemini"); "" for a nil client.
func (c *Client) Provider() string {
	if c == nil {
		return ""
	}
	return c.provider
}

// External reports whether calls leave the server (a real provider, not the in-process fake).
func (c *Client) External() bool {
	return c != nil && c.provider != config.AIProviderFake
}

// allow checks the global daily budget, then the user's own (when the limiter has one and ctx names the user).
func (c *Client) allow(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	if err := c.limiter.Allow(ctx); err != nil {
		return err
	}
	if ul, ok := c.limiter.(UserLimiter); ok {
		if id := SubjectFrom(ctx).UserID; id > 0 {
			return ul.AllowUser(ctx, id)
		}
	}
	return nil
}

// Transcribe runs speech-to-text for feature. A nil client or a provider without the capability is ErrUnavailable.
// Audio cannot be redacted; the transcript is (by ParseLog) before any further call.
func (c *Client) Transcribe(ctx context.Context, feature Feature, req TranscribeRequest) (Transcript, error) {
	if c == nil {
		return Transcript{}, ErrUnavailable
	}
	cands, err := c.candidates(ctx, feature)
	if err != nil {
		return Transcript{}, err
	}
	cands = slices.DeleteFunc(cands, func(cd candidate) bool { return cd.a.transcriber == nil })
	if len(cands) == 0 {
		return Transcript{}, ErrUnavailable
	}
	if err := c.allow(ctx); err != nil {
		return Transcript{}, err
	}
	size := len(req.Audio.Data)
	var t Transcript
	for i, cd := range cands {
		start := c.now()
		var u Usage
		t, u, err = cd.a.transcriber.Transcribe(cd.ctx, req)
		u.Feature, u.Op, u.AudioBytes, u.OK = feature, OpTranscribe, size, err == nil
		c.record(cd.ctx, u, start)
		if !tryNext(err, i, len(cands)) {
			break
		}
	}
	return t, err
}

// ParseLog maps text to taxonomy candidates for feature. The text is redacted before the provider sees it.
func (c *Client) ParseLog(ctx context.Context, feature Feature, req LogParseRequest) ([]Candidate, error) {
	if c == nil {
		return nil, ErrUnavailable
	}
	cands, err := c.candidates(ctx, feature)
	if err != nil {
		return nil, err
	}
	cands = slices.DeleteFunc(cands, func(cd candidate) bool { return cd.a.logParser == nil })
	if len(cands) == 0 {
		return nil, ErrUnavailable
	}
	if err := c.allow(ctx); err != nil {
		return nil, err
	}
	req.Text = redactCtx(ctx, req.Text)
	var out []Candidate
	for i, cd := range cands {
		start := c.now()
		var u Usage
		out, u, err = cd.a.logParser.ParseLog(cd.ctx, req)
		u.Feature, u.Op, u.OK = feature, OpParseLog, err == nil
		c.record(cd.ctx, u, start)
		if !tryNext(err, i, len(cands)) {
			break
		}
	}
	return out, err
}

func (c *Client) record(ctx context.Context, u Usage, start time.Time) {
	if u.Provider == "" {
		u.Provider = c.provider
	}
	if u.Latency == 0 {
		u.Latency = c.now().Sub(start)
	}
	if u.UserID == 0 {
		u.UserID = SubjectFrom(ctx).UserID
	}
	if r, ok := routeFrom(ctx); ok {
		if u.Provider == "" || u.Provider == c.provider {
			u.Provider = r.Provider
		}
		if u.Model == "" {
			u.Model = r.Model
		}
	}
	u.CostMicros = c.cost(ctx, u)
	if c.recorder != nil {
		c.recorder.Record(context.WithoutCancel(ctx), u)
	}
}

// cost is the estimated cost of u: the route's own prices (ai_models, admin data) when it has them, else the
// AI_PRICES table (an unpriced model is never free there).
func (c *Client) cost(ctx context.Context, u Usage) uint64 {
	if u.Provider == config.AIProviderFake {
		return 0
	}
	if r, ok := routeFrom(ctx); ok && r.Priced {
		return r.cost(u)
	}
	return c.pricer.Cost(u)
}

// DefaultCallTimeout bounds one non-streaming provider call when AI_HTTP_TIMEOUT_SECONDS is not set.
const DefaultCallTimeout = 30 * time.Second

// ProviderHTTPClient is the HTTP client of real providers (B-N6-05b). It has no total http.Client.Timeout — that
// would cut every streamed chat at the same 30 s — but transport timeouts: connect / TLS handshake, the wait for
// the response headers (timeout) and idle keep-alive connections. Each non-streaming call is bounded by its own
// context (Gemini.WithCallTimeout); a stream by MaxChatDuration (Client.Chat) and an idle-read timeout
// (GeminiStreamIdleTimeout). Redirects are never followed (the key header must not travel elsewhere).
func ProviderHTTPClient(timeout time.Duration) *http.Client {
	tr, _ := http.DefaultTransport.(*http.Transport)
	if tr == nil {
		tr = &http.Transport{}
	} else {
		tr = tr.Clone()
	}
	tr.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	tr.TLSHandshakeTimeout = 10 * time.Second
	tr.ResponseHeaderTimeout = timeout
	tr.ExpectContinueTimeout = time.Second
	tr.IdleConnTimeout = 90 * time.Second
	return &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Deps are what New needs.
type Deps struct {
	App    config.App
	Config config.AI
	Logger *slog.Logger
	// HTTPClient overrides the provider HTTP client (tests).
	HTTPClient *http.Client
	// Recorder is the durable usage recorder (internal/ai/usage.Recorder); the slog line is always written too.
	Recorder Recorder
	// Limiter is the global daily cost cap (internal/ai/usage.Budget). nil = no cap: only tests and tools.
	Limiter Limiter
	// Router picks each feature's model from the admin catalogue (internal/ai/credits.Catalog). nil = the adapter
	// named by AI_PROVIDER serves every feature with its configured model.
	Router Router
}

// New builds the Client for the configured provider, or nil when AI is unavailable (provider none, or a real
// provider whose key is missing — AI features then answer 503 ai_unavailable). The fake is never built in
// production, whatever the config says (config.Load already refuses that combination).
func New(d Deps) *Client {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	rec := Recorders{LogRecorder{Logger: logger}}
	if d.Recorder != nil {
		rec = append(rec, d.Recorder)
	}
	base := Options{Recorder: rec, Limiter: d.Limiter, Pricer: NewPricer(d.Config.Prices), Router: d.Router}
	provider := config.DefaultAIProvider(d.Config.Provider, d.App)
	switch provider {
	case config.AIProviderFake:
		if d.App.IsProduction() {
			logger.Error("ai: the fake provider is disabled in production; AI unavailable")
			return nil
		}
		f := NewFake()
		base.Provider, base.Transcriber, base.LogParser, base.Chatter, base.Extractor = config.AIProviderFake, f, f, f, f
		return NewClientWith(base)
	case config.AIProviderGemini, config.AIProviderArvan:
	default:
		return nil
	}
	timeout := d.Config.Timeout
	if timeout <= 0 {
		timeout = DefaultCallTimeout
	}
	client := d.HTTPClient
	if client == nil {
		client = ProviderHTTPClient(timeout)
	}
	// Every real adapter whose key is set is available to the Router, whatever AI_PROVIDER names.
	base.Adapters = map[string]Adapter{}
	if d.Config.Gemini.APIKey != "" {
		g := NewGemini(d.Config.Gemini, client).WithCallTimeout(timeout)
		base.Adapters[ProviderGemini] = Adapter{Transcriber: g, LogParser: g, Chatter: g, Extractor: g}
	}
	if d.Config.Arvan.APIKey != "" && d.Config.Arvan.BaseURL != "" {
		a := NewOpenAICompat(OpenAICompatConfig{Provider: ProviderArvan, BaseURL: d.Config.Arvan.BaseURL,
			APIKey: d.Config.Arvan.APIKey, AuthScheme: "apikey"}, client).WithCallTimeout(timeout)
		base.Adapters[ProviderArvan] = Adapter{LogParser: a, Chatter: a, Extractor: a, Lister: a}
	}
	def, ok := base.Adapters[provider]
	if !ok {
		logger.Error("ai: AI_PROVIDER names a provider whose key (or gateway address) is not set; AI unavailable",
			slog.String("provider", provider))
		return nil
	}
	base.Provider, base.Transcriber, base.LogParser, base.Chatter, base.Extractor = provider, def.Transcriber, def.LogParser, def.Chatter, def.Extractor
	return NewClientWith(base)
}
