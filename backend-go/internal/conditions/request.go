package conditions

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

// pastDate validates a day the user logs: a date, not in the future (Tehran).
func pastDate(required bool) []any {
	first := "nullable"
	if required {
		first = "required"
	}
	return []any{first, "date_format:Y-m-d", "before_or_equal:today"}
}

func messages(locale string) validation.Option {
	future := T("validation.date_future", locale)
	return validation.Messages("date.before_or_equal", future, "enrolled_on.before_or_equal", future)
}

// validate runs rules over the keys of body (plus fields already set in data) and returns the data.
func validate(data, body phpval.Map, keys []string, rules validation.Rules, locale string, now time.Time) (phpval.Map, error) {
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, rules,
		validation.Now(now), validation.Attributes(attributes(locale)...), messages(locale))
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	return data, nil
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldFail is a 422 on one field with a conditions validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

// intOf is a validated integer field.
func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

// boolOf is a validated boolean field (true, 1, "1", "true", "on", "yes").
func boolOf(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	default:
		s := strings.ToLower(phpval.ToString(v))
		return s == "1" || s == "true" || s == "on" || s == "yes"
	}
}

// strings of a validated array, de-duplicated, in the order of allowed.
func listOf(v any, allowed []string) []string {
	_, vals := phpval.Entries(v)
	got := map[string]bool{}
	for _, x := range vals {
		got[phpval.ToString(x)] = true
	}
	out := []string{}
	for _, a := range allowed {
		if got[a] {
			out = append(out, a)
		}
	}
	return out
}

// parseDay validates a {date} segment (GET): any well-formed date.
func parseDay(raw, locale string, now time.Time) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	if _, err := validate(data, phpval.NewMap(), nil,
		validation.Rules{validation.F("date", "required", "date_format:Y-m-d")}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("conditions: date: %w", err)
	}
	return d, nil
}

// parseLogDay validates a {date} segment of a write: a date, not in the future.
func parseLogDay(raw, locale string, now time.Time) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	if _, err := validate(data, phpval.NewMap(), nil, validation.Rules{validation.F("date", pastDate(true)...)}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("conditions: date: %w", err)
	}
	return d, nil
}

// validateEnrol is POST /conditions/enrolments {program, enrolled_on?} (default today).
func validateEnrol(body phpval.Map, locale string, now time.Time) (string, civildate.Date, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, []string{"program", "enrolled_on"}, validation.Rules{
		F("program", "required", "string", validation.In(Programs...)),
		F("enrolled_on", pastDate(false)...),
	}, locale, now)
	if err != nil {
		return "", civildate.Date{}, err
	}
	p, _ := data.Get("program")
	on := civildate.InTehran(now)
	if raw, ok := data.Get("enrolled_on"); ok && raw != nil {
		if on, err = civildate.Parse(phpval.ToString(raw)); err != nil {
			return "", civildate.Date{}, fmt.Errorf("conditions: enrolled_on: %w", err)
		}
	}
	return phpval.ToString(p), on, nil
}

// validateProgram checks the {program} segment of DELETE /conditions/enrolments/{program}.
func validateProgram(raw, locale string, now time.Time) (string, error) {
	data := phpval.NewMap()
	data.Set("program", raw)
	if _, err := validate(data, phpval.NewMap(), nil,
		validation.Rules{validation.F("program", "required", validation.In(Programs...))}, locale, now); err != nil {
		return "", err
	}
	return raw, nil
}

// painFields are the PUT /conditions/pain/{date} keys (anything else is ignored).
var painFields = []string{"score", "locations", "relief", "types", "associated", "missed_activity", "analgesic",
	"analgesic_time", "analgesic_effect"}

