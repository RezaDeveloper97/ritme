package care

import (
	"database/sql"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// TypeAppointment is the reminders.type of a doctor appointment.
const TypeAppointment = "appointment"

// Appointment statuses.
const (
	StatusScheduled = "scheduled"
	StatusCancelled = "cancelled"
)

// Appointment defaults (a POST without the key).
const (
	DefaultRemindBefore = "1d"
	DefaultTopic        = "other"
)

// AppointmentStatuses are the appointment statuses.
var AppointmentStatuses = []string{StatusScheduled, StatusCancelled}

// wallClock is the Tehran wall-clock datetime format of scheduled_at / remind_at.
const wallClock = "2006-01-02 15:04:05"

// PrepItem is one «قبل از نوبت» checklist item; the server assigns the id ("p1", "p2", …),
// stable across edits.
type PrepItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// AppointmentMeta is reminders.meta of an appointment (v1), in storage key order:
// {"v":1,"kind":"in_person","with":"…","specialty":"…","topic":"ultrasound","location":"…",
// "remind_before":"1d","add_to_calendar":true,"prep":[{"id":"p1","text":"…","done":false}],
// "status":"scheduled"}.
type AppointmentMeta struct {
	V             int        `json:"v"`
	Kind          string     `json:"kind"`
	With          *string    `json:"with"`
	Specialty     *string    `json:"specialty"`
	Topic         string     `json:"topic"`
	Location      *string    `json:"location"`
	RemindBefore  string     `json:"remind_before"`
	AddToCalendar bool       `json:"add_to_calendar"`
	Prep          []PrepItem `json:"prep"`
	Status        string     `json:"status"`
	// Pregnancy v2 visit link (T-M7-05), optional and omitted when unset so v1 rows stay byte-identical.
	CareItemKey *string `json:"care_item_key,omitempty"`
	Stage       *string `json:"stage,omitempty"`
	ResultNote  *string `json:"result_note,omitempty"`
	// Private (CB-LOSS-01) marks an owner-only appointment — the pregnancy-loss follow-ups (beta test, visit): never
	// shown to a companion (companion/shared, «ثبت برای …» show/update answer 404). Set by internal/loss, never by a
	// request; kept across the owner's edits. Omitted when false, so other rows stay byte-identical.
	Private bool `json:"private,omitempty"`
	// BookingID links a visit booked and paid through the telemedicine domain (bloom B-N7-03, care.ScheduleVisit).
	// Set by internal/telemed, never by a request; kept across the owner's edits. Omitted when unset.
	BookingID *uint64 `json:"booking_id,omitempty"`
}

// Subtitle is the legacy reminders.subtitle "with · specialty" (the parts that are set);
// NULL when both are empty.
func (m AppointmentMeta) Subtitle() sql.NullString {
	var parts []string
	for _, p := range []*string{m.With, m.Specialty} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	if len(parts) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.Join(parts, " · "), Valid: true}
}

// IsCancelledAppointment reports whether a reminders row (its type and raw meta) is a cancelled
// appointment. Its reminder stays off: switching it back on is a 422 on both PUT
// /care/appointments/{id} and the legacy PUT /reminders/{id} (D-29).
func IsCancelledAppointment(reminderType string, meta []byte) bool {
	if reminderType != TypeAppointment || len(meta) == 0 {
		return false
	}
	var m struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(meta, &m) == nil && m.Status == StatusCancelled
}

// CancelledReminderMessage is the 422 message for switching a cancelled appointment's reminder on.
func CancelledReminderMessage(locale string) string {
	return T("validation.cancelled_reminder", locale)
}

// RemindOffset is the remind_before duration.
func RemindOffset(v string) time.Duration {
	switch v {
	case "1h":
		return time.Hour
	case "3h":
		return 3 * time.Hour
	case "2d":
		return 48 * time.Hour
	default: // "1d"
		return 24 * time.Hour
	}
}

