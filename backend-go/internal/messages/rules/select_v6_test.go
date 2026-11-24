package rules_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Selection v6 (T-M10-04, SPEC §5.4 / §5.4.1 / §6.1, DECISIONS.md M-04…M-09, D-162).

var (
	berlin = mustZone("Europe/Berlin")
	// tNow is 12:00 in Tehran on 2026-10-21 (a Wednesday).
	tNow = time.Date(2026, 10, 21, 12, 0, 0, 0, civildate.Tehran)
)

func mustZone(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// slotRule is a served rule of kind in slot with a band priority.
func slotRule(id uint64, slot, kind string, prio int) rules.Rule {
	r := rule(id, kind, prio)
	r.Placement = slot
	return r
}

// hist is a pure history builder: shown / dismissed events per rule (slot = the rule's placement, kind as given).
type hist struct {
	rows []event
}

type event struct {
	rule  uint64
	event string
	slot  string
	kind  string
	at    time.Time
}

func (h *hist) shown(r rules.Rule, at ...time.Time) *hist {
	for _, t := range at {
		h.rows = append(h.rows, event{r.ID, "impression", r.Placement, r.Kind, t})
	}
	return h
}

func (h *hist) dismissed(r rules.Rule, at ...time.Time) *hist {
	for _, t := range at {
		h.rows = append(h.rows, event{r.ID, "dismiss", r.Placement, r.Kind, t})
	}
	return h
}

// build groups the events like ListUserMessageEventStats does for the windows of (now, facts).
func (h *hist) build(now time.Time, f rules.Facts) *rules.UserHistory {
	w := rules.CapWindows(now, f)
	type key struct {
		rule              uint64
		event, slot, kind string
	}
	agg := map[key]*store.ListUserMessageEventStatsRow{}
	var order []key
	for _, e := range h.rows {
		k := key{e.rule, e.event, e.slot, e.kind}
		r := agg[k]
		if r == nil {
			r = &store.ListUserMessageEventStatsRow{RuleID: e.rule, Event: e.event, Slot: e.slot, Kind: e.kind}
			agg[k] = r
			order = append(order, k)
		}
		if e.at.After(r.LastAt) {
			r.LastAt = e.at
		}
		if !e.at.Before(w.Day) {
			r.DayCount++
		}
		if !e.at.Before(w.Week) {
			r.WeekCount++
		}
		if !e.at.Before(w.Cycle) {
			r.CycleCount++
		}
	}
	rows := make([]store.ListUserMessageEventStatsRow, 0, len(order))
	for _, k := range order {
		rows = append(rows, *agg[k])
	}
	return rules.BuildHistory(w, rows)
}

func ids(cs []rules.Choice) []uint64 {
	out := make([]uint64, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Rule.ID)
	}
	return out
}

func reasonOf(t *testing.T, dropped []rules.Dropped, id uint64) string {
	t.Helper()
	for _, d := range dropped {
		if d.RuleID == id {
			return d.Reason
		}
	}
	t.Fatalf("rule %d was not dropped (dropped: %v)", id, dropped)
	return ""
}

func TestSlots_Table(t *testing.T) {
	for name, want := range map[string]struct {
		capacity int
		list     bool
		kinds    []string
	}{
		"urgent_modal":         {1, false, []string{rules.KindSafety}},
		"status_banner":        {1, false, []string{rules.KindEngine}},
		"smart_tip":            {1, false, []string{rules.KindClinical}},
		"daily_advice":         {3, false, []string{rules.KindContent}},
		"daily_challenge":      {1, false, []string{rules.KindProduct}},
		"push":                 {1, false, rules.Kinds},
		"modal":                {1, false, []string{rules.KindMarketing}},
		rules.SlotAlertsInbox:  {0, true, []string{rules.KindSafety, rules.KindEngine, rules.KindClinical, rules.KindContent, rules.KindProduct}},
		rules.SlotMemberAlert:  {1, false, nil},
		rules.SlotMemberTip:    {1, false, nil},
		"sms":                  {1, false, rules.Kinds},
		"alerts" /* legacy */ : {0, true, []string{rules.KindSafety, rules.KindEngine, rules.KindClinical, rules.KindContent, rules.KindProduct}},
	} {
		s, ok := rules.Slots[name]
		require.True(t, ok, name)
		assert.Equal(t, name, s.Name)
		assert.Equal(t, want.capacity, s.Capacity, name)
		assert.Equal(t, want.list, s.List, name)
		assert.ElementsMatch(t, want.kinds, s.Kinds, name)
	}
	assert.True(t, rules.Slots[rules.SlotMemberAlert].Reserved)
	assert.Empty(t, rules.Slots["smart_tip"].MessageCaps)
	assert.Equal(t, []rules.Cap{{Count: 1, Period: "week"}}, rules.Slots["daily_advice"].MessageCaps)
	assert.Equal(t, []rules.Cap{{Count: 1, Period: "day"}, {Count: 3, Period: "week"}}, rules.Slots["push"].SlotCaps)
	assert.Equal(t, []rules.Cap{{Count: 1, Period: "week"}}, rules.Slots["modal"].SlotCaps)
	assert.Equal(t, &rules.HourWindow{23, 8}, rules.Slots["push"].QuietHours)
	assert.Len(t, rules.SelectionReasons, 17)
}

