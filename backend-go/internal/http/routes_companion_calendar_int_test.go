package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *companionEnv) calendar(token, query string) reply {
	e.t.Helper()
	return e.do(fiber.MethodGet, "/api/v1/companion/calendar"+query, token, nil)
}

func periodDaysOf(t *testing.T, r reply) any {
	t.Helper()
	ps, _ := r.data()["partners"].([]any)
	require.Len(t, ps, 1, r.Raw)
	return ps[0].(map[string]any)["period_days"]
}

// GET /companion/calendar: her likely period days reach the companion only with her per-link consent
// (period_days) on top of the cycle grant, today onwards only; everything else reads null, and every returned list
// is audited as a cycle read.
func TestCompanionCalendar_ConsentGrantAndWindow(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120004001")
	ali := e.user("Ali", "09120004002")
	saraID := e.ids["Sara"]
	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(sara, "goal", map[string]any{"goal": "cycle"})
	// companionNow = 2026-09-23: next start predicted 2026-10-03.
	e.step(sara, "cycle", map[string]any{"last_period_start": "2026-09-05", "period_duration": 5, "cycle_duration": 28})
	e.step(ali, "gender", map[string]any{"gender": "male"})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/onboarding/complete", ali, nil).Status)

	// A woman's account has no companion calendar; without a token it is 401.
	r := e.calendar(sara, "")
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "not_companion_account", r.Body["error_code"])
	assert.Equal(t, fiber.StatusUnauthorized, e.calendar("", "").Status)

	// No link: an empty list, default window today … +41.
	r = e.calendar(ali, "")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "2026-09-23", r.data()["from"])
	assert.Equal(t, "2026-11-03", r.data()["to"])
	assert.Empty(t, r.data()["partners"])

	// Bad windows: 422.
	for _, q := range []string{"?from=2026-10-10&to=2026-10-01", "?from=2026-09-01&to=2026-12-31", "?from=nope"} {
		assert.Equal(t, fiber.StatusUnprocessableEntity, e.calendar(ali, q).Status, q)
	}

	link, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"cycle": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)

	// Cycle shared, consent off (the default): null, nothing audited.
	r = e.calendar(ali, "?from=2026-09-01&to=2026-10-31")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Nil(t, periodDaysOf(t, r))
	assert.Equal(t, "Sara", r.data()["partners"].([]any)[0].(map[string]any)["partner_name"])
	audited := func() int {
		return e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND action = 'read' AND section = 'cycle'`, saraID, e.ids["Ali"])
	}
	assert.Equal(t, 0, audited())

	// She turns it on: the owner echo carries the flag, the companion sees predicted days from today on.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara, map[string]any{"period_days": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["period_days"])
	r = e.calendar(ali, "?from=2026-09-01&to=2026-10-31")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	days, ok := periodDaysOf(t, r).([]any)
	require.True(t, ok, r.Raw)
	require.NotEmpty(t, days, r.Raw)
	assert.Equal(t, "2026-10-03", days[0], "the next predicted start")
	for _, d := range days {
		assert.GreaterOrEqual(t, d.(string), "2026-09-23", "never a past (logged) day")
	}
	assert.NotContains(t, r.Raw, "2026-09-05", "her logged period never leaks")
	for _, k := range []string{"cycle_day", "cycle_length", "confidence", "days_late"} {
		assert.NotContains(t, r.Raw, k)
	}
	assert.Equal(t, 1, audited())

	// The grant is the floor: consent on but cycle unshared → null again.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", link), sara,
		map[string]any{"grants": map[string]any{}}).Status)
	assert.Nil(t, periodDaysOf(t, e.calendar(ali, "?from=2026-09-01&to=2026-10-31")))

	// Turning the consent off with the grant back: null.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", link), sara,
		map[string]any{"grants": map[string]any{"cycle": "view"}}).Status)
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara,
		map[string]any{"period_days": false}).Status)
	assert.Nil(t, periodDaysOf(t, e.calendar(ali, "?from=2026-09-01&to=2026-10-31")))
}
