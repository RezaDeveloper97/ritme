package healthlog

import (
	"net/url"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// notFoundMessage is the (English-only) 404 of show / destroy.
const notFoundMessage = "No health log found for this date"

// Handlers are the DailyHealthLogController actions. Mount them behind auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (the request clock from
// clock.Middleware wins).
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

// Enums is GET /health-logs/enums.
func (h *Handlers) Enums(c fiber.Ctx) error {
	return httpx.OK(c, model.EnumValues())
}

// Index is GET /health-logs: the raw LengthAwarePaginator, 30 per page, newest first,
// optional from_date / to_date (whereDate on the raw values).
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	input := validation.Input(c)
	var f Filter
	if v, ok := input.Get("from_date"); ok {
		f.HasFrom, f.From = true, stringOrNil(v)
	}
	if v, ok := input.Get("to_date"); ok {
		f.HasTo, f.To = true, stringOrNil(v)
	}
	page := httpx.PageParam(c)
	logs, total, err := h.svc.List(c.Context(), userID, f, page)
	if err != nil {
		return err
	}
	p := httpx.NewPage(logs, total, PerPage, page, httpx.RequestURL(c))
	return httpx.OK(c, p.Raw())
}

func stringOrNil(v any) any {
	if s, ok := v.(string); ok {
		return s
	}
	return nil
}

// Store is POST /health-logs.
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := clock.FromContext(c, h.clock).Now()
	attrs, err := ValidateStore(validation.Input(c), i18n.Locale(c), now)
	if err != nil {
		return err
	}
	res, err := h.svc.Store(c.Context(), userID, attrs, i18n.ResolveLocale(c, ""), now)
	if err != nil {
		return err
	}
	msg, status := "Health log updated successfully", fiber.StatusOK
	if res.Created {
		msg, status = "Health log created successfully", fiber.StatusCreated
	}
	body := httpx.Envelope(res.Data(), msg)
	if res.Warning != nil {
		body.Set("warning", res.Warning)
	}
	return httpx.JSON(c, status, body)
}

// Show is GET /health-logs/{date}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, err := h.svc.Find(c.Context(), userID, dateParam(c))
	if err != nil {
		return err
	}
	if l == nil {
		return httpx.Fail(fiber.StatusNotFound, notFoundMessage)
	}
	return httpx.OK(c, l.ToArray())
}

// Destroy is DELETE /health-logs/{date}.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, err := h.svc.Find(c.Context(), userID, dateParam(c))
	if err != nil {
		return err
	}
	if l == nil {
		return httpx.Fail(fiber.StatusNotFound, notFoundMessage)
	}
	if err := h.svc.Delete(c.Context(), l); err != nil {
		return err
	}
	return httpx.Message(c, "Health log deleted successfully")
}

// dateParam is the {date} route segment, URL-decoded as Laravel's router does.
func dateParam(c fiber.Ctx) string {
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}
