package profile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func csData(t *testing.T, r response) map[string]any {
	t.Helper()
	data, ok := r.body["data"].(map[string]any)
	require.True(t, ok, r.raw)
	return data
}

func csReminder(t *testing.T, r response, code string) map[string]any {
	t.Helper()
	for _, it := range csData(t, r)["reminders"].([]any) {
		m := it.(map[string]any)
		if m["code"] == code {
			return m
		}
	}
	t.Fatalf("reminder %s missing: %s", code, r.raw)
	return nil
}

func TestCycleSettings_DefaultsFromEngineMedians(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000101")
	// Four confirmed periods, 29 days apart, 5 days long → median 29 / 5 from 3 valid cycles.
	for _, start := range []string{"2026-06-01", "2026-06-30", "2026-07-29", "2026-08-27"} {
		_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
			VALUES (?, ?, DATE_ADD(?, INTERVAL 4 DAY), 1, 0, 'user', NOW(), NOW())`, id, start, start)
		require.NoError(t, err)
	}

	r := e.do(t, "GET", "/api/v1/profile/cycle-settings", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	l := csData(t, r)["lengths"].(map[string]any)
	assert.Equal(t, true, l["auto"])
	assert.EqualValues(t, 29, l["cycle_length"])
	assert.EqualValues(t, 5, l["period_length"])
	calc := l["calculated"].(map[string]any)
	assert.EqualValues(t, 3, calc["based_on_cycles"])

	bp := csReminder(t, r, "before_period")
	assert.Equal(t, true, bp["enabled"])
	assert.Equal(t, "09:00", bp["time"])
	assert.EqualValues(t, 2, bp["days_before"])
	assert.EqualValues(t, 27, csReminder(t, r, "pms")["cycle_day"])
	assert.Equal(t, false, csReminder(t, r, "fertile_window")["enabled"])
	assert.Equal(t, "22:00", csReminder(t, r, "daily_log")["time"])
	pill := csReminder(t, r, "pill")
	assert.Equal(t, false, pill["enabled"])
	assert.Equal(t, "21:00", pill["time"])
}

func TestCycleSettings_UpdateRoundTripAndIsolation(t *testing.T) {
	e := setup(t)
	_, alice := e.user(t, "09120000102")
	_, bob := e.user(t, "09120000103")

	r := e.do(t, "PUT", "/api/v1/profile/cycle-settings", alice, `{
		"lengths_auto": false, "cycle_length": 31, "period_length": 6,
		"reminders": {"before_period": {"days_before": 3, "time": "20:15"}, "pill": {"enabled": true, "time": "21:30"},
		              "daily_log": {"enabled": false}}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "تنظیمات سیکل ذخیره شد", r.body["message"]) // default language (no locale middleware here)
	l := csData(t, r)["lengths"].(map[string]any)
	assert.Equal(t, false, l["auto"])
	assert.EqualValues(t, 31, l["cycle_length"])
	assert.EqualValues(t, 6, l["period_length"])
	assert.EqualValues(t, 29, csReminder(t, r, "pms")["cycle_day"])
	bp := csReminder(t, r, "before_period")
	assert.Equal(t, "20:15", bp["time"])
	assert.EqualValues(t, 3, bp["days_before"])
	assert.Equal(t, true, csReminder(t, r, "pill")["enabled"])
	assert.Equal(t, false, csReminder(t, r, "daily_log")["enabled"])

	// The manual values are the profile's (same row POST /profile writes).
	var cycle, period int
	require.NoError(t, e.db.QueryRow(`SELECT p.cycle_duration, p.period_duration FROM user_profiles p JOIN users u ON u.id = p.user_id WHERE u.mobile = '09120000102'`).Scan(&cycle, &period))
	assert.Equal(t, [2]int{31, 6}, [2]int{cycle, period})

	// The notification-settings row keeps its quiet hours / neutral copy and gets the new switch values.
	var cats, sched string
	var quiet, neutral bool
	require.NoError(t, e.db.QueryRow(`SELECT n.categories, n.schedule, n.quiet_hours_enabled, n.neutral_copy FROM notification_preferences n JOIN users u ON u.id = n.user_id WHERE u.mobile = '09120000102'`).Scan(&cats, &sched, &quiet, &neutral))
	assert.JSONEq(t, `{"daily_log":false,"pill":true}`, cats)
	assert.JSONEq(t, `{"before_period":{"days_before":3,"time":"20:15"},"pill":{"time":"21:30"}}`, sched)
	assert.True(t, quiet)
	assert.True(t, neutral)

	// A later partial PUT keeps everything else.
	r = e.do(t, "PUT", "/api/v1/profile/cycle-settings", alice, `{"lengths_auto": true}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, csData(t, r)["lengths"].(map[string]any)["auto"])
	assert.Equal(t, "20:15", csReminder(t, r, "before_period")["time"])

	// Bob sees only defaults.
	r = e.do(t, "GET", "/api/v1/profile/cycle-settings", bob, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, csData(t, r)["lengths"].(map[string]any)["auto"])
	assert.Equal(t, "09:00", csReminder(t, r, "before_period")["time"])
	assert.Equal(t, false, csReminder(t, r, "pill")["enabled"])
}

func TestCycleSettings_ValidationAnd401(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000104")

	r := e.do(t, "PUT", "/api/v1/profile/cycle-settings", tok,
		`{"lengths_auto":"x","cycle_length":99,"reminders":{"pill":{"time":"25:00"},"before_period":{"days_before":9}}}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	errs := r.body["errors"].(map[string]any)
	for _, k := range []string{"lengths_auto", "cycle_length", "reminders.pill.time", "reminders.before_period.days_before"} {
		assert.Contains(t, errs, k)
	}

	r = e.do(t, "GET", "/api/v1/profile/cycle-settings", "nope", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
	r = e.do(t, "PUT", "/api/v1/profile/cycle-settings", "nope", `{}`)
	assert.Equal(t, 401, r.status)
}
