package inboxalerts_test

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// The parity table (T-M10-08): over the fixtures of the retired engines, the alerts the v1 save-time rules
// (internal/pregnancy/alerts, recorded here as they answered before their deletion) and the v2 detectors
// (pregnancyalerts.Detect with the seeded 00005 configs, run live) raised, next to the preg.* rules that fire now
// (inboxalerts.Fire, the code-held definitions = the seeded rows). The invariant is SPEC §4.2's: no escalation is
// lost — every v1 / v2 alert has a new rule at the same or a higher level — except the deltas named in the row
// (listed for the advisor in docs/engine-v6/clinical-signoff.md).

var today = civildate.MustParse("2026-10-21")

// rank orders the levels of the three engines (v1 alert_level, v2 level4, stored rules.Level*).
func rank(level string) int {
	switch level {
	case "info":
		return 0
	case "suggestion":
		return 1
	case "warning", "follow_up", "worth_followup":
		return 2
	case "emergency", "urgent", "follow_soon":
		return 3
	}
	panic("level " + level)
}

// v2Configs are the seeded v2 rows (00005_pregnancy_v2.sql): level and window per detector.
var v2Configs = map[string]pregnancyalerts.Config{
	"vomiting_streak":      {Enabled: true, Level: "follow_up", WindowDays: 7, Params: map[string]any{"min_streak_days": 3.0, "severe_min_count": 2.0}},
	"severe_symptom_count": {Enabled: true, Level: "follow_up", WindowDays: 7, Params: map[string]any{"min_count": 3.0, "symptoms": []any{"nausea", "vomiting", "fatigue", "headache", "back_pain", "breast_pain", "heartburn", "constipation"}}},
	"critical_symptom":     {Enabled: true, Level: "urgent", WindowDays: 1, Params: map[string]any{"symptoms": []any{"bleeding", "fluid_leakage", "severe_sudden_pain", "spotting"}, "spotting_until_week": 12.0}},
	"weight_missing_week":  {Enabled: true, Level: "suggestion", WindowDays: 7, Params: map[string]any{}},
	"week_entered":         {Enabled: true, Level: "info", WindowDays: 7, Params: map[string]any{"from_week": 1.0}},
	"bp_high":              {Enabled: true, Level: "follow_up", WindowDays: 7, Params: map[string]any{"systolic_min": 140.0, "diastolic_min": 90.0}},
	"sugar_high":           {Enabled: true, Level: "follow_up", WindowDays: 7, Params: map[string]any{"fasting_max": 95.0, "post_meal_max": 140.0}},
	"fetal_movement":       {Enabled: true, Level: "urgent", WindowDays: 1, Params: map[string]any{"from_week": 24.0, "statuses": []any{"reduced", "none"}}},
	"contractions_511":     {Enabled: true, Level: "urgent", WindowDays: 1, Params: map[string]any{}},
}

// v2Hits are the live v2 detector hits as "rule:level".
func v2Hits(f pregnancyalerts.Facts) []string {
	out := []string{}
	for rule, c := range v2Configs {
		if rule == "week_entered" {
			continue // fires on every evaluation (once a week by its dedupe), as preg.week_entered does
		}
		seen := false
		for _, h := range pregnancyalerts.Detect(rule, c, f) {
			level := c.Level
			if h.Level != "" {
				level = h.Level
			}
			if rule == "bp_high" && h.Level == "" {
				level = pregnancyalerts.HighBPLevel(f.Today)
			}
			if !seen || level == "urgent" {
				out = slices.DeleteFunc(out, func(s string) bool { return strings.HasPrefix(s, rule+":") })
				out = append(out, rule+":"+level)
				seen = true
			}
		}
	}
	slices.Sort(out)
	return out
}

// newHits are the preg.* rules that fire now as "key:level".
func newHits(f pregnancyalerts.Facts, in facts.PregnancyAlertInput) []string {
	in.Facts = f
	out := []string{}
	for _, m := range inboxalerts.Fire(&in, "en") {
		if m.Key == "preg.week_entered" {
			continue // every evaluation (TestWeekEnteredEveryWeek)
		}
		out = append(out, m.Key+":"+m.Level)
	}
	slices.Sort(out)
	return out
}

// pregnancy is a pregnancy at weeks+days on today (LMP dating), evaluated at 12:00.
func pregnancy(weeks, days int) pregnancyalerts.Facts {
	return pregnancyalerts.Facts{Today: today, Week: weeks + 1, WeekDay: days, TotalDays: weeks*7 + days, Source: "lmp",
		Days: map[civildate.Date]pregnancyalerts.Day{}, Now: today.TehranMidnight().Add(12 * time.Hour), HasWeight: true}
}

