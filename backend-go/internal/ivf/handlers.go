package ivf

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/ivf actions. Mount them behind the locale middleware and auth RequireUser.
// Health data: every action reads and writes the caller's own rows only.
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

// param is a route segment, URL-decoded.
func param(c fiber.Ctx, name string) string {
	raw := c.Params(name)
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

// mapErr turns the service's domain errors into responses.
func mapErr(err error, locale string) error {
	var fe *FieldError
	switch {
	case errors.Is(err, ErrNoCycle):
		return fieldFail(locale, "cycle", "no_cycle")
	case errors.Is(err, ErrCycleOpen):
		return fieldFail(locale, "cycle", "cycle_open")
	case errors.Is(err, ErrMedNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.med_not_found", locale))
	case errors.Is(err, ErrMedicationLimit):
		msg := T("messages.medication_limit", locale)
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg,
			"errors", jsonx.Obj("limit", []string{msg}), "error_code", care.ErrorCodeLimitReached)
	case errors.As(err, &fe):
		return fieldFail(locale, fe.Field, fe.Key)
	}
	return err
}

// medID is the {id} segment; a malformed id is the same 404 as another user's.
func medID(c fiber.Ctx, locale string) (uint64, error) {
	id, err := strconv.ParseUint(param(c, "id"), 10, 64)
	if err != nil || id == 0 {
		return 0, mapErr(ErrMedNotFound, locale)
	}
	return id, nil
}

func (h *Handlers) home(c fiber.Ctx, userID uint64, now time.Time, status int, msg ...string) error {
	v, err := h.svc.Home(c, userID, now)
	if err != nil {
		return err
	}
	if status == fiber.StatusCreated {
		return httpx.Created(c, HomeJSON(v), msg...)
	}
	return httpx.OK(c, HomeJSON(v), msg...)
}

func (h *Handlers) meds(c fiber.Ctx, userID uint64, now time.Time, status int, msg ...string) error {
	v, err := h.svc.Meds(c, userID, now)
	if err != nil {
		return err
	}
	if status == fiber.StatusCreated {
		return httpx.Created(c, MedsJSON(v), msg...)
	}
	return httpx.OK(c, MedsJSON(v), msg...)
}

func (h *Handlers) scans(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	v, err := h.svc.Scans(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, ScansJSON(v), msg...)
}

func (h *Handlers) tww(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	v, err := h.svc.TWW(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, TWWJSON(v), msg...)
}

// Show is GET /ivf: the IVF home — the «IVF/IUI» switch, the open cycle with its timeline, today's injections, the
// next appointment, the latest scan and the companion context.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.home(c, userID, h.now(c), fiber.StatusOK)
}

