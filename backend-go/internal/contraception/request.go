package contraception

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// methodFields are the PUT /contraception/method body keys (anything else is ignored).
var methodFields = []string{
	"method", "pack_type", "pack_started_on", "packs_left", "reminder_time", "reminder_enabled",
	"inserted_on", "iud_lifetime_years", "followup_done", "injected_on", "replace_on",
}

func messages(locale string) validation.Option {
	future := T("validation.date_future", locale)
	return validation.Messages(
		"date.before_or_equal", future,
		"pack_started_on.before_or_equal", future,
		"inserted_on.before_or_equal", future,
		"injected_on.before_or_equal", future,
	)
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

// fieldError is a 422 with one message on field.
func fieldError(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

// pastDate is an optional day up to today (Tehran), required when the required_if rule (if any) says so.
func pastDate(requiredIf string) []any {
	if requiredIf == "" {
		return []any{"nullable", "date_format:Y-m-d", "before_or_equal:today"}
	}
	return []any{requiredIf, "nullable", "date_format:Y-m-d", "before_or_equal:today"}
}

func validatedDate(data phpval.Map, key string) (civildate.Date, error) {
	raw, ok := data.Get(key)
	if !ok || raw == nil || phpval.ToString(raw) == "" {
		return civildate.Date{}, nil
	}
	d, err := civildate.Parse(phpval.ToString(raw))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("contraception: %s: %w", key, err)
	}
	return d, nil
}

func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

// boolOf reads a validated `boolean` field (true, false, 1, 0, "1", "0").
func boolOf(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	default:
		s := strings.TrimSpace(phpval.ToString(v))
		return s == "1" || s == "true"
	}
}

// validateMethod is PUT /contraception/method.
func validateMethod(body phpval.Map, locale string, now time.Time) (Input, error) {
	F, in := validation.F, validation.In
	pills := "required_if:method," + MethodCombinedPill + "," + MethodProgestinPill
	iuds := "required_if:method," + MethodCopperIUD + "," + MethodHormonalIUD
	data, err := validate(phpval.NewMap(), body, methodFields, validation.Rules{
		F("method", "required", in(Methods...)),
		F("pack_type", "required_if:method,"+MethodCombinedPill, "nullable", in(PackTypes...)),
		F("pack_started_on", pastDate(pills)...),
		F("packs_left", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxPacksLeft)),
		F("reminder_time", "nullable", "date_format:H:i"),
		F("reminder_enabled", "sometimes", "boolean"),
		F("inserted_on", pastDate(iuds)...),
		F("iud_lifetime_years", iuds, "nullable", "integer",
			"min:"+strconv.Itoa(MinIUDLifetime), "max:"+strconv.Itoa(MaxIUDLifetime)),
		F("followup_done", "sometimes", "boolean"),
		F("injected_on", pastDate("required_if:method,"+MethodInjection)...),
		F("replace_on", "nullable", "date_format:Y-m-d"),
	}, locale, now)
	if err != nil {
		return Input{}, err
	}
	out := Input{}
	if v, ok := data.Get("method"); ok {
		out.Method = phpval.ToString(v)
	}
	if v, ok := data.Get("pack_type"); ok && v != nil {
		out.PackType = phpval.ToString(v)
	}
	if v, ok := data.Get("packs_left"); ok && v != nil {
		n := intOf(v)
		out.PacksLeft = &n
	}
	if v, ok := data.Get("reminder_time"); ok && v != nil {
		if m, ok := notifications.ParseClock(phpval.ToString(v)); ok {
			out.ReminderMinute = &m
		}
	}
	if v, ok := data.Get("reminder_enabled"); ok {
		b := boolOf(v)
		out.ReminderEnabled = &b
	}
	if v, ok := data.Get("iud_lifetime_years"); ok && v != nil {
		out.IUDLifetimeYears = intOf(v)
	}
	if v, ok := data.Get("followup_done"); ok {
		out.FollowupDone = boolOf(v)
	}
	for key, dst := range map[string]*civildate.Date{
		"pack_started_on": &out.PackStartedOn, "inserted_on": &out.InsertedOn,
		"injected_on": &out.InjectedOn, "replace_on": &out.ReplaceOn,
	} {
		if *dst, err = validatedDate(data, key); err != nil {
			return Input{}, err
		}
	}
	return out, nil
}

// PillInput is a validated POST /contraception/pills.
type PillInput struct {
	Date   civildate.Date
	Status string
}

// validatePill is POST /contraception/pills {date?, status?}: date defaults to today, status to taken.
func validatePill(body phpval.Map, locale string, now time.Time) (PillInput, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, []string{"date", "status"}, validation.Rules{
		F("date", pastDate("")...),
		F("status", "nullable", validation.In(PillStatuses...)),
	}, locale, now)
	if err != nil {
		return PillInput{}, err
	}
	out := PillInput{Date: civildate.InTehran(now), Status: StatusTaken}
	if d, err := validatedDate(data, "date"); err != nil {
		return PillInput{}, err
	} else if !d.IsZero() {
		out.Date = d
	}
	if v, ok := data.Get("status"); ok && v != nil {
		out.Status = phpval.ToString(v)
	}
	return out, nil
}

// parseDate validates the {date} segment alone: any well-formed date.
func parseDate(raw, locale string, now time.Time) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	if _, err := validate(data, phpval.NewMap(), nil,
		validation.Rules{validation.F("date", "required", "date_format:Y-m-d")}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("contraception: date: %w", err)
	}
	return d, nil
}