func weekly(day civildate.Date, mod func(*store.PregnancyWeeklyLog)) store.PregnancyWeeklyLog {
	w := store.PregnancyWeeklyLog{LogDate: day}
	mod(&w)
	return w
}

func i32(v int32) sql.NullInt32   { return sql.NullInt32{Int32: v, Valid: true} }
func dec(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }

type parityCase struct {
	name string
	f    pregnancyalerts.Facts
	in   facts.PregnancyAlertInput // LastWeight / Start / NextVisit
	// v1 are the levels the v1 save-time rules raised for the same log (recorded from internal/pregnancy/alerts
	// before its rules were deleted; "" = the case has no v1 log).
	v1 []string
	// v2 and now are the live hits, pinned.
	v2, now []string
	// delta names an accepted difference (no escalation lost unless it says so).
	delta string
}

func parityCases() []parityCase {
	pr := func(weeks, days int, mod func(*pregnancyalerts.Facts)) pregnancyalerts.Facts {
		f := pregnancy(weeks, days)
		if mod != nil {
			mod(&f)
		}
		return f
	}
	day := func(f *pregnancyalerts.Facts, back int, d pregnancyalerts.Day) { f.Days[today.AddDays(-back)] = d }
	kick := func(count int, after time.Duration) pregnancyalerts.Kick {
		start := today.TehranMidnight().Add(9 * time.Hour)
		return pregnancyalerts.Kick{ID: 1, Start: start, End: start.Add(after), Count: count}
	}
	start := today.AddDays(-20 * 7)
	return []parityCase{
		{name: "bleeding severe", f: pr(20, 3, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"bleeding": "severe"}) }),
			v1: []string{"emergency"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "bleeding mild", f: pr(20, 3, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"bleeding": "mild"}) }),
			v1: []string{"warning"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "bleeding logged for 3 days ago", f: pr(20, 3, func(f *pregnancyalerts.Facts) { day(f, 3, pregnancyalerts.Day{"bleeding": "mild"}) }),
			v1: []string{"warning"}, v2: []string{}, now: []string{"preg.critical_symptom:follow_soon"},
			delta: "v2 read today only; the rule reads the 7-day window, as the v1 save-time rule raised any log date"},
		{name: "spotting week 12", f: pr(11, 2, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"spotting": "mild"}) }),
			v1: []string{"warning"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "spotting severe week 20", f: pr(19, 2, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"spotting": "severe"}) }),
			v1: []string{"warning"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "spotting mild week 13", f: pr(12, 2, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"spotting": "mild"}) }),
			v1: []string{"info"}, v2: []string{}, now: []string{},
			delta: "v1 only: the «light spotting» info note after week 12 is not carried over (v2 and SPEC §5.4.1 have no such rule)"},
		{name: "fluid leakage", f: pr(30, 0, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"fluid_leakage": "mild"}) }),
			v1: []string{"emergency"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "severe sudden pain", f: pr(30, 1, func(f *pregnancyalerts.Facts) { day(f, 0, pregnancyalerts.Day{"severe_sudden_pain": "moderate"}) }),
			v1: []string{"emergency"}, v2: []string{"critical_symptom:urgent"}, now: []string{"preg.critical_symptom:follow_soon"}},
		{name: "three severe symptoms in one log (v1 list)", f: pr(20, 3, func(f *pregnancyalerts.Facts) {
			day(f, 0, pregnancyalerts.Day{"dizziness": "severe", "cramping": "severe", "pelvic_pressure": "severe"})
		}), v1: []string{"warning"}, v2: []string{}, now: []string{"preg.severe_symptoms:worth_followup"},
			delta: "the 7-day count also reads the v1 fields v2 lacked (dizziness, lower abdominal pain, cramping, pelvic pressure)"},
		{name: "three severe symptoms over the week (v2 list)", f: pr(20, 3, func(f *pregnancyalerts.Facts) {
			day(f, 0, pregnancyalerts.Day{"nausea": "severe"})
			day(f, 2, pregnancyalerts.Day{"headache": "severe"})
			day(f, 4, pregnancyalerts.Day{"back_pain": "severe"})
		}), v2: []string{"severe_symptom_count:follow_up"}, now: []string{"preg.severe_symptoms:worth_followup"}},
		{name: "vomiting 3 days in a row", f: pr(10, 3, func(f *pregnancyalerts.Facts) {
			for i := range 3 {
				day(f, i, pregnancyalerts.Day{"vomiting": "mild"})
			}
		}), v2: []string{"vomiting_streak:follow_up"}, now: []string{"preg.vomiting:worth_followup"}},
		{name: "vomiting 2 severe days", f: pr(10, 3, func(f *pregnancyalerts.Facts) {
			day(f, 0, pregnancyalerts.Day{"vomiting": "severe"})
			day(f, 1, pregnancyalerts.Day{"vomiting": "severe"})
		}), v2: []string{"vomiting_streak:follow_up"}, now: []string{"preg.vomiting:worth_followup"}},
		{name: "vomiting 4 scattered days", f: pr(10, 3, func(f *pregnancyalerts.Facts) {
			for _, i := range []int{0, 2, 4, 6} {
				day(f, i, pregnancyalerts.Day{"vomiting": "mild"})
			}
		}), v2: []string{}, now: []string{"preg.vomiting:worth_followup"}, delta: "SPEC §5.4.1: 4 of the last 7 days also counts (superset of the v2 streak)"},
		{name: "vomiting 2 mild days", f: pr(10, 3, func(f *pregnancyalerts.Facts) {
			day(f, 0, pregnancyalerts.Day{"vomiting": "mild"})
			day(f, 1, pregnancyalerts.Day{"vomiting": "mild"})
		}), v2: []string{}, now: []string{}},
		{name: "BP 165/112 (severe)", f: pr(30, 2, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) { w.SystolicPressure, w.DiastolicPressure = i32(165), i32(112) })}
		}), v1: []string{"emergency"}, v2: []string{"bp_high:urgent"}, now: []string{"preg.bp_severe:follow_soon"}},
		{name: "BP 145/92 (high)", f: pr(30, 2, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) { w.SystolicPressure, w.DiastolicPressure = i32(145), i32(92) })}
		}), v1: []string{"warning"}, v2: []string{"bp_high:" + pregnancyalerts.HighBPLevel(today)}, now: []string{highBPRule(today)}},
		{name: "BP 120/80", f: pr(30, 2, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) { w.SystolicPressure, w.DiastolicPressure = i32(120), i32(80) })}
		}), v1: []string{}, v2: []string{}, now: []string{}},
		{name: "BP 165/112 through /vitals", f: pr(30, 2, func(f *pregnancyalerts.Facts) {
			f.Readings = []pregnancyalerts.Reading{{Date: today, Systolic: 165, Diastolic: 112}}
		}), v2: []string{"bp_high:urgent"}, now: []string{"preg.bp_severe:follow_soon"}},
		{name: "fasting sugar 98", f: pr(26, 2, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) { w.FastingBloodSugar = dec("98") })}
		}), v1: []string{"warning"}, v2: []string{"sugar_high:follow_up"}, now: []string{"preg.gdm:worth_followup"}},
		{name: "post-meal sugar 139", f: pr(26, 2, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) { w.PostMealBloodSugar = dec("139") })}
		}), v1: []string{}, v2: []string{}, now: []string{}},
		{name: "2 h glucose at the target through /vitals", f: pr(26, 2, func(f *pregnancyalerts.Facts) {
			f.Readings = []pregnancyalerts.Reading{{Date: today, MgDl: pregnancyalerts.GDMTarget(pregnancyalerts.GlucoseTwoHour, today), Slot: pregnancyalerts.GlucoseTwoHour}}
		}), v2: []string{"sugar_high:follow_up"}, now: []string{"preg.gdm:worth_followup"}},
		{name: "depression severe", f: pr(20, 3, func(f *pregnancyalerts.Facts) {
			f.Weekly = []store.PregnancyWeeklyLog{weekly(today, func(w *store.PregnancyWeeklyLog) {
				w.HasDepressionFeelings, w.DepressionSeverity = sql.NullBool{Bool: true, Valid: true}, dec("severe")
			})}
		}), v1: []string{"warning"}, v2: []string{}, now: []string{"preg.depression:worth_followup"}},
		{name: "fetal movement none week 30", f: pr(29, 2, func(f *pregnancyalerts.Facts) {
			f.Fetal = []store.PregnancyFetalMovement{{LogDate: today, PregnancyWeek: 30, MovementStatus: "none"}}
		}), v1: []string{"emergency"}, v2: []string{"fetal_movement:urgent"}, now: []string{"preg.fetal_movement:follow_soon"}},
		{name: "fetal movement reduced week 30", f: pr(29, 2, func(f *pregnancyalerts.Facts) {
			f.Fetal = []store.PregnancyFetalMovement{{LogDate: today, PregnancyWeek: 30, MovementStatus: "reduced"}}
		}), v1: []string{"warning"}, v2: []string{"fetal_movement:urgent"}, now: []string{"preg.fetal_movement:follow_soon"}},
		{name: "fetal movement reduced week 20", f: pr(19, 2, func(f *pregnancyalerts.Facts) {
			f.Fetal = []store.PregnancyFetalMovement{{LogDate: today, PregnancyWeek: 20, MovementStatus: "reduced"}}
		}), v1: []string{}, v2: []string{}, now: []string{}},
		{name: "kick count short of the target", f: pr(30, 2, func(f *pregnancyalerts.Facts) {
			f.Kicks = []pregnancyalerts.Kick{kick(4, 2*time.Hour)}
		}), v2: []string{"fetal_movement:urgent"}, now: []string{"preg.fetal_movement_kicks:follow_soon"}},
		{name: "contractions 5-1-1", f: pr(38, 2, func(f *pregnancyalerts.Facts) {
			f.ContractionSession = 7
			s := today.TehranMidnight().Add(8 * time.Hour)
			for i := range 14 {
				c := s.Add(time.Duration(i) * 5 * time.Minute)
				f.Contractions = append(f.Contractions, labor.Contraction{Start: c, End: c.Add(time.Minute)})
			}
		}), v2: []string{"contractions_511:urgent"}, now: []string{"preg.contractions_511:follow_soon"}},
		{name: "weight missing for 7 days (week day 5)", f: pr(20, 5, func(f *pregnancyalerts.Facts) { f.HasWeight = false }),
			in: facts.PregnancyAlertInput{LastWeight: today.AddDays(-7), Start: start},
			v2: []string{"weight_missing_week:suggestion"}, now: []string{"preg.weight_missing:suggestion"}},
		{name: "weight missing, day 2 of the week, last weight 9 days ago", f: pr(20, 2, func(f *pregnancyalerts.Facts) { f.HasWeight = false }),
			in: facts.PregnancyAlertInput{LastWeight: today.AddDays(-9), Start: start},
			v2: []string{}, now: []string{"preg.weight_missing:suggestion"},
			delta: "SPEC §5.4.1: ≥ 7 days since the last weight (v2 waited for day 5 of a week without one)"},
		{name: "next visit in 2 days", f: pr(20, 2, nil),
			in: facts.PregnancyAlertInput{NextVisit: today.AddDays(2).TehranMidnight().Add(10 * time.Hour)},
			v2: []string{}, now: []string{"preg.next_visit:info"}, delta: "new: SPEC §5.4.1 next-visit reminder"},
	}
}

