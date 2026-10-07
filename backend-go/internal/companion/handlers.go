package companion

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Error codes of the controller-style failures.
const (
	ErrorCodeInviteInvalid    = "invite_invalid"
	ErrorCodeSectionNotShared = "section_not_shared"
	ErrorCodeUnavailable      = "companion_unavailable"
	ErrorCodeLimitReached     = "limit_reached"
	ErrorCodeSpouseExists     = "spouse_exists"
	ErrorCodeNotPending       = "not_pending"
	ErrorCodeTooManyAttempts  = "too_many_requests"
	// ErrorCodeParentNeedsTeen: a parent invite from an owner who is not in teen mode (CB-TEEN-01).
	ErrorCodeParentNeedsTeen = "parent_needs_teen"
	// ErrorCodeTeenParentOnly: a teen-mode owner tried to invite a partner or spouse (CB-TEEN-01).
	ErrorCodeTeenParentOnly = "teen_parent_only"
	// ErrorCodeInviteNotAllowed: accept refused because the owner's life mode no longer allows the link type.
	ErrorCodeInviteNotAllowed = "invite_not_allowed"
)

// MaxAuditEntries caps GET /companions/audit.
const MaxAuditEntries = 200

// InviteSMS delivers an invite code (internal/sms implements it).
type InviteSMS interface {
	Delivers() bool
	// SendInvite texts the code; discreet is the owner's «اعلان‌های محرمانه» flag (CB-PRIV-01).
	SendInvite(ctx context.Context, mobile, code string, discreet bool) error
}

// SectionReader builds the access-filtered view of one section of an owner's data (internal/companion/shared). The
// link's grants make the view mode-aware: without the pregnancy grant no pregnancy signal leaves (CMP-H1).
type SectionReader interface {
	ReadFor(ctx context.Context, link Link, section Section, locale string, now time.Time) (any, error)
}

// ChildOwnership verifies that child ids belong to the owner (children.Service, bloom B-N5-02). Without one the
// default owns nothing, so a non-empty shared-children list is refused instead of linking ids nobody checked.
type ChildOwnership interface {
	OwnsChildren(ctx context.Context, ownerID uint64, childIDs []uint64) (bool, error)
}

type noChildren struct{}

func (noChildren) OwnsChildren(_ context.Context, _ uint64, ids []uint64) (bool, error) {
	return len(ids) == 0, nil
}

// HandlerOptions wires the handlers.
type HandlerOptions struct {
	Service  Options
	SMS      InviteSMS
	Reader   SectionReader
	Children ChildOwnership // children.Service; nil = refuse every child id
	// Disabled answers 503 on invite creation / renewal / acceptance (production without COMPANION_CODE_PEPPER).
	Disabled bool
	Logger   *slog.Logger
	// SendGate rate-limits invite SMS (per recipient, per owner per day, per link resend); nil = no limits (tests).
	SendGate SendGate
	// Breaker is the global failed-accept circuit breaker (CMP-M3); nil = none (tests).
	Breaker AcceptBreaker
}

// Handlers are the /api/v1/companions actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	conn     Conn
	base     clock.Clock
	opt      HandlerOptions
	children ChildOwnership
	logger   *slog.Logger
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(conn Conn, base clock.Clock, opt HandlerOptions) *Handlers {
	h := &Handlers{conn: conn, base: base, opt: opt, children: opt.Children, logger: opt.Logger}
	if h.children == nil {
		h.children = noChildren{}
	}
	if h.logger == nil {
		h.logger = slog.Default()
	}
	return h
}

// svc is the domain service on the request clock.
func (h *Handlers) svc(c fiber.Ctx) *Service {
	return NewService(h.conn, clock.FromContext(c, h.base), h.opt.Service)
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.base).Now() }

func currentUser(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func linkID(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

// ---------------------------------------------------------------- owner

// Index is GET /companions: the owner's invited and active companions (Hamdam_List).
func (h *Handlers) Index(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	svc := h.svc(c)
	links, err := svc.ListForOwner(c.Context(), ownerID)
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.CompanionUserID)
	}
	names, err := svc.Names(c.Context(), ids...)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(links))
	for _, l := range links {
		out = append(out, ownerLinkJSON(l, names))
	}
	return httpx.OK(c, out)
}

