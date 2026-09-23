package messages

import (
	"database/sql"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// T-M7-06: create rows only for registered (group, item_key) pairs, with the payload validated
// against the item's schema; the registry and the missing rows are listed for admin-web.

// msgFunc adapts form.Msg for registry.Check.
func msgFunc(c fiber.Ctx) func(rule, field string) string {
	return func(rule, field string) string { return form.Msg(c, rule, field) }
}

// PayloadRules are the rules + extra check of a registry item's payload under prefix.
func PayloadRules(c fiber.Ctx, item registry.Item, prefix string) (validation.Rules, form.Check) {
	rules := registry.Rules(prefix, item.Fields)
	check := func(in phpval.Map, add form.Add) error {
		v, ok := phpval.Get(in, trimDot(prefix))
		if prefix == "" {
			v, ok = in, true
		}
		if ok && phpval.IsArray(v) {
			registry.Check(prefix, item.Fields, v, add, msgFunc(c))
		}
		return nil
	}
	return rules, check
}

func trimDot(prefix string) string {
	if prefix != "" && prefix[len(prefix)-1] == '.' {
		return prefix[:len(prefix)-1]
	}
	return prefix
}

// typedPayload is PUT /messages/:id for a typed item: the stored payload with the sent top-level
// keys replaced, validated as a whole against the schema (errors on `payload.<field>`) and
// rebuilt in schema order (unknown keys dropped).
func typedPayload(c fiber.Ctx, item registry.Item, original, input any) (any, error) {
	merged := phpval.NewMap()
	for _, f := range item.Fields {
		if v, ok := phpval.Get(input, f.Key); ok && phpval.IsArray(input) {
			merged.Set(f.Key, v)
		} else if v, ok := phpval.Get(original, f.Key); ok {
			merged.Set(f.Key, v)
		}
	}
	in := phpval.NewMap()
	in.Set("payload", merged)
	rules, check := PayloadRules(c, item, "payload.")
	if _, err := form.ValidateInput(c, in, rules, check); err != nil {
		return nil, err
	}
	return registry.Build(item.Fields, merged), nil
}

// Store is POST /messages {group, item_key, locale, payload, label?, is_active?, is_approved?,
// sort_order?}: 201 {message}. Only a registered (group, item_key) in an active language; a
// row that exists already is a 422 unique on item_key. is_active / is_approved default to true.
func (h *Handlers) Store(c fiber.Ctx) error {
	in := validation.Input(c)
	group, _ := phpval.Get(in, "group")
	key, _ := phpval.Get(in, "item_key")
	groupName, _ := group.(string)
	keyName, _ := key.(string)
	item, known := registry.Lookup(groupName, keyName)
	codes := i18n.LanguagesOf(c).Codes()

	rules := validation.Rules{
		validation.F("group", "required|string", validation.In(registry.GroupNames()...)),
		validation.F("item_key", "required|string|max:255"),
		validation.F("locale", "required|string", validation.In(codes...)),
		validation.F("payload", "required|array"),
		validation.F("label", "nullable|string|max:255"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("is_approved", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:65535"),
	}
	checks := []form.Check{func(in phpval.Map, add form.Add) error {
		if keyName == "" || !slices.Contains(registry.GroupNames(), groupName) {
			return nil
		}
		if !known {
			add("item_key", form.Msg(c, "validation.in", "item_key"))
			return nil
		}
		locale, _ := phpval.Get(in, "locale")
		code, _ := locale.(string)
		if !slices.Contains(codes, code) {
			return nil
		}
		found, err := h.pq.MessageContentExists(c.Context(), pstore.MessageContentExistsParams{
			MsgGroup: groupName, ItemKey: keyName, Locale: code,
		})
		if err != nil {
			return err
		}
		if found {
			add("item_key", form.Msg(c, "validation.unique", "item_key"))
		}
		return nil
	}}
	if known {
		r, check := PayloadRules(c, item, "payload.")
		rules = append(rules, r...)
		checks = append(checks, check)
	}
	data, err := form.ValidateInput(c, in, rules, checks...)
	if err != nil {
		return err
	}
	payload, _ := phpval.Get(in, "payload")
	label := form.Str(data, "label")
	if !label.Valid {
		label = sql.NullString{String: groupName + " / " + keyName, Valid: true}
	}
	flag := func(k string) bool {
		if !form.Has(data, k) {
			return true
		}
		return httpadmin.Bool(data, k)
	}
	id, err := h.pq.InsertMessageContent(c.Context(), pstore.InsertMessageContentParams{
		MsgGroup: groupName, ItemKey: keyName, Locale: httpadmin.String(data, "locale"), Label: label,
		Payload:  form.JSON(registry.Build(item.Fields, payload)),
		IsActive: flag("is_active"), IsApproved: flag("is_approved"),
		SortOrder: uint32(form.Int(data, "sort_order", 0)), //nolint:gosec // G115: validated 0…65535
		Now:       httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	httpadmin.Audit(c, h.logger, "message.create", "message", uid)
	m, err := h.q.GetMessageContent(c.Context(), uid)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("message", messageJSON(&m)), "Message created.")
}

// missing lists the rows registered groups still lack: every registered item × active
// language without a row, optionally narrowed to one group / locale (the list filters).
func (h *Handlers) missing(c fiber.Ctx, group, locale string) ([]*jsonx.OrderedMap, error) {
	rows, err := h.pq.ListMessageContentKeys(c.Context())
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(rows))
	for _, r := range rows {
		have[r.Group+"\x00"+r.ItemKey+"\x00"+r.Locale] = true
	}
	codes := i18n.LanguagesOf(c).Codes()
	out := []*jsonx.OrderedMap{}
	for _, g := range registry.Groups() {
		if group != "" && g.Name != group {
			continue
		}
		for _, it := range g.Items {
			for _, code := range codes {
				if locale != "" && code != locale {
					continue
				}
				if !have[g.Name+"\x00"+it.Key+"\x00"+code] {
					out = append(out, jsonx.Obj("group", g.Name, "item_key", it.Key, "locale", code))
				}
			}
		}
	}
	return out, nil
}

// Registry is GET /messages/registry: {groups: [{group, typed, keys[]}]} — what POST /messages
// accepts.
func (h *Handlers) Registry(c fiber.Ctx) error {
	out := make([]*jsonx.OrderedMap, 0, len(registry.Groups()))
	for _, g := range registry.Groups() {
		typed := len(g.Items) > 0 && g.Items[0].Typed
		out = append(out, jsonx.Obj("group", g.Name, "typed", typed, "keys", jsonx.List(registry.Keys(g.Name))))
	}
	return httpadmin.OK(c, jsonx.Obj("groups", out))
}

// RegistryItem is GET /messages/registry/:group/:key?locale=: the item's schema, placeholders,
// the existing rows ({id, locale}) and a template payload to prefill a new row — the
// default-language row, else any row, else the code fallback copy in ?locale.
func (h *Handlers) RegistryItem(c fiber.Ctx) error {
	group, key := c.Params("group"), c.Params("key")
	item, ok := registry.Lookup(group, key)
	if !ok {
		return httpadmin.NotFound("Message item")
	}
	rows, err := h.pq.ListMessageContentsOfGroup(c.Context(), group)
	if err != nil {
		return err
	}
	langs := i18n.LanguagesOf(c)
	existing := []*jsonx.OrderedMap{}
	var template any
	for _, r := range rows {
		if r.ItemKey != key {
			continue
		}
		existing = append(existing, jsonx.Obj("id", r.ID, "locale", r.Locale))
		if template == nil || r.Locale == langs.DefaultCode() {
			template = form.Raw(r.Payload)
		}
	}
	if template == nil {
		locale := c.Query("locale")
		if locale == "" {
			locale = langs.DefaultCode()
		}
		if content.HasItem(group, key) {
			template = defaultCopy(group, key, locale)
		}
	}
	if template == nil {
		template = registry.Build(item.Fields, nil)
	}
	placeholders := item.Placeholders
	if placeholders == nil {
		placeholders = []string{}
	}
	return httpadmin.OK(c, jsonx.Obj(
		"group", group,
		"item_key", key,
		"typed", item.Typed,
		"fields", registry.Schema(item.Fields),
		"placeholders", placeholders,
		"existing", existing,
		"template", template,
	))
}

// defaultCopy is the code-fallback payload of an item built onto its derived fields.
func defaultCopy(group, key, locale string) any {
	item, _ := registry.Lookup(group, key)
	p := content.Default(group, key, locale)
	m := phpval.NewMap()
	for _, f := range item.Fields {
		if v, ok := p.Field(f.Key); ok {
			m.Set(f.Key, v)
		}
	}
	return registry.Build(item.Fields, m)
}
