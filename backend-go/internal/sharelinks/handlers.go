package sharelinks

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the share-link actions: owner routes under /api/v1/health-record/share-links (locale, auth
// RequireUser; creation also behind the plus.PDFShare gate) and the public GET /api/v1/shared-reports/{token}
// (locale, IP throttle, no auth). No action logs a token or a report.
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

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", CodeNotFound)
}

// Store is POST /health-record/share-links {range, from?, sections[], question?} → 201 {link…, token}. The token is
// returned only here; the client builds the public URL from it.
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	req, err := healthrecord.ParseReportRequest(validation.Input(c), locale, now, true)
	if err != nil {
		return err
	}
	created, err := h.svc.Create(c, userID, req, now, locale, i18n.LanguagesOf(c).DefaultCode())
	if errors.Is(err, ErrLimit) {
		return httpx.Fail(fiber.StatusConflict,
			Tp("messages.limit", map[string]string{"max": strconv.Itoa(MaxActive)}, locale), "error_code", CodeLimit)
	}
	if err != nil {
		return err
	}
	body := created.Link.JSON(now)
	body.Set("token", created.Token)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.Created(c, body, T("messages.created", locale))
}

// Index is GET /health-record/share-links: the owner's links of the last 30 days (metadata only), newest first.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	links, err := h.svc.List(c, userID, now)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(links))
	active := 0
	for _, l := range links {
		if l.Status(now) == StatusActive {
			active++
		}
		items = append(items, l.JSON(now))
	}
	return httpx.OK(c, jsonx.Obj("items", items, "active_count", active, "max_active", MaxActive,
		"ttl_days", int(LinkTTL/(24*time.Hour))))
}

// Destroy is DELETE /health-record/share-links/{id}: revokes the link now (404 for an unknown or foreign id).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, perr := strconv.ParseUint(c.Params("id"), 10, 64)
	if perr != nil || id == 0 {
		return notFound(locale)
	}
	link, err := h.svc.Revoke(c, userID, id, now)
	if errors.Is(err, ErrNotFound) {
		return notFound(locale)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, link.JSON(now), T("messages.revoked", locale))
}

// Show is the public GET /shared-reports/{token}: the frozen report {version, locale, created_at, range, record,
// question, expires_at}; 404 share_link_not_found, 410 share_link_expired / share_link_revoked. Never cached or
// indexed.
func (h *Handlers) Show(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
	// CB-REC-03: every open writes the access log (coarse client class only); a 24h summary that stopped working is 404.
	opened, err := h.svc.OpenAs(c, c.Params("token"), now, ViewerOf(ViaLink, c.Get(fiber.HeaderUserAgent)))
	switch {
	case errors.Is(err, ErrNotFound):
		return notFound(locale)
	case errors.Is(err, ErrRevoked):
		return httpx.Fail(fiber.StatusGone, T("messages.revoked_link", locale), "error_code", CodeRevoked)
	case errors.Is(err, ErrExpired):
		return httpx.Fail(fiber.StatusGone, T("messages.expired", locale), "error_code", CodeExpired)
	case err != nil:
		return err
	}
	opened.Report.Set("expires_at", jsonx.ISO8601(opened.ExpiresAt))
	return httpx.OK(c, opened.Report)
}
