package pelvic

import (
	"fmt"
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

// Session bounds: one POST is one guided session (the board's level 2 is 3 sets, 5 minutes).
const (
	maxSets        = 20
	maxDurationSec = 3600
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
	return validation.Messages(
		"date.before_or_equal", future,
		"started_on.before_or_equal", future,
	)
}

// validate runs rules over the keys of body (plus extra fields already set in data) and returns the data.
func validate(data phpval.Map, body phpval.Map, keys []string, rules validation.Rules, locale string, now time.Time) (phpval.Map, error) {
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

// programRequired is the 422 of a session saved without a program.
func programRequired(locale string) error {
	return failValidation(locale, jsonx.Obj("program", []string{T("validation.program_required", locale)}))
}

// dateOr parses a validated optional date, def when absent or null.
func dateOr(data phpval.Map, key string, def civildate.Date) (civildate.Date, error) {
	raw, ok := data.Get(key)
	if !ok || raw == nil {
		return def, nil
	}
	d, err := civildate.Parse(phpval.ToString(raw))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("pelvic: %s: %w", key, err)
	}
	return d, nil
}

// intOf is a validated integer field.
func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

// validateProgram is POST /pelvic/program: {started_on?} (default today).
func validateProgram(body phpval.Map, locale string, now time.Time) (civildate.Date, error) {
	data, err := validate(phpval.NewMap(), body, []string{"started_on"},
		validation.Rules{validation.F("started_on", pastDate(false)...)}, locale, now)
	if err != nil {
		return civildate.Date{}, err
	}
	return dateOr(data, "started_on", civildate.InTehran(now))
}

// SessionInput is a validated POST /pelvic/sessions.
type SessionInput struct {
	Date          civildate.Date
	SetsCompleted int
	DurationSec   int
}

func validateSession(body phpval.Map, locale string, now time.Time) (SessionInput, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, []string{"date", "sets_completed", "duration_sec"}, validation.Rules{
		F("date", pastDate(false)...),
		F("sets_completed", "required", "integer", "min:0", "max:"+strconv.Itoa(maxSets)),
		F("duration_sec", "required", "integer", "min:1", "max:"+strconv.Itoa(maxDurationSec)),
	}, locale, now)
	if err != nil {
		return SessionInput{}, err
	}
	date, err := dateOr(data, "date", civildate.InTehran(now))
	if err != nil {
		return SessionInput{}, err
	}
	sets, _ := data.Get("sets_completed")
	dur, _ := data.Get("duration_sec")
	return SessionInput{Date: date, SetsCompleted: intOf(sets), DurationSec: intOf(dur)}, nil
}

// parseDiaryDate validates the {date} segment alone (GET): any well-formed date.
func parseDiaryDate(raw, locale string, now time.Time) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	if _, err := validate(data, phpval.NewMap(), nil,
		validation.Rules{validation.F("date", "required", "date_format:Y-m-d")}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("pelvic: date: %w", err)
	}
	return d, nil
}

// diaryFields are the PUT body keys (anything else is ignored).
var diaryFields = []string{"leak", "night_voids", "uti_symptoms"}

// DiaryInput is a validated PUT /pelvic/diary/{date}: Set says which keys the body sent (a sent null clears).
type DiaryInput struct {
	Date        civildate.Date
	Set         map[string]bool
	Leak        *string
	NightVoids  *int
	UTISymptoms []string // nil = clear
}

func validateDiary(rawDate string, body phpval.Map, locale string, now time.Time) (DiaryInput, error) {
	F, in := validation.F, validation.In
	data := phpval.NewMap()
	data.Set("date", rawDate)
	data, err := validate(data, body, diaryFields, validation.Rules{
		F("date", pastDate(true)...),
		F("leak", "nullable", in(Leaks...)),
		F("night_voids", "nullable", "integer", "min:0", "max:"+strconv.Itoa(maxNightVoids)),
		F("uti_symptoms", "nullable", "array"),
		F("uti_symptoms.*", "required", in(UTISymptoms...)),
	}, locale, now)
	if err != nil {
		return DiaryInput{}, err
	}
	date, err := civildate.Parse(rawDate)
	if err != nil {
		return DiaryInput{}, fmt.Errorf("pelvic: date: %w", err)
	}
	out := DiaryInput{Date: date, Set: map[string]bool{}}
	for _, k := range diaryFields {
		v, ok := data.Get(k)
		if !ok {
			continue
		}
		out.Set[k] = true
		if v == nil {
			continue
		}
		switch k {
		case "leak":
			s := phpval.ToString(v)
			out.Leak = &s
		case "night_voids":
			n := intOf(v)
			out.NightVoids = &n
		case "uti_symptoms":
			_, vals := phpval.Entries(v)
			got := map[string]bool{}
			for _, x := range vals {
				got[phpval.ToString(x)] = true
			}
			out.UTISymptoms = []string{}
			for _, s := range UTISymptoms {
				if got[s] {
					out.UTISymptoms = append(out.UTISymptoms, s)
				}
			}
		}
	}
	return out, nil
}
