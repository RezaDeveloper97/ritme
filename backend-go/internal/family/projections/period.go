package projections

import (
	"context"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PeriodForecastMaxDays caps how far ahead a member may look (about three cycles).
const PeriodForecastMaxDays = 92

// PeriodForecast is the period_forecast DTO: the days in [from, to] her period is likely to fall on, today onwards
// only. It exists only behind the owner's own «روزهای احتمالی پریود» consent on the link (companion
// preferences.period_days) on top of a readable cycle_phase grant, for the partner role. Never logged or past days
// (history), cycle day or length, lateness, confidence or fertility; while she is late the list is empty, because a
// late period has no honest prediction to show.
type PeriodForecast struct {
	Days []string `json:"days"`
}

// PeriodSource resolves which days of a window fall in the menstrual phase of the owner's cycle prediction.
// late reports whether the owner is late on today (then days is ignored).
type PeriodSource interface {
	PeriodDays(ctx context.Context, ownerID uint64, today, from, to civildate.Date) (days []civildate.Date, late bool, err error)
}

// ForecastWindow clamps a requested window to [today, today+PeriodForecastMaxDays]; ok is false when nothing is left.
func ForecastWindow(from, to, today civildate.Date) (civildate.Date, civildate.Date, bool) {
	if from.Before(today) {
		from = today
	}
	if last := today.AddDays(PeriodForecastMaxDays); to.After(last) {
		to = last
	}
	return from, to, !to.Before(from)
}

// PeriodForecast is the member's period_forecast of s.OwnerID over [from, to]. It decides the cycle_phase read first
// (a denied, paused or out-of-mode grant is ErrDenied and nothing is loaded) and refuses every role but the partner.
// The caller checks the owner's per-link consent.
func (b *Builder) PeriodForecast(ctx context.Context, s family.Snapshot, from, to civildate.Date, now time.Time) (PeriodForecast, error) {
	out := PeriodForecast{Days: []string{}}
	if s.Role != family.RolePartner {
		return out, ErrDenied
	}
	if family.Decide(s, family.Subject{}, family.DomainCyclePhase, family.ActionRead, now).Outcome != family.Allow {
		return out, ErrDenied
	}
	src, ok := b.cycle.(PeriodSource)
	if !ok {
		return out, nil
	}
	today := civildate.InTehran(now)
	from, to, ok = ForecastWindow(from, to, today)
	if !ok {
		return out, nil
	}
	days, late, err := src.PeriodDays(ctx, s.OwnerID, today, from, to)
	if err != nil || late {
		return out, err
	}
	for _, d := range days {
		out.Days = append(out.Days, d.String())
	}
	return out, nil
}

// PeriodDays (v1 resolver): each day of the window resolved on its own.
func (s v1) PeriodDays(ctx context.Context, ownerID uint64, today, from, to civildate.Date) ([]civildate.Date, bool, error) {
	sn, err := s.svc.Load(ctx, ownerID, from, to, today)
	if err != nil {
		return nil, false, err
	}
	if len(sn.Histories) == 0 && sn.Profile == nil {
		return nil, false, nil
	}
	profile := sn.EngineProfile()
	m := metrics.Calculate(sn.Histories, profile)
	if st := resolver.Resolve(sn.Histories, profile, today, today, m); st.DaysLate != nil && *st.DaysLate > 0 {
		return nil, true, nil
	}
	var out []civildate.Date
	for d := from; !d.After(to); d = d.AddDays(1) {
		if resolver.Resolve(sn.Histories, profile, d, today, m).MainPhase == enums.MainPhaseMenstrual {
			out = append(out, d)
		}
	}
	return out, false, nil
}

// PeriodDays (v6 engine): one Range over the window.
func (s v6Cycle) PeriodDays(ctx context.Context, ownerID uint64, today, from, to civildate.Date) ([]civildate.Date, bool, error) {
	es, err := s.states.ForUser(ctx, ownerID, today)
	if err != nil {
		return nil, false, err
	}
	t := engine.DayOf(today)
	if es.Resolve(t, t).DaysLate.Or(0) > 0 {
		return nil, true, nil
	}
	var out []civildate.Date
	for _, d := range es.Range(engine.DayOf(from), engine.DayOf(to), t) {
		if d.MainPhase == engine.MainPhaseMenstrual {
			out = append(out, d.Date.Civil())
		}
	}
	return out, false, nil
}
