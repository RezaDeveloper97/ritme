package challenges

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// legacyAllows is the B-N9-07 Eligibility.Allows, kept verbatim as the reference the converted rules must match.
func legacyAllows(e legacyRule, mode, phase string) bool {
	if slices.Contains(e.ModesExclude, mode) {
		return false
	}
	if len(e.ModesInclude) > 0 && !slices.Contains(e.ModesInclude, mode) {
		return false
	}
	if phase != "" && slices.Contains(e.PhasesExclude, phase) {
		return false
	}
	if len(e.PhasesInclude) > 0 && !slices.Contains(e.PhasesInclude, phase) {
		return false
	}
	return true
}

// userFacts are the facts Context builds from a life mode and a phase ("" = no phase: main_phase absent).
func userFacts(mode, phase string) rules.Facts {
	f := rules.Facts{FactLifeMode: mode, FactMode: mode}
	if phase != "" {
		f[FactMainPhase] = phase
	}
	return f
}

func raw(s string) db.NullRawJSON { return db.NullRawJSON{V: json.RawMessage(s), Valid: true} }

// The legacy include / exclude rows (the old fixtures) behave exactly as before after the on-the-fly conversion.
func TestEligibility_LegacyRowsUnchanged(t *testing.T) {
	cases := []struct {
		name        string
		e           legacyRule
		mode, phase string
		want        bool
	}{
		{"no rules", legacyRule{}, "cycle", "", true},
		{"mode include hit", legacyRule{ModesInclude: []string{"cycle", "ttc"}}, "ttc", "", true},
		{"mode include miss", legacyRule{ModesInclude: []string{"cycle", "ttc"}}, "pregnancy", "", false},
		{"mode exclude wins over include", legacyRule{ModesInclude: []string{"cycle"}, ModesExclude: []string{"cycle"}}, "cycle", "", false},
		{"phase include hit", legacyRule{PhasesInclude: []string{"luteal", "menstrual"}}, "cycle", "luteal", true},
		{"phase include miss", legacyRule{PhasesInclude: []string{"luteal"}}, "cycle", "follicular", false},
		{"phase include without a phase", legacyRule{PhasesInclude: []string{"luteal"}}, "pregnancy", "", false},
		{"phase exclude", legacyRule{PhasesExclude: []string{"fertile"}}, "cycle", "fertile", false},
		{"phase exclude other", legacyRule{PhasesExclude: []string{"fertile"}}, "cycle", "luteal", true},
		{"phase exclude without a phase", legacyRule{PhasesExclude: []string{"fertile"}}, "cycle", "", true},
		{"artboard rule", legacyRule{ModesInclude: []string{"cycle", "ttc"}, PhasesInclude: []string{"luteal", "menstrual"}, ModesExclude: []string{"pregnancy", "postpartum"}}, "cycle", "luteal", true},
		{"artboard rule, postpartum", legacyRule{ModesInclude: []string{"cycle", "ttc"}, PhasesInclude: []string{"luteal", "menstrual"}, ModesExclude: []string{"pregnancy", "postpartum"}}, "postpartum", "luteal", false},
		{"unknown phase is a value", legacyRule{PhasesInclude: []string{"unknown"}}, "cycle", "unknown", true},
	}
	for _, tc := range cases {
		b, err := json.Marshal(tc.e)
		require.NoError(t, err)
		r := ParseRule(raw(string(b)))
		assert.Equal(t, tc.want, legacyAllows(tc.e, tc.mode, tc.phase), "reference: "+tc.name)
		assert.Equal(t, tc.want, r.Allows(userFacts(tc.mode, tc.phase)), tc.name)
		assert.Equal(t, tc.want, Decide(r, Rule{}, userFacts(tc.mode, tc.phase)) == "", tc.name)
	}

	// Exhaustive: every combination of the lists over every mode and phase agrees with the reference.
	modes := []string{"cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen", ""}
	phases := []string{"menstrual", "follicular", "fertile", "luteal", "period_expected", "unknown", ""}
	lists := [][]string{nil, {"cycle"}, {"pregnancy", "postpartum"}}
	plists := [][]string{nil, {"luteal"}, {"menstrual", "fertile"}}
	n := 0
	for _, mi := range lists {
		for _, me := range lists {
			for _, pi := range plists {
				for _, pe := range plists {
					e := legacyRule{ModesInclude: mi, ModesExclude: me, PhasesInclude: pi, PhasesExclude: pe}
					b, _ := json.Marshal(e)
					r := ParseRule(raw(string(b)))
					for _, m := range modes {
						for _, p := range phases {
							n++
							require.Equal(t, legacyAllows(e, m, p), r.Allows(userFacts(m, p)), "%s mode=%q phase=%q", b, m, p)
						}
					}
				}
			}
		}
	}
	assert.Equal(t, 3*3*3*3*7*7, n)

	// NULL / null / invalid JSON allowed everyone before; they still do.
	for _, v := range []db.NullRawJSON{{}, raw("null"), raw("{}"), raw("not json")} {
		assert.True(t, ParseRule(v).Allows(userFacts("pregnancy", "")), string(v.V))
	}
}