// SPEC §6.1: safety beats any priority, even a priority-999 engine message with a smaller id.
func TestSelectSlot_SafetyFirst(t *testing.T) {
	engine := slotRule(1, rules.SlotAlertsInbox, rules.KindEngine, 999)
	safety := slotRule(2, rules.SlotAlertsInbox, rules.KindSafety, 1000)
	safety.Priority = 1 // even a mis-banded safety row stays first
	got, dropped := rules.SelectSlot(rules.SlotAlertsInbox, []rules.Rule{engine, safety}, rules.Facts{}, nil, 7, tNow)
	assert.Equal(t, []uint64{2, 1}, ids(got))
	assert.Empty(t, dropped)
}

// SPEC §6.1 / §5.4.1: requires_user_input keeps safety and engine and hides clinical and product.
func TestSelectSlot_RequiresUserInput(t *testing.T) {
	all := []rules.Rule{
		slotRule(1, rules.SlotAlertsInbox, rules.KindSafety, 1000),
		slotRule(2, rules.SlotAlertsInbox, rules.KindEngine, 900),
		slotRule(3, rules.SlotAlertsInbox, rules.KindClinical, 500),
		slotRule(4, rules.SlotAlertsInbox, rules.KindProduct, 50),
		slotRule(5, rules.SlotAlertsInbox, rules.KindContent, 300),
	}
	got, dropped := rules.SelectSlot(rules.SlotAlertsInbox, all, rules.Facts{"requires_user_input": true}, nil, 7, tNow)
	assert.Equal(t, []uint64{1, 2}, ids(got))
	for _, id := range []uint64{3, 4, 5} {
		assert.Equal(t, rules.ReasonRequiresUserInput, reasonOf(t, dropped, id))
	}
	got, _ = rules.SelectSlot(rules.SlotAlertsInbox, all, rules.Facts{"requires_user_input": false}, nil, 7, tNow)
	assert.Len(t, got, 5)
}

// SPEC §6.1: the cooldown counts from the last dismissal too.
func TestSelectSlot_CooldownFromDismissed(t *testing.T) {
	r := slotRule(1, "smart_tip", rules.KindClinical, 300)
	r.CooldownDays = 3
	h := (&hist{}).shown(r, tNow.Add(-10*24*time.Hour)).dismissed(r, tNow.Add(-2*24*time.Hour))
	got, dropped := rules.SelectSlot("daily_challenge", nil, nil, nil, 1, tNow)
	assert.Empty(t, got)
	assert.Empty(t, dropped)
	// A fresh cycle, so nothing but the cooldown can bite.
	f := rules.Facts{"current_period_start": tNow.AddDate(0, 0, -1).Format("2006-01-02")}
	got, dropped = rules.SelectSlot("smart_tip", []rules.Rule{r}, f, h.build(tNow, f), 1, tNow)
	assert.Empty(t, got)
	assert.Equal(t, rules.ReasonCooldown, reasonOf(t, dropped, 1))
}

func TestSelectSlot_CooldownBoundary(t *testing.T) {
	r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	r.CooldownDays = 2
	last := tNow.Add(-48 * time.Hour)
	for _, tc := range []struct {
		now  time.Time
		want string
	}{{last.Add(48*time.Hour - time.Second), rules.ReasonCooldown}, {last.Add(48 * time.Hour), ""}} {
		for _, ev := range []string{"impression", "dismiss"} {
			h := &hist{rows: []event{{1, ev, "daily_challenge", rules.KindProduct, last}}}
			got, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, h.build(tc.now, nil), 1, tc.now)
			if tc.want == "" {
				assert.Len(t, got, 1, "%s at %s", ev, tc.now)
			} else {
				assert.Equal(t, tc.want, reasonOf(t, dropped, 1), ev)
			}
		}
	}
}

// SPEC §6.1: stable A/B — the same user always gets the same variant, and the pick is Assign.
func TestSelectSlot_StableVariant(t *testing.T) {
	r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	r.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "b", Weight: 1}, {Key: "c", Weight: 2}}
	seen := map[string]bool{}
	for uid := uint64(1); uid <= 200; uid++ {
		got, _ := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, nil, uid, tNow)
		require.Len(t, got, 1)
		again, _ := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, nil, uid, tNow.Add(time.Hour))
		assert.Equal(t, got[0].Variant, again[0].Variant)
		assert.Equal(t, r.Assign(uid), got[0].Variant)
		seen[got[0].Variant] = true
	}
	assert.Len(t, seen, 3)
}

// Safety is exempt from audience and A/B (M-07).
func TestSelectSlot_SafetyExemptFromAudienceAndAB(t *testing.T) {
	s := slotRule(1, "urgent_modal", rules.KindSafety, 1000)
	s.AudiencePercent = 0
	s.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "b", Weight: 99}}
	for uid := uint64(1); uid <= 50; uid++ {
		got, _ := rules.SelectSlot("urgent_modal", []rules.Rule{s}, rules.Facts{}, nil, uid, tNow)
		require.Len(t, got, 1)
		assert.Equal(t, "a", got[0].Variant)
	}
	p := slotRule(2, "daily_challenge", rules.KindProduct, 50)
	p.AudiencePercent = 0
	_, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{p}, rules.Facts{}, nil, 1, tNow)
	assert.Equal(t, rules.ReasonAudience, reasonOf(t, dropped, 2))
}

