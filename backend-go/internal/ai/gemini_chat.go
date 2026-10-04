package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/config"
)

// maxGeminiStream bounds what is read from one streamed answer (and one SSE line).
const (
	maxGeminiStream = 4 << 20
	maxGeminiLine   = 1 << 20
)

// geminiChatSuffix is appended to every system instruction: the reply language and the redaction placeholders.
const geminiChatSuffix = `

Reply in the language with code %q unless the user writes in another language.
Placeholders like [name], [phone], [id], [number] and [email] stand for details removed for privacy: never ask for them.`

// ChatStream implements Chatter over streamGenerateContent (server-sent events). Images travel inline; the
// request body is zeroed once sent. Every chunk carries the latest cumulative usageMetadata seen (B-N6-05b), so a
// stream cut short still reports what was billed so far. ctx bounds the whole stream (Client.Chat owns it); a
// silence longer than streamIdle between two lines abandons it too.
func (g *Gemini) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	u := Usage{Provider: config.AIProviderGemini, Model: g.model}
	body := gRequest{
		SystemInstruction: &gContent{Parts: []gPart{{Text: req.System + fmt.Sprintf(geminiChatSuffix, req.Language)}}},
		// Thinking at the model's minimum (B-N6-05b, M1): reasoning tokens are billed as output and are not bounded
		// by what the user sees; B-N7-06 may raise the budget deliberately (it is counted either way).
		GenerationConfig: map[string]any{"temperature": 0.4, "maxOutputTokens": req.MaxOutputTokens,
			"thinkingConfig": map[string]any{"thinkingBudget": g.thinkingBudget()}},
	}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		parts := make([]gPart, 0, 1+len(m.Images))
		if m.Text != "" {
			parts = append(parts, gPart{Text: m.Text})
		}
		for _, img := range m.Images {
			parts = append(parts, gPart{InlineData: &gInline{MimeType: img.MIME, Data: img.Data}})
		}
		if len(parts) == 0 {
			parts = append(parts, gPart{Text: " "})
		}
		body.Contents = append(body.Contents, gContent{Role: role, Parts: parts})
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: encode request", ErrUpstream)
	}
	endpoint := g.baseURL + "/v1beta/models/" + url.PathEscape(g.model) + ":streamGenerateContent?alt=sse"
	// sctx is cancelled by the idle timer (or when the stream goroutine ends); ctx stays the caller's.
	sctx, scancel := context.WithCancel(ctx)
	hreq, err := http.NewRequestWithContext(sctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		scancel()
		clear(raw)
		return nil, fmt.Errorf("%w: build request", ErrUpstream)
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "text/event-stream")
	hreq.Header.Set("x-goog-api-key", g.key)
	start := time.Now()
	res, err := g.http.Do(hreq) //nolint:bodyclose // closed by the stream goroutine below (or right away on a non-200)
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
	idle := g.streamIdle
	if idle <= 0 {
		idle = GeminiStreamIdleTimeout
	}
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
			var chunk gResponse
			if err := json.Unmarshal(bytes.TrimSpace(line[len("data:"):]), &chunk); err != nil {
				fail("decode stream")
				return
			}
			if chunk.UsageMetadata.known() {
				chunk.UsageMetadata.apply(&u) // cumulative: the last one seen is the call's usage so far
			}
			if chunk.PromptFeedback != nil && chunk.PromptFeedback.BlockReason != "" {
				finish = FinishSafety
				break
			}
			if len(chunk.Candidates) == 0 {
				continue
			}
			cand := chunk.Candidates[0]
			for _, p := range cand.Content.Parts {
				if p.Text == "" {
					continue
				}
				gotText = true
				if !emit(ChatChunk{Delta: p.Text, Usage: u}) {
					return
				}
			}
			if f := geminiFinish(cand.FinishReason); f != "" {
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

// geminiFinish maps a Gemini finishReason to ours ("" while the stream goes on).
func geminiFinish(reason string) string {
	switch strings.ToUpper(reason) {
	case "":
		return ""
	case "STOP":
		return FinishStop
	case "MAX_TOKENS":
		return FinishLength
	default: // SAFETY, RECITATION, BLOCKLIST, PROHIBITED_CONTENT, SPII, OTHER, …
		return FinishSafety
	}
}
