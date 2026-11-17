package inbox

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Error codes of the 409 answers.
const (
	CodeAckRequired = "ack_required"
	CodeClosed      = "alert_closed"
)

// Handlers serve /api/v1/alerts (Go-only, D-164). Every answer is private, no-store.
type Handlers struct {
	s     *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewHandlers(s *Service, base clock.Clock) *Handlers { return &Handlers{s: s, clock: base} }

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return uid, nil
}

func invalid(field, msg string) error {
	ve := httpx.NewValidationError()
	ve.Add(field, msg)
	errs, _ := ve.Body().Get("errors")
	return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
}

// Index is GET /alerts?state=open|history&cursor= → data {items: [item…], next_cursor}.
func (h *Handlers) Index(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	state := c.Query("state", ListOpen)
	if state != ListOpen && state != ListHistory {
		return invalid("state", T("messages.invalid_state", locale))
	}
	cur, err := ParseCursor(c.Query("cursor"))
	if err != nil {
		return invalid("cursor", T("messages.invalid_cursor", locale))
	}
	page, err := h.s.List(c.Context(), uid, state, cur)
	if err != nil {
		return err
	}
	items, err := h.s.Items(c.Context(), page.Rows, locale)
	if err != nil {
		return err
	}
	var next any
	if page.Next != nil {
		next = page.Next.Encode()
	}
	return httpx.OK(c, jsonx.Obj("items", items, "next_cursor", next))
}

// id is the :id parameter; a malformed id is a 404 like a foreign one.
func id(c fiber.Ctx) (uint64, bool) {
	n, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return n, err == nil && n > 0
}

// fail maps the action errors.
func fail(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound()
	case errors.Is(err, ErrAckRequired):
		return httpx.Fail(fiber.StatusConflict, T("messages.ack_required", locale), "error_code", CodeAckRequired)
	case errors.Is(err, ErrClosed):
		return httpx.Fail(fiber.StatusConflict, T("messages.alert_closed", locale), "error_code", CodeClosed)
	}
	return err
}

// act runs one action and answers {alert} with msg.
func (h *Handlers) act(c fiber.Ctx, run func(uid, id uint64) (Row, error), msgKey string) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	n, ok := id(c)
	if !ok {
		return httpx.NotFound()
	}
	row, err := run(uid, n)
	if err != nil {
		return fail(err, locale)
	}
	items, err := h.s.Items(c.Context(), []Row{row}, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("alert", items[0]), T(msgKey, locale))
}

// Ack is POST /alerts/{id}/ack.
func (h *Handlers) Ack(c fiber.Ctx) error {
	return h.act(c, func(uid, n uint64) (Row, error) {
		r, err := h.s.Ack(c.Context(), uid, n, civildate.LocalNow(c, clock.FromContext(c, h.clock)))
		return r, err
	}, "messages.acknowledged")
}

// Dismiss is POST /alerts/{id}/dismiss: 409 ack_required for an urgent row not yet acknowledged.
func (h *Handlers) Dismiss(c fiber.Ctx) error {
	return h.act(c, func(uid, n uint64) (Row, error) {
		r, err := h.s.Dismiss(c.Context(), uid, n, civildate.LocalNow(c, clock.FromContext(c, h.clock)))
		return r, err
	}, "messages.dismissed")
}
