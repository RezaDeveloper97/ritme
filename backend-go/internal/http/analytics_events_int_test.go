package http

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analyticsevents"
)

func grantStats(t *testing.T, e *companionEnv, name string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_consents (user_id, consent, granted, version, granted_at, created_at, updated_at) VALUES (?, 'anonymous_stats', 1, 1, NOW(), NOW(), NOW())`, e.ids[name])
	require.NoError(t, err)
}

func TestAnalyticsEvents_Ingest(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120007001")
	const p = "/api/v1/analytics/events"
	ok := map[string]any{"events": []any{
		map[string]any{"name": "app_open"},
		map[string]any{"name": "log_saved", "props": map[string]any{"source": "quick"}},
		map[string]any{"name": "plus_purchase", "props": map[string]any{"plan_months": 3}},
	}}

	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, p, "", ok).Status)

	// No consent: nothing stored, the client is told to stop.
	r := e.do(fiber.MethodPost, p, sara, ok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 0, r.data()["accepted"])
	assert.Equal(t, false, r.data()["tracking"])
	assert.Zero(t, e.count("SELECT COUNT(*) FROM analytics_events"))

	grantStats(t, e, "Sara")
	r = e.do(fiber.MethodPost, p, sara, ok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 3, r.data()["accepted"])
	assert.Equal(t, true, r.data()["tracking"])
	assert.Equal(t, 3, e.count("SELECT COUNT(*) FROM analytics_events WHERE user_id = ?", e.ids["Sara"]))
	assert.Equal(t, 1, e.count("SELECT COUNT(*) FROM analytics_events WHERE name = 'log_saved' AND JSON_VALUE(props, '$.source') = 'quick'"))
}

func TestAnalyticsEvents_AllowListAndNoHealthKeys(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120007002")
	grantStats(t, e, "Sara")
	const p = "/api/v1/analytics/events"
	post := func(ev map[string]any) reply {
		return e.do(fiber.MethodPost, p, sara, map[string]any{"events": []any{ev}})
	}

	// Unknown event name.
	r := post(map[string]any{"name": "symptom_logged"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "events.0.name")

	// Health-looking and unknown props are refused, with the dedicated message for health keys.
	for _, key := range []string{"symptom", "cycle_length", "pregnancy_week", "lab_value", "mood", "weight", "note", "period_start", "random_key"} {
		r = post(map[string]any{"name": "log_saved", "props": map[string]any{key: "x"}})
		assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, key+" "+r.Raw)
		assert.Contains(t, r.Raw, "events.0.props."+key)
		if key != "random_key" {
			assert.Contains(t, r.Raw, "Health data is never accepted", key)
		}
	}
	// A known key with a free-text / out-of-range value.
	assert.Equal(t, fiber.StatusUnprocessableEntity, post(map[string]any{"name": "log_saved", "props": map[string]any{"source": "heavy bleeding day 3"}}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, post(map[string]any{"name": "plus_purchase", "props": map[string]any{"plan_months": 99}}).Status)
	// Props on an event that takes none.
	assert.Equal(t, fiber.StatusUnprocessableEntity, post(map[string]any{"name": "app_open", "props": map[string]any{"source": "quick"}}).Status)
	// Time rules.
	assert.Equal(t, fiber.StatusUnprocessableEntity, post(map[string]any{"name": "app_open", "occurred_at": "2020-01-01T00:00:00Z"}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, post(map[string]any{"name": "app_open", "occurred_at": "2999-01-01T00:00:00Z"}).Status)
	// Shape: empty batch, too many.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p, sara, map[string]any{"events": []any{}}).Status)
	many := make([]any, analyticsevents.MaxBatch+1)
	for i := range many {
		many[i] = map[string]any{"name": "app_open"}
	}
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p, sara, map[string]any{"events": many}).Status)

	// One bad event drops the whole batch: nothing stored.
	r = e.do(fiber.MethodPost, p, sara, map[string]any{"events": []any{map[string]any{"name": "app_open"}, map[string]any{"name": "nope"}}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Zero(t, e.count("SELECT COUNT(*) FROM analytics_events"))

	// Structural guarantee: no allow-listed prop key looks like health data.
	for name, props := range analyticsevents.Spec {
		for k := range props {
			assert.False(t, analyticsevents.LooksLikeHealth(k), name+"."+k)
		}
	}
}

func TestAnalyticsEvents_RateLimited(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120007003")
	body := map[string]any{"events": []any{map[string]any{"name": "app_open"}}}
	for i := 0; i < AnalyticsEventsMax; i++ {
		r := e.do(fiber.MethodPost, "/api/v1/analytics/events", sara, body)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	}
	r := e.do(fiber.MethodPost, "/api/v1/analytics/events", sara, body)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	// Another user has a counter of their own.
	ali := e.user("Ali", "09120007004")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/analytics/events", ali, body).Status)
}
