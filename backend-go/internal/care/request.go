package care

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// medicationFields are the request keys a medication accepts (anything else is ignored).
var medicationFields = []string{
	"title", "dose", "unit", "form", "times", "weekdays", "amount", "duration",
	"starts_on", "ends_on", "notify", "notes", "is_active", "frequency", "doses",
}

// medicationRules validate the full medication (a POST body, or a PUT body merged over
// the stored medication). Defaults for the `sometimes` fields apply after validation.
func medicationRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("title", "required", "string", "max:255"),
		F("dose", "nullable", "string", "max:50"),
		F("unit", "nullable", "string", "max:50"),
		F("form", "required_without:doses", validation.In(Forms...)),
		F("times", "required_without:doses", "array", "min:1", "max:4"),
		F("times.*", "required", "date_format:H:i"),
		F("weekdays", "sometimes", "array", "min:1", "max:7"),
		F("weekdays.*", "integer", "min:0", "max:6"),
		F("amount", "sometimes", "integer", "min:1", "max:10"),
		F("duration", "sometimes", validation.In(Durations...)),
		F("starts_on", "required", "date_format:Y-m-d"),
		F("ends_on", "nullable", "required_if:duration,"+DurationUntilDate, "date_format:Y-m-d", "after_or_equal:starts_on"),
		F("notify", "sometimes", "boolean"),
		F("notes", "nullable", "string", "max:2000"),
		F("is_active", "sometimes", "boolean"),
		// The interval schedule: with `doses` the server derives times / weekdays / amount (and form when not sent).
		F("frequency", "nullable", "array"),
		F("frequency.every", "sometimes", "integer", "min:1", "max:"+strconv.Itoa(MaxDayInterval)),
		F("frequency.unit", "sometimes", validation.In(FrequencyUnits...)),
		F("doses", "nullable", "array", "min:1", "max:"+strconv.Itoa(MaxDoses)),
		F("doses.*.time", "required", "date_format:H:i"),
		F("doses.*.amount", "required", "integer", "min:1", "max:99"),
		F("doses.*.unit", "required", validation.In(DoseUnits...)),
		F("doses.*.meal", "nullable", validation.In(Meals...)),
	}
}

// medicationInput is a validated medication before pregnancy_end is resolved.
type medicationInput struct {
	Title    string
	Notes    *string
	StartsOn civildate.Date
	EndsOn   *civildate.Date // until_date only
	IsActive bool
	Meta     MedicationMeta
}

// pickMedication copies the known medication keys of in over base (a POST: an empty base;
// a PUT: the stored medication). Numeric doses/units are taken as their string form.
func pickMedication(base, in phpval.Map) phpval.Map {
	// A client that only knows the weekday form sends `times` without `doses`: its edit replaces an interval
	// schedule instead of being silently overridden by the stored doses.
	if _, times := in.Get("times"); times {
		if _, doses := in.Get("doses"); !doses {
			base.Set("doses", nil)
			base.Set("frequency", nil)
		}
	}
	for _, k := range medicationFields {
		v, ok := in.Get(k)
		if !ok {
			continue
		}
		if k == "dose" || k == "unit" {
			if _, isStr := v.(string); !isStr && v != nil && phpval.IsNumeric(v) {
				v = phpval.ToString(v)
			}
		}
		base.Set(k, v)
	}
	return base
}