// AssignPrepIDs gives every item a stable id: an id already known (from the stored list) is
// kept once; any other item (new, unknown or duplicate id) gets the next "pN" after the
// highest N in use.
func AssignPrepIDs(items []PrepItem, known []PrepItem) []PrepItem {
	next := 0
	bump := func(id string) {
		if n, err := strconv.Atoi(strings.TrimPrefix(id, "p")); err == nil && strings.HasPrefix(id, "p") && n > next {
			next = n
		}
	}
	knownIDs := make([]string, 0, len(known))
	for _, k := range known {
		knownIDs = append(knownIDs, k.ID)
		bump(k.ID)
	}
	out := make([]PrepItem, 0, len(items))
	used := map[string]bool{}
	for _, it := range items {
		if it.ID != "" && slices.Contains(knownIDs, it.ID) && !used[it.ID] {
			used[it.ID] = true
			out = append(out, it)
			continue
		}
		it.ID = ""
		out = append(out, it)
	}
	for i := range out {
		if out[i].ID == "" {
			next++
			out[i].ID = "p" + strconv.Itoa(next)
		}
	}
	return out
}

// Appointment is an appointment reminder row with its parsed meta.
type Appointment struct {
	Row  store.Reminder
	Meta AppointmentMeta
}

// ParseAppointment reads an appointment row. Rows without a valid v1 meta (created through
// the legacy POST /reminders) get defaults: in-person, topic other, 1 day before, no prep,
// status scheduled.
func ParseAppointment(r store.Reminder) Appointment {
	var m AppointmentMeta
	if !r.Meta.Valid || json.Unmarshal(r.Meta.V, &m) != nil || m.V != MetaVersion {
		m = AppointmentMeta{V: MetaVersion}
	}
	if !slices.Contains(AppointmentKinds, m.Kind) {
		m.Kind = AppointmentKinds[0]
	}
	if !slices.Contains(AppointmentTopics, m.Topic) {
		m.Topic = DefaultTopic
	}
	if !slices.Contains(RemindBefore, m.RemindBefore) {
		m.RemindBefore = DefaultRemindBefore
	}
	if !slices.Contains(AppointmentStatuses, m.Status) {
		m.Status = StatusScheduled
	}
	if m.Prep == nil {
		m.Prep = []PrepItem{}
	}
	if m.Stage != nil && !slices.Contains(VisitStages, *m.Stage) {
		m.Stage = nil
	}
	return Appointment{Row: r, Meta: m}
}

// HiddenFrom reports whether actorID may not see ownerID's appointment: a private one, to anyone but her.
func (a Appointment) HiddenFrom(ownerID, actorID uint64) bool {
	return a.Meta.Private && ownerID != actorID
}

// Upcoming reports whether the appointment is still ahead: scheduled (not cancelled) and at
// or after now.
func (a Appointment) Upcoming(now time.Time) bool {
	return a.Meta.Status == StatusScheduled && a.Row.ScheduledAt.Valid && !a.Row.ScheduledAt.Time.Before(now)
}

// JSON is the /care/appointments resource; remind_at = scheduled_at − remind_before and
// days_until = civil days from today (Tehran) to the appointment day (negative when past).
func (a Appointment) JSON(now time.Time) *jsonx.OrderedMap {
	r := a.Row
	var scheduledAt, remindAt, daysUntil any
	if r.ScheduledAt.Valid {
		at := r.ScheduledAt.Time.In(civildate.Tehran)
		scheduledAt = at.Format(wallClock)
		remindAt = at.Add(-RemindOffset(a.Meta.RemindBefore)).Format(wallClock)
		daysUntil = civildate.InTehran(now).DiffDays(civildate.FromTime(at))
	}
	return jsonx.Obj(
		"id", r.ID,
		"type", r.Type,
		"title", r.Title,
		"subtitle", nullString(r.Subtitle),
		"notes", nullString(r.Notes),
		"kind", a.Meta.Kind,
		"with", a.Meta.With,
		"specialty", a.Meta.Specialty,
		"topic", a.Meta.Topic,
		"location", a.Meta.Location,
		"scheduled_at", scheduledAt,
		"remind_before", a.Meta.RemindBefore,
		"remind_at", remindAt,
		"days_until", daysUntil,
		"add_to_calendar", a.Meta.AddToCalendar,
		"prep", a.Meta.Prep,
		"status", a.Meta.Status,
		"care_item_key", a.Meta.CareItemKey,
		"stage", a.Meta.Stage,
		"result_note", a.Meta.ResultNote,
		"is_active", r.IsActive,
		"created_at", nullDateTime(r.CreatedAt),
		"updated_at", nullDateTime(r.UpdatedAt),
	)
}