// hardExercise is the fixture contraindication: «hard exercise in pregnancy or less than 6 weeks after birth».
// postpartum_week is the registry's 1-based week (day 0–6 = week 1), so «< 6 weeks» (day < 42) is week ≤ 6.
const hardExercise = `{"all_of":[],"any_of":[[{"fact":"mode","op":"eq","value":"pregnancy"},{"fact":"life_mode","op":"eq","value":"pregnancy"},{"fact":"postpartum_week","op":"lte","value":6}]],"exclude":[]}`

// postpartumWeek is guide.StatusOn's 1-based week for days since birth.
func postpartumWeek(days int) float64 { return float64(days/7 + 1) }

func TestContraindications_WinOverEligibility(t *testing.T) {
	contra := ParseRule(raw(hardExercise))
	require.False(t, contra.Broken)
	elig := ParseRule(raw(`{"all_of":[{"fact":"life_mode","op":"in","value":["cycle","pregnancy","postpartum"]}]}`))

	preg := userFacts("pregnancy", "")
	preg["pregnancy_week"] = 20.0
	assert.True(t, elig.Allows(preg), "the eligibility itself includes pregnancy")
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, preg), "pregnancy + hard exercise → contraindicated")

	pp := func(days int) rules.Facts {
		f := userFacts("postpartum", "")
		f["postpartum_week"] = postpartumWeek(days)
		return f
	}
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, pp(0)), "day of birth")
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, pp(35)), "5 weeks postpartum (day 35)")
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, pp(41)), "5 weeks 6 days (day 41): still < 6 weeks")
	assert.Empty(t, Decide(elig, contra, pp(42)), "6 weeks postpartum (day 42) allowed")
	assert.Empty(t, Decide(elig, contra, pp(70)), "10 weeks postpartum allowed")

	// fail closed: postpartum without a known week (no birth date) is refused; a cycle user (week does not apply) is not
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, userFacts("postpartum", "")), "postpartum, unknown week")
	assert.Empty(t, Decide(elig, contra, userFacts("cycle", "luteal")), "cycle user")
	assert.Equal(t, ReasonNotEligible, Decide(elig, contra, userFacts("menopause", "")), "not contraindicated, not eligible")

	// the engine mode alone is enough (life mode still cycle, e.g. the registry already sees the pregnancy)
	f := userFacts("cycle", "")
	f[FactMode] = "pregnancy"
	assert.Equal(t, ReasonContraindicated, Decide(elig, contra, f))

	// a broken contraindications value fails closed; an empty one never refuses
	assert.Equal(t, ReasonContraindicated, Decide(Rule{}, ParseRule(raw(`{"all_of": 5}`)), userFacts("cycle", "")))
	assert.Equal(t, ReasonContraindicated, Decide(Rule{}, ParseRule(raw(`[1,2`)), userFacts("cycle", "")))
	for _, v := range []db.NullRawJSON{{}, raw("null"), raw("{}"), raw(`{"all_of":[],"any_of":[],"exclude":[]}`)} {
		assert.Empty(t, Decide(Rule{}, ParseRule(v), userFacts("pregnancy", "")), string(v.V))
	}
}

