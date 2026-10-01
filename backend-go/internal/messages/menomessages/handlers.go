package menomessages

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Handlers serve the caller's menopause messages (read-only). Mount behind the locale middleware and auth
// RequireUser.
type Handlers struct {
	eng   *Engine
	clock clock.Clock
}

// NewHandlers wires the handlers on db and the menopause facts; base is the fallback clock.
func NewHandlers(db store.DBTX, src SignalSource, base clock.Clock) *Handlers {
	return &Handlers{eng: NewEngine(db, src), clock: base}
}

// Index is GET /messages/menopause: {success, data: Result.JSON()}.
func (h *Handlers) Index(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	langs := i18n.LanguagesOf(c)
	loc := catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
	res, err := h.eng.Evaluate(c.Context(), uid, now, loc)
	if err != nil {
		return err
	}
	return httpx.OK(c, res.JSON(loc))
}
