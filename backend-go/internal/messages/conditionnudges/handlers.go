package conditionnudges

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Handlers serve the nudges of the caller (read-only). Mount behind the locale middleware and auth RequireUser.
type Handlers struct {
	eng   *Engine
	clock clock.Clock
}

// NewHandlers wires the handlers on db; base is the fallback clock. No doctors directory yet (see DoctorDirectory).
func NewHandlers(db store.DBTX, base clock.Clock) *Handlers {
	return &Handlers{eng: NewEngine(db, nil), clock: base}
}

// Index is the caller's nudges for today's cycle: {success, data: Result.JSON()}.
func (h *Handlers) Index(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	today := civildate.Today(clock.FromContext(c, h.clock))
	res, err := h.eng.Evaluate(c.Context(), uid, today, i18n.Locale(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, res.JSON())
}
