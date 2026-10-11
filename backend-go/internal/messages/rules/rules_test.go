package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type fakeHistory struct {
	impressions map[uint64][]time.Time
}

func (f fakeHistory) Count(rule uint64, event string, since time.Time) (int, error) {
	n := 0
	for _, t := range f.impressions[rule] {
		if !t.Before(since) {
			n++
		}
	}
	return n, nil
}

func (f fakeHistory) Last(rule uint64, event string) (time.Time, bool, error) {
	var last time.Time
	for _, t := range f.impressions[rule] {
		if t.After(last) {
			last = t
		}
	}
	return last, !last.IsZero(), nil
}

func rule(id uint64, kind string, prio int) rules.Rule {
	return rules.Rule{
		ID: id, Level: rules.LevelInfo, Kind: kind, Priority: prio, Placement: "smart_tip",
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}}, ABSplit: 50,
		HoursFrom: -1, HoursTo: -1, AudiencePercent: 100, Status: rules.StatusApproved, IsActive: true,
	}
}

var noon = time.Date(2026, 10, 11, 12, 0, 0, 0, civildate.Tehran)

func TestSelect_SafetyFirstThenPriority(t *testing.T) {
	all := []rules.Rule{rule(1, rules.KindProduct, 60), rule(2, rules.KindClinical, 700), rule(3, rules.KindSafety, 1000), rule(4, rules.KindEngine, 900)}
	ch, err := rules.Select(all, rules.Facts{}, 7, noon, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), ch.Rule.ID, "safety wins whatever the others are")

	all = all[:2]
	all = append(all, rule(4, rules.KindEngine, 900))
	ch, _ = rules.Select(all, rules.Facts{}, 7, noon, nil)
	assert.Equal(t, uint64(4), ch.Rule.ID, "engine band above clinical above product")

	// A tie goes to the lower id.
	ch, _ = rules.Select([]rules.Rule{rule(9, rules.KindClinical, 300), rule(5, rules.KindClinical, 300)}, rules.Facts{}, 1, noon, nil)
	assert.Equal(t, uint64(5), ch.Rule.ID)

	ch, _ = rules.Select(nil, rules.Facts{}, 1, noon, nil)
	assert.Nil(t, ch)
}

func TestSelect_SafetyBypassesCapsMedicalNeedsApproval(t *testing.T) {
	s := rule(1, rules.KindSafety, 1000)
	s.CapCount, s.CapPeriod, s.CooldownDays = 1, "day", 5
	h := fakeHistory{impressions: map[uint64][]time.Time{1: {noon.Add(-time.Hour)}}}
	ch, _ := rules.Select([]rules.Rule{s}, rules.Facts{}, 1, noon, h)
	require.NotNil(t, ch, "safety has no repeat cap or cooldown")

	s.Status = rules.StatusPending
	ch, _ = rules.Select([]rules.Rule{s}, rules.Facts{}, 1, noon, h)
	assert.Nil(t, ch, "an unapproved medical message is never served")

	p := rule(2, rules.KindProduct, 30)
	p.Level = rules.LevelSuggestion
	p.Status = rules.StatusDraft
	ch, _ = rules.Select([]rules.Rule{p}, rules.Facts{}, 1, noon, nil)
	assert.NotNil(t, ch, "a product message needs no medical approval")
}

func TestCooldownAndCaps(t *testing.T) {
	r := rule(1, rules.KindProduct, 30)
	r.CooldownDays, r.CapCount, r.CapPeriod = 7, 2, "week"
	h := fakeHistory{impressions: map[uint64][]time.Time{1: {noon.Add(-8 * 24 * time.Hour)}}}
	ok, err := rules.Eligible(&r, rules.Facts{}, 1, noon, h)
	require.NoError(t, err)
	assert.True(t, ok, "cooldown elapsed, cap not reached")

	h = fakeHistory{impressions: map[uint64][]time.Time{1: {noon.Add(-3 * 24 * time.Hour)}}}
	ok, _ = rules.Eligible(&r, rules.Facts{}, 1, noon, h)
	assert.False(t, ok, "inside the cooldown")

	r.CooldownDays = 0
	h = fakeHistory{impressions: map[uint64][]time.Time{1: {noon.Add(-3 * 24 * time.Hour), noon.Add(-2 * 24 * time.Hour)}}}
	ok, _ = rules.Eligible(&r, rules.Facts{}, 1, noon, h)
	assert.False(t, ok, "weekly cap of 2 reached")
	r.CapPeriod = "day"
	ok, _ = rules.Eligible(&r, rules.Facts{}, 1, noon, h)
	assert.True(t, ok, "the same events are outside the daily window")
}