// storedMedication is the stored medication as request data, the base a PUT merges over.
// ends_on is carried only for until_date (for the other durations the server derives it).
func storedMedication(m Medication) phpval.Map {
	data := phpval.NewMap()
	times := make([]any, 0, len(m.Meta.Times))
	for _, t := range m.Meta.Times {
		times = append(times, t)
	}
	weekdays := make([]any, 0, len(m.Meta.Weekdays))
	for _, w := range m.Meta.Weekdays {
		weekdays = append(weekdays, int64(w))
	}
	data.Set("title", m.Row.Title)
	data.Set("dose", ptrValue(m.Meta.Dose))
	data.Set("unit", ptrValue(m.Meta.Unit))
	data.Set("form", m.Meta.Form)
	data.Set("times", times)
	data.Set("weekdays", weekdays)
	data.Set("amount", int64(m.Meta.Amount))
	data.Set("duration", m.Meta.Duration)
	switch {
	case m.Row.StartsOn.Valid:
		data.Set("starts_on", m.Row.StartsOn.Date.String())
	case m.Row.CreatedAt.Valid: // legacy POST /reminders row: starts when it was created
		data.Set("starts_on", civildate.FromTime(m.Row.CreatedAt.Time).String())
	}
	if m.Meta.Duration == DurationUntilDate && m.Row.EndsOn.Valid {
		data.Set("ends_on", m.Row.EndsOn.Date.String())
	}
	data.Set("notify", m.Meta.Notify)
	data.Set("notes", nullString(m.Row.Notes))
	data.Set("is_active", m.Row.IsActive)
	if f := m.Meta.Frequency; f != nil {
		freq := phpval.NewMap()
		freq.Set("every", int64(f.Every))
		freq.Set("unit", f.Unit)
		data.Set("frequency", freq)
		doses := make([]any, 0, len(m.Meta.Doses))
		for _, d := range m.Meta.Doses {
			dm := phpval.NewMap()
			dm.Set("time", d.Time)
			dm.Set("amount", int64(d.Amount))
			dm.Set("unit", d.Unit)
			if d.Meal != nil {
				dm.Set("meal", *d.Meal)
			} else {
				dm.Set("meal", nil)
			}
			doses = append(doses, dm)
		}
		data.Set("doses", doses)
	}
	return data
}

// switchFields are the keys of a toggle-only PUT (the list switches).
var switchFields = []string{"is_active", "notify"}

// switchesOnly reports whether the PUT body carries medication keys and all of them are
// switches (is_active / notify).
func switchesOnly(in phpval.Map) bool {
	sent := 0
	for _, k := range medicationFields {
		if _, ok := in.Get(k); ok {
			if !slices.Contains(switchFields, k) {
				return false
			}
			sent++
		}
	}
	return sent > 0
}

// switchRules validate a toggle-only PUT.
func switchRules() validation.Rules {
	return validation.Rules{
		validation.F("is_active", "sometimes", "boolean"),
		validation.F("notify", "sometimes", "boolean"),
	}
}

func ptrValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// validateMedication runs the rules over data and builds the input.
func validateMedication(locale string, data phpval.Map, now time.Time) (medicationInput, error) {
	interval := false
	if d, _ := data.Get("doses"); d != nil {
		interval = true
		// Derived from the doses below; whatever came with them is not validated.
		data.Delete("times")
		data.Delete("weekdays")
		data.Delete("amount")
	}
	v := validation.Make(lang.Default(), locale, data, medicationRules(),
		validation.Now(now),
		validation.Attributes(attributes(locale)...),
		validation.Messages("ends_on.required_if", T("validation.ends_on_required", locale)))
	if v.Fails() {
		return medicationInput{}, failValidation(locale, v.ErrorBag())
	}
	get := func(k string) (any, bool) { return data.Get(k) }

	in := medicationInput{IsActive: true}
	in.Meta = MedicationMeta{V: MetaVersion, Amount: 1, Duration: DurationOngoing, Notify: true}
	title, _ := get("title")
	in.Title = phpval.ToString(title)
	if n, _ := get("notes"); n != nil {
		s := phpval.ToString(n)
		in.Notes = &s
	}
	if d, _ := get("dose"); d != nil {
		s := phpval.ToString(d)
		in.Meta.Dose = &s
	}
	if u, _ := get("unit"); u != nil {
		s := phpval.ToString(u)
		in.Meta.Unit = &s
	}
	form, _ := get("form")
	in.Meta.Form = phpval.ToString(form)
	if interval {
		if err := intervalSchedule(locale, data, &in.Meta); err != nil {
			return medicationInput{}, err
		}
	}

	times, _ := get("times")
	if interval {
		times = nil
	}
	_, tv := phpval.Entries(times)
	for _, t := range tv {
		s := phpval.ToString(t)
		if !slices.Contains(in.Meta.Times, s) {
			in.Meta.Times = append(in.Meta.Times, s)
		}
	}
	slices.Sort(in.Meta.Times)
	if interval {
		in.Meta.Times = TimesOf(*in.Meta.Frequency, in.Meta.Doses)
	}

	in.Meta.Weekdays = slices.Clone(AllWeekdays)
	if w, ok := get("weekdays"); ok {
		_, wv := phpval.Entries(w)
		days := []int{}
		for _, x := range wv {
			if d := int(phpval.ToFloat(x)); !slices.Contains(days, d) {
				days = append(days, d)
			}
		}
		slices.Sort(days)
		in.Meta.Weekdays = days
	}
	if a, ok := get("amount"); ok {
		in.Meta.Amount = int(phpval.ToFloat(a))
	}
	if interval {
		in.Meta.Amount = min(in.Meta.Doses[0].Amount, 10) // the older per-intake count, for readers of `amount`
	}
	if d, ok := get("duration"); ok {
		in.Meta.Duration = phpval.ToString(d)
	}
	if n, ok := get("notify"); ok {
		in.Meta.Notify = phpval.Truthy(n)
	}
	if a, ok := get("is_active"); ok {
		in.IsActive = phpval.Truthy(a)
	}
	starts, _ := get("starts_on")
	var err error
	if in.StartsOn, err = civildate.Parse(phpval.ToString(starts)); err != nil {
		return medicationInput{}, fmt.Errorf("care: starts_on: %w", err)
	}
	if in.Meta.Duration == DurationUntilDate {
		ends, _ := get("ends_on")
		d, err := civildate.Parse(phpval.ToString(ends))
		if err != nil {
			return medicationInput{}, fmt.Errorf("care: ends_on: %w", err)
		}
		in.EndsOn = &d
	}
	return in, nil
}

