package postpartum

import (
	"fmt"
	"math"
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

func messages(locale string) validation.Option {
	return validation.Messages("birth_date.before_or_equal", T("validation.birth_date_future", locale))
}

// validate runs rules over the keys of body and returns the validated data (keys not sent stay absent).
func validate(body phpval.Map, keys []string, rules validation.Rules, locale string, now time.Time) (phpval.Map, error) {
	data := phpval.NewMap()
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

// fieldFail is a 422 on one field with a postpartum validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

func parseDate(v any) (civildate.Date, error) {
	d, err := civildate.Parse(phpval.ToString(v))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("postpartum: date: %w", err)
	}
	return d, nil
}

// stringsOf are the strings of a validated array, de-duplicated, in the order of allowed.
func stringsOf(v any, allowed []string) []string {
	_, vals := phpval.Entries(v)
	got := make([]string, 0, len(vals))
	for _, x := range vals {
		got = append(got, phpval.ToString(x))
	}
	return ordered(got, allowed)
}

// validateActivate is POST /postpartum/activate {birth_date, delivery_type?, baby_count?}: the birth in the last
// MaxBirthAgeDays days, not in the future.
func validateActivate(body phpval.Map, locale string, now time.Time) (ActivateInput, error) {
	F := validation.F
	data, err := validate(body, []string{"birth_date", "delivery_type", "baby_count"}, validation.Rules{
		F("birth_date", "required", "date_format:Y-m-d", "before_or_equal:today"),
		F("delivery_type", "nullable", validation.In(DeliveryTypes...)),
		F("baby_count", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxBabyCount)),
	}, locale, now)
	if err != nil {
		return ActivateInput{}, err
	}
	raw, _ := data.Get("birth_date")
	birth, err := parseDate(raw)
	if err != nil {
		return ActivateInput{}, err
	}
	if birth.Before(civildate.InTehran(now).AddDays(-MaxBirthAgeDays)) {
		return ActivateInput{}, fieldFail(locale, "birth_date", "birth_date_too_old")
	}
	in := ActivateInput{BirthDate: birth, BabyCount: 1}
	if v, ok := data.Get("delivery_type"); ok && v != nil {
		in.DeliveryType = phpval.ToString(v)
	}
	if v, ok := data.Get("baby_count"); ok && v != nil {
		in.BabyCount = intOf(v)
	}
	return in, nil
}

// validateDate is ?date= / body date (a past or current day; default today).
func validateDate(src phpval.Map, locale string, now time.Time) (civildate.Date, error) {
	data, err := validate(src, []string{"date"}, validation.Rules{
		validation.F("date", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
	}, locale, now)
	if err != nil {
		return civildate.Date{}, err
	}
	if v, ok := data.Get("date"); ok && v != nil {
		return parseDate(v)
	}
	return civildate.InTehran(now), nil
}

var recoveryFields = []string{"lochia_amount", "lochia_color", "pain_level", "pain_locations", "breasts", "feeds_count", "sleep_hours"}

// validateRecovery is PUT /postpartum/recovery {lochia_amount?, lochia_color?, pain_level?, pain_locations?,
// breasts?, feeds_count?, sleep_hours?} — partial: keys not sent stay, null clears. Pain is one value: the level with
// its locations (a level without a location is refused; the stored side fills a half sent); breasts [] = normal.
func validateRecovery(body phpval.Map, stored Recovery, locale string, now time.Time) (RecoveryInput, error) {
	F := validation.F
	data, err := validate(body, recoveryFields, validation.Rules{
		F("lochia_amount", "sometimes", "nullable", validation.In(LochiaAmounts...)),
		F("lochia_color", "sometimes", "nullable", validation.In(LochiaColors...)),
		F("pain_level", "sometimes", "nullable", validation.In(PainLevels...)),
		F("pain_locations", "sometimes", "nullable", "array"),
		F("pain_locations.*", "required", "string", validation.In(PainLocations...)),
		F("breasts", "sometimes", "nullable", "array"),
		F("breasts.*", "required", "string", validation.In(BreastSymptoms...)),
		F("feeds_count", "sometimes", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxFeeds)),
		F("sleep_hours", "sometimes", "nullable", "numeric", "min:0", "max:"+strconv.Itoa(MaxSleepHours)),
	}, locale, now)
	if err != nil {
		return RecoveryInput{}, err
	}
	in := RecoveryInput{Set: map[string]bool{}, PainLevel: stored.PainLevel, PainLocations: stored.PainLocations}
	for _, k := range recoveryFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		in.Set[k] = true
		switch k {
		case "lochia_amount":
			if v != nil {
				in.LochiaAmount = phpval.ToString(v)
			}
		case "lochia_color":
			if v != nil {
				in.LochiaColor = phpval.ToString(v)
			}
		case "pain_level":
			in.PainLevel = ""
			if v != nil {
				in.PainLevel = phpval.ToString(v)
			}
		case "pain_locations":
			in.PainLocations = nil
			if v != nil {
				in.PainLocations = stringsOf(v, PainLocations)
			}
		case "breasts":
			if v == nil {
				in.BreastsNull = true
			} else {
				in.Breasts = stringsOf(v, BreastSymptoms)
			}
		case "feeds_count":
			if v != nil {
				n := intOf(v)
				in.FeedsCount = &n
			}
		case "sleep_hours":
			if v != nil {
				f := phpval.ToFloat(v)
				if f*2 != math.Trunc(f*2) {
					return RecoveryInput{}, fieldFail(locale, "sleep_hours", "sleep_step")
				}
				in.SleepHours = &f
			}
		}
	}
	painSent := in.Set["pain_level"] || in.Set["pain_locations"]
	if painSent && slices.Contains([]string{"mild", "moderate", "severe"}, in.PainLevel) && len(in.PainLocations) == 0 {
		return RecoveryInput{}, fieldFail(locale, "pain_locations", "pain_locations_required")
	}
	return in, nil
}

