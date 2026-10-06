package labs

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Upload parsing (POST /labs): the multipart body is read by hand from the request bytes — only the known fields,
// at most MaxFiles files and maxParts parts, every file bounded while it is read — so nothing is parsed unbounded
// and no multipart temp file is ever written. The type of each file comes from its bytes (ai.SniffDocument):
// JPEG / PNG / WebP photos are re-encoded (EXIF and GPS dropped, anything appended after the image dropped, fit
// into imageOptimizer's box) before they are stored; PDFs are stored as sent (encrypted) and only ever handed to the extractor
// or back to their owner as an attachment.

// maxParts bounds the multipart parts (files + the four text fields + slack).
const maxParts = MaxFiles + 8

// imageOptimizer is the re-encode box of lab photos: large enough for small print on an A4 sheet.
var imageOptimizer = media.Optimizer{MaxWidth: 2400, MaxHeight: 2400, Quality: 85, MaxPixels: MaxImagePixels}

// encodeSlots caps concurrent photo re-encodes process-wide (each can hold ~160 MB of pixels at the limit).
var encodeSlots = make(chan struct{}, 2)

// uploadFields are the accepted text fields.
var uploadFields = map[string]bool{"category": true, "title": true, "taken_on": true, "fasting": true}

// parsedUpload is the read multipart body.
type parsedUpload struct {
	fields phpval.Map
	files  []Page
}

func (p *parsedUpload) wipe() {
	for i := range p.files {
		clear(p.files[i].Data)
		p.files[i].Data = nil
	}
}

// readUpload parses the request body. Errors are ready-made 422 / 503 responses.
func readUpload(c fiber.Ctx, raw []byte, locale string) (*parsedUpload, error) {
	if c.Request().Header.ContentLength() > MaxUploadBytes || len(raw) > MaxUploadBytes {
		return nil, fieldFail(locale, "files", "upload_too_large")
	}
	missing := fieldFail(locale, "files", "files_required")
	if c.Get(fiber.HeaderContentEncoding) != "" {
		return nil, missing
	}
	mt, params, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		return nil, missing
	}
	r := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	out := &parsedUpload{fields: phpval.NewMap()}
	fail := func(e error) (*parsedUpload, error) {
		out.wipe()
		return nil, e
	}
	for parts := 0; ; parts++ {
		part, err := r.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || parts >= maxParts {
			return fail(missing)
		}
		name := part.FormName()
		switch {
		case name == "files" || name == "files[]":
			if part.FileName() == "" {
				return fail(fieldFail(locale, "files", "file_type"))
			}
			if len(out.files) >= MaxFiles {
				return fail(fieldFail(locale, "files", "too_many_files"))
			}
			page, err := readPage(part, locale)
			if err != nil {
				return fail(err)
			}
			out.files = append(out.files, page)
		case uploadFields[name]:
			b, _ := io.ReadAll(io.LimitReader(part, 512))
			if s := strings.TrimSpace(string(b)); s != "" { // ConvertEmptyStringsToNull: an empty field is absent
				out.fields.Set(name, s)
			}
		default:
			return fail(missing)
		}
	}
	if len(out.files) == 0 {
		return fail(missing)
	}
	return out, nil
}

// readPage reads one file part: bounded, sniffed, photos re-encoded.
func readPage(part *multipart.Part, locale string) (Page, error) {
	buf := make([]byte, MaxPDFBytes+1)
	n, err := io.ReadFull(part, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		clear(buf)
		return Page{}, fieldFail(locale, "files", "file_type")
	}
	data := buf[:n]
	kind := ai.SniffDocument(data)
	switch {
	case n == 0 || kind == "":
		clear(buf)
		return Page{}, fieldFail(locale, "files", "file_type")
	case kind == "application/pdf":
		if n > MaxPDFBytes {
			clear(buf)
			return Page{}, fieldFail(locale, "files", "pdf_too_large")
		}
		page := Page{Data: append([]byte(nil), data...), MIME: kind} // the exact size; the 10 MB buffer is zeroed
		clear(buf)
		return page, nil
	case n > MaxImageBytes:
		clear(buf)
		return Page{}, fieldFail(locale, "files", "image_too_large")
	}
	defer clear(buf)
	img, err := media.Inspect(data)
	if err != nil || img.Width*img.Height > MaxImagePixels {
		return Page{}, fieldFail(locale, "files", "image_unreadable")
	}
	select {
	case encodeSlots <- struct{}{}:
		defer func() { <-encodeSlots }()
	default:
		return Page{}, httpx.Fail(fiber.StatusServiceUnavailable, T("messages.storage_unavailable", locale, nil),
			"error_code", "lab_busy").WithHeader("Retry-After", "5")
	}
	out, err := imageOptimizer.Optimize(img)
	if err != nil || ai.SniffDocument(out) != "image/webp" {
		return Page{}, fieldFail(locale, "files", "image_unreadable")
	}
	return Page{Data: out, MIME: "image/webp"}, nil
}
