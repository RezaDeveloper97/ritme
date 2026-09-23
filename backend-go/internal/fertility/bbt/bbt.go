// Package bbt is the basal-body-temperature engine of the TTC chart (docs/fertility-ttc/README.md,
// T-M5-02): per-cycle series by cycle day, the coverline and the 3-over-6 shift rule, the phase and
// the chart stats. Pure: no DB, no clock (today is a parameter), no HTTP. Temperatures are whole
// hundredths of a degree (decimal(4,2) → 3655), so the comparisons carry no float noise.
package bbt

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Rule constants (README → BBT shift rule).
const (
	// BaselineReadings is the "6" of 3-over-6: the readings the coverline is the maximum of, and
	// the readings the pre-ovulation average is taken over.
	BaselineReadings = 6
	// HighReadings is the "3": consecutive readings above the coverline that confirm a shift.
	HighReadings = 3
	// MinRise is how far (hundredths °C) the first high reading must clear the 6 before it.
	MinRise = 20
)

// Phase of a cycle's chart.
type Phase string

// Phases.
const (
	PhasePreShift  Phase = "pre_shift"
	PhasePostShift Phase = "post_shift"
)

// Reading is one logged temperature.
type Reading struct {
	Date  civildate.Date
	Value int // hundredths °C
}

// DayRange is an inclusive range of cycle days.
type DayRange struct {
	FromDay, ToDay int
}

// CycleInput is one cycle's boundaries, from the cycle engine's period history.
type CycleInput struct {
	Start civildate.Date // first day (cycle day 1)
	// End is the cycle's last day: the day before the next period start, or for the current cycle
	// the end of its predicted span. Readings after it belong to the next cycle.
	End civildate.Date
	// FertileWindow is the cycle engine's fertile window in cycle days; nil = none.
	FertileWindow *DayRange
}

// Point is a reading placed on its cycle.
type Point struct {
	CycleDay int
	Date     civildate.Date
	Value    int
}

// Stats are the chart's stat cards.
type Stats struct {
	// PreOvulationAvg is the mean of the first BaselineReadings readings (fewer when fewer exist,
	// rounded half away from zero); nil without a reading.
	PreOvulationAvg *int
	LoggedDays      int // readings in the cycle up to today
	CycleDaysSoFar  int // days from the cycle start up to today (or the cycle end), inclusive
	Gaps            int // CycleDaysSoFar − LoggedDays
}

// Cycle is one analysed cycle.
type Cycle struct {
	Start         civildate.Date
	Points        []Point // by cycle day
	Coverline     *int    // nil = fewer than BaselineReadings readings
	FertileWindow *DayRange
	ShiftDay      *int // cycle day of the first high reading of a confirmed shift
	Phase         Phase
	Stats         Stats
}

// OvulationDay is the estimated ovulation cycle day (the day before the first high reading) of a
// confirmed shift; nil without one.
func (c Cycle) OvulationDay() *int {
	if c.ShiftDay == nil {
		return nil
	}
	d := *c.ShiftDay - 1
	return &d
}

// Analyze places the readings that fall in the cycle (up to today) on it and runs the shift rule.
// readings may be in any order and may include other cycles' readings.
func Analyze(in CycleInput, readings []Reading, today civildate.Date) Cycle {
	last := in.End
	if today.Before(last) {
		last = today
	}
	c := Cycle{Start: in.Start, Points: []Point{}, FertileWindow: in.FertileWindow, Phase: PhasePreShift}
	if !last.Before(in.Start) {
		c.Stats.CycleDaysSoFar = in.Start.DiffDays(last) + 1
	}
	byDay := map[int]Point{}
	for _, r := range readings {
		if r.Date.Before(in.Start) || r.Date.After(last) {
			continue
		}
		day := in.Start.DiffDays(r.Date) + 1
		byDay[day] = Point{CycleDay: day, Date: r.Date, Value: r.Value}
	}
	for day := 1; day <= c.Stats.CycleDaysSoFar; day++ {
		if p, ok := byDay[day]; ok {
			c.Points = append(c.Points, p)
		}
	}
	c.Stats.LoggedDays = len(c.Points)
	c.Stats.Gaps = c.Stats.CycleDaysSoFar - c.Stats.LoggedDays
	c.Stats.PreOvulationAvg = average(c.Points[:min(BaselineReadings, len(c.Points))])

	coverline, shift := detectShift(c.Points)
	c.Coverline = coverline
	if shift >= 0 {
		day := c.Points[shift].CycleDay
		c.ShiftDay = &day
		c.Phase = PhasePostShift
	}
	return c
}

