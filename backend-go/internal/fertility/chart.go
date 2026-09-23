package fertility

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// BBTRanges are the chart's range tabs (cycles shown); the first is the default.
var BBTRanges = []int{1, 3, 6}

// maxBBTCycles is how many cycles the chart analyses: the largest range. Past shift days always
// come from all of them, whatever the tab.
const maxBBTCycles = 6

// maxWindowScan caps the days scanned for a cycle's fertile window (a months-long "cycle" is a
// logging gap, not a cycle).
const maxWindowScan = 60

// BBT is GET /fertility/bbt: the current cycle and up to maxBBTCycles−1 previous ones (period
// starts from the cycle history), their readings and the cycle engine's fertile windows, in three
// queries (profile, cycle history, readings).
func (s *Service) BBT(ctx context.Context, userID uint64, today civildate.Date, rangeSize int) (bbt.Result, error) {
	in, err := s.cycleInputs(ctx, userID)
	if err != nil {
		return bbt.Result{}, err
	}
	cycles := in.bbtCycles(today)
	if len(cycles) == 0 {
		return bbt.Build(nil, nil, today, rangeSize), nil
	}
	readings, err := s.bbtReadings(ctx, userID, cycles[len(cycles)-1].Start, today)
	if err != nil {
		return bbt.Result{}, err
	}
	return bbt.Build(cycles, readings, today, rangeSize), nil
}

// bbtReadings are the user's basal temperatures from `from` to `to` (inclusive), oldest first.
func (s *Service) bbtReadings(ctx context.Context, userID uint64, from, to civildate.Date) ([]bbt.Reading, error) {
	rows, err := store.New(s.db).ListBBTReadings(ctx, store.ListBBTReadingsParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("fertility: load bbt readings: %w", err)
	}
	readings := make([]bbt.Reading, 0, len(rows))
	for _, r := range rows {
		if !r.BasalBodyTemperature.Valid {
			continue
		}
		v, err := bbt.ParseValue(r.BasalBodyTemperature.String)
		if err != nil {
			return nil, fmt.Errorf("fertility: bbt reading of %s: %w", r.LogDate, err)
		}
		readings = append(readings, bbt.Reading{Date: r.LogDate, Value: v})
	}
	return readings, nil
}

// bbtCycles are the chart's cycle boundaries, newest first: the current cycle is the cycle
// engine's anchor for today (running to the day before the predicted next period, or today when
// late); previous cycles run between the history's period starts (logged or onboarding rows).
// Each carries the cycle engine's fertile window. None without a resolvable cycle.
func (in cycleInputs) bbtCycles(today civildate.Date) []bbt.CycleInput {
	st := in.resolve(today, today)
	if st.CurrentPeriodStart.IsZero() {
		return nil
	}
	end := today
	if next := st.PredictedNextPeriodStart; !next.IsZero() && next.AddDays(-1).After(end) {
		end = next.AddDays(-1)
	}
	starts := make([]civildate.Date, 0, len(in.histories))
	for _, h := range in.histories {
		if h.IsConfirmed || h.IsEstimated {
			starts = append(starts, h.PeriodStart)
		}
	}
	cycles := bbt.Boundaries(st.CurrentPeriodStart, end, starts, maxBBTCycles)
	for i := range cycles {
		cycles[i].FertileWindow = in.fertileWindow(cycles[i], today)
	}
	return cycles
}

// fertileWindow is the cycle days the cycle engine shows as the fertile zone (main_phase fertile)
// in the cycle; nil when none. No second prediction model: it is the cycle view's own window.
func (in cycleInputs) fertileWindow(c bbt.CycleInput, today civildate.Date) *bbt.DayRange {
	var w *bbt.DayRange
	for day := 1; day <= maxWindowScan; day++ {
		date := c.Start.AddDays(day - 1)
		if date.After(c.End) {
			break
		}
		if in.resolve(date, today).MainPhase != enums.MainPhaseFertile {
			continue
		}
		if w == nil {
			w = &bbt.DayRange{FromDay: day}
		}
		w.ToDay = day
	}
	return w
}

// BBTJSON is the GET /fertility/bbt body: {range, cycles[current, …], stats (current cycle),
// past_shift_days, tip}.
func BBTJSON(res bbt.Result, rangeSize int, locale string) *jsonx.OrderedMap {
	cycles := make([]any, 0, len(res.Cycles))
	for _, c := range res.Cycles {
		cycles = append(cycles, cycleJSON(c))
	}
	var stats bbt.Stats
	if len(res.Cycles) > 0 {
		stats = res.Cycles[0].Stats
	}
	return jsonx.Obj(
		"range", rangeSize,
		"cycles", cycles,
		"stats", jsonx.Obj(
			"pre_ovulation_avg", temperature(stats.PreOvulationAvg),
			"logged_days", stats.LoggedDays,
			"cycle_days_so_far", stats.CycleDaysSoFar,
			"gaps", stats.Gaps,
		),
		"past_shift_days", res.PastShiftDays,
		"tip", bbtTip(res.PastShiftDays, locale),
	)
}

func cycleJSON(c bbt.Cycle) *jsonx.OrderedMap {
	points := make([]any, 0, len(c.Points))
	for _, p := range c.Points {
		points = append(points, jsonx.Obj("cycle_day", p.CycleDay, "date", p.Date.String(), "value", bbt.Format(p.Value)))
	}
	var window any
	if w := c.FertileWindow; w != nil {
		window = jsonx.Obj("from_day", w.FromDay, "to_day", w.ToDay)
	}
	var shift any
	if c.ShiftDay != nil {
		shift = *c.ShiftDay
	}
	return jsonx.Obj(
		"start_date", c.Start.String(),
		"points", points,
		"coverline", temperature(c.Coverline),
		"fertile_window", window,
		"shift_day", shift,
		"phase", string(c.Phase),
	)
}

// temperature is hundredths as the decimal(4,2) string ("36.55"), or null.
func temperature(v *int) any {
	if v == nil {
		return nil
	}
	return bbt.Format(*v)
}

// bbtTip is the chart's tip {title, body}; the body adds the past cycles' shift days when known.
func bbtTip(pastShiftDays []int, locale string) *jsonx.OrderedMap {
	body := T("bbt.tip.body", locale)
	if n := len(pastShiftDays); n > 0 {
		key := "bbt.tip.past_shift_one"
		if n > 1 {
			key = "bbt.tip.past_shift_many"
		}
		body += " " + tr(key, locale, "count", num(n, locale), "days", joinDays(pastShiftDays, locale))
	}
	return jsonx.Obj("title", T("bbt.tip.title", locale), "body", body)
}

// joinDays lists days in the locale's digits: "15", "15 and 16", "14, 15 and 16".
func joinDays(days []int, locale string) string {
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = num(d, locale)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], T("list.separator", locale)) + T("list.last_separator", locale) + parts[len(parts)-1]
}

// tr is T with :placeholders.
func tr(key, locale string, kv ...string) string {
	params := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		params[kv[i]] = kv[i+1]
	}
	return translator().Trans("fertility."+key, params, locale)
}

// num is n in the locale's digits (the "digits" line; ASCII when missing).
func num(n int, locale string) string {
	s := strconv.Itoa(n)
	digits := []rune(T("digits", locale))
	if len(digits) != 10 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(digits[r-'0'])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
