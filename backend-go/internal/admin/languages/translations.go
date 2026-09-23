package languages

import (
	"log/slog"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// namespaceOf is TranslationController::namespace: the requested namespace when it
// exists, else the first one ("common" when there are none).
func namespaceOf(requested string, namespaces []string) string {
	if slices.Contains(namespaces, requested) {
		return requested
	}
	if len(namespaces) > 0 {
		return namespaces[0]
	}
	return "common"
}

// Translations is GET /languages/:id/translations?namespace=: the language's own strings
// for one namespace next to the default language's (reference), one row per key of
// either, sorted by key.
func (h *Handlers) Translations(c fiber.Ctx) error {
	l, err := h.find(c)
	if err != nil {
		return err
	}
	langs := h.reg.All(c.Context())
	def := langs.DefaultCode()
	namespaces := h.bundles.Namespaces(def)
	ns := namespaceOf(c.Query("namespace"), namespaces)
	current := Flatten(h.bundles.Raw(l.Code, ns))
	reference := Flatten(h.bundles.Raw(def, ns))
	var keys []string
	for k := range reference {
		keys = append(keys, k)
	}
	for k := range current {
		if _, dup := reference[k]; !dup {
			keys = append(keys, k)
		}
	}
	SortKeys(keys)
	rows := make([]*jsonx.OrderedMap, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, jsonx.Obj("key", k, "value", current[k], "reference", reference[k]))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"language", languageJSON(l),
		"namespaces", jsonx.List(namespaces),
		"namespace", ns,
		"rows", rows,
		"default_code", def,
		"default_name", langs.Name(def),
		"is_default_locale", l.Code == def,
	))
}

// UpdateTranslations is PUT /languages/:id/translations {namespace, rows: [{key, value}]}:
// replaces the language's stored strings for the namespace (empty values are dropped so
// they fall back to the default language). Rows are a list, not a map, because keys
// contain dots.
func (h *Handlers) UpdateTranslations(c fiber.Ctx) error {
	l, err := h.find(c)
	if err != nil {
		return err
	}
	def := h.reg.DefaultCode(c.Context())
	namespaces := h.bundles.Namespaces(def)
	ns := namespaceOf(httpadmin.String(validation.Input(c), "namespace"), namespaces)
	data, err := form.Validate(c, validation.Rules{
		validation.F("rows", "nullable|array"),
		validation.F("rows.*.key", "required|string|max:255"),
		validation.F("rows.*.value", "nullable|string"),
	})
	if err != nil {
		return err
	}
	flat := map[string]string{}
	var order []string
	rows, _ := data.Get("rows")
	_, items := phpval.Entries(rows)
	for _, it := range items {
		key := phpval.ToString(mustGet(it, "key"))
		if _, dup := flat[key]; !dup {
			order = append(order, key)
		}
		flat[key] = phpval.ToString(mustGet(it, "value"))
	}
	if err := h.bundles.WriteNamespace(l.Code, ns, Unflatten(flat, order)); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.translations", "language", l.ID, slog.String("namespace", ns),
		slog.Int("rows", len(order)))
	return httpadmin.OK(c, jsonx.Obj("language", languageJSON(l), "namespace", ns, "saved", len(order)),
		"Translations saved.")
}

func mustGet(v any, key string) any {
	x, _ := phpval.Get(v, key)
	return x
}
