package conditions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PMDD daily questionnaire (nbl_Cond_PMDD): every catalog `pmdd_items` item rated 1 (not at all) … 6 (extreme).
// The chart reads the user's cycles from cycle_histories and compares the late-luteal window (the last 10 days of the
// cycle) with the follicular window (cycle days 4–10). The thresholds below are [needs clinical review].
const (
	PMDDMin = 1
	PMDDMax = 6

	// RequiredCycles is the "needs 2 complete cycles" rule.
	RequiredCycles = 2
	// LateLutealDays is the late-luteal window: the last N days of the cycle («۱۰ روز آخر»).
	LateLutealDays = 10
	// FollicularFrom … FollicularTo are the cycle days of the follicular comparison window.
	FollicularFrom = 4
	FollicularTo   = 10
	// A closed cycle is complete with at least CompleteLuteal rated late-luteal days and CompleteFollicular rated
	// follicular days.
	CompleteLuteal     = 7
	CompleteFollicular = 4
	// A cycle is evaluable for the pattern with at least EvalDays rated days in each window.
	EvalDays = 3
	// LutealRise: the late-luteal mean is at least this factor of the follicular mean (a 30 % rise).
	LutealRise = 1.3
)

// Pattern codes (catalog condition_alerts pmdd_pattern_luteal / _unclear / pmdd_not_enough_data).
const (
	PatternLuteal        = "luteal"
	PatternUnclear       = "unclear"
	PatternNotEnoughData = "not_enough_data"
)

// PMDDDay is one questionnaire day: scores by item code (catalog order, then codes no longer in the catalog).
type PMDDDay struct {
	Date   civildate.Date
	Scores []ItemScore
}

// ItemScore is one rated item.
type ItemScore struct {
	Code  string
	Score int
}

