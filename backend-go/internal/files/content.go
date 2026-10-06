package files

import (
	"bytes"

	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/ai"
)

// MaxImagePixels bounds a decoded photo (~80 MB of RGBA pixels at the limit; a 12 MP phone photo is well inside).
const MaxImagePixels = 20_000_000

// imageOptimizer re-encodes every stored photo (EXIF / GPS and anything appended after the image dropped): large
// enough for small print on an A4 document.
var imageOptimizer = media.Optimizer{MaxWidth: 2400, MaxHeight: 2400, Quality: 85, MaxPixels: MaxImagePixels}

// encodeSlots caps concurrent photo re-encodes process-wide.
var encodeSlots = make(chan struct{}, 2)

var (
	pngSig   = []byte("\x89PNG\r\n\x1a\n")
	pdfMagic = []byte("%PDF-")
	pdfEOF   = []byte("%%EOF")
	utf8BOM  = []byte("\xef\xbb\xbf")
)

// strictPDF: "%PDF-" at offset 0 (after an optional BOM / whitespace) and "%%EOF" within the last KB — stricter than
// the shared sniffer (which also accepts junk before the header), so a polyglot cannot pass as a document here.
func strictPDF(data []byte) bool {
	head := bytes.TrimLeft(bytes.TrimPrefix(data, utf8BOM), " \t\r\n\f\x00")
	tail := data[max(0, len(data)-1024):]
	return bytes.HasPrefix(head, pdfMagic) && bytes.Contains(tail, pdfEOF)
}

// pngTooDeep: a PNG whose IHDR bit depth is above 8 (16-bit images double the decode memory) — refused before decoding.
func pngTooDeep(data []byte) bool {
	return bytes.HasPrefix(data, pngSig) && len(data) > 24 && string(data[12:16]) == "IHDR" && data[24] > 8
}

// prepare turns an upload into the bytes to store: the kind comes from the bytes (never the client's name or
// Content-Type); photos are re-encoded to WebP, PDFs kept as sent. Errors: ErrTooLarge, ErrType, ErrBusy.
func prepare(p Purpose, data []byte) (stored []byte, mime string, err error) {
	if len(data) == 0 {
		return nil, "", ErrType
	}
	if len(data) > p.MaxBytes {
		return nil, "", ErrTooLarge
	}
	switch sniffed := ai.SniffDocument(data); sniffed {
	case "application/pdf":
		if !p.Accepts(KindPDF) || !strictPDF(data) {
			return nil, "", ErrType
		}
		return append([]byte(nil), data...), sniffed, nil
	case "":
		return nil, "", ErrType
	}
	if !p.Accepts(KindImage) || pngTooDeep(data) {
		return nil, "", ErrType
	}
	img, err := media.Inspect(data)
	if err != nil || img.Width*img.Height > MaxImagePixels {
		return nil, "", ErrType
	}
	select {
	case encodeSlots <- struct{}{}:
		defer func() { <-encodeSlots }()
	default:
		return nil, "", ErrBusy
	}
	out, err := imageOptimizer.Optimize(img)
	if err != nil || ai.SniffDocument(out) != "image/webp" {
		return nil, "", ErrType
	}
	return out, "image/webp", nil
}
