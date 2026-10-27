package manager

// T-M9-38 (D-130, DECISIONS.md I-11): the daily message and the v2 facts on the v6 engine, over engine personas built
// in memory (engine.Prepare) — no database.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

var v6Today = civildate.MustParse("2026-09-23")

func eday(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

// closedPeriods are n closed periods of length days every gap days, the last starting on last.
func closedPeriods(last string, n, gap, length int) []engine.Period {
	out := make([]engine.Period, n)
	l := eday(last)
	for i := range n {
		s := l - engine.Day((n-1-i)*gap)
		out[i] = engine.Period{Start: s, End: s + engine.Day(length-1)}
	}
	return out
}

// v6Personas are the eight personas of the agreement test (the home v6 set plus late 15).
func v6Personas() map[string]engine.Input {
	regular := closedPeriods("2026-09-10", 6, 28, 5)
	open := append(closedPeriods("2026-08-24", 5, 28, 5), engine.Period{Start: eday("2026-09-21"), End: engine.NoDay})
	return map[string]engine.Input{
		"regular":    {Periods: regular},
		"late_8":     {Periods: closedPeriods("2026-08-18", 6, 28, 5)},
		"late_15":    {Periods: closedPeriods("2026-08-11", 6, 28, 5)},
		"pill":       {Periods: regular, Hormonal: true},
		"onboarding": {Declared: engine.Declared{LastPeriodStart: engine.Some(eday("2026-09-10")), CycleLength: engine.Some(28), PeriodLength: engine.Some(5)}},
		"short_18":   {Periods: closedPeriods("2026-09-15", 6, 18, 4)},
		"open":       {Periods: open},
		"teen":       {Periods: regular, Mode: engine.ModeTeen},
	}
}

// engineSrc is a fakeSource with a v6 engine state (and optionally a stored life mode).
type engineSrc struct {
	fakeSource
	st    *engine.State
	today civildate.Date
	life  string
}

func (s *engineSrc) EngineDay(_ context.Context, date civildate.Date) (EngineDay, bool, error) {
	return EngineDay{
		Status: s.st.Resolve(engine.DayOf(date), engine.DayOf(s.today)), Mode: s.st.Mode(),
		HasAnchor: len(s.st.Periods()) > 0 || s.st.Declared().LastPeriodStart.OK(),
	}, true, nil
}

func (s *engineSrc) LifeMode(context.Context) (string, error) { return s.life, nil }

func newEngineSrc(in engine.Input, goal, life string) *engineSrc {
	return &engineSrc{
		fakeSource: fakeSource{profile: &Profile{UserGoal: goal, SubscriptionType: "free", HasLastPeriodStart: true}},
		st:         engine.Prepare(in, engineparams.For(v6Today)), today: v6Today, life: life,
	}
}

func generate(t *testing.T, src Source, date civildate.Date) map[string]any {
	t.Helper()
	res, err := New(src, defaultsContent{}, "fa", v6Today).Generate(context.Background(), date, "")
	require.NoError(t, err)
	return toJSON(t, res.JSON())
}

// The manager never calls Cycle when the source has an engine (a calc that would show is never read).
func TestV6_EngineReplacesLegacyCycle(t *testing.T) {
	src := newEngineSrc(v6Personas()["regular"], "non_ttc", "")
	src.calc = cycleCalc() // day 20 luteal — must not leak
	ctx := generate(t, src, v6Today)["context_info"].(map[string]any)
	s := src.st.Resolve(engine.DayOf(v6Today), engine.DayOf(v6Today))
	assert.EqualValues(t, s.CycleDay.Or(0), ctx["cycle_day"])
	assert.NotEqualValues(t, 20, ctx["cycle_day"])
}

// Late 8 and 15 days: the cycle day keeps counting past the cycle length (never "day N of a new cycle"), the phase is
// never menstruation and there is no fertile window.
func TestV6_LateIsNeverANewCycle(t *testing.T) {
	for _, name := range []string{"late_8", "late_15"} {
		src := newEngineSrc(v6Personas()[name], "non_ttc", "")
		s := src.st.Resolve(engine.DayOf(v6Today), engine.DayOf(v6Today))
		late, ok := s.DaysLate.Get()
		require.True(t, ok, name)
		require.GreaterOrEqual(t, late, 8, name)

		m := generate(t, src, v6Today)
		ci := m["context_info"].(map[string]any)
		cd, _ := ci["cycle_day"].(float64)
		assert.Greater(t, int(cd), s.Metrics.EffectiveCycleLength, name)
		assert.NotEqual(t, "menstruation", ci["phase"], name)
		assert.Equal(t, false, ci["is_fertile_window"], name)
		pm := m["primary_message"].(map[string]any)
		assert.Equal(t, ci["cycle_day"], pm["cycle_day"], name)
		assert.NotEmpty(t, pm["short_message"], name)

		f := EngineFacts(&EngineDay{Status: s, Mode: src.st.Mode()}, false)
		assert.EqualValues(t, late, f["days_late"], name)
		assert.EqualValues(t, 0, f["days_to_period"], name)
		assert.EqualValues(t, cd, f["cycle_day"], name)
		assert.NotEqual(t, "menstruation", f["phase"], name)
	}
}

// An unknown day (I-11) gets the neutral "default" copy, no phase and no phase supplements — never the
// «enter your last period» message of a user without data.
func TestV6_UnknownIsNeutral(t *testing.T) {
	found := false
	for name, in := range v6Personas() {
		src := newEngineSrc(in, "non_ttc", "")
		for i := range 90 {
			date := v6Today.AddDays(i)
			s := src.st.Resolve(engine.DayOf(date), engine.DayOf(v6Today))
			if s.Subphase != engine.SubphaseUnknown {
				continue
			}
			found = true
			m := generate(t, src, date)
			pm := m["primary_message"].(map[string]any)
			assert.Nil(t, pm["phase"], name)
			assert.Equal(t, "به سلامت خود توجه کنید", pm["short_message"], name)
			assert.Equal(t, "هر روز فرصتی برای مراقبت از خودته.", pm["long_message"], name+": the neutral default, not «enter your last period»")
			assert.Equal(t, []any{}, m["supplements"].(map[string]any)["nutrition"], name)
			assert.Nil(t, m["context_info"].(map[string]any)["phase"], name)
			break
		}
	}
	assert.True(t, found, "some persona reaches an unknown day")
}

// A teen with goal TTC and a pill user get no fertile facts and no ovulation content on any day; O+1 is never
// fertile for anyone.
func TestV6_FertilityStripped(t *testing.T) {
	cases := []struct {
		name, persona, life string
		strip               bool
	}{
		{"teen ttc", "teen", "teen", true},
		{"teen ttc (engine mode only)", "teen", "", true},
		{"pill", "pill", "", true},
		{"regular ttc", "regular", "", false},
	}
	for _, tc := range cases {
		src := newEngineSrc(v6Personas()[tc.persona], "ttc", tc.life)
		sawFertile := false
		for i := -30; i < 30; i++ {
			date := v6Today.AddDays(i)
			s := src.st.Resolve(engine.DayOf(date), engine.DayOf(v6Today))
			d := EngineDay{Status: s, Mode: src.st.Mode()}
			f := EngineFacts(&d, tc.life == "teen")
			m := generate(t, src, date)
			ci := m["context_info"].(map[string]any)
			msg := tc.name + " " + date.String()
			if tc.strip {
				assert.NotContains(t, f, "is_fertile_window", msg)
				assert.Equal(t, true, f["fertility_suppressed"], msg)
				assert.NotEqual(t, "ovulation", f["phase"], msg)
				assert.NotEqual(t, "fertile", f["main_phase"], msg)
				assert.NotContains(t, []any{"fertile_rising", "high_fertility", "ovulation_likely", "post_ovulation"}, f["subphase"], msg)
				assert.Equal(t, false, ci["is_fertile_window"], msg)
				assert.NotEqual(t, "ovulation", ci["phase"], msg)
				assert.Nil(t, ci["estimated_ovulation_day"], msg)
				assert.NotContains(t, m["primary_message"], "ttc_tips", msg)
				if tc.life == "teen" {
					assert.Equal(t, "non_ttc", m["user_goal"], msg)
				}
			} else {
				assert.Equal(t, false, f["fertility_suppressed"], msg)
				if f["is_fertile_window"] == true {
					sawFertile = true
				}
			}
			// O+1 (post_ovulation) is never fertile.
			if ov := s.Anchors.EstimatedOvulationDate; ov.IsSet() && engine.DayOf(date) == ov+1 {
				assert.NotEqual(t, true, f["is_fertile_window"], msg)
				assert.Equal(t, false, ci["is_fertile_window"], msg)
				assert.NotEqual(t, "ovulation", ci["phase"], msg)
			}
		}
		if !tc.strip {
			assert.True(t, sawFertile, tc.name+" has a fertile window")
		}
	}
}

// heroPhase is the phase the home hero / cycle_view shows for a day (view.LegacyFromV6: the I-11 content phase), with
// the fertility strip of a mode without fertility copy.
func heroPhase(st *engine.State, s *engine.Status, today engine.Day) any {
	ui := st.TodayUI(*s)
	lg := view.LegacyFromV6(view.V6Input{State: st, Status: s, UI: &ui, Today: today, Locale: "fa"})
	if lg.Phase == nil || !s.CycleDay.OK() {
		return nil
	}
	p := *lg.Phase
	if s.Limits.FertilitySuppressed || view.NoFertilityCopy(st.Mode()) {
		p = strippedPhase(p)
	}
	return string(p)
}

// Agreement: the daily message phase (context_info and primary_message) and the facts phase equal the home hero
// phase, for 8 personas × 60 days.
func TestV6_DailyPhaseEqualsHeroPhase(t *testing.T) {
	today := engine.DayOf(v6Today)
	for name, in := range v6Personas() {
		src := newEngineSrc(in, "non_ttc", "")
		for i := -30; i < 30; i++ {
			date := v6Today.AddDays(i)
			s := src.st.Resolve(engine.DayOf(date), today)
			want := heroPhase(src.st, &s, today)
			m := generate(t, src, date)
			msg := name + " " + date.String()
			assert.Equal(t, want, m["context_info"].(map[string]any)["phase"], msg)
			if pm := m["primary_message"].(map[string]any); s.CycleDay.OK() {
				assert.Equal(t, want, pm["phase"], msg)
			}
			f := EngineFacts(&EngineDay{Status: s, Mode: src.st.Mode()}, false)
			got, ok := f["phase"]
			if !ok {
				got = nil
			}
			assert.Equal(t, want, got, msg)
		}
	}
}

// Go-native goldens of the v6 daily message and facts per persona (today, ±10 days). Rewrite with
// UPDATE_GOLDEN_V6=1 go test ./internal/messages/manager/ and review the diff by hand.
func TestV6_Goldens(t *testing.T) {
	names := make([]string, 0, 8)
	for n := range v6Personas() {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		src := newEngineSrc(v6Personas()[name], "non_ttc", "")
		out := jsonx.NewObject()
		for _, off := range []int{-10, 0, 10} {
			date := v6Today.AddDays(off)
			res, err := New(src, defaultsContent{}, "fa", v6Today).Generate(context.Background(), date, "")
			require.NoError(t, err)
			s := src.st.Resolve(engine.DayOf(date), engine.DayOf(v6Today))
			facts := EngineFacts(&EngineDay{Status: s, Mode: src.st.Mode()}, false)
			out.Set(date.String(), jsonx.Obj("daily", res.JSON(), "facts", sortedFacts(facts)))
		}
		got, err := jsonx.Marshal(out, jsonx.UnescapedUnicode)
		require.NoError(t, err)
		var pretty map[string]any
		require.NoError(t, json.Unmarshal(got, &pretty))
		path := filepath.Join("testdata", "golden-v6", name+".json")
		if os.Getenv("UPDATE_GOLDEN_V6") == "1" {
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, append(got, '\n'), 0o600))
			continue
		}
		want, err := os.ReadFile(path)
		require.NoError(t, err, "missing golden: run with UPDATE_GOLDEN_V6=1")
		assert.JSONEq(t, string(want), string(got), name)
	}
}

