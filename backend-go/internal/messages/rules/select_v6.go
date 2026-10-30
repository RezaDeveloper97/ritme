package rules

import (
	"hash/fnv"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Selection v6 (T-M10-04, SPEC §5.4 / §5.4.1, DECISIONS.md M-04…M-09, D-162): per-slot kinds and capacities, the
// fixed invariants of §5.4.1, caps counted from shown or dismissed, real-cycle caps, the least-recently-shown
// tie-break, weighted variants and the user-local clock. SelectSlot is pure: the caller brings the rules, the facts,
// the user's history (LoadHistory: one grouped query) and now in the user's zone (civildate.LocalNow).

// Member slots, reserved for M11 (nothing is served there yet).
const (
	SlotMemberAlert = "member_alert"
	SlotMemberTip   = "member_tip"
)

// Stored event names (M-01: the columns keep the pre-v6 names; shown = impression, dismissed = dismiss).
const (
	EventShown     = "impression"
	EventDismissed = "dismiss"
)

// Drop reasons of SelectSlot, in pipeline order (the simulator translates them). conditions, audience and cooldown
// reuse the explain.go constants.
const (
	ReasonKindNotAllowed    = "kind_not_allowed"
	ReasonSchedule          = "schedule"
	ReasonHours             = "hours"
	ReasonExclude           = "exclude"
	ReasonRequiresUserInput = "requires_user_input"
	ReasonLowConfidence     = "low_confidence"
	ReasonContraception     = "contraception"
	ReasonPostSafety        = "post_safety_24h"
	ReasonQuietHours        = "quiet_hours"
	ReasonCapDay            = "cap_day"
	ReasonCapWeek           = "cap_week"
	ReasonCapCycle          = "cap_cycle"
	ReasonSlotCap           = "slot_cap"
	ReasonCapacity          = "capacity"
)

// SelectionReasons is the reasons enum of SelectSlot (an unserved rule is dropped with inactive / not_approved).
var SelectionReasons = []string{ReasonKindNotAllowed, ReasonSchedule, ReasonHours, ReasonConditions, ReasonExclude,
	ReasonAudience, ReasonRequiresUserInput, ReasonLowConfidence, ReasonContraception, ReasonPostSafety,
	ReasonQuietHours, ReasonCapDay, ReasonCapWeek, ReasonCapCycle, ReasonCooldown, ReasonSlotCap, ReasonCapacity}

// PostSafetyBlock is how long product and marketing stay blocked after a safety message was shown.
const PostSafetyBlock = 24 * time.Hour

// Slot is one slot of SPEC §5.4: its capacity (List = every eligible message), allowed kinds and default caps.
type Slot struct {
	Name     string
	Capacity int // winners per request; 0 with List
	List     bool
	Kinds    []string
	// MessageCaps apply to each message of the slot on top of its own caps; SlotCaps count the shows of every
	// message of the slot together.
	MessageCaps []Cap
	SlotCaps    []Cap
	// QuietHours is a user-local [from, to) window in which nothing is served.
	QuietHours *HourWindow
	Reserved   bool
}

// Slots is the slot table (SPEC §5.4, §5.4.1, §1.8).
var Slots = map[string]Slot{
	"urgent_modal":  {Name: "urgent_modal", Capacity: 1, Kinds: SlotKinds["urgent_modal"]},
	"status_banner": {Name: "status_banner", Capacity: 1, Kinds: SlotKinds["status_banner"]}, // while its condition holds
	"smart_tip": {Name: "smart_tip", Capacity: 1, Kinds: SlotKinds["smart_tip"],
		MessageCaps: []Cap{{Count: 1, Period: "cycle"}}},
	"daily_advice": {Name: "daily_advice", Capacity: 3, Kinds: SlotKinds["daily_advice"],
		MessageCaps: []Cap{{Count: 1, Period: "week"}}}, // no repeat within 7 local days
	"daily_challenge": {Name: "daily_challenge", Capacity: 1, Kinds: SlotKinds["daily_challenge"]},
	"push": {Name: "push", Capacity: 1, Kinds: SlotKinds["push"],
		SlotCaps: []Cap{{Count: 1, Period: "day"}, {Count: 3, Period: "week"}}, QuietHours: &HourWindow{23, 8}},
	"modal":         {Name: "modal", Capacity: 1, Kinds: SlotKinds["modal"], SlotCaps: []Cap{{Count: 1, Period: "week"}}},
	SlotAlertsInbox: {Name: SlotAlertsInbox, List: true, Kinds: SlotKinds[SlotAlertsInbox]},
	"alerts":        {Name: "alerts", List: true, Kinds: SlotKinds["alerts"]}, // pre-v6 name of the inbox
	"sms":           {Name: "sms", Capacity: 1, Kinds: SlotKinds["sms"]},
	SlotMemberAlert: {Name: SlotMemberAlert, Capacity: 1, Reserved: true},
	SlotMemberTip:   {Name: SlotMemberTip, Capacity: 1, Reserved: true},
}

// Dropped is a candidate that did not win, with the first gate that stopped it.
type Dropped struct {
	RuleID uint64
	Reason string
}

// WindowCounts are event counts since the three cap windows.
type WindowCounts struct{ Day, Week, Cycle int }

func (w WindowCounts) add(o WindowCounts) WindowCounts {
	return WindowCounts{w.Day + o.Day, w.Week + o.Week, w.Cycle + o.Cycle}
}

func (w WindowCounts) of(period string) int {
	switch period {
	case "day":
		return w.Day
	case "week":
		return w.Week
	case "cycle":
		return w.Cycle
	}
	return 0
}

// RuleHistory is one rule's shown / dismissed history.
type RuleHistory struct {
	LastShown, LastDismissed time.Time // zero = never
	Shown, Dismissed         WindowCounts
}

// capCount is what a cap counts: the shows, or the dismissals when more were recorded (a dismissal is a show).
func (h RuleHistory) capCount(period string) int {
	return max(h.Shown.of(period), h.Dismissed.of(period))
}

// lastSeen is the later of the last shown and the last dismissed (the cooldown counts from it).
func (h RuleHistory) lastSeen() time.Time {
	if h.LastDismissed.After(h.LastShown) {
		return h.LastDismissed
	}
	return h.LastShown
}

// HistoryWindows are the cap-window starts: the user-local day, the last 7 user-local days and the current cycle.
type HistoryWindows struct{ Day, Week, Cycle time.Time }

// UserHistory is a user's message history for one request.
type UserHistory struct {
	Windows HistoryWindows
	Rules   map[uint64]RuleHistory
	// Slots are the shows per slot (every message) since the day and week windows.
	Slots           map[string]WindowCounts
	LastSafetyShown time.Time // zero = never
}

// CapWindows are the windows of now (in the user's zone): day = local midnight; week = local midnight 6 days ago;
// cycle = local midnight of the engine's current_period_start (M-04), else now − the effective cycle length.
func CapWindows(now time.Time, f Facts) HistoryWindows {
	loc := now.Location()
	today := civildate.FromTime(now)
	w := HistoryWindows{Day: today.Midnight(loc), Week: today.AddDays(-6).Midnight(loc)}
	if s, ok := f["current_period_start"].(string); ok {
		if d, err := civildate.Parse(s); err == nil && !d.After(today) {
			w.Cycle = d.Midnight(loc)
			return w
		}
	}
	w.Cycle = now.Add(-capWindow("cycle", f))
	return w
}

// BuildHistory folds the grouped ListUserMessageEventStats rows (all_rules = 1) into a UserHistory.
func BuildHistory(w HistoryWindows, rows []store.ListUserMessageEventStatsRow) *UserHistory {
	h := &UserHistory{Windows: w, Rules: make(map[uint64]RuleHistory, len(rows)), Slots: map[string]WindowCounts{}}
	for i := range rows {
		r := &rows[i]
		c := WindowCounts{int(r.DayCount), int(r.WeekCount), int(r.CycleCount)}
		rh := h.Rules[r.RuleID]
		switch r.Event {
		case EventShown, "shown":
			rh.Shown = rh.Shown.add(c)
			if r.LastAt.After(rh.LastShown) {
				rh.LastShown = r.LastAt
			}
			h.Slots[r.Slot] = h.Slots[r.Slot].add(WindowCounts{Day: c.Day, Week: c.Week})
			if r.Kind == KindSafety && r.LastAt.After(h.LastSafetyShown) {
				h.LastSafetyShown = r.LastAt
			}
		case EventDismissed, "dismissed":
			rh.Dismissed = rh.Dismissed.add(c)
			if r.LastAt.After(rh.LastDismissed) {
				rh.LastDismissed = r.LastAt
			}
		default:
			continue
		}
		h.Rules[r.RuleID] = rh
	}
	return h
}

// HashOf is FNV-32a of s.
func HashOf(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// StableHash is the stable per-(user, rule) hash of a salt: fnv32a over "user:rule:salt". The byte layout is the
// pre-v6 one, so no user's audience or variant moves on deploy.
func StableHash(userID, ruleID uint64, salt string) uint32 {
	return HashOf(strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(ruleID, 10) + ":" + salt)
}

// WeightedVariant is the stable weighted pick: StableHash(user, rule, "ab") mod Σweights walked over the weights
// (negative weights count as 0; no positive weight → the first key).
func WeightedVariant(userID, ruleID uint64, vs []VariantWeight) string {
	if len(vs) == 0 {
		return "a"
	}
	sum := 0
	for _, v := range vs {
		sum += max(v.Weight, 0)
	}
	if sum <= 0 || len(vs) == 1 {
		return vs[0].Key
	}
	x := int(StableHash(userID, ruleID, "ab") % uint32(sum)) //nolint:gosec // sum > 0 and small
	for _, v := range vs {
		if x < max(v.Weight, 0) {
			return v.Key
		}
		x -= max(v.Weight, 0)
	}
	return vs[0].Key
}

// inZone reads a Tehran-stored wall-clock time as the same wall clock in loc (schedules are user-local, M-08).
func inZone(t time.Time, loc *time.Location) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

func factTrue(f Facts, name string) bool {
	switch v := f[name].(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

// testsFact reports whether one of the rule's condition groups reads the fact.
func (r *Rule) testsFact(name string) bool {
	for _, g := range r.Groups {
		if slices.ContainsFunc(g.Items, func(c Condition) bool { return c.Fact == name }) {
			return true
		}
	}
	return false
}

// isFertileWindow reports whether the rule is a fertile-window message: the topic (M-06), or an older rule whose
// conditions or texts read fertility facts without it.
func (r *Rule) isFertileWindow() bool {
	return slices.Contains(r.Topics, TopicFertileWindow) || r.UsesFertility()
}

// candidateOf reports whether a rule of placement p runs in slot (legacy `alerts` rows are inbox rows).
func candidateOf(slot, p string) bool {
	return p == slot || (slot == SlotAlertsInbox && p == "alerts")
}

// SelectSlot runs the v6 pipeline for one slot: kind → schedule → hours → conditions → exclude → audience →
// invariants → caps → ordering → capacity → variant. now is the request time in the user's zone; h may be nil (no
// history). Rules of other slots are not candidates. all is not modified.
func SelectSlot(slot string, all []Rule, f Facts, h *UserHistory, userID uint64, now time.Time) ([]Choice, []Dropped) {
	if h == nil {
		h = &UserHistory{}
	}
	s, known := Slots[slot]
	if !known {
		return nil, nil
	}
	var (
		dropped []Dropped
		pass    []cand
	)
	ctx := gateCtx{slot: &s, f: f, h: h, now: now, userID: userID, hour: now.Hour(),
		needsInput: factTrue(f, "requires_user_input"), lowConf: f["confidence"] == "low",
		hormonal: factTrue(f, "hormonal_contraception")}
	for i := range all {
		r := &all[i]
		if !candidateOf(slot, r.Placement) {
			continue
		}
		if why := ctx.gate(r); why != "" {
			dropped = append(dropped, Dropped{RuleID: r.ID, Reason: why})
			continue
		}
		pass = append(pass, cand{r: r, last: h.Rules[r.ID].LastShown})
	}
	slices.SortFunc(pass, func(x, y cand) int {
		a, b := x.r, y.r
		if as, bs := a.Kind == KindSafety, b.Kind == KindSafety; as != bs {
			if as {
				return -1
			}
			return 1
		}
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		if c := x.last.Compare(y.last); c != 0 {
			return c // never shown (zero) first
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	n := len(pass)
	if !s.List {
		n = min(n, s.Capacity)
	}
	winners := make([]Choice, 0, n)
	for i, c := range pass {
		r := c.r
		if i >= n {
			dropped = append(dropped, Dropped{RuleID: r.ID, Reason: ReasonCapacity})
			continue
		}
		v := WeightedVariant(userID, r.ID, r.EffectiveVariants())
		if r.Kind == KindSafety { // M-07: no A/B
			v = r.EffectiveVariants()[0].Key
		}
		winners = append(winners, Choice{Rule: r, Variant: v})
	}
	return winners, dropped
}

// cand is a candidate that passed every gate, with its last show for the tie-break.
type cand struct {
	r    *Rule
	last time.Time
}

type gateCtx struct {
	slot                          *Slot
	f                             Facts
	h                             *UserHistory
	now                           time.Time
	userID                        uint64
	hour                          int
	needsInput, lowConf, hormonal bool
}

// gate is the first reason that stops r ("" = eligible), in pipeline order.
func (g *gateCtx) gate(r *Rule) string {
	if !r.IsActive {
		return ReasonInactive
	}
	if !r.Served() {
		return ReasonNotApproved
	}
	if g.slot.Reserved || !slices.Contains(g.slot.Kinds, r.Kind) {
		return ReasonKindNotAllowed
	}
	start, end := r.EffectiveSchedule()
	loc := g.now.Location()
	if start = inZone(start, loc); !start.IsZero() && g.now.Before(start) {
		return ReasonSchedule
	}
	if end = inZone(end, loc); !end.IsZero() && !g.now.Before(end) {
		return ReasonSchedule
	}
	if ws := r.EffectiveHourWindows(); len(ws) > 0 && !slices.ContainsFunc(ws, func(w HourWindow) bool { return w.inHours(g.hour) }) {
		return ReasonHours
	}
	for _, gr := range r.Groups {
		if !MatchGroup(gr, g.f) {
			return ReasonConditions
		}
	}
	if slices.ContainsFunc(r.Exceptions, func(c Condition) bool { return Match(c, g.f) }) {
		return ReasonExclude
	}
	safety := r.Kind == KindSafety
	if !safety && !r.InAudience(g.userID) {
		return ReasonAudience
	}
	if why := g.invariant(r); why != "" {
		return why
	}
	if safety {
		return "" // SPEC §5.3: no repeat cap
	}
	return g.caps(r)
}

// invariant applies the fixed rules of SPEC §5.4.1 (never admin data).
func (g *gateCtx) invariant(r *Rule) string {
	if g.needsInput && r.Kind != KindSafety && r.Kind != KindEngine {
		return ReasonRequiresUserInput
	}
	if g.lowConf && r.Kind == KindClinical && !r.testsFact("confidence") {
		return ReasonLowConfidence
	}
	if g.hormonal && r.isFertileWindow() {
		return ReasonContraception
	}
	if (r.Kind == KindProduct || r.Kind == KindMarketing) && !g.h.LastSafetyShown.IsZero() &&
		g.now.Sub(g.h.LastSafetyShown) < PostSafetyBlock {
		return ReasonPostSafety
	}
	if q := g.slot.QuietHours; q != nil && q.inHours(g.hour) {
		return ReasonQuietHours
	}
	return ""
}

// caps checks the rule's own caps, its cooldown, then the slot's default caps.
func (g *gateCtx) caps(r *Rule) string {
	rh := g.h.Rules[r.ID]
	for _, c := range r.EffectiveCaps() {
		if rh.capCount(c.Period) >= c.Count {
			switch c.Period {
			case "day":
				return ReasonCapDay
			case "week":
				return ReasonCapWeek
			}
			return ReasonCapCycle
		}
	}
	if r.CooldownDays > 0 {
		if last := rh.lastSeen(); !last.IsZero() && g.now.Sub(last) < time.Duration(r.CooldownDays)*24*time.Hour {
			return ReasonCooldown
		}
	}
	for _, c := range g.slot.MessageCaps {
		if rh.capCount(c.Period) >= c.Count {
			return ReasonSlotCap
		}
	}
	sh := g.h.Slots[g.slot.Name]
	for _, c := range g.slot.SlotCaps {
		if sh.of(c.Period) >= c.Count {
			return ReasonSlotCap
		}
	}
	return ""
}
