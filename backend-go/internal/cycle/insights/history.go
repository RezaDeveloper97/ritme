// Package insights holds the Night & Bloom cycle read models (B-N1-08, Go only): the cycle history
// screen (GET /cycle/history) and the per-symptom pattern over the typical cycle
// (GET /cycle/symptom-pattern). Both are pure functions of the engine inputs (cycle_histories,
// profile, daily_health_logs); the handlers only load a snapshot and serialise the result.
//
// No second prediction model: the "predicted" length is the cycle metrics' effective length, the
// same number /cycle/today and /home/cycle-overview show. What is new here is the descriptive
// summary of the last few cycles (median, spread, regularity verdict), which only this screen needs.
package insights

import (
	"slices"
	"sort"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

const (
	// HistoryWindow is how many completed cycles the history screen summarises («۶ سیکل اخیر»).
	HistoryWindow = 6
	// validCycleMin/Max bound a plausible cycle length (the metrics' 21–45); outside it a gap
	// usually means a missed log, so it never feeds the median.
	validCycleMin = 21
	validCycleMax = 45
	// validPeriodMin/Max bound a plausible bleed length (the metrics' 2–10).
	validPeriodMin = 2
	validPeriodMax = 10
	// minSpreadDays is the narrowest band around the median: a couple of days either way is ordinary
	// cycle-to-cycle variation, so a very steady history still gets a ±2 band.
	minSpreadDays = 2
	// maxRegularSpread: a wider typical band (more than ±4, i.e. a > 8-day range) reads irregular.
	maxRegularSpread = 4
)

// Regularity verdicts.
const (
	RegularityRegular       = "regular"
	RegularityIrregular     = "irregular"
	RegularityNotEnoughData = "not_enough_data"
)

// CycleRow is one bar of the history list, newest first.
type CycleRow struct {
	Start civildate.Date
	// End is the cycle's last day (the day before the next start); zero for the current cycle.
	End civildate.Date
	// PeriodEnd is the logged last bleeding day; zero while the period is open.
	PeriodEnd civildate.Date
	// PeriodDays is the bleed length: end−start+1 when closed; for an open period the days so far,
	// capped at the effective period length.
	PeriodDays    int
	PeriodOngoing bool
	// Length is the completed cycle's length, or the days so far (today inclusive) for the current one.
	Length    int
	IsCurrent bool
	// InRange says whether a completed cycle sits inside the median ± spread band (nil when there
	// is no band yet, and always nil for the current cycle).
	InRange *bool
}

// History is the response of GET /cycle/history.
type History struct {
	Date civildate.Date
	// BasedOn counts the completed cycles in the window (≤ HistoryWindow).
	BasedOn int
	// MedianCycle is the median of the window's valid (21–45) lengths; nil without one.
	MedianCycle *int
	// Spread is the ± band around MedianCycle: max(2, round(median absolute deviation)).
	Spread *int
	// MedianPeriod is the median of the closed, plausible (2–10) bleed lengths in the window.
	MedianPeriod *int
	// Predicted is the metrics' effective cycle length (what the predictions use).
	Predicted int
	// InRange / Total: completed cycles inside the band, out of the window.
	InRange, Total int
	Regularity     string
	Cycles         []CycleRow
}

// confirmedStarts returns the user-confirmed, non-estimated periods that started on or before
// today, oldest first (a later row with the same start wins nothing: starts are unique per user).
func confirmedStarts(histories []model.History, today civildate.Date) []model.History {
	out := make([]model.History, 0, len(histories))
	for _, h := range histories {
		if h.IsConfirmed && !h.IsEstimated && !h.PeriodStart.After(today) {
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PeriodStart.Before(out[j].PeriodStart) })
	return out
}

func validLength(n int) bool { return n >= validCycleMin && n <= validCycleMax }

// closedPeriodDays is end−start+1 of a closed period (0 when open or inconsistent).
func closedPeriodDays(h model.History) int {
	if !h.HasEnd() || h.PeriodEnd.Before(h.PeriodStart) {
		return 0
	}
	return h.PeriodStart.DiffDays(h.PeriodEnd) + 1
}

// BuildHistory computes the history screen from the engine inputs.
func BuildHistory(histories []model.History, profile *model.Profile, today civildate.Date) History {
	m := metrics.Calculate(histories, profile)
	periods := confirmedStarts(histories, today)
	out := History{Date: today, Predicted: m.EffectiveCycleLength, Regularity: RegularityNotEnoughData, Cycles: []CycleRow{}}
	if len(periods) == 0 {
		return out
	}

	// Completed cycles: consecutive starts; keep the newest HistoryWindow of them.
	type completed struct {
		period model.History
		length int
	}
	done := make([]completed, 0, len(periods))
	for i := 0; i+1 < len(periods); i++ {
		done = append(done, completed{periods[i], periods[i].PeriodStart.DiffDays(periods[i+1].PeriodStart)})
	}
	if len(done) > HistoryWindow {
		done = done[len(done)-HistoryWindow:]
	}

	var lengths, bleeds []int
	for _, c := range done {
		if validLength(c.length) {
			lengths = append(lengths, c.length)
		}
		if d := closedPeriodDays(c.period); d >= validPeriodMin && d <= validPeriodMax {
			bleeds = append(bleeds, d)
		}
	}
	current := periods[len(periods)-1]
	if d := closedPeriodDays(current); d >= validPeriodMin && d <= validPeriodMax {
		bleeds = append(bleeds, d)
	}

	out.BasedOn = len(done)
	out.Total = len(done)
	out.MedianCycle = median(lengths)
	out.MedianPeriod = median(bleeds)
	var lo, hi int
	if out.MedianCycle != nil {
		s := max(minSpreadDays, int(phpround.Round(mad(lengths, *out.MedianCycle), 0)))
		out.Spread = &s
		lo, hi = *out.MedianCycle-s, *out.MedianCycle+s
	}

	// Current cycle first, then the completed ones newest → oldest.
	daysSoFar := current.PeriodStart.DiffDays(today) + 1
	cur := CycleRow{Start: current.PeriodStart, Length: daysSoFar, IsCurrent: true}
	if d := closedPeriodDays(current); d > 0 {
		cur.PeriodEnd, cur.PeriodDays = current.PeriodEnd, d
	} else {
		cur.PeriodOngoing = true
		cur.PeriodDays = min(daysSoFar, max(1, m.EffectivePeriodDuration))
	}
	out.Cycles = append(out.Cycles, cur)

	for i := len(done) - 1; i >= 0; i-- {
		c := done[i]
		row := CycleRow{Start: c.period.PeriodStart, End: c.period.PeriodStart.AddDays(c.length - 1), Length: c.length}
		if d := closedPeriodDays(c.period); d > 0 {
			row.PeriodEnd, row.PeriodDays = c.period.PeriodEnd, min(d, c.length)
		} else {
			// A past period with no logged end: show the usual bleed length.
			row.PeriodDays = min(c.length, max(1, m.EffectivePeriodDuration))
		}
		if out.MedianCycle != nil {
			in := c.length >= lo && c.length <= hi
			row.InRange = &in
			if in {
				out.InRange++
			}
		}
		out.Cycles = append(out.Cycles, row)
	}

	if len(lengths) >= 2 {
		out.Regularity = RegularityIrregular
		if out.Total-out.InRange <= 1 && *out.Spread <= maxRegularSpread {
			out.Regularity = RegularityRegular
		}
	}
	return out
}

// median of values (nil for none); an even count averages the middle pair with PHP round().
func median(values []int) *int {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	m := sorted[mid]
	if len(sorted)%2 == 0 {
		m = int(phpround.Round(float64(sorted[mid-1]+sorted[mid])/2, 0))
	}
	return &m
}

// mad is the median absolute deviation of values around center.
func mad(values []int, center int) float64 {
	dev := make([]float64, len(values))
	for i, v := range values {
		d := float64(v - center)
		if d < 0 {
			d = -d
		}
		dev[i] = d
	}
	slices.Sort(dev)
	mid := len(dev) / 2
	if len(dev)%2 == 0 {
		return (dev[mid-1] + dev[mid]) / 2
	}
	return dev[mid]
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func intOrNil(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

// JSON is the `data` object of GET /cycle/history.
func (h History) JSON() *jsonx.OrderedMap {
	var band any
	if h.MedianCycle != nil && h.Spread != nil {
		band = jsonx.Obj("min", *h.MedianCycle-*h.Spread, "max", *h.MedianCycle+*h.Spread)
	}
	cycles := make([]*jsonx.OrderedMap, 0, len(h.Cycles))
	for _, c := range h.Cycles {
		var in any
		if c.InRange != nil {
			in = *c.InRange
		}
		cycles = append(cycles, jsonx.Obj(
			"start", c.Start.String(),
			"end", dateOrNil(c.End),
			"period_end", dateOrNil(c.PeriodEnd),
			"period_days", c.PeriodDays,
			"period_ongoing", c.PeriodOngoing,
			"length", c.Length,
			"is_current", c.IsCurrent,
			"in_range", in,
		))
	}
	return jsonx.Obj(
		"date", h.Date.String(),
		"based_on_cycles", h.BasedOn,
		"cycle_length", jsonx.Obj(
			"median", intOrNil(h.MedianCycle),
			"spread_days", intOrNil(h.Spread),
			"range", band,
			"predicted", h.Predicted,
		),
		"period_length", jsonx.Obj("median", intOrNil(h.MedianPeriod)),
		"regularity", jsonx.Obj("status", h.Regularity, "in_range", h.InRange, "total", h.Total),
		"cycles", cycles,
	)
}