// A rule cap at N−1, N and N+1 shows: allowed, blocked, blocked. Every period, counted from shown or dismissed.
func TestSelectSlot_CapBoundaries(t *testing.T) {
	f := rules.Facts{"current_period_start": "2026-10-01"}
	for _, tc := range []struct {
		period, reason string
		gap            time.Duration
	}{
		{"day", rules.ReasonCapDay, time.Hour},
		{"week", rules.ReasonCapWeek, 24 * time.Hour},
		{"cycle", rules.ReasonCapCycle, 2 * 24 * time.Hour},
	} {
		for _, ev := range []string{"impression", "dismiss"} {
			for shows, want := range map[int]string{2: "", 3: tc.reason, 4: tc.reason} {
				r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
				r.Caps = []rules.Cap{{Count: 3, Period: tc.period}}
				h := &hist{}
				for i := range shows {
					h.rows = append(h.rows, event{1, ev, "daily_challenge", rules.KindProduct,
						tNow.Add(-time.Minute - time.Duration(i)*tc.gap/time.Duration(max(shows, 1)))})
				}
				got, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{r}, f, h.build(tNow, f), 1, tNow)
				if want == "" {
					assert.Len(t, got, 1, "%s %s %d", tc.period, ev, shows)
				} else {
					assert.Equal(t, want, reasonOf(t, dropped, 1), "%s %s %d", tc.period, ev, shows)
				}
			}
		}
	}
}

// The legacy single cap still applies (EffectiveCaps), and a day cap is the user-local day, not 24 h.
func TestSelectSlot_DayCapIsTheLocalDay(t *testing.T) {
	r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	r.CapCount, r.CapPeriod = 1, "day"
	now := time.Date(2026, 10, 21, 0, 30, 0, 0, civildate.Tehran)
	h := (&hist{}).shown(r, now.Add(-time.Hour)) // 23:30 yesterday
	got, _ := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, h.build(now, nil), 1, now)
	assert.Len(t, got, 1, "yesterday's show does not count for today's cap")
	h = (&hist{}).shown(r, now.Add(-10*time.Minute))
	_, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, h.build(now, nil), 1, now)
	assert.Equal(t, rules.ReasonCapDay, reasonOf(t, dropped, 1))
}

// M-04: a cycle cap counts since the engine's current_period_start; a show before the new period does not count.
func TestSelectSlot_CycleCapAcrossARealPeriodStart(t *testing.T) {
	r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	r.Caps = []rules.Cap{{Count: 1, Period: "cycle"}}
	shownAt := time.Date(2026, 10, 19, 20, 0, 0, 0, civildate.Tehran)
	h := (&hist{}).shown(r, shownAt)

	old := rules.Facts{"current_period_start": "2026-09-25", "cycle_length": 28.0}
	_, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{r}, old, h.build(tNow, old), 1, tNow)
	assert.Equal(t, rules.ReasonCapCycle, reasonOf(t, dropped, 1), "same cycle")

	fresh := rules.Facts{"current_period_start": "2026-10-20", "cycle_length": 28.0}
	got, _ := rules.SelectSlot("daily_challenge", []rules.Rule{r}, fresh, h.build(tNow, fresh), 1, tNow)
	assert.Len(t, got, 1, "a new period started after the show")

	// No anchor: a rolling effective cycle length.
	w := rules.CapWindows(tNow, rules.Facts{"cycle_length": 30.0})
	assert.Equal(t, tNow.Add(-30*24*time.Hour), w.Cycle)
	w = rules.CapWindows(tNow, rules.Facts{})
	assert.Equal(t, tNow.Add(-28*24*time.Hour), w.Cycle)
	w = rules.CapWindows(tNow, fresh)
	assert.Equal(t, time.Date(2026, 10, 20, 0, 0, 0, 0, civildate.Tehran), w.Cycle)
}

func TestCapWindows_UserLocal(t *testing.T) {
	now := time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC).In(berlin) // 03:00 Berlin
	w := rules.CapWindows(now, rules.Facts{"current_period_start": "2026-10-10"})
	assert.True(t, w.Day.Equal(time.Date(2026, 10, 21, 0, 0, 0, 0, berlin)))
	assert.True(t, w.Week.Equal(time.Date(2026, 10, 15, 0, 0, 0, 0, berlin)), "the last 7 local days")
	assert.True(t, w.Cycle.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, berlin)))
}

