package healthlog

import (
	"database/sql"
	"errors"
	"slices"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// customItemModel names the custom item in 404 bodies (framework ModelNotFound shape).
const customItemModel = `App\Models\HealthLogCustomItem`

// prefsMode is the mode a preferences request is about: ?mode=<mode>, else the user's current mode.
func (h *LogHandlers) prefsMode(c fiber.Ctx, userID uint64) (string, error) {
	v, _ := validation.Query(c).Get("mode")
	if v == nil {
		return h.svc.LifeMode(c.Context(), userID)
	}
	m, _ := v.(string)
	if !taxonomy.IsMode(m) {
		return "", fieldFail(c, "mode", "validation.in", nil)
	}
	return m, nil
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

func customItemJSON(r store.HealthLogCustomItem) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"code", taxonomy.CustomItemCode(r.ID),
		"category", r.Category,
		"param", r.Param,
		"label", r.Label,
		"created_at", nullDateTime(r.CreatedAt),
		"updated_at", nullDateTime(r.UpdatedAt),
		"deleted_at", nullDateTime(r.DeletedAt),
	)
}

func customItemsJSON(rows []store.HealthLogCustomItem) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		out = append(out, customItemJSON(r))
	}
	return out
}

// prefsJSON renders the preferences in force for mode with the category labels and the active custom
// items.
func (h *LogHandlers) prefsJSON(c fiber.Ctx, userID uint64, mode string) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(clock.FromContext(c, h.clock).Now())
	eff, err := h.svc.Preferences(c.Context(), userID, mode, today)
	if err != nil {
		return nil, err
	}
	items, err := h.svc.CustomItems(c.Context(), userID, true)
	if err != nil {
		return nil, err
	}
	ns := h.bundles.NamespaceMessages(i18n.Locale(c), TaxonomyNamespace, h.languages.DefaultCode(c.Context()))
	labels := taxonomy.NewLabels(ns)
	cats := make([]*jsonx.OrderedMap, 0, len(eff.Order))
	for _, code := range eff.Order {
		cat, _ := taxonomy.CategoryByCode(code)
		var customParam any
		if p := cat.CustomParam(); p != nil && cat.Available(p, mode) {
			customParam = p.Code
		}
		cats = append(cats, jsonx.Obj(
			"code", code,
			"label", labels.Category(code),
			"hidden", slices.Contains(eff.Hidden, code),
			"pinned", slices.Contains(eff.Pinned, code),
			"custom_param", customParam,
		))
	}
	var phase any
	if eff.Phase != "" {
		phase = eff.Phase
	}
	return jsonx.Obj(
		"mode", mode,
		"phase", phase,
		"is_default", eff.Default,
		"customized", jsonx.Obj("order", eff.Custom.Order, "hidden", eff.Custom.Hidden, "pinned", eff.Custom.Pinned),
		"max_pinned", taxonomy.MaxPinned,
		"pinned", eff.Pinned,
		"categories", cats,
		"max_custom_items", MaxCustomItems,
		"custom_items", customItemsJSON(items),
	), nil
}

// Preferences is GET /logs/preferences[?mode=]: order, visibility and quick tiles of the log sheet for the
// user's current mode (or mode), defaults filled in, plus her active custom items.
func (h *LogHandlers) Preferences(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	mode, err := h.prefsMode(c, userID)
	if err != nil {
		return err
	}
	data, err := h.prefsJSON(c, userID, mode)
	if err != nil {
		return err
	}
	return httpx.OK(c, data)
}

// listField reads an optional, nullable list of strings: set = the key was sent, list nil = null (reset).
func listField(body phpval.Map, key string) (list []string, set bool) {
	v, ok := body.Get(key)
	if !ok {
		return nil, false
	}
	items, _ := v.([]any)
	if v == nil {
		return nil, true
	}
	list = make([]string, 0, len(items))
	for _, it := range items {
		s, _ := it.(string)
		list = append(list, s)
	}
	return list, true
}

// SavePreferences is PUT /logs/preferences[?mode=] with {"order"?, "hidden"?, "pinned"?}: each list sent
// replaces the stored one (null resets it to the default), lists not sent are kept. Returns the
// preferences as GET does.
func (h *LogHandlers) SavePreferences(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	mode, err := h.prefsMode(c, userID)
	if err != nil {
		return err
	}
	body := validation.Input(c)
	cats := taxonomy.ModeCategories(mode)
	v := validation.Make(lang.Default(), i18n.Locale(c), body, validation.Rules{
		validation.F("order", "sometimes", "nullable", "list"),
		validation.F("order.*", "required", "string", validation.In(cats...)),
		validation.F("hidden", "sometimes", "nullable", "list"),
		validation.F("hidden.*", "required", "string", validation.In(cats...)),
		validation.F("pinned", "sometimes", "nullable", "list", "max:"+strconv.Itoa(taxonomy.MaxPinned)),
		validation.F("pinned.*", "required", "string", validation.In(taxonomy.TileKeys(mode)...)),
	})
	errs := httpx.NewValidationError()
	if v.Fails() {
		errs = v.Errors()
	}
	var ch PrefsChange
	ch.Order, ch.OrderSet = listField(body, "order")
	ch.Hidden, ch.HiddenSet = listField(body, "hidden")
	ch.Pinned, ch.PinnedSet = listField(body, "pinned")
	for _, f := range []struct {
		key  string
		list []string
	}{{"order", ch.Order}, {"hidden", ch.Hidden}, {"pinned", ch.Pinned}} {
		seen := map[string]bool{}
		for i, s := range f.list {
			if seen[s] {
				field := f.key + "." + strconv.Itoa(i)
				errs.Add(field, lang.Default().Trans("validation.distinct", map[string]string{"attribute": field}, i18n.Locale(c)))
			}
			seen[s] = true
		}
	}
	if !errs.Empty() {
		return errs
	}
	if err := h.svc.SavePreferences(c.Context(), userID, mode, ch, clock.FromContext(c, h.clock).Now()); err != nil {
		return err
	}
	data, err := h.prefsJSON(c, userID, mode)
	if err != nil {
		return err
	}
	return httpx.OK(c, data)
}

