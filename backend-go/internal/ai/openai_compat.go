package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// OpenAICompat is a provider that speaks the OpenAI chat-completions API: POST {base}/chat/completions, with
// server-sent events when streaming. The ArvanCloud AI gateway (docs.arvancloud.ir/fa/aiaas/gateway) is one: its
// address ends in /v1, the key travels as `Authorization: apikey <key>`, and the `model` field picks one of the
// models attached to the gateway — so one adapter serves every model the catalogue lists for it (route.go).
//
// It implements Chatter (streaming, photos as data: URLs), LogParser and Extractor (images only — a PDF is refused
// with ErrUpstream so a fallback model that reads PDFs can take the call). It has no speech-to-text: a feature
// that needs CapAudio must be routed to a provider that has it.
//
// Like Gemini: the key is only ever in the Authorization header, error bodies are never read into errors (they
// could echo the request) and nothing of the payload is logged.
type OpenAICompat struct {
	provider    string
	baseURL     string
	auth        string // the whole Authorization header value
	model       string // default model when the call carries no route (tests)
	http        *http.Client
	callTimeout time.Duration
	streamIdle  time.Duration
}

// OpenAICompatConfig configures an OpenAI-compatible provider.
type OpenAICompatConfig struct {
	Provider string // the provider id recorded in the usage log (ProviderArvan)
	BaseURL  string // ends in /v1
	APIKey   string
	// AuthScheme is the word before the key in the Authorization header ("Bearer" when empty; "apikey" for Arvan).
	AuthScheme string
	// Model is used only when a call carries no route.
	Model string
}

// Output caps of the non-chat calls (the same bounds as Gemini's).
const (
	openAIParseMaxTokens   = GeminiParseMaxOutputTokens
	openAIExtractMaxTokens = GeminiExtractMaxOutputTokens
)

// NewOpenAICompat builds the provider.
func NewOpenAICompat(cfg OpenAICompatConfig, client *http.Client) *OpenAICompat {
	scheme := cfg.AuthScheme
	if scheme == "" {
		scheme = "Bearer"
	}
	return &OpenAICompat{provider: cfg.Provider, baseURL: strings.TrimRight(cfg.BaseURL, "/"), auth: scheme + " " + cfg.APIKey,
		model: cfg.Model, http: client, callTimeout: DefaultCallTimeout, streamIdle: GeminiStreamIdleTimeout}
}

// WithCallTimeout sets the bound of one non-streaming call (≤ 0 keeps the current one).
func (o *OpenAICompat) WithCallTimeout(d time.Duration) *OpenAICompat {
	if d > 0 {
		o.callTimeout = d
	}
	return o
}

// WithStreamIdleTimeout sets the longest silence between two lines of a stream (≤ 0 keeps the current one).
func (o *OpenAICompat) WithStreamIdleTimeout(d time.Duration) *OpenAICompat {
	if d > 0 {
		o.streamIdle = d
	}
	return o
}

type oaPart struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *oaImageURL `json:"image_url,omitempty"`
}

type oaImageURL struct {
	URL string `json:"url"`
}

// oaMessage is one message: Content is a string, or parts when the turn carries images.
type oaMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type oaRequest struct {
	Model         string         `json:"model"`
	Messages      []oaMessage    `json:"messages"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
	Temperature   float64        `json:"temperature"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions map[string]any `json:"stream_options,omitempty"`
}

type oaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (m *oaUsage) apply(u *Usage) {
	if m == nil || (m.PromptTokens == 0 && m.CompletionTokens == 0) {
		return
	}
	u.InputTokens, u.OutputTokens = m.PromptTokens, m.CompletionTokens
}

type oaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *oaUsage `json:"usage"`
}

func dataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (o *OpenAICompat) usage(ctx context.Context) Usage {
	return Usage{Provider: o.provider, Model: routeModel(ctx, o.model)}
}

