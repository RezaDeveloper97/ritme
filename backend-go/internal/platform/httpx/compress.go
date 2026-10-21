package httpx

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// CompressMinBytes is the smallest JSON body Compress encodes (T-M9-10, D-132): below ~1 KB the saving does not pay
// for the CPU and the encoding header.
const CompressMinBytes = 1024

// Compress encodes JSON responses of at least CompressMinBytes with brotli or gzip (D-132), whichever the client's
// Accept-Encoding offers first in that order. It leaves alone: HEAD, non-2xx-200 bodies (204/206/304), Range
// requests, streamed bodies (SSE, files, media), anything already carrying a Content-Encoding, non-JSON types and
// `Cache-Control: no-transform`. Errors returned up the chain are rendered later by ErrorHandler and stay plain
// (they are small). nginx in front of stage/beta does not gzip /api, so nothing is compressed twice.
func Compress() fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}
		enc := encoding(c)
		if enc == "" || !compressible(c) {
			return nil
		}
		resp := c.Response()
		body := resp.Body()
		var out []byte
		if enc == "br" {
			out = fasthttp.AppendBrotliBytesLevel(nil, body, fasthttp.CompressBrotliDefaultCompression)
		} else {
			out = fasthttp.AppendGzipBytesLevel(nil, body, fasthttp.CompressDefaultCompression)
		}
		resp.SetBodyRaw(out)
		resp.Header.Set(fiber.HeaderContentEncoding, enc)
		c.Vary(fiber.HeaderAcceptEncoding)
		return nil
	}
}

// encoding picks br or gzip from Accept-Encoding ("" = neither offered).
func encoding(c fiber.Ctx) string {
	h := &c.Request().Header
	switch {
	case h.HasAcceptEncoding("br"):
		return "br"
	case h.HasAcceptEncoding("gzip"):
		return "gzip"
	}
	return ""
}

func compressible(c fiber.Ctx) bool {
	if c.Method() == fiber.MethodHead || c.Get(fiber.HeaderRange) != "" {
		return false
	}
	resp := c.Response()
	status := resp.StatusCode()
	if status < 200 || status >= 300 || status == fiber.StatusNoContent || status == fiber.StatusPartialContent {
		return false
	}
	if resp.IsBodyStream() || len(resp.Header.ContentEncoding()) > 0 {
		return false
	}
	if strings.Contains(strings.ToLower(c.GetRespHeader(fiber.HeaderCacheControl)), "no-transform") {
		return false
	}
	ct := strings.ToLower(string(resp.Header.ContentType()))
	if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) && !strings.Contains(ct, "+json") {
		return false
	}
	return len(resp.Body()) >= CompressMinBytes
}
