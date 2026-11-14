package care

import (
	"database/sql"
	"encoding/json"
	"slices"
	"strconv"
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
	// Frequency and Doses are the interval schedule («هر ۱ روز», «هر ۲ ساعت» + doses with amount, unit, time and
	// meal relation). nil / empty = the weekday schedule above. When set, Times is derived from them (TimesOf),
	// Weekdays is every day and Amount is the first dose's, so readers of the older fields keep working.
	Frequency *Frequency `json:"frequency,omitempty"`
	Doses     []DoseSpec `json:"doses,omitempty"`
}

// Frequency is the repeat interval of a medication: every N days or every N hours.
type Frequency struct {
	Every int    `json:"every"`
	Unit  string `json:"unit"`
}

// DoseSpec is one dose of the schedule: how much, of what, when, and before or after a meal (nil = not said).
// With an hourly frequency Time is the first intake of the day's chain (Time + k·Every hours).
type DoseSpec struct {
	Time   string  `json:"time"`
	Amount int     `json:"amount"`
	Unit   string  `json:"unit"`
	Meal   *string `json:"meal"`
}

// Frequency units, dose units and meal relations.
const (
	FrequencyDay  = "day"
	FrequencyHour = "hour"
	MealBefore    = "before"
	MealAfter     = "after"
)

var (
	// FrequencyUnits are the units of Frequency.
	FrequencyUnits = []string{FrequencyDay, FrequencyHour}
	// HourIntervals are the hourly intervals offered: the divisors of 24, so the slots are the same every day.
	HourIntervals = []int{1, 2, 3, 4, 6, 8, 12}
	// MaxDayInterval is the longest «هر N روز».
	MaxDayInterval = 30
	// DoseUnits are the units of a dose amount («۱ قرص», «۵ قطره»).
	DoseUnits = []string{"pill", "capsule", "drop", "ml", "spoon", "puff", "injection"}
	// Meals are the meal relations of a dose.
	Meals = []string{MealBefore, MealAfter}
	// MaxDoses is the most doses one medication carries.
	MaxDoses = 6
)

// formOfDoseUnit is the medication form shown for a dose unit (the icon of the lists).
var formOfDoseUnit = map[string]string{
	"pill": "tablet", "capsule": "capsule", "drop": "drops", "ml": "syrup", "spoon": "syrup",
	"puff": "tablet", "injection": "injection",
}

// FormOfDoseUnit is the form a dose unit implies (tablet for an unknown one).
func FormOfDoseUnit(unit string) string {
	if f, ok := formOfDoseUnit[unit]; ok {
		return f
	}
	return "tablet"
}