// Push quiet hours 23:00–08:00 user-local, with the server clock in UTC.
func TestSelectSlot_PushQuietHours(t *testing.T) {
	r := slotRule(1, "push", rules.KindContent, 300)
	s := slotRule(2, "push", rules.KindSafety, 1000)
	for _, tc := range []struct {
		utc     time.Time
		loc     *time.Location
		blocked bool
	}{
		// Tehran = UTC+3:30. UTC 20:30 = 00:00 Tehran.
		{time.Date(2026, 10, 21, 19, 29, 0, 0, time.UTC), civildate.Tehran, false}, // 22:59
		{time.Date(2026, 10, 21, 19, 30, 0, 0, time.UTC), civildate.Tehran, true},  // 23:00
		{time.Date(2026, 10, 21, 20, 30, 0, 0, time.UTC), civildate.Tehran, true},  // 00:00
		{time.Date(2026, 10, 21, 4, 29, 0, 0, time.UTC), civildate.Tehran, true},   // 07:59
		{time.Date(2026, 10, 21, 4, 30, 0, 0, time.UTC), civildate.Tehran, false},  // 08:00
		// Berlin on 2026-10-21 = UTC+2 (CEST).
		{time.Date(2026, 10, 21, 20, 59, 0, 0, time.UTC), berlin, false}, // 22:59
		{time.Date(2026, 10, 21, 21, 0, 0, 0, time.UTC), berlin, true},   // 23:00
		{time.Date(2026, 10, 21, 5, 59, 0, 0, time.UTC), berlin, true},   // 07:59
		{time.Date(2026, 10, 21, 6, 0, 0, 0, time.UTC), berlin, false},   // 08:00
	} {
		now := tc.utc.In(tc.loc)
		got, dropped := rules.SelectSlot("push", []rules.Rule{r, s}, rules.Facts{}, nil, 1, now)
		if tc.blocked {
			assert.Empty(t, got, now.String())
			assert.Equal(t, rules.ReasonQuietHours, reasonOf(t, dropped, 1))
			assert.Equal(t, rules.ReasonQuietHours, reasonOf(t, dropped, 2), "nothing is pushed in quiet hours")
		} else {
			assert.Equal(t, []uint64{2}, ids(got), now.String())
			assert.Equal(t, rules.ReasonCapacity, reasonOf(t, dropped, 1))
		}
	}
	// Quiet hours are push only.
	c := slotRule(3, "daily_advice", rules.KindContent, 300)
	got, _ := rules.SelectSlot("daily_advice", []rules.Rule{c}, rules.Facts{}, nil, 1, time.Date(2026, 10, 21, 20, 30, 0, 0, time.UTC).In(civildate.Tehran))
	assert.Len(t, got, 1)
}

