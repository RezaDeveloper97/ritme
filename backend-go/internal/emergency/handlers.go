package emergency

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the emergency card actions: owner routes under /api/v1/health-record/emergency-card (locale, auth
// RequireUser, writes throttled) and the public GET /api/v1/emergency-cards/{token} (locale, IP throttle, no auth).
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

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

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func (h *Handlers) owner(c fiber.Ctx, userID uint64, now time.Time) (*jsonx.OrderedMap, error) {
	card, err := h.svc.Build(c, userID, civildate.InTehran(now), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return nil, err
	}
	return card.OwnerJSON(), nil
}

// Show is GET /health-record/emergency-card (nbl_Rec_Emergency): the card as the owner sees it, its settings and the
// public link state.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	body, err := h.owner(c, userID, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

var (
	rePhone = regexp.MustCompile(`^\+?[0-9]{4,15}$`)
	reLast4 = regexp.MustCompile(`^[0-9]{4}$`)
)

// asciiDigits turns Persian / Arabic digits into ASCII and drops spaces and dashes.
func asciiDigits(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == ' ' || r == '-':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func text(m phpval.Map, key string) string {
	v, ok := m.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.Join(strings.Fields(phpval.ToString(v)), " ")
}

// parseInput validates PUT {show_on_lock_screen?, show_pregnancy?, emergency_contact?: {name, relation?, phone}|null,
// insurance?: {label?, last4}|null}: a present key replaces its part (null clears), an absent key keeps it.
func parseInput(body phpval.Map, locale string, now time.Time) (Input, error) {
	data := phpval.NewMap()
	for _, k := range []string{"show_on_lock_screen", "show_pregnancy", "emergency_contact", "insurance"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	rules := validation.Rules{
		validation.F("show_on_lock_screen", "sometimes", "boolean"),
		validation.F("show_pregnancy", "sometimes", "boolean"),
		validation.F("emergency_contact", "nullable", "array"),
		validation.F("insurance", "nullable", "array"),
	}
	if v, _ := data.Get("emergency_contact"); v != nil {
		rules = append(rules,
			validation.F("emergency_contact.name", "required", "string", "max:60"),
			validation.F("emergency_contact.relation", "nullable", "string", "max:30"),
			validation.F("emergency_contact.phone", "required", "string", "max:20"))
	}
	if v, _ := data.Get("insurance"); v != nil {
		rules = append(rules,
			validation.F("insurance.label", "nullable", "string", "max:60"),
			validation.F("insurance.last4", "required", "string", "max:32")) // only 4 digits pass the check below
	}
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now),
		validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return Input{}, failValidation(locale, v.ErrorBag())
	}
	var in Input
	if x, ok := data.Get("show_on_lock_screen"); ok {
		in.SetLockScreen, in.LockScreen = true, phpval.Truthy(x)
	}
	if x, ok := data.Get("show_pregnancy"); ok {
		in.SetPregnancy, in.Pregnancy = true, phpval.Truthy(x)
	}
	if x, ok := data.Get("emergency_contact"); ok {
		in.SetContact = true
		if m, isMap := x.(phpval.Map); isMap {
			c := Contact{Name: text(m, "name"), Relation: text(m, "relation"), Phone: asciiDigits(text(m, "phone"))}
			if !rePhone.MatchString(c.Phone) {
				return Input{}, failValidation(locale, jsonx.Obj("emergency_contact.phone", []string{T("validation.phone", locale)}))
			}
			in.Contact = &c
		}
	}
	if x, ok := data.Get("insurance"); ok {
		in.SetInsurance = true
		if m, isMap := x.(phpval.Map); isMap {
			i := Insurance{Label: text(m, "label"), Last4: asciiDigits(text(m, "last4"))}
			if !reLast4.MatchString(i.Last4) {
				return Input{}, failValidation(locale, jsonx.Obj("insurance.last4", []string{T("validation.last4", locale)}))
			}
			in.Insurance = &i
		}
	}
	return in, nil
}

// Update is PUT /health-record/emergency-card: saves the settings and returns the card.
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := parseInput(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Save(c, userID, in, now); err != nil {
		return err
	}
	body, err := h.owner(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, T("messages.saved", locale))
}

// EnablePublic is POST /health-record/emergency-card/public-link → 201 {token, path, card…}: a new public link (the
// previous one stops working). The token is returned only here.
func (h *Handlers) EnablePublic(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	token, err := h.svc.EnablePublic(c, userID, now)
	if err != nil {
		return err
	}
	body, err := h.owner(c, userID, now)
	if err != nil {
		return err
	}
	body.Set("token", token)
	body.Set("path", "/"+locale+"/emergency/"+token)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.Created(c, body, T("messages.public_enabled", locale))
}

// DisablePublic is DELETE /health-record/emergency-card/public-link: the public link stops working now.
func (h *Handlers) DisablePublic(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	if err := h.svc.DisablePublic(c, userID, now); err != nil {
		return err
	}
	body, err := h.owner(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, T("messages.public_disabled", locale))
}

// Lock is GET /health-record/emergency-card/lock (CB-PRIV-01, D-73), what the app-lock screen reads: {enabled:false}
// with no data while the lock-screen flag is off, else {enabled:true, card} with the minimal (public) card. Not
// cached.
func (h *Handlers) Lock(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	now := h.now(c)
	card, err := h.svc.Lock(c, userID, civildate.InTehran(now), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	if card == nil {
		return httpx.OK(c, jsonx.Obj("enabled", false))
	}
	return httpx.OK(c, jsonx.Obj("enabled", true, "card", card))
}

// Public is GET /emergency-cards/{token}: the minimal card (404 for an unknown or disabled token). Never cached or
// indexed.
func (h *Handlers) Public(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
	card, err := h.svc.Public(c, c.Params("token"), civildate.InTehran(now), now, locale, i18n.LanguagesOf(c).DefaultCode())
	if errors.Is(err, ErrNotFound) {
		return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", CodeNotFound)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, card)
}
