package fertility

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// dayFields are the PUT body keys (anything else is ignored).
var dayFields = []string{"lh", "mucus", "bbt", "bbt_time", "intercourse", "symptoms", "note"}

// BBT bounds of the log (°C). POST /health-logs allows 35–42; the TTC log is stricter.
const (
	bbtMin = "35"
	bbtMax = "38.5"
)

// dateRules validate the {date} route segment; a write refuses a future day.
func dateRules(write bool) []any {
	r := []any{"required", "date_format:Y-m-d"}
	if write {
		r = append(r, "before_or_equal:today")
	}
	return r
}

// dayRules validate a PUT body plus its {date}. Every field is optional; null clears it.
func dayRules() validation.Rules {
	F, in := validation.F, validation.In
	return validation.Rules{
		F("date", dateRules(true)...),
		F("lh", "nullable", in(LHResults...)),
		F("mucus", "nullable", in(CervicalMucus...)),
		F("bbt", "nullable", "numeric", "min:"+bbtMin, "max:"+bbtMax),
		F("bbt_time", "nullable", "date_format:H:i"),
		F("intercourse", "nullable", in(IntercourseTypes...)),
		F("symptoms", "nullable", "array"),
		F("symptoms.*", "required", in(Symptoms...)),
		F("note", "nullable", "string", "max:2000"),
	}
}

func messages(locale string) validation.Option {
	bbt := T("validation.bbt_range", locale)
	return validation.Messages(
		"bbt.min", bbt,
		"bbt.max", bbt,
		"date.before_or_equal", T("validation.date_future", locale),
	)
}

// DayInput is a validated PUT: Set says which keys the body sent (a sent null clears).
type DayInput struct {
	Date civildate.Date
	Set  map[string]bool
	// Values of the sent keys; nil = clear. BBT keeps the request's number/string form.
	LH, Mucus, BBTTime, Intercourse, Note *string
	BBT                                   any
	Symptoms                              []string
}

// sent reports whether any of keys was in the body.
func (in DayInput) sent(keys ...string) bool {
	return slices.ContainsFunc(keys, func(k string) bool { return in.Set[k] })
}

// parseDate validates the {date} segment alone (GET).
func parseDate(raw, locale string, now time.Time) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	v := validation.Make(lang.Default(), locale, data, validation.Rules{validation.F("date", dateRules(false)...)},
		validation.Now(now), validation.Attributes(attributes(locale)...), messages(locale))
	if v.Fails() {
		return civildate.Date{}, failValidation(locale, v.ErrorBag())
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("fertility: date: %w", err)
	}
	return d, nil
}

// validateDay runs the PUT rules over the body's known keys and the {date} segment.
func validateDay(rawDate string, body phpval.Map, locale string, now time.Time) (DayInput, error) {
	data := phpval.NewMap()
	data.Set("date", rawDate)
	for _, k := range dayFields {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, dayRules(),
		validation.Now(now), validation.Attributes(attributes(locale)...), messages(locale))
	if v.Fails() {
		return DayInput{}, failValidation(locale, v.ErrorBag())
	}
	date, err := civildate.Parse(rawDate)
	if err != nil {
		return DayInput{}, fmt.Errorf("fertility: date: %w", err)
	}
	in := DayInput{Date: date, Set: map[string]bool{}}
	str := func(k string) *string {
		x, _ := data.Get(k)
		if x == nil {
			return nil
		}
		s := phpval.ToString(x)
		return &s
	}
	for _, k := range dayFields {
		if _, ok := data.Get(k); !ok {
			continue
		}
		in.Set[k] = true
		switch k {
		case "lh":
			in.LH = str(k)
		case "mucus":
			in.Mucus = str(k)
		case "bbt_time":
			in.BBTTime = str(k)
		case "intercourse":
			in.Intercourse = str(k)
		case "note":
			in.Note = str(k)
		case "bbt":
			in.BBT, _ = data.Get(k)
		case "symptoms":
			raw, _ := data.Get(k)
			_, vals := phpval.Entries(raw)
			got := map[string]bool{}
			for _, x := range vals {
				got[phpval.ToString(x)] = true
			}
			in.Symptoms = []string{}
			for _, s := range Symptoms {
				if got[s] {
					in.Symptoms = append(in.Symptoms, s)
				}
			}
		}
	}
	return in, nil
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// parseRange validates GET /fertility/bbt's ?range= (1, 3 or 6; missing = 1).
func parseRange(query phpval.Map, locale string, now time.Time) (int, error) {
	data := phpval.NewMap()
	if v, ok := query.Get("range"); ok {
		data.Set("range", v)
	}
	allowed := make([]string, len(BBTRanges))
	for i, r := range BBTRanges {
		allowed[i] = strconv.Itoa(r)
	}
	v := validation.Make(lang.Default(), locale, data,
		validation.Rules{validation.F("range", "nullable", validation.In(allowed...))},
		validation.Now(now), validation.Attributes(attributes(locale)...), messages(locale))
	if v.Fails() {
		return 0, failValidation(locale, v.ErrorBag())
	}
	raw, _ := data.Get("range")
	if raw == nil {
		return BBTRanges[0], nil
	}
	n, err := strconv.Atoi(phpval.ToString(raw))
	if err != nil {
		return 0, fmt.Errorf("fertility: range: %w", err)
	}
	return n, nil
}
