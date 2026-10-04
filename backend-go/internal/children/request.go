package children

import (
	"database/sql"
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
	"github.com/ritme/backend-go/seeds/who"
)

// validate runs rules over the keys of body and returns the validated data (keys not sent stay absent).
func validate(body phpval.Map, keys []string, rules validation.Rules, locale string, now time.Time) (phpval.Map, error) {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, rules,
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	return data, nil
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldFail is a 422 on one field with a children validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func parseDate(v any) (civildate.Date, error) {
	d, err := civildate.Parse(phpval.ToString(v))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("children: date: %w", err)
	}
	return d, nil
}

// decimal is a validated number as the column's text with n decimals (NULL when absent or null).
func decimal(data phpval.Map, key string, n int) sql.NullString {
	v, ok := data.Get(key)
	if !ok || v == nil || strings.TrimSpace(phpval.ToString(v)) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strconv.FormatFloat(phpval.ToFloat(v), 'f', n, 64), Valid: true}
}

func optString(data phpval.Map, key string) sql.NullString {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return sql.NullString{}
	}
	s := strings.TrimSpace(phpval.ToString(v))
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func between(lo, hi float64) string {
	return "between:" + strconv.FormatFloat(lo, 'f', -1, 64) + "," + strconv.FormatFloat(hi, 'f', -1, 64)
}

var childKeys = []string{"name", "birth_date", "sex", "birth_weight_kg", "birth_length_cm", "birth_head_cm", "delivery_type"}

// validateChild is POST /children (and PUT, where a key not sent keeps its stored value: base).
func validateChild(body phpval.Map, base *ChildInput, locale string, now time.Time) (ChildInput, error) {
	F := validation.F
	presence := "required"
	if base != nil {
		presence = "sometimes"
	}
	data, err := validate(body, childKeys, validation.Rules{
		F("name", presence, "string", "max:"+strconv.Itoa(MaxNameLen)),
		F("birth_date", presence, "date_format:Y-m-d", "before_or_equal:today"),
		F("sex", "nullable", validation.In(Sexes...)),
		F("birth_weight_kg", "nullable", "numeric", between(MinWeightKg, MaxBirthKg)),
		F("birth_length_cm", "nullable", "numeric", between(MinLengthCm, MaxBirthLenCm)),
		F("birth_head_cm", "nullable", "numeric", between(MinHeadCm, MaxBirthHead)),
		F("delivery_type", "nullable", validation.In(DeliveryTypes...)),
	}, locale, now)
	if err != nil {
		return ChildInput{}, err
	}
	in := ChildInput{}
	if base != nil {
		in = *base
	}
	if v, ok := data.Get("name"); ok {
		name := strings.TrimSpace(phpval.ToString(v))
		if name == "" {
			return ChildInput{}, failValidation(locale, jsonx.Obj("name", []string{
				lang.Default().Trans("validation.required", map[string]string{"attribute": attrName("name", locale)}, locale),
			}))
		}
		in.Name = name
	}
	if v, ok := data.Get("birth_date"); ok {
		d, err := parseDate(v)
		if err != nil {
			return ChildInput{}, err
		}
		if d.Before(civildate.InTehran(now).AddDays(-MaxAgeYears * 366)) {
			return ChildInput{}, fieldFail(locale, "birth_date", "birth_date_too_old")
		}
		in.BirthDate = d
	}
	set := func(key string, dst *sql.NullString, v sql.NullString) {
		if _, ok := data.Get(key); ok {
			*dst = v
		}
	}
	set("sex", &in.Sex, optString(data, "sex"))
	set("delivery_type", &in.DeliveryType, optString(data, "delivery_type"))
	set("birth_weight_kg", &in.BirthWeightKg, decimal(data, "birth_weight_kg", 3))
	set("birth_length_cm", &in.BirthLengthCm, decimal(data, "birth_length_cm", 1))
	set("birth_head_cm", &in.BirthHeadCm, decimal(data, "birth_head_cm", 1))
	return in, nil
}

func attrName(key, locale string) string {
	attrs := attributes(locale)
	for i := 0; i+1 < len(attrs); i += 2 {
		if attrs[i] == key {
			return attrs[i+1]
		}
	}
	return key
}

var measurementKeys = []string{"measured_on", "weight_kg", "length_cm", "head_cm"}

