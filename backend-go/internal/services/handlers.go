package services

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Handlers are the /api/v1/services actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

// Show is GET /services: the hub of the current user.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	langs := i18n.LanguagesOf(c)
	loc := catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
	hub, err := h.svc.Hub(c, userID, now, loc)
	if err != nil {
		return err
	}
	return httpx.OK(c, hub)
}
