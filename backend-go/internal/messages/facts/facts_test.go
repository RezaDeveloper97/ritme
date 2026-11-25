package facts_test

// T-M10-03 (D-161): the producer groups over engine personas built in memory (engine.Prepare) and counting fakes of
// the module sources — no database.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/racedet"
)

var today = civildate.MustParse("2026-10-21")

// closed is n closed 5-day periods every gap days, the last starting on last.
func closed(last civildate.Date, n, gap int) []engine.Period {
	out := make([]engine.Period, n)
	l := engine.DayOf(last)
	for i := range n {
		s := l - engine.Day((n-1-i)*gap)
		out[i] = engine.Period{Start: s, End: s + 4}
	}
	return out
}

// fake is every module source, counting the calls.
type fake struct {
	in        *engine.Input
	account   *facts.Account
	hormonal  bool
	birth     civildate.Date
	preg      *facts.PregnancyDating
	vitals    facts.LastVitals
	stats     facts.LogStats
	symptoms  []facts.SymptomEntry
	calls     map[string]int
	stateMemo *engine.State
}

func newFake() *fake { return &fake{calls: map[string]int{}} }

func (f *fake) EngineState(context.Context) (*engine.State, error) {
	f.calls["engine"]++
	if f.in == nil {
		return nil, nil
	}
	if f.stateMemo == nil {
		f.stateMemo = engine.Prepare(*f.in, engineparams.For(today))
	}
	return f.stateMemo, nil
}

func (f *fake) Account(context.Context) (*facts.Account, error) {
	f.calls["account"]++
	return f.account, nil
}

func (f *fake) HormonalContraception(context.Context) (bool, error) {
	f.calls["contraception"]++
	return f.hormonal, nil
}

func (f *fake) PostpartumBirth(context.Context) (civildate.Date, bool, error) {
	f.calls["postpartum"]++
	return f.birth, !f.birth.IsZero(), nil
}

func (f *fake) PregnancyDating(context.Context, civildate.Date) (*facts.PregnancyDating, error) {
	f.calls["pregnancy"]++
	return f.preg, nil
}

func (f *fake) LastVitals(context.Context, civildate.Date, civildate.Date) (facts.LastVitals, error) {
	f.calls["vitals"]++
	return f.vitals, nil
}

func (f *fake) LogStats(context.Context, civildate.Date, civildate.Date) (facts.LogStats, error) {
	f.calls["logs"]++
	return f.stats, nil
}

func (f *fake) SymptomEntries(context.Context, civildate.Date, civildate.Date) ([]facts.SymptomEntry, error) {
	f.calls["symptoms"]++
	return f.symptoms, nil
}

func (f *fake) sources() facts.Sources {
	return facts.Sources{Engine: f, Account: f, Logs: f, Vitals: f, Pregnancy: f, Postpartum: f, Contraception: f}
}

func all(t *testing.T, f *fake) facts.Values {
	t.Helper()
	fs := facts.New(f.sources(), today, "fa")
	require.NoError(t, fs.All(context.Background()))
	return fs.Values()
}

