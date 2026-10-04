package ai

import (
	"context"
	"log/slog"
	"net"
	"net/http"
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
	provider    string
	transcriber Transcriber
	logParser   LogParser
	chatter     Chatter
	extractor   Extractor
	recorder    Recorder
	limiter     Limiter
	pricer      Pricer
	chatMax     time.Duration
	now         func() time.Time
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
	// ChatMaxDuration overrides MaxChatDuration (tests).
	ChatMaxDuration time.Duration
}

// NewClientWith builds a client from o.
func NewClientWith(o Options) *Client {
	return &Client{provider: o.Provider, transcriber: o.Transcriber, logParser: o.LogParser, chatter: o.Chatter,
		extractor: o.Extractor, recorder: o.Recorder, limiter: o.Limiter, pricer: o.Pricer, chatMax: o.ChatMaxDuration,
		now: time.Now}
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
	if c == nil || c.transcriber == nil {
		return Transcript{}, ErrUnavailable
	}
	if err := c.allow(ctx); err != nil {
		return Transcript{}, err
	}
	start := c.now()
	size := len(req.Audio.Data)
	t, u, err := c.transcriber.Transcribe(ctx, req)
	u.Feature, u.Op, u.AudioBytes, u.OK = feature, OpTranscribe, size, err == nil
	c.record(ctx, u, start)
	return t, err
}

// ParseLog maps text to taxonomy candidates for feature. The text is redacted before the provider sees it.
func (c *Client) ParseLog(ctx context.Context, feature Feature, req LogParseRequest) ([]Candidate, error) {
	if c == nil || c.logParser == nil {
		return nil, ErrUnavailable
	}
	if err := c.allow(ctx); err != nil {
		return nil, err
	}
	req.Text = redactCtx(ctx, req.Text)
	start := c.now()
	out, u, err := c.logParser.ParseLog(ctx, req)
	u.Feature, u.Op, u.OK = feature, OpParseLog, err == nil
	c.record(ctx, u, start)
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
	u.CostMicros = c.pricer.Cost(u)
	if c.recorder != nil {
		c.recorder.Record(context.WithoutCancel(ctx), u)
	}
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
	opts := func(provider string, all interface {
		Transcriber
		LogParser
		Chatter
		Extractor
	},
	) *Client {
		return NewClientWith(Options{Provider: provider, Transcriber: all, LogParser: all, Chatter: all, Extractor: all,
			Recorder: rec, Limiter: d.Limiter, Pricer: NewPricer(d.Config.Prices)})
	}
	switch config.DefaultAIProvider(d.Config.Provider, d.App) {
	case config.AIProviderFake:
		if d.App.IsProduction() {
			logger.Error("ai: the fake provider is disabled in production; AI unavailable")
			return nil
		}
		return opts(config.AIProviderFake, NewFake())
	case config.AIProviderGemini:
		if d.Config.Gemini.APIKey == "" {
			logger.Error("ai: AI_PROVIDER=gemini but GEMINI_API_KEY is empty; AI unavailable")
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
		return opts(config.AIProviderGemini, NewGemini(d.Config.Gemini, client).WithCallTimeout(timeout))
	default:
		return nil
	}
}
