package care

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Public API for other domains: a visit booked through the telemedicine domain (bloom B-N7-03) becomes an ordinary
// care appointment of the user — listed in /care/appointments, its reminder fires remind_before ahead, the user may
// edit or cancel it like any other. The functions take the caller's transaction-bound queries so booking and
// appointment commit together.

// TopicConsult is the appointment topic of a doctor visit.
const TopicConsult = "consult"

// BookedVisit is one appointment created for a booking.
type BookedVisit struct {
	BookingID    uint64
	Title        string // already localized by the caller (the user's request locale)
	Kind         string // one of AppointmentKinds
	With         string // doctor name (optional)
	Specialty    string // specialty label (optional)
	Location     string // clinic address for in-person visits (optional)
	RemindBefore string // one of RemindBefore
	ScheduledAt  time.Time
}

func optionalText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ScheduleVisit inserts the appointment of a booking and returns its id.
func ScheduleVisit(ctx context.Context, q *store.Queries, userID uint64, v BookedVisit, now time.Time) (uint64, error) {
	bookingID := v.BookingID
	m := AppointmentMeta{
		V: MetaVersion, Kind: v.Kind, With: optionalText(v.With), Specialty: optionalText(v.Specialty),
		Topic: TopicConsult, Location: optionalText(v.Location), RemindBefore: v.RemindBefore, AddToCalendar: true,
		Prep: []PrepItem{}, Status: StatusScheduled, BookingID: &bookingID,
	}
	meta, err := json.Marshal(m)
	if err != nil {
		return 0, fmt.Errorf("care: visit meta: %w", err)
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	id, err := q.InsertAppointment(ctx, store.InsertAppointmentParams{
		UserID: userID, Title: v.Title, Subtitle: m.Subtitle(), ScheduledAt: sql.NullTime{Time: v.ScheduledAt, Valid: true},
		IsActive: true, Meta: rootdb.NullRawJSON{V: meta, Valid: true}, CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil {
		return 0, fmt.Errorf("care: schedule visit: %w", err)
	}
	return uint64(id), nil //nolint:gosec // G115: auto-increment id
}

// MoveVisit moves the user's appointment id to at (a rescheduled booking) and re-opens it when the user had
// cancelled it; her own edits (title, notes, reminder offset) are kept. A deleted appointment is not an error.
func MoveVisit(ctx context.Context, q *store.Queries, userID, id uint64, at, now time.Time) error {
	return updateVisit(ctx, q, userID, id, now, func(a *Appointment) {
		a.Row.ScheduledAt = sql.NullTime{Time: at, Valid: true}
		a.Meta.Status = StatusScheduled
		a.Row.IsActive = true
	})
}

// CancelVisit cancels the user's appointment id (a cancelled booking): status cancelled, reminder off, row kept.
// A deleted appointment is not an error.
func CancelVisit(ctx context.Context, q *store.Queries, userID, id uint64, now time.Time) error {
	return updateVisit(ctx, q, userID, id, now, func(a *Appointment) {
		a.Meta.Status = StatusCancelled
		a.Row.IsActive = false
	})
}

func updateVisit(ctx context.Context, q *store.Queries, userID, id uint64, now time.Time, change func(*Appointment)) error {
	row, err := q.GetAppointment(ctx, store.GetAppointmentParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("care: visit: %w", err)
	}
	a := ParseAppointment(row)
	change(&a)
	meta, err := json.Marshal(a.Meta)
	if err != nil {
		return fmt.Errorf("care: visit meta: %w", err)
	}
	if err := q.UpdateAppointment(ctx, store.UpdateAppointmentParams{
		Title: a.Row.Title, Subtitle: a.Row.Subtitle, Notes: a.Row.Notes, ScheduledAt: a.Row.ScheduledAt,
		IsActive: a.Row.IsActive, Meta: rootdb.NullRawJSON{V: meta, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true}, ID: a.Row.ID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("care: update visit: %w", err)
	}
	return nil
}

// PrescribedMedication is one e-prescription line (bloom B-N7-04) the user turns into a medication reminder with one
// tap: every day at Times, from StartsOn for DurationDays days (0 = ongoing). The caller validated the fields with
// the medication limits (title ≤ 255, dose ≤ 50, Forms, 1–4 "HH:MM" times).
type PrescribedMedication struct {
	Title        string
	Dose         string // optional
	Form         string // one of Forms
	Times        []string
	DurationDays int
	Notes        string // the doctor's instructions (optional)
	StartsOn     civildate.Date
}

// ErrMedicationLimit: the user already keeps MaxMedications medication reminders.
var ErrMedicationLimit = errors.New("care: medication limit reached")

// AddPrescribedMedication inserts the medication reminder of a prescription line and returns its id. subtitleLocale
// is the default language (the legacy subtitle column, like POST /care/medications).
func AddPrescribedMedication(ctx context.Context, q *store.Queries, userID uint64, m PrescribedMedication, subtitleLocale string, now time.Time) (uint64, error) {
	rows, err := q.ListMedications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("care: count medications: %w", err)
	}
	if len(rows) >= MaxMedications {
		return 0, ErrMedicationLimit
	}
	times := slices.Clone(m.Times)
	slices.Sort(times)
	times = slices.Compact(times)
	meta := MedicationMeta{V: MetaVersion, Dose: optionalText(m.Dose), Form: m.Form, Times: times,
		Weekdays: slices.Clone(AllWeekdays), Amount: 1, Duration: DurationOngoing, Notify: true}
	ends := civildate.NullDate{}
	if m.DurationDays > 0 {
		meta.Duration = DurationUntilDate
		ends = civildate.NullDate{Date: m.StartsOn.AddDays(m.DurationDays - 1), Valid: true}
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return 0, fmt.Errorf("care: medication meta: %w", err)
	}
	notes := sql.NullString{String: m.Notes, Valid: m.Notes != ""}
	stamp := sql.NullTime{Time: now, Valid: true}
	id, err := q.InsertMedication(ctx, store.InsertMedicationParams{
		UserID: userID, Title: m.Title, Subtitle: meta.Subtitle(subtitleLocale), Notes: notes,
		Recurrence: meta.Recurrence(), RecurrenceTime: meta.RecurrenceTime(),
		StartsOn: civildate.NullDate{Date: m.StartsOn, Valid: true}, EndsOn: ends, IsActive: true,
		Meta: rootdb.NullRawJSON{V: raw, Valid: true}, CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("care: insert prescribed medication (id %d): %w", id, err)
	}
	return uint64(id), nil
}
