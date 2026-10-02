package companions

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/admin/store"
	companionhome "github.com/ritme/backend-go/internal/companion/home"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Field limits of the companion_tip slots (= the registry schema).
const (
	NoteMaxLen  = 500
	TitleMaxLen = 255
	BodyMaxLen  = 1000
)

// Slot sources: where the text the panel shows today comes from (the order internal/companion/home resolves it).
const (
	SourceLocale  = "locale"           // a live row of this language
	SourceDefault = "default_language" // a live row of the default language
	SourceBuiltIn = "built_in"         // the embedded copy (companion/home lang files)
)

func noteKey(phase string) string       { return phase + "_note" }
func tipKey(phase string, n int) string { return phase + "_tip_" + strconv.Itoa(n) }

// phaseKeys are the item keys of one phase in registry order.
func phaseKeys(phase string) []string {
	keys := []string{noteKey(phase)}
	for i := 1; i <= registry.CompanionTipsPerPhase; i++ {
		keys = append(keys, tipKey(phase, i))
	}
	return keys
}

// tipRows indexes the companion_tip rows: item key → locale → row.
type tipRows map[string]map[string]*store.MessageContent

func (h *Handlers) loadTipRows(c fiber.Ctx) (tipRows, error) {
	rows, err := h.q.ListCompanionTipRows(c.Context())
	if err != nil {
		return nil, err
	}
	out := tipRows{}
	for i := range rows {
		r := &rows[i]
		if out[r.ItemKey] == nil {
			out[r.ItemKey] = map[string]*store.MessageContent{}
		}
		out[r.ItemKey][r.Locale] = r
	}
	return out, nil
}

func payloadText(r *store.MessageContent, field string) string {
	var p map[string]any
	if json.Unmarshal(r.Payload, &p) != nil {
		return ""
	}
	s, _ := p[field].(string)
	return strings.TrimSpace(s)
}

// builtIn is the embedded copy of a slot field in locale ("" when there is none).
func builtIn(key, field, locale string) string {
	k := "tips." + key
	if !strings.HasSuffix(key, "_note") {
		k += "." + field
	}
	s := companionhome.T(k, locale)
	if s == "companion_home."+k {
		return ""
	}
	return s
}

// resolve is a slot field as the companion panel shows it in locale, with its source: a live row of the locale,
// else a live row of the default language, else the embedded copy (internal/companion/home tipCopy.field).
func (rows tipRows) resolve(key, field, locale, deflt string) (string, string) {
	for _, l := range []string{locale, deflt} {
		if r := rows[key][l]; r != nil && r.IsActive && r.IsApproved {
			src := SourceLocale
			if l != locale {
				src = SourceDefault
			}
			return payloadText(r, field), src
		}
	}
	return builtIn(key, field, locale), SourceBuiltIn
}

// phaseJSON is one phase: per active language the note and the three tip slots as the panel shows them now, whether
// the language has its own rows (customized) and the rows themselves.
func (h *Handlers) phaseJSON(c fiber.Ctx, phase string, rows tipRows) *jsonx.OrderedMap {
	langs := i18n.LanguagesOf(c)
	deflt := langs.DefaultCode()
	texts := jsonx.NewObject()
	for _, code := range langs.Codes() {
		customized := false
		for _, k := range phaseKeys(phase) {
			if rows[k][code] != nil {
				customized = true
			}
		}
		note, noteSrc := rows.resolve(noteKey(phase), "body", code, deflt)
		tips := make([]*jsonx.OrderedMap, 0, registry.CompanionTipsPerPhase)
		for i := 1; i <= registry.CompanionTipsPerPhase; i++ {
			k := tipKey(phase, i)
			title, src := rows.resolve(k, "title", code, deflt)
			body, _ := rows.resolve(k, "body", code, deflt)
			tips = append(tips, jsonx.Obj("slot", i, "title", title, "body", body, "source", src))
		}
		texts.Set(code, jsonx.Obj(
			"customized", customized,
			"note", jsonx.Obj("body", note, "source", noteSrc),
			"tips", tips,
		))
	}
	rowList := []*jsonx.OrderedMap{}
	var updated any
	var latest string
	for _, k := range phaseKeys(phase) {
		byLocale := rows[k]
		locales := make([]string, 0, len(byLocale))
		for l := range byLocale {
			locales = append(locales, l)
		}
		slices.Sort(locales)
		for _, l := range locales {
			r := byLocale[l]
			rowList = append(rowList, jsonx.Obj("id", r.ID, "item_key", r.ItemKey, "locale", r.Locale,
				"is_active", r.IsActive, "is_approved", r.IsApproved, "updated_at", httpadmin.Time(r.UpdatedAt)))
			if t, ok := httpadmin.Time(r.UpdatedAt).(string); ok && t > latest {
				latest, updated = t, t
			}
		}
	}
	return jsonx.Obj("phase", phase, "texts", texts, "rows", rowList, "updated_at", updated)
}

// ListTips is GET /companions/tips: every phase in display order with the copy per active language, plus the
// limits and placeholders admin-web needs for the form.
func (h *Handlers) ListTips(c fiber.Ctx) error {
	rows, err := h.loadTipRows(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(registry.CompanionTipPhases))
	for _, p := range registry.CompanionTipPhases {
		items = append(items, h.phaseJSON(c, p, rows))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", items,
		"phases", registry.CompanionTipPhases,
		"tips_per_phase", registry.CompanionTipsPerPhase,
		"placeholders", []string{"name"},
		"limits", jsonx.Obj("note", NoteMaxLen, "title", TitleMaxLen, "body", BodyMaxLen),
		"default_locale", i18n.LanguagesOf(c).DefaultCode(),
	))
}