func TestABStableAndSplit(t *testing.T) {
	r := rule(42, rules.KindProduct, 30)
	r.VariantsEnabled = true
	r.Content.B = &rules.Variant{Title: "tb", Body: "bb"}
	r.ABSplit = 50
	counts := map[string]int{}
	for u := uint64(1); u <= 2000; u++ {
		v := r.Assign(u)
		assert.Equal(t, v, r.Assign(u), "stable per user")
		counts[v]++
	}
	assert.InDelta(t, 1000, counts["a"], 150)
	r.ABSplit = 100
	// split is a percent of A; at 99 almost everyone is A and the assignment of a user that stays A is monotonic.
	r.VariantsEnabled = false
	assert.Equal(t, "a", r.Assign(5), "variants off → always A")
}

func TestAudienceMonotonic(t *testing.T) {
	r := rule(7, rules.KindProduct, 30)
	r.AudiencePercent = 10
	in10 := map[uint64]bool{}
	for u := uint64(1); u <= 1000; u++ {
		in10[u] = r.InAudience(u)
	}
	r.AudiencePercent = 50
	n := 0
	for u := uint64(1); u <= 1000; u++ {
		if in10[u] {
			assert.True(t, r.InAudience(u), "a user inside 10% stays inside at 50%")
		}
		if r.InAudience(u) {
			n++
		}
	}
	assert.InDelta(t, 500, n, 80)
}

func TestConditionsGroupsAndExceptions(t *testing.T) {
	r := rule(1, rules.KindClinical, 60)
	r.Groups = []rules.Group{
		{Op: "and", Items: []rules.Condition{
			{Fact: "mode", Op: "in", Value: []any{"cycle", "ttc"}},
			{Fact: "days_to_period", Op: "between", Value: []any{1.0, 4.0}},
		}},
		{Op: "or", Items: []rules.Condition{
			{Fact: "subphase", Op: "eq", Value: "late_luteal"},
			{Fact: "is_pms_window", Op: "eq", Value: true},
		}},
	}
	r.Exceptions = []rules.Condition{{Fact: "requires_user_input", Op: "eq", Value: true}}
	f := rules.Facts{"mode": "cycle", "days_to_period": 3.0, "subphase": "early_follicular", "is_pms_window": true}
	assert.True(t, rules.MatchRule(&r, f))
	f["is_pms_window"] = false
	assert.False(t, rules.MatchRule(&r, f), "neither OR branch holds")
	f["is_pms_window"], f["days_to_period"] = true, 9.0
	assert.False(t, rules.MatchRule(&r, f), "between fails")
	f["days_to_period"], f["requires_user_input"] = 3.0, true
	assert.False(t, rules.MatchRule(&r, f), "the exception removes the rule")
	assert.False(t, rules.Match(rules.Condition{Fact: "missing", Op: "gt", Value: 1.0}, f), "a missing fact matches nothing")
	assert.True(t, rules.Match(rules.Condition{Fact: "missing", Op: "ne", Value: 1.0}, f))
	assert.True(t, rules.Match(rules.Condition{Fact: "symptoms", Op: "contains", Value: "nausea"}, rules.Facts{"symptoms": []string{"nausea"}}))
}

func TestWindowAndHours(t *testing.T) {
	r := rule(1, rules.KindProduct, 30)
	r.HoursFrom, r.HoursTo = 8, 22
	r.StartsOn = civildate.MustParse("2026-10-01")
	assert.True(t, rules.InWindow(&r, noon))
	assert.False(t, rules.InWindow(&r, time.Date(2026, 10, 11, 23, 0, 0, 0, civildate.Tehran)))
	r.EndsOn = civildate.MustParse("2026-10-10")
	assert.False(t, rules.InWindow(&r, noon), "ended")
	r.EndsOn, r.HoursFrom, r.HoursTo = civildate.Date{}, 22, 6
	assert.True(t, rules.InWindow(&r, time.Date(2026, 10, 11, 1, 0, 0, 0, civildate.Tehran)), "overnight window wraps")
}

