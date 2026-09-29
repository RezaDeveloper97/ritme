package care

import (
	"database/sql"
	"encoding/json"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// TypeMedication is the reminders.type of a medication reminder.
const TypeMedication = "medication"

// MetaVersion is the current version of the meta JSON shapes.
const MetaVersion = 1

// MedicationMeta is reminders.meta of a medication (v1), in storage key order:
// {"v":1,"dose":"400","unit":"mcg","form":"tablet","times":["08:00","20:00"],
// "weekdays":[0,…,6],"amount":1,"duration":"pregnancy_end","notify":true}.
// Weekdays are Saturday-based (Saturday = 0 … Friday = 6); times are sorted "HH:MM".
type MedicationMeta struct {
	V        int      `json:"v"`
	Dose     *string  `json:"dose"`
	Unit     *string  `json:"unit"`
	Form     string   `json:"form"`
	Times    []string `json:"times"`
	Weekdays []int    `json:"weekdays"`
	Amount   int      `json:"amount"`
	Duration string   `json:"duration"`
	Notify   bool     `json:"notify"`
}

// AllWeekdays is every day of the week (Saturday = 0).
var AllWeekdays = []int{0, 1, 2, 3, 4, 5, 6}

// SaturdayWeekday is d's day in the project week: Saturday = 0 … Friday = 6.
func SaturdayWeekday(d civildate.Date) int { return (int(d.Weekday()) + 1) % 7 }

// Recurrence is the legacy reminders.recurrence for the meta: daily on every weekday,
// weekly on a subset.
func (m MedicationMeta) Recurrence() string {
	if len(m.Weekdays) == len(AllWeekdays) {
		return "daily"
	}
	return "weekly"
}

// RecurrenceTime is the legacy reminders.recurrence_time: the first slot ("08:00:00").
func (m MedicationMeta) RecurrenceTime() sql.NullString {
	if len(m.Times) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: m.Times[0] + ":00", Valid: true}
}

// Subtitle is the legacy reminders.subtitle, "dose unit" in the locale ("۴۰۰ میکروگرم");
// a known unit shows its label, a free-text unit as typed. NULL without dose and unit.
func (m MedicationMeta) Subtitle(locale string) sql.NullString {
	var parts []string
	if m.Dose != nil && *m.Dose != "" {
		parts = append(parts, LocalizeDigits(*m.Dose, locale))
	}
	if m.Unit != nil && *m.Unit != "" {
		unit := *m.Unit
		if label, ok := Label("units", unit, locale); ok {
			unit = label
		}
		parts = append(parts, unit)
	}
	if len(parts) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.Join(parts, " "), Valid: true}
}

// Medication is a medication reminder row with its parsed meta.
type Medication struct {
	Row  store.Reminder
	Meta MedicationMeta
}

// ParseMedication reads a medication row. Rows without a valid v1 meta (created through the
// legacy POST /reminders) get a meta derived from the columns: one slot at recurrence_time,
// every weekday, amount 1, duration until_date when ends_on is set (else ongoing).
func ParseMedication(r store.Reminder) Medication {
	var m MedicationMeta
	if r.Meta.Valid && json.Unmarshal(r.Meta.V, &m) == nil && m.V == MetaVersion {
		if m.Times == nil {
			m.Times = []string{}
		}
		if len(m.Weekdays) == 0 {
			m.Weekdays = slices.Clone(AllWeekdays)
		}
		if m.Amount < 1 {
			m.Amount = 1
		}
		if !slices.Contains(Durations, m.Duration) {
			m.Duration = DurationOngoing
		}
		return Medication{Row: r, Meta: m}
	}
	m = MedicationMeta{
		V: MetaVersion, Form: "tablet", Times: []string{}, Weekdays: slices.Clone(AllWeekdays),
		Amount: 1, Duration: DurationOngoing, Notify: true,
	}
	if r.RecurrenceTime.Valid && len(r.RecurrenceTime.String) >= 5 {
		m.Times = []string{r.RecurrenceTime.String[:5]}
	}
	if r.EndsOn.Valid {
		m.Duration = DurationUntilDate
	}
	return Medication{Row: r, Meta: m}
}

// Covers reports whether the medication is scheduled on d: d inside [starts_on, ends_on]
// (open ends allowed) and on one of its weekdays. is_active is not considered.
func (m Medication) Covers(d civildate.Date) bool {
	if m.Row.StartsOn.Valid && d.Before(m.Row.StartsOn.Date) {
		return false
	}
	if m.Row.EndsOn.Valid && d.After(m.Row.EndsOn.Date) {
		return false
	}
	return slices.Contains(m.Meta.Weekdays, SaturdayWeekday(d))
}

// DisplaySubtitle is the subtitle in the reading locale: built from the meta (dose + unit) at
// render time, so a medication saved in one language reads right in another; the stored
// column (written in the default language for GET /reminders readers) only for legacy rows
// without a dose/unit in their meta.
func (m Medication) DisplaySubtitle(locale string) sql.NullString {
	if sub := m.Meta.Subtitle(locale); sub.Valid {
		return sub
	}
	return m.Row.Subtitle
}

// JSON is the /care/medications resource in the request locale.
func (m Medication) JSON(locale string) *jsonx.OrderedMap {
	r := m.Row
	var recurrenceTime any
	if r.RecurrenceTime.Valid && len(r.RecurrenceTime.String) >= 5 {
		recurrenceTime = r.RecurrenceTime.String[:5]
	}
	return jsonx.Obj(
		"id", r.ID,
		"type", r.Type,
		"title", r.Title,
		"subtitle", nullString(m.DisplaySubtitle(locale)),
		"notes", nullString(r.Notes),
		"form", m.Meta.Form,
		"dose", m.Meta.Dose,
		"unit", m.Meta.Unit,
		"times", m.Meta.Times,
		"weekdays", m.Meta.Weekdays,
		"amount", m.Meta.Amount,
		"duration", m.Meta.Duration,
		"notify", m.Meta.Notify,
		"starts_on", nullDate(r.StartsOn),
		"ends_on", nullDate(r.EndsOn),
		"is_active", r.IsActive,
		"recurrence", r.Recurrence,
		"recurrence_time", recurrenceTime,
		"created_at", nullDateTime(r.CreatedAt),
		"updated_at", nullDateTime(r.UpdatedAt),
	)
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}