// validateKind is ?kind= (short | full, default short).
func validateKind(query phpval.Map, locale string, now time.Time) (string, error) {
	data, err := validate(query, []string{"kind"}, validation.Rules{
		validation.F("kind", "nullable", validation.In(Kinds...)),
	}, locale, now)
	if err != nil {
		return "", err
	}
	if v, ok := data.Get("kind"); ok && v != nil {
		return phpval.ToString(v), nil
	}
	return KindShort, nil
}

// validateCheck is POST /postpartum/epds {kind, answers: {item code: score 0–3}}: every item of the kind answered,
// no other key.
func validateCheck(body phpval.Map, locale string, now time.Time) (string, map[string]int, error) {
	F := validation.F
	kindData, err := validate(body, []string{"kind", "answers"}, validation.Rules{
		F("kind", "required", validation.In(Kinds...)),
		F("answers", "required", "array"),
	}, locale, now)
	if err != nil {
		return "", nil, err
	}
	rawKind, _ := kindData.Get("kind")
	kind := phpval.ToString(rawKind)
	items := ItemsOf(kind)
	rules := validation.Rules{F("answers", "required", "array")}
	codes := make([]string, 0, len(items))
	for _, it := range items {
		codes = append(codes, it.Code)
		rules = append(rules, F("answers."+it.Code, "required", "integer", "min:0", "max:"+strconv.Itoa(ItemMax)))
	}
	data, err := validate(body, []string{"answers"}, rules, locale, now)
	if err != nil {
		return "", nil, err
	}
	raw, _ := data.Get("answers")
	keys, vals := phpval.Entries(raw)
	answers := make(map[string]int, len(codes))
	for i, k := range keys {
		code := phpval.ToString(k)
		if !slices.Contains(codes, code) {
			return "", nil, failValidation(locale, jsonx.Obj("answers."+code, []string{
				lang.Default().Trans("validation.in", map[string]string{"attribute": attributeOf("answers.*", locale)}, locale),
			}))
		}
		answers[code] = intOf(vals[i])
	}
	if len(answers) != len(codes) {
		return "", nil, fieldFail(locale, "answers", "answers_missing")
	}
	return kind, answers, nil
}

// validateLimit is ?limit= (1–HistoryLimit, default 26).
func validateLimit(query phpval.Map, locale string, now time.Time) (int, error) {
	data, err := validate(query, []string{"limit"}, validation.Rules{
		validation.F("limit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(HistoryLimit)),
	}, locale, now)
	if err != nil {
		return 0, err
	}
	if v, ok := data.Get("limit"); ok && v != nil {
		return intOf(v), nil
	}
	return HistoryLimit / 2, nil
}

// attributeOf is the validation attribute name of key (the key itself when there is none).
func attributeOf(key, locale string) string {
	kv := attributes(locale)
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i] == key {
			return kv[i+1]
		}
	}
	return strings.ReplaceAll(key, "_", " ")
}