// Mean is the day's mean score (0 when nothing was rated).
func (d PMDDDay) Mean() float64 {
	if len(d.Scores) == 0 {
		return 0
	}
	sum := 0
	for _, s := range d.Scores {
		sum += s.Score
	}
	return round2(float64(sum) / float64(len(d.Scores)))
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func decodeScores(raw json.RawMessage) map[string]int {
	m := map[string]int{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func orderScores(m map[string]int, order []string) []ItemScore {
	out := []ItemScore{}
	for _, c := range order {
		if v, ok := m[c]; ok {
			out = append(out, ItemScore{Code: c, Score: v})
		}
	}
	var rest []string
	for c := range m {
		if !slices.Contains(order, c) {
			rest = append(rest, c)
		}
	}
	slices.Sort(rest)
	for _, c := range rest {
		out = append(out, ItemScore{Code: c, Score: m[c]})
	}
	return out
}

// PMDDItemCodes are the active pmdd_items codes (catalog order).
func (s *Service) PMDDItemCodes(ctx context.Context) ([]string, error) {
	_, codes, err := s.codes(ctx, GroupPMDDItems)
	return codes, err
}

// PMDD loads one questionnaire day.
func (s *Service) PMDD(ctx context.Context, userID uint64, date civildate.Date) (PMDDDay, error) {
	days, err := s.pmddDays(ctx, store.New(s.conn), userID, date, date)
	if err != nil {
		return PMDDDay{}, err
	}
	if len(days) == 0 {
		return PMDDDay{Date: date, Scores: []ItemScore{}}, nil
	}
	return days[0], nil
}

func (s *Service) pmddDays(ctx context.Context, q *store.Queries, userID uint64, from, to civildate.Date) ([]PMDDDay, error) {
	order, err := s.PMDDItemCodes(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPmddEntries(ctx, store.ListPmddEntriesParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("conditions: load pmdd entries: %w", err)
	}
	out := make([]PMDDDay, len(rows))
	for i, r := range rows {
		out[i] = PMDDDay{Date: r.EntryDate, Scores: orderScores(decodeScores(r.Scores), order)}
	}
	return out, nil
}

// PMDDInput is a validated PUT /conditions/pmdd/{date}: per item a score, or nil to clear it; items not sent stay.
type PMDDInput struct {
	Date   civildate.Date
	Scores map[string]*int
}

// SavePMDD merges the sent items over the day (a day left empty is deleted). ErrNotEnrolled without the PMDD
// program.
func (s *Service) SavePMDD(ctx context.Context, userID uint64, in PMDDInput, now time.Time) (PMDDDay, error) {
	var saved PMDDDay
	err := s.inTx(ctx, func(q *store.Queries, _ *healthlog.Service) error {
		if err := requireEnrolled(ctx, q, userID, ProgramPMDD); err != nil {
			return err
		}
		days, err := s.pmddDays(ctx, q, userID, in.Date, in.Date)
		if err != nil {
			return err
		}
		m := map[string]int{}
		if len(days) > 0 {
			for _, sc := range days[0].Scores {
				m[sc.Code] = sc.Score
			}
		}
		for c, v := range in.Scores {
			if v == nil {
				delete(m, c)
			} else {
				m[c] = *v
			}
		}
		if len(m) == 0 {
			if err := q.DeletePmddEntry(ctx, store.DeletePmddEntryParams{UserID: userID, EntryDate: in.Date}); err != nil {
				return fmt.Errorf("conditions: delete pmdd entry: %w", err)
			}
		} else {
			raw, err := json.Marshal(m)
			if err != nil {
				return fmt.Errorf("conditions: encode scores: %w", err)
			}
			if err := q.UpsertPmddEntry(ctx, store.UpsertPmddEntryParams{UserID: userID, EntryDate: in.Date, Scores: raw, Now: tehranNow(now)}); err != nil {
				return fmt.Errorf("conditions: save pmdd entry: %w", err)
			}
		}
		days, err = s.pmddDays(ctx, q, userID, in.Date, in.Date)
		if err != nil {
			return err
		}
		saved = PMDDDay{Date: in.Date, Scores: []ItemScore{}}
		if len(days) > 0 {
			saved = days[0]
		}
		return nil
	})
	return saved, err
}

// Window is an inclusive date range.
type Window struct{ From, To civildate.Date }

// Has reports whether d is inside the window.
func (w Window) Has(d civildate.Date) bool { return !d.Before(w.From) && !d.After(w.To) }

// ChartDay is one rated day of a cycle.
type ChartDay struct {
	Day        PMDDDay
	CycleDay   int
	InPeriod   bool
	LateLuteal bool
}

// ChartCycle is one cycle of the PMDD chart.
type ChartCycle struct {
	Start          civildate.Date
	End            civildate.Date // the last day (actual for a closed cycle, expected for the current one)
	Closed         bool
	Period         Window
	LateLuteal     Window
	Follicular     Window
	Days           []ChartDay
	LutealDays     int
	FollicularDays int
	LutealMean     *float64
	FollicularMean *float64
	Complete       bool
}

// Chart is GET /conditions/pmdd/chart: the current cycle and up to two closed ones, newest first.
type Chart struct {
	Cycles         []ChartCycle
	CompleteCycles int
	Ready          bool
	Pattern        string
	Alerts         []catalog.Item
}

// BuildCycles lays out the current cycle (open, expected length cycleLen) and the closed cycles before it from the
// newest-first period starts, and fills them with the rated days.
func BuildCycles(starts []PeriodStart, today civildate.Date, cycleLen, periodLen int, days []PMDDDay) []ChartCycle {
	if cycleLen < 15 || cycleLen > 60 {
		cycleLen = defaultCycleDays
	}
	out := []ChartCycle{}
	var next *civildate.Date // the start of the cycle after this one (nil for the current cycle)
	for _, p := range starts {
		c := ChartCycle{Start: p.Start}
		if next == nil {
			c.End = p.Start.AddDays(cycleLen - 1)
			if c.End.Before(today) { // overdue: the cycle runs at least until today
				c.End = today
			}
		} else {
			c.End = next.AddDays(-1)
			c.Closed = true
		}
		start := p.Start
		next = &start
		c.Period = Window{p.Start, minDate(p.LastPeriodDay(periodLen), c.End)}
		c.LateLuteal = Window{maxDate(c.End.AddDays(-(LateLutealDays - 1)), p.Start), c.End}
		c.Follicular = Window{p.Start.AddDays(FollicularFrom - 1), minDate(p.Start.AddDays(FollicularTo-1), c.End)}
		var lut, fol []float64
		for _, d := range days {
			if !(Window{c.Start, c.End}).Has(d.Date) || len(d.Scores) == 0 {
				continue
			}
			cd := ChartDay{Day: d, CycleDay: c.Start.DiffDays(d.Date) + 1, InPeriod: c.Period.Has(d.Date), LateLuteal: c.LateLuteal.Has(d.Date)}
			c.Days = append(c.Days, cd)
			if cd.LateLuteal {
				lut = append(lut, d.Mean())
			}
			if c.Follicular.Has(d.Date) {
				fol = append(fol, d.Mean())
			}
		}
		c.LutealDays, c.FollicularDays = len(lut), len(fol)
		c.LutealMean, c.FollicularMean = mean(lut), mean(fol)
		c.Complete = c.Closed && c.LutealDays >= CompleteLuteal && c.FollicularDays >= CompleteFollicular
		out = append(out, c)
	}
	return out
}

// Summarise counts complete cycles and reads the pattern over the evaluable cycles.
func Summarise(cycles []ChartCycle) (complete int, pattern string) {
	evaluable, rising := 0, 0
	for _, c := range cycles {
		if c.Complete {
			complete++
		}
		if c.LutealDays < EvalDays || c.FollicularDays < EvalDays {
			continue
		}
		evaluable++
		if *c.LutealMean >= *c.FollicularMean*LutealRise && *c.LutealMean > *c.FollicularMean {
			rising++
		}
	}
	switch {
	case evaluable == 0:
		return complete, PatternNotEnoughData
	case rising == evaluable:
		return complete, PatternLuteal
	default:
		return complete, PatternUnclear
	}
}

func mean(v []float64) *float64 {
	if len(v) == 0 {
		return nil
	}
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	m := round2(sum / float64(len(v)))
	return &m
}

func minDate(a, b civildate.Date) civildate.Date {
	if a.Before(b) {
		return a
	}
	return b
}

func maxDate(a, b civildate.Date) civildate.Date {
	if a.After(b) {
		return a
	}
	return b
}

// PMDDChart builds the chart as of today. Alerts: pmdd_crisis always, the pattern sentence, and
// pmdd_needs_two_cycles until RequiredCycles cycles are complete.
func (s *Service) PMDDChart(ctx context.Context, userID uint64, today civildate.Date) (Chart, error) {
	q := store.New(s.conn)
	starts, err := periodStarts(ctx, q, userID, today)
	if err != nil {
		return Chart{}, err
	}
	cycleLen, periodLen, err := cycleDefaults(ctx, q, userID)
	if err != nil {
		return Chart{}, err
	}
	var days []PMDDDay
	if len(starts) > 0 {
		if days, err = s.pmddDays(ctx, q, userID, starts[len(starts)-1].Start, today); err != nil {
			return Chart{}, err
		}
	}
	cycles := BuildCycles(starts, today, cycleLen, periodLen, days)
	complete, pattern := Summarise(cycles)
	ch := Chart{Cycles: cycles, CompleteCycles: min(complete, RequiredCycles), Ready: complete >= RequiredCycles, Pattern: pattern}
	codes := []string{"pmdd_pattern_" + pattern}
	if pattern == PatternNotEnoughData {
		codes = []string{"pmdd_not_enough_data"}
	}
	if !ch.Ready {
		codes = append(codes, "pmdd_needs_two_cycles")
	}
	codes = append(codes, "pmdd_crisis")
	if ch.Alerts, err = s.alerts(ctx, codes...); err != nil {
		return Chart{}, err
	}
	return ch, nil
}