// Engine personas (SPEC §2.35/§2.42): late 9 days, pill user, TTC, teen.
func TestEngineFacts_Personas(t *testing.T) {
	regular := closed(today.AddDays(-12), 6, 28) // cycle day 13 = O−2 of a 28-day cycle
	for _, tc := range []struct {
		name  string
		in    engine.Input
		check func(t *testing.T, v facts.Values)
	}{
		{"late_9", engine.Input{Periods: closed(today.AddDays(-37), 6, 28)}, func(t *testing.T, v facts.Values) {
			assert.Equal(t, 9.0, v["days_late"])
			assert.Equal(t, 38.0, v["cycle_day"], "a late cycle keeps counting")
			assert.Equal(t, 0.0, v["days_to_period"])
			assert.Equal(t, "period_expected", v["main_phase"])
			assert.Contains(t, v["warnings"], "predicted_period_overdue")
			assert.Equal(t, "overdue", v["today_ui.banner"])
			assert.Equal(t, "cycle", v["mode"])
			assert.Equal(t, today.AddDays(-37).String(), v["current_period_start"])
		}},
		{"pill", engine.Input{Periods: regular, Hormonal: true}, func(t *testing.T, v facts.Values) {
			assert.Equal(t, true, v["fertility_suppressed"])
			assert.NotContains(t, v, "is_fertile_window", "no fertile window under the pill")
			assert.NotEqual(t, "fertile", v["main_phase"])
			assert.Contains(t, v["warnings"], "hormonal_contraception_active")
			assert.Equal(t, false, v["suggest_pregnancy_test"])
		}},
		{"ttc", engine.Input{Periods: regular, Mode: engine.ModeTTC}, func(t *testing.T, v facts.Values) {
			assert.Equal(t, "ttc", v["mode"])
			assert.Equal(t, false, v["fertility_suppressed"])
			assert.Equal(t, true, v["is_fertile_window"])
			assert.Equal(t, "fertile", v["main_phase"])
			assert.Equal(t, "regular", v["cycle_regularity"])
			assert.Equal(t, "none", v["today_ui.banner"])
		}},
		{"teen", engine.Input{Periods: regular, Mode: engine.ModeTeen}, func(t *testing.T, v facts.Values) {
			assert.Equal(t, "teen", v["mode"])
			assert.Equal(t, true, v["fertility_suppressed"])
			assert.NotContains(t, v, "is_fertile_window")
			assert.Equal(t, "follicular", v["main_phase"], "the fertile main phase reads follicular")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			f.in = &tc.in
			tc.check(t, all(t, f))
		})
	}
}

// An ambiguous period (a stale onboarding anchor, no logged period): the engine asks for input and says why.
func TestEngineFacts_AmbiguousPeriodRequiresInput(t *testing.T) {
	f := newFake()
	f.in = &engine.Input{Declared: engine.Declared{LastPeriodStart: engine.Some(engine.DayOf(today.AddDays(-200))),
		CycleLength: engine.Some(28), PeriodLength: engine.Some(5)}}
	v := all(t, f)
	assert.Equal(t, true, v["requires_user_input"])
	require.IsType(t, []string{}, v["warnings"])
	assert.NotEmpty(t, v["warnings"])
	assert.Equal(t, "ask_last_period", v["today_ui.banner"])
}

// Missing inputs give absent facts, never a guess: no engine, no account, no pregnancy, no readings.
func TestFacts_MissingInputsAreAbsent(t *testing.T) {
	v := all(t, newFake())
	for _, n := range []string{"mode", "main_phase", "cycle_day", "days_late", "confidence", "warnings", "age", "plan",
		"days_since_signup", "pregnancy_week", "pregnancy_trimester", "postpartum_week", "bp_last_category",
		"glucose_last_category", "days_since_last_log", "user_goal", "symptom_history.cramps"} {
		assert.NotContains(t, v, n, n)
	}
	assert.Equal(t, false, v["hormonal_contraception"])
	assert.Equal(t, 0.0, v["logs_30d"])
	assert.Equal(t, []string{}, v["segment"], "no segment source")
	assert.Equal(t, false, v["push_opt_in"], "M-09")
}

// Account, module and log groups.
func TestFacts_AccountAndModules(t *testing.T) {
	f := newFake()
	f.in = &engine.Input{Periods: closed(today.AddDays(-12), 6, 28), Hormonal: true}
	f.account = &facts.Account{Name: "Sara Test", SignupDay: today.AddDays(-40), Birthday: civildate.MustParse("2000-11-01"),
		UserGoal: "ttc", SubscriptionType: "free", Plan: "trial", LifeMode: "cycle"}
	f.hormonal = true
	f.stats = facts.LogStats{Last: today.AddDays(-3), Days30d: 12}
	f.vitals = facts.LastVitals{BP: "stage1"}
	v := all(t, f)
	assert.Equal(t, "trial", v["plan"])
	assert.Equal(t, 40.0, v["days_since_signup"])
	assert.Equal(t, 25.0, v["age"])
	assert.Equal(t, "fa", v["locale"])
	assert.Equal(t, "cycle", v["life_mode"])
	assert.Equal(t, "Sara", v["name"])
	assert.Equal(t, "non_ttc", v["user_goal"], "a pill user with goal ttc gets no TTC targeting")
	assert.Equal(t, true, v["hormonal_contraception"])
	assert.Equal(t, 3.0, v["days_since_last_log"])
	assert.Equal(t, 12.0, v["logs_30d"])
	assert.Equal(t, "stage1", v["bp_last_category"])
	assert.NotContains(t, v, "glucose_last_category")
}

