package clock

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestReal_IsTehran(t *testing.T) {
	now := Real{}.Now()
	assert.Equal(t, "Asia/Tehran", now.Location().String())
	assert.WithinDuration(t, time.Now(), now, time.Second)
}

func TestFixed(t *testing.T) {
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	assert.Equal(t, at, At(at).Now())
	assert.Equal(t, "2026-09-23", civildate.Today(At(at)).String())
}

// Mirrors TestClock::isEnabled(): app()->environment(['local','testing','contract'])
// && filter_var(env('TEST_CLOCK_ENABLED', false), FILTER_VALIDATE_BOOLEAN).
//
//	php > var_dump(filter_var('yes', FILTER_VALIDATE_BOOLEAN), filter_var('On', FILTER_VALIDATE_BOOLEAN),
//	php >          filter_var('0', FILTER_VALIDATE_BOOLEAN), filter_var('', FILTER_VALIDATE_BOOLEAN));
//	bool(true) bool(true) bool(false) bool(false)
func TestTestClockEnabled(t *testing.T) {
	cases := []struct {
		env, flag string
		want      bool
	}{
		{"local", "true", true},
		{"testing", "1", true},
		{"contract", "On", true},
		{"contract", " yes ", true},
		{"production", "true", false},
		{"staging", "true", false},
		{"", "true", false},
		{"local", "", false},
		{"local", "0", false},
		{"local", "false", false},
		{"local", "nope", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, TestClockEnabled(c.env, c.flag), "%s/%s", c.env, c.flag)
	}
	t.Setenv("APP_ENV", "contract")
	t.Setenv("TEST_CLOCK_ENABLED", "true")
	assert.True(t, TestClockEnabledFromEnv())
	t.Setenv("APP_ENV", "production")
	assert.False(t, TestClockEnabledFromEnv())
}

func TestFromContext_Fallback(t *testing.T) {
	base := At(time.Unix(0, 0))
	//nolint:staticcheck // nil context is exactly the case under test
	assert.Equal(t, base, FromContext(nil, base))
	assert.Equal(t, base, FromContext(context.Background(), base))
	other := At(time.Unix(100, 0))
	assert.Equal(t, other, FromContext(WithClock(context.Background(), other), base))
}

func newApp(enabled bool, base Clock) *fiber.App {
	app := fiber.New()
	app.Use(Middleware(base, enabled))
	app.Get("/now", func(c fiber.Ctx) error {
		fromCtx := FromContext(c, nil).Now()
		fromStd := FromContext(c.Context(), nil).Now()
		if !fromCtx.Equal(fromStd) {
			return fiber.ErrInternalServerError
		}
		return c.SendString(fromCtx.Format(time.RFC3339Nano))
	})
	return app
}

func get(t *testing.T, app *fiber.App, header string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/now", nil)
	if header != "" {
		req.Header.Set(Header, header)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func TestMiddleware(t *testing.T) {
	base := At(time.Date(2026, 1, 1, 12, 0, 0, 0, civildate.Tehran))

	// Enabled: ISO with offset, without offset (Tehran wall-clock), DST era.
	app := newApp(true, base)
	status, body := get(t, app, "2026-09-23T10:00:00Z")
	assert.Equal(t, 200, status)
	assert.Equal(t, "2026-09-23T13:30:00+03:30", body)
	_, body = get(t, app, "2026-09-23 09:15:00")
	assert.Equal(t, "2026-09-23T09:15:00+03:30", body)
	_, body = get(t, app, "2021-06-01T08:00:00")
	assert.Equal(t, "2021-06-01T08:00:00+04:30", body)
	_, body = get(t, app, "")
	assert.Equal(t, "2026-01-01T12:00:00+03:30", body)

	// Laravel: response()->json(['message' => 'Invalid X-Test-Now header.'], 400)
	status, body = get(t, app, "garbage")
	assert.Equal(t, 400, status)
	assert.JSONEq(t, `{"message":"Invalid X-Test-Now header."}`, body)
	assert.Equal(t, `{"message":"Invalid X-Test-Now header."}`, body)

	// Disabled: the header is ignored, even an invalid one.
	app = newApp(false, base)
	status, body = get(t, app, "2026-09-23T10:00:00Z")
	assert.Equal(t, 200, status)
	assert.Equal(t, "2026-01-01T12:00:00+03:30", body)
	status, _ = get(t, app, "garbage")
	assert.Equal(t, 200, status)
}
