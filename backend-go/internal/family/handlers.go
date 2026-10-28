package family

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// The family copy is data: lang/<code>/family.json (family.md §8), English fallback.
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err)
	}
	return t
})

// T is the family line for key ("messages.domain_denied") in locale.
func T(key, locale string) string { return translator().Trans("family."+key, nil, locale) }

// ErrorCodeDomainDenied is the 403 error_code of an active link without the domain (F-21).
const ErrorCodeDomainDenied = "family_domain_denied"

// CacheControlNoStore: member-facing family answers are never cached (T-M9-02).
const CacheControlNoStore = "private, no-store"

// DenyError is the HTTP answer of a denied member request (F-21): 404 for an unknown, revoked or expired link, a
// blocked owner or a child the actor does not guard (no existence oracle); 403 family_domain_denied on an active link
// without the domain. Neither carries any data.
func DenyError(locale string, d Decision) error {
	if d.Reason.NotFound() {
		return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale))
	}
	return httpx.Fail(fiber.StatusForbidden, T("messages.domain_denied", locale), "error_code", ErrorCodeDomainDenied)
}

// LevelNotAllowed is the 422 of a grant above the registry's maximum.
func LevelNotAllowed(locale string) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.level_not_allowed", locale), "error_code", ErrorCodeLevelNotAllowed)
}

// Handlers serve /api/v1/family/* (Go only, D-204).
type Handlers struct {
	db   *sql.DB
	base clock.Clock
}

// NewHandlers returns the family handlers on db.
func NewHandlers(db *sql.DB, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{db: db, base: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.base).Now().In(civildate.Tehran).Truncate(time.Second)
}

func isoTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

// Me is GET /api/v1/family/me: the caller's links as owner (role, template, status, expiry and every grant with its
// state: active | paused (stale epoch, F-23) | inactive (out of the owner's mode)) and as member (owner, role and the
// effective domains only — no paused or inactive hint, no expired or blocked link).
func (h *Handlers) Me(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, CacheControlNoStore)
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	now := h.now(c)
	ctx := c.Context()
	owned, err := h.asOwner(ctx, uid, now)
	if err != nil {
		return err
	}
	member, err := h.asMember(c, uid, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("as_owner", owned, "as_member", member))
}

func (h *Handlers) names(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := store.New(h.db).GetUserNames(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("family: names: %w", err)
	}
	for _, r := range rows {
		if n := strings.TrimSpace(r.Name.String); n != "" {
			out[r.ID] = n
		}
	}
	return out, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *Handlers) asOwner(ctx context.Context, ownerID uint64, now time.Time) ([]*jsonx.OrderedMap, error) {
	q := store.New(h.db)
	links, err := q.ListOwnerCompanions(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("family: links: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(links))
	if len(links) == 0 {
		return out, nil
	}
	st, err := SyncMode(ctx, h.db, ownerID, now)
	if err != nil {
		return nil, err
	}
	grants, err := q.ListOwnerGrants(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("family: grants: %w", err)
	}
	ids := make([]uint64, 0, len(links))
	for _, l := range links {
		if l.CompanionUserID.Valid {
			ids = append(ids, u64(l.CompanionUserID))
		}
	}
	names, err := h.names(ctx, ids)
	if err != nil {
		return nil, err
	}
	mode := ModeOf(st.Mode)
	for _, l := range links {
		role := ParseRole(l.Role.String, l.Type)
		status := l.Status
		if l.ExpiresAt.Valid && !now.Before(l.ExpiresAt.Time) {
			status = "expired"
		}
		list := make([]*jsonx.OrderedMap, 0, len(GrantDomains))
		for _, d := range GrantDomains {
			for _, g := range grants {
				if g.CompanionID != l.ID {
					continue
				}
				e, stamped := u32(g.ModeEpoch)
				gd, lv, ok := grantRow(g.Section, g.Level)
				if !ok || gd != d {
					continue
				}
				if !stamped && legacyEpochOne(g.Section, g.Level) {
					e, stamped = 1, true
				}
				spec, _ := Lookup(d)
				state := "active"
				switch {
				case spec.Modes&mode == 0:
					state = "inactive"
				case spec.EpochBound && (!stamped || e != st.Epoch):
					state = "paused"
				}
				list = append(list, jsonx.Obj("domain", d.String(), "level", lv.String(), "state", state))
			}
		}
		name := l.DisplayName.String
		if name == "" {
			name = names[u64(l.CompanionUserID)]
		}
		var member any
		if l.CompanionUserID.Valid {
			member = jsonx.Obj("id", u64(l.CompanionUserID), "name", nullable(name))
		}
		out = append(out, jsonx.Obj(
			"id", l.ID,
			"member", member,
			"role", nullable(role.String()),
			"template", nullable(l.Template.String),
			"status", status,
			"expires_at", isoTime(l.ExpiresAt.Time),
			"review_pending", st.ReviewPending,
			"grants", list,
		))
	}
	return out, nil
}

func (h *Handlers) asMember(c fiber.Ctx, memberID uint64, now time.Time) ([]*jsonx.OrderedMap, error) {
	snaps, err := Request(c, h.db, func() time.Time { return now }).ForMember(c.Context(), memberID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(snaps))
	for _, s := range snaps {
		ids = append(ids, s.OwnerID)
	}
	names, err := h.names(c.Context(), ids)
	if err != nil {
		return nil, err
	}
	out := make([]*jsonx.OrderedMap, 0, len(snaps))
	for _, s := range snaps {
		if Decide(s, Subject{}, DomainUrgentAlert, ActionRead, now).Reason.NotFound() {
			continue // expired link or blocked owner: like a revoked link, not listed (F-21)
		}
		domains := make([]*jsonx.OrderedMap, 0, len(GrantDomains))
		for _, d := range GrantDomains {
			subj := Subject{}
			if d == DomainHousehold {
				subj.Kind = SubjectHousehold
			}
			level := LevelOff
			switch {
			case Decide(s, subj, d, ActionWrite, now).Outcome != Deny:
				level = LevelWrite
			case Decide(s, subj, d, ActionNudge, now).Outcome != Deny:
				level = LevelNudge
			case Decide(s, subj, d, ActionRead, now).Outcome != Deny:
				level = LevelView
			}
			if level > LevelOff {
				domains = append(domains, jsonx.Obj("domain", d.String(), "level", level.String()))
			}
		}
		out = append(out, jsonx.Obj(
			"id", s.LinkID,
			"owner", jsonx.Obj("id", s.OwnerID, "name", nullable(names[s.OwnerID])),
			"role", nullable(s.Role.String()),
			"domains", domains,
		))
	}
	return out, nil
}
