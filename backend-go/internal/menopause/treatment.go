package menopause

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Treatment kinds and units (treatment_items, CB-MENO-01). The home reads the items and their intakes here; the
// treatment endpoints (CB-MENO-03) are in treatment_items.go / treatment_screen.go.
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
	intakes, err := s.itemIntakes(ctx, userID, items, from, today)
	if err != nil {
		return nil, err
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
		for day, amount := range intakes[it.ID] {
			t.DaysTaken++
			if day == today {
				t.TakenToday = true
			}
			t.Amount += amountOf(it, amount)
		}
		out = append(out, t)
	}
	return out, nil
}

// amountOf is what one intake adds to a weekly total: its minutes or sessions logged as a number, else one session
// (or one intake); a minutes goal without minutes adds nothing.
func amountOf(it store.TreatmentItem, amount sql.NullInt16) int {
	switch {
	case amount.Valid && amount.Int16 > 0:
		return int(amount.Int16)
	case it.GoalUnit.String != UnitMinutes:
		return 1
	}
	return 0
}

// itemIntakes maps item id → taken day → amount over [from, to]: the item's own intakes plus the days its care
// medication reminder was ticked in /care (no amount), so both screens agree.
func (s *Service) itemIntakes(ctx context.Context, userID uint64, items []store.TreatmentItem, from, to civildate.Date,
) (map[uint64]map[civildate.Date]sql.NullInt16, error) {
	rows, err := s.q.ListTreatmentIntakesInRange(ctx, store.ListTreatmentIntakesInRangeParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("menopause: list intakes: %w", err)
	}
	out := map[uint64]map[civildate.Date]sql.NullInt16{}
	add := func(id uint64, day civildate.Date, amount sql.NullInt16) {
		if out[id] == nil {
			out[id] = map[civildate.Date]sql.NullInt16{}
		}
		if _, seen := out[id][day]; !seen || amount.Valid {
			out[id][day] = amount
		}
	}
	for _, r := range rows {
		add(r.TreatmentItemID, r.IntakeDate, r.Amount)
	}
	byReminder := map[uint64]uint64{}
	for _, it := range items {
		if it.ReminderID.Valid {
			byReminder[uint64(it.ReminderID.Int64)] = it.ID //nolint:gosec // FK id
		}
	}
	if len(byReminder) == 0 {
		return out, nil
	}
	ticks, err := s.q.ListReminderIntakeDays(ctx, store.ListReminderIntakeDaysParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("menopause: list care intakes: %w", err)
	}
	for _, t := range ticks {
		if id, ok := byReminder[t.ReminderID]; ok {
			add(id, t.IntakeDate, sql.NullInt16{})
		}
	}
	return out, nil
}
