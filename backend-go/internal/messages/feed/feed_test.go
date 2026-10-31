package feed

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
)

// M-12: the product / marketing eligible sample is a stable per-user 10 %.
func TestSampled_StableTenPercent(t *testing.T) {
	in := 0
	for uid := uint64(1); uid <= 10000; uid++ {
		if Sampled(uid) {
			in++
		}
		assert.Equal(t, Sampled(uid), Sampled(uid))
	}
	assert.InDelta(t, 1000, in, 150)
}

// The events package keeps its own copies (no rules import on the write paths): they must stay equal.
func TestCatalogueCopies(t *testing.T) {
	assert.Equal(t, rules.GoalEvents, events.GoalEvents)
	assert.Equal(t, rules.EventShown, events.Shown)
	assert.Equal(t, rules.EventDismissed, events.Dismissed)
	for _, s := range DefaultSlots {
		_, ok := rules.Slots[s]
		assert.True(t, ok, s)
	}
}

// Without Redis, the day's history rows decide: a shown row today in that slot means "already logged"; a second
// candidate of the same key in one build is not first either.
func TestDBFirst(t *testing.T) {
	r1, r2 := &rules.Rule{ID: 1}, &rules.Rule{ID: 2}
	rows := []store.ListUserMessageEventStatsRow{
		{RuleID: 1, Event: events.Shown, Slot: "smart_tip", DayCount: 1},
		{RuleID: 2, Event: events.Shown, Slot: "smart_tip", DayCount: 0, WeekCount: 1},
	}
	got := dbFirst(rows, []candidate{
		{slot: "smart_tip", ch: rules.Choice{Rule: r1}, event: events.Shown},
		{slot: "daily_advice", ch: rules.Choice{Rule: r1}, event: events.Shown},
		{slot: "smart_tip", ch: rules.Choice{Rule: r2}, event: events.Shown},
		{slot: "smart_tip", ch: rules.Choice{Rule: r2}, event: events.Shown},
		{slot: "smart_tip", ch: rules.Choice{Rule: r1}, event: events.Eligible},
	})
	assert.Equal(t, []bool{false, true, true, false, true}, got)
}

func TestServedVersion(t *testing.T) {
	assert.Equal(t, 3, ServedVersion(&rules.Rule{Kind: rules.KindClinical, Level: rules.LevelInfo, Version: 4, ApprovedVersion: 3}))
	assert.Equal(t, 4, ServedVersion(&rules.Rule{Kind: rules.KindProduct, Level: rules.LevelInfo, Version: 4, ApprovedVersion: 3}))
	assert.Equal(t, 4, ServedVersion(&rules.Rule{Kind: rules.KindClinical, Level: rules.LevelInfo, Version: 4}))
}

// eligible: the top K; product only for sampled users; shown for every winner.
func TestAppendLog(t *testing.T) {
	p, c := &rules.Rule{ID: 1, Kind: rules.KindProduct}, &rules.Rule{ID: 2, Kind: rules.KindContent}
	winners := []rules.Choice{{Rule: p}}
	top := []rules.Choice{{Rule: p}, {Rule: c}}
	out := appendLog(nil, "daily_advice", winners, top, false)
	assert.Equal(t, []string{events.Shown, events.Eligible}, []string{out[0].event, out[1].event})
	assert.EqualValues(t, 2, out[1].ch.Rule.ID)
	assert.Len(t, appendLog(nil, "daily_advice", winners, top, true), 3)
}
