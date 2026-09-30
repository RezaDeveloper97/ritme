package home

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func d(s string) civildate.Date { return civildate.MustParse(s) }

func period(start, end string) model.History {
	return model.History{PeriodStart: d(start), PeriodEnd: d(end), IsConfirmed: true, Source: "user_logged"}
}

func TestLoggingStreak(t *testing.T) {
	today := d("2026-10-01")
	days := []civildate.Date{d("2026-10-01"), d("2026-09-30"), d("2026-09-29"), d("2026-09-27"), d("2026-09-30")}
	n, logged := loggingStreak(days, today)
	assert.Equal(t, 3, n)
	assert.True(t, logged)

	// Today not logged yet: the run ending yesterday still counts.
	n, logged = loggingStreak(days[1:], today)
	assert.Equal(t, 2, n)
	assert.False(t, logged)

	n, _ = loggingStreak(nil, today)
	assert.Zero(t, n)
}

func TestBuildCycleOverviewRegular(t *testing.T) {
	hist := []model.History{
		period("2026-09-14", "2026-09-18"), period("2026-08-15", "2026-08-19"),
		period("2026-07-17", "2026-07-21"), period("2026-06-19", "2026-06-23"),
	}
	o := BuildCycleOverview(hist, nil, []civildate.Date{d("2026-10-01")}, d("2026-10-01"))

	assert.Equal(t, enums.RegularityStatusRelativelyRegular, o.Regularity)
	assert.Equal(t, 3, o.BasedOnCycles)
	require.NotNil(t, o.Spread)
	assert.Equal(t, 1, *o.Spread) // lengths 28, 29, 30 → range 2 → ±1
	assert.Equal(t, 29, o.CycleLength)
	assert.Equal(t, d("2026-10-13"), o.NextPeriodStart)
	assert.Equal(t, d("2026-10-17"), o.NextPeriodEnd)
	assert.Equal(t, d("2026-10-10"), o.PmsStart)
	assert.Equal(t, d("2026-10-12"), o.PmsEnd)
	assert.Equal(t, 1, o.StreakDays)

	raw, err := json.Marshal(o.JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-10-01","logging":{"streak_days":1,"logged_today":true},
		"next_period":{"start":"2026-10-13","end":"2026-10-17"},"pms_window":{"start":"2026-10-10","end":"2026-10-12"},
		"cycle_length":{"median":29,"source":"recent_valid_cycles","based_on_cycles":3,"spread_days":1,
		"regularity":"relatively_regular"}}`, string(raw))
}

func TestBuildCycleOverviewNoData(t *testing.T) {
	o := BuildCycleOverview(nil, nil, nil, d("2026-10-01"))
	raw, err := json.Marshal(o.JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-10-01","logging":{"streak_days":0,"logged_today":false},
		"next_period":null,"pms_window":null,
		"cycle_length":{"median":28,"source":"default","based_on_cycles":null,"spread_days":null,
		"regularity":"not_enough_data"}}`, string(raw))
}

// No bearer user on the context → the shared 401 body, before anything is loaded.
func TestCycleOverviewUnauthenticated(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	h := &Handlers{}
	app.Get("/api/v1/home/cycle-overview", h.CycleOverview)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/home/cycle-overview", nil))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "Unauthenticated.", body["message"])
	assert.Equal(t, "unauthenticated", body["error_code"])
}