// validateMeasurement is POST /children/{id}/measurements (base nil) or PUT …/{mid} (base = the stored row: keys not
// sent keep their value). measured_on defaults to today on create; it must be between the birth and today, and at
// least one value must remain.
func validateMeasurement(body phpval.Map, birth civildate.Date, base *MeasurementInput, locale string, now time.Time) (MeasurementInput, error) {
	F := validation.F
	data, err := validate(body, measurementKeys, validation.Rules{
		F("measured_on", "nullable", "date_format:Y-m-d"),
		F("weight_kg", "nullable", "numeric", between(MinWeightKg, MaxWeightKg)),
		F("length_cm", "nullable", "numeric", between(MinLengthCm, MaxLengthCm)),
		F("head_cm", "nullable", "numeric", between(MinHeadCm, MaxHeadCm)),
	}, locale, now)
	if err != nil {
		return MeasurementInput{}, err
	}
	today := civildate.InTehran(now)
	in := MeasurementInput{MeasuredOn: today}
	if base != nil {
		in = *base
	}
	if v, ok := data.Get("measured_on"); ok && v != nil {
		d, err := parseDate(v)
		if err != nil {
			return MeasurementInput{}, err
		}
		in.MeasuredOn = d
	}
	if in.MeasuredOn.After(today) {
		return MeasurementInput{}, fieldFail(locale, "measured_on", "measured_future")
	}
	if in.MeasuredOn.Before(birth) {
		return MeasurementInput{}, fieldFail(locale, "measured_on", "measured_before_birth")
	}
	for _, f := range []struct {
		key string
		dst *sql.NullString
		n   int
	}{{"weight_kg", &in.WeightKg, 3}, {"length_cm", &in.LengthCm, 1}, {"head_cm", &in.HeadCm, 1}} {
		if _, ok := data.Get(f.key); ok {
			*f.dst = decimal(data, f.key, f.n)
		}
	}
	if !in.WeightKg.Valid && !in.LengthCm.Valid && !in.HeadCm.Valid {
		return MeasurementInput{}, fieldFail(locale, "weight_kg", "measurement_empty")
	}
	return in, nil
}

// validateDose is PUT /children/{id}/vaccines/{code} and POST …/visits/{visit}: {given_on?, note?}.
func validateDose(body phpval.Map, birth civildate.Date, locale string, now time.Time) (civildate.Date, sql.NullString, error) {
	F := validation.F
	data, err := validate(body, []string{"given_on", "note"}, validation.Rules{
		F("given_on", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNoteLen)),
	}, locale, now)
	if err != nil {
		return civildate.Date{}, sql.NullString{}, err
	}
	day := civildate.InTehran(now)
	if v, ok := data.Get("given_on"); ok && v != nil {
		if day, err = parseDate(v); err != nil {
			return civildate.Date{}, sql.NullString{}, err
		}
	}
	if day.Before(birth) {
		return civildate.Date{}, sql.NullString{}, fieldFail(locale, "given_on", "given_before_birth")
	}
	return day, optString(data, "note"), nil
}

// validateCheck is PUT /children/{id}/milestones/{code}: {checked: bool, checked_on?}.
func validateCheck(body phpval.Map, birth civildate.Date, locale string, now time.Time) (bool, civildate.Date, error) {
	F := validation.F
	data, err := validate(body, []string{"checked", "checked_on"}, validation.Rules{
		F("checked", "required", "boolean"),
		F("checked_on", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
	}, locale, now)
	if err != nil {
		return false, civildate.Date{}, err
	}
	v, _ := data.Get("checked")
	checked := phpval.Truthy(v)
	day := civildate.InTehran(now)
	if d, ok := data.Get("checked_on"); ok && d != nil {
		if day, err = parseDate(d); err != nil {
			return false, civildate.Date{}, err
		}
	}
	if day.Before(birth) {
		day = birth
	}
	return checked, day, nil
}

// validateIndicator is ?indicator=weight|length|head (default weight).
func validateIndicator(q phpval.Map, locale string, now time.Time) (who.Indicator, error) {
	inds := make([]string, 0, len(who.Indicators))
	for _, i := range who.Indicators {
		inds = append(inds, string(i))
	}
	data, err := validate(q, []string{"indicator"}, validation.Rules{
		validation.F("indicator", "nullable", validation.In(inds...)),
	}, locale, now)
	if err != nil {
		return "", err
	}
	if v, ok := data.Get("indicator"); ok && v != nil {
		return who.Indicator(phpval.ToString(v)), nil
	}
	return who.Weight, nil
}

// validateMonth is ?month= (a milestone band; nil = the child's current band).
func validateMonth(q phpval.Map, locale string, now time.Time) (*int, error) {
	data, err := validate(q, []string{"month"}, validation.Rules{
		validation.F("month", "nullable", "integer", "min:0", "max:240"),
	}, locale, now)
	if err != nil {
		return nil, err
	}
	if v, ok := data.Get("month"); ok && v != nil {
		n := int(phpval.ToFloat(v))
		return &n, nil
	}
	return nil, nil //nolint:nilnil // nil = not asked
}

// validateTopic is ?topic= (a learn topic; "" = all).
func validateTopic(q phpval.Map, locale string, now time.Time) (string, error) {
	data, err := validate(q, []string{"topic"}, validation.Rules{
		validation.F("topic", "nullable", validation.In(append([]string{"all"}, Topics...)...)),
	}, locale, now)
	if err != nil {
		return "", err
	}
	if v, ok := data.Get("topic"); ok && v != nil && slices.Contains(Topics, phpval.ToString(v)) {
		return phpval.ToString(v), nil
	}
	return "", nil
}
