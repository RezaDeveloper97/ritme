package media

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/media/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Error codes of the non-2xx bodies.
const (
	ErrorCodeNotFound       = "media_not_found"
	ErrorCodeLessonNotFound = "learning_lesson_not_found"
	ErrorCodeNotReady       = "media_not_ready"
	ErrorCodeOffsetMismatch = "media_offset_mismatch"
	ErrorCodeBusy           = "media_upload_busy"
	ErrorCodeNotUploading   = "media_not_uploading"
	ErrorCodeUploadExpired  = "media_upload_expired"
	ErrorCodeTooLarge       = "media_too_large"
	ErrorCodeChunkTooLarge  = "media_chunk_too_large"
	ErrorCodeBeyondSize     = "media_beyond_size"
	ErrorCodeType           = "media_type_not_accepted"
	ErrorCodeContentType    = "media_content_type"
	ErrorCodeRejected       = "media_rejected"
	ErrorCodeLinkInvalid    = "link_invalid"
	ErrorCodeLinkExpired    = "link_expired"
	ErrorCodeRange          = "range_not_satisfiable"
	ErrorCodeUnavailable    = "media_unavailable"
	ChunkContentType        = "application/offset+octet-stream"
	HeaderUploadOffset      = "Upload-Offset"
	HeaderUploadLength      = "Upload-Length"
)

// LessonOpener checks a student's access to a lesson (learning.Service.OpenLesson).
type LessonOpener interface {
	OpenLesson(ctx context.Context, userID, lessonID uint64, now time.Time) (learning.StudentLesson, error)
}

