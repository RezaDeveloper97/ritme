package conditionnudges

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func d(s string) civildate.Date {
	v, err := civildate.Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

func pain(item, level, score string) taxonomy.Entry {
	e := taxonomy.Entry{Category: "pain", Param: "location", Item: item, Code: ns(level)}
	if score != "" {
		e.Num = ns(score)
	}
	return e
}

func flow(code string) taxonomy.Entry {
	return taxonomy.Entry{Category: "bleeding", Param: "flow", Code: ns(code)}
}

func day(date string, entries ...taxonomy.Entry) healthlog.DayEntries {
	return healthlog.DayEntries{Date: d(date), Entries: entries}
}

func facts(mode string, days ...healthlog.DayEntries) Facts {
	return Facts{Mode: mode, Enrolled: map[string]bool{}, From: d("2026-09-01"), To: d("2026-09-20"), Days: days}
}

func rulesOf(hits []Hit) []string {
	out := []string{}
	for _, h := range hits {
		out = append(out, h.Rule)
	}
	return out
}

func TestHeavyPain_TwoDaysAtSevenOrMore(t *testing.T) {
	hits := Detect(facts("cycle",
		day("2026-09-02", pain("abdomen", "severe", "7")),
		day("2026-09-03", pain("pelvis", "moderate", "5"), pain("back", "severe", "8.0")),
	))
	require.Len(t, hits, 1)
	assert.Equal(t, RuleHeavyPain, hits[0].Rule)
	assert.Equal(t, ProgramEndo, hits[0].Program)
	assert.Equal(t, "/programs/pain", hits[0].Link)
	assert.Equal(t, []civildate.Date{d("2026-09-02"), d("2026-09-03")}, hits[0].Dates)
}

func TestHeavyPain_Thresholds(t *testing.T) {
	cases := map[string][]healthlog.DayEntries{
		"one day only": {day("2026-09-02", pain("abdomen", "severe", "9"))},
		"scores of 6": {
			day("2026-09-02", pain("abdomen", "moderate", "6")),
			day("2026-09-03", pain("abdomen", "moderate", "6")),
		},
		// A score below 7 wins over a severe level.
		"score beats level": {
			day("2026-09-02", pain("abdomen", "severe", "6")),
			day("2026-09-03", pain("abdomen", "severe", "6")),
		},
		"no items never count": {
			day("2026-09-02", pain("abdomen", taxonomy.No, "9")),
			day("2026-09-03", pain("abdomen", taxonomy.No, "9")),
		},
		"outside the cycle": {
			day("2026-08-30", pain("abdomen", "severe", "9")),
			day("2026-09-02", pain("abdomen", "severe", "9")),
		},
	}
	for name, days := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Empty(t, Detect(facts("cycle", days...)))
		})
	}
}

func TestHeavyPain_SevereLevelWithoutScore(t *testing.T) {
	hits := Detect(facts("ttc",
		day("2026-09-02", pain("abdomen", "severe", "")),
		day("2026-09-05", pain("head", "severe", "")),
	))
	assert.Equal(t, []string{RuleHeavyPain}, rulesOf(hits))
}

func TestHeavyBleeding_TwoHeavyDays(t *testing.T) {
	hits := Detect(facts("teen",
		day("2026-09-01", flow("very_heavy")),
		day("2026-09-02", flow("medium")),
		day("2026-09-03", flow("heavy")),
	))
	require.Len(t, hits, 1)
	assert.Equal(t, RuleHeavyBleeding, hits[0].Rule)
	assert.Equal(t, ProgramHeavyBleeding, hits[0].Program)
	assert.Equal(t, "/programs/bleeding", hits[0].Link)
	assert.Len(t, hits[0].Dates, 2)

	assert.Empty(t, Detect(facts("cycle", day("2026-09-01", flow("heavy")), day("2026-09-02", flow("medium")))))
}

func TestBothRules_InOrder(t *testing.T) {
	hits := Detect(facts("cycle",
		day("2026-09-01", flow("heavy"), pain("abdomen", "severe", "8")),
		day("2026-09-02", flow("heavy"), pain("abdomen", "severe", "10")),
	))
	assert.Equal(t, []string{RuleHeavyPain, RuleHeavyBleeding}, rulesOf(hits))
}