func TestContraindications_ThreeValued(t *testing.T) {
	c := func(fact, op string, v any) rules.Condition { return rules.Condition{Fact: fact, Op: op, Value: v} }
	weekLT := c("postpartum_week", "lte", 6.0)
	pp := userFacts("postpartum", "") // week unknown
	cyc := userFacts("cycle", "")

	// all_of: unknown holds (refuse) unless another condition is false
	assert.True(t, Rule{AllOf: []rules.Condition{weekLT}}.Holds(pp))
	assert.False(t, Rule{AllOf: []rules.Condition{weekLT, c("life_mode", "eq", "cycle")}}.Holds(pp))
	// exclude: an unknown exclusion does not lift the contraindication
	assert.True(t, Rule{AllOf: []rules.Condition{c("life_mode", "eq", "postpartum")}, Exclude: []rules.Condition{c("postpartum_week", "gt", 6.0)}}.Holds(pp))
	pp7 := userFacts("postpartum", "")
	pp7["postpartum_week"] = 7.0
	assert.False(t, Rule{AllOf: []rules.Condition{c("life_mode", "eq", "postpartum")}, Exclude: []rules.Condition{c("postpartum_week", "gt", 6.0)}}.Holds(pp7))
	// a fact absent outside its mode is plain absent (Match semantics)
	assert.False(t, Rule{AllOf: []rules.Condition{weekLT}}.Holds(cyc))
	assert.True(t, Rule{AllOf: []rules.Condition{c("postpartum_week", "ne", 3.0)}}.Holds(cyc), "ne on an absent fact holds (rules.Match)")
	// pregnancy facts are expected in pregnancy
	assert.True(t, Rule{AllOf: []rules.Condition{c("pregnancy_week", "lt", 12.0)}}.Holds(userFacts("pregnancy", "")))
	assert.False(t, Rule{AllOf: []rules.Condition{c("pregnancy_week", "lt", 12.0)}}.Holds(cyc))
}

func TestParseRule_V6Shape(t *testing.T) {
	r := ParseRule(raw(`{"all_of":[{"fact":"life_mode","op":"in","value":["cycle"]}],"any_of":[[{"fact":"main_phase","op":"eq","value":"luteal"},{"fact":"main_phase","op":"eq","value":"menstrual"}]],"exclude":[{"fact":"is_premium","op":"eq","value":true}]}`))
	assert.True(t, r.Allows(userFacts("cycle", "luteal")))
	assert.False(t, r.Allows(userFacts("cycle", "fertile")), "no any_of condition holds")
	f := userFacts("cycle", "menstrual")
	f["is_premium"] = true
	assert.False(t, r.Allows(f), "exclude")
	assert.Equal(t, []string{"is_premium", "life_mode", "main_phase"}, r.Facts())
	p := Program{Eligibility: r, StartRule: StartOnJoin}
	assert.False(t, p.NeedsPhase())
	assert.True(t, needOf(p).Phase, "a rule on main_phase loads the phase")
	assert.True(t, needOf(Program{StartRule: StartOnPhaseStart}).PrevPhase)
}

func TestStartable(t *testing.T) {
	today := civildate.MustParse("2026-10-06")
	luteal := Program{StartRule: StartOnPhaseStart, StartPhase: "luteal"}
	assert.True(t, startable(luteal, UserContext{Phase: "luteal", PrevPhase: "fertile"}, today), "first luteal day")
	assert.False(t, startable(luteal, UserContext{Phase: "luteal", PrevPhase: "luteal"}, today), "second luteal day")
	assert.False(t, startable(luteal, UserContext{Phase: "follicular", PrevPhase: "fertile"}, today))
	assert.False(t, startable(luteal, UserContext{Phase: "luteal"}, today), "yesterday unknown to the caller")
	assert.False(t, startable(Program{StartRule: StartOnPhaseStart}, UserContext{Phase: "luteal", PrevPhase: "fertile"}, today), "no start phase")
	for _, r := range []string{StartOnJoin, StartAnytime, ""} {
		assert.True(t, startable(Program{StartRule: r}, UserContext{}, today), r)
	}
	assert.True(t, startable(Program{StartRule: StartMenstrual}, UserContext{Phase: "menstrual", PrevPhase: "menstrual"}, today), "legacy menstrual: any day of the phase")
	assert.False(t, startable(Program{StartRule: StartMenstrual}, UserContext{Phase: "follicular"}, today))
	fixed := Program{StartRule: StartFixedDate, StartsOn: civildate.NullDate{Date: today.AddDays(1), Valid: true}}
	assert.False(t, startable(fixed, UserContext{}, today))
	assert.True(t, startable(fixed, UserContext{}, today.AddDays(1)))
}

