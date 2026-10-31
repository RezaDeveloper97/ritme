package compiled_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// tNow is 12:00 in Tehran on 2026-10-21.
var tNow = time.Date(2026, 10, 21, 12, 0, 0, 0, civildate.Tehran)

var zones = func() []*time.Location {
	out := []*time.Location{civildate.Tehran, time.UTC}
	for _, n := range []string{"Europe/Berlin", "America/New_York"} {
		if l, err := time.LoadLocation(n); err == nil {
			out = append(out, l)
		}
	}
	return out
}()

func pick[T any](r *rand.Rand, xs ...T) T { return xs[r.IntN(len(xs))] }

var factNames = []string{"mode", "cycle_day", "symptoms", "confidence", "flag", "phase", "score", "is_fertile_window",
	"current_period_start", "missing"}

func randValue(r *rand.Rand) any {
	switch r.IntN(11) {
	case 0:
		return float64(r.IntN(30))
	case 1:
		return r.IntN(30)
	case 2:
		return json.Number(fmt.Sprint(r.IntN(30)))
	case 3:
		return pick(r, "cycle", "pregnancy", "low", "high", "s1", "s2", "true", "false", "ovulation", "2026-10-10")
	case 4:
		return r.IntN(2) == 0
	case 5:
		return []any{pick[any](r, "cycle", "s1", 3.0, true), pick[any](r, "pregnancy", "s2", 12.0, "false", "low")}
	case 6:
		return []string{pick(r, "s1", "cycle", "high"), pick(r, "s3", "ttc")}
	case 7:
		return pick(r, "s1, s2", "cycle,ttc", "true,low")
	case 8:
		return []any{float64(r.IntN(10)), float64(10 + r.IntN(20))}
	case 9:
		return nil
	}
	return []any{"5", "20"}
}

func randCond(r *rand.Rand) rules.Condition {
	return rules.Condition{Fact: pick(r, factNames...), Op: pick(r, rules.Ops...), Value: randValue(r)}
}

func randFacts(r *rand.Rand) rules.Facts {
	f := rules.Facts{}
	if r.IntN(5) > 0 {
		f["mode"] = pick(r, "cycle", "pregnancy", "ttc")
	}
	if r.IntN(5) > 0 {
		f["cycle_day"] = pick[any](r, float64(r.IntN(30)), r.IntN(30), int64(r.IntN(30)))
	}
	if r.IntN(3) > 0 {
		f["symptoms"] = pick(r, []string{"s1"}, []string{"s2", "s3"}, []string{})
	}
	if r.IntN(3) == 0 {
		f["confidence"] = pick(r, "low", "high")
	}
	if r.IntN(3) == 0 {
		f["flag"] = pick[any](r, true, false, "true")
	}
	if r.IntN(3) == 0 {
		f["phase"] = pick(r, "ovulation", "luteal")
	}
	if r.IntN(3) == 0 {
		f["score"] = json.Number(fmt.Sprint(r.IntN(30)))
	}
	if r.IntN(6) == 0 {
		f["requires_user_input"] = pick[any](r, true, "true", false)
	}
	if r.IntN(5) == 0 {
		f["hormonal_contraception"] = true
	}
	if r.IntN(2) == 0 {
		f["current_period_start"] = "2026-10-10"
	}
	return f
}

