package companionhome

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// «ریتمی همراه» v2 day signals: a coarse yes/no per signal the owner consented to, derived from her health log of
// today (Tehran day). Only the boolean leaves this file — never a count, a code, a severity, a score or text — and the
// copy built on it is probabilistic («احتمالاً…»). The mapping is deliberately coarse:
//
//	pain   any pain location logged today (any level; a «no pain» tick is not a location row)
//	energy energy logged low (or the legacy very_low)
//	sleep  sleep quality poor, or a short night: duration 0_3 / 3_6, or (post-partum form) fewer than 6 hours
//
// A day without a matching entry is false, not null: null means only that the owner does not share that signal.
//
// Mood is NOT derived and never crosses: mood (and EPDS) is a never-shared category for every domain and role
// (SPEC §1.7, family.md §4, D-133; asserted by TestMember_MoodNeverShared / TestMember_NeverSharedNotReachable), so the
// member payload has no `mood` key at all and the owner's share_mood consent is stored but has no member-side effect
// until that rule is changed by a product decision.

// memberSignals are the signals a member payload may carry, in card order (SignalOrder minus mood, see above).
var memberSignals = []companion.Signal{companion.SignalPain, companion.SignalEnergy, companion.SignalSleep}

// shortSleepHours is the post-partum hours threshold of a short night.
const shortSleepHours = 6

// DayEntry is one health-log slot the signals read (the shape of store.ListOwnersDaySignalEntriesRow).
type DayEntry struct {
	Category, Param, Item, Code string
	Num                         string // value_num as MariaDB prints it ("" when NULL)
}

// DaySignals are the derived booleans of one owner's day.
type DaySignals struct{ Pain, Energy, Sleep bool }

// DeriveSignals maps one owner's day entries to the coarse signals. Pure.
func DeriveSignals(entries []DayEntry) DaySignals {
	var out DaySignals
	for _, e := range entries {
		switch e.Category + "." + e.Param {
		case "pain.location":
			out.Pain = out.Pain || e.Item != ""
		case "appetite_energy.energy":
			out.Energy = out.Energy || e.Code == "low" || e.Code == "very_low"
		case "sleep.quality":
			out.Sleep = out.Sleep || e.Code == "poor"
		case "sleep.duration":
			out.Sleep = out.Sleep || e.Code == "0_3" || e.Code == "3_6"
		case "sleep.hours":
			if h, err := strconv.ParseFloat(e.Num, 64); err == nil {
				out.Sleep = out.Sleep || h < shortSleepHours
			}
		}
	}
	return out
}

// signalsJSON is the partner card's signals: a signal's day value when the owner shares it and the link may show it
// (shown), null otherwise.
func signalsJSON(scope companion.SignalScope, day DaySignals, shown bool) *jsonx.OrderedMap {
	vals := map[companion.Signal]bool{
		companion.SignalPain: day.Pain, companion.SignalEnergy: day.Energy, companion.SignalSleep: day.Sleep,
	}
	out := jsonx.NewObject()
	for _, s := range memberSignals {
		if shown && scope.On(s) {
			out.Set(string(s), vals[s])
		} else {
			out.Set(string(s), nil)
		}
	}
	return out
}

// signalsShown reports whether a link's day signals may be evaluated at all: the link is active (ListForCompanion
// only lists active, unexpired links of unblocked owners) and the coarse cycle section is effectively readable. The
// signals ride on the cycle_phase decision (family.Decide over the snapshot), so its pause rules — mode scope, the
// mode epoch an owner's mode change bumps (F-23), a male owner — switch them off exactly as they switch off the cycle
// card. The read itself is audited with the cycle section (same request, before the card is built).
func signalsShown(l companion.Link) bool {
	return l.Status == companion.StatusActive && l.Grants.Of(companion.SectionCycle).CanRead()
}

// daySignals loads today's signal entries of every owner that needs them in one statement.
func daySignals(ctx context.Context, q *store.Queries, ownerIDs []uint64, now time.Time) (map[uint64]DaySignals, error) {
	out := map[uint64]DaySignals{}
	if len(ownerIDs) == 0 {
		return out, nil
	}
	rows, err := q.ListOwnersDaySignalEntries(ctx, store.ListOwnersDaySignalEntriesParams{
		OwnerIds: ownerIDs, LogDate: civildate.FromTime(now.In(civildate.Tehran)),
	})
	if err != nil {
		return nil, fmt.Errorf("companion home: day signals: %w", err)
	}
	byOwner := map[uint64][]DayEntry{}
	for _, r := range rows {
		byOwner[r.UserID] = append(byOwner[r.UserID], DayEntry{Category: r.Category, Param: r.Param, Item: r.Item,
			Code: r.ValueCode.String, Num: r.ValueNum.String})
	}
	for id, es := range byOwner {
		out[id] = DeriveSignals(es)
	}
	return out, nil
}
