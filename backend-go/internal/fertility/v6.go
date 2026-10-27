package fertility

// The /fertility reads on the cycle engine v6 (T-M9-35, D-128), under CYCLE_ENGINE=shadow|v6 (WithEngine; nil keeps
// the v1 path byte-identical):
//
//   - today / days: cycle day and chance from State.Resolve(date, local today).
//   - bbt: the chart's cycles are the engine's current anchor and its real periods (never the onboarding date or an
//     estimated row), the readings are the engine's own BBT input (State.BBT, no query), the shift is the engine's
//     rule (engine/bbt via bbt.BuildRule) and each cycle's fertile window comes from one State.Range over the cycle.
//   - insights: the same cycles and ranges; the fertile window is today's prediction card (§2.19) or, once today's
//     ovulation has passed, future_cycles[2]'s; the history's ovulation day and source are the engine's per-cycle
//     anchors (BBT > LH > calendar, the LH/BBT conflict resolved to BBT, §2.17); the cycles row reads the engine's
//     valid-cycle count, effective length, variability and cycle_regularity. No per-day resolve loop: at most one
//     Range per chart cycle plus one Resolve for today.

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// WithEngine makes the service read the v6 engine (states nil = CYCLE_ENGINE v1, unchanged).
func (s *Service) WithEngine(states state.UserStates) *Service {
	s.states = states
	return s
}

// Engine reports whether the service reads the v6 engine.
func (s *Service) Engine() bool { return s.states != nil }

// engineState is uid's state at the parameters of today, with a fresh revision read (Source.ForUser).
func (s *Service) engineState(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error) {
	st, err := s.states.ForUser(ctx, uid, today)
	if err != nil {
		return nil, fmt.Errorf("fertility: engine state: %w", err)
	}
	return st, nil
}

// view is one request's read of a prepared state on a user-local day. ranges / resolves count the engine calls (the
// insights budget test).
type view struct {
	st       *engine.State
	today    engine.Day
	ranges   int
	resolves int
}

func newView(st *engine.State, today civildate.Date) *view {
	return &view{st: st, today: engine.DayOf(today)}
}

func (v *view) resolve(d civildate.Date) engine.Status {
	v.resolves++
	return v.st.Resolve(engine.DayOf(d), v.today)
}

func (v *view) rangeOf(from, to civildate.Date) []engine.Status {
	v.ranges++
	return v.st.Range(engine.DayOf(from), engine.DayOf(to), v.today)
}

// contextV6 is the cycle day and chance of date from the engine.
func contextV6(v *view, date civildate.Date) Context {
	st := v.resolve(date)
	cx := Context{Chance: chanceV6(st.FertilityLevel)}
	if n, ok := st.CycleDay.Get(); ok {
		cx.CycleDay = &n
	}
	return cx
}

// chanceV6 maps the engine's fertility_level (§2.26) onto the card; unknown (no anchor, fertility suppressed) is the
// unknown card.
func chanceV6(level engine.FertilityLevel) Chance {
	var l string
	var bars int
	switch level {
	case engine.FertilityLow:
		l, bars = "low", 1
	case engine.FertilityMedium:
		l, bars = "medium", 3
	case engine.FertilityHigh:
		l, bars = "high", 4
	case engine.FertilityPeak:
		l, bars = "peak", 5
	default:
		return Chance{}
	}
	return Chance{Level: &l, Bars: bars}
}

// LoadV6 is Load on the engine state st (the merged-day query only).
func (s *Service) LoadV6(ctx context.Context, st *engine.State, userID uint64, date, today civildate.Date) (Day, Context, error) {
	row, err := s.mergedDay(ctx, userID, date)
	if err != nil {
		return Day{}, Context{}, err
	}
	return DayFromRow(date, row), contextV6(newView(st, today), date), nil
}

// chartCycle is one chart cycle with its engine statuses (one Range).
type chartCycle struct {
	in       bbt.CycleInput
	statuses []engine.Status
}

