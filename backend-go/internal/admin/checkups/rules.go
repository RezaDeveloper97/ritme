package checkups

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Limits and defaults of the catalog form.
const (
	MaxSteps              = 10  // prep_steps, guide_steps, finding_options items
	MaxCycleDay           = 45  // cycle_day_from / cycle_day_to
	MaxIntervalMonths     = 120 // interval_months / interval_months_max
	MaxAge                = 120
	MaxRemindLeadDays     = 365
	DefaultTone           = "neutral"
	DefaultRemindLeadDays = 7
)

// Categories an admin type can have (`custom` belongs to users' own checkups).
var Categories = []string{
	string(engine.CategoryMonthly), string(engine.CategorySixMonthly), string(engine.CategoryAnnual),
	string(engine.CategoryMultiYear), string(engine.CategoryAgeBased),
}

// Performers are checkup_types.performed_by values.
var Performers = []string{"self", "doctor", "lab", "dentist"}

// Tones are checkup_types.tone values (docs/checkups/README.md).
var Tones = []string{"rose", "violet", "amber", "teal", "green", "neutral"}

// Audiences are the values checkup_types.audiences may list: the life modes (enums.LifeMode, screen
// order). NULL / an empty list = every mode; otherwise only users whose resolved mode is listed see
// the row (CB-MENO-01b).
func Audiences() []string { return enums.LifeModeValues() }

// Icons are the names the frontend resolves (frontend/src/entities/checkup/model/icon.ts,
// the seeded names included). Unknown names would fall back by performer there; the admin
// form only offers these.
var Icons = []string{
	"ribbon", "breast", "shield", "shieldCheck", "flask", "blood", "tooth", "stetho",
	"heart", "calendar", "drop", "pill", "note",
}

// keyPattern: lowercase snake case, starting with a letter (the frontend routes on keys).
const keyPattern = "/^[a-z][a-z0-9_]*$/"

// findingKeyPattern: the keys stored in checkup_records.findings.
const findingKeyPattern = "/^[a-z][a-z0-9_]*$/"

// sent reports whether the raw input carries key with a non-null value.
func sent(in phpval.Map, key string) bool {
	v, ok := phpval.Get(in, key)
	return ok && v != nil
}

// allLanguages is a translatable field that needs text in every active language when
// required (the catalog is shown in each of them).
func allLanguages(field string, required bool, codes []string, maxLen int) validation.Rules {
	presence := "nullable"
	if required {
		presence = "required"
	}
	rules := validation.Rules{validation.F(field, presence+"|array")}
	for _, code := range codes {
		rules = append(rules, validation.F(field+"."+code, presence+"|string|max:"+strconv.Itoa(maxLen)))
	}
	return rules
}

// rules are the field rules. Optional texts (subtitle, why) are all-languages once sent.
func rules(in phpval.Map, codes []string, create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("key", "required|string|max:64", validation.Regex(keyPattern)))
	}
	r = append(r, allLanguages("title", true, codes, 255)...)
	r = append(r, allLanguages("subtitle", sent(in, "subtitle"), codes, 255)...)
	r = append(r, allLanguages("why", sent(in, "why"), codes, 2000)...)

	interval := "nullable|integer|min:1|max:" + strconv.Itoa(MaxIntervalMonths)
	age := "nullable|integer|min:0|max:" + strconv.Itoa(MaxAge)
	day := "nullable|integer|min:1|max:" + strconv.Itoa(MaxCycleDay)
	steps := "nullable|array|max:" + strconv.Itoa(MaxSteps)
	r = append(r,
		validation.F("category", "required", validation.In(Categories...)),
		validation.F("performed_by", "required", validation.In(Performers...)),
		validation.F("icon", "nullable", validation.In(Icons...)),
		validation.F("tone", "nullable", validation.In(Tones...)),
		validation.F("interval_months", "required|integer|min:1|max:"+strconv.Itoa(MaxIntervalMonths)),
		validation.F("interval_months_max", interval),
		validation.F("age_min", age),
		validation.F("age_max", age),
		validation.F("cycle_day_from", day),
		validation.F("cycle_day_to", day),
		validation.F("remind_lead_days", "nullable|integer|min:0|max:"+strconv.Itoa(MaxRemindLeadDays)),
		validation.F("prep_steps", steps),
		validation.F("prep_steps.*", "required|array"),
		validation.F("guide_steps", steps),
		validation.F("guide_steps.*", "required|array"),
		validation.F("guide_steps.*.title", "required|array"),
		validation.F("guide_steps.*.body", "required|array"),
		validation.F("finding_options", steps),
		validation.F("finding_options.*", "required|array"),
		validation.F("finding_options.*.key", "required|string|max:40", validation.Regex(findingKeyPattern)),
		validation.F("finding_options.*.exclusive", "nullable|boolean"),
		validation.F("finding_options.*.label", "required|array"),
	)
	for _, code := range codes {
		r = append(r,
			validation.F("prep_steps.*."+code, "required|string|max:500"),
			validation.F("guide_steps.*.title."+code, "required|string|max:120"),
			validation.F("guide_steps.*.body."+code, "required|string|max:1000"),
			validation.F("finding_options.*.label."+code, "required|string|max:120"),
		)
	}
	return append(r,
		validation.F("hide_in_pregnancy", "nullable"),
		validation.F("audiences", "nullable|array|max:"+strconv.Itoa(len(Audiences()))),
		validation.F("audiences.*", "required|string", validation.In(Audiences()...)),
		validation.F("is_active", "nullable"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("source_note", "nullable|string|max:1000"),
	)
}

