package invites

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Error codes of the controller-style failures (the /companions codes where the meaning is the same).
const (
	ErrorCodeInviteInvalid   = "invite_invalid"
	ErrorCodeUnavailable     = "companion_unavailable"
	ErrorCodeLimitReached    = "limit_reached"
	ErrorCodeTeenParentOnly  = "teen_parent_only"
	ErrorCodeMutualExists    = "mutual_exists"
	ErrorCodeTooManyAttempts = "too_many_requests"
)

// CacheControlNoStore: invite answers are private (a preview is the owner's data).
const CacheControlNoStore = "private, no-store"

// Breaker is the global failed-code circuit breaker shared with /companions/accept (CMP-M3; internal/http implements
// it on Redis): preview, accept and decline refusals all count, and an open breaker pauses all three.
type Breaker interface {
	Open(ctx context.Context) (open bool, retryAfter int, err error)
	Failed(ctx context.Context) (tripped bool, err error)
}

// Sender texts an invite code to a phone (the /companions SMS adapter and its per-recipient / per-owner gates); it
// reports whether an SMS went out.
type Sender func(c fiber.Ctx, ownerID, linkID uint64, phone, code string) bool

// HandlerOptions wires the handlers.
type HandlerOptions struct {
	Service Options
	// Disabled answers 503 on every code route (a pepper missing outside local / testing / contract; the server also
	// refuses to start, this is the second line).
	Disabled bool
	Breaker  Breaker
	Send     Sender
	Logger   *slog.Logger
}

// Handlers serve /api/v1/family/members and /api/v1/family/invites/* (Go only, D-196).
type Handlers struct {
	conn   Conn
	base   clock.Clock
	opt    HandlerOptions
	logger *slog.Logger
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(conn Conn, base clock.Clock, opt HandlerOptions) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	h := &Handlers{conn: conn, base: base, opt: opt, logger: opt.Logger}
	if h.logger == nil {
		h.logger = slog.Default()
	}
	return h
}

func (h *Handlers) svc(c fiber.Ctx) *Service {
	return NewService(h.conn, clock.FromContext(c, h.base), h.opt.Service)
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.base).Now().In(civildate.Tehran).Truncate(time.Second)
}