// chartCyclesV6 are the chart's cycles newest first: the current cycle runs from today's anchor to the day before
// the predicted next start (today when late); previous cycles run between the engine's real period starts. Each
// carries its fertile window (the days the engine shows as main_phase fertile, one Range per cycle, at most
// maxWindowScan days). None without an anchor.
func chartCyclesV6(v *view, today civildate.Date) (engine.Status, []chartCycle) {
	now := v.resolve(today)
	cur := now.Anchors.CurrentPeriodStart
	if !cur.IsSet() || cur > v.today {
		return now, nil
	}
	end := today
	if next := now.Anchors.PredictedNextPeriodStart; next.IsSet() && next-1 > v.today {
		end = (next - 1).Civil()
	}
	periods := v.st.Periods()
	starts := make([]civildate.Date, 0, len(periods))
	for _, p := range periods {
		starts = append(starts, p.Start.Civil())
	}
	bounds := bbt.Boundaries(cur.Civil(), end, starts, maxBBTCycles)
	out := make([]chartCycle, len(bounds))
	for i, b := range bounds {
		last := b.End
		if scanEnd := b.Start.AddDays(maxWindowScan - 1); scanEnd.Before(last) {
			last = scanEnd
		}
		out[i] = chartCycle{in: b, statuses: v.rangeOf(b.Start, last)}
		out[i].in.FertileWindow = fertileDays(out[i].statuses)
	}
	return now, out
}

// fertileDays is the cycle days (1-based from the first status) the engine shows as main_phase fertile; nil when none.
func fertileDays(statuses []engine.Status) *bbt.DayRange {
	var w *bbt.DayRange
	for i, s := range statuses {
		if s.MainPhase != engine.MainPhaseFertile {
			continue
		}
		if w == nil {
			w = &bbt.DayRange{FromDay: i + 1}
		}
		w.ToDay = i + 1
	}
	return w
}

// engineReadings are the engine's BBT readings (its input, State.BBT) as chart readings.
func engineReadings(st *engine.State) []bbt.Reading {
	rs := st.BBT()
	out := make([]bbt.Reading, 0, len(rs))
	for _, r := range rs {
		out = append(out, bbt.Reading{Date: r.Date.Civil(), Value: int(r.Centi)})
	}
	return out
}

// engineRule is the engine's BBT rule of the state's parameters.
func engineRule(st *engine.State) bbt.Rule {
	p := st.Params().BBT
	return &p
}

func inputs(cycles []chartCycle) []bbt.CycleInput {
	out := make([]bbt.CycleInput, len(cycles))
	for i, c := range cycles {
		out[i] = c.in
	}
	return out
}

// BBTV6 is GET /fertility/bbt on the engine state st (no query).
func BBTV6(st *engine.State, today civildate.Date, rangeSize int) bbt.Result {
	v := newView(st, today)
	_, cycles := chartCyclesV6(v, today)
	if len(cycles) == 0 {
		return bbt.BuildRule(nil, nil, today, rangeSize, engineRule(st))
	}
	return bbt.BuildRule(inputs(cycles), engineReadings(st), today, rangeSize, engineRule(st))
}

// InsightsV6 is GET /fertility/insights on the engine state st: one query (the LH tests of the current cycle, whose
// faint results the engine does not keep).
func (s *Service) InsightsV6(ctx context.Context, st *engine.State, userID uint64, today civildate.Date) (Insights, error) {
	v := newView(st, today)
	now, cycles := chartCyclesV6(v, today)
	var tests []LHTest
	if len(cycles) > 0 {
		var err error
		if tests, err = s.lhTests(ctx, userID, cycles[0].in.Start, today); err != nil {
			return Insights{}, err
		}
	}
	return insightsV6(v, now, cycles, tests, today), nil
}

// insightsV6 is the pure part of InsightsV6.
func insightsV6(v *view, now engine.Status, cycles []chartCycle, tests []LHTest, today civildate.Date) Insights {
	st := v.st
	res := bbt.BuildRule(inputs(cycles), engineReadings(st), today, len(cycles), engineRule(st))
	length, _ := st.CycleLength()
	out := Insights{CyclesUsed: st.ValidCycleCount(), Window: windowV6(st, now, today), History: []HistoryRow{}}

	cyclesRow := Evidence{Key: EvidenceCycles, Strength: StrengthNone, Cycles: out.CyclesUsed, Length: length}
	if n, ok := st.Variability().Get(); ok {
		cyclesRow.Variability = &n
	}
	switch {
	case out.CyclesUsed > 0 && now.Metrics.CycleRegularity == engine.RegularityRegular:
		cyclesRow.Strength = StrengthStrong
	case out.CyclesUsed > 0:
		cyclesRow.Strength = StrengthMedium
	}

	shiftRow := Evidence{Key: EvidenceBBTShift, Strength: StrengthNone, ShiftDays: []int{}}
	for _, c := range res.Cycles {
		if c.ShiftDay != nil {
			shiftRow.ShiftDays = append(shiftRow.ShiftDays, *c.ShiftDay)
		}
	}
	switch n := len(shiftRow.ShiftDays); {
	case n >= 2:
		shiftRow.Strength = StrengthStrong
	case n == 1:
		shiftRow.Strength = StrengthMedium
	}

	lhRow := Evidence{Key: EvidenceLH, Strength: StrengthNone}
	if len(cycles) > 0 {
		lhRow = lhEvidence(cycles[0].in, tests)
	}
	out.Evidence = []Evidence{cyclesRow, shiftRow, lhRow}
	out.Confidence = confidence(out.Window != nil, cyclesRow.Strength, shiftRow.Strength, lhRow.Strength)

	for i := 1; i < len(cycles) && i <= historyCycles; i++ {
		out.History = append(out.History, historyRowV6(st, cycles[i]))
	}

	if shiftRow.Strength != StrengthStrong {
		out.Tips = append(out.Tips, TipBBTDaily)
	}
	if lhRow.Strength != StrengthStrong && out.Window != nil {
		out.Tips = append(out.Tips, TipLHFromDay)
	}
	if cyclesRow.Strength != StrengthStrong {
		out.Tips = append(out.Tips, TipLogPeriods)
	}
	if len(out.Tips) == 0 {
		out.Tips = append(out.Tips, TipKeepLogging)
	}
	return out
}