func validatePain(date civildate.Date, body phpval.Map, ch PainChoices, locale string, now time.Time) (PainInput, error) {
	F, in := validation.F, validation.In
	data, err := validate(phpval.NewMap(), body, painFields, validation.Rules{
		F("score", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxScore)),
		F("locations", "nullable", "array"),
		F("locations.*", "required", "string", in(ch.Locations...)),
		F("relief", "nullable", "array"),
		F("relief.*", "required", "string", in(ch.Relief...)),
		F("types", "nullable", "array"),
		F("types.*", "required", "string", in(ch.Types...)),
		F("associated", "nullable", "array"),
		F("associated.*", "required", "string", in(ch.Associated...)),
		F("missed_activity", "nullable", "boolean"),
		F("analgesic", "nullable", "string", "max:"+strconv.Itoa(MaxAnalgesicLen)),
		F("analgesic_time", "nullable", "date_format:H:i"),
		F("analgesic_effect", "nullable", in(AnalgesicEffects...)),
	}, locale, now)
	if err != nil {
		return PainInput{}, err
	}
	out := PainInput{Date: date, Set: map[string]bool{}}
	for _, k := range painFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		out.Set[k] = true
		if v == nil {
			continue
		}
		switch k {
		case "score":
			n := intOf(v)
			out.Score = &n
		case "locations":
			out.Locations = listOf(v, ch.Locations)
		case "relief":
			out.Relief = listOf(v, ch.Relief)
		case "types":
			out.Types = listOf(v, ch.Types)
		case "associated":
			out.Associated = listOf(v, ch.Associated)
		case "missed_activity":
			b := boolOf(v)
			out.MissedActivity = &b
		case "analgesic", "analgesic_time", "analgesic_effect":
			s := strings.TrimSpace(phpval.ToString(v))
			if s == "" {
				continue
			}
			switch k {
			case "analgesic":
				out.Analgesic = &s
			case "analgesic_time":
				out.AnalgesicTime = &s
			default:
				out.AnalgesicEffect = &s
			}
		}
	}
	return out, nil
}

// validatePMDD is PUT /conditions/pmdd/{date} {scores: {item: 1–6 | null}}; items are the active pmdd_items codes
// plus the codes the day already stores.
func validatePMDD(date civildate.Date, body phpval.Map, items []string, locale string, now time.Time) (PMDDInput, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, []string{"scores"}, validation.Rules{
		F("scores", "required", "array"),
		// min/max, not between: the fa validation bundle (Laravel's) has no `between` line
		F("scores.*", "nullable", "integer", "min:"+strconv.Itoa(PMDDMin), "max:"+strconv.Itoa(PMDDMax)),
	}, locale, now)
	if err != nil {
		return PMDDInput{}, err
	}
	raw, _ := data.Get("scores")
	keys, vals := phpval.Entries(raw)
	out := PMDDInput{Date: date, Scores: map[string]*int{}}
	for i, k := range keys {
		code := phpval.ToString(k)
		if !slices.Contains(items, code) {
			field := "scores." + code
			return PMDDInput{}, failValidation(locale, jsonx.Obj(field, []string{
				lang.Default().Trans("validation.in", map[string]string{"attribute": attributeOf("scores.*", field, locale)}, locale),
			}))
		}
		if vals[i] == nil {
			out.Scores[code] = nil
			continue
		}
		n := intOf(vals[i])
		out.Scores[code] = &n
	}
	return out, nil
}

// attributeOf is the attribute name of field (its wildcard's name, else the field with spaces).
func attributeOf(wildcard, field, locale string) string {
	kv := attributes(locale)
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i] == wildcard {
			return kv[i+1]
		}
	}
	return strings.ReplaceAll(field, "_", " ")
}

// pbacFields are the PUT /conditions/pbac/{date} keys.
var pbacFields = []string{"light", "medium", "heavy", "clots", "flooding"}

func validatePBAC(date civildate.Date, body phpval.Map, clots []string, locale string, now time.Time) (PBACInput, error) {
	F := validation.F
	pads := []any{"nullable", "integer", "min:0", "max:" + strconv.Itoa(MaxPads)}
	data, err := validate(phpval.NewMap(), body, pbacFields, validation.Rules{
		F("light", pads...),
		F("medium", pads...),
		F("heavy", pads...),
		F("clots", "nullable", validation.In(clots...)),
		F("flooding", "nullable", "boolean"),
	}, locale, now)
	if err != nil {
		return PBACInput{}, err
	}
	out := PBACInput{Date: date, Set: map[string]bool{}}
	for _, k := range pbacFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		out.Set[k] = true
		if v == nil {
			continue
		}
		switch k {
		case "light", "medium", "heavy":
			n := intOf(v)
			switch k {
			case "light":
				out.Light = &n
			case "medium":
				out.Medium = &n
			default:
				out.Heavy = &n
			}
		case "clots":
			s := phpval.ToString(v)
			out.Clots = &s
		case "flooding":
			b := boolOf(v)
			out.Flooding = &b
		}
	}
	return out, nil
}
