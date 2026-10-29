package facts

import "context"

// Contraception facts (DECISIONS.md I-03): hormonal = an active combined_pill, progestin_pill, hormonal_iud, injection
// or implant; copper_iud, condom and other are not; without a saved method the onboarding contraceptive_pill answer
// counts (contraception.HormonalActive — the same rule the engine input reads).

// ContraceptionSource reports whether the user is on hormonal contraception (I-03).
type ContraceptionSource interface {
	HormonalContraception(ctx context.Context) (bool, error)
}

// GroupContraception is the contraception producer group.
const GroupContraception = "contraception"

func init() {
	Register(Def{Name: "hormonal_contraception", Type: TypeBool, Group: GroupContraception,
		Description: desc("پیشگیری هورمونی فعال (I-03)", "Active hormonal contraception (I-03).")})
	RegisterGroup(GroupContraception, func(ctx context.Context, f *Facts) (Values, error) {
		if f.src.Contraception == nil {
			return nil, nil
		}
		on, err := f.src.Contraception.HormonalContraception(ctx)
		if err != nil {
			return nil, err
		}
		return Values{"hormonal_contraception": on}, nil
	})
}
