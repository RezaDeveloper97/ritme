package http_test

// T-M12-15 (T-05): the first-day path through the real routes, end to end — OTP sign-up → onboarding → period
// start → period end → /cycle/today and /cycle/month → edit the period — with the expected days, phases and
// predictions worked out from docs/engine-v6/SPEC.md §2 (and the accepted DECISIONS.md E-xx), not copied from the API.
// Every engine GET is revalidated with its ETag: 304 while nothing changed, a new tag after each write.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/http/testenv"
)

const (
	pathToday   = "/api/v1/cycle/today"
	pathHistory = "/api/v1/cycle/period/history"
)

// cycleView is data.cycle_view of a /cycle/today reply.
func cycleView(t *testing.T, r testenv.Reply) map[string]any {
	t.Helper()
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cv, ok := r.Data()["cycle_view"].(map[string]any)
	require.True(t, ok, r.Raw)
	return cv
}

// expect compares dotted paths of a decoded object ("anchors.predicted_next_period_start") with the wanted values;
// ints are compared as JSON numbers, nil means JSON null.
func expect(t *testing.T, label string, obj map[string]any, want map[string]any) {
	t.Helper()
	for path, w := range want {
		got := testenv.Path(obj, strings.Split(path, ".")...)
		if i, ok := w.(int); ok {
			w = float64(i)
		}
		assert.Equal(t, w, got, "%s: %s", label, path)
	}
}

