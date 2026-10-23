package skips

import (
	"embed"
	"errors"
	"io/fs"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are POST /cycle/skips and DELETE /cycle/skips/{date}. Mount them behind the locale middleware, auth
// RequireUser and the write throttle.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (the request clock wins).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

// Store is POST /cycle/skips {expected_start}: 201 when recorded, 200 when the skip already existed (idempotent),
// 422 on invalid input or a refusal (`code`). The body is {expected_start}; the new cycle_view comes with T-M9-32.
func (h *Handlers) Store(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.ResolveLocale(c, "")
	nw := clock.FromContext(c, h.clock)
	input := validation.Input(c)
	v := validation.Make(lang.Default(), locale, input,
		validation.Rules{validation.F("expected_start", "required", "date_format:Y-m-d")}, validation.Now(nw.Now()))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, v.First(), "errors", v.ErrorBag())
	}
	raw, _ := input.Get("expected_start")
	s, _ := raw.(string)
	expected, err := civildate.Parse(s)
	if err != nil { // date_format passed, so this cannot happen; never a 401 or a 500 for bad input
		return httpx.Fail(fiber.StatusUnprocessableEntity, t(CodePredictionMismatch, locale), "code", CodePredictionMismatch)
	}
	created, err := h.svc.Record(c.Context(), uid, expected, civildate.LocalToday(c, nw))
	var r *Refusal
	if errors.As(err, &r) {
		return httpx.Fail(fiber.StatusUnprocessableEntity, t(r.Code, locale), "code", r.Code)
	}
	if err != nil {
		return err
	}
	data := jsonx.Obj("expected_start", expected)
	if created {
		return httpx.Created(c, data)
	}
	return httpx.OK(c, data)
}

// Destroy is DELETE /cycle/skips/{date}: 204, or 404 when the user has no skip on that date (also for a date that
// is not Y-m-d, and for another user's skip).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.ResolveLocale(c, "")
	date, err := civildate.Parse(c.Params("date"))
	if err != nil {
		return httpx.Fail(fiber.StatusNotFound, t("not_found", locale))
	}
	if err := h.svc.Delete(c.Context(), uid, date); err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpx.Fail(fiber.StatusNotFound, t("not_found", locale))
		}
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// The refusal copy is data: lang/<code>/skips.json (English fallback).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	tr, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return tr
})

func t(key, locale string) string { return translator().Trans("skips."+key, nil, locale) }