// Push: 1 per local day and 3 per 7 local days across all push messages; safety is not capped.
func TestSelectSlot_PushSlotCaps(t *testing.T) {
	a := slotRule(1, "push", rules.KindContent, 300)
	b := slotRule(2, "push", rules.KindProduct, 50)
	b.Caps = []rules.Cap{{Count: 5, Period: "day"}}
	other := slotRule(9, "push", rules.KindMarketing, 10) // not a candidate any more (inactive) but its pushes count
	h := (&hist{}).shown(other, tNow.Add(-time.Hour))
	got, dropped := rules.SelectSlot("push", []rules.Rule{a, b}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Empty(t, got)
	assert.Equal(t, rules.ReasonSlotCap, reasonOf(t, dropped, 1))
	assert.Equal(t, rules.ReasonSlotCap, reasonOf(t, dropped, 2))

	h = (&hist{}).shown(other, tNow.Add(-25*time.Hour), tNow.Add(-50*time.Hour))
	got, _ = rules.SelectSlot("push", []rules.Rule{a, b}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Equal(t, []uint64{1}, ids(got), "2 this week: one more allowed")
	h.shown(other, tNow.Add(-75*time.Hour))
	_, dropped = rules.SelectSlot("push", []rules.Rule{a, b}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Equal(t, rules.ReasonSlotCap, reasonOf(t, dropped, 1), "3 in the last 7 local days")

	s := slotRule(3, "push", rules.KindSafety, 1000)
	got, _ = rules.SelectSlot("push", []rules.Rule{a, s}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Equal(t, []uint64{3}, ids(got))
}

// Modal: 1 per 7 local days across all modal messages.
func TestSelectSlot_ModalWeekly(t *testing.T) {
	m := slotRule(1, "modal", rules.KindMarketing, 10)
	m2 := slotRule(2, "modal", rules.KindMarketing, 20)
	h := (&hist{}).shown(m2, tNow.Add(-6*24*time.Hour))
	_, dropped := rules.SelectSlot("modal", []rules.Rule{m}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Equal(t, rules.ReasonSlotCap, reasonOf(t, dropped, 1))
	h = (&hist{}).shown(m2, tNow.Add(-7*24*time.Hour))
	got, _ := rules.SelectSlot("modal", []rules.Rule{m}, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Len(t, got, 1)
}

// Product and marketing are blocked for 24 h after any safety message was shown.
func TestSelectSlot_PostSafety24h(t *testing.T) {
	p := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	s := slotRule(77, "urgent_modal", rules.KindSafety, 1000)
	shown := tNow.Add(-2 * time.Hour)
	for _, tc := range []struct {
		after   time.Duration
		blocked bool
	}{{24*time.Hour - time.Second, true}, {24 * time.Hour, false}} {
		now := shown.Add(tc.after)
		h := (&hist{}).shown(s, shown).build(now, nil)
		got, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{p}, rules.Facts{}, h, 1, now)
		if tc.blocked {
			assert.Equal(t, rules.ReasonPostSafety, reasonOf(t, dropped, 1))
		} else {
			assert.Len(t, got, 1)
		}
		m := slotRule(2, "modal", rules.KindMarketing, 10)
		got, dropped = rules.SelectSlot("modal", []rules.Rule{m}, rules.Facts{}, h, 1, now)
		if tc.blocked {
			assert.Equal(t, rules.ReasonPostSafety, reasonOf(t, dropped, 2))
		} else {
			assert.Len(t, got, 1)
		}
		c := slotRule(3, "daily_advice", rules.KindContent, 300)
		got, _ = rules.SelectSlot("daily_advice", []rules.Rule{c}, rules.Facts{}, h, 1, now)
		assert.Len(t, got, 1, "content is not blocked")
	}
}

// Ordering: priority, then least recently shown (never shown first), then the smaller id.
func TestSelectSlot_LeastRecentlyShownThenID(t *testing.T) {
	f := rules.Facts{"current_period_start": tNow.Format("2006-01-02")} // a fresh cycle
	a := slotRule(5, "smart_tip", rules.KindClinical, 300)
	b := slotRule(3, "smart_tip", rules.KindClinical, 300)
	c := slotRule(4, "smart_tip", rules.KindClinical, 300)
	h := (&hist{}).shown(b, tNow.Add(-24*time.Hour)).shown(c, tNow.Add(-48*time.Hour)).shown(a, tNow.Add(-72*time.Hour))
	// shows before the cycle start: they order but do not cap.
	got, dropped := rules.SelectSlot("smart_tip", []rules.Rule{b, c, a}, f, h.build(tNow, f), 1, tNow)
	assert.Equal(t, []uint64{5}, ids(got), "the least recently shown wins the tie")
	assert.Equal(t, rules.ReasonCapacity, reasonOf(t, dropped, 3))

	never := slotRule(9, "smart_tip", rules.KindClinical, 300)
	got, _ = rules.SelectSlot("smart_tip", []rules.Rule{b, c, a, never}, f, h.build(tNow, f), 1, tNow)
	assert.Equal(t, []uint64{9}, ids(got), "never shown comes first")

	// Equal history → the smaller id.
	x, y := slotRule(8, "smart_tip", rules.KindClinical, 300), slotRule(6, "smart_tip", rules.KindClinical, 300)
	got, _ = rules.SelectSlot("smart_tip", []rules.Rule{x, y}, f, nil, 1, tNow)
	assert.Equal(t, []uint64{6}, ids(got))

	// Priority beats recency.
	hi := slotRule(10, "smart_tip", rules.KindClinical, 301)
	h.shown(hi, tNow.Add(-time.Hour*25))
	got, _ = rules.SelectSlot("smart_tip", []rules.Rule{never, hi}, f, h.build(tNow, f), 1, tNow)
	assert.Equal(t, []uint64{10}, ids(got))
}

// daily_advice returns 3 distinct messages, none shown in the last 7 local days.
func TestSelectSlot_DailyAdviceThreeFresh(t *testing.T) {
	var all []rules.Rule
	for i := range 8 {
		all = append(all, slotRule(uint64(i+1), "daily_advice", rules.KindContent, 300+i)) //nolint:gosec // small
	}
	h := &hist{}
	h.shown(all[7], tNow.Add(-time.Hour))          // today
	h.shown(all[6], tNow.Add(-6*24*time.Hour))     // within the last 7 local days
	h.shown(all[5], tNow.Add(-7*24*time.Hour))     // 7 days ago = outside the last 7 local days
	h.dismissed(all[4], tNow.Add(-3*24*time.Hour)) // dismissed counts like shown
	got, dropped := rules.SelectSlot("daily_advice", all, rules.Facts{}, h.build(tNow, nil), 1, tNow)
	assert.Equal(t, []uint64{6, 4, 3}, ids(got))
	for _, id := range []uint64{8, 7, 5} {
		assert.Equal(t, rules.ReasonSlotCap, reasonOf(t, dropped, id))
	}
	assert.Equal(t, rules.ReasonCapacity, reasonOf(t, dropped, 2))
	assert.Equal(t, rules.ReasonCapacity, reasonOf(t, dropped, 1))
}

// smart_tip has no per-cycle default: a tip already shown this cycle is served again while it still matches.
func TestSelectSlot_SmartTipRepeatsWithinCycle(t *testing.T) {
	r := slotRule(1, "smart_tip", rules.KindClinical, 300)
	f := rules.Facts{"current_period_start": "2026-10-05"}
	h := (&hist{}).shown(r, time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran))
	got, _ := rules.SelectSlot("smart_tip", []rules.Rule{r}, f, h.build(tNow, f), 1, tNow)
	assert.Equal(t, []uint64{1}, ids(got))
}

func TestSelectSlot_HormonalContraceptionDropsFertileWindow(t *testing.T) {
	tagged := slotRule(1, "smart_tip", rules.KindClinical, 300)
	tagged.Topics = []string{rules.TopicFertileWindow}
	untaggedButFertile := slotRule(2, "smart_tip", rules.KindClinical, 300)
	untaggedButFertile.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "cycle_day", Op: "gte", Value: 1.0}, {Fact: "days_to_ovulation", Op: "gte", Value: -99.0}}}}
	plain := slotRule(3, "smart_tip", rules.KindClinical, 300)
	all := []rules.Rule{tagged, untaggedButFertile, plain}
	f := rules.Facts{"hormonal_contraception": true, "cycle_day": 3.0, "days_to_ovulation": 9.0}
	got, dropped := rules.SelectSlot("smart_tip", all, f, nil, 1, tNow)
	assert.Equal(t, []uint64{3}, ids(got))
	assert.Equal(t, rules.ReasonContraception, reasonOf(t, dropped, 1))
	assert.Equal(t, rules.ReasonContraception, reasonOf(t, dropped, 2))

	f["hormonal_contraception"] = false
	got, _ = rules.SelectSlot("smart_tip", all, f, nil, 1, tNow)
	assert.Equal(t, []uint64{1}, ids(got))
}

