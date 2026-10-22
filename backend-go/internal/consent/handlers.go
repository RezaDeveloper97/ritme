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

// Stale is the 409 of a grant naming a version that is not the one in force:
//
//	{success:false, message, error_code:"consent_version_stale", consent:"<code>", version:<in force>}
func Stale(code, locale string) error {
	return httpx.Fail(fiber.StatusConflict, T("version_stale", locale),
		"error_code", CodeVersionStale, "consent", code, "version", CurrentVersion(code))
}

// ToInt is a validated `integer` input as an int (see toInt).
func ToInt(v any) int { return toInt(v) }

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
	// A feature's consent sheet is source feature; the mandatory signup consents are accepted here from the one-time
	// pending sheet (or withdrawn) — source settings.
	source := SourceFeature
	if d, _ := Lookup(code); d.Mandatory {
		source = SourceSettings
	}
	msg := "withdrawn"
	if phpval.Truthy(granted) {
		raw, _ := phpval.Get(body, "version")
		err = h.svc.AcceptFrom(c, id, code, toInt(raw), source, now)
		if errors.Is(err, ErrStaleVersion) {
			return Stale(code, locale)
		}
		msg = "saved"
	} else {
		err = h.svc.WithdrawFrom(c, id, code, source, now)
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

// Pending is GET /consents/pending: the mandatory consents not accepted at the version in force, each with its text
// ({consents: [...]}, display order; empty once accepted). Accept each with PUT /consents/{code}. Nothing is blocked
// while it is non-empty (H-18).
func (h *Handlers) Pending(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	states, err := h.svc.Pending(c, id)
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

// eventJSON is one ledger row: {id, consent, title, version, action, source, scope, created_at}. The title is the
// one of the version the event names (the text in force when that version has none, e.g. a backfilled version 0).
func eventJSON(e Event, locale string) *jsonx.OrderedMap {
	text, ok := TextOf(e.Consent, e.Version, locale)
	if !ok {
		v := CurrentVersion(e.Consent)
		if d, ev := lookupEvidence(e.Consent); ev {
			v = d.Version
		}
		text, _ = TextOf(e.Consent, v, locale)
	}
	var title, scope, at any
	if text.Title != "" {
		title = text.Title
	}
	if e.Scope != nil {
		scope = e.Scope
	}
	if e.At.Valid {
		at = jsonx.ISO8601(e.At.Time)
	}
	return jsonx.Obj(
		"id", e.ID,
		"consent", e.Consent,
		"title", title,
		"version", e.Version,
		"action", e.Action,
		"source", e.Source,
		"scope", scope,
		"created_at", at,
	)
}

// History is GET /consents/history: the user's own ledger, newest first ({events: [...]}, at most HistoryLimit).
func (h *Handlers) History(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	events, err := h.svc.History(c, id)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	list := make([]any, len(events))
	for i, e := range events {
		list[i] = eventJSON(e, locale)
	}
	return httpx.OK(c, jsonx.Obj("events", list))
}
