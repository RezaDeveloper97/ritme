package projections

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Handlers serve GET /api/v1/family/owners/{ownerId}/domains/{domain} (Go only, D-194, D-204).
type Handlers struct {
	db      *sql.DB
	builder *Builder
	audit   *Auditor
	base    clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, b *Builder, a *Auditor, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{db: db, builder: b, audit: a, base: base}
}

// NoStore marks every /family/* answer private and uncacheable (errors included). Mount it first.
func NoStore(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	return c.Next()
}

// Domain is GET /api/v1/family/owners/{ownerId}/domains/{domain}: the member projection of one domain. 404 for an
// unknown owner, link or domain, an expired link or a blocked owner (F-21); 403 family_domain_denied on an active
// link without the domain (off, paused by a mode change, out of the owner's mode). A denied read carries no data and
// is not audited; an allowed one is audited before the data is built.
func (h *Handlers) Domain(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, family.CacheControlNoStore)
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	notFound := family.DenyError(locale, family.Decision{Reason: family.ReasonNoLink})
	ownerID, err := strconv.ParseUint(c.Params("ownerId"), 10, 64)
	if err != nil || ownerID == 0 {
		return notFound
	}
	d, ok := family.ParseDomain(c.Params("domain"))
	if !ok || !Readable(d) {
		return notFound
	}
	now := clock.FromContext(c, h.base).Now().In(civildate.Tehran).Truncate(time.Second)
	snap, err := family.Request(c, h.db, func() time.Time { return now }).Pair(c.Context(), ownerID, uid)
	if err != nil {
		return err
	}
	if dec := family.Decide(snap, family.Subject{}, d, family.ActionRead, now); dec.Outcome != family.Allow {
		return family.DenyError(locale, dec)
	}
	if err := h.audit.Read(c.Context(), snap, d, now); err != nil {
		return err
	}
	data, err := h.builder.Project(c.Context(), snap, d, locale, now)
	if errors.Is(err, ErrDenied) {
		return family.DenyError(locale, family.Decision{Reason: family.ReasonLevel})
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("owner_id", ownerID, "domain", d.String(), "data", data))
}