// Pregnant 30+2 and postpartum week 5.
func TestFacts_PregnancyAndPostpartum(t *testing.T) {
	f := newFake()
	f.preg = &facts.PregnancyDating{Week: 31, Trimester: 3}
	v := all(t, f)
	assert.Equal(t, 31.0, v["pregnancy_week"])
	assert.Equal(t, 3.0, v["pregnancy_trimester"])

	f = newFake()
	f.birth = today.AddDays(-30) // 4 weeks + 2 days → week 5
	assert.Equal(t, 5.0, all(t, f)["postpartum_week"])
	f = newFake()
	f.birth = today.AddDays(3)
	assert.NotContains(t, all(t, f), "postpartum_week", "a birth after today is not a postpartum week")
}

// symptom_history.cramps = 3: cramps logged on menstrual days of three cycles (twice in one of them), plus a luteal
// day that does not count; today is a menstrual day.
func TestFacts_SymptomHistory(t *testing.T) {
	last := today.AddDays(-1) // today = cycle day 2 (menstrual)
	f := newFake()
	f.in = &engine.Input{Periods: closed(last, 6, 28)}
	cramp := func(d civildate.Date, level string) facts.SymptomEntry {
		return facts.SymptomEntry{Date: d, Category: "pain", Param: "location", Item: "abdomen", Level: level}
	}
	f.symptoms = []facts.SymptomEntry{
		cramp(last.AddDays(-56), "mild"), cramp(last.AddDays(-55), "severe"), // one cycle, two days
		cramp(last.AddDays(-28), "moderate"),
		cramp(last, "severe"),
		cramp(last.AddDays(-10), "severe"), // luteal day of the previous cycle
		cramp(last.AddDays(-84), "no"),     // logged as «no»
		{Date: last.AddDays(-84), Category: "symptoms", Param: "digestive", Item: "nausea", Level: "yes"},
	}
	v := all(t, f)
	assert.Equal(t, "menstrual", v["main_phase"])
	assert.Equal(t, 3.0, v["symptom_history.cramps"])
	assert.Equal(t, 1.0, v["symptom_history.nausea"])
	assert.Equal(t, 0.0, v["symptom_history.headache"])
}

// Laziness: rules that read only engine facts load the engine state once and run no pregnancy, vitals or log query.
func TestFacts_Laziness(t *testing.T) {
	f := newFake()
	f.in = &engine.Input{Periods: closed(today.AddDays(-12), 6, 28)}
	fs := facts.New(f.sources(), today, "fa")
	ctx := context.Background()
	require.NoError(t, fs.Need(ctx, "main_phase", "days_late", "warnings", "today_ui.banner"))
	require.NoError(t, fs.Need(ctx, "cycle_day", "unknown_fact"))
	assert.Equal(t, 1, f.calls["engine"], "one engine state load")
	for _, g := range []string{"pregnancy", "vitals", "logs", "symptoms", "postpartum", "contraception", "account"} {
		assert.Zero(t, f.calls[g], g)
	}
	// symptom_history reuses the same state; user_goal of a non-ttc goal never needs the engine.
	require.NoError(t, fs.Need(ctx, "symptom_history.cramps"))
	assert.Equal(t, 1, f.calls["engine"])
	assert.Equal(t, 1, f.calls["symptoms"])

	g := newFake()
	g.account = &facts.Account{UserGoal: "non_ttc"}
	gs := facts.New(g.sources(), today, "")
	require.NoError(t, gs.Need(ctx, "user_goal", "plan", "age"))
	assert.Zero(t, g.calls["engine"])
	assert.Equal(t, 1, g.calls["account"], "the account row is read once for the account and goal groups")
}

