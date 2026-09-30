package profile

import (
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// The hourly limiter answers the 6th report of a user with a localized 429 before the handler (and any decode)
// runs; another user has their own budget.
func TestSupportReportLimiter(t *testing.T) {
	mr := miniredis.RunT(t)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	reached := 0
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	// The user id comes from a header here (stand-in for auth.ThrottleIdentity).
	app.Post("/r", SupportReportLimiterWith(c, clock.Real{}, func(ctx fiber.Ctx) string { return ctx.Get("X-User") }),
		func(ctx fiber.Ctx) error { reached++; return ctx.SendStatus(201) })
	post := func(user string) int {
		req := httptest.NewRequest(fiber.MethodPost, "/r", nil)
		req.Header.Set("X-User", user)
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp.StatusCode
	}
	for i := range SupportReportsPerHour {
		require.Equal(t, 201, post("1"), "report %d", i)
	}
	assert.Equal(t, 429, post("1"))
	assert.Equal(t, SupportReportsPerHour, reached, "the rejected request never reached the handler")
	assert.Equal(t, 201, post("2"))
}

func TestSupportReportLimiter_NoCacheIsNoop(t *testing.T) {
	app := fiber.New()
	app.Post("/r", SupportReportLimiter(nil, clock.Real{}), func(ctx fiber.Ctx) error { return ctx.SendStatus(201) })
	for range SupportReportsPerHour + 2 {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/r", nil))
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	}
}

func TestRemoveSupportFiles_OnlyTheSupportDirectory(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) string {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
		require.NoError(t, os.WriteFile(abs, []byte("x"), 0o600))
		return abs
	}
	shot := write(supportReportDir + "/a.webp")
	other := write("app/public/banner.webp")
	RemoveSupportFiles(root, []sql.NullString{
		{String: supportReportDir + "/a.webp", Valid: true},
		{String: supportReportDir + "/missing.webp", Valid: true}, // already gone: fine
		{String: "app/public/banner.webp", Valid: true},           // outside the directory: untouched
		{String: supportReportDir + "/../../public/banner.webp", Valid: true},
	}, nil)
	_, err := os.Stat(shot)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(other)
	assert.NoError(t, err)
}
