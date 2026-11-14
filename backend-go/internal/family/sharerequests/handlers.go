package sharerequests

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion/store"
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

// Error codes.
const (
	ErrorCodeTooManyRequests = "too_many_requests"
	ErrorCodeLinkPending     = "link_pending"
	ErrorCodeTeenParentOnly  = "teen_parent_only"
)

// Handlers serve /api/v1/family/share-requests (Go only, D-196).
type Handlers struct {
	conn Conn
	base clock.Clock
}

// NewHandlers wires the handlers on conn.
func NewHandlers(conn Conn, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{conn: conn, base: base}
}

func (h *Handlers) svc(c fiber.Ctx) *Service { return NewService(h.conn, clock.FromContext(c, h.base)) }

func currentUser(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// Store is POST /family/share-requests {to_user_id, domains: {domain: level}}: ask a linked person for more access to
// her data. 201; 404 when the two are not linked; 422 above the registry's caps; 429 when a domain was asked of her
// within 7 days.
func (h *Handlers) Store(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	uid, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		validation.F("to_user_id", "required", "integer", "min:1"),
		validation.F("domains", "required", "array", "max:12"),
	}, validation.Now(clock.FromContext(c, h.base).Now()),
		validation.Messages("domains.required", T("validation.domains_required", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	to, _ := strconv.ParseUint(phpval.ToString(must(body, "to_user_id")), 10, 64)
	domains := map[family.Domain]family.Level{}
	keys, vals := phpval.Entries(must(body, "domains"))
	for i, k := range keys {
		d, ok := family.ParseDomain(k)
		if !ok {
			return fieldError(locale, "domains."+k, T("validation.domain_invalid", locale))
		}
		l, ok := family.ParseLevel(phpval.ToString(vals[i]))
		if !ok || l == family.LevelOff {
			return fieldError(locale, "domains."+k, T("validation.level_invalid", locale))
		}
		domains[d] = l
	}
	r, err := h.svc(c).Create(c.Context(), uid, to, domains)
	if err != nil {
		return mapError(locale, err)
	}
	names, err := h.names(c, r)
	if err != nil {
		return err
	}
	return httpx.Created(c, requestJSON(r, names), T("messages.created", locale))
}

// Index is GET /family/share-requests?box=in|out (default in): requests asked of me (live only) or by me.
func (h *Handlers) Index(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	uid, err := currentUser(c)
	if err != nil {
		return err
	}
	box := BoxIn
	if strings.EqualFold(c.Query("box"), string(BoxOut)) {
		box = BoxOut
	}
	list, err := h.svc(c).List(c.Context(), uid, box)
	if err != nil {
		return err
	}
	names, err := h.names(c, list...)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, r := range list {
		out = append(out, requestJSON(r, names))
	}
	return httpx.OK(c, out)
}

// Approve is POST /family/share-requests/{id}/approve: only the data owner; the requested levels apply now.
func (h *Handlers) Approve(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	uid, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := paramID(c)
	if !ok {
		return notFoundErr(locale)
	}
	linkID, err := h.svc(c).Approve(c.Context(), uid, id)
	if err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, jsonx.Obj("id", id, "status", StatusAccepted, "link_id", linkID), T("messages.approved", locale))
}

// Decline is POST /family/share-requests/{id}/decline: only the data owner; nothing else changes.
func (h *Handlers) Decline(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	uid, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := paramID(c)
	if !ok {
		return notFoundErr(locale)
	}
	if err := h.svc(c).Decline(c.Context(), uid, id); err != nil {
		return mapError(locale, err)
	}
	return httpx.OK(c, jsonx.Obj("id", id, "status", StatusDeclined), T("messages.declined", locale))
}

func (h *Handlers) names(c fiber.Ctx, rs ...Request) (map[uint64]string, error) {
	ids := []uint64{}
	for _, r := range rs {
		ids = append(ids, r.FromUserID, r.ToUserID)
	}
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := store.New(h.conn).GetUserNames(c.Context(), ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if n := strings.TrimSpace(r.Name.String); n != "" {
			out[r.ID] = n
		}
	}
	return out, nil
}

func paramID(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func must(m phpval.Map, key string) any {
	v, _ := m.Get(key)
	return v
}

func mapError(locale string, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return notFoundErr(locale)
	case errors.Is(err, ErrSelf):
		return fieldError(locale, "to_user_id", T("validation.self", locale))
	case errors.Is(err, ErrEmpty):
		return fieldError(locale, "domains", T("validation.domains_required", locale))
	case errors.Is(err, family.ErrLevelNotAllowed):
		return family.LevelNotAllowed(locale)
	case errors.Is(err, ErrRateLimited):
		return httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many_requests", locale), "error_code", ErrorCodeTooManyRequests)
	case errors.Is(err, ErrTeenParentOnly):
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.teen_parent_only", locale), "error_code", ErrorCodeTeenParentOnly)
	case errors.Is(err, ErrLinkPending):
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.link_pending", locale), "error_code", ErrorCodeLinkPending)
	}
	return err
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

func isoTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(civildate.Tehran).Format(time.RFC3339)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func requestJSON(r Request, names map[uint64]string) *jsonx.OrderedMap {
	domains := make([]*jsonx.OrderedMap, 0, len(r.Domains))
	for _, d := range family.GrantDomains {
		if l, ok := r.Domains[d]; ok {
			domains = append(domains, jsonx.Obj("domain", d.String(), "level", l.String()))
		}
	}
	return jsonx.Obj(
		"id", r.ID,
		"from", jsonx.Obj("id", r.FromUserID, "name", nullable(names[r.FromUserID])),
		"to", jsonx.Obj("id", r.ToUserID, "name", nullable(names[r.ToUserID])),
		"domains", domains,
		"status", r.Status,
		"expires_at", isoTime(r.ExpiresAt),
		"decided_at", isoTime(r.DecidedAt),
		"created_at", isoTime(r.CreatedAt),
	)
}