func sortedFacts(f map[string]any) *jsonx.OrderedMap {
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := jsonx.NewObject()
	for _, k := range keys {
		out.Set(k, f[k])
	}
	return out
}

// The I-11 table (DECISIONS.md) through EngineDay.ContentPhase.
func TestV6_ContentPhaseTable(t *testing.T) {
	for sub, want := range map[engine.Subphase]enums.CyclePhase{
		engine.SubphaseMenstrual: enums.CyclePhaseMenstruation, engine.SubphaseMenstrualPossible: enums.CyclePhaseMenstruation,
		engine.SubphaseEarlyFollicular: enums.CyclePhaseFollicular, engine.SubphaseFertileRising: enums.CyclePhaseFollicular,
		engine.SubphaseHighFertility: enums.CyclePhaseOvulation, engine.SubphaseOvulationLikely: enums.CyclePhaseOvulation,
		engine.SubphasePostOvulation: enums.CyclePhaseLuteal, engine.SubphasePMSPossible: enums.CyclePhaseLuteal,
		engine.SubphasePeriodExpected: enums.CyclePhaseLuteal, engine.SubphaseUnknown: "",
	} {
		d := EngineDay{Status: engine.Status{Subphase: sub, MainPhase: sub.Main(), CycleDay: engine.Some(10)}}
		assert.Equal(t, want, d.ContentPhase(), sub.String())
	}
}
