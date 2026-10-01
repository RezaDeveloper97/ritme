package menopause

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// startSlack tolerates a client clock slightly ahead of the server's.
const startSlack = time.Minute

func messages(locale string) validation.Option {
	future := T("validation.date_future", locale)
	return validation.Messages("last_period.before_or_equal", future, "month.before_or_equal", future)
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

// fieldFail is a 422 on one field with a menopause validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

func boolOf(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	s := strings.ToLower(phpval.ToString(v))
	return s == "1" || s == "true" || s == "on" || s == "yes"
}

// listOf are the strings of a validated array, de-duplicated, in the order of allowed.
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

func parseDate(v any) (civildate.Date, error) {
	d, err := civildate.Parse(phpval.ToString(v))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("menopause: date: %w", err)
	}
	return d, nil
}

var profileFields = []string{"stage", "last_period", "surgical", "hrt"}

// validateProfile is PUT /menopause/profile: a partial update of {stage, last_period, surgical, hrt}; null clears
// last_period / surgical / hrt (the stage, once given, is required).
func validateProfile(body phpval.Map, locale string, now time.Time) (ProfileChange, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, profileFields, validation.Rules{
		F("stage", "sometimes", "required", validation.In(enums.MenopauseStageValues()...)),
		F("last_period", "sometimes", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("surgical", "sometimes", "nullable", "boolean"),
		F("hrt", "sometimes", "nullable", "boolean"),
	}, locale, now)
	if err != nil {
		return ProfileChange{}, err
	}
	ch := ProfileChange{Set: map[string]bool{}}
	for _, k := range profileFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		ch.Set[k] = true
		if v == nil {
			continue
		}
		switch k {
		case "stage":
			ch.Stage = phpval.ToString(v)
		case "last_period":
			d, err := parseDate(v)
			if err != nil {
				return ProfileChange{}, err
			}
			ch.LastPeriod = &d
		case "surgical":
			b := boolOf(v)
			ch.Surgical = &b
		case "hrt":
			b := boolOf(v)
			ch.HRT = &b
		}
	}
	return ch, nil
}

var flashDetailFields = []string{"severity", "night", "sweat", "triggers"}

func flashDetailRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("severity", "nullable", validation.In(Severities...)),
		F("night", "nullable", "boolean"),
		F("sweat", "nullable", "boolean"),
		F("triggers", "nullable", "array"),
		F("triggers.*", "required", "string", validation.In(Triggers()...)),
	}
}

func flashDetails(data phpval.Map, in *FlashInput) {
	for _, k := range flashDetailFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		in.Set[k] = true
		if v == nil {
			continue
		}
		switch k {
		case "severity":
			in.Severity = phpval.ToString(v)
		case "night":
			b := boolOf(v)
			in.Night = &b
		case "sweat":
			b := boolOf(v)
			in.Sweat = &b
		case "triggers":
			in.Triggers = listOf(v, Triggers())
		}
	}
}

// validateFlashStart is POST /menopause/hot-flashes {started_at?, duration_s?, severity?, night?, sweat?,
// triggers?}: without duration_s it starts the timer, with it it logs a finished flash. started_at is a date-time
// (Tehran when it has no offset) in the last MaxBackfillDays days, not in the future.
func validateFlashStart(body phpval.Map, locale string, now time.Time) (FlashInput, error) {
	F := validation.F
	rules := append(validation.Rules{
		F("started_at", "nullable", "string", "date"),
		F("duration_s", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxFlashSeconds)),
	}, flashDetailRules()...)
	data, err := validate(phpval.NewMap(), body, append([]string{"started_at", "duration_s"}, flashDetailFields...), rules, locale, now)
	if err != nil {
		return FlashInput{}, err
	}
	in := FlashInput{Set: map[string]bool{}}
	if v, ok := data.Get("started_at"); ok && v != nil {
		t, err := civildate.ParseLenient(phpval.ToString(v), now, civildate.Tehran)
		if err != nil {
			return FlashInput{}, fieldFail(locale, "started_at", "date_future")
		}
		t = t.In(civildate.Tehran)
		switch {
		case t.After(now.Add(startSlack)):
			return FlashInput{}, fieldFail(locale, "started_at", "date_future")
		case t.Before(now.AddDate(0, 0, -MaxBackfillDays)):
			return FlashInput{}, fieldFail(locale, "started_at", "started_too_old")
		}
		if t.After(now) {
			t = now
		}
		in.StartedAt = &t
	}
	if v, ok := data.Get("duration_s"); ok && v != nil {
		n := intOf(v)
		in.Duration = &n
	}
	flashDetails(data, &in)
	return in, nil
}

