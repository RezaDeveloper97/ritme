package projections

import (
	"context"
	"database/sql"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PeriodSoonDays: period_soon is true when the next period starts within this many days (F-01).
const PeriodSoonDays = 3

// CycleFacts is what a cycle source reports about the owner's day. It holds more than any member may see; only
// CyclePhaseOf and TTCOf turn it into a DTO, and they copy nothing but the allow-listed fields.
type CycleFacts struct {
	HasData bool
	Phase   enums.MainPhase
	// FertilityKnown: the source could tell fertility suppression. Without it the fertile label and the TTC window
	// are never exposed (fail closed).
	FertilityKnown      bool
	FertilitySuppressed bool
	// DaysToPeriod is set when the next period is predicted and the owner is not late.
	DaysToPeriod *int
	Late         bool
	// TTC window (v6 engine, not suppressed): zero dates when unknown.
	FertileStart, FertileEnd, Ovulation civildate.Date
	BestDays                            []civildate.Date
}

// CycleSource loads CycleFacts for an owner.
type CycleSource interface {
	Facts(ctx context.Context, ownerID uint64, today civildate.Date) (CycleFacts, error)
}

// CyclePhase is the cycle_phase DTO (family.md §4, F-01, F-02): the main phase only, plus period_soon for the
// partner role. No date, cycle day, cycle length, lateness, confidence or prediction.
type CyclePhase struct {
	MainPhase  *string `json:"main_phase"`
	PeriodSoon *bool   `json:"period_soon,omitempty"`
}

// TTCFertility is the ttc_fertility DTO (F-02, partner-only switch, TTC mode only): the fertile window, the estimated
// ovulation day and the best days. Never LH, BBT, intercourse or test values.
type TTCFertility struct {
	FertileStart  *string  `json:"fertile_start"`
	FertileEnd    *string  `json:"fertile_end"`
	OvulationDate *string  `json:"ovulation_date"`
	BestDays      []string `json:"best_days"`
}

// CyclePhaseOf is the cycle_phase DTO for the snapshot's member:
//   - no data → main_phase null;
//   - fertile is shown only when the source knows fertility is not suppressed (hormonal contraception hides it, F-02)
//     and, in TTC mode, only when the member may read the ttc_fertility switch; otherwise it reads follicular;
//   - period_soon (next start within PeriodSoonDays, never while late) for the partner role only (F-01).
func CyclePhaseOf(f CycleFacts, s family.Snapshot, now time.Time) CyclePhase {
	var out CyclePhase
	if s.Role == family.RolePartner {
		soon := !f.Late && f.DaysToPeriod != nil && *f.DaysToPeriod >= 0 && *f.DaysToPeriod <= PeriodSoonDays
		out.PeriodSoon = &soon
	}
	if !f.HasData || f.Phase == "" {
		return out
	}
	phase := f.Phase
	if phase == enums.MainPhaseFertile {
		allowed := f.FertilityKnown && !f.FertilitySuppressed
		if allowed && s.OwnerMode == family.ModeOf(string(enums.LifeModeTTC)) {
			allowed = family.Decide(s, family.Subject{}, family.DomainTTCFertility, family.ActionRead, now).Allowed()
		}
		if !allowed {
			phase = enums.MainPhaseFollicular
		}
	}
	p := string(phase)
	out.MainPhase = &p
	return out
}

// TTCOf is the ttc_fertility DTO; everything is null under suppression or when the source cannot tell.
func TTCOf(f CycleFacts) TTCFertility {
	out := TTCFertility{BestDays: []string{}}
	if !f.FertilityKnown || f.FertilitySuppressed {
		return out
	}
	out.FertileStart, out.FertileEnd, out.OvulationDate = dateOrNil(f.FertileStart), dateOrNil(f.FertileEnd), dateOrNil(f.Ovulation)
	for _, d := range f.BestDays {
		out.BestDays = append(out.BestDays, d.String())
	}
	return out
}

// v1 is the v1.1 resolver (CYCLE_ENGINE=v1) for the phase. The resolver knows nothing of hormonal contraception, so
// the suppression comes from the engine input (the same I-03 rule the v6 engine applies) plus the stored modes without
// fertility copy (teen, menopause). The v1 resolver has no TTC window: TTCOf stays empty.
type v1 struct {
	svc    *cycleservice.Service
	inputs inputs.Querier
}

func v1Cycle(db *sql.DB) CycleSource {
	return v1{svc: cycleservice.New(db, nil), inputs: cyclestore.New(db)}
}

func (s v1) Facts(ctx context.Context, ownerID uint64, today civildate.Date) (CycleFacts, error) {
	sn, err := s.svc.Load(ctx, ownerID, today, today, today)
	if err != nil {
		return CycleFacts{}, err
	}
	profile := sn.EngineProfile()
	st := resolver.Resolve(sn.Histories, profile, today, today, metrics.Calculate(sn.Histories, profile))
	in, err := inputs.Load(ctx, s.inputs, ownerID, today)
	if err != nil {
		return CycleFacts{}, err
	}
	f := CycleFacts{HasData: len(sn.Histories) > 0 || sn.Profile != nil, Phase: st.MainPhase,
		Late: st.DaysLate != nil && *st.DaysLate > 0, FertilityKnown: true,
		FertilitySuppressed: in.Hormonal || (profile != nil && profile.NoFertilityCopy)}
	if !f.Late && st.DaysToPeriod != nil {
		v := *st.DaysToPeriod
		f.DaysToPeriod = &v
	}
	return f, nil
}

// v6Cycle is the v6 engine state (T-M9-36): main phase, the E-24 fertility suppression and the TTC window.
type v6Cycle struct{ states state.UserStates }

func (s v6Cycle) Facts(ctx context.Context, ownerID uint64, today civildate.Date) (CycleFacts, error) {
	es, err := s.states.ForUser(ctx, ownerID, today)
	if err != nil {
		return CycleFacts{}, err
	}
	t := engine.DayOf(today)
	st := es.Resolve(t, t)
	decl := es.Declared()
	f := CycleFacts{
		HasData: len(es.Periods()) > 0 || decl.LastPeriodStart.OK() || decl.CycleLength.OK() || decl.PeriodLength.OK(),
		Phase:   enums.MainPhase(st.MainPhase.String()), FertilityKnown: true,
		FertilitySuppressed: st.Limits.FertilitySuppressed || es.Mode() == engine.ModeTeen,
		Late:                st.DaysLate.Or(0) > 0,
	}
	if v, ok := st.DaysToPeriod.Get(); ok && !f.Late {
		f.DaysToPeriod = &v
	}
	if f.FertilitySuppressed {
		return f, nil
	}
	card, ok := es.TodayUI(st).PredictionCard.Get()
	if !ok {
		return f, nil
	}
	if ov, ok := card.Ovulation.Get(); ok {
		f.Ovulation = ov.Date.Civil()
	}
	if w, ok := card.Fertile.Get(); ok {
		f.FertileStart, f.FertileEnd = w.Start.Civil(), w.End.Civil()
		for _, d := range es.Range(w.Start, w.End, t) {
			if d.FertilityLevel == engine.FertilityHigh || d.FertilityLevel == engine.FertilityPeak {
				f.BestDays = append(f.BestDays, d.Date.Civil())
			}
		}
	}
	return f, nil
}