func slotMinutes(slot string) (int, bool) {
	if len(slot) != 5 || slot[2] != ':' {
		return 0, false
	}
	h, m := int(slot[0]-'0')*10+int(slot[1]-'0'), int(slot[3]-'0')*10+int(slot[4]-'0')
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func minutesSlot(min int) string {
	min = ((min % 1440) + 1440) % 1440
	return string([]byte{byte('0' + min/600), byte('0' + min/60%10), ':', byte('0' + min%60/10), byte('0' + min%10)})
}

// TimesOf are the day's slots of an interval schedule, sorted and unique: the dose times for a daily
// frequency; each dose's chain (time + k·every hours, around the clock) for an hourly one.
func TimesOf(f Frequency, doses []DoseSpec) []string {
	out := []string{}
	add := func(s string) {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, d := range doses {
		start, ok := slotMinutes(d.Time)
		if !ok {
			continue
		}
		if f.Unit != FrequencyHour || f.Every < 1 || f.Every > 24 {
			add(d.Time)
			continue
		}
		for k := 0; k*f.Every < 24; k++ {
			add(minutesSlot(start + k*f.Every*60))
		}
	}
	slices.Sort(out)
	return out
}

// DoseAt is the dose taken at slot: the dose at that time (daily), or the one whose hourly chain holds the
// slot. nil for a weekday-schedule medication or an unknown slot.
func (m MedicationMeta) DoseAt(slot string) *DoseSpec {
	at, ok := slotMinutes(slot)
	if !ok || m.Frequency == nil {
		return nil
	}
	for i := range m.Doses {
		start, ok := slotMinutes(m.Doses[i].Time)
		if !ok {
			continue
		}
		if start == at {
			return &m.Doses[i]
		}
		if m.Frequency.Unit == FrequencyHour && m.Frequency.Every > 0 && (at-start+1440)%(m.Frequency.Every*60) == 0 {
			return &m.Doses[i]
		}
	}
	return nil
}

// DoseLabel is «۱ قرص» / "1 pill" for a dose in the locale.
func DoseLabel(d DoseSpec, locale string) string {
	unit := d.Unit
	if label, ok := Label("dose_units", d.Unit, locale); ok {
		unit = label
	}
	return LocalizeDigits(strconv.Itoa(d.Amount), locale) + " " + unit
}

// DoseLine is the dose with its meal relation, «۱ قرص · بعد از غذا».
func DoseLine(d DoseSpec, locale string) string {
	line := DoseLabel(d, locale)
	if d.Meal != nil {
		if label, ok := Label("meals", *d.Meal, locale); ok {
			line += " · " + label
		}
	}
	return line
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
		if len(m.Doses) > 0 { // interval schedule without a strength: the first dose, «۱ قرص»
			return sql.NullString{String: DoseLabel(m.Doses[0], locale), Valid: true}
		}
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
		if m.Frequency != nil && (len(m.Doses) == 0 || m.Frequency.Every < 1 || !slices.Contains(FrequencyUnits, m.Frequency.Unit)) {
			m.Frequency, m.Doses = nil, nil // not a usable interval schedule: read as the weekday one
		}
		if m.Frequency == nil {
			m.Doses = nil
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
	if f := m.Meta.Frequency; f != nil {
		if f.Unit == FrequencyDay && f.Every > 1 && m.Row.StartsOn.Valid {
			return m.Row.StartsOn.Date.DiffDays(d)%f.Every == 0
		}
		return true
	}
	return slices.Contains(m.Meta.Weekdays, SaturdayWeekday(d))
}

// SlotsOn are the slots of day d ("HH:MM", sorted), empty when d is not covered. An hourly chain starts at
// its first dose time on the start date, so that day's earlier slots are left out.
func (m Medication) SlotsOn(d civildate.Date) []string {
	if !m.Covers(d) {
		return nil
	}
	f := m.Meta.Frequency
	if f == nil || f.Unit != FrequencyHour || !m.Row.StartsOn.Valid || m.Row.StartsOn.Date != d {
		return m.Meta.Times
	}
	first := ""
	for _, dose := range m.Meta.Doses {
		if first == "" || dose.Time < first {
			first = dose.Time
		}
	}
	out := make([]string, 0, len(m.Meta.Times))
	for _, s := range m.Meta.Times {
		if s >= first {
			out = append(out, s)
		}
	}
	return out
}

// dosesJSON is the doses list of the resource (amount, unit + its label, time, meal).
func (m MedicationMeta) dosesJSON(locale string) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(m.Doses))
	for _, d := range m.Doses {
		var label any
		if l, ok := Label("dose_units", d.Unit, locale); ok {
			label = l
		}
		var meal any
		if d.Meal != nil && slices.Contains(Meals, *d.Meal) {
			meal = *d.Meal
		}
		out = append(out, jsonx.Obj("time", d.Time, "amount", d.Amount, "unit", d.Unit, "unit_label", label, "meal", meal))
	}
	return out
}

func (m MedicationMeta) frequencyJSON() any {
	if m.Frequency == nil {
		return nil
	}
	return jsonx.Obj("every", m.Frequency.Every, "unit", m.Frequency.Unit)
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
		"frequency", m.Meta.frequencyJSON(),
		"doses", m.Meta.dosesJSON(locale),
		"starts_on", nullDate(r.StartsOn),
		"ends_on", nullDate(r.EndsOn),
		"is_active", r.IsActive,
		"recurrence", r.Recurrence,
		"recurrence_time", recurrenceTime,
		"created_at", nullDateTime(r.CreatedAt),
		"updated_at", nullDateTime(r.UpdatedAt),
	)
}

// MemberJSON is what a family member (companion) may see of the owner's medication (T-M9-02, FAM-M2): an explicit
// allow-list — name, form and dose, schedule, status. Never the owner's notes, a free-text subtitle or unit, or any
// other field the owner typed for herself; unknown forms and units are dropped, not echoed. JSON is the owner's.
func (m Medication) MemberJSON(locale string) *jsonx.OrderedMap {
	r := m.Row
	var form, dose, unit, unitLabel any
	if slices.Contains(Forms, m.Meta.Form) {
		form = m.Meta.Form
	}
	if m.Meta.Dose != nil && isDoseNumber(*m.Meta.Dose) {
		dose = *m.Meta.Dose
	}
	if m.Meta.Unit != nil && slices.Contains(Units, *m.Meta.Unit) {
		unit = *m.Meta.Unit
		if label, ok := Label("units", *m.Meta.Unit, locale); ok {
			unitLabel = label
		}
	}
	return jsonx.Obj(
		"id", r.ID,
		"type", r.Type,
		"title", r.Title,
		"form", form,
		"dose", dose,
		"unit", unit,
		"unit_label", unitLabel,
		"times", m.Meta.Times,
		"weekdays", m.Meta.Weekdays,
		"frequency", m.Meta.frequencyJSON(),
		"doses", m.Meta.dosesJSON(locale),
		"starts_on", nullDate(r.StartsOn),
		"ends_on", nullDate(r.EndsOn),
		"is_active", r.IsActive,
	)
}

// isDoseNumber reports whether a stored dose is a plain number ("400", "0.5") — anything else is free text.
func isDoseNumber(s string) bool {
	if s == "" || len(s) > 12 {
		return false
	}
	dot := false
	for _, ch := range s {
		switch {
		case ch >= '0' && ch <= '9':
		case ch == '.' && !dot:
			dot = true
		default:
			return false
		}
	}
	return true
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
