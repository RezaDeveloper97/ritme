package ai

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ritme/backend-go/internal/platform/config"
)

// Recorder receives the Usage of every call (B-N6-05 adds a DB recorder for the per-call usage + cost log).
type Recorder interface {
	Record(ctx context.Context, u Usage)
}

// LogRecorder writes one structured slog line per call: provider, model, feature, op, token counts, audio
// size, latency and outcome. Never the payload, never who asked.
type LogRecorder struct{ Logger *slog.Logger }

// Record implements Recorder.
func (r LogRecorder) Record(ctx context.Context, u Usage) {
	if r.Logger == nil {
		return
	}
	r.Logger.LogAttrs(ctx, slog.LevelInfo, "ai: call",
		slog.String("provider", u.Provider), slog.String("model", u.Model), slog.String("feature", string(u.Feature)),
		slog.String("op", u.Op), slog.Int("input_tokens", u.InputTokens), slog.Int("output_tokens", u.OutputTokens),
		slog.Int("audio_bytes", u.AudioBytes), slog.Int64("latency_ms", u.Latency.Milliseconds()), slog.Bool("ok", u.OK))
}

// Client is the AI layer a domain holds: the configured provider's capabilities plus usage metering.
type Client struct {
	provider    string
	transcriber Transcriber
	logParser   LogParser
	recorder    Recorder
	now         func() time.Time
}

// NewClient wires a client from capabilities (tests; production code uses New).
func NewClient(provider string, t Transcriber, p LogParser, rec Recorder) *Client {
	return &Client{provider: provider, transcriber: t, logParser: p, recorder: rec, now: time.Now}
}

// Provider is the configured provider id ("fake", "gemini"); "" for a nil client.
func (c *Client) Provider() string {
	if c == nil {
		return ""
	}
	return c.provider
}

// Transcribe runs speech-to-text for feature. A nil client or a provider without the capability is ErrUnavailable.
func (c *Client) Transcribe(ctx context.Context, feature Feature, req TranscribeRequest) (Transcript, error) {
	if c == nil || c.transcriber == nil {
		return Transcript{}, ErrUnavailable
	}
	start := c.now()
	size := len(req.Audio.Data)
	t, u, err := c.transcriber.Transcribe(ctx, req)
	u.Feature, u.Op, u.AudioBytes, u.OK = feature, "transcribe", size, err == nil
	c.record(ctx, u, start)
	return t, err
}

// ParseLog maps text to taxonomy candidates for feature.
func (c *Client) ParseLog(ctx context.Context, feature Feature, req LogParseRequest) ([]Candidate, error) {
	if c == nil || c.logParser == nil {
		return nil, ErrUnavailable
	}
	start := c.now()
	out, u, err := c.logParser.ParseLog(ctx, req)
	u.Feature, u.Op, u.OK = feature, "parse_log", err == nil
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
	if c.recorder != nil {
		c.recorder.Record(ctx, u)
	}
}

// Deps are what New needs.
type Deps struct {
	App    config.App
	Config config.AI
	Logger *slog.Logger
	// HTTPClient overrides the provider HTTP client (tests).
	HTTPClient *http.Client
	// Recorder overrides the usage recorder (default LogRecorder on Logger).
	Recorder Recorder
}

// New builds the Client for the configured provider, or nil when AI is unavailable (provider none, or a real
// provider whose key is missing — AI features then answer 503 ai_unavailable). The fake is never built in
// production, whatever the config says (config.Load already refuses that combination).
func New(d Deps) *Client {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	rec := d.Recorder
	if rec == nil {
		rec = LogRecorder{Logger: logger}
	}
	switch config.DefaultAIProvider(d.Config.Provider, d.App) {
	case config.AIProviderFake:
		if d.App.IsProduction() {
			logger.Error("ai: the fake provider is disabled in production; AI unavailable")
			return nil
		}
		f := NewFake()
		return NewClient(config.AIProviderFake, f, f, rec)
	case config.AIProviderGemini:
		if d.Config.Gemini.APIKey == "" {
			logger.Error("ai: AI_PROVIDER=gemini but GEMINI_API_KEY is empty; AI unavailable")
			return nil
		}
		client := d.HTTPClient
		if client == nil {
			timeout := d.Config.Timeout
			if timeout <= 0 {
				timeout = 30 * time.Second
			}
			client = &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		}
		g := NewGemini(d.Config.Gemini, client)
		return NewClient(config.AIProviderGemini, g, g, rec)
	default:
		return nil
	}
}
