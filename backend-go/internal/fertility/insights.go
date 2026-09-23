package fertility

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// GET /fertility/insights (T-M5-03, docs/fertility-ttc/README.md → Insights): where the fertile
// window comes from. No second prediction model: the window and the ovulation day are the cycle
// view's; the evidence rows and the confidence only explain how much logged data backs them.
//
// Evidence rows (strength strong | medium | none):
//   - cycles:    complete valid cycles (cycle metrics). strong = the metrics call the last 3 cycles
//     relatively_regular; medium = at least one valid cycle; none = no complete cycle yet.
//   - bbt_shift: confirmed BBT shifts (3-over-6) in the current and the last 5 cycles.
//     strong = 2 or more; medium = 1; none = 0.
//   - lh:        LH tests of the current cycle. strong = a positive test; medium = a faint test;
//     none = only negative tests or no test.
//
// Confidence: points strong = 2, medium = 1, none = 0 summed over the three rows; high = 4 or
// more with a cycles row that is not none, medium = 2 or more, low otherwise (and always low
// without a window).
//
// History: the last 5 finished cycles, newest first; ovulation day = the day before a confirmed BBT
// shift, else the day after the cycle's first positive LH test, else the cycle engine's estimate
// (the predicted next period − 14 days, from the cycle's own start).

// Evidence keys.
const (
	EvidenceCycles   = "cycles"
	EvidenceBBTShift = "bbt_shift"
	EvidenceLH       = "lh"
)

// Evidence strengths.
const (
	StrengthStrong = "strong"
	StrengthMedium = "medium"
	StrengthNone   = "none"
)

// Confidence levels.
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

// History ovulation sources.
const (
	SourceBBT      = "bbt"
	SourceLH       = "lh"
	SourceEstimate = "estimate"
)

// Tip keys (lang: fertility.insights.tips.<key>), in display order.
const (
	TipBBTDaily    = "bbt_daily"
	TipLHFromDay   = "lh_from_day"
	TipLogPeriods  = "log_periods"
	TipKeepLogging = "keep_logging"
)

// historyCycles is how many finished cycles the history lists.
const historyCycles = 5

// maxWindowDays caps the backward scan for the fertile window's first day.
const maxWindowDays = 10

// LHTest is one logged LH test.
type LHTest struct {
	Date  civildate.Date
	Value string // negative | faint | positive
}

// Window is the fertile window and the estimated ovulation day (dates).
type Window struct {
	Start, End, Ovulation civildate.Date
	// FirstCycleDay is the window's first day as a cycle day of its cycle (the LH tip).
	FirstCycleDay int
}

// Evidence is one «این تخمین از کجا آمده؟» row, before localisation.
type Evidence struct {
	Key      string
	Strength string
	// cycles
	Cycles, Length int
	Variability    *int
	// bbt_shift: shift days, newest cycle first
	ShiftDays []int
	// lh: the deciding test's cycle day (0 = none) and the number of tests this cycle
	LHDay, LHTests int
}

// HistoryRow is one finished cycle's ovulation estimate.
type HistoryRow struct {
	Start        civildate.Date
	Ovulation    civildate.Date // zero when unknown
	OvulationDay *int
	Source       string
}

// Insights is GET /fertility/insights before localisation.
type Insights struct {
	CyclesUsed int
	Window     *Window
	Confidence string
	Evidence   []Evidence
	History    []HistoryRow
	Tips       []string
}

// Insights reads the user's cycle history, BBT readings and LH tests (4 queries: profile, cycle
// history, readings, LH tests) and explains the current fertile window.
func (s *Service) Insights(ctx context.Context, userID uint64, today civildate.Date) (Insights, error) {
	in, err := s.cycleInputs(ctx, userID)
	if err != nil {
		return Insights{}, err
	}
	cycles := in.bbtCycles(today)
	var readings []bbt.Reading
	var tests []LHTest
	if len(cycles) > 0 {
		from := cycles[len(cycles)-1].Start
		if readings, err = s.bbtReadings(ctx, userID, from, today); err != nil {
			return Insights{}, err
		}
		if tests, err = s.lhTests(ctx, userID, from, today); err != nil {
			return Insights{}, err
		}
	}
	return in.insights(cycles, readings, tests, today), nil
}

// lhTests are the user's LH tests from `from` to `to` (inclusive), oldest first.
func (s *Service) lhTests(ctx context.Context, userID uint64, from, to civildate.Date) ([]LHTest, error) {
	rows, err := store.New(s.db).ListLHTests(ctx, store.ListLHTestsParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("fertility: load lh tests: %w", err)
	}
	out := make([]LHTest, 0, len(rows))
	for _, r := range rows {
		if r.LhTest.Valid {
			out = append(out, LHTest{Date: r.LogDate, Value: r.LhTest.String})
		}
	}
	return out, nil
}

