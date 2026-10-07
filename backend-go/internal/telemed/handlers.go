package telemed

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Error codes of the 403 / 404 / 409 bodies.
const (
	ErrorCodeDoctorNotFound   = "doctor_not_found"
	ErrorCodeReviewNotFound   = "review_not_found"
	ErrorCodeReviewNotAllowed = "review_not_allowed"
	ErrorCodeReviewExists     = "review_exists"
)

// Handlers are the /api/v1/telemed actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc    *Service
	clock  clock.Clock
	appURL string
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request);
// appURL builds public photo URLs.
func NewHandlers(svc *Service, base clock.Clock, appURL string) *Handlers {
	return &Handlers{svc: svc, clock: base, appURL: appURL}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	s := c.Params(name)
	if s == "" || len(s) > 20 || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	id, err := strconv.ParseUint(s, 10, 64)
	return id, err == nil && id > 0
}

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.doctor_not_found", locale), "error_code", ErrorCodeDoctorNotFound)
}

// fail maps the service errors to their bodies.
func fail(err error, locale string) error {
	switch {
	case errors.Is(err, ErrDoctorNotFound):
		return notFound(locale)
	case errors.Is(err, ErrReviewNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.review_not_found", locale), "error_code", ErrorCodeReviewNotFound)
	case errors.Is(err, ErrReviewNotAllowed):
		return httpx.Fail(fiber.StatusForbidden, T("messages.review_not_allowed", locale), "error_code",
			ErrorCodeReviewNotAllowed)
	case errors.Is(err, ErrReviewExists):
		return httpx.Fail(fiber.StatusConflict, T("messages.review_exists", locale), "error_code", ErrorCodeReviewExists)
	case errors.Is(err, ErrModeNotOffered):
		msg := lang.Default().Trans("validation.in", map[string]string{"attribute": T("attributes.mode", locale)}, locale)
		return failValidation(locale, jsonx.Obj("mode", []string{msg}))
	}
	return err
}

func (h *Handlers) view(c fiber.Ctx) (View, error) {
	labels, err := h.svc.LoadLabels(c)
	if err != nil {
		return View{}, err
	}
	langs := i18n.LanguagesOf(c)
	return View{
		Loc:    catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs},
		Labels: labels, AppURL: h.appURL,
	}, nil
}

// List is GET /telemed/doctors (nbl_v17_Doctors): the listed doctors matching the filters in display order, paged by
// PerPage — {items, meta}. The «امروز آزاد» filter keeps doctors with a free slot today (in the filtered mode).
func (h *Handlers) List(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	f, err := ValidateFilter(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	cards, err := h.svc.Directory(c, f, now)
	if err != nil {
		return err
	}
	v, err := h.view(c)
	if err != nil {
		return err
	}
	page := httpx.NewPage([]any{}, len(cards), PerPage, httpx.PageParam(c), httpx.RequestURL(c))
	from := min(page.Offset(), len(cards))
	to := min(from+PerPage, len(cards))
	items := make([]any, 0, to-from)
	for _, card := range cards[from:to] {
		items = append(items, v.CardJSON(card, f.Mode))
	}
	return httpx.OK(c, jsonx.Obj("items", jsonx.List(items), "meta", page.Meta()))
}

// option is one filter chip: {code, title, count}.
func (v View) option(group, code string, count int64) *jsonx.OrderedMap {
	o := v.code(group, code)
	o.Set("count", count)
	return o
}

// Filters is GET /telemed/doctors/filters: the chips of the directory — modes and kinds (client copy), and the catalog
// specialties / cities / insurers with the number of listed doctors (catalog order; codes used by doctors but missing
// from the catalog come last with a null title).
func (h *Handlers) Filters(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	v, err := h.view(c)
	if err != nil {
		return err
	}
	counts, err := h.svc.Counts(c)
	if err != nil {
		return err
	}
	group := func(g string, n map[string]int64) ([]any, error) {
		items, err := h.svc.Catalog().Items(c, g)
		if err != nil {
			return nil, err
		}
		out := []any{}
		seen := map[string]bool{}
		for _, it := range items {
			seen[it.Code] = true
			out = append(out, v.option(g, it.Code, n[it.Code]))
		}
		for _, code := range sortedKeys(n) {
			if !seen[code] {
				out = append(out, v.option(g, code, n[code]))
			}
		}
		return out, nil
	}
	total := int64(0)
	for _, n := range counts.Specialties {
		total += n
	}
	body := jsonx.Obj("total", total, "modes", jsonx.List(Modes), "kinds", jsonx.List(Kinds))
	for _, g := range []struct {
		key, group string
		counts     map[string]int64
	}{
		{"specialties", GroupSpecialties, counts.Specialties},
		{"cities", GroupCities, counts.Cities},
		{"insurers", GroupInsurers, counts.Insurers},
	} {
		opts, err := group(g.group, g.counts)
		if err != nil {
			return err
		}
		body.Set(g.key, opts)
	}
	return httpx.OK(c, body)
}

// Show is GET /telemed/doctors/{id} (nbl_v17_DoctorProfile).
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return notFound(locale)
	}
	card, err := h.svc.Doctor(c, id, now)
	if err != nil {
		return fail(err, locale)
	}
	reviews, _, err := h.svc.Reviews(c, id, PreviewReviews, 0)
	if err != nil {
		return err
	}
	canReview, err := h.svc.CanReview(c, userID, id)
	if err != nil {
		return err
	}
	v, err := h.view(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("doctor", v.ProfileJSON(card, reviews, userID, canReview)))
}

// Slots is GET /telemed/doctors/{id}/slots?mode=&from=&days=.
func (h *Handlers) Slots(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return notFound(locale)
	}
	in, err := ValidateSlots(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := h.svc.Slots(c, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	v, err := h.view(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, v.SlotsJSON(res, in))
}

// Reviews is GET /telemed/doctors/{id}/reviews?page=: visible reviews, newest first — {items, meta}.
func (h *Handlers) Reviews(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return notFound(locale)
	}
	if err := h.svc.Listed(c, id); err != nil {
		return fail(err, locale)
	}
	page := httpx.NewPage([]any{}, 0, ReviewsPerPage, httpx.PageParam(c), httpx.RequestURL(c))
	reviews, total, err := h.svc.Reviews(c, id, ReviewsPerPage, page.Offset())
	if err != nil {
		return err
	}
	page.Total = total
	items := make([]any, 0, len(reviews))
	for _, r := range reviews {
		items = append(items, ReviewJSON(r, userID))
	}
	return httpx.OK(c, jsonx.Obj("items", jsonx.List(items), "meta", page.Meta()))
}

// StoreReview is POST /telemed/doctors/{id}/reviews {rating, body?} (201 {review}): only for a completed visit not
// yet reviewed (403 review_not_allowed; a second review of the same visit is 409 review_exists).
func (h *Handlers) StoreReview(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return notFound(locale)
	}
	in, err := ValidateReview(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	r, err := h.svc.CreateReview(c, userID, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("review", ReviewJSON(r, userID)), T("messages.review_saved", locale))
}

// DestroyReview is DELETE /telemed/doctors/{id}/reviews/{review}: the caller's own review only (anything else is a
// uniform 404 review_not_found).
func (h *Handlers) DestroyReview(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	reviewID, ok2 := idParam(c, "review")
	if !ok || !ok2 {
		return fail(ErrReviewNotFound, locale)
	}
	if err := h.svc.DeleteReview(c, userID, id, reviewID); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("id", reviewID), T("messages.review_deleted", locale))
}
