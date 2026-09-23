package pregnancy

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// IllustrationKeys are the fetus-size drawings the app bundles — exactly
// FETUS_ILLUSTRATION_KEYS in frontend/src/shared/ui/illustrations/fetus-keys.ts (an unknown
// key renders the neutral `fetus` drawing there).
var IllustrationKeys = []string{
	"fetus", "poppy_seed", "sesame_seed", "lentil", "blueberry", "raspberry", "cherry", "strawberry", "fig",
	"lime", "plum", "lemon", "peach", "apple", "orange", "avocado", "pear", "bell_pepper", "mango", "banana",
	"carrot", "papaya", "grapefruit", "corn", "cauliflower", "lettuce", "cabbage", "eggplant", "squash",
	"cucumber", "coconut", "pineapple", "cantaloupe", "honeydew", "watermelon", "pumpkin",
	"pomegranate", "butternut_squash", "melon", "romaine_lettuce", "swiss_chard", "leek", "small_watermelon",
}

// HighlightIcons are the icons of a week highlight — HIGHLIGHT_ICONS in
// frontend/src/entities/pregnancy/model/v2-types.ts.
var HighlightIcons = []string{"hand", "heart", "eye", "brain", "drop", "scale", "sparkle", "moon", "baby", "face"}

// HighlightTones are the optional highlight tints — HIGHLIGHT_TONES in the same file.
var HighlightTones = []string{"brand", "pink", "teal"}

// MaxWeekItems caps highlights, body_symptoms, tasks and sources.
const MaxWeekItems = 10

// measurePattern: ASCII measurements as seeded — "1.6", "<1", "150-170".
const measurePattern = `/^<?[0-9]{1,4}(\.[0-9]{1,2})?(-[0-9]{1,4}(\.[0-9]{1,2})?)?$/`

// weekDetailsJSON is one week (a missing row answers every field null / [] with exists=false).
func weekDetailsJSON(week int, d *store.PregnancyWeekDetail) *jsonx.OrderedMap {
	if d == nil {
		d = &store.PregnancyWeekDetail{}
	}
	return jsonx.Obj(
		"week_number", week,
		"exists", d.ID != 0,
		"size_label", form.NullRaw(d.SizeLabel),
		"illustration_key", httpadmin.NullString(d.IllustrationKey),
		"length_cm", httpadmin.NullString(d.LengthCm),
		"weight_g", httpadmin.NullString(d.WeightG),
		"heart_rate", httpadmin.NullString(d.HeartRate),
		"headline", form.NullRaw(d.Headline),
		"highlights", listOrEmpty(d.Highlights),
		"body_symptoms", listOrEmpty(d.BodySymptoms),
		"body_text", form.NullRaw(d.BodyText),
		"tasks", listOrEmpty(d.Tasks),
		"warning", form.NullRaw(d.Warning),
		"reviewer_name", form.NullRaw(d.ReviewerName),
		"reviewed_at", d.ReviewedAt,
		"sources", listOrEmpty(d.Sources),
		"created_at", httpadmin.Time(d.CreatedAt),
		"updated_at", httpadmin.Time(d.UpdatedAt),
	)
}

// weekParam reads :n (1–42); anything else is a 404.
func weekParam(c fiber.Ctx) (int, error) {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil || n < registry.MinWeek || n > registry.MaxWeek || strconv.Itoa(n) != c.Params("n") {
		return 0, httpadmin.NotFound("Pregnancy week")
	}
	return n, nil
}

