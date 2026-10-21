package pregnancyalerts

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// T-M9-05 — the fixed emergency paths.

// pregnantOn is facts of a pregnancy at weeks+days on today, evaluated at now.
func pregnantOn(weeks, days int, now time.Time) Facts {
	f := facts(nil)
	f.TotalDays, f.Week, f.WeekDay, f.Now = weeks*7+days, weeks+1, days, now
	return f
}

var kickStart = today.TehranMidnight().Add(9 * time.Hour)

func kick(id uint64, count int, endAfter, targetAfter time.Duration) Kick {
	k := Kick{ID: id, Start: kickStart, Count: count}
	if endAfter > 0 {
		k.End = kickStart.Add(endAfter)
	}
	if targetAfter > 0 {
		k.TargetAt = kickStart.Add(targetAfter)
	}
	return k
}

func fetalCfg() Config { return cfg(1, map[string]any{"statuses": []any{"reduced", "none"}}) }

func TestKickSessions(t *testing.T) {
	clinicalconfig.Reset()
	now := kickStart.Add(3 * time.Hour)
	run := func(weeks, days int, k Kick, at time.Time) []Hit {
		f := pregnantOn(weeks, days, at)
		f.Kicks = []Kick{k}
		return Detect("fetal_movement", fetalCfg(), f)
	}
	// 28+0, 9 kicks, ended at 120 min → urgent.
	hits := run(28, 0, kick(1, 9, 120*time.Minute, 0), now)
	require.Len(t, hits, 1)
	assert.Equal(t, "urgent", hits[0].Level)
	assert.Equal(t, VariantKicks, hits[0].Variant)
	assert.Equal(t, "kick:1", hits[0].Dedupe)
	assert.Equal(t, [][2]string{{"count", "9"}, {"target", "10"}, {"minutes", "120"}}, hits[0].Vars)
	// 10 kicks, the 10th at 120 min → none; at 121 min → urgent.
	assert.Empty(t, run(28, 0, kick(2, 10, 120*time.Minute, 120*time.Minute), now))
	assert.Len(t, run(28, 0, kick(3, 10, 121*time.Minute, 121*time.Minute), now), 1)
	// 27+6 with 3 kicks → none (from_week 28 completed weeks).
	assert.Empty(t, run(27, 6, kick(4, 3, 120*time.Minute, 0), now))
	// Started 2 h 1 min ago with 4 kicks and never ended → urgent on the next read.
	hits = run(30, 0, kick(5, 4, 0, 0), kickStart.Add(121*time.Minute))
	require.Len(t, hits, 1)
	assert.Equal(t, "urgent", hits[0].Level)
	// Still inside the window → nothing yet.
	assert.Empty(t, run(30, 0, kick(6, 4, 0, 0), kickStart.Add(119*time.Minute)))
	// Running and already at 10 within the window → nothing, even after the window.
	assert.Empty(t, run(30, 0, kick(7, 10, 0, 40*time.Minute), kickStart.Add(5*time.Hour)))
	// A session that ended yesterday is still evaluated (lookback), one from three days ago is not.
	old := kick(8, 2, 30*time.Minute, 0)
	old.Start, old.End = old.Start.AddDate(0, 0, -1), old.End.AddDate(0, 0, -1)
	assert.Len(t, run(30, 0, old, now), 1)
	old.Start, old.End = old.Start.AddDate(0, 0, -2), old.End.AddDate(0, 0, -2)
	assert.Empty(t, run(30, 0, old, now))
}

func TestKickSessions_ThresholdsFromRegistry(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPregnancyKicks, Version: 2,
		Value: clinicalconfig.Value{"target": 10, "window_min": 120, "from_week": 26}, EffectiveFrom: today.AddDays(-30)}})
	f := pregnantOn(26, 0, kickStart.Add(3*time.Hour))
	f.Kicks = []Kick{kick(1, 3, time.Hour, 0)}
	assert.Len(t, Detect("fetal_movement", fetalCfg(), f), 1)
}