// M-05: confidence low hides clinical messages unless the rule tests confidence.
func TestSelectSlot_LowConfidence(t *testing.T) {
	c := slotRule(1, rules.SlotAlertsInbox, rules.KindClinical, 300)
	allowed := slotRule(2, rules.SlotAlertsInbox, rules.KindClinical, 300)
	allowed.Groups = []rules.Group{{Op: "or", Items: []rules.Condition{{Fact: "confidence", Op: "in", Value: []any{"low", "medium"}}}}}
	e := slotRule(3, rules.SlotAlertsInbox, rules.KindEngine, 900)
	ct := slotRule(4, rules.SlotAlertsInbox, rules.KindContent, 300)
	got, dropped := rules.SelectSlot(rules.SlotAlertsInbox, []rules.Rule{c, allowed, e, ct}, rules.Facts{"confidence": "low"}, nil, 1, tNow)
	assert.Equal(t, []uint64{3, 2, 4}, ids(got))
	assert.Equal(t, rules.ReasonLowConfidence, reasonOf(t, dropped, 1))
	got, _ = rules.SelectSlot(rules.SlotAlertsInbox, []rules.Rule{c, e}, rules.Facts{"confidence": "high"}, nil, 1, tNow)
	assert.Len(t, got, 2)
}

// M-18: a slot/kind mismatch the migration did not convert is dropped with kind_not_allowed.
func TestSelectSlot_KindNotAllowed(t *testing.T) {
	p := slotRule(1, "smart_tip", rules.KindProduct, 50)
	c := slotRule(2, "smart_tip", rules.KindClinical, 300)
	other := slotRule(3, "push", rules.KindProduct, 50) // another slot: not a candidate
	got, dropped := rules.SelectSlot("smart_tip", []rules.Rule{p, c, other}, rules.Facts{}, nil, 1, tNow)
	assert.Equal(t, []uint64{2}, ids(got))
	assert.Equal(t, rules.ReasonKindNotAllowed, reasonOf(t, dropped, 1))
	assert.Len(t, dropped, 1)

	m := slotRule(4, rules.SlotAlertsInbox, rules.KindMarketing, 10)
	_, dropped = rules.SelectSlot(rules.SlotAlertsInbox, []rules.Rule{m}, rules.Facts{}, nil, 1, tNow)
	assert.Equal(t, rules.ReasonKindNotAllowed, reasonOf(t, dropped, 4))

	// member slots are reserved for M11: nothing is served there yet.
	mt := slotRule(5, rules.SlotMemberTip, rules.KindContent, 300)
	got, dropped = rules.SelectSlot(rules.SlotMemberTip, []rules.Rule{mt}, rules.Facts{}, nil, 1, tNow)
	assert.Empty(t, got)
	assert.Equal(t, rules.ReasonKindNotAllowed, reasonOf(t, dropped, 5))
}

// The pipeline's order: each dropped candidate carries the first gate that stopped it.
func TestSelectSlot_PipelineReasons(t *testing.T) {
	mk := func(id uint64) rules.Rule { return slotRule(id, "daily_challenge", rules.KindProduct, 50) }
	sched := mk(1)
	sched.StartsAt = tNow.Add(time.Hour)
	ended := mk(2)
	ended.EndsAt = tNow
	hours := mk(3)
	hours.HourWindows = []rules.HourWindow{{18, 22}}
	cond := mk(4)
	cond.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: "pregnancy"}}}}
	excl := mk(5)
	excl.Exceptions = []rules.Condition{{Fact: "mode", Op: "eq", Value: "cycle"}}
	aud := mk(6)
	aud.AudiencePercent = 0
	inv := mk(7)
	inv.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: "pregnancy"}}}}
	inv.AudiencePercent = 0 // conditions are checked before audience
	off := mk(8)
	off.IsActive = false
	ok := mk(9)
	f := rules.Facts{"mode": "cycle"}
	got, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{sched, ended, hours, cond, excl, aud, inv, off, ok}, f, nil, 1, tNow)
	assert.Equal(t, []uint64{9}, ids(got))
	for id, want := range map[uint64]string{1: rules.ReasonSchedule, 2: rules.ReasonSchedule, 3: rules.ReasonHours,
		4: rules.ReasonConditions, 5: rules.ReasonExclude, 6: rules.ReasonAudience, 7: rules.ReasonConditions, 8: rules.ReasonInactive} {
		assert.Equal(t, want, reasonOf(t, dropped, id), "rule %d", id)
	}
	for _, d := range dropped {
		if d.Reason != rules.ReasonInactive {
			assert.Contains(t, rules.SelectionReasons, d.Reason)
		}
	}
}