func (h *Handlers) loadWeek(c fiber.Ctx, week int) (*store.PregnancyWeekDetail, error) {
	d, err := h.q.GetWeekDetails(c.Context(), uint8(week)) //nolint:gosec // G115: 1…42
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListWeekDetails is GET /pregnancy-week-details: one summary per week 1–42 for the editor's
// week picker ({week_number, exists, illustration_key, size_label, headline, reviewed_at,
// updated_at}).
func (h *Handlers) ListWeekDetails(c fiber.Ctx) error {
	rows, err := h.q.ListWeekDetails(c.Context())
	if err != nil {
		return err
	}
	byWeek := map[int]*store.PregnancyWeekDetail{}
	for i := range rows {
		byWeek[int(rows[i].WeekNumber)] = &rows[i]
	}
	items := make([]*jsonx.OrderedMap, 0, registry.MaxWeek)
	for w := registry.MinWeek; w <= registry.MaxWeek; w++ {
		d := byWeek[w]
		if d == nil {
			d = &store.PregnancyWeekDetail{}
		}
		items = append(items, jsonx.Obj(
			"week_number", w,
			"exists", d.ID != 0,
			"illustration_key", httpadmin.NullString(d.IllustrationKey),
			"size_label", form.NullRaw(d.SizeLabel),
			"headline", form.NullRaw(d.Headline),
			"reviewed_at", d.ReviewedAt,
			"updated_at", httpadmin.Time(d.UpdatedAt),
		))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

// WeekDetailsOptions is GET /pregnancy-week-details/options: the enums and limits of the form.
func (h *Handlers) WeekDetailsOptions(c fiber.Ctx) error {
	return httpadmin.OK(c, jsonx.Obj(
		"min_week", registry.MinWeek,
		"max_week", registry.MaxWeek,
		"illustration_keys", IllustrationKeys,
		"highlight_icons", HighlightIcons,
		"highlight_tones", HighlightTones,
		"log_symptom_keys", registry.LogSymptoms,
		"max_items", MaxWeekItems,
	))
}

// ShowWeekDetails is GET /pregnancy-weeks/:n/details.
func (h *Handlers) ShowWeekDetails(c fiber.Ctx) error {
	week, err := weekParam(c)
	if err != nil {
		return err
	}
	d, err := h.loadWeek(c, week)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("week_details", weekDetailsJSON(week, d)))
}

func weekRules(in phpval.Map, codes []string) validation.Rules {
	items := "nullable|array|max:" + strconv.Itoa(MaxWeekItems)
	r := validation.Rules{}
	r = append(r, translatable("size_label", sent(in, "size_label"), codes, 60)...)
	r = append(r,
		validation.F("illustration_key", "nullable|string", validation.In(IllustrationKeys...)),
		validation.F("length_cm", "nullable|string|max:20", validation.Regex(measurePattern)),
		validation.F("weight_g", "nullable|string|max:20", validation.Regex(measurePattern)),
		validation.F("heart_rate", "nullable|string|max:20", validation.Regex(measurePattern)),
	)
	r = append(r, translatable("headline", sent(in, "headline"), codes, 255)...)
	r = append(r,
		validation.F("highlights", items),
		validation.F("highlights.*", "required|array"),
		validation.F("highlights.*.icon", "nullable|string", validation.In(HighlightIcons...)),
		validation.F("highlights.*.tone", "nullable|string", validation.In(HighlightTones...)),
		validation.F("highlights.*.title", "required|array"),
		validation.F("highlights.*.body", "required|array"),
		validation.F("body_symptoms", items),
		validation.F("body_symptoms.*", "required|array"),
		validation.F("body_symptoms.*.key", "required|string|max:40", validation.Regex(keyPattern)),
		validation.F("body_symptoms.*.label", "required|array"),
		validation.F("tasks", items),
		validation.F("tasks.*", "required|array"),
		validation.F("tasks.*.key", "required|string|max:40", validation.Regex(keyPattern)),
		validation.F("tasks.*.text", "required|array"),
		validation.F("sources", items),
		validation.F("sources.*", "required|array"),
		validation.F("sources.*.title", "required|array"),
		validation.F("sources.*.url", "nullable|string|max:1000"),
		validation.F("reviewed_at", "nullable|date_format:Y-m-d|before_or_equal:today"),
	)
	for _, code := range codes {
		r = append(r,
			validation.F("highlights.*.title."+code, "required|string|max:120"),
			validation.F("highlights.*.body."+code, "required|string|max:1000"),
			validation.F("body_symptoms.*.label."+code, "required|string|max:60"),
			validation.F("tasks.*.text."+code, "required|string|max:255"),
			validation.F("sources.*.title."+code, "required|string|max:255"),
		)
	}
	r = append(r, translatable("body_text", sent(in, "body_text"), codes, 2000)...)
	r = append(r, translatable("warning", sent(in, "warning"), codes, 2000)...)
	r = append(r, translatable("reviewer_name", sent(in, "reviewer_name"), codes, 255)...)
	return r
}

func sourceURLs(c fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		v, _ := phpval.Get(in, "sources")
		if !phpval.IsArray(v) {
			return nil
		}
		keys, items := phpval.Entries(v)
		for i, item := range items {
			u, _ := phpval.Get(item, "url")
			if s, isStr := u.(string); isStr && s != "" && !registry.IsLink(s) {
				f := "sources." + keys[i] + ".url"
				add(f, form.Msg(c, "validation.url", f))
			}
		}
		return nil
	}
}

// UpdateWeekDetails is PUT /pregnancy-weeks/:n/details (upsert). An absent field keeps its
// stored value (null on a new row); null — or [] for a list — clears it.
func (h *Handlers) UpdateWeekDetails(c fiber.Ctx) error {
	week, err := weekParam(c)
	if err != nil {
		return err
	}
	cur, err := h.loadWeek(c, week)
	if err != nil {
		return err
	}
	if cur == nil {
		cur = &store.PregnancyWeekDetail{}
	}
	in := validation.Input(c)
	codes := i18n.LanguagesOf(c).Codes()
	data, err := form.ValidateInput(c, in, weekRules(in, codes),
		distinctKeys(c, "body_symptoms"), distinctKeys(c, "tasks"), sourceURLs(c))
	if err != nil {
		return err
	}

	has := func(k string) bool { _, ok := data.Get(k); return ok }
	text := func(k string, stored db.NullRawJSON) db.NullRawJSON {
		if !has(k) {
			return stored
		}
		v, _ := data.Get(k)
		return nullJSON(translatedObj(v, codes))
	}
	str := func(k string, stored sql.NullString) sql.NullString {
		if !has(k) {
			return stored
		}
		return form.Str(data, k)
	}
	list := func(k string, stored db.NullRawJSON, item func(v any) any) db.NullRawJSON {
		if !has(k) {
			return stored
		}
		v, _ := data.Get(k)
		_, vals := phpval.Entries(v)
		if len(vals) == 0 {
			return db.NullRawJSON{}
		}
		out := make([]any, 0, len(vals))
		for _, x := range vals {
			out = append(out, item(x))
		}
		return nullJSON(out)
	}
	field := func(v any, k string) any { x, _ := phpval.Get(v, k); return x }
	optStr := func(v any, k string) any {
		x := field(v, k)
		if x == nil || phpval.ToString(x) == "" {
			return nil
		}
		return phpval.ToString(x)
	}

	reviewed := cur.ReviewedAt
	if has("reviewed_at") {
		reviewed = civildate.NullDate{}
		if s := httpadmin.String(data, "reviewed_at"); s != "" {
			d, perr := civildate.Parse(s)
			if perr == nil {
				reviewed = civildate.NullDate{Date: d, Valid: true}
			}
		}
	}

	p := store.UpsertWeekDetailsParams{
		WeekNumber:      uint8(week), //nolint:gosec // G115: 1…42
		SizeLabel:       text("size_label", cur.SizeLabel),
		IllustrationKey: str("illustration_key", cur.IllustrationKey),
		LengthCm:        str("length_cm", cur.LengthCm),
		WeightG:         str("weight_g", cur.WeightG),
		HeartRate:       str("heart_rate", cur.HeartRate),
		Headline:        text("headline", cur.Headline),
		Highlights: list("highlights", cur.Highlights, func(v any) any {
			kv := []any{"icon", optStr(v, "icon")}
			if tone := optStr(v, "tone"); tone != nil {
				kv = append(kv, "tone", tone)
			}
			return jsonx.Obj(append(kv,
				"title", translatedObj(field(v, "title"), codes),
				"body", translatedObj(field(v, "body"), codes))...)
		}),
		BodySymptoms: list("body_symptoms", cur.BodySymptoms, func(v any) any {
			return jsonx.Obj("key", phpval.ToString(field(v, "key")), "label", translatedObj(field(v, "label"), codes))
		}),
		BodyText: text("body_text", cur.BodyText),
		Tasks: list("tasks", cur.Tasks, func(v any) any {
			return jsonx.Obj("key", phpval.ToString(field(v, "key")), "text", translatedObj(field(v, "text"), codes))
		}),
		Warning:      text("warning", cur.Warning),
		ReviewerName: text("reviewer_name", cur.ReviewerName),
		ReviewedAt:   reviewed,
		Sources: list("sources", cur.Sources, func(v any) any {
			return jsonx.Obj("title", translatedObj(field(v, "title"), codes), "url", optStr(v, "url"))
		}),
		Now: httpadmin.DBTime(httpadmin.Now(c)),
	}
	if err := h.q.UpsertWeekDetails(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_week_details.update", "pregnancy_week", uint64(week)) //nolint:gosec // G115: 1…42
	d, err := h.loadWeek(c, week)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("week_details", weekDetailsJSON(week, d)), "Week details saved.")
}