// insights is the pure part: cycles are the chart's boundaries (newest first, current first),
// readings and tests cover them.
func (in cycleInputs) insights(cycles []bbt.CycleInput, readings []bbt.Reading, tests []LHTest, today civildate.Date) Insights {
	res := bbt.Build(cycles, readings, today, len(cycles))
	out := Insights{
		CyclesUsed: in.metrics.ValidCyclesCount,
		Window:     in.window(today),
		History:    []HistoryRow{},
	}

	cyclesRow := Evidence{Key: EvidenceCycles, Strength: StrengthNone, Cycles: in.metrics.ValidCyclesCount,
		Length: in.metrics.EffectiveCycleLength, Variability: in.metrics.CycleVariabilityRange}
	switch {
	case in.metrics.ValidCyclesCount > 0 && in.metrics.RegularityStatus == enums.RegularityStatusRelativelyRegular:
		cyclesRow.Strength = StrengthStrong
	case in.metrics.ValidCyclesCount > 0:
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
		lhRow = lhEvidence(cycles[0], tests)
	}
	out.Evidence = []Evidence{cyclesRow, shiftRow, lhRow}
	out.Confidence = confidence(out.Window != nil, cyclesRow.Strength, shiftRow.Strength, lhRow.Strength)

	for i := 1; i < len(cycles) && i <= historyCycles; i++ {
		out.History = append(out.History, in.historyRow(cycles[i], res.Cycles[i], tests, today))
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

// lhEvidence is the lh row of the current cycle: its first positive test, else its first faint one.
func lhEvidence(c bbt.CycleInput, tests []LHTest) Evidence {
	row := Evidence{Key: EvidenceLH, Strength: StrengthNone}
	faintDay := 0
	for _, t := range tests {
		if t.Date.Before(c.Start) || t.Date.After(c.End) {
			continue
		}
		row.LHTests++
		day := c.Start.DiffDays(t.Date) + 1
		switch {
		case t.Value == "positive" && row.LHDay == 0:
			row.LHDay = day
		case t.Value == "faint" && faintDay == 0:
			faintDay = day
		}
	}
	switch {
	case row.LHDay > 0:
		row.Strength = StrengthStrong
	case faintDay > 0:
		row.Strength, row.LHDay = StrengthMedium, faintDay
	}
	return row
}

func points(strength string) int {
	switch strength {
	case StrengthStrong:
		return 2
	case StrengthMedium:
		return 1
	}
	return 0
}

// confidence is the rule in the package comment above.
func confidence(hasWindow bool, cycles, shift, lh string) string {
	if !hasWindow {
		return ConfidenceLow
	}
	score := points(cycles) + points(shift) + points(lh)
	switch {
	case score >= 4 && cycles != StrengthNone:
		return ConfidenceHigh
	case score >= 2:
		return ConfidenceMedium
	}
	return ConfidenceLow
}

// window is the cycle view's fertile window: the current cycle's while its ovulation day is today
// or later, else the next predicted cycle's (the current one's when the period is late). Its
// days are the ones the cycle engine shows as main_phase fertile, ending on the ovulation day.
// nil without a resolvable cycle or when the engine shows no fertile day.
func (in cycleInputs) window(today civildate.Date) *Window {
	st := in.resolve(today, today)
	if st.CurrentPeriodStart.IsZero() || st.EstimatedOvulationDate.IsZero() {
		return nil
	}
	if today.After(st.EstimatedOvulationDate) && st.PredictedNextPeriodStart.After(today) {
		st = in.resolve(st.PredictedNextPeriodStart, today)
	}
	ov := st.EstimatedOvulationDate
	if in.resolve(ov, today).MainPhase != enums.MainPhaseFertile {
		return nil
	}
	start := ov
	for i := 0; i < maxWindowDays; i++ {
		prev := start.AddDays(-1)
		if in.resolve(prev, today).MainPhase != enums.MainPhaseFertile {
			break
		}
		start = prev
	}
	return &Window{Start: start, End: ov, Ovulation: ov, FirstCycleDay: st.CurrentPeriodStart.DiffDays(start) + 1}
}

// historyRow is a finished cycle's ovulation: the confirmed BBT shift, else the first positive
// LH + 1, else the cycle engine's estimate from the cycle's start.
func (in cycleInputs) historyRow(c bbt.CycleInput, analysed bbt.Cycle, tests []LHTest, today civildate.Date) HistoryRow {
	row := HistoryRow{Start: c.Start}
	day := 0
	switch {
	case analysed.OvulationDay() != nil:
		day, row.Source = *analysed.OvulationDay(), SourceBBT
	default:
		for _, t := range tests {
			if t.Value == "positive" && !t.Date.Before(c.Start) && !t.Date.After(c.End) {
				day, row.Source = c.Start.DiffDays(t.Date)+2, SourceLH
				break
			}
		}
	}
	if row.Source == "" {
		row.Source = SourceEstimate
		if ov := in.resolve(c.Start, today).EstimatedOvulationDate; !ov.IsZero() {
			day = c.Start.DiffDays(ov) + 1
		}
	}
	if day < 1 || c.Start.AddDays(day-1).After(c.End) {
		return HistoryRow{Start: c.Start, Source: row.Source}
	}
	row.OvulationDay = &day
	row.Ovulation = c.Start.AddDays(day - 1)
	return row
}

// InsightsJSON is the GET /fertility/insights body.
func InsightsJSON(ins Insights, locale string) *jsonx.OrderedMap {
	var window any
	if w := ins.Window; w != nil {
		window = jsonx.Obj("start", w.Start.String(), "end", w.End.String(), "ovulation", w.Ovulation.String())
	}
	evidence := make([]any, 0, len(ins.Evidence))
	for _, e := range ins.Evidence {
		evidence = append(evidence, jsonx.Obj(
			"key", e.Key,
			"title", T("insights.evidence."+e.Key+".title", locale),
			"detail", evidenceDetail(e, locale),
			"strength", e.Strength,
		))
	}
	history := make([]any, 0, len(ins.History))
	for _, h := range ins.History {
		var day, date any
		label := monthLabel(h.Start, locale)
		if h.OvulationDay != nil {
			day, date = *h.OvulationDay, h.Ovulation.String()
			label = monthLabel(h.Ovulation, locale)
		}
		history = append(history, jsonx.Obj(
			"month_label", label,
			"ovulation_day", day,
			"date", date,
			"source", h.Source,
			"cycle_start", h.Start.String(),
		))
	}
	tips := make([]any, 0, len(ins.Tips))
	for _, key := range ins.Tips {
		day := 0
		if ins.Window != nil {
			day = ins.Window.FirstCycleDay
		}
		tips = append(tips, tr("insights.tips."+key, locale, "day", num(day, locale)))
	}
	return jsonx.Obj(
		"cycles_used", ins.CyclesUsed,
		"window", window,
		"confidence", ins.Confidence,
		"evidence", evidence,
		"history", history,
		"tips", tips,
	)
}

// evidenceDetail is the row's localized detail line.
func evidenceDetail(e Evidence, locale string) string {
	base := "insights.evidence." + e.Key + "."
	switch e.Key {
	case EvidenceCycles:
		switch {
		case e.Cycles == 0:
			return T(base+"none", locale)
		case e.Variability == nil:
			return tr(base+"length", locale, "count", num(e.Cycles, locale), "length", num(e.Length, locale))
		}
		return tr(base+"variability", locale, "count", num(e.Cycles, locale), "length", num(e.Length, locale),
			"variability", num((*e.Variability+1)/2, locale))
	case EvidenceBBTShift:
		switch len(e.ShiftDays) {
		case 0:
			return T(base+"none", locale)
		case 1:
			return tr(base+"one", locale, "days", joinDays(e.ShiftDays, locale))
		}
		return tr(base+"many", locale, "count", num(len(e.ShiftDays), locale), "days", joinDays(e.ShiftDays, locale))
	case EvidenceLH:
		switch {
		case e.Strength == StrengthStrong:
			return tr(base+"positive", locale, "day", num(e.LHDay, locale))
		case e.Strength == StrengthMedium:
			return tr(base+"faint", locale, "day", num(e.LHDay, locale))
		case e.LHTests > 0:
			return tr(base+"negative", locale, "count", num(e.LHTests, locale))
		}
		return T(base+"none", locale)
	}
	return ""
}

// monthLabel is d's month name in the locale's calendar (fertility.calendar: jalali | gregorian).
func monthLabel(d civildate.Date, locale string) string {
	m := int(d.Month)
	if T("calendar", locale) == "jalali" {
		_, m, _ = engine.ToJalali(d)
	}
	line, ok := translator().Get("fertility.months", locale)
	if !ok {
		return num(m, locale)
	}
	_, months := phpval.Entries(line)
	if m < 1 || m > len(months) {
		return num(m, locale)
	}
	return phpval.ToString(months[m-1])
}
