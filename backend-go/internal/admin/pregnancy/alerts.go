package pregnancy

import (
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Alert rules are message_contents rows of group pregnancy_alert, one per rule and locale:
// {enabled, level, window_days, params, title, what_we_saw, how_sure, advice, actions, contact}.
// The behaviour half (enabled … params) is written to every locale's row so they never drift;
// the engine reads it from the default-language row (docs/pregnancy-v2/README.md § Stored shapes).

// ruleRows are one rule's rows keyed by locale, in row order.
type ruleRows struct {
	byLocale map[string]*store.MessageContent
	locales  []string
}

func (h *Handlers) alertRows(c fiber.Ctx) (map[string]*ruleRows, error) {
	rows, err := h.q.ListMessageContentsOfGroup(c.Context(), registry.AlertGroup)
	if err != nil {
		return nil, err
	}
	out := map[string]*ruleRows{}
	for i := range rows {
		r := &rows[i]
		rr := out[r.ItemKey]
		if rr == nil {
			rr = &ruleRows{byLocale: map[string]*store.MessageContent{}}
			out[r.ItemKey] = rr
		}
		rr.byLocale[r.Locale] = r
		rr.locales = append(rr.locales, r.Locale)
	}
	return out, nil
}

// behaviourRow is the row the behaviour is read from: the default language's, else the first.
func (rr *ruleRows) behaviourRow(defaultCode string) *store.MessageContent {
	if rr == nil || len(rr.locales) == 0 {
		return nil
	}
	if r := rr.byLocale[defaultCode]; r != nil {
		return r
	}
	return rr.byLocale[rr.locales[0]]
}

func decode(m *store.MessageContent) any {
	if m == nil {
		return nil
	}
	v, err := phpval.Decode(m.Payload)
	if err != nil {
		return nil
	}
	return v
}

// behaviourJSON is {enabled, level, window_days, params} typed by the rule's schema (params
// as an object even when empty).
func behaviourJSON(rule registry.AlertRule, payload any) *jsonx.OrderedMap {
	b := registry.Build(registry.AlertBehaviourFields(rule), payload)
	if payload == nil {
		b.Set("enabled", false)
		b.Set("level", nil)
		b.Set("window_days", nil)
	}
	return b
}

func textsJSON(payload any) *jsonx.OrderedMap {
	return registry.Build(registry.AlertTextFields(), payload)
}

func (h *Handlers) ruleJSON(c fiber.Ctx, rule registry.AlertRule, rr *ruleRows, full bool) *jsonx.OrderedMap {
	langs := i18n.LanguagesOf(c)
	br := rr.behaviourRow(langs.DefaultCode())
	payload := decode(br)
	o := jsonx.Obj("key", rule.Key, "configured", br != nil)
	b := behaviourJSON(rule, payload)
	for _, k := range b.Keys() {
		v, _ := b.Get(k)
		o.Set(k, v)
	}
	locales := []string{}
	var updated sql.NullTime
	if rr != nil {
		locales = rr.locales
		for _, l := range rr.locales {
			if u := rr.byLocale[l].UpdatedAt; u.Valid && (!updated.Valid || u.Time.After(updated.Time)) {
				updated = u
			}
		}
	}
	missing := []string{}
	for _, code := range langs.Codes() {
		if rr == nil || rr.byLocale[code] == nil {
			missing = append(missing, code)
		}
	}
	placeholders := rule.Placeholders
	if !full {
		title, _ := phpval.Get(payload, "title")
		o.Set("title", title)
		o.Set("locales", locales)
		o.Set("missing_locales", missing)
		o.Set("updated_at", httpadmin.Time(updated))
		return o
	}
	texts := jsonx.NewObject()
	rows := []*jsonx.OrderedMap{}
	if rr != nil {
		for _, l := range rr.locales {
			m := rr.byLocale[l]
			texts.Set(l, textsJSON(decode(m)))
			rows = append(rows, jsonx.Obj("id", m.ID, "locale", m.Locale, "is_active", m.IsActive,
				"is_approved", m.IsApproved, "updated_at", httpadmin.Time(m.UpdatedAt)))
		}
	}
	o.Set("texts", texts)
	o.Set("rows", rows)
	o.Set("missing_locales", missing)
	o.Set("params_schema", registry.Schema(rule.Params))
	o.Set("placeholders", placeholders)
	o.Set("updated_at", httpadmin.Time(updated))
	return o
}

// ListAlertRules is GET /pregnancy-alert-rules: every registered rule in registry order, with
// its behaviour, default-language title, the locales that have a row and the active ones that
// do not; `legend` points at the level legend's rows (edited in the messages screen).
func (h *Handlers) ListAlertRules(c fiber.Ctx) error {
	all, err := h.alertRows(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(registry.AlertRules))
	for _, rule := range registry.AlertRules {
		items = append(items, h.ruleJSON(c, rule, all[rule.Key], false))
	}
	legend := []*jsonx.OrderedMap{}
	if rr := all[registry.LegendKey]; rr != nil {
		for _, l := range rr.locales {
			legend = append(legend, jsonx.Obj("id", rr.byLocale[l].ID, "locale", l))
		}
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", items,
		"legend", jsonx.Obj("group", registry.AlertGroup, "item_key", registry.LegendKey, "rows", legend),
	))
}

