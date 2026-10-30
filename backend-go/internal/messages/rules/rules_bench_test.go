package rules_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ritme/backend-go/internal/messages/rules"
)

// T-M9-08 baseline: rules.Select over 200 active rules (README §8 ceiling ≤ 200 µs) with the history prefetched
// in one grouped query (T-M9-13). The mix is what an admin-grown table looks like: a third of the rules do not
// match the user's facts, a quarter are capped or cooling down after a recent impression, the rest compete on
// priority for the 7 slots.
func BenchmarkSelect_200Rules(b *testing.B) {
	const n = 200
	all := make([]rules.Rule, 0, n)
	imp := map[uint64][]time.Time{}
	ids := make([]uint64, 0, n)
	for i := range n {
		id := uint64(i + 1) //nolint:gosec // small
		r := capped(id, 999-i, 1+i%3, []string{"day", "week", "cycle"}[i%3], i%5)
		mode := "cycle"
		if i%3 == 0 {
			mode = "pregnancy" // never matches this user
		}
		r.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{
			{Fact: "mode", Op: "eq", Value: mode},
			{Fact: "cycle_day", Op: "between", Value: []any{float64(1 + i%10), float64(20 + i%20)}},
			{Fact: "symptoms", Op: "nin", Value: fmt.Sprintf("s%d", i%17)},
		}}}
		all = append(all, r)
		if i%4 == 0 {
			imp[id] = []time.Time{noon.Add(-time.Duration(i+1) * time.Hour)}
		}
		ids = append(ids, id)
	}
	facts := rules.Facts{"mode": "cycle", "cycle_day": 12.0, "cycle_length": 29.0, "symptoms": []string{"s1"}}
	w := rules.Windows(noon, facts)
	stats := rules.NewStats(nil, ids, w, statRows(imp, ids, w))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := rules.Select(all, facts, 7, noon, stats); err != nil {
			b.Fatal(err)
		}
	}
}

// T-M10-04 budget: SelectSlot for one slot over 200 parsed rules with prefetched facts and history ≤ 200 µs.
func BenchmarkSelectSlot_200Rules(b *testing.B) {
	const n = 200
	all := make([]rules.Rule, 0, n)
	h := &hist{}
	for i := range n {
		id := uint64(i + 1) //nolint:gosec // small
		r := slotRule(id, "daily_advice", rules.KindContent, 100+i%500)
		r.Caps = []rules.Cap{{Count: 1 + i%3, Period: []string{"day", "week", "cycle"}[i%3]}}
		r.CooldownDays = i % 5
		mode := "cycle"
		if i%3 == 0 {
			mode = "pregnancy"
		}
		r.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{
			{Fact: "mode", Op: "eq", Value: mode},
			{Fact: "cycle_day", Op: "between", Value: []any{float64(1 + i%10), float64(20 + i%20)}},
			{Fact: "symptoms", Op: "nin", Value: fmt.Sprintf("s%d", i%17)},
		}}}
		if i%4 == 0 {
			h.shown(r, tNow.Add(-time.Duration(i+1)*time.Hour))
		}
		all = append(all, r)
	}
	facts := rules.Facts{"mode": "cycle", "cycle_day": 12.0, "cycle_length": 29.0, "symptoms": []string{"s1"},
		"current_period_start": "2026-10-10", "confidence": "high"}
	uh := h.build(tNow, facts)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = rules.SelectSlot("daily_advice", all, facts, uh, 7, tNow)
	}
}