// strs is a JSON list of strings.
func strs(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, x := range l {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// getToday is GET /cycle/today at now, optionally conditional.
func getToday(e *testenv.Env, tok, now, ifNoneMatch string) testenv.Reply {
	e.T.Helper()
	h := map[string]string{}
	if ifNoneMatch != "" {
		h[fiber.HeaderIfNoneMatch] = ifNoneMatch
	}
	return e.Send(testenv.Request{Method: fiber.MethodGet, Path: pathToday, Token: tok, Now: now, Header: h})
}

// monthDays is GET /cycle/month/{y}/{m}?view=calendar keyed by calculation_date.
func monthDays(t *testing.T, e *testenv.Env, tok string, year, month int) map[string]map[string]any {
	t.Helper()
	r := e.MustOK(fiber.StatusOK, fiber.MethodGet, fmt.Sprintf("/api/v1/cycle/month/%d/%d?view=calendar", year, month), nil, tok)
	list, ok := r.Data()["calculations"].([]any)
	require.True(t, ok, r.Raw)
	out := map[string]map[string]any{}
	for _, x := range list {
		d, _ := x.(map[string]any)
		date, _ := d["calculation_date"].(string)
		out[date] = d
	}
	return out
}

// onboarding answers the onboarding steps of a woman tracking her cycle; cycle (the cycle step body) and birthday
// are optional.
func onboard(t *testing.T, e *testenv.Env, tok string, cycle map[string]any, birthday string) {
	t.Helper()
	e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/name", map[string]any{"name": "Sara"}, tok)
	e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/gender", map[string]any{"gender": "female"}, tok)
	e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/goal", map[string]any{"goal": "cycle"}, tok)
	if cycle != nil {
		e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/cycle", cycle, tok)
	}
	if birthday != "" {
		e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/health", map[string]any{"birthday": birthday}, tok)
	}
	r := e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/onboarding/complete", nil, tok)
	assert.Equal(t, true, r.Data()["completed"], r.Raw)
}

// logPeriod stores a past period with POST /cycle/period.
func logPeriod(t *testing.T, e *testenv.Env, tok, start, end string) {
	t.Helper()
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/cycle/period", map[string]any{"start_date": start, "end_date": end}, tok)
}

func TestRoutine_SignupOnboardingPeriodPrediction(t *testing.T) {
	e := testenv.New(t)
	const t0 = "2026-09-23T10:00:00+03:30" // 1405-07-01
	e.SetNow(t0)

	// --- OTP sign-up ---
	mobile := testenv.Mobile()
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/auth/send-otp", map[string]any{"mobile": mobile}, "")
	code := e.OTP(mobile)
	require.Len(t, code, 4)
	wrong := "1000"
	if code == wrong {
		wrong = "1001"
	}
	r := e.Do(fiber.MethodPost, "/api/v1/auth/verify-otp", map[string]any{"mobile": mobile, "code": wrong}, "")
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "a wrong code: %s", r.Raw)
	// The right code without the mandatory consents creates nothing and names terms + health_data_processing.
	consents := e.SignupConsents(mobile, code)
	assert.Contains(t, consents, "terms")
	assert.Contains(t, consents, "health_data_processing")
	assert.Zero(t, e.Count(`SELECT COUNT(*) FROM users WHERE mobile = ?`, mobile), "no account without consents")
	r = e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/auth/verify-otp", map[string]any{"mobile": mobile, "code": code, "consents": consents}, "")
	assert.Equal(t, true, r.Data()["new_user"])
	assert.Equal(t, false, r.Data()["profile_completed"])
	tok, _ := r.Data()["access_token"].(string)
	require.NotEmpty(t, tok)
	// The code is spent.
	r = e.Do(fiber.MethodPost, "/api/v1/auth/verify-otp", map[string]any{"mobile": mobile, "code": code, "consents": consents}, "")
	assert.Equal(t, fiber.StatusBadRequest, r.Status, "a spent code: %s", r.Raw)
	r = e.Do(fiber.MethodGet, pathToday, nil, "")
	assert.Equal(t, fiber.StatusUnauthorized, r.Status)
	assert.Equal(t, "unauthenticated", r.Body["error_code"])

	// --- Onboarding: regular 28 / 5, last period 2026-08-26 (declared, not a log) ---
	r = e.Do(fiber.MethodPut, "/api/v1/onboarding/steps/cycle", map[string]any{"last_period_start": "2026-09-24"}, tok)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "a last period after today: %s", r.Raw)
	onboard(t, e, tok, map[string]any{"last_period_start": "2026-08-26", "period_duration": 5, "cycle_duration": 28}, "")
	r = e.MustOK(fiber.StatusOK, fiber.MethodGet, "/api/v1/auth/user", nil, tok)
	assert.Equal(t, true, r.Data()["profile_completed"])

	// Today = declared start + 28 = the predicted start: day 29, period_expected, not late (§2.22). Ovulation is
	// next − 14 (§2.17 calendar), the end is assumed from the declared length (§2.11), all onboarding-based (§2.30).
	r0 := getToday(e, tok, t0, "")
	expect(t, "after onboarding", cycleView(t, r0), map[string]any{
		"date": "2026-09-23", "cycle_day": 29, "main_phase": "period_expected", "subphase": "period_expected",
		"fertility_level": "unknown", "days_late": 0, "days_to_period": 0,
		"anchors.current_period_start": "2026-08-26", "anchors.current_period_start_source": "onboarding_declared",
		"anchors.current_period_end": "2026-08-30", "anchors.current_period_end_source": "assumed_from_effective_duration",
		"anchors.current_period_end_is_confirmed": false,
		"anchors.predicted_next_period_start":     "2026-09-23", "anchors.estimated_ovulation_date": "2026-09-09",
		"metrics.effective_cycle_length": 28, "metrics.effective_period_length": 5,
		"resolution_source": "onboarding_based", "data_quality": "partial",
	})
	// E-04: while today ≥ the predicted start with no new start, future_cycles[1] starts tomorrow.
	fc := strs(nil)
	for _, x := range cycleView(t, r0)["future_cycles"].([]any) {
		fc = append(fc, testenv.Path(x, "period_start").(string))
	}
	assert.Equal(t, []string{"2026-09-24", "2026-10-22", "2026-11-19", "2026-12-17", "2027-01-14", "2027-02-11"}, fc)
	e0 := r0.ETag()
	require.NotEmpty(t, e0)
	assert.Equal(t, fiber.StatusNotModified, getToday(e, tok, t0, e0).Status, "nothing changed: 304")

	// A refused write changes nothing: the tag still revalidates.
	r = e.Do(fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2026-09-24"}, tok)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "a start after today: %s", r.Raw)
	assert.Equal(t, fiber.StatusNotModified, getToday(e, tok, t0, e0).Status, "a refused write keeps the tag")

	// --- Period start today ---
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2026-09-23"}, tok)
	r1 := getToday(e, tok, t0, e0)
	expect(t, "period day 1", cycleView(t, r1), map[string]any{
		"date": "2026-09-23", "cycle_day": 1, "main_phase": "menstrual", "subphase": "menstrual", "fertility_level": "low",
		"anchors.current_period_start": "2026-09-23", "anchors.current_period_start_source": "user_logged",
		"anchors.current_period_end": "2026-09-27", "anchors.current_period_end_source": "assumed_from_effective_duration",
		"anchors.current_period_end_is_confirmed": false,
		// No valid actual cycle yet (the declared start is not a log, §2.9) → the declared 28.
		"anchors.predicted_next_period_start": "2026-10-21", "anchors.estimated_ovulation_date": "2026-10-07",
		"days_late": 0, "days_to_period": 28, "resolution_source": "user_logged_with_assumed_end",
	})
	e1 := r1.ETag()
	require.NotEmpty(t, e1)
	assert.NotEqual(t, e0, e1, "a write moves the tag")
	assert.Equal(t, fiber.StatusNotModified, getToday(e, tok, t0, e1).Status)

	// --- Period end on day 5 ---
	const t2 = "2026-09-27T20:00:00+03:30"
	e.SetNow(t2)
	r = e.Do(fiber.MethodPost, "/api/v1/cycle/period/end", map[string]any{"date": "2026-09-22"}, tok)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "an end before the start: %s", r.Raw)
	r2 := getToday(e, tok, t2, "")
	e2pre := r2.ETag()
	expect(t, "day 5, end not logged yet", cycleView(t, r2), map[string]any{
		"cycle_day": 5, "main_phase": "menstrual", "anchors.current_period_end_is_confirmed": false,
	})
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/cycle/period/end", map[string]any{"date": "2026-09-27"}, tok)
	r2 = getToday(e, tok, t2, e2pre)
	expect(t, "day 5, end logged", cycleView(t, r2), map[string]any{
		"date": "2026-09-27", "cycle_day": 5, "main_phase": "menstrual", "subphase": "menstrual",
		"anchors.current_period_end": "2026-09-27", "anchors.current_period_end_source": "user_logged",
		"anchors.current_period_end_is_confirmed": true, "resolution_source": "user_logged",
		"anchors.predicted_next_period_start": "2026-10-21",
	})
	assert.NotEqual(t, e2pre, r2.ETag())

	// --- 2026-10-05 = day 13: the §2.36 28-day table — high_fertility on days 13–14, ovulation day 15 ---
	const t3 = "2026-10-05T09:00:00+03:30"
	e.SetNow(t3)
	r3 := getToday(e, tok, t3, "")
	expect(t, "day 13", cycleView(t, r3), map[string]any{
		"date": "2026-10-05", "cycle_day": 13, "main_phase": "fertile", "subphase": "high_fertility",
		"fertility_level": "high", "days_to_ovulation": 2, "days_to_period": 16, "resolution_source": "prediction",
		"anchors.estimated_ovulation_date": "2026-10-07",
	})
	e3 := r3.ETag()
	assert.Equal(t, fiber.StatusNotModified, getToday(e, tok, t3, e3).Status)

	// The calendar of October: the §2.36 28-day layout from 2026-09-23, then the next period from future_cycles (E-03).
	days := monthDays(t, e, tok, 2026, 10)
	require.Len(t, days, 31)
	for date, want := range map[string]map[string]any{
		"2026-10-01": {"cycle_day": 9, "main_phase": "follicular", "subphase": "late_follicular_transition", "is_period": false},
		"2026-10-02": {"cycle_day": 10, "main_phase": "fertile", "subphase": "fertile_rising"},
		"2026-10-07": {"cycle_day": 15, "main_phase": "fertile", "subphase": "ovulation_likely", "fertility_level": "peak"},
		"2026-10-08": {"cycle_day": 16, "main_phase": "luteal", "subphase": "post_ovulation"},
		"2026-10-18": {"cycle_day": 26, "main_phase": "luteal", "subphase": "pms_possible"},
		"2026-10-21": {"cycle_day": 1, "main_phase": "menstrual", "is_period": true, "is_predicted": true},
		"2026-10-25": {"cycle_day": 5, "main_phase": "menstrual", "is_period": true},
		"2026-10-26": {"cycle_day": 6, "is_period": false},
	} {
		require.Contains(t, days, date)
		expect(t, "month "+date, days[date], want)
	}

	// --- Edit the period two days earlier: every prediction moves two days earlier ---
	r = e.MustOK(fiber.StatusOK, fiber.MethodGet, pathHistory, nil, tok)
	var id float64
	for _, p := range r.Data()["periods"].([]any) {
		if testenv.Path(p, "period_start_date") == "2026-09-23" {
			id, _ = testenv.Path(p, "id").(float64)
		}
	}
	require.NotZero(t, id, "the logged period in the history: %s", r.Raw)
	e.MustOK(fiber.StatusOK, fiber.MethodPut, fmt.Sprintf("/api/v1/cycle/period/%d", int(id)),
		map[string]any{"start_date": "2026-09-21", "end_date": "2026-09-25"}, tok)
	r4 := getToday(e, tok, t3, e3)
	expect(t, "after the edit", cycleView(t, r4), map[string]any{
		"date": "2026-10-05", "cycle_day": 15, "main_phase": "fertile", "subphase": "ovulation_likely",
		"fertility_level": "peak", "days_to_ovulation": 0,
		"anchors.current_period_start": "2026-09-21", "anchors.current_period_end": "2026-09-25",
		// Still no valid actual cycle (one logged period; the declared 2026-08-26 is not one) → 28.
		"metrics.effective_cycle_length":      28,
		"anchors.predicted_next_period_start": "2026-10-19", "anchors.estimated_ovulation_date": "2026-10-05",
	})
	assert.NotEqual(t, e3, r4.ETag(), "the edit moves the tag")
	days = monthDays(t, e, tok, 2026, 10)
	expect(t, "month after edit 2026-10-19", days["2026-10-19"], map[string]any{"cycle_day": 1, "is_period": true})
	expect(t, "month after edit 2026-10-21", days["2026-10-21"], map[string]any{"cycle_day": 3, "is_period": true})
	expect(t, "month after edit 2026-10-24", days["2026-10-24"], map[string]any{"cycle_day": 6, "is_period": false})
}

