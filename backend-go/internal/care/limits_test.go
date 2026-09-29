package care

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

// TestTooManyWrites: the write throttle's 429 is the controller envelope in the request
// language (a language without the line falls back to English), with retry_after and the
// throttle headers (T-M7-23).
func TestTooManyWrites(t *testing.T) {
	headers := map[string]string{
		"Retry-After": "42", "X-RateLimit-Limit": "60", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1790145042",
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Post("/w", func(c fiber.Ctx) error { return TooManyWrites(c.Query("locale"), 42, headers) })

	en := "Too many changes in the last minute. Wait a moment and save again."
	for _, tc := range []struct{ locale, msg string }{
		{"fa", "یه لحظه صبر کن و دوباره ذخیره کن؛ توی این یک دقیقه تغییرهای زیادی فرستاده شده."},
		{"en", en},
		{"ar", en},
	} {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/w?locale="+tc.locale, nil))
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, tc.locale)
		var got map[string]any
		require.NoError(t, json.Unmarshal(body, &got), string(body))
		assert.Equal(t, map[string]any{
			"success": false, "message": tc.msg, "error_code": "too_many_requests", "retry_after": float64(42),
		}, got, tc.locale)
		for k, v := range headers {
			assert.Equal(t, v, resp.Header.Get(k), k)
		}
	}
}
