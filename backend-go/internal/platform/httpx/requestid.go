package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

// HeaderRequestID is the request-correlation header (T-M9-07): accepted from the client when valid, else generated,
// always echoed on the response and present on every log line of the request.
const HeaderRequestID = "X-Request-ID"

// maxRequestIDLen bounds an accepted incoming id.
const maxRequestIDLen = 64

type requestIDKey struct{}

// RequestIDKey is the context / fiber.Locals key of the request id.
var RequestIDKey any = requestIDKey{}

// ValidRequestID reports whether id may be adopted: 1–64 characters of [A-Za-z0-9-].
func ValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '-' {
			return false
		}
	}
	return true
}

// NewRequestID returns a random 32-hex-character id.
func NewRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never fails (Go ≥ 1.24)
	return hex.EncodeToString(b[:])
}

// RequestID is the middleware: it adopts a valid incoming X-Request-ID or generates one, echoes it, and stores it in
// the request context (for slog, see LogHandler) and in Locals (handlers passing the fiber.Ctx as a context).
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get(HeaderRequestID)
		if !ValidRequestID(id) {
			id = NewRequestID()
		}
		c.Set(HeaderRequestID, id)
		c.SetContext(context.WithValue(c.Context(), RequestIDKey, id))
		c.Locals(RequestIDKey, id)
		return c.Next()
	}
}

// RequestIDFrom returns the request id carried by ctx, or "".
func RequestIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// LogHandler wraps h so every record logged with a request context (logger.InfoContext(ctx, …)) carries a
// request_id attribute, unless the record already has one (the access log sets it itself).
func LogHandler(h slog.Handler) slog.Handler { return requestIDHandler{h} }

type requestIDHandler struct{ slog.Handler }

func (h requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" && !hasRequestID(r) {
		r = r.Clone()
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func hasRequestID(r slog.Record) bool {
	found := false
	r.Attrs(func(a slog.Attr) bool {
		found = a.Key == "request_id"
		return !found
	})
	return found
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{h.Handler.WithAttrs(attrs)}
}

func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{h.Handler.WithGroup(name)}
}
