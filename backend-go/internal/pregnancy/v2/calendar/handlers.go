package calendar

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Handlers serve the pregnancy v2 calendar; mount behind the locale middleware and RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(q store.Querier, base clock.Clock) *Handlers {
	return &Handlers{svc: NewService(q), clock: base}
}

// Calendar is GET /pregnancy/v2/calendar?month=YYYY-MM (the locale's calendar; default: this month).
func (h *Handlers) Calendar(c fiber.Ctx) error {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
	data := phpval.NewMap()
	if raw := c.Query("month"); raw != "" {
		data.Set("month", raw)
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("month", "nullable", "string", `regex:/^\d{4}-(0[1-9]|1[0-2])$/`),
	}, validation.Now(now), validation.Attributes("month", T("attributes.month", locale)))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, v2.T("messages.validation_failed", locale), "errors", v.ErrorBag())
	}
	var month *[2]int
	if raw, ok := data.Get("month"); ok {
		s := phpval.ToString(raw)
		y, _ := strconv.Atoi(s[:4])
		m, _ := strconv.Atoi(s[5:])
		month = &[2]int{y, m}
	}
	out, err := h.svc.Calendar(c, userID, month, now, v2.Lang{Locale: locale, Default: i18n.LanguagesOf(c).DefaultCode()})
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}
