package contraception

import (
	"errors"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/contraception actions. Mount them behind the locale middleware and auth RequireUser.
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

// overview renders the contraception screen of today, optionally with a message.
func (h *Handlers) overview(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	o, err := h.svc.Overview(c, userID, civildate.InTehran(now))
	if err != nil {
		return err
	}
	return httpx.OK(c, OverviewJSON(o), msg...)
}

// Show is GET /contraception: the switch, the method (null when none), the pill pack (pill methods) and the care
// reminders the method created.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.overview(c, userID, h.now(c))
}

// SaveMethod is PUT /contraception/method: saves the method (a full replace), switches tracking on, sets the one
// pill reminder and syncs the method's care reminders; then the overview.
func (h *Handlers) SaveMethod(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateMethod(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SaveMethod(c, userID, in, now, locale); err != nil {
		return err
	}
	return h.overview(c, userID, now, T("messages.method_saved", locale))
}

// StopMethod is DELETE /contraception/method: method and its reminders go, tracking and the pill reminder go off,
// the pill log stays.
func (h *Handlers) StopMethod(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	if err := h.svc.Stop(c, userID, now); err != nil {
		return err
	}
	return h.overview(c, userID, now, T("messages.method_removed", i18n.Locale(c)))
}

// LogPill is POST /contraception/pills {date?, status?}: logs a pill day (taken by default), then the overview.
// 422 on `method` without a pill method, on `date` before the pack start or on a break day.
func (h *Handlers) LogPill(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validatePill(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	switch err := h.svc.LogPill(c, userID, in.Date, in.Status, now); {
	case errors.Is(err, ErrNoPillMethod):
		return fieldError(locale, "method", "pill_method_required")
	case errors.Is(err, ErrBeforePack):
		return fieldError(locale, "date", "date_before_pack")
	case errors.Is(err, ErrBreakDay):
		return fieldError(locale, "date", "break_day")
	case err != nil:
		return err
	}
	return h.overview(c, userID, now, T("messages.pill_saved", locale))
}

// UnlogPill is DELETE /contraception/pills/{date}: removes the day's log (undo), then the overview.
func (h *Handlers) UnlogPill(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		raw = s
	}
	date, err := parseDate(raw, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UnlogPill(c, userID, date); err != nil {
		return err
	}
	return h.overview(c, userID, now, T("messages.pill_removed", locale))
}
