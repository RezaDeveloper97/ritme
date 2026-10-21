package httpx_test

import (
	"bytes"
	"context"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestValidRequestID(t *testing.T) {
	for id, want := range map[string]bool{
		"abc-123-DEF":            true,
		strings.Repeat("a", 64):  true,
		"":                       false,
		strings.Repeat("a", 65):  false,
		"has space":              false,
		"semi;colon":             false,
		"under_score":            false,
		"line\nbreak":            false,
		"ünïcode":                false,
		"0123456789abcdef-ABCDE": true,
	} {
		assert.Equal(t, want, httpx.ValidRequestID(id), "%q", id)
	}
	assert.Len(t, httpx.NewRequestID(), 32)
	assert.True(t, httpx.ValidRequestID(httpx.NewRequestID()))
	assert.NotEqual(t, httpx.NewRequestID(), httpx.NewRequestID())
}

func TestRequestID_EchoedOrGenerated(t *testing.T) {
	var seenCtx, seenLocals string
	app := fiber.New()
	app.Use(httpx.RequestID())
	app.Get("/", func(c fiber.Ctx) error {
		seenCtx = httpx.RequestIDFrom(c.Context())
		seenLocals = httpx.RequestIDFrom(c) // the fiber.Ctx as a context reads Locals
		return c.SendString("ok")
	})

	call := func(header string) string {
		req := httptest.NewRequest(nethttp.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set(httpx.HeaderRequestID, header)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.Header.Get(httpx.HeaderRequestID)
	}

	assert.Equal(t, "client-id-42", call("client-id-42"), "a valid id is echoed")
	assert.Equal(t, "client-id-42", seenCtx)
	assert.Equal(t, "client-id-42", seenLocals)

	for _, bad := range []string{"", "bad id", "<script>", strings.Repeat("x", 65)} {
		got := call(bad)
		assert.NotEqual(t, bad, got)
		assert.True(t, httpx.ValidRequestID(got), "generated id for %q", bad)
		assert.Equal(t, got, seenCtx)
	}
}

func TestLogHandler_AddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(httpx.LogHandler(slog.NewJSONHandler(&buf, nil))).With("svc", "api")
	ctx := context.WithValue(context.Background(), httpx.RequestIDKey, "rid-1")

	logger.InfoContext(ctx, "hello")
	assert.Contains(t, buf.String(), `"request_id":"rid-1"`)
	assert.Contains(t, buf.String(), `"svc":"api"`)

	buf.Reset()
	logger.InfoContext(ctx, "own", slog.String("request_id", "rid-1"))
	assert.Equal(t, 1, strings.Count(buf.String(), "request_id"), "not duplicated when the record has one")

	buf.Reset()
	logger.Info("no request")
	assert.NotContains(t, buf.String(), "request_id")
}