// detectShift is the 3-over-6 rule over the readings in order (days without a reading are
// skipped, not counted): the first reading at least MinRise above the maximum of the
// BaselineReadings readings before it starts a shift when it and the next HighReadings−1
// readings all stay above that maximum (the coverline). It returns the coverline and the index of
// the first high reading (−1 = no confirmed shift). Without a confirmed shift the coverline is
// provisional: the baseline of a started-but-unconfirmed rise, else the maximum of the latest
// BaselineReadings readings; nil with fewer readings.
func detectShift(points []Point) (*int, int) {
	n := len(points)
	if n < BaselineReadings {
		return nil, -1
	}
	pending := -1
	for i := BaselineReadings; i < n; i++ {
		cover := maxValue(points[i-BaselineReadings : i])
		if points[i].Value < cover+MinRise {
			continue
		}
		if i+HighReadings-1 >= n { // the rise has started but has too few readings yet
			if pending < 0 {
				pending = cover
			}
			continue
		}
		confirmed := true
		for j := i + 1; j < i+HighReadings; j++ {
			if points[j].Value <= cover {
				confirmed = false
				break
			}
		}
		if confirmed {
			return &cover, i
		}
	}
	if pending >= 0 {
		return &pending, -1
	}
	cover := maxValue(points[n-BaselineReadings:])
	return &cover, -1
}

func maxValue(points []Point) int {
	m := points[0].Value
	for _, p := range points[1:] {
		m = max(m, p.Value)
	}
	return m
}

// average is the mean of the points' values rounded half away from zero (values are positive).
func average(points []Point) *int {
	if len(points) == 0 {
		return nil
	}
	sum := 0
	for _, p := range points {
		sum += p.Value
	}
	n := len(points)
	avg := (2*sum + n) / (2 * n)
	return &avg
}

// Result is the chart: the cycles newest first (the current one first) and the confirmed shift
// days of the previous cycles.
type Result struct {
	Cycles []Cycle
	// PastShiftDays are the shift days of the previous cycles that have a confirmed shift, newest
	// cycle first.
	PastShiftDays []int
}

// Build analyses cycles (newest first; cycles[0] is the current one). rangeSize limits the
// returned Cycles; PastShiftDays always looks at every previous cycle given, so the tip does not
// change with the range tab.
func Build(cycles []CycleInput, readings []Reading, today civildate.Date, rangeSize int) Result {
	res := Result{Cycles: []Cycle{}, PastShiftDays: []int{}}
	for i, in := range cycles {
		c := Analyze(in, readings, today)
		if i < rangeSize {
			res.Cycles = append(res.Cycles, c)
		}
		if i > 0 && c.ShiftDay != nil {
			res.PastShiftDays = append(res.PastShiftDays, *c.ShiftDay)
		}
	}
	return res
}

// ParseValue reads a decimal(4,2) temperature ("36.55", "36.5", "36") as hundredths.
func ParseValue(s string) (int, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ".")
	if len(frac) > 2 {
		return 0, fmt.Errorf("bbt: %q has more than 2 decimals", s)
	}
	frac += strings.Repeat("0", 2-len(frac))
	w, err := strconv.Atoi(whole)
	if err != nil || w < 0 {
		return 0, fmt.Errorf("bbt: bad temperature %q", s)
	}
	f, err := strconv.Atoi(frac)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("bbt: bad temperature %q", s)
	}
	return w*100 + f, nil
}

// Format writes hundredths as a decimal(4,2) string (3655 → "36.55"), like the column.
func Format(v int) string { return fmt.Sprintf("%d.%02d", v/100, v%100) }

// Boundaries are the cycles of the chart, newest first: the current cycle (currentStart to
// currentEnd) and up to count−1 previous ones, each running from a period start in starts (any
// order, duplicates and starts on or after currentStart ignored) to the day before the next
// start. Fertile windows are left for the caller (the cycle engine).
func Boundaries(currentStart, currentEnd civildate.Date, starts []civildate.Date, count int) []CycleInput {
	if count < 1 || currentStart.IsZero() {
		return []CycleInput{}
	}
	out := []CycleInput{{Start: currentStart, End: currentEnd}}
	prev := slices.Clone(starts)
	slices.SortFunc(prev, func(a, b civildate.Date) int { return b.Compare(a) })
	next := currentStart
	for _, s := range slices.Compact(prev) {
		if len(out) == count {
			break
		}
		if !s.Before(next) {
			continue
		}
		out = append(out, CycleInput{Start: s, End: next.AddDays(-1)})
		next = s
	}
	return out
}
