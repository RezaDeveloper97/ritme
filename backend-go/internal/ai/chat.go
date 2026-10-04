package ai

import (
	"context"
	"time"
	"unicode/utf8"
)

// Streaming text chat (B-N6-05; the assistant clinic B-N7-06 streams it to the browser over SSE).
//
// A provider answers with a channel of chunks; the Client forwards them as ChatEvents on a ChatStream and records
// the usage once when the stream ends (finished, failed, cancelled or timed out). B-N6-05b:
//
//   - the stream runs on a context the Client owns: derived from the caller's, bounded by MaxChatDuration, and
//     cancelled by ChatStream.Close. Fiber's c.Context() is never cancelled when the browser goes away, so the
//     consumer MUST `defer stream.Close()` and also call Close as soon as a write to the client fails;
//   - the usage is never lost: providers attach their latest cumulative usage to every chunk, and when none was
//     reported (or the stream was cut short) the Client records a conservative estimate from the request size and
//     the text already delivered — an interrupted stream is never recorded as free;
//   - Plus quota is reserved by the HTTP gate before the stream starts (internal/ai/access); the feature refunds
//     it (access.ReservationFrom(ctx).Refund) only when the stream failed before its first delta.
//
//	stream, err := client.Chat(ctx, ai.FeatureAssistant, req) // ErrUnavailable / ErrBudgetExceeded / ErrUserBudgetExceeded / ErrInvalidRequest / ErrUpstream
//	if err != nil { … }
//	defer stream.Close()
//	for e := range stream.Events {
//	    if e.Err != nil { … send an SSE error event; break }
//	    if e.Done { … e.Finish; break }
//	    if writeSSE(e.Delta) != nil { stream.Close(); break } // client went away: stop the provider now
//	}
//
// History (L4): Messages must be built server-side from the stored conversation (B-N7-06 keeps it per user). A
// feature never accepts earlier turns — above all assistant turns — from the client, which could otherwise put
// words in the assistant's mouth or replay another conversation; only the user's new message comes from the request.

// ChatRole is who wrote a message.
type ChatRole string

// Roles.
const (
	RoleUser      ChatRole = "user"
	RoleAssistant ChatRole = "assistant"
)

// Image is a photo attached to a chat message, in memory only (sniffed MIME: image/jpeg | image/png | image/webp).
type Image struct {
	Data []byte
	MIME string
}

// ChatMessage is one turn of the conversation.
type ChatMessage struct {
	Role   ChatRole
	Text   string
	Images []Image // user turns only
}

// ChatRequest is one chat completion.
type ChatRequest struct {
	// System is the system instruction (department prompt + guardrails, admin-authored in B-N7-06). It is redacted
	// like the messages: profile context goes in as age / mode / medications, never a name.
	System   string
	Messages []ChatMessage // oldest first; the last one is the user's new message
	Language string        // reply language hint (request locale)
	// MaxOutputTokens caps the answer (0 → DefaultChatMaxOutputTokens; at most MaxChatOutputTokens).
	MaxOutputTokens int
}

// Chat limits enforced by the Client (ErrInvalidRequest).
const (
	MaxChatMessages            = 60
	MaxChatTextRunes           = 48000 // system + every message together
	MaxChatImages              = 4
	MaxImageBytes              = 5 << 20  // per photo
	MaxChatImageBytes          = 16 << 20 // all photos of one request: fits RequestBodyLimit with the text
	DefaultChatMaxOutputTokens = 1024
	MaxChatOutputTokens        = 4096
	// MaxChatDuration bounds one streamed answer, provider request to last chunk (B-N6-05b).
	MaxChatDuration = 120 * time.Second
)

// Finish reasons.
const (
	FinishStop   = "stop"   // the model finished
	FinishLength = "length" // MaxOutputTokens reached
	FinishSafety = "safety" // the provider blocked the answer
)

// ChatChunk is what a provider streams: text deltas, then one last chunk with Done (Finish and the Usage) or Err.
type ChatChunk struct {
	Delta  string
	Done   bool
	Finish string
	Usage  Usage // on the Done / Err chunk: the call's metering so far
	Err    error // ErrUpstream (wrapped); the stream ends after it
}

// Chatter streams chat completions.
type Chatter interface {
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

// ChatEvent is what the Client hands to a feature: a text delta, the end (Done + Finish) or an error.
type ChatEvent struct {
	Delta  string
	Done   bool
	Finish string
	Err    error
}

// ChatStream is a running chat completion: Events delivers the deltas, then one Done or Err event, and is closed.
// The stream ends by itself after MaxChatDuration; Close ends it earlier (idempotent, safe from any goroutine).
type ChatStream struct {
	Events <-chan ChatEvent
	cancel context.CancelFunc
	done   chan struct{}
}

// Close cancels the stream: the provider request is aborted and the goroutines behind Events exit. The usage is
// still recorded (an estimate when the provider reported none).
func (s *ChatStream) Close() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

// Wait blocks until the stream ended and its usage was recorded (tests, graceful shutdown).
func (s *ChatStream) Wait() {
	if s != nil && s.done != nil {
		<-s.done
	}
}

func validateChat(req ChatRequest) error {
	if len(req.Messages) == 0 || len(req.Messages) > MaxChatMessages {
		return ErrInvalidRequest
	}
	runes, images, imageBytes := len([]rune(req.System)), 0, 0
	for _, m := range req.Messages {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			return ErrInvalidRequest
		}
		runes += len([]rune(m.Text))
		for _, img := range m.Images {
			// L3: the bytes must be the image type claimed (central magic-byte sniffing, sniff.go).
			if m.Role != RoleUser || len(img.Data) == 0 || len(img.Data) > MaxImageBytes || SniffImage(img.Data) != img.MIME ||
				img.MIME == "" {
				return ErrInvalidRequest
			}
			images++
			imageBytes += len(img.Data)
		}
	}
	if runes > MaxChatTextRunes || images > MaxChatImages || imageBytes > MaxChatImageBytes || req.MaxOutputTokens < 0 {
		return ErrInvalidRequest
	}
	return nil
}