func randRule(r *rand.Rand, id uint64) rules.Rule {
	kind := pick(r, rules.Kinds...)
	ru := rules.Rule{
		ID: id, Level: pick(r, rules.Levels...), Kind: kind, Priority: pick(r, 1000, 900, 900, 500, 500, 500, 50, 50),
		Placement: pick(r, rules.Placements...), ABSplit: r.IntN(101), HoursFrom: -1, HoursTo: -1,
		AudiencePercent: pick(r, 100, 100, 0, 30, 70, 101), IsActive: r.IntN(10) > 0,
		Status:  pick(r, rules.StatusApproved, rules.StatusApproved, rules.StatusDraft, rules.StatusPending),
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}},
	}
	if r.IntN(4) == 0 {
		ru.ApprovedVersion = 1
	}
	switch r.IntN(3) {
	case 0:
		ru.VariantsEnabled = true
		ru.Content.B = &rules.Variant{Title: "tb"}
	case 1:
		for k := range 1 + r.IntN(4) {
			ru.Variants = append(ru.Variants, rules.VariantWeight{Key: fmt.Sprintf("v%d", k), Weight: r.IntN(60) - 5})
		}
	}
	if r.IntN(2) == 0 {
		for range 1 + r.IntN(2) {
			ru.Caps = append(ru.Caps, rules.Cap{Count: r.IntN(3), Period: pick(r, "day", "week", "cycle", "month")})
		}
	} else if r.IntN(3) == 0 {
		ru.CapCount, ru.CapPeriod = 1+r.IntN(2), pick(r, "day", "week", "cycle")
	}
	ru.CooldownDays = pick(r, 0, 0, 1, 3)
	switch r.IntN(4) {
	case 0:
		ru.HourWindows = []rules.HourWindow{{r.IntN(24), r.IntN(24)}}
	case 1:
		ru.HoursFrom, ru.HoursTo = r.IntN(24), r.IntN(24)
	}
	if r.IntN(4) == 0 {
		ru.StartsAt = tNow.Add(time.Duration(r.IntN(96)-48) * time.Hour)
	}
	if r.IntN(4) == 0 {
		ru.EndsAt = tNow.Add(time.Duration(r.IntN(96)-24) * time.Hour)
	}
	if r.IntN(6) == 0 {
		ru.StartsOn = civildate.FromTime(tNow).AddDays(r.IntN(5) - 2)
	}
	if r.IntN(6) == 0 {
		ru.Topics = []string{rules.TopicFertileWindow}
	}
	for range r.IntN(3) {
		g := rules.Group{Op: pick(r, "and", "or", "")}
		for range r.IntN(4) {
			g.Items = append(g.Items, randCond(r))
		}
		ru.Groups = append(ru.Groups, g)
	}
	for range r.IntN(2) {
		ru.Exceptions = append(ru.Exceptions, randCond(r))
	}
	return ru
}

func randHistory(r *rand.Rand, rs []rules.Rule, now time.Time) *rules.UserHistory {
	if r.IntN(8) == 0 {
		return nil
	}
	h := &rules.UserHistory{Rules: map[uint64]rules.RuleHistory{}, Slots: map[string]rules.WindowCounts{}}
	wc := func() rules.WindowCounts {
		return rules.WindowCounts{Day: r.IntN(2), Week: r.IntN(3), Cycle: r.IntN(4)}
	}
	for _, ru := range rs {
		if r.IntN(2) == 0 {
			continue
		}
		rh := rules.RuleHistory{Shown: wc(), Dismissed: wc()}
		if r.IntN(3) > 0 {
			rh.LastShown = now.Add(-time.Duration(1+r.IntN(5*24)) * time.Hour)
		}
		if r.IntN(3) == 0 {
			rh.LastDismissed = now.Add(-time.Duration(1+r.IntN(5*24)) * time.Hour)
		}
		h.Rules[ru.ID] = rh
	}
	for name := range rules.Slots {
		if r.IntN(3) == 0 {
			h.Slots[name] = wc()
		}
	}
	if r.IntN(4) == 0 {
		h.LastSafetyShown = now.Add(-time.Duration(r.IntN(48)) * time.Hour)
	}
	return h
}

type win struct {
	ID      uint64
	Variant string
}

func wins(cs []rules.Choice) []win {
	out := make([]win, 0, len(cs))
	for _, c := range cs {
		out = append(out, win{c.Rule.ID, c.Variant})
	}
	return out
}

