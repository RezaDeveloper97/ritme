package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M10-11: conflict check v2 — slot, schedule, hours, audience and per-fact value-set intersection.

func cond(fact, op string, v any) rules.Condition {
	return rules.Condition{Fact: fact, Op: op, Value: v}
}

func and(cs ...rules.Condition) rules.Group { return rules.Group{Op: "and", Items: cs} }

func or(cs ...rules.Condition) rules.Group { return rules.Group{Op: "or", Items: cs} }

func ruleWith(id uint64, kind string, prio int, slot string, gs ...rules.Group) rules.Rule {
	r := rule(id, kind, prio)
	r.Placement, r.Groups = slot, gs
	return r
}

func between(lo, hi float64) rules.Condition { return cond("cycle_day", "between", []any{lo, hi}) }

func factRange(t *testing.T, o rules.Overlap, fact string) rules.FactRange {
	t.Helper()
	for _, f := range o.Facts {
		if f.Fact == fact {
			return f
		}
	}
	t.Fatalf("no range for %s in %+v", fact, o.Facts)
	return rules.FactRange{}
}

func TestConflictsV2_CycleDayIntervals(t *testing.T) {
	a := ruleWith(1, rules.KindClinical, 300, "smart_tip", and(between(1, 5)))
	b := ruleWith(2, rules.KindClinical, 200, "smart_tip", and(between(10, 14)))
	assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{a, b}), "1–5 vs 10–14: no common day")

	a.Groups = []rules.Group{and(between(1, 7))}
	b.Groups = []rules.Group{and(between(5, 10))}
	cs := rules.FindConflicts(&a, []rules.Rule{a, b})
	require.Len(t, cs, 1)
	assert.Equal(t, "this", cs[0].Winner, "higher priority")
	assert.Equal(t, "smart_tip", cs[0].Slot)
	r := factRange(t, cs[0].Overlap, "cycle_day")
	require.NotNil(t, r.Min)
	require.NotNil(t, r.Max)
	assert.Equal(t, 5.0, *r.Min)
	assert.Equal(t, 7.0, *r.Max)
	assert.False(t, r.Absent, "between needs a value")

	// int fact: open bounds become closed integers; gt 7 vs lt 8 has no integer.
	a.Groups = []rules.Group{and(cond("cycle_day", "gt", 7.0))}
	b.Groups = []rules.Group{and(cond("cycle_day", "lt", 8.0))}
	assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{b}))
	b.Groups = []rules.Group{and(cond("cycle_day", "lte", 8.0), cond("cycle_day", "ne", 8.0))}
	assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{b}), "the only integer is excluded")
}

func TestConflictsV2_SafetyAlwaysWins(t *testing.T) {
	safety := ruleWith(9, rules.KindSafety, 1000, "push", and(cond("days_late", "gte", 3.0)))
	product := ruleWith(1, rules.KindProduct, 50, "push", and(cond("days_late", "between", []any{0.0, 10.0})))
	cs := rules.FindConflicts(&product, []rules.Rule{safety})
	require.Len(t, cs, 1)
	assert.Equal(t, "other", cs[0].Winner)
	assert.True(t, cs[0].Safety)
	assert.False(t, cs[0].TieBreak)

	pairs := rules.ConflictPairs([]rules.Rule{product, safety})
	require.Len(t, pairs, 1)
	assert.Equal(t, uint64(9), pairs[0].Winner)
	assert.Equal(t, rules.WinBySafety, pairs[0].Reason)
	r := factRange(t, pairs[0].Overlap, "days_late")
	assert.Equal(t, 3.0, *r.Min)
	assert.Equal(t, 10.0, *r.Max)

	// Equal priority: decided by the least recently shown, then the lower id.
	p2 := ruleWith(2, rules.KindProduct, 50, "push")
	pairs = rules.ConflictPairs([]rules.Rule{p2, product})
	require.Len(t, pairs, 1)
	assert.Equal(t, uint64(1), pairs[0].Winner)
	assert.Equal(t, rules.WinByRecency, pairs[0].Reason)
	draft := ruleWith(0, rules.KindProduct, 50, "push")
	cs = rules.FindConflicts(&draft, []rules.Rule{product})
	require.Len(t, cs, 1)
	assert.Equal(t, "tie", cs[0].Winner)
}