func (o *OpenAICompat) request(ctx context.Context, body oaRequest) (*http.Request, []byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: encode request", ErrUpstream)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		clear(raw)
		return nil, nil, fmt.Errorf("%w: build request", ErrUpstream)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.auth)
	return req, raw, nil
}

// complete runs one non-streaming completion and returns the answer text.
func (o *OpenAICompat) complete(ctx context.Context, body oaRequest) (string, Usage, error) {
	u := o.usage(ctx)
	body.Model = u.Model
	if o.callTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.callTimeout)
		defer cancel()
	}
	req, raw, err := o.request(ctx, body)
	if err != nil {
		return "", u, err
	}
	start := time.Now()
	res, err := o.http.Do(req)
	clear(raw)
	u.Latency = time.Since(start)
	if err != nil {
		return "", u, fmt.Errorf("%w: request failed", ErrUpstream)
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, maxGeminiResponse))
	if err != nil {
		return "", u, fmt.Errorf("%w: read response", ErrUpstream)
	}
	if res.StatusCode != http.StatusOK {
		return "", u, fmt.Errorf("%w: status %d", ErrUpstream, res.StatusCode)
	}
	var out oaResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", u, fmt.Errorf("%w: decode response", ErrUpstream)
	}
	out.Usage.apply(&u)
	if len(out.Choices) == 0 {
		return "", u, fmt.Errorf("%w: no choices", ErrUpstream)
	}
	if out.Choices[0].FinishReason == "content_filter" {
		return "", u, fmt.Errorf("%w: blocked", ErrUpstream)
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), u, nil
}

// ParseLog implements LogParser.
func (o *OpenAICompat) ParseLog(ctx context.Context, req LogParseRequest) ([]Candidate, Usage, error) {
	text, u, err := o.complete(ctx, oaRequest{MaxTokens: openAIParseMaxTokens, Messages: []oaMessage{
		{Role: "system", Content: geminiParseInstruction},
		{Role: "user", Content: parseLogPrompt(req)},
	}})
	if err != nil {
		return nil, u, err
	}
	out, err := decodeCandidates(text)
	return out, u, err
}

// Extract implements Extractor for images. A PDF cannot travel as an image part: it is refused before any request.
func (o *OpenAICompat) Extract(ctx context.Context, req ExtractRequest) (Extraction, Usage, error) {
	if !strings.HasPrefix(req.Document.MIME, "image/") {
		return Extraction{}, o.usage(ctx), fmt.Errorf("%w: document type not supported by this provider", ErrUpstream)
	}
	text, u, err := o.complete(ctx, oaRequest{MaxTokens: openAIExtractMaxTokens, Messages: []oaMessage{
		{Role: "system", Content: geminiExtractInstruction},
		{Role: "user", Content: []oaPart{
			{Type: "text", Text: extractPrompt(req)},
			{Type: "image_url", ImageURL: &oaImageURL{URL: dataURL(req.Document.MIME, req.Document.Data)}},
		}},
	}})
	if err != nil {
		return Extraction{}, u, err
	}
	out, err := decodeExtraction(text)
	return out, u, err
}

func oaFinish(reason string) string {
	switch strings.ToLower(reason) {
	case "":
		return ""
	case "stop":
		return FinishStop
	case "length":
		return FinishLength
	default: // content_filter, …
		return FinishSafety
	}
}

