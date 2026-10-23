package auth_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// T-M9-31: RequireUser exposes the user's revisions from the auth query (0 / 0 without a user_revisions row).
func TestRequireUser_CurrentRevisions(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234599", t0)

	guard := auth.NewGuardWith(&testKey.PublicKey, store.New(e.db), clock.Real{}, quiet)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Get("/r", guard.RequireUser, func(c fiber.Ctx) error {
		r, ok := auth.CurrentRevisions(c)
		require.True(t, ok)
		return c.JSON(fiber.Map{"engine": r.Engine, "analysis": r.Analysis})
	})
	get := func() string {
		req := httptest.NewRequest(http.MethodGet, "/r", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, 200, resp.StatusCode, string(raw))
		return string(raw)
	}
	assert.JSONEq(t, `{"engine":0,"analysis":0}`, get())

	_, err := e.db.Exec(`INSERT INTO user_revisions (user_id, engine_rev, analysis_rev, updated_at)
		SELECT id, 4, 6, NOW() FROM users WHERE mobile = '09121234599'`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"engine":4,"analysis":6}`, get())
}