// The schedule and the hour windows read the user-local clock (M-08).
func TestSelectSlot_ScheduleAndHoursUserLocal(t *testing.T) {
	r := slotRule(1, "daily_challenge", rules.KindProduct, 50)
	r.HourWindows = []rules.HourWindow{{8, 12}}
	r.StartsAt = time.Date(2026, 10, 21, 9, 0, 0, 0, civildate.Tehran) // wall clock 09:00, read in the user's zone
	nowBerlin := time.Date(2026, 10, 21, 9, 30, 0, 0, berlin)          // 11:00 in Tehran
	got, _ := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, nil, 1, nowBerlin)
	assert.Len(t, got, 1)
	early := time.Date(2026, 10, 21, 8, 30, 0, 0, berlin) // after 09:00 Tehran, but before 09:00 Berlin
	_, dropped := rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, nil, 1, early)
	assert.Equal(t, rules.ReasonSchedule, reasonOf(t, dropped, 1))
	late := time.Date(2026, 10, 21, 12, 0, 0, 0, berlin)
	_, dropped = rules.SelectSlot("daily_challenge", []rules.Rule{r}, rules.Facts{}, nil, 1, late)
	assert.Equal(t, rules.ReasonHours, reasonOf(t, dropped, 1))
}

// The inbox lists every eligible message (no capacity) in the pipeline's order; legacy `alerts` rows count as inbox rows.
func TestSelectSlot_InboxIsAList(t *testing.T) {
	var all []rules.Rule
	for i := range 12 {
		all = append(all, slotRule(uint64(i+1), rules.SlotAlertsInbox, rules.KindContent, 100+i)) //nolint:gosec // small
	}
	legacy := slotRule(50, "alerts", rules.KindEngine, 900)
	got, dropped := rules.SelectSlot(rules.SlotAlertsInbox, append(all, legacy), rules.Facts{}, nil, 1, tNow)
	assert.Len(t, got, 13)
	assert.Equal(t, uint64(50), got[0].Rule.ID)
	assert.Empty(t, dropped)
}

func TestBuildHistory_Aggregates(t *testing.T) {
	w := rules.CapWindows(tNow, nil)
	h := rules.BuildHistory(w, []store.ListUserMessageEventStatsRow{
		{RuleID: 1, Event: "impression", Slot: "push", Kind: "product", LastAt: tNow.Add(-time.Hour), DayCount: 1, WeekCount: 2, CycleCount: 2},
		{RuleID: 1, Event: "impression", Slot: "smart_tip", Kind: "product", LastAt: tNow.Add(-3 * time.Hour), DayCount: 1, WeekCount: 1, CycleCount: 1},
		{RuleID: 1, Event: "dismiss", Slot: "push", Kind: "product", LastAt: tNow.Add(-30 * time.Minute), DayCount: 1, WeekCount: 1, CycleCount: 1},
		{RuleID: 2, Event: "impression", Slot: "urgent_modal", Kind: "safety", LastAt: tNow.Add(-5 * time.Hour), DayCount: 1, WeekCount: 1, CycleCount: 1},
		{RuleID: 3, Event: "impression", Slot: "alerts_inbox", Kind: "safety", LastAt: tNow.Add(-9 * time.Hour)},
		{RuleID: 4, Event: "click", Slot: "push", Kind: "product", LastAt: tNow, DayCount: 9, WeekCount: 9, CycleCount: 9},
	})
	r := h.Rules[1]
	assert.True(t, r.LastShown.Equal(tNow.Add(-time.Hour)))
	assert.True(t, r.LastDismissed.Equal(tNow.Add(-30*time.Minute)))
	assert.Equal(t, rules.WindowCounts{Day: 2, Week: 3, Cycle: 3}, r.Shown)
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 1, Cycle: 1}, r.Dismissed)
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 2}, h.Slots["push"], "shown only; clicks do not count")
	assert.True(t, h.LastSafetyShown.Equal(tNow.Add(-5*time.Hour)))
}

// ---- property tests (seeded loops, no new dependency) ----

// Golden vectors pin the hash: changing it would move every user's variant and audience.
func TestHash_GoldenVectors(t *testing.T) {
	// fnv-32a reference values (computed outside Go).
	assert.Equal(t, uint32(0x811c9dc5), rules.HashOf(""))
	assert.Equal(t, uint32(0xe40c292c), rules.HashOf("a"))
	assert.Equal(t, uint32(0xbf9cf968), rules.HashOf("foobar"))
	for _, tc := range []struct {
		user, rule uint64
		salt       string
		want       uint32
	}{
		{1, 1, "ab", 0x996a7afe},
		{42, 7, "aud", 0x7280ede8},
		{7, 3, "ab", 0x179029e2},
		{1000, 11, "ab", 0xaaae6b0d},
	} {
		assert.Equal(t, tc.want, rules.StableHash(tc.user, tc.rule, tc.salt), "%d:%d:%s", tc.user, tc.rule, tc.salt)
	}
	// A 20/30/50 split on rule 11: the pinned variant of five users.
	r := slotRule(11, "daily_advice", rules.KindContent, 300)
	r.Variants = []rules.VariantWeight{{Key: "a", Weight: 20}, {Key: "b", Weight: 30}, {Key: "c", Weight: 50}}
	for uid, want := range map[uint64]string{1: "b", 2: "c", 3: "b", 1000: "c", 987654: "c"} {
		assert.Equal(t, want, r.Assign(uid), "user %d", uid)
		assert.Equal(t, goldenVariant(uid), r.Assign(uid))
	}
}