// ChatStream implements Chatter over a streamed completion. ctx bounds the whole stream (Client.Chat owns it); a
// silence longer than streamIdle abandons it. Usage arrives in the last chunk (stream_options.include_usage);
// when a gateway does not send it, the Client records its own estimate.
func (o *OpenAICompat) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	u := o.usage(ctx)
	body := oaRequest{Model: u.Model, MaxTokens: req.MaxOutputTokens, Temperature: 0.4, Stream: true,
		StreamOptions: map[string]any{"include_usage": true},
		Messages:      []oaMessage{{Role: "system", Content: req.System + fmt.Sprintf(geminiChatSuffix, req.Language)}}}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "assistant"
		}
		if len(m.Images) == 0 {
			text := m.Text
			if text == "" {
				text = " "
			}
			body.Messages = append(body.Messages, oaMessage{Role: role, Content: text})
			continue
		}
		parts := make([]oaPart, 0, 1+len(m.Images))
		if m.Text != "" {
			parts = append(parts, oaPart{Type: "text", Text: m.Text})
		}
		for _, img := range m.Images {
			parts = append(parts, oaPart{Type: "image_url", ImageURL: &oaImageURL{URL: dataURL(img.MIME, img.Data)}})
		}
		body.Messages = append(body.Messages, oaMessage{Role: role, Content: parts})
	}
	sctx, scancel := context.WithCancel(ctx)
	hreq, raw, err := o.request(sctx, body)
	if err != nil {
		scancel()
		return nil, err
	}
	hreq.Header.Set("Accept", "text/event-stream")
	start := time.Now()
	res, err := o.http.Do(hreq) //nolint:bodyclose // closed by the stream goroutine below (or right away on a non-200)
	clear(raw)
	if err != nil {
		scancel()
		return nil, fmt.Errorf("%w: request failed", ErrUpstream)
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		scancel()
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, res.StatusCode)
	}
	idle := o.streamIdle
	idleTimer := time.AfterFunc(idle, scancel)
	out := make(chan ChatChunk)
	go func() {
		defer close(out)
		defer scancel()
		defer idleTimer.Stop()
		defer func() { _ = res.Body.Close() }()
		emit := func(c ChatChunk) bool {
			select {
			case out <- c:
				return true
			case <-ctx.Done():
				return false
			}
		}
		fail := func(msg string) {
			u.Latency = time.Since(start)
			emit(ChatChunk{Err: fmt.Errorf("%w: %s", ErrUpstream, msg), Usage: u})
		}
		sc := bufio.NewScanner(io.LimitReader(res.Body, maxGeminiStream))
		sc.Buffer(make([]byte, 0, 64<<10), maxGeminiLine)
		finish, gotText := "", false
		for sc.Scan() {
			idleTimer.Reset(idle)
			line := sc.Bytes()
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			payload := bytes.TrimSpace(line[len("data:"):])
			if bytes.Equal(payload, []byte("[DONE]")) {
				break
			}
			var chunk oaResponse
			if err := json.Unmarshal(payload, &chunk); err != nil {
				fail("decode stream")
				return
			}
			chunk.Usage.apply(&u)
			if len(chunk.Choices) == 0 {
				continue
			}
			c := chunk.Choices[0]
			if c.Delta.Content != "" {
				gotText = true
				if !emit(ChatChunk{Delta: c.Delta.Content, Usage: u}) {
					return
				}
			}
			if f := oaFinish(c.FinishReason); f != "" {
				finish = f
			}
		}
		if err := sc.Err(); err != nil {
			if ctx.Err() != nil {
				return // the consumer cancelled or the stream's time ran out: nobody is reading
			}
			if sctx.Err() != nil {
				fail("stream idle")
				return
			}
			fail("read stream")
			return
		}
		if finish == "" {
			if !gotText {
				fail("empty stream")
				return
			}
			finish = FinishStop
		}
		u.Latency = time.Since(start)
		emit(ChatChunk{Done: true, Finish: finish, Usage: u})
	}()
	return out, nil
}

// ListModels asks the gateway which models it serves (GET {base}/models), sorted. Used by the admin «همگام‌سازی»
// of the catalogue; a gateway without the endpoint answers ErrUpstream and the admin adds models by hand.
func (o *OpenAICompat) ListModels(ctx context.Context) ([]string, error) {
	if o.callTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.callTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: build request", ErrUpstream)
	}
	req.Header.Set("Authorization", o.auth)
	res, err := o.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed", ErrUpstream)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, res.StatusCode)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxGeminiResponse)).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: decode response", ErrUpstream)
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