// Show is GET /companions/{id}: one of the owner's companions.
func (h *Handlers) Show(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	svc := h.svc(c)
	l, err := h.ownerLink(c, svc, ownerID)
	if err != nil {
		return err
	}
	return h.ownerLinkOK(c, svc, l, "")
}

// Store is POST /companions {type, display_name?, phone?, grants?, child_ids?}: a pending companion with its
// one-time code (Hamdam_Type → Access → Children → Invite). With a phone the code also goes out by SMS through the
// adapter; the code is returned either way so the owner can share it herself. 201.
func (h *Handlers) Store(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if h.opt.Disabled {
		return unavailable(locale)
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("type", "required", "string", validation.In(typeStrings()...)),
		validation.F("display_name", "nullable", "string", "max:100"),
		validation.F("phone", "nullable", "string", "max:20"),
		validation.F("grants", "nullable", "array"),
		validation.F("child_ids", "nullable", "array", "max:20"),
		validation.F("child_ids.*", "integer", "min:1"),
	}, validation.Now(h.now(c)),
		validation.Messages("type.in", T("validation.type_invalid", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	in := InviteInput{Type: Type(str(body, "type")), DisplayName: str(body, "display_name"), Phone: str(body, "phone")}
	if in.Grants, err = parseGrants(locale, body, in.Type); err != nil {
		return err
	}
	if in.ChildIDs, err = h.childIDs(c, ownerID, body); err != nil {
		return err
	}
	svc := h.svc(c)
	created, err := svc.CreateInvite(c.Context(), ownerID, in)
	if err != nil {
		return mapError(locale, err)
	}
	sent := h.sendInvite(c, created)
	link, err := svc.OwnerLink(c.Context(), ownerID, created.Link.ID)
	if err != nil {
		return err
	}
	names, err := svc.Names(c.Context(), link.CompanionUserID)
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("companion", ownerLinkJSON(link, names), "invite", inviteJSON(created, sent)),
		T("messages.invite_created", locale))
}

// Renew is POST /companions/{id}/renew {phone?}: a fresh code for a pending companion (the old code stops working).
func (h *Handlers) Renew(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if h.opt.Disabled {
		return unavailable(locale)
	}
	id, ok := linkID(c)
	if !ok {
		return notFoundErr(locale)
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("phone", "nullable", "string", "max:20"),
	}, validation.Now(h.now(c)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	svc := h.svc(c)
	created, err := svc.RenewInvite(c.Context(), ownerID, id, str(body, "phone"))
	if err != nil {
		return mapError(locale, err)
	}
	sent := h.sendInvite(c, created)
	link, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("companion", ownerLinkJSON(link, nil), "invite", inviteJSON(created, sent)),
		T("messages.invite_renewed", locale))
}

// UpdateGrants is PUT /companions/{id}/grants {grants: {section: none|view|edit}}: replaces the access (missing
// sections become none). Takes effect on the companion's next request.
func (h *Handlers) UpdateGrants(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	if !ok {
		return notFoundErr(locale)
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("grants", "present", "array"),
	}, validation.Now(h.now(c)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	svc := h.svc(c)
	cur, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	grants, err := parseGrants(locale, body, cur.Type)
	if err != nil {
		return err
	}
	if _, err := svc.SetGrants(c.Context(), ownerID, id, grants); err != nil {
		return mapError(locale, err)
	}
	l, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	return h.ownerLinkOK(c, svc, l, T("messages.grants_updated", locale))
}

// UpdateChildren is PUT /companions/{id}/children {child_ids: []}: the shared children of a spouse companion.
func (h *Handlers) UpdateChildren(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	if !ok {
		return notFoundErr(locale)
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("child_ids", "present", "array", "max:20"),
		validation.F("child_ids.*", "integer", "min:1"),
	}, validation.Now(h.now(c)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	svc := h.svc(c)
	if _, err := h.ownerLink(c, svc, ownerID); err != nil {
		return err
	}
	ids, err := h.childIDs(c, ownerID, body)
	if err != nil {
		return err
	}
	if _, err := svc.SetSharedChildren(c.Context(), ownerID, id, ids); err != nil {
		return mapError(locale, err)
	}
	l, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	return h.ownerLinkOK(c, svc, l, T("messages.children_updated", locale))
}

// Destroy is DELETE /companions/{id}: the owner removes a companion (pending or active). Access ends at once.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	svc := h.svc(c)
	l, err := h.ownerLink(c, svc, ownerID)
	if err != nil {
		return err
	}
	if err := svc.Revoke(c.Context(), ownerID, l.ID); err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, nil, T("messages.revoked", locale))
}

// Audit is GET /companions/audit?limit=&before_id=&action=: who of the owner's companions read or wrote what, and
// when (no payload), newest first. before_id pages back (the id of the last entry of the previous page); action keeps
// one kind of entry, so write and lifecycle rows stay reachable however many reads there are (CMP-L1).
func (h *Handlers) Audit(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	q := validation.Query(c)
	v := validation.Make(lang.Default(), locale, q, validation.Rules{
		validation.F("limit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxAuditEntries)),
		validation.F("before_id", "nullable", "integer", "min:1"),
		validation.F("action", "nullable", "string", validation.In(auditActionStrings()...)),
	}, validation.Now(h.now(c)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	limit := 50
	if raw := str(q, "limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	var before uint64
	if raw := str(q, "before_id"); raw != "" {
		if before, err = strconv.ParseUint(raw, 10, 64); err != nil {
			return fieldError(locale, "before_id", T("messages.validation_failed", locale))
		}
	}
	svc := h.svc(c)
	entries, err := svc.AuditPage(c.Context(), ownerID, before, Action(str(q, "action")), limit)
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ActorID)
	}
	names, err := svc.Names(c.Context(), ids...)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(entries))
	for _, e := range entries {
		var section, actor, companion any
		if e.Section != "" {
			section = string(e.Section)
		}
		if e.ActorID != 0 {
			actor = jsonx.Obj("id", e.ActorID, "name", nullable(names[e.ActorID]), "is_me", e.ActorID == ownerID)
		}
		if e.CompanionID != 0 {
			companion = e.CompanionID
		}
		out = append(out, jsonx.Obj("id", e.ID, "action", string(e.Action), "section", section,
			"companion_id", companion, "actor", actor, "at", isoTime(e.At)))
	}
	return httpx.OK(c, out)
}

// ---------------------------------------------------------------- companion

// Accept is POST /companions/accept {code}: redeem an invite code (at sign-up or later). Every refusal — unknown,
// expired, used, locked, bound to another number, the owner's own code, an existing link — is the same 422, so the
// answer never tells whether a code exists. Rate-limited per user and per IP (internal/http).
func (h *Handlers) Accept(c fiber.Ctx) error {
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if h.opt.Disabled {
		return unavailable(locale)
	}
	if err := h.acceptPaused(c, locale); err != nil {
		return err
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("code", "required", "string", "max:32"),
	}, validation.Now(h.now(c)),
		validation.Messages("code.required", T("validation.code_required", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	svc := h.svc(c)
	link, err := svc.Accept(c.Context(), viewerID, str(body, "code"))
	if errors.Is(err, ErrInviteNotAllowed) {
		msg := T("messages.invite_not_allowed", locale)
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg, "errors", jsonx.Obj("code", []string{msg}),
			"error_code", ErrorCodeInviteNotAllowed)
	}
	if err != nil {
		if isInviteRefusal(err) {
			h.acceptRefused(c)
			msg := T("messages.invite_invalid", locale)
			return httpx.Fail(fiber.StatusUnprocessableEntity, msg, "errors", jsonx.Obj("code", []string{msg}),
				"error_code", ErrorCodeInviteInvalid)
		}
		return err
	}
	if err := svc.NotifyOwner(c.Context(), link, NoticeAccepted, i18n.LanguagesOf(c).Codes()); err != nil {
		// The link is active and audited; the owner's inbox notice is best effort.
		h.logger.ErrorContext(c.Context(), "companion accept notice failed", slog.Uint64("companion_id", link.ID),
			slog.String("error", err.Error()))
	}
	names, err := svc.Names(c.Context(), link.OwnerID)
	if err != nil {
		return err
	}
	return httpx.OK(c, companionLinkJSON(link, names), T("messages.accepted", locale))
}

// Links is GET /companions/links: the active links in which the caller is the companion (whose data she/he sees).
func (h *Handlers) Links(c fiber.Ctx) error {
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	svc := h.svc(c)
	links, err := svc.ListForCompanion(c.Context(), viewerID)
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.OwnerID)
	}
	names, err := svc.Names(c.Context(), ids...)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(links))
	for _, l := range links {
		out = append(out, companionLinkJSON(l, names))
	}
	return httpx.OK(c, out)
}

