package ai

import (
	"context"
)

// Streaming text chat (B-N6-05; the assistant clinic B-N7-06 streams it to the browser over SSE).
//
// A provider answers with a channel of chunks; the Client forwards them as ChatEvents and records the usage when
// the stream ends (finished, failed or cancelled). The consumer ranges over the channel and MUST cancel ctx when it
// stops early (client went away): the goroutines behind the stream exit on ctx.Done(), never block forever.
//
//	ev, err := client.Chat(ctx, ai.FeatureAssistant, req)   // err: ErrUnavailable / ErrBudgetExceeded / ErrInvalidRequest / ErrUpstream
//	for e := range ev {
//	    if e.Err != nil { … send an SSE error event; break }
//	    if e.Done { … e.Finish; break }
//	    … send e.Delta as an SSE data line
//	}

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
	MaxImageBytes              = 8 << 20
	DefaultChatMaxOutputTokens = 1024
	MaxChatOutputTokens        = 4096
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

func validateChat(req ChatRequest) error {
	if len(req.Messages) == 0 || len(req.Messages) > MaxChatMessages {
		return ErrInvalidRequest
	}
	runes, images := len([]rune(req.System)), 0
	for _, m := range req.Messages {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			return ErrInvalidRequest
		}
		runes += len([]rune(m.Text))
		for _, img := range m.Images {
			if m.Role != RoleUser || len(img.Data) == 0 || len(img.Data) > MaxImageBytes || !imageMIME[img.MIME] {
				return ErrInvalidRequest
			}
			images++
		}
	}
	if runes > MaxChatTextRunes || images > MaxChatImages || req.MaxOutputTokens < 0 {
		return ErrInvalidRequest
	}
	return nil
}

var imageMIME = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// Chat streams a completion for feature. The request is validated, the budget checked and every text redacted
// before the provider sees it; the usage is recorded once when the stream ends.
func (c *Client) Chat(ctx context.Context, feature Feature, req ChatRequest) (<-chan ChatEvent, error) {
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
	start := c.now()
	chunks, err := c.chatter.ChatStream(ctx, out)
	if err != nil {
		c.record(ctx, Usage{Feature: feature, Op: OpChat, ImageBytes: imageBytes}, start)
		return nil, err
	}
	events := make(chan ChatEvent)
	go func() {
		defer close(events)
		var u Usage
		finished := false
		defer func() {
			u.Feature, u.Op, u.ImageBytes, u.OK = feature, OpChat, imageBytes, finished
			u.Latency = 0 // measured by record: the whole stream
			c.record(context.WithoutCancel(ctx), u, start)
		}()
		send := func(ev ChatEvent) bool {
			select {
			case events <- ev:
				return true
			case <-ctx.Done():
				return false
			}
		}
		for {
			select {
			case <-ctx.Done():
				go drain(chunks)
				return
			case ch, ok := <-chunks:
				if !ok {
					// The provider closed without Done: an unusable answer.
					send(ChatEvent{Err: ErrUpstream})
					return
				}
				switch {
				case ch.Err != nil:
					u = ch.Usage
					send(ChatEvent{Err: ch.Err})
					go drain(chunks)
					return
				case ch.Done:
					u, finished = ch.Usage, true
					send(ChatEvent{Done: true, Finish: ch.Finish})
					go drain(chunks)
					return
				case ch.Delta != "":
					if !send(ChatEvent{Delta: ch.Delta}) {
						go drain(chunks)
						return
					}
				}
			}
		}
	}()
	return events, nil
}

// drain empties a provider channel so its goroutine can finish after the consumer stopped.
func drain(ch <-chan ChatChunk) {
	for range ch { //nolint:revive // empty block: draining
	}
}
