package v2

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
)

// Handlers are the /api/v1/pregnancy/v2 actions; mount them behind the locale middleware and
// auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(q store.Querier, base clock.Clock) *Handlers {
	return &Handlers{svc: NewService(q), clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func langOf(c fiber.Ctx) Lang {
	return Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

// weekParam is {n}: 1..MaxWeek, else 404.
func weekParam(c fiber.Ctx) (int, error) {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil || n < 1 || n > MaxWeek {
		return 0, httpx.NotFound()
	}
	return n, nil
}

// DatingPreview is POST /pregnancy/v2/dating-preview (no write).
func (h *Handlers) DatingPreview(c fiber.Ctx) error {
	if _, err := h.user(c); err != nil {
		return err
	}
	out, err := h.svc.Preview(c, validation.Input(c), h.now(c), langOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// Today is GET /pregnancy/v2/today.
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	out, err := h.svc.Today(c, userID, h.now(c), langOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// Week is GET /pregnancy/v2/weeks/{n}.
func (h *Handlers) Week(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	n, err := weekParam(c)
	if err != nil {
		return err
	}
	out, err := h.svc.Week(c, userID, n, h.now(c), langOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// WeekState is PUT /pregnancy/v2/weeks/{n}/state: {bookmarked?, done_task_keys?}.
func (h *Handlers) WeekState(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	n, err := weekParam(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	body := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range []string{"bookmarked", "done_task_keys"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	F := validation.F
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		F("bookmarked", "nullable", "boolean"),
		F("done_task_keys", "nullable", "array", "max:50"),
		F("done_task_keys.*", "required", "string", "max:64"),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	var in StateInput
	if x, _ := data.Get("bookmarked"); x != nil {
		b := phpval.Truthy(x)
		in.Bookmarked = &b
	}
	if x, ok := data.Get("done_task_keys"); ok {
		in.DoneTaskKeys = []string{}
		_, vals := phpval.Entries(x)
		for _, k := range vals {
			in.DoneTaskKeys = append(in.DoneTaskKeys, phpval.ToString(k))
		}
	}
	out, err := h.svc.SaveState(c, userID, n, in, now, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.state_saved", locale))
}

// SetupCopy is GET /pregnancy/v2/setup-copy (no pregnancy needed: the Setup screen runs before it).
func (h *Handlers) SetupCopy(c fiber.Ctx) error {
	if _, err := h.user(c); err != nil {
		return err
	}
	out, err := h.svc.SetupCopy(c, langOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}