// goldenVariant is the variant formula written out by hand: fnv32a("user:rule:ab") mod 100 over a 20/30/50 split.
func goldenVariant(uid uint64) string {
	x := rules.HashOf(fmt.Sprintf("%d:11:ab", uid)) % 100
	switch {
	case x < 20:
		return "a"
	case x < 50:
		return "b"
	}
	return "c"
}

func TestHash_ChiSquareUniformAudience(t *testing.T) {
	const users = 100_000
	var buckets [100]int
	for uid := uint64(1); uid <= users; uid++ {
		buckets[rules.StableHash(uid, 17, "aud")%100]++
	}
	chi := 0.0
	exp := float64(users) / 100
	for _, n := range buckets {
		d := float64(n) - exp
		chi += d * d / exp
	}
	assert.Less(t, chi, 148.23, "χ²(99) at p = 0.001")
	// Every audience share lands within ±2 points of its percent.
	for _, p := range []int{1, 5, 10, 25, 50, 75, 90, 99} {
		r := slotRule(17, "daily_advice", rules.KindContent, 300)
		r.AudiencePercent = p
		in := 0
		for uid := uint64(1); uid <= users; uid++ {
			if r.InAudience(uid) {
				in++
			}
		}
		assert.InDelta(t, float64(p), float64(in)*100/users, 2, "audience %d%%", p)
	}
}

func TestAudience_Monotonic(t *testing.T) {
	for seed := uint64(1); seed <= 40; seed++ {
		r := slotRule(seed, "daily_advice", rules.KindContent, 300)
		for uid := uint64(1); uid <= 500; uid++ {
			was := false
			for p := 0; p <= 100; p += 5 {
				r.AudiencePercent = p
				in := r.InAudience(uid)
				require.False(t, was && !in, "rule %d user %d left the audience at %d%%", seed, uid, p)
				was = in
			}
		}
	}
}

func TestVariants_SharesFollowWeights(t *testing.T) {
	const users = 60_000
	for _, ws := range [][]rules.VariantWeight{
		{{Key: "a", Weight: 50}, {Key: "b", Weight: 50}},
		{{Key: "a", Weight: 10}, {Key: "b", Weight: 30}, {Key: "c", Weight: 60}},
		{{Key: "x", Weight: 1}, {Key: "y", Weight: 1}, {Key: "z", Weight: 1}, {Key: "w", Weight: 1}},
	} {
		r := slotRule(23, "daily_advice", rules.KindContent, 300)
		r.Variants = ws
		got := map[string]int{}
		for uid := uint64(1); uid <= users; uid++ {
			got[r.Assign(uid)]++
		}
		sum := 0
		for _, w := range ws {
			sum += w.Weight
		}
		for _, w := range ws {
			assert.InDelta(t, float64(w.Weight)/float64(sum), float64(got[w.Key])/users, 0.02, "%v %s", ws, w.Key)
		}
	}
}

// No user flips variant on deploy: a legacy a/b rule's assignment equals the pre-v6 formula for every user.
func TestVariants_LegacyABUnchanged(t *testing.T) {
	legacy := func(uid, rid uint64, split int) string { // rules.go before T-M10-04
		if int(rules.HashOf(fmt.Sprintf("%d:%d:ab", uid, rid))%100) < split {
			return "a"
		}
		return "b"
	}
	for _, split := range []int{0, 1, 30, 50, 99, 100} {
		r := slotRule(31, "daily_advice", rules.KindContent, 300)
		r.VariantsEnabled, r.ABSplit, r.Content.B = true, split, &rules.Variant{Title: "B", Body: "b"}
		for uid := uint64(1); uid <= 10_000; uid++ {
			require.Equal(t, legacy(uid, 31, split), r.Assign(uid), "split %d user %d", split, uid)
		}
		got, _ := rules.SelectSlot("daily_advice", []rules.Rule{r}, rules.Facts{}, nil, 77, tNow)
		require.Len(t, got, 1)
		assert.Equal(t, legacy(77, 31, split), got[0].Variant)
	}
}

// The selection never mutates the caller's rules (ActiveRows shares them).
func TestSelectSlot_DoesNotMutate(t *testing.T) {
	all := []rules.Rule{slotRule(2, "daily_advice", rules.KindContent, 100), slotRule(1, "daily_advice", rules.KindContent, 300)}
	before := slices.Clone(all)
	_, _ = rules.SelectSlot("daily_advice", all, rules.Facts{}, nil, 1, tNow)
	assert.Equal(t, before, all)
}
