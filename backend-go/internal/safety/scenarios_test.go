package safety

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

func byName(list []Scenario) map[string]Scenario {
	out := map[string]Scenario{}
	for _, s := range list {
		out[s.Name] = s
	}
	return out
}

func failures(r Run) map[string][]string {
	out := map[string][]string{}
	for _, res := range r.Results {
		if !res.Passed {
			out[res.Scenario] = res.Detail.Failures()
		}
	}
	return out
}

// TestUrgentScenarios is the nightly urgent-path run (.github/workflows/safety-scenarios-nightly.yml): every path
// scenario gives its verdict at threshold − 1, threshold and threshold + 1 with the clinical config in force today.
func TestUrgentScenarios(t *testing.T) {
	now := time.Now()
	r := RunScenarios(context.Background(), PureEnv(civildate.InTehran(now)), PathScenarios, now)
	assert.Empty(t, failures(r))
	assert.Equal(t, len(PathScenarios), r.Passed)
	assert.GreaterOrEqual(t, r.Total, 20)
	for _, res := range r.Results {
		assert.NotEmpty(t, res.Detail.Threshold, res.Scenario)
		assert.GreaterOrEqual(t, len(res.Detail.Probes), 2, res.Scenario)
	}
}

func TestScenarioNamesUniqueAndKinds(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Scenarios() {
		assert.False(t, seen[s.Name], s.Name)
		seen[s.Name] = true
		assert.Equal(t, s.Kind, KindOf(s.Name), s.Name)
		assert.LessOrEqual(t, len(s.Name), 64) // safety_scenario_runs.scenario
	}
}

func TestRunScenarios_ReportsFailuresAndPanics(t *testing.T) {
	list := []Scenario{
		{"ok", KindPath, func(context.Context, *Env) Detail { return Detail{Probes: []Probe{{"x", true, true}}} }},
		{"wrong", KindPath, func(context.Context, *Env) Detail { return Detail{Probes: []Probe{{"x", true, false}}} }},
		{"boom", KindPath, func(context.Context, *Env) Detail { panic("x") }},
	}
	r := RunScenarios(context.Background(), PureEnv(civildate.MustParse("2026-10-21")), list, time.Now())
	assert.Equal(t, 1, r.Passed)
	assert.Equal(t, []string{"wrong", "boom"}, r.Failed)
	assert.Equal(t, []string{"x: want true, got false"}, r.Results[1].Detail.Failures())
	assert.Equal(t, []string{"panic: x"}, r.Results[2].Detail.Problems)
}

// Without a database the configuration checks fail (never pass silently).
func TestConfigChecks_NeedTheDatabase(t *testing.T) {
	r := RunScenarios(context.Background(), PureEnv(civildate.MustParse("2026-10-21")), ConfigChecks, time.Now())
	assert.Len(t, r.Failed, len(ConfigChecks)-1, "config.registry needs no database")
	assert.Equal(t, []string{"no database"}, r.Results[0].Detail.Problems)
}

func TestNightlyCron_0300Tehran(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 11, h, m, 0, 0, civildate.Tehran) }
	next := func(after time.Time) time.Time {
		n, err := queue.Next(NightlyCron, after)
		require.NoError(t, err)
		return n
	}
	assert.True(t, at(3, 0).Equal(next(at(1, 0))))
	assert.True(t, at(3, 0).AddDate(0, 0, 1).Equal(next(at(3, 0))))
	assert.True(t, at(3, 0).AddDate(0, 0, 1).Equal(next(at(22, 30))))
	assert.True(t, at(3, 0).Equal(next(at(1, 0).UTC())), "the schedule is Tehran wall clock whatever the input zone")
}

func TestReducedKicks(t *testing.T) {
	assert.True(t, ReducedKicks(6, 2*time.Hour))
	assert.False(t, ReducedKicks(6, time.Hour))
	assert.False(t, ReducedKicks(10, 3*time.Hour))
}

func TestWindows(t *testing.T) {
	cur, prev := windows(time.Date(2026, 10, 11, 15, 0, 0, 0, civildate.Tehran))
	assert.Equal(t, "2026-10-05", cur.fromDay.String())
	assert.Equal(t, "2026-10-12", cur.toDayEx.String())
	assert.Equal(t, "2026-09-28", prev.fromDay.String())
	assert.Equal(t, cur.from, prev.to)
}

// The path scenarios follow the registry: a moved boundary (valid version) moves the probes and they still pass.
func TestScenarios_FollowTheRegistry(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	day := civildate.MustParse("2026-10-21")
	from := civildate.MustParse("2026-10-01")
	v := func(key string, val clinicalconfig.Value) clinicalconfig.Version {
		return clinicalconfig.Version{Key: key, Version: 2, Value: val, EffectiveFrom: from}
	}
	for _, total := range []float64{13, 20, 31} {
		clinicalconfig.SetVersions([]clinicalconfig.Version{
			v(clinicalconfig.KeyEPDSUrgentTotal, clinicalconfig.Value{"score": total}),
			v(clinicalconfig.KeyPostpartumRedFlags, clinicalconfig.Value{"fever_c": 38.5}),
			v(clinicalconfig.KeyPregnancyKicks, clinicalconfig.Value{"target": 8, "window_min": 90, "from_week": 26}),
			v(clinicalconfig.KeyBPCrisis, clinicalconfig.Value{"systolic": 170, "diastolic": 115}),
			v(clinicalconfig.KeyPregnancyBPHigh, clinicalconfig.Value{"systolic": 140, "diastolic": 90, "urgent": 0}),
			v(clinicalconfig.KeyPregnancyContractions511, clinicalconfig.Value{"interval_min": 4, "duration_s": 60, "period_min": 60}),
		})
		r := RunScenarios(context.Background(), PureEnv(day), PathScenarios, time.Now())
		assert.Empty(t, failures(r), "urgent_total %v", total)
		res := r.Results[indexOf(r, "preg.kicks_count")]
		assert.Equal(t, "< 8 kicks in 1h30m0s from week 26", res.Detail.Threshold)
	}
	// An invalid stored value (out of the key's range) fails exactly config.registry.
	clinicalconfig.SetVersions([]clinicalconfig.Version{v(clinicalconfig.KeyPregnancyKicks,
		clinicalconfig.Value{"target": 2, "window_min": 120, "from_week": 28})})
	r := RunScenarios(context.Background(), PureEnv(day), append(append([]Scenario{}, PathScenarios...),
		byName(ConfigChecks)["config.registry"]), time.Now())
	assert.Equal(t, []string{"config.registry"}, r.Failed)
	assert.Equal(t, []string{"pregnancy.kicks v2: range:target"}, r.Results[len(r.Results)-1].Detail.Problems)
}

func indexOf(r Run, name string) int {
	for i, res := range r.Results {
		if res.Scenario == name {
			return i
		}
	}
	return -1
}

func TestDetail_JSON(t *testing.T) {
	b, err := json.Marshal(Detail{Threshold: "> 180", Probes: []Probe{{"181/119", true, true}}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"threshold":"> 180","probes":[{"input":"181/119","want":true,"got":true}]}`, string(b))
}

func TestNewULID(t *testing.T) {
	a := newULID(time.UnixMilli(1_700_000_000_000))
	b := newULID(time.UnixMilli(1_700_000_000_001))
	assert.Regexp(t, runIDPattern, a)
	assert.Less(t, a, b)
	assert.Equal(t, a[:10], newULID(time.UnixMilli(1_700_000_000_000))[:10])
}
