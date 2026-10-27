package conditions

import (
	"context"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Cycle bounds of the PMDD chart and the PBAC period (T-M9-37, D-136). Under CYCLE_ENGINE=v1 they come from the last
// three cycle_histories rows (any row, confirmed or not) and the profile's declared lengths. Under shadow / v6 they
// come from the engine: its actual periods (estimated rows dropped, a conflicting row — e.g. a back-dated one-day
// bleeding row inside a logged period — excluded, so it no longer splits a cycle), its effective period length, and
// the current cycle's end from predicted_next_period_start (which moves past a recorded skip «پریود نداشتم»).

// CycleStates hands out a user's prepared v6 engine state for the user-local day (state.Source satisfies it). nil
// means CYCLE_ENGINE=v1.
type CycleStates interface {
	ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error)
}

// WithEngine makes the PMDD and PBAC cycle bounds come from the v6 engine (CYCLE_ENGINE shadow / v6); nil keeps v1.
func (s *Service) WithEngine(states CycleStates) *Service {
	s.cycle = states
	return s
}

// maxStarts is how many period starts the chart and the PBAC window read (the current cycle and two closed ones).
const maxStarts = 3

// cycleBounds are the period starts on or before a day (newest first, at most maxStarts), the period length used for
// a period without a logged end, and the last day of the current cycle (zero: from cycleLen, v1).
type cycleBounds struct {
	starts     []PeriodStart
	cycleLen   int
	periodLen  int
	currentEnd civildate.Date
}

// bounds reads the cycle bounds of userID on day (today = the user-local day).
func (s *Service) bounds(ctx context.Context, q *store.Queries, userID uint64, day, today civildate.Date) (cycleBounds, error) {
	if s.cycle == nil {
		starts, err := periodStarts(ctx, q, userID, day)
		if err != nil {
			return cycleBounds{}, err
		}
		cycleLen, periodLen, err := cycleDefaults(ctx, q, userID)
		if err != nil {
			return cycleBounds{}, err
		}
		return cycleBounds{starts: starts, cycleLen: cycleLen, periodLen: periodLen}, nil
	}
	st, err := s.cycle.ForUser(ctx, userID, today)
	if err != nil {
		return cycleBounds{}, fmt.Errorf("conditions: engine state: %w", err)
	}
	return engineBounds(st, day, today), nil
}

// engineBounds are the cycle bounds of day from a v6 engine state: the engine's actual periods starting on or before
// day (newest first, at most maxStarts) with their logged ends, the effective period and cycle lengths, and — when day
// is today — the current cycle's last day: predicted_next_period_start − 1 (never before today: an overdue cycle runs
// at least until today).
func engineBounds(st *engine.State, day, today civildate.Date) cycleBounds {
	d := engine.DayOf(day)
	ps := st.Periods()
	i, _ := slices.BinarySearchFunc(ps, d+1, func(p engine.Period, t engine.Day) int {
		switch {
		case p.Start < t:
			return -1
		case p.Start > t:
			return 1
		}
		return 0
	})
	b := cycleBounds{}
	b.cycleLen, _ = st.CycleLength()
	b.periodLen, _ = st.PeriodLength()
	for j := i - 1; j >= 0 && len(b.starts) < maxStarts; j-- {
		p := PeriodStart{Start: ps[j].Start.Civil()}
		if !ps[j].Open() {
			end := ps[j].End.Civil()
			p.End = &end
		}
		b.starts = append(b.starts, p)
	}
	if len(b.starts) > 0 && day == today {
		t := engine.DayOf(today)
		if next := st.Resolve(t, t).Anchors.PredictedNextPeriodStart; next != engine.NoDay && next > ps[i-1].Start {
			b.currentEnd = next.Civil().AddDays(-1)
		}
	}
	return b
}

// localToday is the user-local day of ctx (the request's zone and pinned clock, else now in Asia/Tehran).
func localToday(ctx context.Context) civildate.Date {
	return civildate.LocalToday(ctx, clock.FromContext(ctx, clock.Real{}))
}
