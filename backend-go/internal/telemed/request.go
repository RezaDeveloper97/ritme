package telemed

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
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

func truthy(data phpval.Map, key string) bool {
	v, _ := data.Get(key)
	return phpval.Truthy(v)
}

// Filter is a validated directory query.
type Filter struct {
	Mode      string // "" = any
	Kind      string // "" = any
	Specialty string // catalog code, "" = any
	City      string // catalog code, "" = any
	Insurance string // catalog code, InsurerAny, or "" = any
	Today     bool   // only doctors with a free slot today
	Q         string // name search
}

var filterKeys = []string{"mode", "kind", "specialty", "city", "insurance", "today", "q"}

// MaxCodeLen is the longest specialty / city / insurer code (catalog_items.code).
const MaxCodeLen = 64

// MaxSearchLen is the longest name search.
const MaxSearchLen = 100

// ValidateFilter is the query of GET /telemed/doctors?mode=&kind=&specialty=&city=&insurance=&today=&q=. An unknown
// mode or kind is a 422; an unknown specialty / city / insurer code simply matches nobody.
func ValidateFilter(query phpval.Map, locale string, now time.Time) (Filter, error) {
	data := pick(query, filterKeys)
	rules := validation.Rules{
		validation.F("mode", "nullable", "string", validation.In(Modes...)),
		validation.F("kind", "nullable", "string", validation.In(Kinds...)),
		validation.F("specialty", "nullable", "string", "max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("city", "nullable", "string", "max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("insurance", "nullable", "string", "max:"+strconv.Itoa(MaxCodeLen)),
		validation.F("today", "nullable", "boolean"),
		validation.F("q", "nullable", "string", "max:"+strconv.Itoa(MaxSearchLen)),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return Filter{}, err
	}
	return Filter{
		Mode: str(data, "mode"), Kind: str(data, "kind"), Specialty: str(data, "specialty"), City: str(data, "city"),
		Insurance: str(data, "insurance"), Today: truthy(data, "today"), Q: str(data, "q"),
	}, nil
}

// SlotsInput is a validated GET /telemed/doctors/{id}/slots query.
type SlotsInput struct {
	Mode string         // "" = the doctor's first offered mode
	From civildate.Date // never before today
	Days int
}

var slotKeys = []string{"mode", "from", "days"}

// ValidateSlots is ?mode=&from=Y-m-d&days=1…31 (from defaults to today and is clamped to it, days to 7).
func ValidateSlots(query phpval.Map, locale string, now time.Time) (SlotsInput, error) {
	data := pick(query, slotKeys)
	rules := validation.Rules{
		validation.F("mode", "nullable", "string", validation.In(Modes...)),
		validation.F("from", "nullable", "date_format:Y-m-d"),
		validation.F("days", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxSlotDays)),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return SlotsInput{}, err
	}
	today := civildate.InTehran(now)
	in := SlotsInput{Mode: str(data, "mode"), From: today, Days: DefaultSlotDays}
	if s := str(data, "from"); s != "" {
		if d, err := civildate.Parse(s); err == nil && d.After(today) {
			in.From = d
		}
	}
	if s := str(data, "days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			in.Days = n
		}
	}
	return in, nil
}

// ReviewInput is a validated POST /telemed/doctors/{id}/reviews body.
type ReviewInput struct {
	Rating int
	Body   string
}

var reviewKeys = []string{"rating", "body"}

// ValidateReview is {rating: 1–5, body?: ≤ 1000 characters}.
func ValidateReview(body phpval.Map, locale string, now time.Time) (ReviewInput, error) {
	data := pick(body, reviewKeys)
	rules := validation.Rules{
		validation.F("rating", "required", "integer", "min:1", "max:5"),
		validation.F("body", "nullable", "string", "max:"+strconv.Itoa(MaxReviewLen)),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return ReviewInput{}, err
	}
	n, _ := strconv.Atoi(str(data, "rating"))
	return ReviewInput{Rating: n, Body: str(data, "body")}, nil
}