// Leave is DELETE /companions/links/{id}: the companion ends the link.
func (h *Handlers) Leave(c fiber.Ctx) error {
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	if !ok {
		return notFoundErr(locale)
	}
	svc := h.svc(c)
	if _, err := svc.CompanionLink(c.Context(), viewerID, id); err != nil {
		return mapError(locale, err)
	}
	if err := svc.Revoke(c.Context(), viewerID, id); err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, nil, T("messages.left", locale))
}

// Section is GET /companions/links/{id}/sections/{section}: the access-filtered view of one section of the owner's
// data. 404 when the caller is not the active companion of that link (or the section is unknown, or not one the link
// type may hold), 403 section_not_shared without a view/edit grant. A parent link (CB-TEEN-01) has no section reads
// at all: its only view is the teen card (GET /teen/linked). Every successful read is audited before the data is
// built.
func (h *Handlers) Section(c fiber.Ctx) error {
	viewerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	section := Section(c.Params("section"))
	if !ok || !section.Valid() {
		return notFoundErr(locale)
	}
	svc := h.svc(c)
	link, err := svc.CompanionLink(c.Context(), viewerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	if link.Type == TypeParent || !section.AllowedFor(link.Type) {
		return notFoundErr(locale)
	}
	level, err := svc.Level(c.Context(), link.OwnerID, viewerID, section)
	if err != nil {
		return err
	}
	if !level.CanRead() {
		return httpx.Fail(fiber.StatusForbidden, T("messages.section_not_shared", locale), "error_code", ErrorCodeSectionNotShared)
	}
	if err := svc.Audit(c.Context(), link.OwnerID, viewerID, link.ID, section, ActionRead); err != nil {
		return err
	}
	data, err := h.opt.Reader.ReadFor(c.Context(), link, section, locale, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("companion_id", link.ID, "owner_id", link.OwnerID, "section", string(section),
		"level", string(level), "data", data))
}

// ---------------------------------------------------------------- helpers

func (h *Handlers) ownerLink(c fiber.Ctx, svc *Service, ownerID uint64) (Link, error) {
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	if !ok {
		return Link{}, notFoundErr(locale)
	}
	l, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return Link{}, mapError(locale, err)
	}
	return l, nil
}