// StartCycle is POST /ivf/cycles: opens a cycle (201 with the home); 422 on `cycle` when one is open.
func (h *Handlers) StartCycle(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateStart(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.StartCycle(c, userID, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.home(c, userID, now, fiber.StatusCreated, T("messages.cycle_started", locale))
}

// UpdateCycle is PUT /ivf/cycles/current: a partial update of the open cycle (stage, dates, protocol, companion
// reminders) that also syncs its care appointments; then the home.
func (h *Handlers) UpdateCycle(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	p, err := validatePatch(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateCycle(c, userID, p, now, locale); err != nil {
		return mapErr(err, locale)
	}
	return h.home(c, userID, now, fiber.StatusOK, T("messages.cycle_saved", locale))
}

// RecordOutcome is POST /ivf/cycles/current/outcome {result, date?}: closes the cycle; the data is the closed cycle
// and the next steps (positive → pregnancy setup, negative → loss path or a new cycle, cancelled → a new cycle).
func (h *Handlers) RecordOutcome(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateOutcome(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	closed, err := h.svc.RecordOutcome(c, userID, in.Result, in.Date, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, OutcomeJSON(closed, civildate.InTehran(now)), T("messages.outcome_saved", locale))
}

// ShowMeds is GET /ivf/meds: the injection schedule.
func (h *Handlers) ShowMeds(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.meds(c, userID, h.now(c), fiber.StatusOK)
}

// AddMed is POST /ivf/meds: adds a medicine to the open cycle (a care medication reminder); 201 with the schedule.
func (h *Handlers) AddMed(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateMed(validation.Input(c), locale, now, i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	if err := h.svc.AddMed(c, userID, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.meds(c, userID, now, fiber.StatusCreated, T("messages.med_added", locale))
}

// UpdateMed is PUT /ivf/meds/{id}: replaces the medicine; then the schedule.
func (h *Handlers) UpdateMed(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := medID(c, locale)
	if err != nil {
		return err
	}
	in, err := validateMed(validation.Input(c), locale, now, i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	if err := h.svc.UpdateMed(c, userID, id, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.meds(c, userID, now, fiber.StatusOK, T("messages.med_saved", locale))
}

// DeleteMed is DELETE /ivf/meds/{id}: removes the medicine and its care reminder; then the schedule.
func (h *Handlers) DeleteMed(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := medID(c, locale)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteMed(c, userID, id); err != nil {
		return mapErr(err, locale)
	}
	return h.meds(c, userID, now, fiber.StatusOK, T("messages.med_removed", locale))
}

// LogDose is POST /ivf/meds/{id}/doses {date?, slot, site?}: marks a dose taken (care intake + injection site);
// then the schedule.
func (h *Handlers) LogDose(c fiber.Ctx) error {
	return h.dose(c, true)
}

// UnlogDose is DELETE /ivf/meds/{id}/doses {date, slot}: undoes a dose; then the schedule.
func (h *Handlers) UnlogDose(c fiber.Ctx) error {
	return h.dose(c, false)
}

func (h *Handlers) dose(c fiber.Ctx, logging bool) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := medID(c, locale)
	if err != nil {
		return err
	}
	in, err := validateDose(validation.Input(c), locale, now, logging)
	if err != nil {
		return err
	}
	msg := "messages.dose_removed"
	if logging {
		msg = "messages.dose_saved"
		err = h.svc.LogDose(c, userID, id, in.Date, in.Slot, in.Site, now)
	} else {
		err = h.svc.UnlogDose(c, userID, id, in.Date, in.Slot)
	}
	if err != nil {
		return mapErr(err, locale)
	}
	return h.meds(c, userID, now, fiber.StatusOK, T(msg, locale))
}

// ShowScans is GET /ivf/scans: the open cycle's scans and the growth chart.
func (h *Handlers) ShowScans(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.scans(c, userID, h.now(c))
}

// SaveScan is PUT /ivf/scans/{date}: records (replaces) the scan of a day of the open cycle; then the scans.
func (h *Handlers) SaveScan(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseDate(param(c, "date"), locale, now, true)
	if err != nil {
		return err
	}
	in, err := validateScan(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SaveScan(c, userID, date, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.scans(c, userID, now, T("messages.scan_saved", locale))
}

// DeleteScan is DELETE /ivf/scans/{date}; then the scans.
func (h *Handlers) DeleteScan(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseDate(param(c, "date"), locale, now, false)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteScan(c, userID, date); err != nil {
		return mapErr(err, locale)
	}
	return h.scans(c, userID, now, T("messages.scan_removed", locale))
}

// ShowTWW is GET /ivf/tww: the two-week wait.
func (h *Handlers) ShowTWW(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.tww(c, userID, h.now(c))
}

// SaveMood is PUT /ivf/tww/{date} {mood}: the day's mood check-in (null clears); then the two-week wait.
func (h *Handlers) SaveMood(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseDate(param(c, "date"), locale, now, true)
	if err != nil {
		return err
	}
	mood, err := validateMood(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SaveMood(c, userID, date, mood, now); err != nil {
		return mapErr(err, locale)
	}
	return h.tww(c, userID, now, T("messages.mood_saved", locale))
}