// A user who never answers onboarding: no anchor at all, then a first logged start on the default lengths.
func TestRoutine_SkipOnboarding(t *testing.T) {
	e := testenv.New(t)
	a := e.SignUp()
	cv := cycleView(t, getToday(e, a.Token, "", ""))
	expect(t, "no data", cv, map[string]any{
		"date": "2026-09-23", "cycle_day": nil, "main_phase": "unknown", "subphase": "unknown",
		"data_quality": "insufficient", "confidence": "unknown", "resolution_source": "unknown",
		"requires_user_input": true, "anchors.predicted_next_period_start": nil,
	})
	assert.Contains(t, strs(cv["warnings"]), "insufficient_anchor_data")

	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2026-09-20"}, a.Token)
	cv = cycleView(t, getToday(e, a.Token, "", ""))
	// §2.9 fallbacks: no history, nothing declared → 28 / 5.
	expect(t, "first start, defaults", cv, map[string]any{
		"cycle_day": 4, "main_phase": "menstrual", "anchors.current_period_start_source": "user_logged",
		"anchors.current_period_end": "2026-09-24", "anchors.predicted_next_period_start": "2026-10-18",
		"anchors.estimated_ovulation_date": "2026-10-04",
		"metrics.effective_cycle_length":   28, "metrics.effective_period_length": 5,
	})
	assert.Contains(t, strs(cv["warnings"]), "default_values_used")
}

