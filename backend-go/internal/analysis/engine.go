package analysis

// The analysis tab on the cycle engine v6 (T-M9-35, D-129, A-01, E-09), under CYCLE_ENGINE=shadow|v6 (Input.Engine
// set; nil keeps v1 byte-identical):
//
//   - cycles: the engine's real periods (never the onboarding date or an estimated row) up to today; a cycle that
//     holds a recorded skip (E-15) is shown but never counted.
//   - regularity: the verdict is the engine's cycle_regularity (§2.27, the same value as cycle_view.metrics) —
//     regular | irregular | insufficient_data. The FIGO band is the reference only (A-01). The variation shown next
//     to it is max − min of the engine's own regularity cycles (real complete cycles that started in the
//     RegularityMonths before today, skips excluded, outliers included — E-09, so no markOutlier) against the band of
//     the user's age; when that ever disagrees with the verdict the days are left out rather than contradict it.
//   - typical(): window medians as before, falling back to the engine's effective lengths.
//   - TTC: each cycle's ovulation and source are the engine's (BBT > LH > calendar, the LH/BBT conflict → BBT); the
//     chart uses the engine's BBT rule.

import (
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ExcludedSkipped marks a v6 cycle row that holds a recorded skip (E-15): not a cycle length.
const ExcludedSkipped = "skipped"

// engineCycles are Input.cycles() under v6: the engine's real periods started on or before today, oldest first.
func (in *Input) engineCycles() []Cycle {
	st := in.Engine
	t := engine.DayOf(in.Today)
	periods := st.Periods()
	var skipped map[engine.Day]bool
	for _, c := range st.Cycles() {
		if c.Skipped {
			if skipped == nil {
				skipped = map[engine.Day]bool{}
			}
			skipped[c.Start] = true
		}
	}
	out := make([]Cycle, 0, len(periods))
	for i, p := range periods {
		if p.Start > t {
			break
		}
		c := Cycle{Start: p.Start.Civil(), Current: i == len(periods)-1 || periods[i+1].Start > t, Skipped: skipped[p.Start]}
		if !c.Current {
			c.Length = int(periods[i+1].Start - p.Start)
		}
		if !p.Open() && p.End >= p.Start {
			if d := int(p.End-p.Start) + 1; insights.ValidPeriodLength(d) {
				c.PeriodDays = d
			}
		}
		out = append(out, c)
		if c.Current {
			break
		}
	}
	return out
}

// engineVerdict is the engine's cycle_regularity at ref (resolved as of the report day).
func (in *Input) engineVerdict(ref civildate.Date) engine.CycleRegularity {
	d := engine.DayOf(ref)
	return in.Engine.Resolve(d, d).Metrics.CycleRegularity
}

// monthsBack is d minus n calendar months, the day clamped to the month's length (the engine's window rule).
func monthsBack(d civildate.Date, n int) civildate.Date {
	total := d.Year*12 + int(d.Month) - 1 - n
	y, m := total/12, time.Month(total%12+1)
	last := civildate.Date{Year: y, Month: m + 1, Day: 1}
	if m == time.December {
		last = civildate.Date{Year: y + 1, Month: time.January, Day: 1}
	}
	return civildate.Date{Year: y, Month: m, Day: min(d.Day, last.AddDays(-1).Day)}
}

// engineVariation is the variation of the engine's regularity cycles at ref and the FIGO band of the age at ref:
// days nil below the engine's minimum cycle count.
func (in *Input) engineVariation(ref civildate.Date) (days *int, limit int) {
	st := in.Engine
	p := st.Params()
	limit = p.FIGO.RegularityAdult
	switch age := in.ageOn(ref); {
	case age > 0 && age < p.RegularityAdultFromAge:
		limit = p.FIGO.RegularityYoung
	case age >= p.RegularityLateFromAge:
		limit = p.FIGO.RegularityLate
	case age == 0 && st.Mode() == engine.ModeTeen:
		limit = p.FIGO.RegularityYoung
	}
	from, to := engine.DayOf(monthsBack(ref, p.RegularityMonths)), engine.DayOf(ref)
	n, lo, hi := 0, 0, 0
	for _, c := range st.Cycles() {
		if c.Skipped || c.Start < from || c.NextStart > to {
			continue
		}
		if n == 0 {
			lo, hi = c.Length, c.Length
		}
		lo, hi = min(lo, c.Length), max(hi, c.Length)
		n++
	}
	if n < p.RegularityMinCycles {
		return nil, limit
	}
	v := hi - lo
	return &v, limit
}

// ageOn is the user's age on d (0 when the birthday is unknown).
func (in *Input) ageOn(d civildate.Date) int {
	if in.Birthday.IsZero() {
		return 0
	}
	return in.Birthday.AgeOn(d)
}

// engineRegularity is the v6 regularity block at ref: the engine's verdict, the variation (nil when it would
// contradict the verdict or there are too few cycles) and the band limit.
func (in *Input) engineRegularity(ref civildate.Date) (verdict string, variation *int, limit int) {
	v := in.engineVerdict(ref)
	variation, limit = in.engineVariation(ref)
	if variation != nil {
		irregular := *variation > limit
		if v == engine.RegularityInsufficientData || irregular != (v == engine.RegularityIrregular) {
			variation = nil
		}
	}
	return v.String(), variation, limit
}

// engineRule is the engine's BBT rule (nil under v1: the legacy chart rule).
func (in *Input) engineRule() bbt.Rule {
	if in.Engine == nil {
		return nil
	}
	p := in.Engine.Params().BBT
	return &p
}

// engineMarker is the engine's marker ovulation (BBT, else LH; §2.17) of cycle c within its first days days: the
// cycle day (0 = none) and source.
func (in *Input) engineMarker(c Cycle, days int) (int, string) {
	st := in.Engine
	start := engine.DayOf(c.Start)
	for i, p := range st.Periods() {
		if p.Start != start {
			continue
		}
		if o := st.Ovulations()[i]; o.Date.IsSet() {
			day := int(o.Date-start) + 1
			if day >= 1 && day <= days {
				if o.Source == engine.OvulationBBTConfirmed {
					return day, OvulationBBT
				}
				return day, OvulationLH
			}
		}
		break
	}
	return 0, ""
}

// engineEstimate is the engine's ovulation estimate of cycle c (resolved on its first day): the cycle day, 0 when
// none (no anchor, fertility suppressed, outside the cycle).
func (in *Input) engineEstimate(c Cycle) int {
	st := in.Engine
	start := engine.DayOf(c.Start)
	t := engine.DayOf(in.Today)
	s := st.Resolve(start, t)
	if s.Anchors.CurrentPeriodStart != start || !s.Anchors.EstimatedOvulationDate.IsSet() || s.Limits.FertilitySuppressed {
		return 0
	}
	day := int(s.Anchors.EstimatedOvulationDate-start) + 1
	if day < 1 || (!c.Current && day > c.Length) {
		return 0
	}
	return day
}
