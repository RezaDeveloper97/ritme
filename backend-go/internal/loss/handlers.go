package loss

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/loss actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (*auth.User, error) {
	u := auth.CurrentUser(c)
	if u == nil {
		return nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return u, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// mapErr turns the service's domain errors into calm responses: no loss → 404 loss_not_found, notes switched off →
// 503 note_unavailable, a note that no configured key opens → 409 note_unreadable (she may delete it).
func mapErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNoLoss):
		return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", "loss_not_found")
	case errors.Is(err, ErrNoteUnavailable):
		return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.note_unavailable", locale), "error_code", "note_unavailable")
	case errors.Is(err, ErrNoteUnreadable):
		return httpx.Fail(fiber.StatusConflict, T("messages.note_unreadable", locale), "error_code", "note_unreadable")
	}
	return err
}

// state answers the current GET /loss state (status 200, or 201 when created).
func (h *Handlers) state(c fiber.Ctx, userID uint64, created bool, msg ...string) error {
	st, err := h.svc.State(c, userID, h.now(c))
	if err != nil {
		return err
	}
	if created {
		return httpx.Created(c, StateJSON(st), msg...)
	}
	return httpx.OK(c, StateJSON(st), msg...)
}

// Show is GET /loss: the newest loss with its follow-up, moods and note flag (all null without one).
func (h *Handlers) Show(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	return h.state(c, u.ID, false)
}

// Store is POST /loss {type?, occurred_on?, notify_companion?}: records the loss and stops pregnancy content; 201 with
// the state (200 when it corrected today's record).
func (h *Handlers) Store(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateRecord(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	created, err := h.svc.Record(c, u, in, now, i18n.LanguagesOf(c))
	if err != nil {
		return err
	}
	return h.state(c, u.ID, created, T("messages.recorded", locale))
}

// Followup is PUT /loss/followup {bleeding_stopped?, beta_next_on?, beta_negative?, visit_at?}.
func (h *Handlers) Followup(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateFollowup(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Followup(c, u.ID, in, now, locale); err != nil {
		return mapErr(err, locale)
	}
	return h.state(c, u.ID, false, T("messages.followup_saved", locale))
}

// StoreMood is POST /loss/moods {mood, date?}.
func (h *Handlers) StoreMood(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	mood, day, err := validateMood(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Mood(c, u.ID, mood, day, now); err != nil {
		return mapErr(err, locale)
	}
	return h.state(c, u.ID, false, T("messages.mood_saved", locale))
}

// ShowNote is GET /loss/note: the private note, decrypted for its owner only.
func (h *Handlers) ShowNote(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	note, at, err := h.svc.Note(c, u.ID)
	if err != nil {
		return mapErr(err, i18n.Locale(c))
	}
	return httpx.OK(c, NoteJSON(note, at))
}

// UpdateNote is PUT /loss/note {note}: null or blank clears it.
func (h *Handlers) UpdateNote(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	if h.svc.notes.Disabled() {
		return mapErr(ErrNoteUnavailable, locale)
	}
	note, err := validateNote(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SetNote(c, u.ID, note, now); err != nil {
		return mapErr(err, locale)
	}
	saved, at, err := h.svc.Note(c, u.ID)
	if err != nil {
		return mapErr(err, locale)
	}
	msg := T("messages.note_saved", locale)
	if note == "" {
		msg = T("messages.note_cleared", locale)
	}
	return httpx.OK(c, NoteJSON(saved, at), msg)
}

// NextStep is PUT /loss/next-step {choice: cycle|ttc|nothing}.
func (h *Handlers) NextStep(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	choice, err := validateNextStep(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.NextStep(c, u, choice, now); err != nil {
		return mapErr(err, locale)
	}
	return h.state(c, u.ID, false, T("messages.next_step_saved", locale))
}

// DestroyNote is DELETE /loss/note: erases the newest loss's private note (also while notes are switched off).
func (h *Handlers) DestroyNote(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeleteNote(c, u.ID, h.now(c)); err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, NoteJSON("", sql.NullTime{}), T("messages.note_cleared", locale))
}

// Destroy is DELETE /loss: erases the newest loss record (moods, note, companion notices); pregnancy content stays
// stopped. Answers the state that remains (an earlier loss, or all null).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	u, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.Delete(c, u.ID); err != nil {
		return mapErr(err, locale)
	}
	return h.state(c, u.ID, false, T("messages.erased", locale))
}
