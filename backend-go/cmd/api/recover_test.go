package main

import (
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"

	apihttp "github.com/ritme/backend-go/internal/http"
)

// A handler panic (security audit of CB-REC-02, HIGH-1) answers the framework 500 and the server keeps serving; the
// log line names only the method, the masked path and the panic type — never the panic value or the body.
func TestApp_PanicIsA500AndTheServerSurvives(t *testing.T) {
	var logs strings.Builder
	deps := testDeps()
	deps.Logger = slog.New(slog.NewJSONHandler(&logs, nil))
	reg := apihttp.NewRegistry()
	reg.Register("boom", func(r fiber.Router, _ *apihttp.Deps) {
		r.Post("/api/v1/boom", func(_ fiber.Ctx) error {
			panic(errors.New("secret-health-value"))
		})
		r.Get("/api/v1/ok", func(c fiber.Ctx) error { return c.SendString("ok") })
	})
	app := newApp(reg, deps)

	for range 2 {
		req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/boom", strings.NewReader(`{"fields":[]}`))
		req.Header.Set("Accept", "application/json")
		resp := do(t, app, req)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Server Error")
	}
	resp := do(t, app, httptest.NewRequest(nethttp.MethodGet, "/api/v1/ok", nil))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, logs.String(), `"panic_type":"*errors.errorString"`)
	assert.NotContains(t, logs.String(), "secret-health-value")
	assert.NotContains(t, logs.String(), "fields")
}
