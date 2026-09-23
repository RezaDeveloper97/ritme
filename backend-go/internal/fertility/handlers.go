package fertility

import (
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

// Handlers are the /api/v1/fertility actions. Mount them behind the locale middleware and auth
// RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(db DB, base clock.Clock) *Handlers {
	return &Handlers{svc: NewService(db), clock: base}
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

// Today is GET /fertility/today: the home tiles (LH, BBT, intercourse), the chance and the
// cycle day of today (Tehran), in three queries.
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	today := civildate.InTehran(h.now(c))
	day, cx, err := h.svc.Load(c, userID, today, today)
	if err != nil {
		return err
	}
	return httpx.OK(c, day.TodayJSON(cx, i18n.Locale(c)))
}

// ShowDay is GET /fertility/days/{date}: the merged day (any date; a future one carries the
// predicted chance).
func (h *Handlers) ShowDay(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseDate(dateParam(c), locale, now)
	if err != nil {
		return err
	}
	day, cx, err := h.svc.Load(c, userID, date, civildate.InTehran(now))
	if err != nil {
		return err
	}
	return httpx.OK(c, day.JSON(cx, locale))
}

// UpdateDay is PUT /fertility/days/{date}: a partial update (omitted keys stay, null clears),
// then the merged day. A luteal-spotting warning from the health log rides along as the
// top-level `warning`, like POST /health-logs.
func (h *Handlers) UpdateDay(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateDay(dateParam(c), validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	warning, err := h.svc.Save(c, userID, in, locale, now)
	if err != nil {
		return err
	}
	day, cx, err := h.svc.Load(c, userID, in.Date, civildate.InTehran(now))
	if err != nil {
		return err
	}
	body := httpx.Envelope(day.JSON(cx, locale), T("messages.day_saved", locale))
	if warning != nil {
		body.Set("warning", warning)
	}
	return httpx.JSON(c, fiber.StatusOK, body)
}

// dateParam is the {date} route segment, URL-decoded.
func dateParam(c fiber.Ctx) string {
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

// BBT is GET /fertility/bbt?range=1|3|6: the basal-temperature chart of the current cycle and,
// for range 3 / 6, the previous cycles aligned by cycle day (internal/fertility/bbt).
func (h *Handlers) BBT(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	rangeSize, err := parseRange(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := h.svc.BBT(c, userID, civildate.InTehran(now), rangeSize)
	if err != nil {
		return err
	}
	return httpx.OK(c, BBTJSON(res, rangeSize, locale))
}

// Insights is GET /fertility/insights: the cycle view's fertile window with the confidence,
// evidence rows, past ovulation days and tips that explain it (insights.go).
func (h *Handlers) Insights(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	ins, err := h.svc.Insights(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	return httpx.OK(c, InsightsJSON(ins, i18n.Locale(c)))
}
