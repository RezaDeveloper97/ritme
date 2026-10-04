package tools

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Error codes of the non-validation failures.
const (
	ErrorCodeNotFound           = "session_not_found"
	ErrorCodeKickActive         = "kick_session_active"
	ErrorCodeSessionEnded       = "session_ended"
	ErrorCodeKickLimit          = "kick_limit"
	ErrorCodeContractionRunning = "contraction_running"
	ErrorCodeNoContraction      = "no_contraction_running"
	ErrorCodePregnancyNotActive = "pregnancy_not_active"
)

// Handlers are the /api/v1/pregnancy/kick-sessions and /contractions actions. Mount them behind the locale
// middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

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

func langOf(c fiber.Ctx) v2.Lang {
	return v2.Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

// fail maps the domain errors onto {success:false, message, error_code}.
func fail(err error, locale string) error {
	code := func(status int, key, code string) error {
		return httpx.Fail(status, T("messages."+key, locale), "error_code", code)
	}
	switch {
	case errors.Is(err, ErrNotActive):
		return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", locale), "error_code", ErrorCodePregnancyNotActive)
	case errors.Is(err, ErrNotFound):
		return code(fiber.StatusNotFound, "not_found", ErrorCodeNotFound)
	case errors.Is(err, ErrSessionActive):
		return code(fiber.StatusConflict, "kick_active", ErrorCodeKickActive)
	case errors.Is(err, ErrSessionEnded):
		return code(fiber.StatusConflict, "session_ended", ErrorCodeSessionEnded)
	case errors.Is(err, ErrKickLimit):
		return code(fiber.StatusUnprocessableEntity, "kick_limit", ErrorCodeKickLimit)
	case errors.Is(err, ErrContractionRunning):
		return code(fiber.StatusConflict, "contraction_running", ErrorCodeContractionRunning)
	case errors.Is(err, ErrNoContraction):
		return code(fiber.StatusConflict, "no_contraction", ErrorCodeNoContraction)
	}
	return err
}

// target resolves the user and :id (a malformed id is the uniform 404).
func (h *Handlers) target(c fiber.Ctx) (uint64, uint64, error) {
	userID, err := h.user(c)
	if err != nil {
		return 0, 0, err
	}
	id, perr := strconv.ParseUint(c.Params("id"), 10, 64)
	if perr != nil || id == 0 {
		return 0, 0, fail(ErrNotFound, i18n.Locale(c))
	}
	return userID, id, nil
}

// ---------------------------------------------------------------- kick counter

// Kicks is GET /pregnancy/kick-sessions: the running session, today's total and the last sessions.
func (h *Handlers) Kicks(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	o, err := h.svc.Kicks(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, KickOverviewJSON(o, now))
}

// StartKicks is POST /pregnancy/kick-sessions (201; 409 while one runs or outside pregnancy).
func (h *Handlers) StartKicks(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	k, err := h.svc.StartKicks(c, userID, now)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.Created(c, KickJSON(k, now))
}

// Kick is POST /pregnancy/kick-sessions/{id}/kicks: one movement.
func (h *Handlers) Kick(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	k, err := h.svc.Kick(c, userID, id, now)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.OK(c, KickJSON(k, now))
}

// UndoKick is DELETE /pregnancy/kick-sessions/{id}/kicks: takes the last movement back.
func (h *Handlers) UndoKick(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	k, err := h.svc.UndoKick(c, userID, id, now)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.OK(c, KickJSON(k, now))
}

// StopKicks is POST /pregnancy/kick-sessions/{id}/stop («پایان و ذخیره»).
func (h *Handlers) StopKicks(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	k, err := h.svc.StopKicks(c, userID, id, now)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.OK(c, KickJSON(k, now), T("messages.saved", i18n.Locale(c)))
}

// DestroyKicks is DELETE /pregnancy/kick-sessions/{id}.
func (h *Handlers) DestroyKicks(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteKicks(c, userID, id, h.now(c)); err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.Message(c, T("messages.deleted", i18n.Locale(c)))
}

// ---------------------------------------------------------------- contraction timer

// Contractions is GET /pregnancy/contractions: the live 5-1-1 thresholds, the running session and the history.
func (h *Handlers) Contractions(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), langOf(c)
	p, err := h.svc.Params(c, l)
	if err != nil {
		return err
	}
	active, err := h.svc.ActiveTiming(c, userID)
	if err != nil {
		return err
	}
	history, err := h.svc.History(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, TimingOverviewJSON(active, history, p, now))
}

// StartContraction is POST /pregnancy/contractions/start: a contraction begins (a session opens on the first one).
func (h *Handlers) StartContraction(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), langOf(c)
	t, err := h.svc.StartContraction(c, userID, now)
	if err != nil {
		return fail(err, l.Locale)
	}
	p, err := h.svc.Params(c, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, TimingJSON(t, p, now, true))
}

// StopContraction is POST /pregnancy/contractions/stop: the contraction ends; the 5-1-1 rule runs and any alert it
// raises (pregnancy alert engine) comes back in alerts.
func (h *Handlers) StopContraction(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), langOf(c)
	t, alerts, err := h.svc.StopContraction(c, userID, now, l)
	if err != nil {
		return fail(err, l.Locale)
	}
	p, err := h.svc.Params(c, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("session", TimingJSON(t, p, now, true), "alerts", jsonx.List(alerts)))
}

// Timing is GET /pregnancy/contractions/sessions/{id}.
func (h *Handlers) Timing(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), langOf(c)
	t, err := h.svc.TimingByID(c, userID, id)
	if err != nil {
		return fail(err, l.Locale)
	}
	p, err := h.svc.Params(c, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, TimingJSON(t, p, now, true))
}

// FinishTiming is POST /pregnancy/contractions/sessions/{id}/finish («توقف و ذخیره»).
func (h *Handlers) FinishTiming(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), langOf(c)
	t, err := h.svc.FinishTiming(c, userID, id, now)
	if err != nil {
		return fail(err, l.Locale)
	}
	p, err := h.svc.Params(c, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, TimingJSON(t, p, now, true), T("messages.saved", l.Locale))
}

// DestroyTiming is DELETE /pregnancy/contractions/sessions/{id}.
func (h *Handlers) DestroyTiming(c fiber.Ctx) error {
	userID, id, err := h.target(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteTiming(c, userID, id); err != nil {
		return fail(err, i18n.Locale(c))
	}
	return httpx.Message(c, T("messages.deleted", i18n.Locale(c)))
}