func (h *Handlers) ownerLinkOK(c fiber.Ctx, svc *Service, l Link, msg string) error {
	names, err := svc.Names(c.Context(), l.CompanionUserID)
	if err != nil {
		return err
	}
	if msg == "" {
		return httpx.OK(c, ownerLinkJSON(l, names))
	}
	return httpx.OK(c, ownerLinkJSON(l, names), msg)
}

// sendInvite texts the code when the invite is bound to a phone and the adapter delivers. A provider failure only
// means the owner shares the code herself (sms_sent=false); it is logged with a masked number, never the code.
func (h *Handlers) sendInvite(c fiber.Ctx, inv CreatedInvite) bool {
	if inv.Phone == "" || h.opt.SMS == nil || !h.opt.SMS.Delivers() {
		return false
	}
	if !h.smsAllowed(c, inv) {
		return false
	}
	// CB-PRIV-01: the owner's «اعلان‌های محرمانه» picks the neutral wording; a read error fails safe (neutral).
	discreet, err := notifications.Discreet(c.Context(), profilestore.New(h.conn), inv.Link.OwnerID)
	if err != nil {
		h.logger.WarnContext(c.Context(), "companion invite SMS: notification preferences unreadable, sending neutral",
			slog.String("error", err.Error()))
	}
	if err := h.opt.SMS.SendInvite(c.Context(), inv.Phone, inv.Code, discreet); err != nil {
		h.logger.WarnContext(c.Context(), "companion invite SMS failed",
			slog.String("mobile", MaskMobile(inv.Phone)), slog.String("error", err.Error()))
		return false
	}
	return true
}

