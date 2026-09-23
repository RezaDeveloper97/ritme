package care

import (
	"cmp"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Dose is one scheduled medication slot of a day.
type Dose struct {
	ReminderID uint64
	Title      string
	Form       string
	Slot       string
	Taken      bool
}

// TodayDoses are the doses of day: one per slot of every active medication whose
// weekday/start/end window covers day, sorted by slot (then reminder id); taken when an
// intake row exists for (reminder, slot).
func TodayDoses(day civildate.Date, meds []store.Reminder, intakes []store.ListIntakesOnDateRow) []Dose {
	type key struct {
		id   uint64
		slot string
	}
	taken := make(map[key]bool, len(intakes))
	for _, in := range intakes {
		taken[key{in.ReminderID, in.Slot}] = true
	}
	doses := []Dose{}
	for _, r := range meds {
		if !r.IsActive {
			continue
		}
		m := ParseMedication(r)
		if !m.Covers(day) {
			continue
		}
		title := r.Title
		if r.Subtitle.Valid && r.Subtitle.String != "" {
			title += " " + r.Subtitle.String
		}
		for _, slot := range m.Meta.Times {
			doses = append(doses, Dose{
				ReminderID: r.ID, Title: title, Form: m.Meta.Form, Slot: slot, Taken: taken[key{r.ID, slot}],
			})
		}
	}
	slices.SortStableFunc(doses, func(a, b Dose) int {
		return cmp.Or(cmp.Compare(a.Slot, b.Slot), cmp.Compare(a.ReminderID, b.ReminderID))
	})
	return doses
}

// NextAppointment is the earliest scheduled (not cancelled) appointment at or after now,
// from rows ordered scheduled_at ASC; nil when none. A reminder switched off still counts.
func NextAppointment(rows []store.Reminder, now time.Time) *Appointment {
	for _, r := range rows {
		if a := ParseAppointment(r); a.Upcoming(now) {
			return &a
		}
	}
	return nil
}

// TodayJSON is the data of GET /care/today.
func TodayJSON(day civildate.Date, doses []Dose, next *Appointment, now time.Time) *jsonx.OrderedMap {
	list := make([]*jsonx.OrderedMap, 0, len(doses))
	takenCount := 0
	for _, d := range doses {
		if d.Taken {
			takenCount++
		}
		list = append(list, jsonx.Obj(
			"reminder_id", d.ReminderID,
			"title", d.Title,
			"form", d.Form,
			"slot", d.Slot,
			"taken", d.Taken,
		))
	}
	var appointment any
	if next != nil {
		// Upcoming guarantees scheduled_at is set.
		at := next.Row.ScheduledAt.Time.In(civildate.Tehran)
		appointment = jsonx.Obj(
			"id", next.Row.ID,
			"kind", next.Meta.Kind,
			"title", next.Row.Title,
			"with", next.Meta.With,
			"scheduled_at", at.Format(wallClock),
			"days_until", civildate.InTehran(now).DiffDays(civildate.FromTime(at)),
			"location", next.Meta.Location,
			"remind_before", next.Meta.RemindBefore,
		)
	}
	return jsonx.Obj(
		"date", day.String(),
		"doses", list,
		"taken_count", takenCount,
		"total", len(doses),
		"next_appointment", appointment,
	)
}

// Today is GET /care/today?date=Y-m-d (default today, Tehran): the day's doses with their
// taken state and the next appointment, in three queries (medications, intakes, appointments).
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Query(c),
		validation.Rules{validation.F("date", "nullable", "date_format:Y-m-d")},
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	day := civildate.InTehran(now)
	if raw, _ := v.Validated().Get("date"); raw != nil && phpval.ToString(raw) != "" {
		if day, err = civildate.Parse(phpval.ToString(raw)); err != nil {
			return fmt.Errorf("care: today date: %w", err)
		}
	}

	meds, err := h.q.ListActiveMedications(c, userID)
	if err != nil {
		return fmt.Errorf("care: today medications: %w", err)
	}
	intakes, err := h.q.ListIntakesOnDate(c, store.ListIntakesOnDateParams{UserID: userID, IntakeDate: day})
	if err != nil {
		return fmt.Errorf("care: today intakes: %w", err)
	}
	appts, err := h.q.ListAppointmentsFrom(c, store.ListAppointmentsFromParams{
		UserID: userID, ScheduledAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("care: today appointments: %w", err)
	}
	return httpx.OK(c, TodayJSON(day, TodayDoses(day, meds, intakes), NextAppointment(appts, now), now))
}