// AlertRuleOptions is GET /pregnancy-alert-rules/options: levels, action keys, window bounds,
// the text fields and every rule's params schema + placeholders.
func (h *Handlers) AlertRuleOptions(c fiber.Ctx) error {
	rules := make([]*jsonx.OrderedMap, 0, len(registry.AlertRules))
	for _, r := range registry.AlertRules {
		rules = append(rules, jsonx.Obj("key", r.Key, "params_schema", registry.Schema(r.Params),
			"placeholders", r.Placeholders))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"levels", registry.AlertLevels,
		"actions", registry.AlertActions,
		"min_window_days", registry.MinWindowDays,
		"max_window_days", registry.MaxWindowDays,
		"text_fields", registry.Schema(registry.AlertTextFields()),
		"rules", rules,
	))
}

func findRule(c fiber.Ctx) (registry.AlertRule, error) {
	rule, ok := registry.FindAlertRule(c.Params("key"))
	if !ok {
		return registry.AlertRule{}, httpadmin.NotFound("Alert rule")
	}
	return rule, nil
}

// ShowAlertRule is GET /pregnancy-alert-rules/:key.
func (h *Handlers) ShowAlertRule(c fiber.Ctx) error {
	rule, err := findRule(c)
	if err != nil {
		return err
	}
	all, err := h.alertRows(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(c, rule, all[rule.Key], true)))
}

// UpdateAlertRule is PUT /pregnancy-alert-rules/:key {enabled, level, window_days, params{…},
// texts?: {code: {title, what_we_saw, how_sure, advice, actions[{key,label}], contact?}}}.
// The behaviour is required and written to every row of the rule; texts are per active
// language (the default language's are required while its row does not exist yet); a
// language without a row gets one (active, approved) when its texts are sent.
func (h *Handlers) UpdateAlertRule(c fiber.Ctx) error {
	rule, err := findRule(c)
	if err != nil {
		return err
	}
	all, err := h.alertRows(c)
	if err != nil {
		return err
	}
	rr := all[rule.Key]
	langs := i18n.LanguagesOf(c)
	def := langs.DefaultCode()
	in := validation.Input(c)

	behaviour := registry.AlertBehaviourFields(rule)
	rules := registry.Rules("", behaviour)
	rules = append(rules, validation.F("texts", "nullable|array"))
	checks := []form.Check{func(in phpval.Map, add form.Add) error {
		registry.Check("", behaviour, in, add, msgFunc(c))
		return nil
	}}
	var textCodes []string
	for _, code := range langs.Codes() {
		prefix := "texts." + code
		hasRow := rr != nil && rr.byLocale[code] != nil
		if !sent(in, prefix) && (hasRow || code != def) {
			continue
		}
		textCodes = append(textCodes, code)
		rules = append(rules, validation.F(prefix, "required|array"))
		rules = append(rules, registry.Rules(prefix+".", registry.AlertTextFields())...)
		checks = append(checks, func(in phpval.Map, add form.Add) error {
			if v, ok := phpval.Get(in, prefix); ok && phpval.IsArray(v) {
				registry.Check(prefix+".", registry.AlertTextFields(), v, add, msgFunc(c))
			}
			return nil
		})
	}
	if _, err := form.ValidateInput(c, in, rules, checks...); err != nil {
		return err
	}

	payloadFor := func(code string, stored any) []byte {
		merged := phpval.NewMap()
		for _, k := range registry.AlertBehaviourKeys {
			v, _ := phpval.Get(in, k)
			merged.Set(k, v)
		}
		texts := stored
		if v, ok := phpval.Get(in, "texts."+code); ok && v != nil {
			texts = v
		}
		for _, k := range registry.AlertTextKeys {
			v, _ := phpval.Get(texts, k)
			merged.Set(k, v)
		}
		return form.JSON(registry.Build(registry.AlertFields(rule), merged))
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	now := httpadmin.DBTime(httpadmin.Now(c))
	written := map[string]bool{}
	if rr != nil {
		for _, l := range rr.locales {
			m := rr.byLocale[l]
			if err := q.SetMessageContentPayload(c.Context(), store.SetMessageContentPayloadParams{
				Payload: payloadFor(l, decode(m)), Now: now, ID: m.ID,
			}); err != nil {
				return err
			}
			written[l] = true
		}
	}
	created := 0
	for _, code := range textCodes {
		if written[code] {
			continue
		}
		if _, err := q.InsertMessageContent(c.Context(), store.InsertMessageContentParams{
			MsgGroup: registry.AlertGroup, ItemKey: rule.Key, Locale: code,
			Label:   sql.NullString{String: registry.AlertGroup + " / " + rule.Key, Valid: true},
			Payload: payloadFor(code, nil), IsActive: true, IsApproved: true, Now: now,
		}); err != nil {
			return err
		}
		created++
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_alert_rule.update", "pregnancy_alert_rule", 0,
		slog.String("rule", rule.Key), slog.Int("rows_created", created))
	all, err = h.alertRows(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("rule", h.ruleJSON(c, rule, all[rule.Key], true)), "Alert rule saved.")
}

func msgFunc(c fiber.Ctx) func(rule, field string) string {
	return func(rule, field string) string { return form.Msg(c, rule, field) }
}