// Irregular history: last three cycles 35, 26, 31 → effective 31 (median, §2.9), variability 9 (§2.27). Regularity
// uses the FIGO age bands: a 9-day spread is irregular at 30 (≤ 7) but regular at 22 (≤ 9, the boundary).
//
// D-207 (deviations.md, user OK 2026-11-08): a user without a cycle_preferences row is "manual" — lengths she declared
// in onboarding win over the history median until she switches «خودکار» on. The age-30 arm declares 28 and checks
// that first; the age-22 arm declares no lengths, so the median applies straight away.
func TestRoutine_IrregularCycle(t *testing.T) {
	for _, tc := range []struct {
		name, birthday, regularity string
		declared                   bool
	}{
		{"age 30 declared 28", "1996-03-01", "irregular", true},
		{"age 22 (boundary 9) nothing declared", "2004-03-01", "regular", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testenv.New(t)
			e.SetNow("2026-09-10T12:00:00+03:30")
			a := e.SignUp()
			var cycle map[string]any
			if tc.declared {
				cycle = map[string]any{"period_duration": 5, "cycle_duration": 28}
			}
			onboard(t, e, a.Token, cycle, tc.birthday)
			for _, p := range [][2]string{{"2026-06-01", "2026-06-05"}, {"2026-07-06", "2026-07-10"}, {"2026-08-01", "2026-08-05"}, {"2026-09-01", "2026-09-05"}} {
				logPeriod(t, e, a.Token, p[0], p[1])
			}
			cv := cycleView(t, getToday(e, a.Token, "", ""))
			if tc.declared {
				// Manual (D-207): the declared 28 → next 09-29, ovulation 09-15; the history is still measured.
				expect(t, tc.name+" manual", cv, map[string]any{
					"metrics.effective_cycle_length": 28, "anchors.predicted_next_period_start": "2026-09-29",
					"anchors.estimated_ovulation_date": "2026-09-15", "metrics.cycle_variability": 9,
				})
				e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/profile/cycle-settings", map[string]any{"lengths_auto": true}, a.Token)
				cv = cycleView(t, getToday(e, a.Token, "", ""))
			}
			// next = 09-01 + 31 = 10-02; ovulation 09-18; display fertile 09-13..09-18; follicular 09-06..09-12
			// (7 days: early 2, mid 2, late 3, §2.23) → 09-10 (day 10) is late_follicular_transition.
			expect(t, tc.name, cv, map[string]any{
				"cycle_day": 10, "main_phase": "follicular", "subphase": "late_follicular_transition",
				"fertility_level":                     "medium",
				"anchors.predicted_next_period_start": "2026-10-02", "anchors.estimated_ovulation_date": "2026-09-18",
				"metrics.effective_cycle_length": 31, "metrics.effective_period_length": 5,
				"metrics.cycle_variability": 9, "metrics.cycle_regularity": tc.regularity,
				"confidence": "low",
			})
			assert.Contains(t, strs(cv["warnings"]), "high_cycle_variability")
			assert.Contains(t, strs(cv["confidence_reasons"]), "high_cycle_variability")
		})
	}
}

