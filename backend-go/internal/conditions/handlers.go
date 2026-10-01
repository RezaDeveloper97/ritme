package conditions

import (
	"errors"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/conditions actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func localizer(c fiber.Ctx) catalog.Localizer {
	langs := i18n.LanguagesOf(c)
	return catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
}

// param is a route segment, URL-decoded.
func param(c fiber.Ctx, name string) string {
	raw := c.Params(name)
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

// writeErr maps the service's 422s.
func writeErr(err error, locale string) error {
	var fe *FieldError
	switch {
	case errors.Is(err, ErrNotEnrolled):
		return fieldFail(locale, "program", "program_required")
	case errors.As(err, &fe):
		return fieldFail(locale, fe.Field, fe.Key)
	}
	return err
}

func (h *Handlers) overview(c fiber.Ctx, userID uint64, msg ...string) error {
	list, err := h.svc.Enrolments(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, OverviewJSON(list), msg...)
}

// Show is GET /conditions: every program with the user's enrolment (hub order).
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.overview(c, userID)
}

// Enrol is POST /conditions/enrolments {program, enrolled_on?}: joins (again) and returns the overview.
func (h *Handlers) Enrol(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	program, on, err := validateEnrol(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Enrol(c, userID, program, on, now); err != nil {
		return err
	}
	return h.overview(c, userID, T("messages.enrolled", locale))
}

// Leave is DELETE /conditions/enrolments/{program}: the enrolment goes, the diary stays.
func (h *Handlers) Leave(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	program, err := validateProgram(param(c, "program"), locale, h.now(c))
	if err != nil {
		return err
	}
	if err := h.svc.Leave(c, userID, program); err != nil {
		return err
	}
	return h.overview(c, userID, T("messages.left", locale))
}

// ShowPain is GET /conditions/pain/{date}.
func (h *Handlers) ShowPain(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := parseDay(param(c, "date"), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	d, err := h.svc.Pain(c, userID, date)
	if err != nil {
		return err
	}
	return httpx.OK(c, PainJSON(d))
}

// SavePain is PUT /conditions/pain/{date}: a partial update (omitted keys stay, null clears), then the day.
func (h *Handlers) SavePain(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseLogDay(param(c, "date"), locale, now)
	if err != nil {
		return err
	}
	choices, err := h.svc.PainChoices(c, userID, date)
	if err != nil {
		return err
	}
	in, err := validatePain(date, validation.Input(c), choices, locale, now)
	if err != nil {
		return err
	}
	d, err := h.svc.SavePain(c, userID, in, locale, now)
	if err != nil {
		return writeErr(err, locale)
	}
	return httpx.OK(c, PainJSON(d), T("messages.pain_saved", locale))
}

// PMDDChart is GET /conditions/pmdd/chart: the current and up to two closed cycles, the pattern and the alerts.
func (h *Handlers) PMDDChart(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	ch, err := h.svc.PMDDChart(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	return httpx.OK(c, ChartJSON(ch, localizer(c)))
}

// ShowPMDD is GET /conditions/pmdd/{date}.
func (h *Handlers) ShowPMDD(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := parseDay(param(c, "date"), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	d, err := h.svc.PMDD(c, userID, date)
	if err != nil {
		return err
	}
	return httpx.OK(c, PMDDJSON(d))
}

// SavePMDD is PUT /conditions/pmdd/{date} {scores}: per item a 1–6 score or null; items not sent stay.
func (h *Handlers) SavePMDD(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseLogDay(param(c, "date"), locale, now)
	if err != nil {
		return err
	}
	items, err := h.svc.PMDDItemCodes(c)
	if err != nil {
		return err
	}
	cur, err := h.svc.PMDD(c, userID, date)
	if err != nil {
		return err
	}
	for _, s := range cur.Scores {
		if !contains(items, s.Code) {
			items = append(items, s.Code)
		}
	}
	in, err := validatePMDD(date, validation.Input(c), items, locale, now)
	if err != nil {
		return err
	}
	d, err := h.svc.SavePMDD(c, userID, in, now)
	if err != nil {
		return writeErr(err, locale)
	}
	return httpx.OK(c, PMDDJSON(d), T("messages.pmdd_saved", locale))
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ShowPBAC is GET /conditions/pbac/{date}: the day, its period score and the ≥ 100 alert.
func (h *Handlers) ShowPBAC(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := parseDay(param(c, "date"), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	v, err := h.svc.PBAC(c, userID, date)
	if err != nil {
		return err
	}
	return httpx.OK(c, PBACJSON(v, localizer(c)))
}

// SavePBAC is PUT /conditions/pbac/{date}: a partial update (omitted keys stay, null clears), then the day.
func (h *Handlers) SavePBAC(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseLogDay(param(c, "date"), locale, now)
	if err != nil {
		return err
	}
	clots, err := h.svc.ClotOptions(c, userID, date)
	if err != nil {
		return err
	}
	in, err := validatePBAC(date, validation.Input(c), clots, locale, now)
	if err != nil {
		return err
	}
	v, err := h.svc.SavePBAC(c, userID, in, locale, now)
	if err != nil {
		return writeErr(err, locale)
	}
	return httpx.OK(c, PBACJSON(v, localizer(c)), T("messages.pbac_saved", locale))
}