// highBPRule is the ≥ 140/90 rule the registry selects on day (pregnancy.bp.high.urgent, H-02).
func highBPRule(day civildate.Date) string {
	if pregnancyalerts.HighBPLevel(day) == "urgent" {
		return "preg.bp_high_urgent:follow_soon"
	}
	return "preg.bp_high:worth_followup"
}

func TestParity_V1V2AgainstTheMessageRules(t *testing.T) {
	clinicalconfig.Reset()
	for _, c := range parityCases() {
		t.Run(c.name, func(t *testing.T) {
			v2 := v2Hits(c.f)
			now := newHits(c.f, c.in)
			assert.Equal(t, c.v2, v2, "v2 detectors")
			assert.Equal(t, c.now, now, "message rules")
			// No escalation lost: every old alert has a new one at the same or a higher level.
			top := -1
			for _, h := range now {
				top = max(top, rank(h[strings.LastIndex(h, ":")+1:]))
			}
			for _, lv := range c.v1 {
				if top < rank(lv) {
					assert.NotEmpty(t, c.delta, "v1 %s lost", lv)
				}
			}
			for _, h := range v2 {
				if top < rank(h[strings.LastIndex(h, ":")+1:]) {
					assert.NotEmpty(t, c.delta, "v2 %s lost", h)
				}
			}
		})
	}
}