func TestManualFetalMovement_FromWeek24(t *testing.T) {
	clinicalconfig.Reset()
	// Manual reduced at 26+0 still follows manual_from_week = 24 (H-04); the stored week is the 1-based week.
	f := pregnantOn(26, 0, kickStart)
	f.Fetal = []store.PregnancyFetalMovement{{LogDate: today, PregnancyWeek: 27, MovementStatus: "reduced"}}
	hits := Detect("fetal_movement", fetalCfg(), f)
	require.Len(t, hits, 1)
	assert.Empty(t, hits[0].Level, "the manual path keeps the row's level")
	f.Fetal[0].PregnancyWeek = 23
	assert.Empty(t, Detect("fetal_movement", fetalCfg(), f))
}

func weekly(sys, dia int32) store.PregnancyWeeklyLog {
	return store.PregnancyWeeklyLog{LogDate: today,
		SystolicPressure: sql.NullInt32{Int32: sys, Valid: true}, DiastolicPressure: sql.NullInt32{Int32: dia, Valid: true}}
}

func TestBPTiers(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	bp := cfg(7, nil)
	level := func(sys, dia int32) string {
		f := facts(nil)
		f.Weekly = []store.PregnancyWeeklyLog{weekly(sys, dia)}
		hits := Detect("bp_high", bp, f)
		if len(hits) == 0 {
			return ""
		}
		return hits[0].Level + "/" + hits[0].Variant
	}
	assert.Empty(t, level(139, 89))
	assert.Equal(t, "urgent/", level(140, 80), "high tier → level from pregnancy.bp.high.urgent = 1")
	assert.Equal(t, "urgent/", level(120, 90))
	assert.Equal(t, "urgent/", level(159, 109))
	assert.Equal(t, "urgent/severe", level(160, 80))
	assert.Equal(t, "urgent/severe", level(120, 110))

	// The advisor sets urgent = 0 (H-02): the high tier is follow_up, the severe tier stays urgent.
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPregnancyBPHigh, Version: 2,
		Value: clinicalconfig.Value{"systolic": 140, "diastolic": 90, "urgent": 0}, EffectiveFrom: today.AddDays(-1)}})
	assert.Equal(t, "follow_up/", level(140, 80))
	assert.Equal(t, "follow_up/", level(159, 109))
	assert.Equal(t, "urgent/severe", level(165, 112))

	// A forced (missing / disabled) row only keeps the severe tier.
	forced := bp
	forced.Forced = true
	f := facts(nil)
	f.Weekly = []store.PregnancyWeeklyLog{weekly(150, 95), weekly(170, 100)}
	f.Weekly[1].LogDate = today.AddDays(-1)
	hits := Detect("bp_high", forced, f)
	require.Len(t, hits, 1)
	assert.Equal(t, VariantSevere, hits[0].Variant)
}

func TestReadings_BPAndGDM(t *testing.T) {
	clinicalconfig.Reset()
	f := facts(nil)
	f.Readings = []Reading{{Date: today, Systolic: 165, Diastolic: 112}}
	hits := Detect("bp_high", cfg(7, nil), f)
	require.Len(t, hits, 1)
	assert.Equal(t, "urgent", hits[0].Level)
	assert.Equal(t, "2026-09-23:165/112", hits[0].Dedupe, "same dedupe as the weekly log of that day and value")

	sugar := func(slot string, v float64) int {
		g := facts(nil)
		g.Readings = []Reading{{Date: today, MgDl: v, Slot: slot}}
		return len(Detect("sugar_high", cfg(7, nil), g))
	}
	for _, tc := range []struct {
		slot      string
		below, at float64
	}{{GlucoseFasting, 94, 95}, {GlucoseOneHour, 139, 140}, {GlucoseTwoHour, 119, 120}} {
		assert.Equal(t, 0, sugar(tc.slot, tc.below), tc.slot)
		assert.Equal(t, 1, sugar(tc.slot, tc.at), tc.slot)
	}
	assert.Equal(t, 0, sugar("random", 300), "no GDM target outside the three slots")
}