// validateFlashStop is POST /menopause/hot-flashes/{id}/stop {severity?, night?, sweat?, triggers?} (keys not sent
// stay; null clears severity / triggers).
func validateFlashStop(body phpval.Map, locale string, now time.Time) (FlashInput, error) {
	data, err := validate(phpval.NewMap(), body, flashDetailFields, flashDetailRules(), locale, now)
	if err != nil {
		return FlashInput{}, err
	}
	in := FlashInput{Set: map[string]bool{}}
	flashDetails(data, &in)
	return in, nil
}

// parseID is the {id} segment (a positive integer; anything else is a 404 like a missing row).
func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && id > 0
}

// validateDayQuery is ?date= (a day; default today).
func validateDayQuery(query phpval.Map, locale string, now time.Time) (civildate.Date, error) {
	data, err := validate(phpval.NewMap(), query, []string{"date"},
		validation.Rules{validation.F("date", "nullable", "date_format:Y-m-d")}, locale, now)
	if err != nil {
		return civildate.Date{}, err
	}
	if v, ok := data.Get("date"); ok && v != nil {
		return parseDate(v)
	}
	return civildate.InTehran(now), nil
}

// validateMonths is ?months= (1–MaxHistoryMonths, default TrendMonths).
func validateMonths(query phpval.Map, locale string, now time.Time) (int, error) {
	data, err := validate(phpval.NewMap(), query, []string{"months"}, validation.Rules{
		validation.F("months", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxHistoryMonths)),
	}, locale, now)
	if err != nil {
		return 0, err
	}
	if v, ok := data.Get("months"); ok && v != nil {
		return intOf(v), nil
	}
	return TrendMonths, nil
}

// ScoreInput is a validated POST /menopause/scores.
type ScoreInput struct {
	Month   civildate.Date // first day of the Jalali month
	Answers map[string]int
}

// validateScore is POST /menopause/scores {month?, answers: {question: 0–max}}: every active question answered,
// month = any day of the (past or current) month, default this month.
func validateScore(body phpval.Map, sc Scale, locale string, now time.Time) (ScoreInput, error) {
	F := validation.F
	rules := validation.Rules{
		F("month", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("answers", "required", "array"),
	}
	for _, q := range sc.Questions {
		rules = append(rules, F("answers."+q.Code, "required", "integer", "min:0", "max:"+strconv.Itoa(q.Max)))
	}
	data, err := validate(phpval.NewMap(), body, []string{"month", "answers"}, rules, locale, now)
	if err != nil {
		return ScoreInput{}, err
	}
	in := ScoreInput{Month: MonthStart(civildate.InTehran(now)), Answers: map[string]int{}}
	if v, ok := data.Get("month"); ok && v != nil {
		d, err := parseDate(v)
		if err != nil {
			return ScoreInput{}, err
		}
		in.Month = MonthStart(d)
	}
	raw, _ := data.Get("answers")
	keys, vals := phpval.Entries(raw)
	codes := sc.Codes()
	for i, k := range keys {
		code := phpval.ToString(k)
		if !slices.Contains(codes, code) {
			field := "answers." + code
			return ScoreInput{}, failValidation(locale, jsonx.Obj(field, []string{
				lang.Default().Trans("validation.in", map[string]string{"attribute": attributeOf("answers.*", locale)}, locale),
			}))
		}
		in.Answers[code] = intOf(vals[i])
	}
	if len(codes) == 0 {
		return ScoreInput{}, fieldFail(locale, "answers", "answers_missing")
	}
	return in, nil
}

// attributeOf is the validation attribute name of key (the key itself when there is none).
func attributeOf(key, locale string) string {
	kv := attributes(locale)
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i] == key {
			return kv[i+1]
		}
	}
	return key
}
