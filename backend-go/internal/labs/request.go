package labs

import (
	"database/sql"
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

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale, nil), "errors", errs)
}

// fieldFail is a 422 on one field with a labs validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale, nil)}))
}

func pick(body phpval.Map, keys []string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

// numericKeys are the marker fields typed as numbers; Persian / Arabic digits and the Persian decimal separator are
// accepted («۱۱٫۲») and turned into ASCII before validation.
var numericKeys = []string{"value", "ref_low", "ref_high"}

func asciiNumbers(data phpval.Map) phpval.Map {
	for _, k := range numericKeys {
		v, ok := data.Get(k)
		s, isStr := v.(string)
		if !ok || !isStr {
			continue
		}
		data.Set(k, strings.Map(func(r rune) rune {
			switch {
			case r >= '۰' && r <= '۹':
				return '0' + (r - '۰')
			case r >= '٠' && r <= '٩':
				return '0' + (r - '٠')
			case r == '٫':
				return '.'
			}
			return r
		}, s))
	}
	return data
}

func floatPtr(v any, ok bool) *float64 {
	if !ok || v == nil {
		return nil
	}
	f, valid := parseNumber(phpval.ToString(v))
	if !valid {
		return nil
	}
	return &f
}

func numericRange() string {
	m := strconv.FormatFloat(MaxValue, 'f', 0, 64)
	return "between:-" + m + "," + m
}

// metaRules validate the lab fields (upload form, manual lab, PUT /labs/{id}).
func metaRules(prefix string) validation.Rules {
	return validation.Rules{
		validation.F(prefix+"category", "required", "string", validation.In(Categories...)),
		validation.F(prefix+"title", "nullable", "string", "max:"+strconv.Itoa(MaxTitleLen)),
		validation.F(prefix+"taken_on", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		validation.F(prefix+"fasting", "nullable", "boolean"),
	}
}

var metaKeys = []string{"category", "title", "taken_on", "fasting"}

func metaFrom(data phpval.Map) Meta {
	m := Meta{Category: str(data, "category"), Title: str(data, "title")}
	if d, err := civildate.Parse(str(data, "taken_on")); err == nil {
		m.TakenOn = civildate.NullDate{Date: d, Valid: true}
	}
	if v, ok := data.Get("fasting"); ok && v != nil {
		m.Fasting = sql.NullBool{Bool: phpval.Truthy(v), Valid: true}
	}
	return m
}

// validateMeta is PUT /labs/{id} and the text fields of the upload.
func validateMeta(body phpval.Map, locale string, now time.Time) (Meta, error) {
	data := pick(body, metaKeys)
	if err := validate(data, metaRules(""), locale, now); err != nil {
		return Meta{}, err
	}
	return metaFrom(data), nil
}

var markerKeys = []string{"name", "value", "value_text", "unit", "ref_low", "ref_high", "ref_text"}

func markerRules(prefix string) validation.Rules {
	return validation.Rules{
		validation.F(prefix+"name", "required", "string", "max:"+strconv.Itoa(MaxNameLen)),
		validation.F(prefix+"value", "nullable", "numeric", numericRange()),
		validation.F(prefix+"value_text", "nullable", "string", "max:"+strconv.Itoa(MaxValueText)),
		validation.F(prefix+"unit", "nullable", "string", "max:"+strconv.Itoa(MaxUnitLen)),
		validation.F(prefix+"ref_low", "nullable", "numeric", numericRange()),
		validation.F(prefix+"ref_high", "nullable", "numeric", numericRange()),
		validation.F(prefix+"ref_text", "nullable", "string", "max:"+strconv.Itoa(MaxRefTextLen)),
	}
}

func markerFrom(data phpval.Map) MarkerInput {
	v, hasV := data.Get("value")
	lo, hasLo := data.Get("ref_low")
	hi, hasHi := data.Get("ref_high")
	in := MarkerInput{Name: str(data, "name"), Value: floatPtr(v, hasV), ValueText: str(data, "value_text"), Unit: str(data, "unit"),
		RefLow: floatPtr(lo, hasLo), RefHigh: floatPtr(hi, hasHi), RefText: str(data, "ref_text")}
	if in.Value != nil {
		in.ValueText = ""
	}
	return in
}

// checkMarker: a value (number or text) is required and the bounds must be ordered.
func checkMarker(in MarkerInput, field, locale string) error {
	if in.Value == nil && in.ValueText == "" {
		return fieldFail(locale, field+"value", "value_required")
	}
	if in.RefLow != nil && in.RefHigh != nil && *in.RefLow > *in.RefHigh {
		return fieldFail(locale, field+"ref_low", "ref_order")
	}
	return nil
}

// validateMarker is POST / PUT /labs/{id}/markers[/{mid}].
func validateMarker(body phpval.Map, locale string, now time.Time) (MarkerInput, error) {
	data := asciiNumbers(pick(body, markerKeys))
	if err := validate(data, markerRules(""), locale, now); err != nil {
		return MarkerInput{}, err
	}
	in := markerFrom(data)
	return in, checkMarker(in, "", locale)
}

// validateManual is POST /labs/manual {category, title?, taken_on?, fasting?, markers:[…]}.
func validateManual(body phpval.Map, locale string, now time.Time) (Meta, []MarkerInput, error) {
	data := pick(body, append(append([]string(nil), metaKeys...), "markers"))
	rules := append(metaRules(""), validation.F("markers", "required", "array", "min:1", "max:"+strconv.Itoa(MaxMarkers)))
	rules = append(rules, markerRules("markers.*.")...)
	if raw, ok := data.Get("markers"); ok {
		switch v := raw.(type) {
		case phpval.Map:
			for _, k := range v.Keys() {
				if item, ok := v.Get(k); ok {
					if m, ok := item.(phpval.Map); ok {
						asciiNumbers(m)
					}
				}
			}
		case []any:
			for _, item := range v {
				if m, ok := item.(phpval.Map); ok {
					asciiNumbers(m)
				}
			}
		}
	}
	if err := validate(data, rules, locale, now); err != nil {
		return Meta{}, nil, err
	}
	raw, _ := data.Get("markers")
	var list []MarkerInput
	switch v := raw.(type) {
	case phpval.Map:
		for i, k := range v.Keys() {
			item, _ := v.Get(k)
			m, ok := item.(phpval.Map)
			if !ok {
				return Meta{}, nil, fieldFail(locale, "markers", "markers_required")
			}
			in := markerFrom(pick(m, markerKeys))
			if err := checkMarker(in, "markers."+strconv.Itoa(i)+".", locale); err != nil {
				return Meta{}, nil, err
			}
			list = append(list, in)
		}
	case []any:
		for i, item := range v {
			m, ok := item.(phpval.Map)
			if !ok {
				return Meta{}, nil, fieldFail(locale, "markers", "markers_required")
			}
			in := markerFrom(pick(m, markerKeys))
			if err := checkMarker(in, "markers."+strconv.Itoa(i)+".", locale); err != nil {
				return Meta{}, nil, err
			}
			list = append(list, in)
		}
	}
	if len(list) == 0 {
		return Meta{}, nil, fieldFail(locale, "markers", "markers_required")
	}
	return metaFrom(data), list, nil
}

// validateFeedback is POST /labs/{id}/feedback {helpful: bool, note?}.
func validateFeedback(body phpval.Map, locale string, now time.Time) (bool, string, error) {
	data := pick(body, []string{"helpful", "note"})
	if err := validate(data, validation.Rules{
		validation.F("helpful", "required", "boolean"),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxFeedbackLen)),
	}, locale, now); err != nil {
		return false, "", err
	}
	v, _ := data.Get("helpful")
	return phpval.Truthy(v), str(data, "note"), nil
}