func TestEnrolledUsersAreNotNudged(t *testing.T) {
	days := []healthlog.DayEntries{
		day("2026-09-01", flow("heavy"), pain("abdomen", "severe", "8")),
		day("2026-09-02", flow("heavy"), pain("abdomen", "severe", "8")),
	}
	f := facts("cycle", days...)
	f.Enrolled[ProgramEndo] = true
	assert.Equal(t, []string{RuleHeavyBleeding}, rulesOf(Detect(f)))

	f.Enrolled[ProgramHeavyBleeding] = true
	assert.Empty(t, Detect(f))

	// Another program's enrolment does not silence a rule.
	f = facts("cycle", days...)
	f.Enrolled["pmdd"] = true
	assert.Len(t, Detect(f), 2)
}

func TestOtherModesAreNotNudged(t *testing.T) {
	for _, mode := range []string{"pregnancy", "postpartum", "menopause", ""} {
		f := facts(mode,
			day("2026-09-01", flow("heavy"), pain("abdomen", "severe", "8")),
			day("2026-09-02", flow("heavy"), pain("abdomen", "severe", "8")),
		)
		assert.Empty(t, Detect(f), mode)
	}
}

func TestCycleWindow(t *testing.T) {
	today := d("2026-09-20")
	start := d("2026-09-05")
	from, to := CycleWindow(today, &start, 28)
	assert.Equal(t, start, from)
	assert.Equal(t, today, to)

	old := d("2026-07-01") // no period logged for > MaxCycleDays
	from, _ = CycleWindow(today, &old, 30)
	assert.Equal(t, d("2026-08-22"), from)

	from, _ = CycleWindow(today, nil, 0)
	assert.Equal(t, today.AddDays(-(FallbackCycleDays - 1)), from)

	future := d("2026-09-25")
	from, _ = CycleWindow(today, &future, 99)
	assert.Equal(t, today.AddDays(-(FallbackCycleDays - 1)), from)
}

func TestFallbackCopy_EveryRuleAndText(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		for _, r := range Rules {
			for _, k := range TextKeys {
				assert.NotEmpty(t, fallbackText(r, k, locale), "%s %s %s", locale, r, k)
			}
		}
	}
	assert.Contains(t, fallbackText(RuleHeavyPain, "body", "en"), "{days}")
}

// T-M9-37 (D-136): the v6 window is the engine's current cycle; 15 days late (unresolved cycle) there is none, so
// nothing is nudged even with qualifying days — v1's 45-day rule would still have given a window.
func TestEngineCycleWindow(t *testing.T) {
	today := d("2026-09-23")
	prep := func(starts ...string) *engine.State {
		var in engine.Input
		for _, s := range starts {
			x := engine.DayOf(d(s))
			in.Periods = append(in.Periods, engine.Period{Start: x, End: x + 4})
		}
		return engine.Prepare(in, engineparams.For(today))
	}
	from, to, ok := EngineCycleWindow(prep("2026-08-16", "2026-09-14"), today)
	require.True(t, ok)
	assert.Equal(t, d("2026-09-14"), from)
	assert.Equal(t, today, to)

	_, _, ok = EngineCycleWindow(prep("2026-07-14", "2026-08-11"), today) // next 09-08: 15 days late
	assert.False(t, ok)
	_, _, ok = EngineCycleWindow(prep("2026-07-17", "2026-08-14"), today) // next 09-11: 12 days late, still current
	assert.True(t, ok)
	_, _, ok = EngineCycleWindow(prep(), today)
	assert.False(t, ok)

	f, _ := CycleWindow(today, ptr(d("2026-08-11")), 28)
	assert.Equal(t, d("2026-08-11"), f, "v1: 43 days < MaxCycleDays is still the current cycle")

	heavy := []healthlog.DayEntries{
		{Date: d("2026-09-20"), Entries: []taxonomy.Entry{{Category: "bleeding", Param: "flow", Code: sql.NullString{String: "heavy", Valid: true}}}},
		{Date: d("2026-09-21"), Entries: []taxonomy.Entry{{Category: "bleeding", Param: "flow", Code: sql.NullString{String: "heavy", Valid: true}}}},
	}
	assert.Empty(t, Detect(Facts{Mode: "cycle", Enrolled: map[string]bool{}, Days: heavy}), "no window → no nudge")
	assert.Len(t, Detect(Facts{Mode: "cycle", Enrolled: map[string]bool{}, From: d("2026-09-14"), To: today, Days: heavy}), 1)
}

func ptr[T any](v T) *T { return &v }