func currentUser(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func paramID(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

// ---------------------------------------------------------------- owner

// Store is POST /family/members {role, template, grants?, children?, expires_at?, urgent_alert?, urgent_sms?,
// mutual_request?: {domains}, phone?, display_name?, consent_version}: an invited link with its grants and one invite
// (code 24 h, phone 7 days, also texted when the SMS adapter delivers). 201.
func (h *Handlers) Store(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if h.opt.Disabled {
		return unavailable(locale)
	}
	now := h.now(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("role", "required", "string", validation.In(roleCodes()...)),
		validation.F("template", "required", "string", validation.In(templateCodes()...)),
		validation.F("grants", "nullable", "array"),
		validation.F("children", "nullable", "string", validation.In(family.LevelOff.String(), family.LevelWrite.String())),
		validation.F("expires_at", "nullable", "date_format:Y-m-d"),
		validation.F("urgent_alert", "nullable", "boolean"),
		validation.F("urgent_sms", "nullable", "boolean"),
		validation.F("mutual_request", "nullable", "array"),
		validation.F("mutual_request.domains", "nullable", "array", "max:2"),
		validation.F("mutual_request.domains.*", "string", validation.In(mutualCodes()...)),
		validation.F("phone", "nullable", "string", "max:20"),
		validation.F("display_name", "nullable", "string", "max:100"),
		validation.F("consent_version", "required", "integer"),
	}, validation.Now(now),
		validation.Messages("role.in", T("validation.role_invalid", locale), "template.in", T("validation.template_invalid", locale),
			"mutual_request.domains.*.in", T("validation.mutual_invalid", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	role := family.ParseRole(str(body, "role"), "")
	in := CreateInput{
		Role: role, Template: family.Template(str(body, "template")), Children: str(body, "children"),
		Phone: str(body, "phone"), DisplayName: str(body, "display_name"),
		UrgentSMS: truthy(body, "urgent_sms"),
	}
	in.ConsentVersion, _ = strconv.Atoi(str(body, "consent_version"))
	if raw, ok := body.Get("urgent_alert"); ok && raw != nil {
		on := phpval.Truthy(raw)
		in.UrgentAlert = &on
	}
	if d := str(body, "expires_at"); d != "" {
		day, err := civildate.Parse(d)
		if err != nil {
			return fieldError(locale, "expires_at", T("validation.expires_invalid", locale))
		}
		in.ExpiresAt = day.AddDays(1).TehranMidnight() // the whole picked day is included
	}
	if in.Edits, in.Children, err = parseGrants(locale, body, in.Children); err != nil {
		return err
	}
	if raw, ok := phpval.Get(body, "mutual_request.domains"); ok && raw != nil {
		_, vals := phpval.Entries(raw)
		for _, val := range vals {
			if d, ok := family.ParseDomain(phpval.ToString(val)); ok {
				in.Mutual = append(in.Mutual, d)
			}
		}
	}
	svc := h.svc(c)
	created, err := svc.Create(c.Context(), ownerID, in)
	if err != nil {
		return mapError(locale, err)
	}
	sent := false
	if created.Phone != "" && h.opt.Send != nil {
		sent = h.opt.Send(c, ownerID, created.LinkID, created.Phone, created.Code)
	}
	return httpx.Created(c, jsonx.Obj("link", createdLinkJSON(created), "invite", inviteJSON(created, sent)),
		T("messages.invite_created", locale))
}

// Confirm is POST /family/members/{linkId}/confirm: the owner confirms an accepted high-impact code invite (F-11).
func (h *Handlers) Confirm(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := paramID(c, "linkId")
	if !ok {
		return notFoundErr(locale)
	}
	l, err := h.svc(c).Confirm(c.Context(), ownerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, jsonx.Obj("id", l.ID, "status", l.Status, "owner_confirmed_at", isoTime(l.OwnerConfirmedAt.Time)),
		T("messages.confirmed", locale))
}

// Cancel is POST /family/members/{linkId}/cancel: the owner withdraws an invited or awaiting link (any time).
func (h *Handlers) Cancel(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := paramID(c, "linkId")
	if !ok {
		return notFoundErr(locale)
	}
	if err := h.svc(c).Cancel(c.Context(), ownerID, id); err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, jsonx.Obj("id", id, "status", StatusRevoked), T("messages.cancelled", locale))
}

// ---------------------------------------------------------------- member

// codeInput validates {code} or {link_id} (a pending acceptance) plus extra rules; refusals are generic.
func (h *Handlers) codeInput(c fiber.Ctx, locale string, extra ...validation.Field) (phpval.Map, string, uint64, error) {
	body := validation.Input(c)
	rules := validation.Rules{
		validation.F("code", "required_without:link_id", "nullable", "string", "max:32"),
		validation.F("link_id", "nullable", "integer", "min:1"),
	}
	rules = append(rules, extra...)
	v := validation.Make(lang.Default(), locale, body, rules, validation.Now(h.now(c)),
		validation.Messages("code.required_without", T("validation.code_required", locale)))
	if v.Fails() {
		return nil, "", 0, failValidation(locale, v.ErrorBag())
	}
	code := str(body, "code")
	var link uint64
	if code == "" {
		link, _ = strconv.ParseUint(str(body, "link_id"), 10, 64)
	}
	return body, code, link, nil
}

// gate is the shared preamble of the code routes: disabled → 503, breaker open → 429.
func (h *Handlers) gate(c fiber.Ctx, locale string) error {
	if h.opt.Disabled {
		return unavailable(locale)
	}
	if h.opt.Breaker == nil {
		return nil
	}
	open, retry, err := h.opt.Breaker.Open(c.Context())
	if err != nil {
		return err // fail closed
	}
	if open {
		return TooManyAttempts(locale, retry, map[string]string{fiber.HeaderRetryAfter: strconv.Itoa(retry)})
	}
	return nil
}

// refuse answers a refusal with the generic invite_invalid 422: every code refusal (counted by the breaker for code
// attempts) and the owner's mode refusal (CB-TEEN-01), which would otherwise tell a valid code from a wrong one
// (L2; a valid code is not a failed guess, so it is not counted). Anything else is mapped.
func (h *Handlers) refuse(c fiber.Ctx, locale string, err error, byCode bool) error {
	notAllowed := errors.Is(err, ErrNotAllowed)
	if !notAllowed && !IsRefusal(err) {
		return mapError(locale, err)
	}
	if !notAllowed && byCode && h.opt.Breaker != nil {
		if tripped, berr := h.opt.Breaker.Failed(c.Context()); berr != nil {
			h.logger.WarnContext(c.Context(), "family invite breaker count failed", slog.String("error", berr.Error()))
		} else if tripped {
			h.logger.ErrorContext(c.Context(), "companion accept circuit breaker tripped: accepts paused for everyone")
		}
	}
	msg := T("messages.invite_invalid", locale)
	return httpx.Fail(fiber.StatusUnprocessableEntity, msg, "errors", jsonx.Obj("code", []string{msg}),
		"error_code", ErrorCodeInviteInvalid)
}

// Preview is POST /family/invites/preview {code}: what accepting would share, with the throttles, breaker and generic
// refusal of accept (a preview is another code oracle, FAM-L3).
func (h *Handlers) Preview(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.gate(c, locale); err != nil {
		return err
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("code", "required", "string", "max:32"),
	}, validation.Now(h.now(c)), validation.Messages("code.required", T("validation.code_required", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	p, err := h.svc(c).Preview(c.Context(), viewerID, str(body, "code"))
	if err != nil {
		return h.refuse(c, locale, err, true)
	}
	return httpx.OK(c, previewJSON(p), T("messages.preview", locale))
}

// Pending is GET /family/invites/pending: the caller's phone invites bound to her account (pending acceptances), each
// answered with {link_id} on accept / decline, no code needed.
func (h *Handlers) Pending(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	list, err := h.svc(c).Pending(c.Context(), viewerID)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, p := range list {
		out = append(out, previewJSON(p))
	}
	return httpx.OK(c, out)
}

// Accept is POST /family/invites/accept {code | link_id, mutual?: {domain: level}}: active, or awaiting_owner when a
// code-only invite includes child write, the urgent alert or vitals (F-11). A mutual answer creates an independent
// reverse link with the caller's grants and consent; accepting without mutual is allowed.
func (h *Handlers) Accept(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.gate(c, locale); err != nil {
		return err
	}
	body, code, link, err := h.codeInput(c, locale, validation.F("mutual", "nullable", "array"))
	if err != nil {
		return err
	}
	in := AcceptInput{Code: code, LinkID: link}
	if raw, ok := body.Get("mutual"); ok && raw != nil {
		in.Mutual = map[family.Domain]family.Level{}
		keys, vals := phpval.Entries(raw)
		for i, k := range keys {
			d, okD := family.ParseDomain(k)
			l, okL := family.ParseLevel(phpval.ToString(vals[i]))
			if !okD {
				return fieldError(locale, "mutual."+k, T("validation.domain_invalid", locale))
			}
			if !okL {
				return fieldError(locale, "mutual."+k, T("validation.level_invalid", locale))
			}
			in.Mutual[d] = l
		}
	}
	a, err := h.svc(c).Accept(c.Context(), viewerID, in)
	if err != nil {
		return h.refuse(c, locale, err, code != "")
	}
	if err := h.svc(c).NotifyOwner(c.Context(), a, i18n.LanguagesOf(c).Codes()); err != nil {
		h.logger.ErrorContext(c.Context(), "family invite accept notice failed", slog.Uint64("link_id", a.LinkID),
			slog.String("error", err.Error()))
	}
	msg := T("messages.accepted", locale)
	if a.Status == StatusAwaitingOwner {
		msg = T("messages.awaiting_owner", locale)
	}
	var mutual any
	if a.MutualLinkID != 0 {
		mutual = jsonx.Obj("id", a.MutualLinkID, "status", StatusActive)
	}
	return httpx.OK(c, jsonx.Obj("link", jsonx.Obj("id", a.LinkID, "status", a.Status), "mutual_link", mutual), msg)
}

// Decline is POST /family/invites/decline {code | link_id}: revoked with revoked_by = declined (F-12).
func (h *Handlers) Decline(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.gate(c, locale); err != nil {
		return err
	}
	_, code, link, err := h.codeInput(c, locale)
	if err != nil {
		return err
	}
	if err := h.svc(c).Decline(c.Context(), viewerID, code, link); err != nil {
		return h.refuse(c, locale, err, code != "")
	}
	return httpx.OK(c, nil, T("messages.declined", locale))
}

// ---------------------------------------------------------------- helpers

func roleCodes() []string {
	out := make([]string, len(Roles))
	for i, r := range Roles {
		out[i] = r.String()
	}
	return out
}

func templateCodes() []string {
	out := make([]string, len(Templates))
	for i, t := range Templates {
		out[i] = string(t)
	}
	return out
}

func mutualCodes() []string {
	out := make([]string, len(MutualDomains))
	for i, d := range MutualDomains {
		out[i] = d.String()
	}
	return out
}

// parseGrants reads {grants: {domain: off|view|nudge|write, children: off|write}}. Unknown domains and levels are a
// 422 on grants.<key>; levels above the registry's caps are refused later by Grants.Apply (level_not_allowed).
func parseGrants(locale string, body phpval.Map, children string) (map[family.Domain]family.Level, string, error) {
	out := map[family.Domain]family.Level{}
	raw, _ := body.Get("grants")
	if raw == nil {
		return out, children, nil
	}
	keys, vals := phpval.Entries(raw)
	for i, k := range keys {
		lv, isStr := vals[i].(string)
		l, ok := family.ParseLevel(lv)
		if !isStr || !ok {
			return nil, "", fieldError(locale, "grants."+k, T("validation.level_invalid", locale))
		}
		if k == family.DomainChildren.String() {
			if l != family.LevelOff && l != family.LevelWrite {
				return nil, "", fieldError(locale, "grants."+k, T("validation.children_invalid", locale))
			}
			if children == "" {
				children = l.String()
			}
			continue
		}
		d, ok := family.ParseDomain(k)
		if !ok {
			return nil, "", fieldError(locale, "grants."+k, T("validation.domain_invalid", locale))
		}
		out[d] = l
	}
	return out, children, nil
}

// mapError turns domain errors into responses; anything else stays a 500.
func mapError(locale string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return notFoundErr(locale)
	case errors.Is(err, family.ErrLevelNotAllowed):
		return family.LevelNotAllowed(locale)
	case errors.Is(err, ErrInvalidRole):
		return fieldError(locale, "role", T("validation.role_invalid", locale))
	case errors.Is(err, ErrInvalidTemplate):
		return fieldError(locale, "template", T("validation.template_invalid", locale))
	case errors.Is(err, ErrInvalidPhone):
		return fieldError(locale, "phone", T("validation.phone_invalid", locale))
	case errors.Is(err, ErrSelf):
		return fieldError(locale, "phone", T("validation.self_invite", locale))
	case errors.Is(err, ErrInvalidName):
		return fieldError(locale, "display_name", T("validation.name_too_long", locale))
	case errors.Is(err, ErrInvalidExpiry):
		return fieldError(locale, "expires_at", T("validation.expires_invalid", locale))
	case errors.Is(err, ErrInvalidChildren):
		return fieldError(locale, "children", T("validation.children_invalid", locale))
	case errors.Is(err, ErrMutualNotAllowed):
		return fieldError(locale, "mutual_request.domains", T("validation.mutual_invalid", locale))
	case errors.Is(err, ErrMutualNotRequested):
		return fieldError(locale, "mutual", T("validation.mutual_not_requested", locale))
	case errors.Is(err, ErrConsentVersion):
		return fieldError(locale, "consent_version", T("validation.consent_version", locale))
	case errors.Is(err, ErrTooMany):
		return coded(locale, "messages.limit_reached", ErrorCodeLimitReached)
	case errors.Is(err, ErrTeenParentOnly):
		return coded(locale, "messages.teen_parent_only", ErrorCodeTeenParentOnly)
	case errors.Is(err, ErrMutualExists):
		return coded(locale, "messages.mutual_exists", ErrorCodeMutualExists)
	}
	return err
}

func coded(locale, key, code string) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T(key, locale), "error_code", code)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldError(locale, field, msg string) error {
	return failValidation(locale, jsonx.Obj(field, []string{msg}))
}

func notFoundErr(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale))
}

func unavailable(locale string) error {
	return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.unavailable", locale), "error_code", ErrorCodeUnavailable)
}

// TooManyAttempts is the localized 429 of the breaker (the throttles use the /companions one, same body).
func TooManyAttempts(locale string, retryAfter int, headers map[string]string) *httpx.FailError {
	e := httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many_attempts", locale),
		"error_code", ErrorCodeTooManyAttempts, "retry_after", retryAfter)
	for k, v := range headers {
		e = e.WithHeader(k, v)
	}
	return e
}