func TestProgress_CompletionRule(t *testing.T) {
	tasks := []store.ChallengeTask{{ID: 1, DayFrom: 1, DayTo: 3}, {ID: 2, DayFrom: 2, DayTo: 3}}
	done := map[[2]uint64]bool{{1, 1}: true, {1, 2}: true, {2, 2}: true}
	p := Program{DurationDays: 3, CompletionMinDays: 2}
	d, req := Progress(p, tasks, done)
	assert.Equal(t, 2, d, "day 1 (task 1) and day 2 (tasks 1+2) are done")
	assert.Equal(t, 2, req)
	_, req = Progress(Program{DurationDays: 3}, tasks, done)
	assert.Equal(t, 3, req, "no minimum = every day")
	_, req = Progress(Program{DurationDays: 3, CompletionMinDays: 9}, tasks, done)
	assert.Equal(t, 3, req, "never more than the days that have tasks")
	d, _ = Progress(p, tasks, map[[2]uint64]bool{{2, 2}: true})
	assert.Equal(t, 0, d, "a day counts only when every task of it is done")
}

func TestDropOff(t *testing.T) {
	today := civildate.MustParse("2026-10-10")
	run := func(started string, status string, last int64) store.ProgramRunsRow {
		return store.ProgramRunsRow{StartedOn: civildate.MustParse(started), Status: status, LastDay: last}
	}
	// 5 runs of a 4-day challenge.
	runs := []store.ProgramRunsRow{
		run("2026-10-01", StatusActive, 0),    // started, never did anything: dropped on day 1
		run("2026-10-01", StatusActive, 2),    // dropped on day 3
		run("2026-10-01", StatusCompleted, 3), // completed (min days 3): retained through the end
		run("2026-10-02", StatusActive, 4),    // every day
		run("2026-10-09", StatusActive, 2),    // started yesterday: has reached days 1-2 only
	}
	got := DropOff(runs, 4, today)
	require.Len(t, got, 4)
	want := []DayRetention{
		{Day: 1, Reached: 5, Retained: 4, Dropped: 1, RetainedPct: 80, DroppedPct: 20},
		{Day: 2, Reached: 5, Retained: 4, Dropped: 0, RetainedPct: 80, DroppedPct: 0},
		{Day: 3, Reached: 4, Retained: 2, Dropped: 1, RetainedPct: 50, DroppedPct: 25},
		{Day: 4, Reached: 4, Retained: 2, Dropped: 0, RetainedPct: 50, DroppedPct: 0},
	}
	assert.Equal(t, want, got)
	assert.Equal(t, []DayRetention{{Day: 1, RetainedPct: -1, DroppedPct: -1}}, DropOff(nil, 1, today), "no runs: rates null")
	assert.Empty(t, DropOff(runs, 0, today))
}

func TestToLegacy_RoundTrip(t *testing.T) {
	mi, me, pi, pe, ok := ToLegacy(FromLegacy([]string{"cycle"}, []string{"pregnancy"}, []string{"luteal"}, nil))
	assert.True(t, ok)
	assert.Equal(t, []string{"cycle"}, mi)
	assert.Equal(t, []string{"pregnancy"}, me)
	assert.Equal(t, []string{"luteal"}, pi)
	assert.Nil(t, pe)
	_, _, _, _, ok = ToLegacy(ParseRule(raw(`{"all_of":[{"fact":"postpartum_week","op":"gte","value":6}]}`)))
	assert.False(t, ok, "another fact needs the v6 shape")
	_, _, _, _, ok = ToLegacy(ParseRule(raw(`{"any_of":[[{"fact":"life_mode","op":"in","value":["cycle"]}]]}`)))
	assert.False(t, ok, "any_of needs the v6 shape")
	_, _, _, _, ok = ToLegacy(Rule{})
	assert.True(t, ok)
}