// intervalSchedule reads the validated frequency + doses into meta (frequency defaults to every day), checks what
// the rules cannot — an hourly interval that divides the day, distinct dose times — and derives the form from the
// first dose's unit when none was sent. Doses are stored sorted by time.
func intervalSchedule(locale string, data phpval.Map, meta *MedicationMeta) error {
	f := Frequency{Every: 1, Unit: FrequencyDay}
	if raw, _ := data.Get("frequency"); raw != nil {
		if e, ok := phpval.Get(raw, "every"); ok && e != nil {
			f.Every = int(phpval.ToFloat(e))
		}
		if u, ok := phpval.Get(raw, "unit"); ok && u != nil {
			f.Unit = phpval.ToString(u)
		}
	}
	if f.Unit == FrequencyHour && !slices.Contains(HourIntervals, f.Every) {
		return fieldError(locale, "frequency.every", T("validation.frequency_hour_interval", locale))
	}
	raw, _ := data.Get("doses")
	_, items := phpval.Entries(raw)
	doses := make([]DoseSpec, 0, len(items))
	for i, item := range items {
		var d DoseSpec
		if x, ok := phpval.Get(item, "time"); ok {
			d.Time = phpval.ToString(x)
		}
		if x, ok := phpval.Get(item, "amount"); ok {
			d.Amount = int(phpval.ToFloat(x))
		}
		if x, ok := phpval.Get(item, "unit"); ok {
			d.Unit = phpval.ToString(x)
		}
		if x, ok := phpval.Get(item, "meal"); ok && x != nil && phpval.ToString(x) != "" {
			meal := phpval.ToString(x)
			d.Meal = &meal
		}
		if slices.ContainsFunc(doses, func(o DoseSpec) bool { return o.Time == d.Time }) {
			return fieldError(locale, "doses."+strconv.Itoa(i)+".time", T("validation.doses_time_distinct", locale))
		}
		doses = append(doses, d)
	}
	slices.SortStableFunc(doses, func(a, b DoseSpec) int { return strings.Compare(a.Time, b.Time) })
	meta.Frequency, meta.Doses = &f, doses
	if meta.Form == "" {
		meta.Form = FormOfDoseUnit(doses[0].Unit)
	}
	return nil
}

// intakeRules validate an intake body {date, slot}; ticking a future day is refused.
func intakeRules(tick bool) validation.Rules {
	F := validation.F
	date := []any{"required", "date_format:Y-m-d"}
	if tick {
		date = append(date, "before_or_equal:today")
	}
	return validation.Rules{
		F("date", date...),
		F("slot", "required", "date_format:H:i"),
	}
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldError is a single-field 422 in the same shape.
func fieldError(locale, field, msg string) error {
	return failValidation(locale, jsonx.Obj(field, []string{msg}))
}

// parseID is the {id} route parameter: a positive integer, anything else is a miss.
func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && id > 0
}
