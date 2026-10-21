package rules_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
)

// prefetchHistory is fakeHistory plus Prefetch: it answers the grouped read from the same impressions, so Select must
// choose the same rule with and without the prefetch.
type prefetchHistory struct {
	fakeHistory
	prefetches *int
	probes     *int
}

func (p prefetchHistory) Count(rule uint64, event string, since time.Time) (int, error) {
	*p.probes++
	return p.fakeHistory.Count(rule, event, since)
}

func (p prefetchHistory) Last(rule uint64, event string) (time.Time, bool, error) {
	*p.probes++
	return p.fakeHistory.Last(rule, event)
}

func (p prefetchHistory) Prefetch(ids []uint64, w [3]time.Time) (rules.History, error) {
	*p.prefetches++
	return rules.NewStats(p, ids, w, statRows(p.impressions, ids, w)), nil
}

func statRows(imp map[uint64][]time.Time, ids []uint64, w [3]time.Time) []store.ListUserMessageEventStatsRow {
	var out []store.ListUserMessageEventStatsRow
	for _, id := range ids {
		if len(imp[id]) == 0 {
			continue
		}
		r := store.ListUserMessageEventStatsRow{RuleID: id, Event: "impression"}
		for _, t := range imp[id] {
			if t.After(r.LastAt) {
				r.LastAt = t
			}
			if !t.Before(w[0]) {
				r.DayCount++
			}
			if !t.Before(w[1]) {
				r.WeekCount++
			}
			if !t.Before(w[2]) {
				r.CycleCount++
			}
		}
		out = append(out, r)
	}
	return out
}

func capped(id uint64, prio, capCount int, period string, cooldown int) rules.Rule {
	r := rule(id, rules.KindProduct, prio)
	r.CapCount, r.CapPeriod, r.CooldownDays = capCount, period, cooldown
	return r
}

func TestSelect_PrefetchMatchesPerRuleProbes(t *testing.T) {
	all := []rules.Rule{
		capped(1, 90, 1, "day", 0),   // shown 2 h ago → capped today
		capped(2, 80, 2, "week", 0),  // shown twice this week → capped
		capped(3, 70, 0, "", 5),      // shown 3 days ago, cooldown 5 → out
		capped(4, 60, 3, "cycle", 0), // shown twice this cycle → eligible
		capped(5, 50, 0, "", 0),
	}
	imp := map[uint64][]time.Time{
		1: {noon.Add(-2 * time.Hour)},
		2: {noon.Add(-2 * 24 * time.Hour), noon.Add(-4 * 24 * time.Hour)},
		3: {noon.Add(-3 * 24 * time.Hour)},
		4: {noon.Add(-10 * 24 * time.Hour), noon.Add(-20 * 24 * time.Hour)},
	}
	facts := rules.Facts{"cycle_length": 30.0}

	want, err := rules.Select(all, facts, 7, noon, fakeHistory{impressions: imp})
	require.NoError(t, err)
	require.NotNil(t, want)
	assert.Equal(t, uint64(4), want.Rule.ID)

	var prefetches, probes int
	got, err := rules.Select(all, facts, 7, noon, prefetchHistory{fakeHistory{imp}, &prefetches, &probes})
	require.NoError(t, err)
	assert.Equal(t, want.Rule.ID, got.Rule.ID)
	assert.Equal(t, 1, prefetches, "one grouped read")
	assert.Equal(t, 0, probes, "no per-rule probe")
}

func TestStats_FallsBackOutsideThePrefetch(t *testing.T) {
	var probes, prefetches int
	base := prefetchHistory{fakeHistory{map[uint64][]time.Time{9: {noon.Add(-time.Hour)}}}, &prefetches, &probes}
	w := rules.Windows(noon, rules.Facts{})
	s := rules.NewStats(base, []uint64{1}, w, nil)

	n, err := s.Count(1, "impression", w[1])
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	_, ok, _ := s.Last(1, "impression")
	assert.False(t, ok, "prefetched without rows = no history")
	assert.Equal(t, 0, probes)

	n, _ = s.Count(9, "impression", w[0]) // not prefetched
	assert.Equal(t, 1, n)
	_, _ = s.Count(1, "impression", noon.Add(-3*time.Hour)) // a window it was not prefetched for
	assert.Equal(t, 2, probes)
}

func TestWindows(t *testing.T) {
	w := rules.Windows(noon, rules.Facts{"cycle_length": 32.0})
	assert.Equal(t, noon.Add(-24*time.Hour), w[0])
	assert.Equal(t, noon.Add(-7*24*time.Hour), w[1])
	assert.Equal(t, noon.Add(-32*24*time.Hour), w[2])
	assert.Equal(t, noon.Add(-28*24*time.Hour), rules.Windows(noon, rules.Facts{})[2])
}

// T-M9-13 budget: Select over 50 active rules with prefetched stats ≤ 200 µs.
func BenchmarkSelect_PrefetchedStats50(b *testing.B) {
	all := make([]rules.Rule, 0, 50)
	imp := map[uint64][]time.Time{}
	ids := make([]uint64, 0, 50)
	for i := range 50 {
		id := uint64(i + 1) //nolint:gosec // small
		r := capped(id, 99-i, 1, []string{"day", "week", "cycle"}[i%3], i%4)
		r.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{
			{Fact: "mode", Op: "eq", Value: "cycle"},
			{Fact: "cycle_day", Op: "between", Value: []any{1.0, 40.0}},
			{Fact: "symptoms", Op: "nin", Value: fmt.Sprintf("s%d", i)},
		}}}
		all = append(all, r)
		imp[id] = []time.Time{noon.Add(-time.Duration(i+1) * time.Hour)} // every rule capped or cooling → full scan
		ids = append(ids, id)
	}
	facts := rules.Facts{"mode": "cycle", "cycle_day": 12.0, "cycle_length": 29.0, "symptoms": []string{"s1"}}
	w := rules.Windows(noon, facts)
	stats := rules.NewStats(nil, ids, w, statRows(imp, ids, w))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := rules.Select(all, facts, 7, noon, stats); err != nil {
			b.Fatal(err)
		}
	}
}
