package voicelog

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
)

// Upload limits. 60 s of browser audio (Opus / AAC) is well under 1 MB; 2 MB leaves room for WAV.
const (
	MaxAudioBytes  = 2 << 20
	MaxUploadBytes = MaxAudioBytes + 64<<10 // multipart overhead
	MaxDurationMs  = 60_000
	// MaxParts caps the multipart parts read (audio + duration_ms, with room for a client's extra boundary quirk).
	MaxParts = 4
	// ProcessTimeout bounds the AI work of one request (the web client waits 45 s).
	ProcessTimeout = 40 * time.Second
	// durationSlackMs tolerates the recorder's timer overshooting the 60 s auto-stop.
	durationSlackMs = 1_500
)

// Error codes of the 503 envelope.
const (
	CodeAIUnavailable = "ai_unavailable"
	CodeAIFailed      = "ai_failed"
)

// AcceptedFormats are the sniffed container types (browser MediaRecorder: Chrome/Edge webm, Firefox ogg,
// Safari mp4) plus wav / mp3, listed in the validation message.
const AcceptedFormats = "webm, ogg, m4a, wav, mp3"

// Handlers serve POST /api/v1/logs/voice. Nothing is logged: no transcript, no health value, no audio.
type Handlers struct {
	svc     *Service
	plus    *plus.Service
	gate    *plus.Gate
	bundles *i18n.TranslationStore
	langs   *i18n.Registry
	clock   clock.Clock
}

// NewHandlers wires the handler; base is the fallback clock.
func NewHandlers(svc *Service, plusSvc *plus.Service, gate *plus.Gate, bundles *i18n.TranslationStore, langs *i18n.Registry, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, plus: plusSvc, gate: gate, bundles: bundles, langs: langs, clock: base}
}

// Voice is POST /logs/voice (multipart: audio file ≤ 2 MB, optional duration_ms ≤ 60000). Plus-gated by the
// route (plus.voice_log); one use is counted only after a non-empty transcript was understood. Returns
// {transcript, language, mode, suggestions:[{category, param, item, value, confidence, label}]}; never saves.
func (h *Handlers) Voice(c fiber.Ctx) error {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	raw := c.Request().Body()
	// The raw upload is zeroed whatever happens; no multipart temp file exists (the form is never parsed by
	// fasthttp), the call is a no-op kept as a guard.
	defer func() {
		clear(raw)
		c.Request().RemoveMultipartFormFiles()
	}()

	audio, err := readUpload(c, raw, locale)
	if err != nil {
		return err
	}
	defer audio.Wipe() // already wiped by Process; also covers early returns
	ns := h.bundles.NamespaceMessages(locale, healthlog.TaxonomyNamespace, h.langs.DefaultCode(c.Context()))
	// One bound for transcription + parsing, below the client's 45 s upload budget (features/voice-log).
	ctx, cancel := context.WithTimeout(c.Context(), ProcessTimeout)
	defer cancel()
	res, err := h.svc.Process(ctx, userID, locale, ns, audio)
	switch {
	case errors.Is(err, ai.ErrUnavailable):
		return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.ai_unavailable", locale, nil), "error_code", CodeAIUnavailable)
	case errors.Is(err, ai.ErrUpstream), errors.Is(err, context.DeadlineExceeded):
		return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.ai_failed", locale, nil), "error_code", CodeAIFailed).
			WithHeader("Retry-After", "10")
	case err != nil:
		return err
	}
	if res.Transcript != "" {
		now := clock.FromContext(c, h.clock).Now()
		if e, err := h.plus.Consume(c.Context(), userID, plus.VoiceLog, now); err != nil {
			return h.gate.GateError(c, err, e)
		}
	}
	list := make([]*jsonx.OrderedMap, 0, len(res.Suggestions))
	for _, s := range res.Suggestions {
		list = append(list, s.JSON())
	}
	return httpx.OK(c, jsonx.Obj(
		"transcript", res.Transcript,
		"language", res.Language,
		"mode", res.Mode,
		"suggestions", list,
	))
}

func fail(locale, field, key string, params map[string]string) error {
	all := map[string]string{"attribute": T("attributes."+field, locale, nil)}
	for k, v := range params {
		all[k] = v
	}
	e := httpx.NewValidationError()
	e.Add(field, lang.Default().Trans(key, all, locale))
	return e
}