func TestConflictsV2_ValuesSetsBoolsAndExceptions(t *testing.T) {
	mk := func(id uint64, gs ...rules.Group) rules.Rule {
		return ruleWith(id, rules.KindClinical, 300-int(id), "smart_tip", gs...)
	}
	none := func(a, b rules.Rule, msg string) {
		t.Helper()
		assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{b}), msg)
	}
	one := func(a, b rules.Rule, msg string) rules.Overlap {
		t.Helper()
		cs := rules.FindConflicts(&a, []rules.Rule{b})
		require.Len(t, cs, 1, msg)
		return cs[0].Overlap
	}

	none(mk(1, and(cond("mode", "eq", "cycle"))), mk(2, and(cond("mode", "eq", "pregnancy"))), "enum eq disjoint")
	o := one(mk(1, and(cond("mode", "in", []any{"cycle", "ttc"}))), mk(2, and(cond("mode", "ne", "cycle"))), "enum in ∩ ne")
	assert.Equal(t, []string{"ttc"}, factRange(t, o, "mode").Values)

	none(mk(1, and(cond("is_premium", "eq", true))), mk(2, and(cond("is_premium", "eq", "false"))), "booleans")
	one(mk(1, and(cond("is_premium", "eq", true))), mk(2, and(cond("is_premium", "ne", false))), "true ∩ ≠false")

	def, ok := facts.Lookup("warnings")
	require.True(t, ok)
	require.NotEmpty(t, def.Enum)
	w := def.Enum[0]
	none(mk(1, and(cond("warnings", "contains", w))), mk(2, and(cond("warnings", "nin", []any{w}))), "set contains vs nin")
	o = one(mk(1, and(cond("warnings", "contains", w))), mk(2, and(cond("warnings", "in", []any{w}))), "set contains ∩ in")
	assert.Equal(t, []string{w}, factRange(t, o, "warnings").Contains)

	// An "or" group branches: one branch overlaps.
	o = one(mk(1, or(cond("mode", "eq", "cycle"), cond("mode", "eq", "ttc"))), mk(2, and(cond("mode", "eq", "ttc"))), "or branch")
	assert.Equal(t, []string{"ttc"}, factRange(t, o, "mode").Values)

	// An exception removes its region: 5–10 except 5–7 leaves 8–10, which misses 1–7.
	b := mk(2, and(between(5, 10)))
	b.Exceptions = []rules.Condition{between(5, 7)}
	none(mk(1, and(between(1, 7))), b, "exception cuts the overlap")
	o = one(mk(1, and(between(1, 9))), b, "8–9 left")
	r := factRange(t, o, "cycle_day")
	assert.Equal(t, 8.0, *r.Min)
	assert.Equal(t, 9.0, *r.Max)

	// Disjoint facts: nothing excludes a user who matches both.
	o = one(mk(1, and(cond("mode", "eq", "cycle"))), mk(2, and(between(1, 3))), "different facts")
	assert.Len(t, o.Facts, 2)
	// ne on both sides: a user without the fact matches both.
	o = one(mk(1, and(cond("user_goal", "ne", "ttc"))), mk(2, and(cond("user_goal", "nin", []any{"non_ttc"}))), "missing fact")
	assert.True(t, factRange(t, o, "user_goal").Absent)
}

func TestConflictsV2_SlotScheduleHoursAudience(t *testing.T) {
	a := ruleWith(1, rules.KindProduct, 50, "push")
	b := ruleWith(2, rules.KindProduct, 40, "push")

	other := b
	other.Placement = "daily_challenge"
	assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{other}), "another slot")
	notAllowed := ruleWith(3, rules.KindProduct, 40, "smart_tip")
	a2 := ruleWith(4, rules.KindProduct, 50, "smart_tip")
	assert.Empty(t, rules.FindConflicts(&a2, []rules.Rule{notAllowed}), "kind never runs in the slot")
	inactive := b
	inactive.IsActive = false
	assert.Empty(t, rules.FindConflicts(&a, []rules.Rule{inactive}))

	// Schedules: [10-01, 10-10) vs [10-10, 10-20) touch only.
	at := func(d string) time.Time { return civildate.MustParse(d).TehranMidnight() }
	s1, s2 := a, b
	s1.StartsAt, s1.EndsAt = at("2026-10-01"), at("2026-10-10")
	s2.StartsAt, s2.EndsAt = at("2026-10-10"), at("2026-10-20")
	assert.Empty(t, rules.FindConflicts(&s1, []rules.Rule{s2}))
	s2.StartsAt = at("2026-10-05")
	cs := rules.FindConflicts(&s1, []rules.Rule{s2})
	require.Len(t, cs, 1)
	assert.Equal(t, at("2026-10-05"), cs[0].Overlap.Start)
	assert.Equal(t, at("2026-10-10"), cs[0].Overlap.End)

	// Hours: push quiet hours 23–08 are removed: 20–02 leaves 20–23, 01–09 leaves 08–09 → nothing in common.
	h1, h2 := a, b
	h1.HourWindows = []rules.HourWindow{{20, 2}}
	h2.HourWindows = []rules.HourWindow{{1, 9}}
	assert.Empty(t, rules.FindConflicts(&h1, []rules.Rule{h2}))
	h2.HourWindows = []rules.HourWindow{{21, 9}}
	cs = rules.FindConflicts(&h1, []rules.Rule{h2})
	require.Len(t, cs, 1)
	assert.Equal(t, []rules.HourWindow{{21, 23}}, cs[0].Overlap.Hours, "quiet hours 23–08 removed")

	// A wrapping window outside the quiet hours of a slot without them.
	t1, t2 := ruleWith(5, rules.KindClinical, 300, "smart_tip"), ruleWith(6, rules.KindClinical, 200, "smart_tip")
	t1.HourWindows = []rules.HourWindow{{22, 3}}
	cs = rules.FindConflicts(&t1, []rules.Rule{t2})
	require.Len(t, cs, 1)
	assert.Equal(t, []rules.HourWindow{{22, 3}}, cs[0].Overlap.Hours)

	// Audience: per-rule buckets, so the expected common share is the product.
	a.AudiencePercent, b.AudiencePercent = 50, 20
	cs = rules.FindConflicts(&a, []rules.Rule{b})
	require.Len(t, cs, 1)
	assert.InDelta(t, 10.0, cs[0].Overlap.AudiencePercent, 1e-9)
	assert.Equal(t, []rules.HourWindow{{8, 23}}, cs[0].Overlap.Hours, "push: every hour but the quiet ones")
	assert.False(t, cs[0].CoShown)

	adv1, adv2 := ruleWith(7, rules.KindContent, 300, "daily_advice"), ruleWith(8, rules.KindContent, 200, "daily_advice")
	cs = rules.FindConflicts(&adv1, []rules.Rule{adv2})
	require.Len(t, cs, 1)
	assert.Nil(t, cs[0].Overlap.Hours, "every hour")
	assert.True(t, cs[0].CoShown, "daily_advice shows 3")
}