// windowV6 is the engine's fertile window: today's prediction card window (§2.19) while today is on or before its
// ovulation day (or the period is late), else the next cycle's (future_cycles[2]: the ovulation before the second
// projected start). nil without one (no anchor, fertility suppressed, an empty display window).
func windowV6(st *engine.State, now engine.Status, today civildate.Date) *Window {
	card, ok := st.TodayUI(now).PredictionCard.Get()
	if !ok {
		return nil
	}
	fertile, okF := card.Fertile.Get()
	ov, okO := card.Ovulation.Get()
	if !okF || !okO {
		return nil
	}
	t := engine.DayOf(today)
	next := now.Anchors.PredictedNextPeriodStart
	if t <= ov.Date || !next.IsSet() || next <= t {
		return &Window{Start: fertile.Start.Civil(), End: fertile.End.Civil(), Ovulation: ov.Date.Civil(),
			FirstCycleDay: int(fertile.Start-now.Anchors.CurrentPeriodStart) + 1}
	}
	future := st.FutureCycles(t)
	if len(future) < 2 || !future[1].Ovulation.IsSet() {
		return nil
	}
	f := future[1]
	start := max(f.FertileStart, future[0].PeriodEnd+1) // §2.19: never inside the projected period
	if start > f.Ovulation {
		return nil
	}
	return &Window{Start: start.Civil(), End: f.Ovulation.Civil(), Ovulation: f.Ovulation.Civil(),
		FirstCycleDay: int(start-future[0].PeriodStart) + 1}
}

// historySource maps the engine's ovulation source onto the history row's.
func historySource(src engine.OvulationSource) string {
	switch src {
	case engine.OvulationBBTConfirmed:
		return SourceBBT
	case engine.OvulationLHPositive:
		return SourceLH
	}
	return SourceEstimate
}

// historyRowV6 is a finished cycle's ovulation as the engine resolved it (its anchors on the cycle's first day): the
// BBT-confirmed day, else LH + 1, else the calendar estimate; the LH/BBT conflict is already BBT (§2.17). The period
// length is the engine's real period that opens the cycle.
func historyRowV6(st *engine.State, c chartCycle) HistoryRow {
	row := HistoryRow{Start: c.in.Start, Source: SourceEstimate, PeriodDays: periodDaysV6(st, c.in)}
	if len(c.statuses) == 0 {
		return row
	}
	a := c.statuses[0].Anchors
	if a.CurrentPeriodStart != engine.DayOf(c.in.Start) {
		return row
	}
	row.Source = historySource(a.OvulationSource)
	ov := a.EstimatedOvulationDate
	if !ov.IsSet() {
		return row
	}
	date := ov.Civil()
	if date.Before(c.in.Start) || date.After(c.in.End) {
		return row
	}
	day := c.in.Start.DiffDays(date) + 1
	row.OvulationDay, row.Ovulation = &day, date
	return row
}

// periodDaysV6 is the logged length of the engine's real period starting on c.Start (nil when open or absent),
// never longer than the cycle.
func periodDaysV6(st *engine.State, c bbt.CycleInput) *int {
	start := engine.DayOf(c.Start)
	for _, p := range st.Periods() {
		if p.Start != start || p.Open() || p.End < p.Start {
			continue
		}
		days := min(int(p.End-p.Start)+1, c.Start.DiffDays(c.End)+1)
		return &days
	}
	return nil
}