// readUpload parses the multipart body by hand (streaming, no temp files) and copies the audio into one
// fixed buffer, so the only copies of the recording are that buffer and the raw body — both zeroed.
func readUpload(c fiber.Ctx, raw []byte, locale string) (*ai.Audio, error) {
	tooBig := fail(locale, "audio", "validation.max.file", map[string]string{"max": strconv.Itoa(MaxAudioBytes / 1024)})
	if c.Request().Header.ContentLength() > MaxUploadBytes || len(raw) > MaxUploadBytes {
		return nil, tooBig
	}
	missing := fail(locale, "audio", "validation.required", nil)
	if c.Get(fiber.HeaderContentEncoding) != "" {
		return nil, fail(locale, "audio", "validation.file", nil)
	}
	mt, params, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		return nil, missing
	}
	r := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	var (
		audio    *ai.Audio
		duration string
	)
	// Only the two known fields, at most MaxParts parts: anything else is refused (no unbounded parsing).
	drop := func() {
		if audio != nil {
			audio.Wipe()
		}
	}
	for parts := 0; ; parts++ {
		part, err := r.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || parts >= MaxParts {
			drop()
			return nil, missing
		}
		switch part.FormName() {
		case "audio":
			if part.FileName() == "" {
				if audio != nil {
					audio.Wipe()
				}
				return nil, fail(locale, "audio", "validation.file", nil)
			}
			if audio != nil { // one recording per request; extra parts are ignored
				continue
			}
			buf := make([]byte, MaxAudioBytes+1)
			n, err := io.ReadFull(part, buf)
			if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
				clear(buf)
				return nil, missing
			}
			if n > MaxAudioBytes {
				clear(buf)
				return nil, tooBig
			}
			audio = &ai.Audio{Data: buf[:n]}
		case "duration_ms":
			b, _ := io.ReadAll(io.LimitReader(part, 16))
			duration = strings.TrimSpace(string(b))
		default:
			drop()
			return nil, missing
		}
	}
	if audio == nil || len(audio.Data) == 0 {
		if audio != nil {
			audio.Wipe()
		}
		return nil, missing
	}
	if duration != "" {
		ms, err := strconv.Atoi(duration)
		switch {
		case err != nil:
			audio.Wipe()
			return nil, fail(locale, "duration_ms", "validation.integer", nil)
		case ms < 0:
			audio.Wipe()
			return nil, fail(locale, "duration_ms", "validation.min.numeric", map[string]string{"min": "0"})
		case ms > MaxDurationMs+durationSlackMs:
			audio.Wipe()
			return nil, fail(locale, "duration_ms", "validation.max.numeric", map[string]string{"max": strconv.Itoa(MaxDurationMs)})
		}
	}
	audio.MIME = Sniff(audio.Data)
	if audio.MIME == "" {
		audio.Wipe()
		return nil, fail(locale, "audio", "validation.mimes", map[string]string{"values": AcceptedFormats})
	}
	return audio, nil
}

// Sniff is the audio container type from the bytes themselves (the client's Content-Type is ignored), or ""
// for anything that is not an accepted audio container.
func Sniff(data []byte) string {
	if len(data) >= 12 && string(data[4:8]) == "ftyp" { // ISO BMFF: only the audio brands Safari / .m4a use
		switch string(data[8:12]) {
		case "M4A ", "isom", "mp42":
			return "audio/mp4"
		}
		return ""
	}
	switch http.DetectContentType(data) {
	case "video/webm":
		return "audio/webm"
	case "application/ogg":
		return "audio/ogg"
	case "audio/wave":
		return "audio/wav"
	case "audio/mpeg":
		return "audio/mpeg"
	}
	return ""
}

// Throttles of POST /logs/voice per user (cost control on top of Plus): a burst limit and an hourly one.
const (
	BurstMax  = 6  // per minute
	HourlyMax = 60 // per hour
)

// ErrorCodeTooMany is the throttle 429's error_code (distinct from plus_quota_exceeded).
const ErrorCodeTooMany = "too_many_requests"

// Throttled answers a rejected recording with the localized controller envelope and the throttle headers.
func Throttled(c fiber.Ctx, r ratelimit.Rejection) error {
	e := httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many", i18n.Locale(c), nil),
		"error_code", ErrorCodeTooMany, "retry_after", r.RetryAfter)
	for k, v := range r.Headers() {
		e = e.WithHeader(k, v)
	}
	return e
}
