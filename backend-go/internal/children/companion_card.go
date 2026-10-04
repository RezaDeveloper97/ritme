package children

import (
	"context"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// CompanionCard is the «فرزند» card of the companion home (nbl_Hamdam_Home: «آوا · ۳ ساله · واکسن بعدی: ۱۲ آبان»)
// for one spouse link: the children the owner shares with viewerID, youngest first, each with age and next vaccine
// visit. nil when nothing is shared. The read is written to the owner's companion audit trail first (fail closed).
func (s *Service) CompanionCard(ctx context.Context, viewerID, ownerID, companionID uint64, locale string, now time.Time) (any, error) {
	kids, err := s.SharedWith(ctx, viewerID, ownerID)
	if err != nil || len(kids) == 0 {
		return nil, err
	}
	if err := s.audit(ctx, ownerID, viewerID, companionID, now); err != nil {
		return nil, err
	}
	today := civildate.InTehran(now)
	items := make([]*jsonx.OrderedMap, 0, len(kids))
	for _, c := range kids {
		age := AgeOn(c.BirthDate, today)
		sch, err := s.Schedule(ctx, c, today)
		if err != nil {
			return nil, err
		}
		var next any
		if v := sch.Next; v != nil {
			next = jsonx.Obj(
				"code", v.Code, "label", VisitLabel(v.AgeMonths, locale), "due_date", v.DueDate.String(),
				"days_left", today.DiffDays(v.DueDate), "status", v.Status,
			)
		}
		items = append(items, jsonx.Obj(
			"id", c.ID, "name", c.Name, "initial", initial(c.Name), "sex", nullStr(c.Sex),
			"age", jsonx.Obj("months", age.Months, "years", age.Years, "label", age.Label(locale)),
			"next_vaccine", next,
		))
	}
	return jsonx.Obj("count", len(items), "items", items), nil
}
