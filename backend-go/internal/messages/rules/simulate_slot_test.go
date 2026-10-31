package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M10-11: SimulateSlot = the feed's winners plus every dropped candidate with its reason.
func TestSimulateSlot_ReasonsAndRenderability(t *testing.T) {
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, civildate.Tehran)
	tip := func(id uint64, prio int, body string, gs ...rules.Group) rules.Rule {
		r := ruleWith(id, rules.KindClinical, prio, "smart_tip", gs...)
		r.Content.A.Body = body
		return r
	}
	all := []rules.Rule{
		tip(1, 700, "needs {days_late}"),                       // would win, cannot render → missing_variable
		tip(2, 600, "b", and(cond("mode", "eq", "pregnancy"))), // conditions
		tip(3, 500, "b"), // capped by its history
		tip(4, 400, "b"), // winner
		tip(5, 300, "b"), // capacity
		ruleWith(6, rules.KindProduct, 50, "smart_tip"), // kind_not_allowed
		ruleWith(7, rules.KindClinical, 900, "push"),    // another slot: not a candidate
	}
	all[2].Caps = []rules.Cap{{Count: 1, Period: "week"}}
	f := rules.Facts{"mode": "cycle"}
	h := &rules.UserHistory{Rules: map[uint64]rules.RuleHistory{3: {Shown: rules.WindowCounts{Week: 1, Cycle: 1}}}}

	winners, dropped := rules.SimulateSlot("smart_tip", all, f, h, 42, now)
	require.Len(t, winners, 1)
	assert.Equal(t, uint64(4), winners[0].Rule.ID)
	reasons := map[uint64]string{}
	for _, d := range dropped {
		reasons[d.RuleID] = d.Reason
	}
	assert.Equal(t, map[uint64]string{1: rules.ReasonMissingVariable, 2: rules.ReasonConditions, 3: rules.ReasonCapWeek,
		5: rules.ReasonCapacity, 6: rules.ReasonKindNotAllowed}, reasons)

	// With the variable filled the rule wins and nothing else changes.
	f["days_late"] = 3.0
	winners, dropped = rules.SimulateSlot("smart_tip", all, f, h, 42, now)
	require.Len(t, winners, 1)
	assert.Equal(t, uint64(1), winners[0].Rule.ID)
	assert.Len(t, dropped, 5)

	// Same winners as SelectSlot over the renderable rules, without history too.
	delete(f, "days_late")
	sw, _ := rules.SimulateSlot("smart_tip", all, f, nil, 42, now)
	ref, _ := rules.SelectSlot("smart_tip", all[1:], f, nil, 42, now)
	require.Len(t, sw, len(ref))
	for i := range sw {
		assert.Equal(t, ref[i].Rule.ID, sw[i].Rule.ID)
		assert.Equal(t, ref[i].Variant, sw[i].Variant)
	}
}