func (h *Handlers) childIDs(c fiber.Ctx, ownerID uint64, body phpval.Map) ([]uint64, error) {
	locale := i18n.Locale(c)
	raw, _ := body.Get("child_ids")
	if raw == nil {
		return nil, nil
	}
	_, vals := phpval.Entries(raw)
	ids := make([]uint64, 0, len(vals))
	for _, v := range vals {
		id, err := strconv.ParseUint(phpval.ToString(v), 10, 64)
		if err != nil || id == 0 {
			return nil, fieldError(locale, "child_ids", T("validation.children_invalid", locale))
		}
		ids = append(ids, id)
	}
	ids = compactIDs(ids)
	ok, err := h.children.OwnsChildren(c.Context(), ownerID, ids)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fieldError(locale, "child_ids", T("validation.child_invalid", locale))
	}
	return ids, nil
}

// parseGrants reads {grants: {section: none|view|edit}} for a link of type t. Unknown sections, sections the type may
// not hold (a parent link: only the teen sections; others: never them) and levels the section does not permit (teen
// sections: view only) are a 422 on grants.<key>.
func parseGrants(locale string, body phpval.Map, t Type) (Grants, error) {
	out := Grants{}
	raw, _ := body.Get("grants")
	if raw == nil {
		return out, nil
	}
	keys, vals := phpval.Entries(raw)
	for i, k := range keys {
		s := Section(k)
		if !s.AllowedFor(t) {
			return nil, fieldError(locale, "grants."+k, T("validation.section_invalid", locale))
		}
		lv, isStr := vals[i].(string)
		l := Level(lv)
		if !isStr || !l.Valid() {
			return nil, fieldError(locale, "grants."+k, T("validation.level_invalid", locale))
		}
		if !s.Permits(l) {
			return nil, fieldError(locale, "grants."+k, T("validation.level_view_only", locale))
		}
		if l != LevelNone {
			out[s] = l
		}
	}
	return out, nil
}

