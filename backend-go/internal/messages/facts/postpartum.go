package facts

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// Postpartum facts: the 1-based week after birth (postpartum guide StatusOn, the week the postpartum screens show),
// only while the user is in postpartum mode. Absent without a birth date or with a birth date after the target day.

// PostpartumSource reads the user's birth date; ok is false when she is not in postpartum mode or has none.
type PostpartumSource interface {
	PostpartumBirth(ctx context.Context) (birth civildate.Date, ok bool, err error)
}

// GroupPostpartum is the postpartum producer group.
const GroupPostpartum = "postpartum"

func init() {
	Register(Def{Name: "postpartum_week", Type: TypeInt, Unit: "week", Group: GroupPostpartum, MemberSafe: true,
		Description: desc("هفته بعد از زایمان (از ۱)", "The week after birth (from 1).")})
	RegisterGroup(GroupPostpartum, func(ctx context.Context, f *Facts) (Values, error) {
		if f.src.Postpartum == nil {
			return nil, nil
		}
		birth, ok, err := f.src.Postpartum.PostpartumBirth(ctx)
		if err != nil || !ok || birth.After(f.today) {
			return nil, err
		}
		return Values{"postpartum_week": float64(guide.StatusOn(birth, f.today).Week)}, nil
	})
}
