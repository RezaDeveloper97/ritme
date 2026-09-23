package content

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Languages is GET /languages (LanguageController::index).
func (h *Handlers) Languages(c fiber.Ctx) error {
	langs := i18n.LanguagesOf(c)
	list := make([]*jsonx.OrderedMap, len(langs))
	for i, l := range langs {
		list[i] = jsonx.Obj("code", l.Code, "name", l.Name, "english_name", l.EnglishName,
			"direction", l.Direction, "is_default", l.IsDefault)
	}
	return httpx.OK(c, jsonx.Obj("default", langs.DefaultCode(), "languages", jsonx.List(list)))
}

// Messages is GET /languages/{code}/messages (LanguageController::messages). Never 404:
// an unknown code resolves to the default language.
func (h *Handlers) Messages(c fiber.Ctx) error {
	langs := i18n.LanguagesOf(c)
	locale := langs.Resolve(c.Params("code"))
	return httpx.OK(c, jsonx.Obj("locale", locale, "direction", langs.Direction(locale),
		"messages", h.tr.Bundle(locale, langs.DefaultCode())))
}