// Every rule of the table is covered by at least one case, and every Def by a firing case (or named as
// not fired yet: cant_keep_fluids has no log field).
func TestParity_CoversEveryRule(t *testing.T) {
	fired := map[string]bool{}
	for _, c := range parityCases() {
		for _, h := range c.now {
			fired[h[:strings.LastIndex(h, ":")]] = true
		}
	}
	for _, d := range inboxalerts.Defs {
		switch d.Key {
		case "preg.cant_keep_fluids", "preg.bp_high_urgent", "preg.bp_high", "preg.week_entered":
			continue // no log field yet / picked by the registry (TestHighBPLevelFollowsTheRegistry) / every evaluation
		}
		assert.True(t, fired[d.Key], "no parity case fires %s", d.Key)
	}
	require.NotEmpty(t, fmt.Sprint(fired))
}

// preg.week_entered holds on every evaluation of a pregnancy (its evidence, the week, keeps it once a week), like v2.
func TestWeekEnteredEveryWeek(t *testing.T) {
	for _, wd := range []int{0, 3, 6} {
		f := pregnancy(19, wd)
		m, ok := inboxalerts.Fires("preg.week_entered", &facts.PregnancyAlertInput{Facts: f}, "en")
		require.True(t, ok)
		assert.Equal(t, "You've entered week 20", m.Title)
		assert.Len(t, pregnancyalerts.Detect("week_entered", v2Configs["week_entered"], f), 1)
	}
}
