package menopause

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Treatment kinds and units (treatment_items, CB-MENO-01). The treatment endpoints are CB-MENO-03's; the home only
// reads the items and their intakes here.
const (
	KindHRT       = "hrt"
	KindLifestyle = "lifestyle"

	ScheduleWeekly = "weekly"
	UnitMinutes    = "minutes"

	// AdherenceDays is the home card's window: «۶ از ۷ روز این هفته» = the last 7 days, today included.
	AdherenceDays = 7
)

// activeOn: started on or before day (or no start date) and not stopped by then.
func activeOn(it store.TreatmentItem, day civildate.Date) bool {
	if it.StartedOn.Valid && it.StartedOn.Date.After(day) {
		return false
	}
	return !it.StoppedOn.Valid || it.StoppedOn.Date.After(day)
}

// TreatmentToday is one active item on the home's treatment card.
type TreatmentToday struct {
	Item       store.TreatmentItem
	TakenToday bool
	// DaysTaken of Days: intake days in the last AdherenceDays days (Days shrinks for an item started inside the
	// window). Daily items only; weekly and lifestyle items report Amount against their weekly goal instead.
	DaysTaken, Days int
	// Amount is the window's total: minutes (goal_unit minutes) or sessions.
	Amount int
}

// Daily reports whether adherence is counted per day (not a weekly schedule or a lifestyle goal).
func (t TreatmentToday) Daily() bool {
	weekly := t.Item.Schedule.Valid && t.Item.Schedule.String == ScheduleWeekly
	return t.Item.Kind != KindLifestyle && !weekly
}

// AdherencePct is DaysTaken / Days in whole percent (nil for non-daily items or an empty window).
func (t TreatmentToday) AdherencePct() *int {
	if !t.Daily() || t.Days == 0 {
		return nil
	}
	p := (t.DaysTaken*100 + t.Days/2) / t.Days
	return &p
}

// Treatment is the active items with their last-7-days intakes, in treatment order (kind, sort order).
func (s *Service) Treatment(ctx context.Context, userID uint64, today civildate.Date) ([]TreatmentToday, error) {
	items, err := s.q.ListTreatmentItems(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("menopause: list treatment: %w", err)
	}
	from := today.AddDays(-(AdherenceDays - 1))
	intakes, err := s.q.ListTreatmentIntakesInRange(ctx, store.ListTreatmentIntakesInRangeParams{UserID: userID, FromDate: from, ToDate: today})
	if err != nil {
		return nil, fmt.Errorf("menopause: list intakes: %w", err)
	}
	out := []TreatmentToday{}
	for _, it := range items {
		if !activeOn(it, today) {
			continue
		}
		t := TreatmentToday{Item: it, Days: AdherenceDays}
		if it.StartedOn.Valid && it.StartedOn.Date.After(from) {
			t.Days = it.StartedOn.Date.DiffDays(today) + 1
		}
		for _, in := range intakes {
			if in.TreatmentItemID != it.ID {
				continue
			}
			t.DaysTaken++
			if in.IntakeDate == today {
				t.TakenToday = true
			}
			switch {
			case in.Amount.Valid && in.Amount.Int16 > 0:
				t.Amount += int(in.Amount.Int16) // minutes, or sessions logged as a number
			case it.GoalUnit.String != UnitMinutes:
				t.Amount++ // one session (or one intake)
			}
		}
		out = append(out, t)
	}
	return out, nil
}