func isInviteRefusal(err error) bool {
	for _, e := range []error{ErrInviteInvalid, ErrInviteExpired, ErrInviteUsed, ErrInviteLocked,
		ErrInvitePhoneMismatch, ErrSelfInvite, ErrAlreadyLinked, ErrNotFound} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// mapError turns domain errors into responses; anything else stays a 500.
func mapError(locale string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return notFoundErr(locale)
	case errors.Is(err, ErrInvalidType):
		return fieldError(locale, "type", T("validation.type_invalid", locale))
	case errors.Is(err, ErrInvalidPhone):
		return fieldError(locale, "phone", T("validation.phone_invalid", locale))
	case errors.Is(err, ErrPhoneRequired):
		return fieldError(locale, "phone", T("validation.phone_required_parent", locale))
	case errors.Is(err, ErrInvalidName):
		return fieldError(locale, "display_name", T("validation.name_too_long", locale))
	case errors.Is(err, ErrInvalidSection), errors.Is(err, ErrInvalidLevel):
		return fieldError(locale, "grants", T("validation.grants_invalid", locale))
	case errors.Is(err, ErrChildrenNoSpouse):
		return fieldError(locale, "child_ids", T("messages.children_no_spouse", locale))
	case errors.Is(err, ErrSelfInvite):
		return fieldError(locale, "phone", T("messages.self_invite", locale))
	case errors.Is(err, ErrTooManyCompanions):
		return coded(locale, "messages.limit_reached", ErrorCodeLimitReached)
	case errors.Is(err, ErrSpouseExists):
		return coded(locale, "messages.spouse_exists", ErrorCodeSpouseExists)
	case errors.Is(err, ErrNotPending):
		return coded(locale, "messages.not_pending", ErrorCodeNotPending)
	case errors.Is(err, ErrParentNeedsTeen):
		return coded(locale, "messages.parent_needs_teen", ErrorCodeParentNeedsTeen)
	case errors.Is(err, ErrTeenParentOnly):
		return coded(locale, "messages.teen_parent_only", ErrorCodeTeenParentOnly)
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

// TooManyAttempts is the localized 429 of the accept / invite throttles (internal/http).
func TooManyAttempts(locale string, retryAfter int, headers map[string]string) *httpx.FailError {
	e := httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many_attempts", locale),
		"error_code", ErrorCodeTooManyAttempts, "retry_after", retryAfter)
	for k, v := range headers {
		e = e.WithHeader(k, v)
	}
	return e
}

func typeStrings() []string {
	out := make([]string, len(Types))
	for i, t := range Types {
		out[i] = string(t)
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

// MaskMobile keeps the operator prefix and the last 3 digits: 09123456789 → 0912****789.
func MaskMobile(m string) string {
	if len(m) < 8 {
		return "***"
	}
	return m[:4] + "****" + m[len(m)-3:]
}

// grantsJSON lists every section the link type may hold (partner / spouse: the regular five; parent: the teen three).
func grantsJSON(g Grants, t Type) *jsonx.OrderedMap {
	out := jsonx.NewObject()
	for _, s := range SectionsFor(t) {
		out.Set(string(s), string(g.Of(s)))
	}
	return out
}

// ownerLinkJSON is a companion as the owner sees it. name = the owner's label, else the companion account's name.
func ownerLinkJSON(l Link, names map[uint64]string) *jsonx.OrderedMap {
	name := l.DisplayName
	if name == "" {
		name = names[l.CompanionUserID]
	}
	var invite, family any
	if !l.InviteExpiresAt.IsZero() {
		var phone any
		if l.InvitePhone != "" {
			phone = MaskMobile(l.InvitePhone)
		}
		invite = jsonx.Obj("expires_at", isoTime(l.InviteExpiresAt), "phone", phone)
	}
	if l.FamilyID != 0 {
		ids := l.SharedChildIDs
		if ids == nil {
			ids = []uint64{}
		}
		family = jsonx.Obj("id", l.FamilyID, "shared_child_ids", ids)
	}
	return jsonx.Obj(
		"id", l.ID,
		"type", string(l.Type),
		"status", string(l.Status),
		"display_name", nullable(l.DisplayName),
		"name", nullable(name),
		"invited_at", isoTime(l.InvitedAt),
		"accepted_at", isoTime(l.AcceptedAt),
		"grants", grantsJSON(l.Grants, l.Type),
		"invite", invite,
		"family", family,
	)
}

// companionLinkJSON is a link as the companion sees it: whose data, which sections, where «ثبت برای …» applies.
func companionLinkJSON(l Link, names map[uint64]string) *jsonx.OrderedMap {
	recordFor := []string{}
	for _, s := range []Section{SectionMeds, SectionAppointments} {
		if l.Grants.Of(s).CanWrite() {
			recordFor = append(recordFor, string(s))
		}
	}
	var family any
	if l.FamilyID != 0 {
		ids := l.SharedChildIDs
		if ids == nil {
			ids = []uint64{}
		}
		family = jsonx.Obj("id", l.FamilyID, "shared_child_ids", ids)
	}
	return jsonx.Obj(
		"id", l.ID,
		"type", string(l.Type),
		"owner", jsonx.Obj("id", l.OwnerID, "name", nullable(names[l.OwnerID])),
		"accepted_at", isoTime(l.AcceptedAt),
		"grants", grantsJSON(l.Grants, l.Type),
		"can_record_for", recordFor,
		"family", family,
	)
}

func inviteJSON(inv CreatedInvite, sent bool) *jsonx.OrderedMap {
	var phone any
	if inv.Phone != "" {
		phone = MaskMobile(inv.Phone)
	}
	return jsonx.Obj("code", inv.Code, "expires_at", isoTime(inv.ExpiresAt), "phone", phone, "sms_sent", sent)
}
