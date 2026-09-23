package manager

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/civildate"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// defaultsContent resolves every entry from the embedded code fallback (no DB rows).
type defaultsContent struct{}

func (defaultsContent) Resolve(_ context.Context, group, itemKey, locale string) (content.Payload, error) {
	return content.Default(group, itemKey, locale), nil
}

// dailyLogColumns parses the daily_health_logs columns out of the goose baseline.
func dailyLogColumns(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("../../../db/migrations/00001_baseline.sql")
	require.NoError(t, err)
	s := string(b)
	start := strings.Index(s, "CREATE TABLE `daily_health_logs`")
	require.GreaterOrEqual(t, start, 0)
	end := strings.Index(s[start:], "ENGINE=")
	col := regexp.MustCompile("(?m)^  `([a-z_]+)` ")
	var cols []string
	for _, m := range col.FindAllStringSubmatch(s[start:start+end], -1) {
		cols = append(cols, m[1])
	}
	require.Contains(t, cols, "energy_level")
	return cols
}

// TestDeadFieldReads documents D-05: of every daily-log field extractSymptoms and PatternLayer
// read, only LoadedLogColumns exist in the schema — the rest always read null.
func TestDeadFieldReads(t *testing.T) {
	cols := dailyLogColumns(t)
	var live []string
	for _, f := range symptomLogFields {
		if slices.Contains(cols, f) {
			live = append(live, f)
		}
	}
	assert.ElementsMatch(t, LoadedLogColumns, live)
	for _, f := range LoadedLogColumns {
		assert.Contains(t, cols, f)
	}
}

// TestOnlyLowEnergyCanFire feeds every stored value of the two live columns (their enums,
// plus the legacy values the code still checks): only "low_energy" is ever extracted, because
// sleep_quality stores good/medium/bad, never poor/very_poor.
func TestOnlyLowEnergyCanFire(t *testing.T) {
	fired := map[string]bool{}
	for _, e := range append(enums.EnergyLevelValues(), "") {
		for _, s := range append(enums.SleepQualityValues(), "") {
			attrs := map[string]any{"energy_level": nil, "sleep_quality": nil}
			if e != "" {
				attrs["energy_level"] = e
			}
			if s != "" {
				attrs["sleep_quality"] = s
			}
			l := NewLog(attrs)
			for _, sym := range ExtractSymptoms(&l) {
				fired[sym] = true
			}
		}
	}
	assert.Equal(t, map[string]bool{"low_energy": true}, fired)
	assert.Equal(t, []string{}, ExtractSymptoms(nil))

	// The dead reads do work when the attribute exists (as in PHP) — proof they are dead only
	// because of the schema.
	l := NewLog(map[string]any{"has_cramps": true, "mood": "sad", "sleep_quality": "poor", "has_acne": 1})
	assert.Equal(t, []string{"cramps", "mood_sad", "poor_sleep", "acne"}, ExtractSymptoms(&l))
}

type fakeSource struct {
	profile *Profile
	preg    *pstore.PregnancyProfile
	log     *Log
	recent  []Log
	calc    legacy.Calculation
}

func (f *fakeSource) Profile(context.Context) (*Profile, error) { return f.profile, nil }
func (f *fakeSource) PregnancyProfile(context.Context) (*pstore.PregnancyProfile, error) {
	return f.preg, nil
}
func (f *fakeSource) DailyLog(context.Context, civildate.Date) (*Log, error) { return f.log, nil }
func (f *fakeSource) RecentLogs(context.Context, civildate.Date, civildate.Date) ([]Log, error) {
	return f.recent, nil
}
func (f *fakeSource) Cycle(context.Context, civildate.Date) (legacy.Calculation, error) {
	return f.calc, nil
}

func lowEnergy() *Log {
	l := NewLog(map[string]any{"energy_level": "low", "sleep_quality": "bad"})
	return &l
}

func logs(n int, energy string) []Log {
	out := make([]Log, n)
	for i := range out {
		out[i] = NewLog(map[string]any{"energy_level": energy, "sleep_quality": "bad"})
	}
	return out
}

var day = civildate.MustParse("2026-09-23")

func cycleCalc() legacy.Calculation {
	return legacy.Calculation{
		Complete: true, Day: 20, Phase: enums.CyclePhaseLuteal, CurrentSubphase: enums.CycleSubphaseEarlyLuteal,
		CycleLength: 30, OvulationDay: 17,
	}
}

func toJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// TestLowEnergyCycleDayIs500 is D-11: a low-energy day in cycle mode reaches the undefined
// OverrideType::LOW_ENERGY.
func TestLowEnergyCycleDayIs500(t *testing.T) {
	src := &fakeSource{profile: &Profile{UserGoal: "non_ttc", SubscriptionType: "premium", HasLastPeriodStart: true},
		log: lowEnergy(), calc: cycleCalc()}
	_, err := New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
	require.ErrorIs(t, err, ErrUndefinedOverrideCase)
}

