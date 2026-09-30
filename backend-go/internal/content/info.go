package content

import (
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// InfoGroups is InfoSection::GROUPS (the screens the table feeds, admin order) plus `support` (B-N1-12, Go
// only): the support channels of the Support screen (tel: → phone line, http(s)/mailto → the chat card).
var InfoGroups = []string{"help", "privacy", "terms", "about", "support"}

// Info is GET /info/{group} (InfoController::show): unknown group → abort(404).
func (h *Handlers) Info(c fiber.Ctx) error {
	group := c.Params("group")
	if !slices.Contains(InfoGroups, group) {
		return httpx.NotFound()
	}
	return h.infoSections(c, group)
}

// Privacy is GET /privacy, the deprecated alias of /info/privacy.
func (h *Handlers) Privacy(c fiber.Ctx) error { return h.infoSections(c, "privacy") }

func (h *Handlers) infoSections(c fiber.Ctx, group string) error {
	// ?locale= wins only when it is exactly fa/en (the controller's hard-coded pair).
	locale, ok := i18n.QueryLocaleIn(c, i18n.LegacyPair...)
	if !ok {
		locale = i18n.ResolveLocale(c, "")
	}
	def := i18n.LanguagesOf(c).DefaultCode()
	rows, err := h.q.ListActiveInfoSections(c, group)
	if err != nil {
		return fmt.Errorf("content: info sections: %w", err)
	}
	sections := make([]*jsonx.OrderedMap, len(rows))
	for i, r := range rows {
		// InfoSection::toLocalizedArray: the link only when both halves are present.
		label, isStr := pickString(r.LinkLabel.V, locale, def)
		if !r.LinkLabel.Valid {
			isStr = false
		}
		var linkLabel, linkURL any
		if r.LinkUrl.Valid && phpTruthy(r.LinkUrl.String) && isStr && phpTruthy(label) {
			linkLabel = label
		}
		if isStr && phpTruthy(label) && r.LinkUrl.Valid {
			linkURL = r.LinkUrl.String
		}
		sections[i] = jsonx.Obj(
			"id", r.ID,
			"heading", phpValue(i18n.Pick(r.Heading, locale, def)),
			"body", phpValue(i18n.Pick(r.Body, locale, def)),
			"link_label", linkLabel,
			"link_url", linkURL,
		)
	}
	return httpx.OK(c, jsonx.Obj("group", group, "sections", jsonx.List(sections)))
}