func str(m phpval.Map, key string) string {
	v, ok := m.Get(key)
	if !ok || v == nil {
		return ""
	}
	return phpval.ToString(v)
}

func truthy(m phpval.Map, key string) bool {
	v, ok := m.Get(key)
	return ok && v != nil && phpval.Truthy(v)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isoTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(civildate.Tehran).Format(time.RFC3339)
}

func grantsJSON(list []DomainLevel) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, g := range list {
		out = append(out, jsonx.Obj("domain", g.Domain.String(), "level", g.Level.String()))
	}
	return out
}

func domainCodes(ds []family.Domain) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.String())
	}
	return out
}

func createdLinkJSON(c Created) *jsonx.OrderedMap {
	var mutual any
	if len(c.Mutual) > 0 {
		mutual = jsonx.Obj("domains", domainCodes(c.Mutual))
	}
	children := c.Children
	if children == family.LevelOff.String() {
		children = ""
	}
	return jsonx.Obj(
		"id", c.LinkID,
		"role", c.Role.String(),
		"template", string(c.Template),
		"status", c.Status,
		"display_name", nullable(c.DisplayName),
		"expires_at", isoTime(c.LinkExpiresAt),
		"urgent_sms", c.UrgentSMS,
		"grants", grantsJSON(c.Grants),
		"children", nullable(children),
		"mutual_request", mutual,
	)
}

func inviteJSON(c Created, sent bool) *jsonx.OrderedMap {
	var phone any
	channel := ChannelCode
	if c.Phone != "" {
		phone, channel = MaskMobile(c.Phone), ChannelPhone
	}
	return jsonx.Obj("code", c.Code, "channel", channel, "expires_at", isoTime(c.ExpiresAt), "phone", phone,
		"sms_sent", sent)
}

func previewJSON(p Preview) *jsonx.OrderedMap {
	var mutual, link any
	if len(p.Mutual) > 0 {
		mutual = jsonx.Obj("domains", domainCodes(p.Mutual))
	}
	if p.LinkID != 0 {
		link = p.LinkID
	}
	return jsonx.Obj(
		"link_id", link,
		"owner", jsonx.Obj("first_name", nullable(p.OwnerFirstName)),
		"role", nullable(p.Role.String()),
		"template", nullable(p.Template),
		"domains", grantsJSON(p.Domains),
		"children", nullable(p.Children),
		"mutual_request", mutual,
		"owner_confirmation", p.OwnerConfirmation,
		"expires_at", isoTime(p.ExpiresAt),
	)
}