// Every fact has a known type, a group with a producer and both seed descriptions; enum facts list their values.
func TestRegistry_Definitions(t *testing.T) {
	groups := facts.Groups()
	for _, d := range facts.Defs() {
		assert.Contains(t, []facts.Type{facts.TypeInt, facts.TypeFloat, facts.TypeBool, facts.TypeEnum, facts.TypeSet,
			facts.TypeDate}, d.Type, d.Name)
		assert.Contains(t, groups, d.Group, d.Name)
		assert.NotEmpty(t, d.Description["en"], d.Name)
		assert.NotEmpty(t, d.Description["fa"], d.Name)
	}
	for _, n := range []string{"mode", "main_phase", "subphase", "cycle_day", "days_to_period", "days_late", "confidence",
		"data_quality", "warnings", "requires_user_input", "cycle_regularity", "suggest_pregnancy_test",
		"hormonal_contraception", "pregnancy_week", "pregnancy_trimester", "postpartum_week", "bp_last_category",
		"glucose_last_category", "symptom_history.cramps", "plan", "days_since_signup", "days_since_last_log", "logs_30d",
		"segment", "age", "push_opt_in", "fertility_suppressed", "is_fertile_window", "is_pms_window", "today_ui.banner",
		"current_period_start", "life_mode", "user_goal", "locale"} {
		_, ok := facts.Lookup(n)
		assert.True(t, ok, n)
	}
	d, _ := facts.Lookup("segment")
	assert.Empty(t, d.Pending)
	assert.Equal(t, facts.GroupSegment, d.Group)
}

// Budget: every engine fact from a warm state (the state already loaded) in ≤ 50 µs.
func TestBudget_EngineFactsWarm(t *testing.T) {
	if racedet.Enabled {
		t.Skip("latency budget is meaningless under -race (5–10× slower)")
	}
	f := newFake()
	f.in = &engine.Input{Periods: closed(today.AddDays(-12), 6, 28)}
	_, _ = f.EngineState(context.Background()) // warm: Prepare done
	var names []string
	for _, d := range facts.Defs() {
		if d.Group == facts.GroupEngine {
			names = append(names, d.Name)
		}
	}
	res := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			fs := facts.New(f.sources(), today, "fa")
			if err := fs.Need(context.Background(), names...); err != nil {
				b.Fatal(err)
			}
		}
	})
	t.Logf("engine facts from a warm state: %d ns/op", res.NsPerOp())
	assert.LessOrEqual(t, res.NsPerOp(), int64(50_000))
}

func BenchmarkEngineFactsWarm(b *testing.B) {
	f := newFake()
	f.in = &engine.Input{Periods: closed(today.AddDays(-12), 6, 28)}
	_, _ = f.EngineState(context.Background())
	for range b.N {
		fs := facts.New(f.sources(), today, "fa")
		_ = fs.Need(context.Background(), "main_phase", "warnings", "today_ui.banner", "cycle_day", "days_late")
	}
}

type segFake struct {
	*fake
	ids []string
}

func (s segFake) Segments(context.Context) ([]string, error) {
	s.calls["segments"]++
	return s.ids, nil
}

// T-M10-32: the segment fact comes from the account source's SegmentSource, only when a rule needs it.
func TestFacts_SegmentFromSource(t *testing.T) {
	src := segFake{fake: newFake(), ids: []string{"3", "7"}}
	f := facts.New(facts.Sources{Account: src}, today, "fa")
	_, _, err := f.Get(context.Background(), "plan")
	require.NoError(t, err)
	assert.Zero(t, src.calls["segments"], "account facts do not read segments")
	v, ok, err := f.Get(context.Background(), "segment")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []string{"3", "7"}, v)
	_, _, _ = f.Get(context.Background(), "segment")
	assert.Equal(t, 1, src.calls["segments"])
}
