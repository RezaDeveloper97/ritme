package catalog

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Handlers serve the public catalog endpoint.
type Handlers struct {
	reader *Reader
}

// NewHandlers builds the handlers.
func NewHandlers(r *Reader) *Handlers { return &Handlers{reader: r} }

// Group is GET /api/v1/catalog/:group[?audience=]: the group's active items in display order, localized
// by the request locale. An unknown group is an empty list; a malformed name is a 404. With `audience`,
// only items for everyone or listing that audience are returned.
func (h *Handlers) Group(c fiber.Ctx) error {
	group := c.Params("group")
	if !ValidGroup(group) {
		return httpx.NotFound()
	}
	audience := strings.TrimSpace(c.Query("audience"))
	items, err := h.reader.Items(c.Context(), group)
	if err != nil {
		return err
	}
	langs := i18n.LanguagesOf(c)
	loc := Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
	out := make([]*jsonx.OrderedMap, 0, len(items))
	for _, it := range items {
		if it.For(audience) {
			out = append(out, loc.Public(it))
		}
	}
	return httpx.OK(c, jsonx.Obj(
		"group", group,
		"locale", loc.Locale,
		"items", jsonx.List(out),
	))
}
