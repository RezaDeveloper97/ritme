package http

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Admin module kill-switches: GET /modules, the onboarding / life-stage checks and the route gate.

func (e *companionEnv) switchModule(key string, on bool) {
	e.t.Helper()
	_, err := e.db.Exec(`UPDATE app_modules SET enabled = ? WHERE `+"`key`"+` = ?`, on, key)
	require.NoError(e.t, err)
	e.mr.FlushAll() // the 60 s cache of the switches
}

func TestModules_PublicSwitchesOnboardingAndLifeStage(t *testing.T) {
	e := newCompanionEnv(t)

	r := e.do(fiber.MethodGet, "/api/v1/modules", "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	for _, k := range modules.Keys {
		assert.Equal(t, true, r.data()[k], k)
	}

	e.switchModule(modules.Companion, false)
	e.switchModule(modules.Menopause, false)
	r = e.do(fiber.MethodGet, "/api/v1/modules", "", nil)
	assert.Equal(t, false, r.data()[modules.Companion])
	assert.Equal(t, false, r.data()[modules.Menopause])
	assert.Equal(t, true, r.data()[modules.TTC])

	tok := e.user("Mina", "09120009001")
	// No man's account while the companion module is off; no menopause goal while menopause is off.
	r = e.do(fiber.MethodPut, "/api/v1/onboarding/steps/gender", tok, map[string]any{"gender": "male"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/onboarding/steps/goal", tok, map[string]any{"goal": "menopause"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/onboarding/steps/name", tok, map[string]any{"name": "Mina"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/onboarding/steps/goal", tok, map[string]any{"goal": "cycle"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	// The gender question was skipped: completing makes the account a woman's.
	r = e.do(fiber.MethodPost, "/api/v1/onboarding/complete", tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "female", r.data()["gender"])

	// A stored mode whose module is off reads as cycle; switching it back on restores it.
	r = e.do(fiber.MethodPut, "/api/v1/profile/life-stage", tok, map[string]any{"mode": "menopause"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	e.switchModule(modules.Menopause, true)
	r = e.do(fiber.MethodPut, "/api/v1/profile/life-stage", tok, map[string]any{"mode": "menopause"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "menopause", r.data()["mode"])
	e.switchModule(modules.Menopause, false)
	r = e.do(fiber.MethodGet, "/api/v1/profile/life-stage", tok, nil)
	assert.Equal(t, "cycle", r.data()["mode"])
	assert.Equal(t, "menopause", r.data()["stored_mode"])

	// IVF / contraception switches cannot be turned on while their module is off.
	e.switchModule(modules.IVF, false)
	r = e.do(fiber.MethodPut, "/api/v1/profile/life-stage", tok, map[string]any{"ivf_iui": true})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
}

func TestModules_GateAnswers403ForAnOffModule(t *testing.T) {
	e := newCompanionEnv(t)
	svc := modules.NewService(e.db, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(modules.Gate(svc))
	app.Get("/*", func(c fiber.Ctx) error { return c.SendString("ok") })

	status := func(path string) int {
		res, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		require.NoError(t, err)
		_ = res.Body.Close()
		return res.StatusCode
	}
	assert.Equal(t, fiber.StatusOK, status("/api/v1/teen/today"))
	e.switchModule(modules.Teen, false)
	assert.Equal(t, fiber.StatusForbidden, status("/api/v1/teen/today"))
	assert.Equal(t, fiber.StatusOK, status("/api/v1/cycle/today"))
	assert.Equal(t, fiber.StatusOK, status("/api/v1/modules"))
}
