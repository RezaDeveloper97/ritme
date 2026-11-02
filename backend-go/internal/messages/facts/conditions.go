package facts

import (
	"context"
)

// Condition nudge facts (T-M10-09, D-166; SPEC §5.4.1): the signals the condition nudges engine detected per request
// (internal/messages/conditionnudges, D-41) as facts, so each nudge is a message rule (rule_key cond.<name>) the admin
// can switch off, re-word or re-threshold. A qualifying day is still the nudge engine's day rule (pain ≥ 7, or level
// severe when unscored; flow heavy or very heavy); how many days raise a nudge (2) and the program that silences it
// are conditions of the seeded rules.
//
// The day counts are present whenever the input is: 0 outside a nudged mode (cycle, ttc, teen) or without a current
// cycle, so an open inbox row resolves when the cycle turns over or the mode changes.

// ConditionInput is one user's condition nudge input on the target day (conditionnudges builds it).
type ConditionInput struct {
	// PainDays / BleedingDays are the qualifying days of the current cycle window (0 without one).
	PainDays, BleedingDays int
	// Programs are the condition programs she is enrolled in (conditions program codes).
	Programs []string
}

// ConditionSource reads a user's condition nudge input on the target day (nil = every fact absent). Any Sources field
// that also implements it produces the group.
type ConditionSource interface {
	ConditionInput(ctx context.Context) (*ConditionInput, error)
}

// GroupConditions is the condition nudge producer group.
const GroupConditions = "conditions"

func init() {
	g := GroupConditions
	Register(
		Def{Name: "cond.heavy_pain_days", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("روزهای چرخهٔ جاری با درد ۷ یا بیشتر (یا «شدید» بدون نمره)؛ بیرون از چرخه ۰",
				"Days of the current cycle with pain of 7 or more (or severe when unscored); 0 without a current cycle.")},
		Def{Name: "cond.heavy_bleeding_days", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("روزهای چرخهٔ جاری با خونریزی سنگین یا خیلی سنگین؛ بیرون از چرخه ۰",
				"Days of the current cycle with heavy or very heavy flow; 0 without a current cycle.")},
		Def{Name: "cond.programs", Type: TypeSet, Group: g,
			Description: desc("برنامه\u200cهای بیماری که کاربر عضوشان است (کد برنامه)",
				"The condition programs the user is enrolled in (program codes).")},
		Def{Name: "cond_pain_days", Type: TypeInt, Group: g, Variable: true,
			Description: desc("متغیر متن: روزهای درد شدید", "Text variable: the strong-pain days.")},
		Def{Name: "cond_bleeding_days", Type: TypeInt, Group: g, Variable: true,
			Description: desc("متغیر متن: روزهای خونریزی سنگین", "Text variable: the heavy-flow days.")},
	)
	RegisterGroup(g, func(ctx context.Context, f *Facts) (Values, error) {
		src, ok := sourceOf[ConditionSource](f.src)
		if !ok {
			return nil, nil
		}
		in, err := src.ConditionInput(ctx)
		if err != nil || in == nil {
			return nil, err
		}
		return ConditionValues(in), nil
	})
}

// ConditionValues are the condition nudge facts and text variables of in.
func ConditionValues(in *ConditionInput) Values {
	programs := append([]string{}, in.Programs...)
	return Values{
		"cond.heavy_pain_days":     float64(in.PainDays),
		"cond.heavy_bleeding_days": float64(in.BleedingDays),
		"cond.programs":            programs,
		"cond_pain_days":           float64(in.PainDays),
		"cond_bleeding_days":       float64(in.BleedingDays),
	}
}
