package consent

import (
	"encoding/json"
	"errors"
	"strconv"
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

// Error codes.
const (
	CodeConsentRequired = "consent_required"      // 403: the feature needs the text in force accepted
	CodeVersionStale    = "consent_version_stale" // 409: PUT accepted a version that is no longer in force
	CodeNotFound        = "consent_not_found"     // 404: unknown consent code
)

// Required is the 403 response for a feature whose consent is missing or outdated:
//
//	{success:false, message, error_code:"consent_required", consent:"<code>", version:<n>, reason:"missing"|"outdated"}
//
// The client fetches GET /consents/{code} (the text of `version`), shows it and accepts with PUT.
func Required(e *RequiredError, locale string) error {
	key := "consent_required"
	if e.Reason == ReasonOutdated {
		key = "consent_outdated"
	}
	return httpx.Fail(fiber.StatusForbidden, T(key, locale),
		"error_code", CodeConsentRequired, "consent", e.Code, "version", e.Version, "reason", e.Reason)
}

// Handlers serve GET /consents, GET /consents/{code} and PUT /consents/{code} (Go only, auth:api, localized).
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (X-Test-Now pins it per request in tests).
func NewHandlers(svc *Service, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{svc: svc, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func userID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// toInt is a validated `integer` input (JSON number or numeric string) as an int; anything else is 0.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(n))
		return i
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

// stateJSON is one consent: {code, version, title, body, points, granted, accepted_version, needs_consent, reason,
// granted_at, revoked_at}. `granted` is true only while the text in force is accepted.
func stateJSON(st State, locale string) *jsonx.OrderedMap {
	text, _ := TextOf(st.Code, st.Current, locale)
	points := text.Points
	if points == nil {
		points = []string{}
	}
	var accepted, grantedAt, revokedAt, reason any
	if st.Version > 0 {
		accepted = st.Version
	}
	if st.GrantedAt.Valid {
		grantedAt = jsonx.ISO8601(st.GrantedAt.Time)
	}
	if st.RevokedAt.Valid {
		revokedAt = jsonx.ISO8601(st.RevokedAt.Time)
	}
	if r := st.Reason(); r != "" {
		reason = r
	}
	return jsonx.Obj(
		"code", st.Code,
		"version", st.Current,
		"title", text.Title,
		"body", text.Body,
		"points", points,
		"granted", st.Valid(),
		"accepted_version", accepted,
		"needs_consent", !st.Valid(),
		"reason", reason,
		"granted_at", grantedAt,
		"revoked_at", revokedAt,
	)
}

// List is GET /consents: every consent with its text in force, in display order.
func (h *Handlers) List(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	states, err := h.svc.List(c, id)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	list := make([]any, len(states))
	for i, st := range states {
		list[i] = stateJSON(st, locale)
	}
	return httpx.OK(c, jsonx.Obj("consents", list))
}

func (h *Handlers) state(c fiber.Ctx, id uint64) (State, error) {
	st, err := h.svc.Get(c, id, c.Params("code"))
	if errors.Is(err, ErrUnknown) {
		return State{}, httpx.Fail(fiber.StatusNotFound, T("not_found", i18n.Locale(c)), "error_code", CodeNotFound)
	}
	return st, err
}

// Show is GET /consents/{code}.
func (h *Handlers) Show(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	st, err := h.state(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("consent", stateJSON(st, i18n.Locale(c))))
}

// Update is PUT /consents/{code} {granted: bool, version: int (required when granting)}: accept the text in force
// (409 consent_version_stale for any other version) or withdraw; then the consent.
func (h *Handlers) Update(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if _, err := h.state(c, id); err != nil {
		return err
	}
	code := c.Params("code")
	now := h.now(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("granted", "required", "boolean"),
		validation.F("version", "required_if:granted,true,1", "nullable", "integer", "min:1", "max:1000"),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("validation_failed", locale), "errors", v.ErrorBag())
	}
	granted, _ := phpval.Get(body, "granted")
	msg := "withdrawn"
	if phpval.Truthy(granted) {
		raw, _ := phpval.Get(body, "version")
		err = h.svc.Accept(c, id, code, toInt(raw), now)
		if errors.Is(err, ErrStaleVersion) {
			return httpx.Fail(fiber.StatusConflict, T("version_stale", locale),
				"error_code", CodeVersionStale, "consent", code, "version", CurrentVersion(code))
		}
		msg = "saved"
	} else {
		err = h.svc.Withdraw(c, id, code, now)
	}
	if err != nil {
		return err
	}
	st, err := h.svc.Get(c, id, code)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("consent", stateJSON(st, locale)), T(msg, locale))
}
