package join

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// parseBody runs parseStep on a JSON body; the payloads here fail rule validation before any store access.
func parseBody(t *testing.T, step int, locale, body string) error {
	t.Helper()
	h := &Handlers{}
	var got error
	app := fiber.New()
	app.Post("/:step", func(c fiber.Ctx) error {
		n, _ := strconv.Atoi(c.Params("step"))
		_, got = h.parseStep(c, n, locale, time.Now())
		return nil
	})
	req := httptest.NewRequest(fiber.MethodPost, "/"+strconv.Itoa(step), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, 200, resp.StatusCode, "parseStep must not panic")
	return got
}

func errorKeys(t *testing.T, err error) map[string][]string {
	t.Helper()
	var fe *httpx.FailError
	require.True(t, errors.As(err, &fe), "want a FailError, got %v", err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, fe.Status)
	require.Len(t, fe.Extras, 2)
	raw, mErr := json.Marshal(fe.Extras[1])
	require.NoError(t, mErr)
	out := map[string][]string{}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// L-1: a JSON object passes `array` but the parser walks lists by index (nil deref on step 2, rule bypass on 3).
func TestParseStep_ObjectInsteadOfListIs422(t *testing.T) {
	for _, locale := range []string{"en", "fa"} {
		errs := errorKeys(t, parseBody(t, 2, locale, `{"weekly_hours":{"a":{"day":1,"open":"09:00","close":"18:00"}}}`))
		require.Contains(t, errs, "weekly_hours", locale)
		assert.NotContains(t, errs["weekly_hours"][0], "validation.", "message is translated (%s)", locale)

		errs = errorKeys(t, parseBody(t, 3, locale, `{"services":{"x":{"kind":"nope"}}}`))
		require.Contains(t, errs, "services", locale)
	}
	// Items missing required fields fail per item instead of panicking.
	errs := errorKeys(t, parseBody(t, 2, "en", `{"weekly_hours":[{"open":"09:00","close":"18:00"}]}`))
	assert.Contains(t, errs, "weekly_hours.0.day")
	errs = errorKeys(t, parseBody(t, 3, "en", `{"services":[{"kind":"session"}]}`))
	assert.Contains(t, errs, "services.0.name")
}

func TestNumOr(t *testing.T) {
	m := phpval.NewMap()
	m.Set("n", float64(3))
	assert.Equal(t, int64(-1), numOr(m, "missing", -1))
	assert.Equal(t, int64(3), numOr(m, "n", -1))
}
