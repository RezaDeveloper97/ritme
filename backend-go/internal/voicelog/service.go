// Package voicelog is «ثبت با صدا» (B-N3-05, Plus): a short recording is transcribed and mapped to log
// taxonomy suggestions through the AI adapter (internal/ai). Nothing is saved here — the client reviews the
// suggestions, merges them into its draft and saves through PUT /logs/days/{date} (voice_params marks them
// source=voice).
//
// Privacy: the recording lives only in one request-scoped buffer (never on disk: the multipart body is parsed
// by hand, so no temp file is ever created) and is zeroed right after transcription, together with the raw
// request body. The transcript is returned to the client and never stored or logged; the provider receives the
// audio, the transcript and the taxonomy vocabulary only — no name, phone or user id.
package voicelog

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthlog/store"
)

// MaxTranscriptRunes caps the transcript handed to the parser and returned to the client.
const MaxTranscriptRunes = 2000

// Result is one processed recording.
type Result struct {
	Transcript  string
	Language    string
	Mode        string
	Suggestions []Suggestion
}

// Logs is what the service reads of the health log (satisfied by *healthlog.Service).
type Logs interface {
	LifeMode(ctx context.Context, userID uint64) (string, error)
	CustomItems(ctx context.Context, userID uint64, activeOnly bool) ([]store.HealthLogCustomItem, error)
}

// Service runs speech-to-text and parsing for a user.
type Service struct {
	ai   *ai.Client
	logs Logs
}

// NewService wires the service; client may be nil (AI unavailable → ai.ErrUnavailable).
func NewService(client *ai.Client, logs Logs) *Service {
	return &Service{ai: client, logs: logs}
}

// Process transcribes audio (wiping it as soon as the provider answered, success or not) and maps the
// transcript to suggestions valid for the user's mode and custom items. ns is the `log-taxonomy` namespace
// for the request language (labels). An empty transcript is a valid result without suggestions.
func (s *Service) Process(ctx context.Context, userID uint64, locale string, ns any, audio *ai.Audio) (Result, error) {
	t, err := s.ai.Transcribe(ctx, ai.FeatureVoiceLog, ai.TranscribeRequest{Audio: *audio, Language: locale})
	audio.Wipe()
	if err != nil {
		return Result{}, err
	}
	text := truncate(strings.TrimSpace(t.Text), MaxTranscriptRunes)
	lang := t.Language
	if lang == "" {
		lang = locale
	}
	mode, err := s.logs.LifeMode(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	res := Result{Transcript: text, Language: lang, Mode: mode, Suggestions: []Suggestion{}}
	if text == "" {
		return res, nil
	}
	custom, err := s.logs.CustomItems(ctx, userID, true)
	if err != nil {
		return Result{}, err
	}
	v := buildVocabulary(mode, ns, custom)
	cands, err := s.ai.ParseLog(ctx, ai.FeatureVoiceLog, ai.LogParseRequest{Text: text, Language: lang, Vocabulary: v.entries})
	if err != nil {
		return Result{}, err
	}
	res.Suggestions = v.validate(cands, mode, locale)
	return res, nil
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
