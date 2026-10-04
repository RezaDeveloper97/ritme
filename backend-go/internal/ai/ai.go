// Package ai is the AI adapter layer (bloom decision: every external AI service sits behind a port with a
// deterministic `fake` provider as the default outside production).
//
// B-N3-05 added speech-to-text and free text → log taxonomy NLU (voice logging); B-N6-05 made it the platform
// every AI feature uses (voice log, lab analysis B-N6-06, assistant B-N7-06, document extraction CB-REC-02):
//
//   - one small interface per capability: Transcriber, LogParser, Chatter (streaming chat, chat.go) and
//     Extractor (image / PDF → structured fields with confidence, extract.go),
//   - providers implement any subset (Fake and Gemini implement all four),
//   - Client is what domains hold: it picks the configured provider (New), refuses calls once the global daily
//     cost cap is spent (Limiter → ErrBudgetExceeded), strips PII from outgoing text (pii.go), validates what
//     comes back (extraction), labels every call with the Feature that made it and reports a Usage with its
//     estimated cost (pricing.go) to the Recorder (slog line + the ai_usage_logs row, internal/ai/usage),
//   - provider errors surface as ErrUpstream / ErrUnavailable so handlers map them to one envelope,
//   - consent and Plus quota are checked before a feature runs by internal/ai/access (HTTP middleware).
//
// Privacy: requests carry only what the capability needs (the audio, the text, the document, the schema, a
// language hint). Never a user id or account field; names, phone numbers, national ids, card numbers and e-mail
// addresses are redacted from every outgoing text (the Subject on the context names the user's own). Neither the
// Client nor a provider logs payloads: the Recorder sees counts, sizes, durations, cost and the outcome only.
package ai

import (
	"context"
	"errors"
	"time"
)

// Feature names the product feature a call is made for (usage log, quotas, consent in B-N6-05).
type Feature string

// Features.
const (
	FeatureVoiceLog    Feature = "voice_log"    // B-N3-05
	FeatureLabAnalysis Feature = "lab_analysis" // B-N6-06: lab sheet extraction + interpretation
	FeatureAssistant   Feature = "assistant"    // B-N7-06: assistant clinic chat
	FeatureDocExtract  Feature = "doc_extract"  // CB-REC-02: non-lab medical document extraction
)

// Errors every provider maps its failures to.
var (
	// ErrUnavailable: no provider is configured (AI_PROVIDER=none, or a real provider without its key).
	ErrUnavailable = errors.New("ai: no provider configured")
	// ErrUpstream: the provider failed, timed out or answered something unusable. Safe to retry later.
	ErrUpstream = errors.New("ai: provider error")
	// ErrBudgetExceeded: the global daily cost cap is spent, or the budget could not be read (fail closed).
	ErrBudgetExceeded = errors.New("ai: daily cost cap reached")
	// ErrInvalidRequest: the request breaks a platform limit (size, type, message count). A caller bug or a
	// client the handler should have refused first; never sent to a provider.
	ErrInvalidRequest = errors.New("ai: invalid request")
)

// Audio is a recording held in memory only. The owner wipes it (Wipe) as soon as it has been transcribed.
type Audio struct {
	Data []byte
	MIME string // sniffed server-side, never the client's claim
}

// Wipe zeroes the recording and drops the reference, so no copy of the audio outlives the request.
func (a *Audio) Wipe() {
	clear(a.Data[:cap(a.Data)]) // the whole buffer, not just the recorded length
	a.Data = nil
}

// TranscribeRequest is one speech-to-text call.
type TranscribeRequest struct {
	Audio Audio
	// Language is the expected spoken language (the request locale, e.g. "fa"); providers treat it as a hint.
	Language string
}

// Transcript is what was heard.
type Transcript struct {
	Text     string
	Language string
}

// Transcriber turns speech into text.
type Transcriber interface {
	Transcribe(ctx context.Context, req TranscribeRequest) (Transcript, Usage, error)
}

// VocabEntry is one loggable slot the parser may answer with: "category.param" or "category.param.item".
type VocabEntry struct {
	Key string // "pain.location.abdomen", or a canvas field "pain_diary.score" (CB-VOICE-01)
	// Type is the taxonomy param type (single | multi | items | number | integer | bool) or, for canvas fields,
	// text (a short free string as said, ≤ MaxLen runes) | time (24 h "HH:MM").
	Type  string
	Label string // human label in the request language ("درد › محل درد › شکم")
	// Values are the accepted codes with their labels: single → options, items → levels.
	Values []VocabValue
	// Min / Max / Unit describe number and integer slots.
	Min, Max float64
	Unit     string
	// MaxLen bounds a text slot.
	MaxLen int
}

// VocabValue is one accepted value code with its label.
type VocabValue struct {
	Code  string
	Label string
}

// LogParseRequest asks for the taxonomy slots a free-text description of the day mentions.
type LogParseRequest struct {
	Text       string
	Language   string
	Vocabulary []VocabEntry // only the slots the user may log now (mode, custom items)
}

// Candidate is one slot the parser believes the text mentions. Value follows the slot type: single → option
// code (string), multi → true, items → level code (string), number/integer → float64, bool → bool, text → string,
// time → "HH:MM". Callers must validate candidates against the taxonomy: a provider is never trusted.
type Candidate struct {
	Key        string
	Value      any
	Confidence float64 // 0–1
	// Alternatives are other slot keys the same words may just as well mean («بی‌حوصله» → bored, sad or tired):
	// the review shows a chooser (nbl_Voice_Review «مطمئن نیستیم»). Key stays the provider's best guess.
	Alternatives []string
}

// LogParser maps free text to log taxonomy slots.
type LogParser interface {
	ParseLog(ctx context.Context, req LogParseRequest) ([]Candidate, Usage, error)
}

// Usage is the metering of one call. No payload; UserID only for the per-user usage log (0 = system job).
type Usage struct {
	Provider     string
	Model        string
	Feature      Feature
	Op           string // OpTranscribe | OpParseLog | OpChat | OpExtract
	InputTokens  int    // every prompt token, audio included
	OutputTokens int
	AudioTokens  int // the audio part of InputTokens when the provider reports it (priced separately)
	AudioBytes   int
	ImageBytes   int // images and documents sent (chat photos, extraction files)
	CostMicros   uint64
	Latency      time.Duration
	OK           bool
	UserID       uint64
}

// Ops.
const (
	OpTranscribe = "transcribe"
	OpParseLog   = "parse_log"
	OpChat       = "chat"
	OpExtract    = "extract"
)
