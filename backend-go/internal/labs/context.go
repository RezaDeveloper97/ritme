package labs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// UserContext is what the interpretation may know about the user (nbl_Lab_Consent «چه چیزی ارسال می‌شود»: age, mode,
// medications — never a name, phone or id). Names are kept only to redact them from AI text.
type UserContext struct {
	Mode  enums.LifeMode
	Age   int    // 0 = unknown
	Phase string // approximate cycle phase (cycle / ttc / teen only), "" = unknown
	// PhaseOn is the day Phase is for: the lab's own date under the v6 engine (labPhase); zero under CYCLE_ENGINE=v1,
	// where Phase is today's v1 estimate (phaseOn).
	PhaseOn     civildate.Date
	PeriodDays  int // the profile's period length (0 = unknown)
	Medications []string
	Names       []string
}

// CycleStates hands out a user's prepared v6 engine state for the user-local day (state.Source satisfies it). nil
// means CYCLE_ENGINE=v1: the phase stays the profile-LMP estimate of today (phaseOn).
type CycleStates interface {
	ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error)
}

// maxRollCycles (v1 only): the phase is estimated from the last period start rolled forward by at most this many cycles.
const maxRollCycles = 3

// userContext reads the context of userID on today.
func (s *Service) userContext(ctx context.Context, userID uint64, today civildate.Date) (UserContext, error) {
	row, err := s.q.GetLabUserContext(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return UserContext{}, fmt.Errorf("labs: user context: %w", err)
	}
	uc := UserContext{Mode: enums.ResolveLifeMode(row.LifeMode.String, row.Pregnant != 0, row.UserGoal.String)}
	if row.Name.Valid && strings.TrimSpace(row.Name.String) != "" {
		uc.Names = []string{row.Name.String}
	}
	if row.Birthday.Valid {
		uc.Age = ageOn(row.Birthday.Date, today)
	}
	if row.PeriodDuration.Valid && row.PeriodDuration.Int16 > 0 {
		uc.PeriodDays = int(row.PeriodDuration.Int16)
	}
	switch uc.Mode {
	case enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModeTeen:
		if s.cycle == nil && row.LastPeriodStart.Valid && row.CycleDuration.Valid { // v6: labPhase
			uc.Phase = phaseOn(row.LastPeriodStart.Date, int(row.CycleDuration.Int16), uc.PeriodDays, today)
		}
	}
	meds, err := s.q.ListLabMedicationNames(ctx, userID)
	if err != nil {
		return UserContext{}, fmt.Errorf("labs: medications: %w", err)
	}
	for _, m := range meds {
		if m = strings.TrimSpace(m); m != "" {
			uc.Medications = append(uc.Medications, m)
		}
	}
	return uc, nil
}

func ageOn(birth, today civildate.Date) int {
	age := today.Year - birth.Year
	if today.Month < birth.Month || (today.Month == birth.Month && today.Day < birth.Day) {
		age--
	}
	if age < 0 || age > 120 {
		return 0
	}
	return age
}

// labPhase is the v6 engine's phase on the lab's own date (T-M9-37, DECISIONS.md I-10): the content phase of the
// day's subphase (I-11: menstruation / follicular / ovulation / luteal), for the cycle, ttc and teen modes only, ""
// when the engine resolves the day as unknown (no data then, pill, pregnancy, unresolved lateness). today is the
// user-local day the state is prepared for.
func (s *Service) labPhase(ctx context.Context, userID uint64, mode enums.LifeMode, on, today civildate.Date) (string, error) {
	switch mode {
	case enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModeTeen:
	default:
		return "", nil
	}
	if on.IsZero() {
		return "", nil
	}
	st, err := s.cycle.ForUser(ctx, userID, today)
	if err != nil {
		return "", fmt.Errorf("labs: engine state: %w", err)
	}
	return phaseOfDay(st, on, today), nil
}

// phaseOfDay is the content phase the engine resolves for day ("" = unknown).
func phaseOfDay(st *engine.State, day, today civildate.Date) string {
	status := st.Resolve(engine.DayOf(day), engine.DayOf(today))
	if p, ok := view.ContentPhase(status.Subphase); ok {
		return string(p)
	}
	return ""
}

// phaseOn (CYCLE_ENGINE v1 only; retired with the v1.1 engine, T-M9-51) is the approximate phase on today from the last period start and the typical cycle / period length.
func phaseOn(start civildate.Date, cycleLen, periodLen int, today civildate.Date) string {
	if cycleLen < 15 || cycleLen > 90 {
		return ""
	}
	if periodLen <= 0 || periodLen >= cycleLen {
		periodLen = 5
	}
	day := start.DiffDays(today) // days since the start (0 = day 1)
	if day < 0 || day >= cycleLen*(maxRollCycles+1) {
		return ""
	}
	cycleDay := day%cycleLen + 1
	return string(resolver.PhaseMapper{}.PhaseFor(cycleDay, max(1, cycleLen-14), periodLen))
}

// ageBand is the age as a 5-year band for prompts ("30–34"), "" when unknown.
func ageBand(age int) string {
	if age <= 0 {
		return ""
	}
	lo := age / 5 * 5
	return fmt.Sprintf("%d–%d", lo, lo+4)
}
