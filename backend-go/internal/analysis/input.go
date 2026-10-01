package analysis

import (
	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Input is everything a report reads. Days should cover Range plus the lookback the report needs
// (LookbackDays before Range.From; the monthly report loads the previous month too).
type Input struct {
	Today civildate.Date
	Range Range
	// Histories are the cycle_histories rows (any order); Profile the engine profile (nil = none).
	Histories []model.History
	Profile   *model.Profile
	// HeightCM is user_profiles.height (0 = unknown); Birthday the zero Date when unknown.
	HeightCM int
	Birthday civildate.Date
	Days     map[civildate.Date]*Day
	// DeepAnalysis is the plus.deep_analysis entitlement (Plus or a running trial).
	DeepAnalysis bool
	// Copy renders the localized sentences (nil → keys only, text "").
	Copy *Copy
}

// LookbackDays is how far before Range.From the handlers load day logs: a 30-day delta of a 7-day
// average needs 36 days of history before the window's first day.
const LookbackDays = 40

// Age is the user's age today (0 when the birthday is unknown).
func (in *Input) Age() int {
	if in.Birthday.IsZero() {
		return 0
	}
	return in.Birthday.AgeOn(in.Today)
}

func (in *Input) day(d civildate.Date) *Day { return in.Days[d] }

// Cycle is one cycle of the confirmed history.
type Cycle struct {
	Start civildate.Date
	// Length is the completed cycle's length (start to the next start); 0 for the current cycle.
	Length int
	// PeriodDays is the logged bleed length of a closed period (0 when open or implausible).
	PeriodDays int
	Current    bool
}

// End is the cycle's last day (the day before the next start); for the current cycle the request day.
func (c Cycle) End(today civildate.Date) civildate.Date {
	if c.Current {
		return today
	}
	return c.Start.AddDays(c.Length - 1)
}

// cycles are the confirmed cycles oldest first; the last one is the current cycle (when any).
func (in *Input) cycles() []Cycle {
	periods := insights.ConfirmedStarts(in.Histories, in.Today)
	out := make([]Cycle, 0, len(periods))
	for i, p := range periods {
		c := Cycle{Start: p.PeriodStart, Current: i == len(periods)-1}
		if !c.Current {
			c.Length = p.PeriodStart.DiffDays(periods[i+1].PeriodStart)
		}
		if d := insights.ClosedPeriodDays(p); insights.ValidPeriodLength(d) {
			c.PeriodDays = d
		}
		out = append(out, c)
	}
	return out
}

// windowCycles are the completed cycles that started inside the range, oldest first.
func (in *Input) windowCycles() []Cycle {
	var out []Cycle
	for _, c := range in.cycles() {
		if !c.Current && !c.Start.Before(in.Range.From) {
			out = append(out, c)
		}
	}
	return out
}

// currentCycle is the running cycle (ok false without any confirmed start).
func (in *Input) currentCycle() (Cycle, bool) {
	all := in.cycles()
	if len(all) == 0 {
		return Cycle{}, false
	}
	return all[len(all)-1], true
}

// typical is the typical cycle and period length: the medians of the range's plausible completed
// cycles, falling back to the engine's effective lengths.
func (in *Input) typical() (cycleLen, periodLen int) {
	m := metrics.Calculate(in.Histories, in.Profile)
	cycleLen, periodLen = m.EffectiveCycleLength, max(1, m.EffectivePeriodDuration)
	var lens, bleeds []int
	for _, c := range in.windowCycles() {
		if insights.ValidCycleLength(c.Length) {
			lens = append(lens, c.Length)
		}
		if c.PeriodDays > 0 {
			bleeds = append(bleeds, c.PeriodDays)
		}
	}
	if v := insights.Median(lens); v != nil {
		cycleLen = *v
	}
	if v := insights.Median(bleeds); v != nil {
		periodLen = *v
	}
	return cycleLen, periodLen
}

// Layout is where the phases fall in a cycle of Length days (1-based cycle days, inclusive).
type Layout struct {
	Length, PeriodDays          int
	FertileStart, FertileEnd    int // the 6-day window ending on the ovulation day (clipped after the period)
	OvulationDay, LutealDays    int
	FollicularDays, FertileDays int
}

// NewLayout splits a cycle: period 1..P, ovulation = next start − 14 (day L − 13), fertile window =
// ovulation and the 5 days before (never inside the period), luteal = after ovulation to the end.
func NewLayout(length, periodDays int) Layout {
	l := Layout{Length: length, PeriodDays: min(max(1, periodDays), length)}
	l.OvulationDay = max(l.PeriodDays+1, length-insights.LutealDays+1)
	l.OvulationDay = min(l.OvulationDay, length)
	l.FertileEnd = l.OvulationDay
	l.FertileStart = max(l.PeriodDays+1, l.OvulationDay-fertileDaysBeforeOvulation)
	if l.FertileStart > l.FertileEnd {
		l.FertileStart = l.FertileEnd
	}
	l.FertileDays = l.FertileEnd - l.FertileStart + 1
	l.FollicularDays = l.FertileStart - l.PeriodDays - 1
	l.LutealDays = length - l.FertileEnd
	return l
}

// Phase of 1-based cycle day n.
func (l Layout) Phase(n int) string {
	switch {
	case n <= l.PeriodDays:
		return PhasePeriod
	case n < l.FertileStart:
		return PhaseFollicular
	case n <= l.FertileEnd:
		return PhaseFertile
	}
	return PhaseLuteal
}

// phaseDays maps every day of the range that lies in a known cycle to its phase. Completed cycles
// use their own length and logged bleed; the current cycle the typical lengths (days past the
// typical length stay unassigned).
func (in *Input) phaseDays() map[civildate.Date]string {
	typCycle, typPeriod := in.typical()
	out := map[civildate.Date]string{}
	for _, c := range in.cycles() {
		length, bleed := c.Length, c.PeriodDays
		if c.Current {
			length = typCycle
		} else if !insights.ValidCycleLength(length) {
			continue // a gap (missed log), not a cycle
		}
		if bleed == 0 {
			bleed = typPeriod
		}
		l := NewLayout(length, bleed)
		for n := 1; n <= length; n++ {
			d := c.Start.AddDays(n - 1)
			if d.After(in.Today) {
				break
			}
			if in.Range.Contains(d) {
				out[d] = l.Phase(n)
			}
		}
	}
	return out
}