func TestCycleResultShape(t *testing.T) {
	src := &fakeSource{profile: &Profile{UserGoal: "ttc", SubscriptionType: "premium", HasLastPeriodStart: true},
		recent: logs(16, "very_low"), calc: cycleCalc()}
	res, err := New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
	require.NoError(t, err)
	m := toJSON(t, res.JSON())

	assert.Equal(t, "cycle", m["mode"])
	pm := m["primary_message"].(map[string]any)
	assert.Equal(t, false, pm["has_override"])
	assert.Equal(t, "luteal", pm["phase"])
	assert.Contains(t, pm, "ttc_tips")
	var types []string
	for _, p := range m["patterns"].([]any) {
		types = append(types, p.(map[string]any)["pattern_type"].(string))
	}
	assert.Equal(t, []string{"ttc_tracking", "chronic_fatigue"}, types)
	nut := m["supplements"].(map[string]any)["nutrition"].(map[string]any)
	assert.Contains(t, nut, "ttc_additions")
	assert.NotContains(t, nut["ttc_additions"], "fertile_window_tip", "only on ovulation")
	assert.Equal(t, []any{}, m["tips"])
	assert.Equal(t, []any{}, m["correlations"])
}

func TestPatternsFreeAndInsufficient(t *testing.T) {
	src := &fakeSource{profile: &Profile{UserGoal: "non_ttc", SubscriptionType: "free", HasLastPeriodStart: true},
		recent: logs(20, "low"), calc: cycleCalc()}
	res, err := New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
	require.NoError(t, err)
	assert.Empty(t, res.Patterns, "patterns are premium only")

	src.profile.SubscriptionType = "premium"
	src.recent = logs(13, "low")
	res, err = New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
	require.NoError(t, err)
	require.Len(t, res.Patterns, 1)
	v, _ := res.Patterns[0].Get("pattern_type")
	assert.Equal(t, "insufficient_data", v)
}

func TestEmptyCalculationUsesDefaultMessage(t *testing.T) {
	src := &fakeSource{profile: &Profile{UserGoal: "non_ttc", SubscriptionType: "free"}}
	res, err := New(src, defaultsContent{}, "fa", day).Generate(context.Background(), day, enums.MessageModeCycle)
	require.NoError(t, err)
	m := toJSON(t, res.JSON())
	pm := m["primary_message"].(map[string]any)
	assert.Nil(t, pm["phase"])
	assert.Equal(t, "به سلامت خود توجه کنید", pm["short_message"])
	ci := m["context_info"].(map[string]any)
	assert.Nil(t, ci["cycle_day"])
	assert.Equal(t, false, ci["is_fertile_window"])
}

func TestPostpartumIsEmptyResult(t *testing.T) {
	res, err := New(&fakeSource{}, defaultsContent{}, "en", day).Generate(context.Background(), day, enums.MessageModePostpartum)
	require.NoError(t, err)
	b, err := json.Marshal(res.JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"mode":"postpartum","date":"2026-09-23","user_goal":"non_ttc","subscription_type":"free",
		"context_info":{"error":"Message engine not found for this mode"},"primary_message":[],"correlations":[],
		"patterns":[],"supplements":{"nutrition":[],"sleep":[],"exercise":[]},"tips":[]}`, string(b))
}

// TestWeekMilestones covers getWeekSpecificMessage: the first closest milestone within ±2
// weeks, otherwise a generated line; the 0-based week 0 gets the default message.
func TestWeekMilestones(t *testing.T) {
	e := pregnancyEngine{content: defaultsContent{}, locale: "en"}
	ctx := context.Background()
	for week, want := range map[int]string{
		2: "Embryo has implanted", 6: "Embryo has implanted", 10: "Baby's heart has started beating",
		16: "Week 16 of pregnancy", 18: "Halfway! Anomaly scan", 38: "Baby is getting ready", 42: "Due date! Be ready to meet",
	} {
		p, err := e.weekSpecific(ctx, week)
		require.NoError(t, err)
		got, _ := p.Field("short")
		assert.Equal(t, want, got, "week %d", week)
	}

	zero, one := 0, 1
	base, err := e.base(ctx, &Context{Mode: enums.MessageModePregnancy, PregnancyWeek: &zero, Trimester: &one})
	require.NoError(t, err)
	w, _ := base.Get("week")
	assert.Nil(t, w, "week 0 is falsy → default message")

	sixteen, two := 16, 2
	base, err = e.base(ctx, &Context{Mode: enums.MessageModePregnancy, Locale: "en", PregnancyWeek: &sixteen, PregnancyDay: &one, Trimester: &two})
	require.NoError(t, err)
	long, _ := base.Get("long_message")
	tm := content.Default("pregnancy_trimester", "2", "en")
	assert.Equal(t, tm.Or("long", ""), long, "no milestone → the trimester copy")
	ga, _ := base.Get("gestational_age")
	assert.Equal(t, "Week 16, Day 1", ga)
}

// TestPregnancyLowEnergyOverride: in pregnancy mode low energy maps to the fatigue override.
func TestPregnancyLowEnergyOverride(t *testing.T) {
	e := pregnancyEngine{content: defaultsContent{}, locale: "en"}
	o, err := e.override(context.Background(), &Context{Symptoms: []string{"low_energy"}})
	require.NoError(t, err)
	v, _ := o.Get("override_type")
	assert.Equal(t, "fatigue", v)
}