// A teen (< 18) is judged on the 21–45 range (§2.27, ACOG CO 651): a declared 40-day cycle is a normal frequency for
// her and infrequent (> 38, FIGO) for an adult with the same data.
func TestRoutine_Teen(t *testing.T) {
	for _, tc := range []struct {
		name, birthday, frequency string
	}{
		{"teen 15", "2011-03-01", "normal"},
		{"adult 30", "1996-03-01", "infrequent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testenv.New(t)
			e.SetNow("2026-09-10T12:00:00+03:30")
			a := e.SignUp()
			onboard(t, e, a.Token, map[string]any{"last_period_start": "2026-09-01", "period_duration": 5, "cycle_duration": 40}, tc.birthday)
			cv := cycleView(t, getToday(e, a.Token, "", ""))
			expect(t, tc.name, cv, map[string]any{
				"cycle_day": 10, "metrics.effective_cycle_length": 40,
				"anchors.predicted_next_period_start": "2026-10-11", "anchors.estimated_ovulation_date": "2026-09-27",
				"clinical_flags.frequency": tc.frequency,
			})
		})
	}
}

// The day turns at Tehran midnight, not UTC: 00:01 Tehran (20:31 UTC the day before) is already the next cycle day,
// and the ETag (which names today) no longer revalidates across midnight.
func TestRoutine_TehranMidnight(t *testing.T) {
	e := testenv.New(t)
	e.SetNow("2026-09-10T12:00:00+03:30")
	a := e.SignUp()
	onboard(t, e, a.Token, map[string]any{"period_duration": 5, "cycle_duration": 28}, "")
	logPeriod(t, e, a.Token, "2026-09-01", "2026-09-05")

	before := getToday(e, a.Token, "2026-09-14T23:59:00+03:30", "")
	expect(t, "23:59", cycleView(t, before), map[string]any{
		"date": "2026-09-14", "cycle_day": 14, "main_phase": "fertile", "subphase": "high_fertility", "days_to_ovulation": 1,
	})
	assert.Equal(t, fiber.StatusNotModified, getToday(e, a.Token, "2026-09-14T23:59:59+03:30", before.ETag()).Status)

	for _, now := range []string{"2026-09-15T00:01:00+03:30", "2026-09-14T20:31:00Z"} {
		after := getToday(e, a.Token, now, before.ETag())
		expect(t, "00:01 as "+now, cycleView(t, after), map[string]any{
			"date": "2026-09-15", "cycle_day": 15, "main_phase": "fertile", "subphase": "ovulation_likely", "days_to_ovulation": 0,
		})
		assert.NotEqual(t, before.ETag(), after.ETag(), now)
	}
}

