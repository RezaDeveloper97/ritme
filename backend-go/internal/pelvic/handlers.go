package pelvic

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

// Handlers are the /api/v1/pelvic actions. Mount them behind the locale middleware and auth RequireUser.
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

// overview renders the plan screen of today, optionally with a message.
func (h *Handlers) overview(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	o, err := h.svc.Overview(c, userID, civildate.InTehran(now))
	if err != nil {
		return err
	}
	return httpx.OK(c, OverviewJSON(o, localizer(c)), msg...)
}

// Show is GET /pelvic: the program overview (null without one) and today's bladder diary.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.overview(c, userID, h.now(c))
}

// StartProgram is POST /pelvic/program {started_on?}: starts or restarts the program, then the overview.
func (h *Handlers) StartProgram(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	startedOn, err := validateProgram(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Start(c, userID, startedOn, now); err != nil {
		return err
	}
	return h.overview(c, userID, now, T("messages.program_started", locale))
}

// StopProgram is DELETE /pelvic/program: the program row goes, trained days and diary stay.
func (h *Handlers) StopProgram(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.svc.Stop(c, userID); err != nil {
		return err
	}
	now := h.now(c)
	return h.overview(c, userID, now, T("messages.program_stopped", i18n.Locale(c)))
}

// AddSession is POST /pelvic/sessions {date?, sets_completed, duration_sec}: adds a session to its day, then the
// overview. 422 on `program` without a program.
func (h *Handlers) AddSession(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateSession(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.AddSession(c, userID, in, now); err != nil {
		if errors.Is(err, ErrNoProgram) {
			return programRequired(locale)
		}
		return err
	}
	return h.overview(c, userID, now, T("messages.session_saved", locale))
}

// dateParam is the {date} route segment, URL-decoded.
func dateParam(c fiber.Ctx) string {
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

// ShowDiary is GET /pelvic/diary/{date}: one bladder-diary day (empty when nothing was logged).
func (h *Handlers) ShowDiary(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := parseDiaryDate(dateParam(c), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	d, err := h.svc.Diary(c, userID, date)
	if err != nil {
		return err
	}
	return httpx.OK(c, DiaryJSON(d))
}

// UpdateDiary is PUT /pelvic/diary/{date}: a partial update (omitted keys stay, null clears), then the day.
func (h *Handlers) UpdateDiary(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateDiary(dateParam(c), validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	d, err := h.svc.SaveDiary(c, userID, in, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, DiaryJSON(d), T("messages.diary_saved", locale))
}
