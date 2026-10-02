package teen

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ErrorCodeProfileRequired is the 422 of PUT /teen/parent-note before onboarding.
const ErrorCodeProfileRequired = "teen_profile_required"

// Handlers are the /api/v1/teen actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func (h *Handlers) clk(c fiber.Ctx) clock.Clock { return clock.FromContext(c, h.clock) }

func localizer(c fiber.Ctx) catalog.Localizer {
	langs := i18n.LanguagesOf(c)
	return catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func (h *Handlers) validate(c fiber.Ctx, rules validation.Rules, opts ...validation.Option) (phpval.Map, error) {
	locale := i18n.Locale(c)
	body := validation.Input(c)
	opts = append([]validation.Option{validation.Now(h.clk(c).Now()), validation.Attributes(attributes(locale)...)}, opts...)
	v := validation.Make(lang.Default(), locale, body, rules, opts...)
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	return body, nil
}

func (h *Handlers) profileOK(c fiber.Ctx, userID uint64, p *Profile, msg ...string) error {
	mode, err := h.svc.LifeMode(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, ProfileEnvelope(p, AllowsFor(mode), isTeen(mode)), msg...)
}

// ShowProfile is GET /teen/profile: her onboarding answers (null before), teen-mode flag and the commercial flags.
func (h *Handlers) ShowProfile(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	p, err := h.svc.Profile(c, userID)
	if err != nil {
		return err
	}
	return h.profileOK(c, userID, p)
}

// SaveProfile is PUT /teen/profile {age_band, menarche} (Teen_Onb).
func (h *Handlers) SaveProfile(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	body, err := h.validate(c, validation.Rules{
		validation.F("age_band", "required", "string", validation.In(strs(AgeBands)...)),
		validation.F("menarche", "required", "string", validation.In(strs(Menarches)...)),
	}, validation.Messages("age_band.in", T("validation.age_band_invalid", locale),
		"menarche.in", T("validation.menarche_invalid", locale)))
	if err != nil {
		return err
	}
	p, err := h.svc.SaveProfile(c, userID, AgeBand(str(body, "age_band")), Menarche(str(body, "menarche")), h.clk(c).Now())
	if err != nil {
		return err
	}
	return h.profileOK(c, userID, p, T("messages.profile_saved", locale))
}

// UpdateParentNote is PUT /teen/parent-note {note: string|null}: the note her parent sees while she grants
// teen_notes. 422 teen_profile_required before onboarding.
func (h *Handlers) UpdateParentNote(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	body, err := h.validate(c, validation.Rules{
		validation.F("note", "present", "nullable", "string", "max:"+strconv.Itoa(MaxParentNote)),
	})
	if err != nil {
		return err
	}
	p, err := h.svc.SetParentNote(c, userID, str(body, "note"), h.clk(c).Now())
	if errors.Is(err, ErrProfileRequired) {
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.profile_required", locale),
			"error_code", ErrorCodeProfileRequired)
	}
	if err != nil {
		return err
	}
	return h.profileOK(c, userID, p, T("messages.note_saved", locale))
}

// Today is GET /teen/today: the Teen_Home read model.
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	t, err := h.svc.Today(c, userID, h.clk(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, TodayJSON(t, localizer(c)))
}

// UpdateKitItem is PUT /teen/kit/{code} {checked: bool}: tick or untick one school-kit item. 404 for a code that is
// not an active teen_kit_items item.
func (h *Handlers) UpdateKitItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	code := c.Params("code")
	if !catalog.ValidCode(code, catalog.MaxCodeLen) {
		return httpx.Fail(fiber.StatusNotFound, T("messages.kit_item_not_found", locale))
	}
	body, err := h.validate(c, validation.Rules{validation.F("checked", "required", "boolean")})
	if err != nil {
		return err
	}
	k, err := h.svc.SetKitItem(c, userID, code, boolOf(body), h.clk(c).Now())
	if errors.Is(err, ErrUnknownKitItem) {
		return httpx.Fail(fiber.StatusNotFound, T("messages.kit_item_not_found", locale))
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, KitJSON(k, localizer(c)), T("messages.kit_saved", locale))
}

// Linked is GET /teen/linked: the read-only cards of the teens who shared with the caller as their parent. Only the
// granted parts are filled (each read audited first); a parent can never write. [] when the caller is nobody's parent.
func (h *Handlers) Linked(c fiber.Ctx) error {
	viewerID, err := h.user(c)
	if err != nil {
		return err
	}
	cards, err := h.svc.ParentCards(c, viewerID, h.clk(c))
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(cards))
	for _, card := range cards {
		out = append(out, CardJSON(card))
	}
	return httpx.OK(c, out)
}

func strs[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

func str(m phpval.Map, key string) string {
	v, ok := m.Get(key)
	if !ok || v == nil {
		return ""
	}
	return phpval.ToString(v)
}

func boolOf(m phpval.Map) bool {
	v, _ := m.Get("checked")
	if b, ok := v.(bool); ok {
		return b
	}
	s := strings.ToLower(phpval.ToString(v))
	return s == "1" || s == "true"
}

func isTeen(mode enums.LifeMode) bool { return mode == enums.LifeModeTeen }