// ResetPreferences is DELETE /logs/preferences[?mode=]: back to the defaults of the mode. Custom items
// are kept. Returns the (default) preferences.
func (h *LogHandlers) ResetPreferences(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	mode, err := h.prefsMode(c, userID)
	if err != nil {
		return err
	}
	if err := h.svc.ResetPreferences(c.Context(), userID, mode); err != nil {
		return err
	}
	data, err := h.prefsJSON(c, userID, mode)
	if err != nil {
		return err
	}
	return httpx.OK(c, data)
}

// CustomItems is GET /logs/custom-items: every custom item of the user, deleted ones included (their
// labels for logged days), oldest first.
func (h *LogHandlers) CustomItems(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.CustomItems(c.Context(), userID, false)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("max_custom_items", MaxCustomItems, "custom_items", customItemsJSON(rows)))
}

// customHosts are the category codes that take custom items.
func customHosts() []string {
	var out []string
	for _, cat := range taxonomy.Categories() {
		if cat.CustomParam() != nil {
			out = append(out, cat.Code)
		}
	}
	return out
}

// labelRules: the "label" field is a required string of at most MaxCustomLabel characters (it is then
// trimmed and its whitespace collapsed, normalizedLabel).
func labelRules() validation.Field {
	return validation.F("label", "required", "string", "max:"+strconv.Itoa(MaxCustomLabel))
}

func (h *LogHandlers) customError(c fiber.Ctx, err error, id uint64) error {
	switch {
	case errors.Is(err, ErrCustomLimit):
		return fieldFail(c, "custom_items", "validation.max.array", map[string]string{
			"attribute": "custom items", "max": strconv.Itoa(MaxCustomItems)})
	case errors.Is(err, ErrCustomDuplicate):
		return fieldFail(c, "label", "validation.unique", nil)
	case errors.Is(err, ErrCustomNotFound):
		return httpx.ModelNotFound(customItemModel, id)
	}
	return err
}

func normalizedLabel(c fiber.Ctx, body phpval.Map) (string, error) {
	raw, _ := body.Get("label")
	s, _ := raw.(string)
	label := NormalizeLabel(s)
	if label == "" {
		return "", fieldFail(c, "label", "validation.required", nil)
	}
	return label, nil
}

// AddCustomItem is POST /logs/custom-items with {"category", "label"}: a new custom item in the
// category's custom param (custom → items, symptoms → general, mood → moods, …). 201 with the item; its
// `code` is the item code PUT /logs/days/{date} accepts in that param.
func (h *LogHandlers) AddCustomItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), body, validation.Rules{
		validation.F("category", "required", "string", validation.In(customHosts()...)),
		labelRules(),
	})
	if v.Fails() {
		return v.Errors()
	}
	label, err := normalizedLabel(c, body)
	if err != nil {
		return err
	}
	catV, _ := body.Get("category")
	cat, _ := taxonomy.CategoryByCode(catV.(string))
	row, err := h.svc.AddCustomItem(c.Context(), userID, cat.Code, cat.CustomParam().Code, label, clock.FromContext(c, h.clock).Now())
	if err != nil {
		return h.customError(c, err, 0)
	}
	return httpx.Created(c, customItemJSON(row))
}

// pathID is the {id} segment (a malformed id is simply not found).
func pathID(c fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, httpx.ModelNotFound(customItemModel, c.Params("id"))
	}
	return id, nil
}

// RenameCustomItem is PATCH /logs/custom-items/{id} with {"label"}: renames an active item of the user
// (404 for another user's, a deleted or an unknown id).
func (h *LogHandlers) RenameCustomItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, err := pathID(c)
	if err != nil {
		return err
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), body, validation.Rules{labelRules()})
	if v.Fails() {
		return v.Errors()
	}
	label, err := normalizedLabel(c, body)
	if err != nil {
		return err
	}
	row, err := h.svc.RenameCustomItem(c.Context(), userID, id, label, clock.FromContext(c, h.clock).Now())
	if err != nil {
		return h.customError(c, err, id)
	}
	return httpx.OK(c, customItemJSON(row))
}

// DeleteCustomItem is DELETE /logs/custom-items/{id}: soft-deletes an active item of the user (logged
// days keep it, new input refuses it). Returns the item with deleted_at.
func (h *LogHandlers) DeleteCustomItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, err := pathID(c)
	if err != nil {
		return err
	}
	row, err := h.svc.DeleteCustomItem(c.Context(), userID, id, clock.FromContext(c, h.clock).Now())
	if err != nil {
		return h.customError(c, err, id)
	}
	return httpx.OK(c, customItemJSON(row))
}