func TestCriticalSymptom_MandatorySymptoms(t *testing.T) {
	c := cfg(1, map[string]any{"symptoms": []any{"spotting"}})
	f := facts(map[int]Day{0: {"fluid_leakage": "mild"}})
	hits := Detect("critical_symptom", c, f)
	require.Len(t, hits, 1, "fluid leakage stays on the path even when the admin list drops it")
}

// Copy row disabled or deleted → the alert still fires with the embedded copy, in fa and en.
func TestEmergencyRulesWithoutCopy(t *testing.T) {
	clinicalconfig.Reset()
	for _, loc := range []string{"fa", "en"} {
		l := v2.Lang{Locale: loc, Default: "fa"}
		f := pregnantOn(30, 0, kickStart.Add(3*time.Hour))
		f.Kicks = []Kick{kick(1, 4, time.Hour, 0)}
		level, title, contact := FiresWithoutCopy("fetal_movement", f, l)
		assert.Equal(t, "urgent", level, loc)
		assert.NotEmpty(t, title, loc)
		assert.Contains(t, contact, map[string]string{"fa": "۱۱۵", "en": "115"}[loc], loc)

		f = facts(map[int]Day{0: {"fluid_leakage": "mild"}})
		level, title, _ = FiresWithoutCopy("critical_symptom", f, l)
		assert.Equal(t, "urgent", level, loc)
		assert.NotEmpty(t, title, loc)

		f = facts(nil)
		f.Weekly = []store.PregnancyWeeklyLog{weekly(162, 100)}
		level, title, _ = FiresWithoutCopy("bp_high", f, l)
		assert.Equal(t, "urgent", level, loc)
		assert.Equal(t, embeddedRule("bp_high", loc)["variants"].(map[string]any)["severe"].(map[string]any)["title"], title)
		f.Weekly = []store.PregnancyWeeklyLog{weekly(150, 95)}
		level, _, _ = FiresWithoutCopy("bp_high", f, l)
		assert.Empty(t, level, "a disabled bp_high row keeps only the severe tier")
	}

	// A disabled row is replaced; an enabled one is kept.
	rs := rows{"fetal_movement": {"fa": {"enabled": false, "level": "info", "title": "x"}},
		"critical_symptom": {"fa": {"enabled": true, "level": "urgent", "title": "admin"}}}
	l := v2.Lang{Locale: "fa", Default: "fa"}
	rs.withEmergency(l)
	c, ok := rs.config("fetal_movement", l)
	require.True(t, ok)
	assert.True(t, c.Enabled && c.Forced)
	assert.Equal(t, "urgent", c.Level)
	c, _ = rs.config("critical_symptom", l)
	assert.False(t, c.Forced)
	assert.Equal(t, "admin", rs.texts("critical_symptom", l)["title"])
	_, ok = rs.config("contractions_511", l)
	assert.True(t, ok, "a deleted row is replaced too")
}

// A kick hit on a live admin row renders the embedded kicks variant (the row has none) and the row's contact.
func TestRenderVariantOverRow(t *testing.T) {
	row := map[string]any{"title": "row title", "what_we_saw": "row {status}", "contact": "row contact"}
	m := meta{Rule: "fetal_movement", Level4: "urgent", Variant: VariantKicks,
		Vars: map[string]string{"count": "4", "target": "10", "minutes": "120"}}
	en := render(row, m, "en")
	assert.Equal(t, "Fewer than 10 movements in 120 minutes", en.title)
	assert.Equal(t, "row contact", en.contact)
	fa := render(row, m, "fa")
	assert.Contains(t, fa.title, "۱۲۰")
	assert.Contains(t, fa.advice, "همین حالا با پزشک یا ماما تماس بگیر")
	row["variants"] = map[string]any{"kicks": map[string]any{"title": "admin kicks"}}
	assert.Equal(t, "admin kicks", render(row, m, "en").title)
	// An empty emergency row still renders.
	assert.NotEmpty(t, render(nil, meta{Rule: "critical_symptom", Level4: "urgent", Vars: map[string]string{"symptom": "bleeding"}}, "fa").contact)
}