// validate runs the rules plus the cross-field checks; cur is the stored row on update (absent
// optional fields keep their value, so the checks compare against it).
func (h *Handlers) validate(c fiber.Ctx, cur *store.CheckupType) (phpval.Map, error) {
	in := validation.Input(c)
	codes := i18n.LanguagesOf(c).Codes()
	return form.Validate(c, rules(in, codes, cur == nil), h.checks(c, cur)...)
}

// effective is the numeric value a field will have: the input when sent (ok=false for
// null/non-numeric), else the stored value on update.
func effective(in phpval.Map, key string, stored sql.NullInt16, cur *store.CheckupType) (float64, bool) {
	if v, present := phpval.Get(in, key); present {
		if v == nil || !phpval.IsNumeric(v) {
			return 0, false
		}
		return phpval.ToFloat(v), true
	}
	if cur == nil || !stored.Valid {
		return 0, false
	}
	return float64(stored.Int16), true
}

func (h *Handlers) checks(c fiber.Ctx, cur *store.CheckupType) []form.Check {
	var zero store.CheckupType
	st := &zero
	if cur != nil {
		st = cur
	}
	interval := sql.NullInt16{Int16: int16(st.IntervalMonths), Valid: cur != nil} //nolint:gosec // G115: ≤ 120
	gte := func(in phpval.Map, add form.Add, field string, fv sql.NullInt16, other string, ov sql.NullInt16) {
		hi, okHi := effective(in, field, fv, cur)
		lo, okLo := effective(in, other, ov, cur)
		if okHi && okLo && hi < lo {
			add(field, form.Msg(c, "validation.gte.numeric", field, "value", phpval.ToString(lo)))
		}
	}
	return []form.Check{
		// key: unique over every row (custom ones have none).
		func(in phpval.Map, add form.Add) error {
			if cur != nil {
				return nil
			}
			v, ok := phpval.Get(in, "key")
			s, isStr := v.(string)
			if !ok || !isStr || s == "" {
				return nil
			}
			found, err := h.q.AdminCheckupTypeKeyExists(c.Context(), sql.NullString{String: s, Valid: true})
			if err != nil {
				return err
			}
			if found {
				add("key", form.Msg(c, "validation.unique", "key"))
			}
			return nil
		},
		// Ranges: max ≥ min for the interval, the age window and the cycle window.
		func(in phpval.Map, add form.Add) error {
			gte(in, add, "interval_months_max", st.IntervalMonthsMax, "interval_months", interval)
			gte(in, add, "age_max", st.AgeMax, "age_min", st.AgeMin)
			gte(in, add, "cycle_day_to", st.CycleDayTo, "cycle_day_from", st.CycleDayFrom)
			return nil
		},
		// The cycle window needs both ends.
		func(in phpval.Map, add form.Add) error {
			_, from := effective(in, "cycle_day_from", st.CycleDayFrom, cur)
			_, to := effective(in, "cycle_day_to", st.CycleDayTo, cur)
			switch {
			case from && !to:
				add("cycle_day_to", form.Msg(c, "validation.required_with", "cycle_day_to",
					"values", httpadmin.AttributeName(c, "cycle_day_from")))
			case to && !from:
				add("cycle_day_from", form.Msg(c, "validation.required_with", "cycle_day_from",
					"values", httpadmin.AttributeName(c, "cycle_day_to")))
			}
			return nil
		},
		// Finding keys are distinct (records store them).
		func(in phpval.Map, add form.Add) error {
			v, _ := phpval.Get(in, "finding_options")
			if !phpval.IsArray(v) {
				return nil
			}
			keys, items := phpval.Entries(v)
			seen := map[string]bool{}
			for i, item := range items {
				k, _ := phpval.Get(item, "key")
				s, isStr := k.(string)
				if !isStr || s == "" {
					continue
				}
				if seen[s] {
					field := "finding_options." + keys[i] + ".key"
					add(field, form.Msg(c, "validation.distinct", field))
				}
				seen[s] = true
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Validated data → columns

// translatedObj is {code: text} over the active languages that have text (nil when none).
func translatedObj(v any, codes []string) *jsonx.OrderedMap {
	kv := make([]any, 0, 2*len(codes))
	for _, code := range codes {
		t, ok := phpval.Get(v, code)
		if !ok || t == nil {
			continue
		}
		if s := phpval.ToString(t); s != "" {
			kv = append(kv, code, s)
		}
	}
	if len(kv) == 0 {
		return nil
	}
	return jsonx.Obj(kv...)
}

// translated is translatedObj encoded for a JSON column (nil when no language has text).
func translated(v any, codes []string) json.RawMessage {
	obj := translatedObj(v, codes)
	if obj == nil {
		return nil
	}
	return form.JSON(obj)
}

func nullJSON(raw json.RawMessage) db.NullRawJSON {
	if raw == nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: raw, Valid: true}
}

// steps builds a JSON array column from a validated list (NULL when empty or null).
func steps(data phpval.Map, key string, item func(v any) any) db.NullRawJSON {
	v, _ := data.Get(key)
	if !phpval.IsArray(v) {
		return db.NullRawJSON{}
	}
	_, vals := phpval.Entries(v)
	out := make([]any, 0, len(vals))
	for _, x := range vals {
		out = append(out, item(x))
	}
	if len(out) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(out), Valid: true}
}

// buildParams maps the validated data onto the columns. On update (cur != nil) an optional
// field that was not sent keeps its stored value; key, sort_order, now and id are the caller's.
func buildParams(c fiber.Ctx, data phpval.Map, cur *store.CheckupType) store.UpdateAdminCheckupTypeParams {
	codes := i18n.LanguagesOf(c).Codes()
	keep := func(key string) bool {
		_, present := data.Get(key)
		return cur != nil && !present
	}
	text := func(key string, stored db.NullRawJSON) db.NullRawJSON {
		if keep(key) {
			return stored
		}
		v, _ := data.Get(key)
		return nullJSON(translated(v, codes))
	}
	int16Of := func(key string, stored sql.NullInt16) sql.NullInt16 {
		if keep(key) {
			return stored
		}
		return form.NullInt16(data, key)
	}
	list := func(key string, stored db.NullRawJSON, item func(v any) any) db.NullRawJSON {
		if keep(key) {
			return stored
		}
		return steps(data, key, item)
	}
	var st store.CheckupType
	if cur != nil {
		st = *cur
	}

	p := store.UpdateAdminCheckupTypeParams{
		Category:          httpadmin.String(data, "category"),
		Title:             translated(mustGet(data, "title"), codes),
		Subtitle:          text("subtitle", st.Subtitle),
		Why:               text("why", st.Why),
		PerformedBy:       httpadmin.String(data, "performed_by"),
		IntervalMonths:    uint16(form.Int(data, "interval_months", 1)), //nolint:gosec // G115: validated 1…120
		IntervalMonthsMax: int16Of("interval_months_max", st.IntervalMonthsMax),
		AgeMin:            int16Of("age_min", st.AgeMin),
		AgeMax:            int16Of("age_max", st.AgeMax),
		CycleDayFrom:      int16Of("cycle_day_from", st.CycleDayFrom),
		CycleDayTo:        int16Of("cycle_day_to", st.CycleDayTo),
		PrepSteps: list("prep_steps", st.PrepSteps, func(v any) any {
			return translatedObj(v, codes)
		}),
		GuideSteps: list("guide_steps", st.GuideSteps, func(v any) any {
			t, _ := phpval.Get(v, "title")
			b, _ := phpval.Get(v, "body")
			return jsonx.Obj("title", translatedObj(t, codes), "body", translatedObj(b, codes))
		}),
		FindingOptions: list("finding_options", st.FindingOptions, func(v any) any {
			k, _ := phpval.Get(v, "key")
			l, _ := phpval.Get(v, "label")
			kv := []any{"key", phpval.ToString(k)}
			if ex, _ := phpval.Get(v, "exclusive"); ex != nil && phpval.Truthy(ex) && ex != "false" {
				kv = append(kv, "exclusive", true)
			}
			return jsonx.Obj(append(kv, "label", translatedObj(l, codes))...)
		}),
		HideInPregnancy: httpadmin.Bool(data, "hide_in_pregnancy"),
		IsActive:        httpadmin.Bool(data, "is_active"),
	}

	if keep("audiences") {
		p.Audiences = st.Audiences
	} else {
		v, _ := data.Get("audiences")
		p.Audiences = audiences(v)
	}

	switch {
	case keep("icon"):
		p.Icon = st.Icon
	default:
		p.Icon = form.Str(data, "icon")
	}
	switch {
	case keep("tone"):
		p.Tone = st.Tone
	case form.Has(data, "tone"):
		p.Tone = httpadmin.String(data, "tone")
	default:
		p.Tone = DefaultTone
	}
	switch {
	case keep("remind_lead_days"):
		p.RemindLeadDays = st.RemindLeadDays
	default:
		p.RemindLeadDays = uint16(form.Int(data, "remind_lead_days", DefaultRemindLeadDays)) //nolint:gosec // G115: validated 0…365
	}
	switch {
	case keep("source_note"):
		p.SourceNote = st.SourceNote
	default:
		p.SourceNote = form.Str(data, "source_note")
	}
	return p
}

// audiences is the validated life modes, distinct, in the order sent (NULL when null or empty =
// every mode).
func audiences(v any) db.NullRawJSON {
	if !phpval.IsArray(v) {
		return db.NullRawJSON{}
	}
	_, vals := phpval.Entries(v)
	seen := map[string]bool{}
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		if s := phpval.ToString(x); s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(out), Valid: true}
}

func mustGet(data phpval.Map, key string) any {
	v, _ := data.Get(key)
	return v
}

// ---------------------------------------------------------------------------
// Reorder

// validateReorder checks {ids:[…]}: integers, each catalog id exactly once.
func validateReorder(c fiber.Ctx, catalog []uint64) ([]uint64, error) {
	known := make(map[uint64]bool, len(catalog))
	for _, id := range catalog {
		known[id] = true
	}
	var ids []uint64
	_, err := form.Validate(c, validation.Rules{
		validation.F("ids", "required|array"),
		validation.F("ids.*", "required|integer|min:1"),
	}, func(in phpval.Map, add form.Add) error {
		v, _ := phpval.Get(in, "ids")
		if !phpval.IsArray(v) {
			return nil
		}
		keys, vals := phpval.Entries(v)
		seen := map[uint64]bool{}
		for i, x := range vals {
			if x == nil || !phpval.IsNumeric(x) || phpval.ToFloat(x) < 1 {
				return nil // the rules report it
			}
			id := uint64(phpval.ToFloat(x))
			if seen[id] {
				field := "ids." + keys[i]
				add(field, form.Msg(c, "validation.distinct", field))
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) != len(catalog) {
			add("ids", form.Msg(c, "validation.in", "ids"))
			return nil
		}
		for _, id := range ids {
			if !known[id] {
				add("ids", form.Msg(c, "validation.in", "ids"))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}
