package care

import (
	"fmt"
	"slices"
	"strconv"
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
	"starts_on", "ends_on", "notify", "notes", "is_active",
}

// medicationRules validate the full medication (a POST body, or a PUT body merged over
// the stored medication). Defaults for the `sometimes` fields apply after validation.
func medicationRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("title", "required", "string", "max:255"),
		F("dose", "nullable", "string", "max:50"),
		F("unit", "nullable", "string", "max:50"),
		F("form", "required", validation.In(Forms...)),
		F("times", "required", "array", "min:1", "max:4"),
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

	times, _ := get("times")
	_, tv := phpval.Entries(times)
	for _, t := range tv {
		s := phpval.ToString(t)
		if !slices.Contains(in.Meta.Times, s) {
			in.Meta.Times = append(in.Meta.Times, s)
		}
	}
	slices.Sort(in.Meta.Times)

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