// Select on compiled rules returns the same winners (order and variant) as T-M10-04's parsed SelectSlot.
func TestSelect_EquivalentToParsedPath(t *testing.T) {
	r := rand.New(rand.NewPCG(10, 5)) //nolint:gosec // test data
	slots := make([]string, 0, len(rules.Slots))
	for s := range rules.Slots {
		slots = append(slots, s)
	}
	slices.Sort(slots)
	nonEmpty := 0
	for set := range 1000 {
		n := 1 + r.IntN(60)
		ids := r.Perm(n * 3)
		rs := make([]rules.Rule, n)
		for i := range rs {
			rs[i] = randRule(r, uint64(ids[i]+1)) //nolint:gosec // small
		}
		c := compiled.CompileRules(rs)
		require.Empty(t, c.Errors)
		for range 3 {
			now := tNow.Add(time.Duration(r.IntN(48*60)-24*60) * time.Minute).In(pick(r, zones...))
			f := randFacts(r)
			h := randHistory(r, rs, now)
			user := uint64(r.IntN(10_000)) //nolint:gosec // small
			for _, s := range slots {
				want, _ := rules.SelectSlot(s, rs, f, h, user, now)
				got := c.Select(s, f, h, user, now)
				if len(want) > 0 {
					nonEmpty++
				}
				require.Equal(t, wins(want), wins(got), "set %d slot %s", set, s)
			}
		}
	}
	assert.Greater(t, nonEmpty, 1000, "the random sets must exercise winners")
}

// NeedFacts of a slot is the union of the facts its candidate rules' conditions and exceptions read.
func TestNeedFacts_UnionOfReferencedFacts(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 9)) //nolint:gosec // test data
	for range 300 {
		n := r.IntN(40)
		rs := make([]rules.Rule, n)
		for i := range rs {
			rs[i] = randRule(r, uint64(i+1)) //nolint:gosec // small
		}
		c := compiled.CompileRules(rs)
		for name, s := range rules.Slots {
			want := map[string]bool{}
			for i := range rs {
				ru := &rs[i]
				if s.Reserved || !ru.Served() || !rules.CandidateOf(name, ru.Placement) || !slices.Contains(s.Kinds, ru.Kind) {
					continue
				}
				for _, g := range ru.Groups {
					for _, it := range g.Items {
						want[it.Fact] = true
					}
				}
				for _, it := range ru.Exceptions {
					want[it.Fact] = true
				}
			}
			got := c.NeedFacts[name]
			assert.True(t, slices.IsSorted(got), name)
			assert.Len(t, got, len(want), name)
			for f := range want {
				assert.True(t, got.Has(f), "%s needs %s", name, f)
			}
		}
		all := c.Need("daily_advice", "smart_tip", "nope")
		for _, f := range c.NeedFacts["daily_advice"] {
			assert.True(t, all.Has(f))
		}
		for _, f := range c.NeedFacts["smart_tip"] {
			assert.True(t, all.Has(f))
		}
		assert.Len(t, all, len(slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(c.NeedFacts["daily_advice"]), c.NeedFacts["smart_tip"]...))))))
	}
}

func TestCompileRules_UnknownOpIsSkipped(t *testing.T) {
	ok := rules.Rule{ID: 1, Kind: rules.KindContent, Level: rules.LevelInfo, Priority: 300, Placement: "daily_advice",
		Status: rules.StatusApproved, IsActive: true, AudiencePercent: 100, HoursFrom: -1, HoursTo: -1}
	bad := ok
	bad.ID = 2
	bad.Groups = []rules.Group{{Items: []rules.Condition{{Fact: "mode", Op: "like", Value: "x"}}}}
	c := compiled.CompileRules([]rules.Rule{ok, bad})
	require.Len(t, c.Errors, 1)
	assert.Equal(t, uint64(2), c.Errors[0].RuleID)
	assert.Equal(t, []win{{1, "a"}}, wins(c.Select("daily_advice", rules.Facts{}, nil, 7, tNow)))
}
