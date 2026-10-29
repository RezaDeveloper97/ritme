package facts

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Pregnancy facts: the gestational week and trimester on the target day from the pregnancy dating the pregnancy
// screens use (pregnancy v2 Resolve: LMP, ultrasound or manual entry), so a message and the pregnancy UI never
// disagree. Absent without an active, dated pregnancy.

// PregnancyDating is the user's active pregnancy on a day.
type PregnancyDating struct {
	Week      int // 1-based gestational week
	Trimester int // 1..3
}

// PregnancySource reads the active pregnancy's dating on day (nil = no active, dated pregnancy).
type PregnancySource interface {
	PregnancyDating(ctx context.Context, day civildate.Date) (*PregnancyDating, error)
}

// GroupPregnancy is the pregnancy producer group.
const GroupPregnancy = "pregnancy"

func init() {
	g := GroupPregnancy
	Register(
		Def{Name: "pregnancy_week", Type: TypeInt, Unit: "week", Group: g, MemberSafe: true,
			Description: desc("هفته بارداری", "The gestational week.")},
		Def{Name: "pregnancy_trimester", Type: TypeInt, Group: g, MemberSafe: true,
			Description: desc("سه\u200cماهه بارداری (۱ تا ۳)", "The pregnancy trimester (1–3).")},
	)
	RegisterGroup(g, func(ctx context.Context, f *Facts) (Values, error) {
		if f.src.Pregnancy == nil {
			return nil, nil
		}
		d, err := f.src.Pregnancy.PregnancyDating(ctx, f.today)
		if err != nil || d == nil {
			return nil, err
		}
		return Values{"pregnancy_week": float64(d.Week), "pregnancy_trimester": float64(d.Trimester)}, nil
	})
}