func findPhase(c fiber.Ctx) (string, error) {
	p := c.Params("phase")
	if !slices.Contains(registry.CompanionTipPhases, p) {
		return "", httpadmin.NotFound("Companion tip phase")
	}
	return p, nil
}

// ShowTips is GET /companions/tips/:phase.
func (h *Handlers) ShowTips(c fiber.Ctx) error {
	phase, err := findPhase(c)
	if err != nil {
		return err
	}
	rows, err := h.loadTipRows(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("tips", h.phaseJSON(c, phase, rows)))
}

// UpdateTips is PUT /companions/tips/:phase {texts: {code: {note: string|null, tips: [{title, body?}] (≤ 3)}}}.
// Each sent active language gets all four slots of the phase written (created when missing; live): the note
// (empty = no note) and the tips in the sent order, the unused slots emptied (an empty title hides the slot).
func (h *Handlers) UpdateTips(c fiber.Ctx) error {
	phase, err := findPhase(c)
	if err != nil {
		return err
	}
	langs := i18n.LanguagesOf(c)
	codes := langs.Codes()
	in := validation.Input(c)
	max := strconv.Itoa(registry.CompanionTipsPerPhase)
	rules := validation.Rules{validation.F("texts", "required|array")}
	for _, code := range codes {
		p := "texts." + code
		rules = append(rules,
			validation.F(p, "nullable|array"),
			validation.F(p+".note", "nullable|string|max:"+strconv.Itoa(NoteMaxLen)),
			validation.F(p+".tips", "nullable|array|max:"+max),
			validation.F(p+".tips.*", "required|array"),
			validation.F(p+".tips.*.title", "required|string|max:"+strconv.Itoa(TitleMaxLen)),
			validation.F(p+".tips.*.body", "nullable|string|max:"+strconv.Itoa(BodyMaxLen)),
		)
	}
	var sentCodes []string
	check := func(in phpval.Map, add form.Add) error {
		texts, ok := phpval.Get(in, "texts")
		if !ok || !phpval.IsArray(texts) {
			return nil
		}
		keys, vals := phpval.Entries(texts)
		for i, k := range keys {
			switch {
			case !slices.Contains(codes, k):
				add("texts."+k, form.Msg(c, "validation.in", "texts."+k))
			case vals[i] != nil:
				sentCodes = append(sentCodes, k)
			}
		}
		if len(keys) > 0 && len(sentCodes) == 0 {
			add("texts", form.Msg(c, "validation.required", "texts"))
		}
		return nil
	}
	if _, err := form.ValidateInput(c, in, rules, check); err != nil {
		return err
	}

	rows, err := h.loadTipRows(c)
	if err != nil {
		return err
	}
	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	now := httpadmin.DBTime(httpadmin.Now(c))
	created := 0
	write := func(key, locale string, payload *jsonx.OrderedMap, sort int) error {
		if r := rows[key][locale]; r != nil {
			return q.SetCompanionTipRow(c.Context(), store.SetCompanionTipRowParams{Payload: form.JSON(payload), Now: now, ID: r.ID})
		}
		created++
		return q.InsertCompanionTipRow(c.Context(), store.InsertCompanionTipRowParams{
			ItemKey: key, Locale: locale, Payload: form.JSON(payload), Now: now, SortOrder: uint32(sort), //nolint:gosec // G115: < 64
			Label: sqlNull(registry.CompanionTipGroup + " / " + key),
		})
	}
	base := slices.Index(registry.CompanionTipPhases, phase) * (registry.CompanionTipsPerPhase + 1)
	for _, code := range sentCodes {
		p := "texts." + code
		note, _ := phpval.Get(in, p+".note")
		if err := write(noteKey(phase), code, jsonx.Obj("body", trimmed(note)), base); err != nil {
			return err
		}
		tipsVal, _ := phpval.Get(in, p+".tips")
		_, tips := phpval.Entries(tipsVal)
		for i := 1; i <= registry.CompanionTipsPerPhase; i++ {
			title, body := "", ""
			if i <= len(tips) {
				t, _ := phpval.Get(tips[i-1], "title")
				b, _ := phpval.Get(tips[i-1], "body")
				title, body = trimmed(t), trimmed(b)
			}
			if err := write(tipKey(phase, i), code, jsonx.Obj("title", title, "body", body), base+i); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "companion_tip.update", "companion_tip", 0,
		slog.String("phase", phase), slog.String("locales", strings.Join(sentCodes, ",")), slog.Int("rows_created", created))
	rows, err = h.loadTipRows(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("tips", h.phaseJSON(c, phase, rows)), "Companion tips saved.")
}

// ResetTips is DELETE /companions/tips/:phase?locale=xx: removes the language's rows of the phase, so the panel
// falls back to the default language's rows, then to the built-in copy.
func (h *Handlers) ResetTips(c fiber.Ctx) error {
	phase, err := findPhase(c)
	if err != nil {
		return err
	}
	codes := i18n.LanguagesOf(c).Codes()
	in := phpval.NewMap()
	in.Set("locale", c.Query("locale"))
	if _, err := form.ValidateInput(c, in, validation.Rules{
		validation.F("locale", "required|string", validation.In(codes...)),
	}); err != nil {
		return err
	}
	locale := c.Query("locale")
	res, err := h.q.DeleteCompanionTipRows(c.Context(), store.DeleteCompanionTipRowsParams{Locale: locale, ItemKeys: phaseKeys(phase)})
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	httpadmin.Audit(c, h.logger, "companion_tip.reset", "companion_tip", 0,
		slog.String("phase", phase), slog.String("locale", locale), slog.Int64("rows_deleted", n))
	rows, err := h.loadTipRows(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("tips", h.phaseJSON(c, phase, rows)), "Companion tips reset.")
}

func sqlNull(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

func trimmed(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}
