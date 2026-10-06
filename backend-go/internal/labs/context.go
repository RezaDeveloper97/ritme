package labs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// UserContext is what the interpretation may know about the user (nbl_Lab_Consent «چه چیزی ارسال می‌شود»: age, mode,
// medications — never a name, phone or id). Names are kept only to redact them from AI text.
type UserContext struct {
	Mode        enums.LifeMode
	Age         int    // 0 = unknown
	Phase       string // approximate cycle phase today (cycle / ttc / teen only), "" = unknown
	PeriodDays  int    // the profile's period length (0 = unknown)
	Medications []string
	Names       []string
}

// maxRollCycles: the phase is estimated from the last period start rolled forward by at most this many cycles.
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
		if row.LastPeriodStart.Valid && row.CycleDuration.Valid {
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

// phaseOn is the approximate phase on today from the last period start and the typical cycle / period length.
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
