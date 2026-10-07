package telemed

import (
	"slices"
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Rule is one weekly availability window (telemed_availability_rules) in Tehran wall-clock minutes.
type Rule struct {
	Weekday     time.Weekday
	StartMinute int      // minutes after midnight
	EndMinute   int      // exclusive; a visit must end by then
	SlotMinutes int      // step of the slot grid
	Modes       []string // nil = every mode
}

// Allows reports whether the rule takes visits of mode.
func (r Rule) Allows(mode string) bool { return len(r.Modes) == 0 || slices.Contains(r.Modes, mode) }

// SlotQuery is one slot generation: the grid of rules for a visit of Duration minutes in Mode over Days calendar days
// from From, minus slots starting before Now + LeadMinutes, past the horizon, or overlapping Blocked (time off and busy
// intervals).
type SlotQuery struct {
	Rules    []Rule
	Mode     string
	Duration int // visit length in minutes (> 0)
	From     civildate.Date
	Days     int
	Now      time.Time
	Blocked  []Interval
}

// Slot is one free start time.
type Slot struct {
	Start, End time.Time
}

// Day is the free slots of one calendar day.
type Day struct {
	Date  civildate.Date
	Slots []Slot
}

// at is the Tehran instant of minute m on day d.
func at(d civildate.Date, m int) time.Time {
	return d.TehranMidnight().Add(time.Duration(m) * time.Minute)
}

// GenerateDays returns one Day per calendar day of the query (empty days included), slots sorted by start. Overlapping
// rules never yield the same start twice. Days beyond today + HorizonDays − 1 are empty.
func GenerateDays(q SlotQuery) []Day {
	days := make([]Day, 0, max(q.Days, 0))
	if q.Duration <= 0 {
		for i := range max(q.Days, 0) {
			days = append(days, Day{Date: q.From.AddDays(i), Slots: []Slot{}})
		}
		return days
	}
	earliest := q.Now.Add(LeadMinutes * time.Minute)
	last := civildate.InTehran(q.Now).AddDays(HorizonDays - 1)
	for i := range max(q.Days, 0) {
		d := q.From.AddDays(i)
		day := Day{Date: d, Slots: []Slot{}}
		if d.After(last) {
			days = append(days, day)
			continue
		}
		seen := map[int]bool{}
		for _, r := range q.Rules {
			if r.Weekday != d.Weekday() || !r.Allows(q.Mode) || r.SlotMinutes <= 0 {
				continue
			}
			for m := r.StartMinute; m+q.Duration <= r.EndMinute; m += r.SlotMinutes {
				if seen[m] {
					continue
				}
				s := Slot{Start: at(d, m), End: at(d, m+q.Duration)}
				if s.Start.Before(earliest) || blocked(s, q.Blocked) {
					continue
				}
				seen[m] = true
				day.Slots = append(day.Slots, s)
			}
		}
		sort.Slice(day.Slots, func(a, b int) bool { return day.Slots[a].Start.Before(day.Slots[b].Start) })
		days = append(days, day)
	}
	return days
}

// FirstSlot is the earliest free slot of the query, if any.
func FirstSlot(q SlotQuery) (Slot, bool) {
	for _, d := range GenerateDays(q) {
		if len(d.Slots) > 0 {
			return d.Slots[0], true
		}
	}
	return Slot{}, false
}

func blocked(s Slot, spans []Interval) bool {
	iv := Interval(s)
	for _, b := range spans {
		if iv.Overlaps(b) {
			return true
		}
	}
	return false
}

// VisitLength is the duration of one offered mode (for FirstSlotAny).
type VisitLength struct {
	Mode     string
	Duration int
}

// FirstSlotAny is the earliest free slot over several modes (the directory card's «اولین وقت»), with its mode.
func FirstSlotAny(rules []Rule, visits []VisitLength, from civildate.Date, days int, now time.Time, blocked []Interval) (Slot, string, bool) {
	var best Slot
	bestMode, found := "", false
	for _, v := range visits {
		s, ok := FirstSlot(SlotQuery{Rules: rules, Mode: v.Mode, Duration: v.Duration, From: from, Days: days, Now: now, Blocked: blocked})
		if ok && (!found || s.Start.Before(best.Start)) {
			best, bestMode, found = s, v.Mode, true
		}
	}
	return best, bestMode, found
}
