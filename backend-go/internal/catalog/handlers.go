package catalog

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PublicGroups is the explicit allow-list of catalog groups readable WITHOUT a login (web SEO / guests): FAQ-type
// content only. Every other group (clinical copy, warning signs, kit items, score items, …) needs a signed-in user.
// Add a group here only after a product decision — groups are data, so an unlisted name is never public by accident.
var PublicGroups = map[string]bool{
	"teen_faq": true,
	"meno_faq": true,
}

// IsPublicGroup reports whether group may be read by a guest.
func IsPublicGroup(group string) bool { return PublicGroups[group] }

// Handlers serve the public catalog endpoint.
type Handlers struct {
	reader *Reader
}

// NewHandlers builds the handlers.
func NewHandlers(r *Reader) *Handlers { return &Handlers{reader: r} }

// Group is GET /api/v1/catalog/:group[?audience=]: the group's active items in display order, localized
// by the request locale. The route uses auth.OptionalUser: a guest may read only PublicGroups (others → 401, as
// for a missing token before); a bad token is a 401 everywhere. An unknown group is an empty list; a malformed name is a 404. With `audience`,
// only items for everyone or listing that audience are returned.
func (h *Handlers) Group(c fiber.Ctx) error {
	group := c.Params("group")
	if !ValidGroup(group) {
		return httpx.NotFound()
	}
	if !IsPublicGroup(group) && auth.CurrentUser(c) == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
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
