package files

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// HTTP (Go only, deviations.md D-59; OpenAPI tag Files):
//
//	POST   /api/v1/files                       auth  multipart purpose + file → 201 the file with its link
//	GET    /api/v1/files/{id}                  auth  the owner's file + a fresh link
//	DELETE /api/v1/files/{id}                  auth  delete (row + blob)
//	GET    /api/v1/files/{id}/download?…       none  signed link → the bytes (private, no-store, attachment)
//	GET    /api/v1/files/public/{id}/{name}    none  public purposes only → the image (cacheable)
//
// A foreign or unknown id is the same 404 as a missing one; a bad / foreign / expired link is 403 (never 401: the
// clients wipe the session only on an authentication 401).

// Error codes.
const (
	ErrorCodeNotFound    = "file_not_found"
	ErrorCodeStorage     = "storage_unavailable"
	ErrorCodeBusy        = "upload_busy"
	ErrorCodeLinkInvalid = "link_invalid"
	ErrorCodeLinkExpired = "link_expired"
)

// maxUploadBody bounds the multipart body: the largest purpose plus the text fields and multipart framing.
const maxUploadBody = 10*mb + 64<<10

//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the files line for key ("messages.deleted") in locale, with :params.
func T(key, locale string, params map[string]string) string {
	return translator().Trans("files."+key, params, locale)
}

// Handlers serve /api/v1/files.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers builds the handlers.
func NewHandlers(svc *Service, c clock.Clock) *Handlers { return &Handlers{svc: svc, clock: c} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func fail(err error, locale string) error {
	f := func(status int, key, code string) error {
		return httpx.Fail(status, T("messages."+key, locale, nil), "error_code", code)
	}
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUnreadable):
		return f(fiber.StatusNotFound, "not_found", ErrorCodeNotFound)
	case errors.Is(err, ErrDisabled):
		return f(fiber.StatusServiceUnavailable, "storage_unavailable", ErrorCodeStorage)
	case errors.Is(err, ErrBusy):
		return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.busy", locale, nil), "error_code", ErrorCodeBusy).
			WithHeader("Retry-After", "5")
	case errors.Is(err, ErrLinkInvalid):
		return f(fiber.StatusForbidden, "link_invalid", ErrorCodeLinkInvalid)
	case errors.Is(err, ErrLinkExpired):
		return f(fiber.StatusForbidden, "link_expired", ErrorCodeLinkExpired)
	}
	return err
}

func fieldFail(locale, field, key string, params map[string]string) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale, nil),
		"errors", jsonx.Obj(field, []string{T("validation."+key, locale, params)}))
}

func idParam(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0 && strconv.FormatUint(id, 10) == c.Params("id")
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// fileJSON is the client view (no path, no owner).
func (h *Handlers) fileJSON(f File, now time.Time) (*jsonx.OrderedMap, error) {
	link, err := h.svc.Link(f, now, DefaultLinkTTL)
	if err != nil {
		return nil, err
	}
	var exp any
	if link.ExpiresAt != nil {
		exp = jsonx.ISO8601(link.ExpiresAt.In(civildate.Tehran))
	}
	return jsonx.Obj(
		"id", f.ID,
		"purpose", f.Purpose,
		"visibility", f.Visibility,
		"mime", f.MIME,
		"size_bytes", f.Size,
		"sha256", f.SHA256,
		"url", link.URL,
		"url_expires_at", exp,
		"created_at", jsonx.ISO8601(f.CreatedAt.In(civildate.Tehran)),
	), nil
}

// upload is the read multipart body.
type upload struct {
	purpose string
	data    []byte
}

// readUpload parses the body by hand — the two known fields, one bounded file, no multipart temp file.
func readUpload(c fiber.Ctx, raw []byte, locale string) (upload, error) {
	missing := fieldFail(locale, "file", "file_required", nil)
	if c.Request().Header.ContentLength() > maxUploadBody || len(raw) > maxUploadBody {
		return upload{}, fieldFail(locale, "file", "file_too_large", map[string]string{"max": "10"})
	}
	if c.Get(fiber.HeaderContentEncoding) != "" {
		return upload{}, missing
	}
	mt, params, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		return upload{}, missing
	}
	r := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	var out upload
	for parts := 0; ; parts++ {
		part, err := r.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || parts >= 4 {
			clear(out.data)
			return upload{}, missing
		}
		switch part.FormName() {
		case "file":
			if part.FileName() == "" || out.data != nil {
				clear(out.data)
				return upload{}, missing
			}
			b, err := io.ReadAll(io.LimitReader(part, 10*mb+1))
			if err != nil {
				clear(b)
				return upload{}, missing
			}
			out.data = b
		case "purpose":
			b, _ := io.ReadAll(io.LimitReader(part, 64))
			out.purpose = strings.TrimSpace(string(b))
		default:
			clear(out.data)
			return upload{}, missing
		}
	}
	return out, nil
}

