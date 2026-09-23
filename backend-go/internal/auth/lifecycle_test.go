package auth_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMustGuard_MissingKeysRefusesToStart(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	g := auth.MustGuard(app, &config.Config{StoragePath: t.TempDir()}, nil, quiet)
	app.Get("/x", g.RequireUser, func(c fiber.Ctx) error { return c.SendString("ok") })

	// Requests (only possible in tests) are 500, never a 401 that would log clients out.
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/x", nil))
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 500, resp.StatusCode)

	// Listen aborts before serving.
	assert.Panics(t, func() { _ = app.Listen("127.0.0.1:0", fiber.ListenConfig{DisableStartupMessage: true}) })
}

func TestMustModule_MissingKeysRefusesToStart(t *testing.T) {
	app := fiber.New()
	m := auth.MustModule(app, auth.Deps{Config: &config.Config{StoragePath: t.TempDir()}, Logger: quiet})
	assert.Nil(t, m)
	assert.Panics(t, func() { _ = app.Listen("127.0.0.1:0", fiber.ListenConfig{DisableStartupMessage: true}) })
}
