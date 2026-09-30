package content

import (
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// InfoPage is GET /info-pages/{group} (B-N1-12, Go only): the admin-managed boxes of one screen for the Night &
// Bloom Support / About / Legal screens. Unlike the Laravel-compatible /info/{group} it carries each box's stable
// `key` (seeded rows: `summary` = the Legal «خلاصه در ۳ خط», one line per "\n") and the page's last edit date
// (`updated_at`, Y-m-d Tehran, null for an empty page). Localized by Accept-Language; unknown group → 404.
func (h *Handlers) InfoPage(c fiber.Ctx) error {
	group := c.Params("group")
	if !slices.Contains(InfoGroups, group) {
		return httpx.NotFound()
	}
	locale := i18n.ResolveLocale(c, "")
	def := i18n.LanguagesOf(c).DefaultCode()
	rows, err := h.q.ListActiveInfoPageSections(c, group)
	if err != nil {
		return fmt.Errorf("content: info page: %w", err)
	}
	var latest time.Time
	sections := make([]*jsonx.OrderedMap, len(rows))
	for i, r := range rows {
		if r.UpdatedAt.Valid && r.UpdatedAt.Time.After(latest) {
			latest = r.UpdatedAt.Time
		}
		var key, linkLabel, linkURL any
		if r.Key.Valid && r.Key.String != "" {
			key = r.Key.String
		}
		if r.LinkUrl.Valid && r.LinkUrl.String != "" && r.LinkLabel.Valid {
			if label, ok := pickString(r.LinkLabel.V, locale, def); ok && label != "" {
				linkLabel, linkURL = label, r.LinkUrl.String
			}
		}
		heading, _ := pickString(r.Heading, locale, def)
		body, _ := pickString(r.Body, locale, def)
		sections[i] = jsonx.Obj(
			"id", r.ID,
			"key", key,
			"heading", heading,
			"body", body,
			"link_label", linkLabel,
			"link_url", linkURL,
		)
	}
	var updated any
	if !latest.IsZero() {
		updated = civildate.FromTime(latest.In(civildate.Tehran))
	}
	return httpx.OK(c, jsonx.Obj("group", group, "updated_at", updated, "sections", jsonx.List(sections)))
}
