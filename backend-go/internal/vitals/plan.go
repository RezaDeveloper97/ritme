package vitals

import (
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// Weekly measurement plan «برنامه اندازه‌گیری این هفته» (nbl_Vitals_Hub: «۴ از ۷ · فشار · صبح · فشار · شب ·
// قند · ناشتا»). An item is (type, slot) on some weekdays with an optional reminder time. A planned day is done when a
// timed reading of the type falls in the slot that day: bp / hr morning = 04:00–11:59, evening = 18:00–03:59 (the
// reading's own day); glucose slots match the reading's context.

// Slots.
const (
	SlotMorning = "morning"
	SlotEvening = "evening"
)

// PlanSlots per type.
var PlanSlots = map[string][]string{
	TypeBP:      {SlotMorning, SlotEvening},
	TypeHR:      {SlotMorning, SlotEvening},
	TypeGlucose: {ContextFasting, ContextBeforeMeal, ContextAfterMeal, ContextBedtime},
}

// MaxPlanItems caps the plan (every type × slot = 8).
const MaxPlanItems = 8

// AllDays is the bitmask of every weekday.
const AllDays uint8 = 0x7f

// ReminderCategory is the notification category of plan reminders (off by default: opt-in).
const ReminderCategory = notifications.Vitals

// PlanItem is one plan row.
type PlanItem struct {
	Type, Slot string
	Days       uint8  // bit 0 = Saturday … bit 6 = Friday
	RemindAt   string // HH:MM, "" = no reminder
}

// WeekdayIndex is d's weekday with Saturday = 0 (the Iranian week).
func WeekdayIndex(d civildate.Date) int { return (int(d.Weekday()) + 1) % 7 }

// On reports whether the item is planned on d.
func (p PlanItem) On(d civildate.Date) bool { return p.Days&(1<<WeekdayIndex(d)) != 0 }

// DayList is the weekday indexes of the mask.
func (p PlanItem) DayList() []int {
	out := []int{}
	for i := range 7 {
		if p.Days&(1<<i) != 0 {
			out = append(out, i)
		}
	}
	return out
}

// Matches reports whether a timed reading fulfils the item on its day.
func (p PlanItem) Matches(r Reading) bool {
	if r.Type != p.Type || !r.Timed() {
		return false
	}
	if p.Type == TypeGlucose {
		return r.Context == p.Slot
	}
	switch p.Slot {
	case SlotMorning:
		return r.Period() == PeriodMorning
	case SlotEvening:
		return r.Period() == PeriodNight
	}
	return false
}

// PlanFromRows maps the stored rows.
func PlanFromRows(rows []store.VitalPlanItem) []PlanItem {
	out := make([]PlanItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, PlanItem{Type: r.Type, Slot: r.Slot, Days: r.Days, RemindAt: strOf(r.RemindAt)})
	}
	return out
}

// done reports whether a reading fulfils item on d.
func done(item PlanItem, d civildate.Date, rs []Reading) bool {
	for _, r := range rs {
		if r.Date == d && item.Matches(r) {
			return true
		}
	}
	return false
}

// WeekJSON is the plan's progress over the week (Saturday–Friday) containing today: planned / done counts per item and
// in total, with per-day state (done, missed = a past planned day without a reading, due = today or later).
func WeekJSON(items []PlanItem, rs []Reading, today civildate.Date) *jsonx.OrderedMap {
	from := today.StartOfWeek()
	planned, doneN := 0, 0
	list := make([]any, 0, len(items))
	for _, it := range items {
		ip, id := 0, 0
		days := []any{}
		for i := range 7 {
			d := from.AddDays(i)
			if !it.On(d) {
				continue
			}
			ip++
			state := "due"
			switch {
			case done(it, d, rs):
				state = "done"
				id++
			case d.Before(today):
				state = "missed"
			}
			days = append(days, jsonx.Obj("date", d.String(), "state", state))
		}
		planned += ip
		doneN += id
		list = append(list, jsonx.Obj("type", it.Type, "slot", it.Slot, "planned", ip, "done", id, "days", days))
	}
	return jsonx.Obj("from", from.String(), "to", from.AddDays(6).String(), "planned", planned, "done", doneN,
		"items", list)
}

// PlanItemJSON is one stored item.
func PlanItemJSON(it PlanItem) *jsonx.OrderedMap {
	return jsonx.Obj("type", it.Type, "slot", it.Slot, "days", it.DayList(), "remind_at", strNull(it.RemindAt))
}

func minuteOf(hhmm string) (int, bool) {
	if len(hhmm) != 5 || hhmm[2] != ':' {
		return 0, false
	}
	h, err1 := strconv.Atoi(hhmm[:2])
	m, err2 := strconv.Atoi(hhmm[3:])
	if err1 != nil || err2 != nil || h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// Occurrence is one plan reminder due on a day.
type Occurrence struct {
	Item     PlanItem
	At       time.Time // Tehran wall-clock
	Decision notifications.Decision
	Message  notifications.Message
}

// PlanReminders lists the plan reminders of day: items planned that day with a reminder time whose slot is not
// already done (rs = the day's readings). Each one goes through notifications.Decide under the vitals category at its
// own time (switched-off category, quiet hours) and its copy is rendered later through notifications.Render, so «متن
// خنثی» hides the vital on the lock screen. There is no push sender in the API yet: a sender asks this for the pushes
// of a day, like children.PlanReminders and reminders.Plan.
func PlanReminders(p notifications.Preferences, items []PlanItem, rs []Reading, day civildate.Date, locale string) []Occurrence {
	var out []Occurrence
	for _, it := range items {
		minute, ok := minuteOf(it.RemindAt)
		if !ok || !it.On(day) || done(it, day, rs) {
			continue
		}
		at := day.TehranMidnight().Add(time.Duration(minute) * time.Minute)
		params := map[string]string{"what": Tp("labels.type."+it.Type, nil, locale), "slot": Tp("labels.slot."+it.Slot, nil, locale)}
		out = append(out, Occurrence{
			Item: it, At: at,
			Decision: notifications.Decide(p, ReminderCategory, at, time.Time{}),
			Message: notifications.Message{
				Category: ReminderCategory,
				Title:    Tp("reminder.title", params, locale),
				Body:     Tp("reminder.body", params, locale),
				URL:      "/vitals",
			},
		})
	}
	return out
}

// validSlot reports whether slot belongs to typ.
func validSlot(typ, slot string) bool { return slices.Contains(PlanSlots[typ], slot) }
