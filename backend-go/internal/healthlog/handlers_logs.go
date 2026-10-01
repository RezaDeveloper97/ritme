package healthlog

import (
	"github.com/gofiber/fiber/v3"

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

// TaxonomyNamespace is the translation namespace of the taxonomy labels.
const TaxonomyNamespace = "log-taxonomy"

// LogHandlers are the log taxonomy v2 endpoints (/api/v1/logs, Go only, B-N3-01). Mount them behind
// auth RequireUser and the locale middleware.
type LogHandlers struct {
	svc       *Service
	clock     clock.Clock
	bundles   *i18n.TranslationStore
	languages *i18n.Registry
}

// NewLogHandlers wires the v2 handlers.
func NewLogHandlers(svc *Service, base clock.Clock, bundles *i18n.TranslationStore, languages *i18n.Registry) *LogHandlers {
	return &LogHandlers{svc: svc, clock: base, bundles: bundles, languages: languages}
}

func (h *LogHandlers) user(c fiber.Ctx) (uint64, error) { return (&Handlers{}).user(c) }

func fieldFail(c fiber.Ctx, field, key string, params map[string]string) error {
	all := map[string]string{"attribute": field}
	for k, v := range params {
		all[k] = v
	}
	e := httpx.NewValidationError()
	e.Add(field, lang.Default().Trans(key, all, i18n.Locale(c)))
	return e
}

// Taxonomy is GET /logs/taxonomy[?mode=<mode>|all]: the categories, params and value sets with labels in
// the request language. Without mode: the user's current mode; `all`: everything, legacy-only values
// flagged.
func (h *LogHandlers) Taxonomy(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	mode, _ := validation.Query(c).Get("mode")
	m, _ := mode.(string)
	switch {
	case mode == nil:
		if m, err = h.svc.LifeMode(c.Context(), userID); err != nil {
			return err
		}
	case m == "all":
		m = ""
	case !taxonomy.IsMode(m):
		return fieldFail(c, "mode", "validation.in", nil)
	}
	locale := i18n.Locale(c)
	ns := h.bundles.NamespaceMessages(locale, TaxonomyNamespace, h.languages.DefaultCode(c.Context()))
	var modeOut any
	if m != "" {
		modeOut = m
	}
	return httpx.OK(c, jsonx.Obj(
		"mode", modeOut,
		"categories", taxonomy.CategoriesJSON(m, taxonomy.NewLabels(ns)),
	))
}

func dayJSON(date civildate.Date, entries []taxonomy.Entry) *jsonx.OrderedMap {
	return jsonx.Obj("date", date, "categories", taxonomy.DayJSON(entries))
}

// pathDate is the {date} segment as a real Y-m-d day.
func pathDate(c fiber.Ctx) (civildate.Date, error) {
	d, err := civildate.Parse(dateParam(c))
	if err != nil || d.String() != dateParam(c) {
		return civildate.Date{}, fieldFail(c, "date", "validation.date_format", map[string]string{"format": "Y-m-d"})
	}
	return d, nil
}

// Days is GET /logs/days?from=Y-m-d&to=Y-m-d: the days in range that have entries (at most 366 days).
func (h *LogHandlers) Days(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	q := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), q, validation.Rules{
		validation.F("from", "required", "date_format:Y-m-d"),
		validation.F("to", "required", "date_format:Y-m-d", "after_or_equal:from"),
	})
	if v.Fails() {
		return v.Errors()
	}
	fromV, _ := q.Get("from")
	toV, _ := q.Get("to")
	from, err1 := civildate.Parse(fromV.(string))
	to, err2 := civildate.Parse(toV.(string))
	if err1 != nil || err2 != nil {
		return fieldFail(c, "from", "validation.date_format", map[string]string{"format": "Y-m-d"})
	}
	if from.DiffDays(to) > MaxRangeDays || to.DiffDays(from) > MaxRangeDays {
		return fieldFail(c, "to", "validation.before_or_equal", map[string]string{"date": from.AddDays(MaxRangeDays).String()})
	}
	days, err := h.svc.Range(c.Context(), userID, from, to)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(days))
	for _, d := range days {
		out = append(out, dayJSON(d.Date, d.Entries))
	}
	return httpx.OK(c, jsonx.Obj("from", from, "to", to, "days", out))
}

// Day is GET /logs/days/{date}: the day in the PUT body shape (empty categories when nothing is logged).
func (h *LogHandlers) Day(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := pathDate(c)
	if err != nil {
		return err
	}
	entries, err := h.svc.Day(c.Context(), userID, date)
	if err != nil {
		return err
	}
	return httpx.OK(c, dayJSON(date, entries))
}

// Save is PUT /logs/days/{date} with {"categories": {cat: {param: value|null} | null}}: each param sent
// replaces the stored one (null clears it, a null category clears the category); params not sent are kept.
// Future days are refused. Returns the whole day.
func (h *LogHandlers) Save(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := pathDate(c)
	if err != nil {
		return err
	}
	now := clock.FromContext(c, h.clock).Now()
	if date.After(civildate.InTehran(now)) {
		return fieldFail(c, "date", "validation.before_or_equal", map[string]string{"date": "today"})
	}
	mode, err := h.svc.LifeMode(c.Context(), userID)
	if err != nil {
		return err
	}
	existing, err := h.svc.Day(c.Context(), userID, date)
	if err != nil {
		return err
	}
	custom, err := h.svc.CustomSet(c.Context(), userID)
	if err != nil {
		return err
	}
	input := validation.Input(c)
	changes, err := taxonomy.Parse(input, mode, i18n.Locale(c), existing, custom)
	if err != nil {
		return err
	}
	markVoice(input, changes)
	day, err := h.svc.SaveDay(c.Context(), userID, date, changes, i18n.ResolveLocale(c, ""), now)
	if err != nil {
		return err
	}
	return httpx.OK(c, dayJSON(date, day))
}

// markVoice sets source=voice on the changes listed in the optional "voice_params" (["cat.param", …]): the
// params whose new value the user confirmed from a voice-log suggestion (B-N3-05). Anything else in the list
// (not a changed param, not a string) is ignored — it only labels rows, it never widens what is saved.
func markVoice(input phpval.Map, changes []taxonomy.Change) {
	raw, ok := input.Get("voice_params")
	list, isList := raw.([]any)
	if !ok || !isList {
		return
	}
	voice := map[string]bool{}
	for _, v := range list {
		if s, isStr := v.(string); isStr {
			voice[s] = true
		}
	}
	for i := range changes {
		if voice[changes[i].Key()] && len(changes[i].Entries) > 0 && voiceFillable(changes[i]) {
			changes[i].Source = SourceVoice
		}
	}
}

// voiceFillable: only the param types a voice suggestion can fill (internal/voicelog) may carry source=voice —
// never free text (note, other meds) or reminder ids.
func voiceFillable(ch taxonomy.Change) bool {
	cat, ok := taxonomy.CategoryByCode(ch.Category)
	if !ok {
		return false
	}
	p, ok := cat.Param(ch.Param)
	if !ok || p.Dynamic && !p.Custom {
		return false
	}
	switch p.Type {
	case taxonomy.Single, taxonomy.Multi, taxonomy.Items, taxonomy.Number, taxonomy.Integer, taxonomy.Bool:
		return true
	}
	return false
}

// Destroy is DELETE /logs/days/{date}: removes the whole day (v2 entries and the legacy row).
func (h *LogHandlers) Destroy(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	date, err := pathDate(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteDay(c.Context(), userID, date); err != nil {
		return err
	}
	return httpx.OK(c, dayJSON(date, nil))
}
