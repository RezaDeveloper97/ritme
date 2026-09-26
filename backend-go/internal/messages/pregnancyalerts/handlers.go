package pregnancyalerts

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Handlers serve GET /pregnancy/v2/alerts and POST /pregnancy/v2/alerts/{id}/actions/{action};
// mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	eng   *Engine
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock.
func NewHandlers(q store.Querier, base clock.Clock) *Handlers {
	return &Handlers{eng: New(q), clock: base}
}

// Engine returns the handlers' engine (the log-save hooks share it).
func (h *Handlers) Engine() *Engine { return h.eng }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// LangOf is the request locale + the site's default language.
func LangOf(c fiber.Ctx) v2.Lang {
	return v2.Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

func mapErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotActive):
		return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", locale), "error_code", "pregnancy_not_active")
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound()
	}
	return err
}

// Index is GET /pregnancy/v2/alerts.
func (h *Handlers) Index(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	l := LangOf(c)
	out, err := h.eng.List(c, uid, h.now(c), l)
	if err != nil {
		return mapErr(err, l.Locale)
	}
	return httpx.OK(c, out)
}

// Action is POST /pregnancy/v2/alerts/{id}/actions/{action} (ack | add_to_visit_note).
func (h *Handlers) Action(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	l := LangOf(c)
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || !contains(Actions, c.Params("action")) {
		return httpx.NotFound()
	}
	act := c.Params("action")
	out, err := h.eng.Act(c, uid, id, act, h.now(c), l)
	if err != nil {
		return mapErr(err, l.Locale)
	}
	msg := T("messages.action_done", l.Locale)
	if act == "add_to_visit_note" {
		msg = T("messages.visit_note_added", l.Locale)
	}
	return httpx.OK(c, out, msg)
}

// AfterSave is the hook the v1 log-save handlers call (errors bubble up as 500).
func (e *Engine) AfterSave(ctx context.Context, userID uint64, locale, defaultLocale string, now time.Time) error {
	_, err := e.Evaluate(ctx, userID, now, v2.Lang{Locale: locale, Default: defaultLocale})
	return err
}