// Handlers are the instructor upload routes (/api/instructor/v1/media, behind auth RequireUser +
// learning RequireInstructor), the playback URL routes and the signed stream (no bearer: the link is the credential).
type Handlers struct {
	svc     *Service
	lessons LessonOpener
	clock   clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(svc *Service, lessons LessonOpener, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, lessons: lessons, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

func failCode(status int, locale, key, code string, extra ...any) error {
	return httpx.Fail(status, T("messages."+key, locale), append([]any{"error_code", code}, extra...)...)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

// fail maps the service errors to their bodies.
func fail(err error, locale string) error {
	var mismatch *OffsetMismatchError
	switch {
	case errors.As(err, &mismatch):
		return failCode(fiber.StatusConflict, locale, "offset_mismatch", ErrorCodeOffsetMismatch, "offset", mismatch.Offset)
	case errors.Is(err, ErrDisabled):
		return failCode(fiber.StatusServiceUnavailable, locale, "unavailable", ErrorCodeUnavailable)
	case errors.Is(err, ErrNotFound):
		return failCode(fiber.StatusNotFound, locale, "not_found", ErrorCodeNotFound)
	case errors.Is(err, ErrLessonNotFound):
		return failCode(fiber.StatusNotFound, locale, "lesson_not_found", ErrorCodeLessonNotFound)
	case errors.Is(err, ErrNotReady):
		return failCode(fiber.StatusNotFound, locale, "not_ready", ErrorCodeNotReady)
	case errors.Is(err, ErrBusy):
		return failCode(fiber.StatusConflict, locale, "busy", ErrorCodeBusy)
	case errors.Is(err, ErrNotUploading):
		return failCode(fiber.StatusConflict, locale, "not_uploading", ErrorCodeNotUploading)
	case errors.Is(err, ErrUploadExpired):
		return failCode(fiber.StatusGone, locale, "upload_expired", ErrorCodeUploadExpired)
	case errors.Is(err, ErrTooLarge):
		return failCode(fiber.StatusRequestEntityTooLarge, locale, "too_large", ErrorCodeTooLarge)
	case errors.Is(err, ErrChunkTooLarge):
		return failCode(fiber.StatusRequestEntityTooLarge, locale, "chunk_too_large", ErrorCodeChunkTooLarge)
	case errors.Is(err, ErrBeyondSize):
		return failCode(fiber.StatusRequestEntityTooLarge, locale, "beyond_size", ErrorCodeBeyondSize)
	case errors.Is(err, ErrType):
		return failCode(fiber.StatusUnsupportedMediaType, locale, "type_not_accepted", ErrorCodeType)
	case errors.Is(err, ErrRejected):
		return failCode(fiber.StatusUnprocessableEntity, locale, "rejected", ErrorCodeRejected)
	case errors.Is(err, ErrMimeMismatch):
		return fieldFail(locale, "mime", "mime_not_allowed")
	case errors.Is(err, ErrTooManyUploads):
		return fieldFail(locale, "lesson_id", "too_many_uploads")
	case errors.Is(err, ErrQuota):
		return fieldFail(locale, "size", "quota")
	case errors.Is(err, ErrLinkInvalid):
		return failCode(fiber.StatusForbidden, locale, "link_invalid", ErrorCodeLinkInvalid)
	case errors.Is(err, ErrLinkExpired):
		return failCode(fiber.StatusGone, locale, "link_expired", ErrorCodeLinkExpired)
	}
	return err
}

func iso(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return jsonx.ISO8601(t.In(civildate.Tehran))
}

func isoN(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return iso(t.Time)
}

// JSON is one upload / file as its instructor sees it.
func JSON(m store.LearningMedium) *jsonx.OrderedMap {
	var lesson, mime any
	if m.LessonID.Valid {
		lesson = m.LessonID.Int64
	}
	if m.Mime.Valid {
		mime = m.Mime.String
	}
	return jsonx.Obj(
		"id", m.ID,
		"lesson_id", lesson,
		"kind", m.Kind,
		"mime", mime,
		"status", m.Status,
		"size_bytes", m.SizeBytes,
		"offset_bytes", m.OffsetBytes,
		"upload_expires_at", isoN(m.UploadExpiresAt),
		"completed_at", isoN(m.CompletedAt),
		"created_at", isoN(m.CreatedAt),
	)
}

// PlaybackJSON is a minted playback URL.
func PlaybackJSON(p Playback) *jsonx.OrderedMap {
	return jsonx.Obj(
		"media_id", p.Media.ID,
		"kind", p.Media.Kind,
		"mime", p.Media.Mime.String,
		"size_bytes", p.Media.SizeBytes,
		"url", p.URL,
		"expires_at", iso(p.ExpiresAt),
	)
}

func uploadHeaders(c fiber.Ctx, m store.LearningMedium) {
	c.Set(HeaderUploadOffset, strconv.FormatUint(m.OffsetBytes, 10))
	c.Set(HeaderUploadLength, strconv.FormatUint(m.SizeBytes, 10))
	c.Set(fiber.HeaderCacheControl, "no-store")
}

func instructorID(c fiber.Ctx) (uint64, error) {
	ins := learning.CurrentInstructor(c)
	if ins.ID == 0 {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return ins.ID, nil
}

// validateCreate is POST /media {lesson_id, size, mime?}.
func validateCreate(body phpval.Map, locale string, now time.Time) (CreateInput, error) {
	data := phpval.NewMap()
	for _, k := range []string{"lesson_id", "size", "mime"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("lesson_id", "required", "integer", "min:1"),
		validation.F("size", "required", "integer", "min:1"),
		validation.F("mime", "nullable", "string", validation.In(AllMimes()...)),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return CreateInput{}, failValidation(locale, v.ErrorBag())
	}
	get := func(k string) string {
		x, _ := data.Get(k)
		if x == nil {
			return ""
		}
		return strings.TrimSpace(phpval.ToString(x))
	}
	lesson, err := strconv.ParseUint(get("lesson_id"), 10, 64)
	if err != nil {
		return CreateInput{}, fail(ErrLessonNotFound, locale)
	}
	size, err := strconv.ParseInt(get("size"), 10, 64)
	if err != nil { // digits only after the integer rule: beyond int64
		return CreateInput{}, fail(ErrTooLarge, locale)
	}
	return CreateInput{LessonID: lesson, Size: size, Mime: get("mime")}, nil
}

// Create is POST /api/instructor/v1/media {lesson_id, size, mime?}: 201 {media, upload} — an upload of `size`
// bytes for one of the caller's lessons (404 for a foreign lesson, 413 over the kind's limit).
func (h *Handlers) Create(c fiber.Ctx) error {
	ins, err := instructorID(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateCreate(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	m, err := h.svc.Create(c, ins, in, now)
	if err != nil {
		return fail(err, locale)
	}
	uploadHeaders(c, m)
	return httpx.Created(c, jsonx.Obj("media", JSON(m),
		"upload", jsonx.Obj("chunk_size", ChunkSize, "max_chunk_bytes", MaxChunkBytes, "content_type", ChunkContentType)),
		T("messages.created", locale))
}

// Show is GET /api/instructor/v1/media/{id}: the upload with its offset (the resume point).
func (h *Handlers) Show(c fiber.Ctx) error {
	ins, err := instructorID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	m, err := h.svc.Owned(c, ins, id)
	if err != nil {
		return fail(err, locale)
	}
	uploadHeaders(c, m)
	return httpx.OK(c, jsonx.Obj("media", JSON(m)))
}

// Patch is PATCH /api/instructor/v1/media/{id} (Upload-Offset: n, Content-Type: application/offset+octet-stream,
// body = the next chunk): 200 {media}; the last chunk answers with status ready (or 422 media_rejected).
func (h *Handlers) Patch(c fiber.Ctx) error {
	ins, err := instructorID(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(c.Get(fiber.HeaderContentType), ";", 2)[0]))
	if enc := strings.TrimSpace(c.Get(fiber.HeaderContentEncoding)); ct != ChunkContentType || (enc != "" && !strings.EqualFold(enc, "identity")) {
		return failCode(fiber.StatusUnsupportedMediaType, locale, "content_type", ErrorCodeContentType)
	}
	raw := c.Get(HeaderUploadOffset)
	offset, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || strconv.FormatUint(offset, 10) != raw {
		return fieldFail(locale, "upload_offset", "upload_offset")
	}
	m, err := h.svc.WriteChunk(c, ins, id, offset, c.Request().Body(), now)
	if err != nil {
		var mismatch *OffsetMismatchError
		if errors.As(err, &mismatch) {
			c.Set(HeaderUploadOffset, strconv.FormatUint(mismatch.Offset, 10))
		}
		return fail(err, locale)
	}
	uploadHeaders(c, m)
	msg := "messages.chunk_saved"
	if m.Status == StatusReady {
		msg = "messages.ready"
	}
	return httpx.OK(c, jsonx.Obj("media", JSON(m)), T(msg, locale))
}

// Destroy is DELETE /api/instructor/v1/media/{id}: cancels an upload or removes a file (the lesson follows).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	ins, err := instructorID(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	if err := h.svc.Cancel(c, ins, id, now); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// InstructorPlayback is GET /api/instructor/v1/media/{id}/playback: a signed URL of one of the caller's ready files
// (preview in the panel).
func (h *Handlers) InstructorPlayback(c fiber.Ctx) error {
	ins, err := instructorID(c)
	if err != nil {
		return err
	}
	userID, _ := auth.CurrentUserID(c)
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	m, err := h.svc.Owned(c, ins, id)
	if err != nil {
		return fail(err, locale)
	}
	p, err := h.svc.PlaybackFor(m, userID, now)
	if err != nil {
		return fail(err, locale)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.OK(c, jsonx.Obj("playback", PlaybackJSON(p)))
}

// LessonPlayback is GET /api/v1/learning/lessons/{id}/playback: a signed URL of the lesson's media for a student
// who may open the lesson now (the lesson's own 404 / 403 access_expired / chapter_locked otherwise).
func (h *Handlers) LessonPlayback(c fiber.Ctx) error {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return learning.Fail(learning.ErrLessonNotFound, locale)
	}
	open, err := h.lessons.OpenLesson(c, userID, id, now)
	if err != nil {
		return learning.Fail(err, locale)
	}
	p, err := h.svc.LessonPlayback(c, open.Lesson.MediaID, userID, now)
	if err != nil {
		return fail(err, locale)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.OK(c, jsonx.Obj("playback", PlaybackJSON(p)))
}

type readCloser struct {
	io.Reader
	io.Closer
}

// Stream is GET /api/v1/media/{id}/stream?u=&expires=&signature= (no bearer): the file, 200 or 206 for one byte
// range (416 when unsatisfiable). 403 link_invalid / 410 link_expired; 404 when the file is gone.
func (h *Handlers) Stream(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrLinkInvalid, locale)
	}
	m, f, err := h.svc.OpenStream(c, id, c.Query("u"), c.Query("expires"), c.Query("signature"), now)
	if err != nil {
		return fail(err, locale)
	}
	size := int64(m.SizeBytes) //nolint:gosec // G115: ≤ 2 GiB
	etag := `"m` + strconv.FormatUint(m.ID, 10) + "-" + strconv.FormatInt(size, 10) + `"`
	c.Set(fiber.HeaderAcceptRanges, "bytes")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
	c.Set(fiber.HeaderETag, etag)
	header := c.Get(fiber.HeaderRange)
	if ir := c.Get(fiber.HeaderIfRange); ir != "" && ir != etag {
		header = "" // the client's copy is another file: send it whole
	}
	r, ok := parseRange(header, size)
	if !ok {
		_ = f.Close()
		c.Set(fiber.HeaderContentRange, "bytes */"+strconv.FormatInt(size, 10))
		return failCode(fiber.StatusRequestedRangeNotSatisfiable, locale, "range_not_satisfiable", ErrorCodeRange)
	}
	if _, err := f.Seek(r.Start, io.SeekStart); err != nil {
		_ = f.Close()
		return fail(ErrNotFound, locale)
	}
	n := r.End - r.Start + 1
	c.Set(fiber.HeaderContentType, m.Mime.String)
	c.Set(fiber.HeaderContentDisposition, "inline")
	status := fiber.StatusOK
	if r.Partial {
		status = fiber.StatusPartialContent
		c.Set(fiber.HeaderContentRange, "bytes "+strconv.FormatInt(r.Start, 10)+"-"+strconv.FormatInt(r.End, 10)+"/"+strconv.FormatInt(size, 10))
	}
	c.Status(status)
	return c.SendStream(readCloser{Reader: io.LimitReader(f, n), Closer: f}, int(n))
}