func TestValidate(t *testing.T) {
	cap := 1
	ok := rules.Input{Title: "x", Level: rules.LevelSuggestion, Kind: rules.KindProduct, Priority: 30, Placement: "smart_tip",
		Content: rules.Content{A: rules.Variant{Title: "{name} hi", Body: "b"}}, CapCount: &cap, CapPeriod: "day"}
	assert.Empty(t, ok.Validate())

	bad := ok
	bad.Priority = 500
	assert.Contains(t, bad.Validate(), "priority", "product outside 1-99")
	bad = ok
	bad.Placement = "urgent_modal"
	assert.Contains(t, bad.Validate(), "placement", "product never on urgent paths")
	bad = ok
	bad.Level = rules.LevelWorthFollowUp
	e := bad.Validate()
	assert.Contains(t, e, "content.seen")
	assert.Contains(t, e, "content.confidence")
	bad = ok
	bad.Level, bad.Kind, bad.Priority = rules.LevelFollowSoon, rules.KindProduct, 30
	assert.Contains(t, bad.Validate(), "level", "follow up soon is safety / clinical only")
	bad = ok
	bad.Content.A.Body = "{bogus}"
	assert.Contains(t, bad.Validate(), "content.a")
	bad = ok
	bad.CapCount = nil
	assert.Contains(t, bad.Validate(), "cap_count", "product needs a cap")
	bad = ok
	bad.VariantsEnabled = true
	assert.Contains(t, bad.Validate(), "content.b")

	safety := rules.Input{Title: "bp", Level: rules.LevelFollowSoon, Kind: rules.KindSafety, Priority: 1000, Placement: "urgent_modal",
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}, Seen: "s", Confidence: "c", Contact: "tel"}}
	assert.Empty(t, safety.Validate())
	assert.True(t, rules.NeedsMedicalApproval(safety.Level, safety.Kind))
	assert.False(t, rules.NeedsMedicalApproval(rules.LevelInfo, rules.KindProduct))
}

func TestConflictsAndRender(t *testing.T) {
	a := rule(1, rules.KindClinical, 60)
	a.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: "cycle"}}}}
	b := rule(2, rules.KindClinical, 55)
	b.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: "cycle"}}}}
	c := rule(3, rules.KindSafety, 1000)
	c.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "bp", Op: "gt", Value: 180.0}}}}
	d := rule(4, rules.KindClinical, 40)
	d.Placement = "push"
	cs := rules.FindConflicts(&a, []rules.Rule{a, b, c, d})
	require.Len(t, cs, 1, "same placement + shared fact only")
	assert.Equal(t, "this", cs[0].Winner)

	out, ok := rules.Render("{name}، {days_to_period} روز", rules.Facts{"name": "سارا", "days_to_period": 3.0})
	assert.True(t, ok)
	assert.Equal(t, "سارا، 3 روز", out)
	_, ok = rules.Render("{name}", rules.Facts{})
	assert.False(t, ok)
}

func TestSimulate_MatchesSelectAndExplains(t *testing.T) {
	now := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	mk := func(id uint64, kind string, prio int, mode string) rules.Rule {
		return rules.Rule{ID: id, Level: rules.LevelInfo, Kind: kind, Priority: prio, Placement: "smart_tip", IsActive: true, Status: rules.StatusApproved,
			Content: rules.Content{A: rules.Variant{Title: "t", Body: "b {name}"}}, HoursFrom: -1, HoursTo: -1, AudiencePercent: 100,
			Groups: []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: mode}}}}}
	}
	all := []rules.Rule{mk(1, rules.KindProduct, 50, "cycle"), mk(2, rules.KindClinical, 300, "cycle"), mk(3, rules.KindEngine, 900, "pregnancy"), mk(4, rules.KindSafety, 1000, "cycle")}
	all[3].Content.A.Body, all[2].Content.A.Body = "b", "b"
	all[1].Content.A.Body = "b {name}"
	inactive := mk(5, rules.KindProduct, 90, "cycle")
	inactive.IsActive = false
	all = append(all, inactive)

	for _, facts := range []rules.Facts{{"mode": "cycle", "name": "S"}, {"mode": "cycle"}, {"mode": "pregnancy", "name": "S"}, {"mode": "x"}} {
		vs, err := rules.Simulate(all, facts, 7, now, nil)
		require.NoError(t, err)
		// the live path drops rules whose text cannot render, then rules.Select
		var renderable []rules.Rule
		for i := range all {
			if _, ok := rules.RenderVariant(all[i].VariantContent(all[i].Assign(7)), facts); ok {
				renderable = append(renderable, all[i])
			}
		}
		ch, err := rules.Select(renderable, facts, 7, now, nil)
		require.NoError(t, err)
		var win uint64
		for _, v := range vs {
			if v.Winner {
				win = v.Rule.ID
			}
		}
		if ch == nil {
			assert.Zero(t, win)
		} else {
			assert.Equal(t, ch.Rule.ID, win, "facts %v", facts)
		}
	}
	vs, _ := rules.Simulate(all, rules.Facts{"mode": "cycle"}, 7, now, nil)
	byID := map[uint64]rules.Verdict{}
	for _, v := range vs {
		byID[v.Rule.ID] = v
	}
	assert.True(t, byID[4].Winner, "safety first")
	assert.Equal(t, rules.ReasonMissingVariable, byID[2].Reason)
	assert.Equal(t, rules.ReasonConditions, byID[3].Reason)
	assert.Equal(t, []int{0}, byID[3].FailedGroups)
	assert.Equal(t, rules.ReasonInactive, byID[5].Reason)
	assert.Equal(t, uint64(4), vs[0].Rule.ID, "ordered by priority")
}