// 30 Esfand 1403 (leap year) = 2025-03-20, the day before Nowruz: a period started that day counts on through the
// Persian new year (1 Farvardin 1404 = 2025-03-21 is day 2) in Gregorian date arithmetic (§2.3).
func TestRoutine_LeapEsfand30(t *testing.T) {
	e := testenv.New(t)
	const esfand30 = "2025-03-20T21:00:00+03:30"
	e.SetNow(esfand30)
	a := e.SignUp()
	onboard(t, e, a.Token, map[string]any{"period_duration": 5, "cycle_duration": 28}, "")
	r := e.Do(fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2025-03-21"}, a.Token)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "1 Farvardin is still tomorrow: %s", r.Raw)
	e.MustOK(fiber.StatusOK, fiber.MethodPost, "/api/v1/cycle/period/start", map[string]any{"date": "2025-03-20"}, a.Token)

	expect(t, "30 Esfand", cycleView(t, getToday(e, a.Token, "", "")), map[string]any{
		"date": "2025-03-20", "cycle_day": 1, "main_phase": "menstrual",
		"anchors.predicted_next_period_start": "2025-04-17", "anchors.estimated_ovulation_date": "2025-04-03",
	})
	expect(t, "1 Farvardin", cycleView(t, getToday(e, a.Token, "2025-03-21T08:00:00+03:30", "")), map[string]any{
		"date": "2025-03-21", "cycle_day": 2, "main_phase": "menstrual",
	})
	e.SetNow("2025-03-21T08:00:00+03:30")
	days := monthDays(t, e, a.Token, 2025, 3)
	expect(t, "month 2025-03-20", days["2025-03-20"], map[string]any{"cycle_day": 1, "is_period": true})
	expect(t, "month 2025-03-21", days["2025-03-21"], map[string]any{"cycle_day": 2, "is_period": true})
	expect(t, "month 2025-03-31", days["2025-03-31"], map[string]any{"cycle_day": 12})
	days = monthDays(t, e, a.Token, 2025, 4)
	expect(t, "month 2025-04-17", days["2025-04-17"], map[string]any{"cycle_day": 1, "is_period": true, "is_predicted": true})
}