// Upload is POST /api/v1/files.
func (h *Handlers) Upload(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	raw := c.Request().Body()
	defer clear(raw) // the document never outlives the request in memory
	if h.svc.Disabled() {
		return fail(ErrDisabled, locale)
	}
	up, err := readUpload(c, raw, locale)
	if err != nil {
		return err
	}
	defer clear(up.data)
	p, ok := h.svc.Purpose(up.purpose)
	if !ok || !p.UserUpload {
		return fieldFail(locale, "purpose", "purpose_invalid", nil)
	}
	if len(up.data) == 0 {
		return fieldFail(locale, "file", "file_required", nil)
	}
	now := h.now(c)
	f, err := h.svc.Put(c.Context(), userID, p.Name, up.data, now)
	switch {
	case errors.Is(err, ErrType):
		return fieldFail(locale, "file", "file_type", nil)
	case errors.Is(err, ErrTooLarge):
		return fieldFail(locale, "file", "file_too_large", map[string]string{"max": strconv.Itoa(p.MaxBytes / mb)})
	case errors.Is(err, ErrQuota):
		return fieldFail(locale, "file", "quota", nil)
	case errors.Is(err, ErrPurpose):
		return fieldFail(locale, "purpose", "purpose_invalid", nil)
	case err != nil:
		return fail(err, locale)
	}
	body, err := h.fileJSON(f, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, body, T("messages.uploaded", locale, nil))
}

// Show is GET /api/v1/files/{id}: the owner's file with a fresh link.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c)
	if !ok {
		return fail(ErrNotFound, locale)
	}
	f, err := h.svc.Get(c.Context(), userID, id)
	if err != nil {
		return fail(err, locale)
	}
	body, err := h.fileJSON(f, h.now(c))
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, body)
}

// Destroy is DELETE /api/v1/files/{id}.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c)
	if !ok {
		return fail(ErrNotFound, locale)
	}
	if err := h.svc.Delete(c.Context(), userID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.Message(c, T("messages.deleted", locale, nil))
}

func ext(mimeType string) string {
	if mimeType == "application/pdf" {
		return "pdf"
	}
	return "webp"
}

// Download is GET /api/v1/files/{id}/download?expires&signature: the decrypted file for whoever holds a valid,
// unexpired link — an attachment that is never cached, sniffed or rendered as a page.
func (h *Handlers) Download(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	id, ok := idParam(c)
	if !ok {
		return fail(ErrLinkInvalid, locale)
	}
	f, data, err := h.svc.OpenSigned(c.Context(), id, c.Query("expires"), c.Query("signature"), h.now(c))
	if err != nil {
		return fail(err, locale)
	}
	defer clear(data)
	c.Set(fiber.HeaderContentType, f.MIME)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="file-`+strconv.FormatUint(f.ID, 10)+"."+ext(f.MIME)+`"`)
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentSecurityPolicy, "sandbox; default-src 'none'")
	c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
	return c.Status(fiber.StatusOK).Send(append([]byte(nil), data...))
}

// Public is GET /api/v1/files/public/{id}/{name}: a public purpose's image, cacheable.
func (h *Handlers) Public(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	id, ok := idParam(c)
	if !ok {
		return fail(ErrNotFound, locale)
	}
	f, data, err := h.svc.OpenPublic(c.Context(), id, c.Params("name"))
	if err != nil {
		return fail(err, locale)
	}
	c.Set(fiber.HeaderContentType, f.MIME)
	c.Set(fiber.HeaderContentDisposition, `inline; filename="image-`+strconv.FormatUint(f.ID, 10)+"."+ext(f.MIME)+`"`)
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentSecurityPolicy, "sandbox; default-src 'none'")
	return c.Status(fiber.StatusOK).Send(data)
}