// Token estimates used when a provider reports no usage (B-N6-05b). Deliberately high: one token per two
// characters (Persian text averages fewer characters per token than English) and Gemini's 258 tokens per image.
const (
	estRunesPerToken  = 2
	estTokensPerImage = 258
)

func estimateTokens(runes int) int { return (runes + estRunesPerToken - 1) / estRunesPerToken }

// estimateChatInput is the conservative input token count of req (≥ 1).
func estimateChatInput(req ChatRequest) int {
	runes, images := utf8.RuneCountInString(req.System), 0
	for _, m := range req.Messages {
		runes += utf8.RuneCountInString(m.Text)
		images += len(m.Images)
	}
	return max(1, estimateTokens(runes)+estTokensPerImage*images)
}

// chatUsage is the usage to record for a stream: what the provider reported last, completed by estimates. A
// stream that ended normally (done) with reported counts keeps them as they are; otherwise the input is at least
// the request estimate and the output at least the estimate of the text delivered — never zero tokens.
func chatUsage(reported Usage, inputEstimate, deliveredRunes int, done bool) Usage {
	u := reported
	if done && u.InputTokens > 0 {
		return u
	}
	if u.InputTokens == 0 {
		u.InputTokens, u.Estimated = inputEstimate, true
	}
	if out := estimateTokens(deliveredRunes); u.OutputTokens < out {
		u.OutputTokens, u.Estimated = out, true
	}
	return u
}

// Chat streams a completion for feature. The request is validated, the budgets checked and every text redacted
// before the provider sees it; the usage is recorded once when the stream ends (see the comment at the top).
func (c *Client) Chat(ctx context.Context, feature Feature, req ChatRequest) (*ChatStream, error) {
	if c == nil || c.chatter == nil {
		return nil, ErrUnavailable
	}
	if err := validateChat(req); err != nil {
		return nil, err
	}
	if err := c.allow(ctx); err != nil {
		return nil, err
	}
	out := ChatRequest{System: redactCtx(ctx, req.System), Language: req.Language, MaxOutputTokens: req.MaxOutputTokens,
		Messages: make([]ChatMessage, len(req.Messages))}
	if out.MaxOutputTokens == 0 {
		out.MaxOutputTokens = DefaultChatMaxOutputTokens
	}
	out.MaxOutputTokens = min(out.MaxOutputTokens, MaxChatOutputTokens)
	imageBytes := 0
	for i, m := range req.Messages {
		out.Messages[i] = ChatMessage{Role: m.Role, Text: redactCtx(ctx, m.Text), Images: m.Images}
		for _, img := range m.Images {
			imageBytes += len(img.Data)
		}
	}
	inputEstimate := estimateChatInput(out)
	maxDur := c.chatMax
	if maxDur <= 0 {
		maxDur = MaxChatDuration
	}
	// The stream's own context: the caller's values (Subject, test clock), its cancellation, plus our bound.
	sctx, cancel := context.WithTimeout(ctx, maxDur)
	start := c.now()
	chunks, err := c.chatter.ChatStream(sctx, out)
	if err != nil {
		cancel()
		// The provider refused the request (bad status, blocked before streaming): nothing was generated.
		c.record(ctx, Usage{Feature: feature, Op: OpChat, ImageBytes: imageBytes}, start)
		return nil, err
	}
	events := make(chan ChatEvent)
	stream := &ChatStream{Events: events, cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(stream.done)
		defer cancel()
		defer close(events)
		var reported Usage
		delivered, finished, done := 0, false, false
		defer func() {
			u := chatUsage(reported, inputEstimate, delivered, done)
			u.Feature, u.Op, u.ImageBytes, u.OK = feature, OpChat, imageBytes, finished
			u.Latency = 0 // measured by record: the whole stream
			c.record(context.WithoutCancel(ctx), u, start)
		}()
		send := func(ev ChatEvent) bool {
			select {
			case events <- ev:
				return true
			case <-sctx.Done():
				return false
			}
		}
		keep := func(u Usage) {
			if u.InputTokens > 0 || u.OutputTokens > 0 {
				reported = u
			}
		}
		for {
			select {
			case <-sctx.Done():
				go drain(chunks)
				return
			case ch, ok := <-chunks:
				if !ok {
					// The provider closed without Done: an unusable answer (or it noticed the cancellation first).
					if sctx.Err() == nil {
						send(ChatEvent{Err: ErrUpstream})
					}
					return
				}
				keep(ch.Usage)
				switch {
				case ch.Err != nil:
					send(ChatEvent{Err: ch.Err})
					go drain(chunks)
					return
				case ch.Done:
					finished, done = true, true
					send(ChatEvent{Done: true, Finish: ch.Finish})
					go drain(chunks)
					return
				case ch.Delta != "":
					if !send(ChatEvent{Delta: ch.Delta}) {
						go drain(chunks)
						return
					}
					delivered += utf8.RuneCountInString(ch.Delta)
				}
			}
		}
	}()
	return stream, nil
}

// drain empties a provider channel so its goroutine can finish after the consumer stopped.
func drain(ch <-chan ChatChunk) {
	for range ch { //nolint:revive // empty block: draining
	}
}
