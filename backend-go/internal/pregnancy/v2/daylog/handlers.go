package daylog

import (
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Handlers are the day-log actions; mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(q store.Querier, base clock.Clock) *Handlers {
	return &Handlers{svc: NewService(q), clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, v2.T("messages.validation_failed", locale), "errors", errs)
}

func fieldError(locale, field, msg string) error {
	return failValidation(locale, jsonx.Obj(field, []string{msg}))
}

// dayParam is {date}: strict YYYY-MM-DD, not after today (422 otherwise).
func dayParam(c fiber.Ctx, today civildate.Date, locale string) (civildate.Date, error) {
	d, err := civildate.Parse(c.Params("date"))
	if err != nil {
		return civildate.Date{}, fieldError(locale, "date", T("validation.invalid_date", locale))
	}
	if d.After(today) {
		return civildate.Date{}, fieldError(locale, "date", T("validation.future_date", locale))
	}
	return d, nil
}

// Show is GET /pregnancy/v2/days/{date}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := dayParam(c, civildate.InTehran(now), locale)
	if err != nil {
		return err
	}
	out, err := h.svc.Day(c, userID, date, now, locale, nil)
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

var bodyFields = []string{"mood", "water_glasses", "weight", "visit_note", "symptoms"}

// Update is PUT /pregnancy/v2/days/{date}: absent field = keep, null = clear.
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := dayParam(c, civildate.InTehran(now), locale)
	if err != nil {
		return err
	}
	body := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range bodyFields {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	F := validation.F
	rules := validation.Rules{
		F("mood", "nullable", "integer", "min:1", "max:5"),
		F("water_glasses", "nullable", "integer", "min:0", "max:15"),
		F("weight", "nullable", "numeric", "min:30", "max:200"),
		F("visit_note", "nullable", "string", "max:2000"),
		F("symptoms", "nullable", "array"),
	}
	for _, s := range Symptoms {
		rules = append(rules, F("symptoms."+s, "nullable", validation.In(Severities...)))
	}
	v := validation.Make(lang.Default(), locale, data, rules,
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}

	var in Input
	if x, ok := data.Get("symptoms"); ok {
		m := map[string]string{}
		if x != nil {
			keys, vals := phpval.Entries(x)
			for i, k := range keys {
				if !slices.Contains(Symptoms, k) {
					return fieldError(locale, "symptoms", T("validation.unknown_symptom", locale, "key", k))
				}
				if vals[i] != nil {
					m[k] = phpval.ToString(vals[i])
				}
			}
		}
		in.Symptoms = &m
	}
	intField := func(k string) **int {
		x, ok := data.Get(k)
		if !ok {
			return nil
		}
		var p *int
		if x != nil {
			n := int(phpval.ToFloat(x))
			p = &n
		}
		return &p
	}
	in.Mood, in.Water = intField("mood"), intField("water_glasses")
	if x, ok := data.Get("weight"); ok {
		var p *float64
		if x != nil {
			f := phpval.ToFloat(x)
			p = &f
		}
		in.Weight, in.WeightRaw = &p, x
	}
	if x, ok := data.Get("visit_note"); ok {
		var p *string
		if x != nil {
			p = trimNote(phpval.ToString(x))
		}
		in.VisitNote = &p
	}

	out, err := h.svc.Save(c, userID, date, in, now, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.day_saved", locale))
}

// Report is GET /pregnancy/v2/report?from=&to= (doctor PDF timeline).
func (h *Handlers) Report(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	data := phpval.NewMap()
	for _, k := range []string{"from", "to"} {
		if s := c.Query(k); s != "" {
			data.Set(k, s)
		}
	}
	F := validation.F
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		F("from", "required", "date_format:Y-m-d"),
		F("to", "required", "date_format:Y-m-d"),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	fs, _ := data.Get("from")
	ts, _ := data.Get("to")
	from, err1 := civildate.Parse(phpval.ToString(fs))
	to, err2 := civildate.Parse(phpval.ToString(ts))
	if err1 != nil || err2 != nil {
		return fieldError(locale, "from", T("validation.invalid_date", locale))
	}
	if to.Before(from) {
		return fieldError(locale, "to", T("validation.range_order", locale))
	}
	if from.DiffDays(to) >= MaxReportDays {
		return fieldError(locale, "to", T("validation.range_too_long", locale))
	}
	out, err := h.svc.Report(c, userID, from, to, now, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}
