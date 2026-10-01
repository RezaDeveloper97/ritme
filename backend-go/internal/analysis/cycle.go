package analysis

import (
	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Status codes of the FIGO checks.
const (
	StatusNormal     = "normal"
	StatusFrequent   = "frequent"   // median cycle < 24 days
	StatusInfrequent = "infrequent" // median cycle > 38 days
	StatusProlonged  = "prolonged"  // median period > 8 days

	RegularityRegular       = "regular"
	RegularityIrregular     = "irregular"
	RegularityNotEnoughData = "not_enough_data"
)

// CycleRow is one completed cycle of the window.
type CycleRow struct {
	Start, End civildate.Date
	Length     int
	PeriodDays int // 0 = no closed, plausible bleed logged
	InFIGO     bool
	// Counted: feeds the median and the variation. Excluded is why not: "implausible" (outside 21–45 d —
	// usually a missed log) or "outlier" (the window's single cycle further than the FIGO variation
	// limit from the median of the others, An_Cycle's «… در میانه لحاظ نشده است»).
	Counted  bool
	Excluded string
}

// Exclusion reasons.
const (
	ExcludedImplausible = "implausible"
	ExcludedOutlier     = "outlier"
)

// markOutlier excludes a lone outlier: when exactly one counted cycle lies further than limit days from
// the median of the other counted cycles, it is shown but left out of the median and the variation.
// Two or more such cycles are the user's real variability and all stay counted.
func markOutlier(rows []CycleRow, limit int) {
	var counted []int
	for i, r := range rows {
		if r.Counted {
			counted = append(counted, i)
		}
	}
	if len(counted) < MinPatternCycles+1 { // the rest must still form a pattern
		return
	}
	outlier := -1
	for _, i := range counted {
		var others []int
		for _, j := range counted {
			if j != i {
				others = append(others, rows[j].Length)
			}
		}
		m := insights.Median(others)
		if d := rows[i].Length - *m; d > limit || -d > limit {
			if outlier >= 0 {
				return // more than one: real variability
			}
			outlier = i
		}
	}
	if outlier >= 0 {
		rows[outlier].Counted, rows[outlier].Excluded = false, ExcludedOutlier
	}
}

// CycleReport is GET /analysis/cycle (and the hub's cycle card).
type CycleReport struct {
	BasedOn                   int
	MedianCycle, MedianPeriod *int
	CycleStatus, PeriodStatus string // "" without a median
	// Variation is shortest-to-longest of the counted lengths (nil below MinPatternCycles).
	Variation      *int
	VariationLimit int
	Regularity     string
	Applies        bool
	Typical        Layout
	Current        *Cycle
	CurrentDay     int
	Cycles         []CycleRow // newest first
}

// Ready: the per-cycle bars are a trend (≥ MinTrendPoints cycles).
func (r CycleReport) Ready() bool { return r.BasedOn >= MinTrendPoints }

// BuildCycle computes the cycle report of the range.
func BuildCycle(in *Input) CycleReport {
	age := in.Age()
	out := CycleReport{
		VariationLimit: FIGOVariationLimit(age), Applies: FIGOApplies(age), Regularity: RegularityNotEnoughData,
		Cycles: []CycleRow{},
	}
	window := in.windowCycles()
	out.BasedOn = len(window)
	var lens, bleeds []int
	for i := len(window) - 1; i >= 0; i-- {
		c := window[i]
		row := CycleRow{
			Start: c.Start, End: c.End(in.Today), Length: c.Length, PeriodDays: c.PeriodDays,
			InFIGO: c.Length >= FIGOCycleMin && c.Length <= FIGOCycleMax, Counted: insights.ValidCycleLength(c.Length),
		}
		if !row.Counted {
			row.Excluded = ExcludedImplausible
		}
		if c.PeriodDays > 0 {
			bleeds = append(bleeds, c.PeriodDays)
		}
		out.Cycles = append(out.Cycles, row)
	}
	markOutlier(out.Cycles, out.VariationLimit)
	for _, row := range out.Cycles {
		if row.Counted {
			lens = append(lens, row.Length)
		}
	}
	if cur, ok := in.currentCycle(); ok {
		out.Current = &cur
		out.CurrentDay = cur.Start.DiffDays(in.Today) + 1
		if cur.PeriodDays > 0 && !cur.Start.Before(in.Range.From) {
			bleeds = append(bleeds, cur.PeriodDays)
		}
	}
	out.MedianCycle = insights.Median(lens)
	out.MedianPeriod = insights.Median(bleeds)
	if m := out.MedianCycle; m != nil {
		switch {
		case *m < FIGOCycleMin:
			out.CycleStatus = StatusFrequent
		case *m > FIGOCycleMax:
			out.CycleStatus = StatusInfrequent
		default:
			out.CycleStatus = StatusNormal
		}
	}
	if m := out.MedianPeriod; m != nil {
		out.PeriodStatus = StatusNormal
		if *m > FIGOPeriodMax {
			out.PeriodStatus = StatusProlonged
		}
	}
	if len(lens) >= MinPatternCycles {
		lo, hi := lens[0], lens[0]
		for _, n := range lens {
			lo, hi = min(lo, n), max(hi, n)
		}
		out.Variation = intPtr(hi - lo)
		out.Regularity = RegularityRegular
		if hi-lo > out.VariationLimit {
			out.Regularity = RegularityIrregular
		}
	}
	typCycle, typPeriod := in.typical()
	out.Typical = NewLayout(typCycle, typPeriod)
	return out
}

func statusOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func positiveOrNil(n int) any {
	if n <= 0 {
		return nil
	}
	return n
}

// FIGOJSON is the reference ranges block.
func FIGOJSON(limit int, applies bool) *jsonx.OrderedMap {
	return jsonx.Obj(
		"applies", applies,
		"cycle", jsonx.Obj("min", FIGOCycleMin, "max", FIGOCycleMax),
		"period", jsonx.Obj("max", FIGOPeriodMax),
		"variation", jsonx.Obj("max", limit),
	)
}

// LayoutJSON is a typical cycle's phase split.
func (l Layout) JSON() *jsonx.OrderedMap {
	return jsonx.Obj(
		"cycle_length", l.Length,
		"ovulation_day", l.OvulationDay,
		"fertile_start_day", l.FertileStart,
		"fertile_end_day", l.FertileEnd,
		"days", jsonx.Obj(
			PhasePeriod, l.PeriodDays, PhaseFollicular, l.FollicularDays, PhaseFertile, l.FertileDays,
			PhaseLuteal, l.LutealDays,
		),
	)
}

func (r CycleReport) rowsJSON() []*jsonx.OrderedMap {
	rows := make([]*jsonx.OrderedMap, 0, len(r.Cycles))
	for _, c := range r.Cycles {
		rows = append(rows, jsonx.Obj(
			"start", c.Start.String(), "end", c.End.String(), "length", c.Length,
			"period_days", positiveOrNil(c.PeriodDays), "in_figo_range", c.InFIGO, "counted", c.Counted,
			"excluded", statusOrNil(c.Excluded),
		))
	}
	return rows
}

func (r CycleReport) currentJSON() any {
	if r.Current == nil {
		return nil
	}
	return jsonx.Obj("start", r.Current.Start.String(), "day", r.CurrentDay)
}

// JSON is the `data` of GET /analysis/cycle (range added by the handler).
func (r CycleReport) JSON() *jsonx.OrderedMap {
	return jsonx.Obj(
		"ready", r.Ready(),
		"based_on_cycles", r.BasedOn,
		"figo", FIGOJSON(r.VariationLimit, r.Applies),
		"cycle_length", jsonx.Obj("median", ptrOrNil(r.MedianCycle), "status", statusOrNil(r.CycleStatus)),
		"period_length", jsonx.Obj("median", ptrOrNil(r.MedianPeriod), "status", statusOrNil(r.PeriodStatus)),
		"variation", jsonx.Obj(
			"days", ptrOrNil(r.Variation), "max", r.VariationLimit, "status", r.Regularity,
			"cycles_needed", MinPatternCycles,
		),
		"typical", r.Typical.JSON(),
		"current", r.currentJSON(),
		"cycles", r.rowsJSON(),
	)
}
