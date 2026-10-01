// Package taxonomy is the log taxonomy v2 (B-N3-01, Log_Taxonomy artboard): the categories, their
// params and value sets, the life-stage modes each one shows in, and the mapping between the legacy
// daily_health_logs columns and the v2 storage (health_log_entries, migration 00020).
//
// The taxonomy is code, the labels are data: every user-facing string is a UI string in the translation
// bundle (namespace `log-taxonomy`, resources/translations/<code>/log-taxonomy.json), so an admin-added
// language only needs a bundle. Later tasks extend the registry by appending categories, params, options
// or items with their own Modes (menopause items CB-MENO-01, condition programs CB-COND-01, custom
// items B-N3-02) — the storage never changes for that.
//
// Storage: one health_log_entries row per (user, day, category, param, item) slot:
//
//	single      item ""     value_code = option        multi       item = option  value_code "yes"
//	items       item = code value_code = level, value_num = optional score
//	number      item ""     value_num                  integer     item ""        value_num
//	text        item ""     value_text                 text_items  item = code    value_text
//	bool        item ""     value_code "yes" | "no"    link        not stored here (owned by another feature)
package taxonomy

import "slices"

// Type is how a param is entered and stored.
type Type string

// Param types.
const (
	Single    Type = "single"     // one option
	Multi     Type = "multi"      // a set of options
	Items     Type = "items"      // per item a level (+ optional 1–N score): pain locations, symptoms
	Number    Type = "number"     // decimal in [Min, Max], Scale places
	Integer   Type = "integer"    // whole number in [Min, Max]
	Text      Type = "text"       // free text up to MaxLen
	TextItems Type = "text_items" // per item a free text up to MaxLen
	Bool      Type = "bool"       // yes / no
	Link      Type = "link"       // read-only tile fed by another feature (kick counter, feeding, …)
)

// Life-stage modes (enums.LifeMode values).
const (
	ModeCycle      = "cycle"
	ModeTTC        = "ttc"
	ModePregnancy  = "pregnancy"
	ModePostpartum = "postpartum"
	ModeMenopause  = "menopause"
	ModeTeen       = "teen"
)

// AllModes are every life-stage mode, in screen order.
var AllModes = []string{ModeCycle, ModeTTC, ModePregnancy, ModePostpartum, ModeMenopause, ModeTeen}

// Mode sets used by the registry (the taxonomy table's «سیکل/TTC · بارداری · بعد زایمان» columns; teen
// follows cycle, menopause gets the general categories until CB-MENO-01 adds its own items).
var (
	all      = AllModes
	cycleish = []string{ModeCycle, ModeTTC, ModeMenopause, ModeTeen}
	fertile  = []string{ModeCycle, ModeTTC}
	preg     = []string{ModePregnancy}
	post     = []string{ModePostpartum}
)

func modes(sets ...[]string) []string {
	var out []string
	for _, s := range sets {
		for _, m := range s {
			if !slices.Contains(out, m) {
				out = append(out, m)
			}
		}
	}
	return out
}

// Stored value codes of bool params and of yes/no item levels.
const (
	Yes = "yes"
	No  = "no"
)

// Item levels.
var (
	painLevels    = []string{"mild", "moderate", "severe"}
	symptomLevels = []string{Yes, No, "mild", "moderate", "severe"}
	yesNo         = []string{Yes, No}
)

// Option is one value of a single/multi param or one item of an items/text_items param.
type Option struct {
	Code string
	// Modes limits the option to these modes (nil = wherever the param shows).
	Modes []string
	// LegacyOnly marks a value only legacy data carries: valid and labelled, never offered for new input.
	LegacyOnly bool
}

// Range is an inclusive numeric range.
type Range struct{ Min, Max float64 }

// Param is one loggable parameter of a category.
type Param struct {
	Code    string
	Type    Type
	Options []Option // single, multi, items, text_items (the items)
	Levels  []string // items: the level codes
	Score   *Range   // items: optional per-item score (pain 1–10)
	Range   *Range   // number, integer
	Scale   int      // number: decimal places
	Unit    string   // number, integer: unit code (labels: log-taxonomy.units)
	MaxLen  int      // text, text_items
	Dynamic bool     // items, text_items: item codes are free (custom items, care reminders, legacy keys)
	Custom  bool     // multi, items: hosts the user's custom items of the category (B-N3-02, CustomItemCode)
	Modes   []string // nil = the category's modes
	Detail  bool     // optional detail behind «جزئیات» (colour, clots, odour…)
	Alert   bool     // always discuss with a doctor in this mode set (bleeding in pregnancy)
	Source  string   // link: the feature that owns the data
}

// Category is one accordion section / quick tile.
type Category struct {
	Code   string
	Group  string // body | mind | lifestyle | measure | mode | other
	Modes  []string
	Params []Param
	// Conditions are per-mode availability notes (postpartum: "after_6_weeks").
	Conditions map[string]string
}

func opts(codes ...string) []Option {
	out := make([]Option, len(codes))
	for i, c := range codes {
		out[i] = Option{Code: c}
	}
	return out
}

func legacy(codes ...string) []Option {
	out := opts(codes...)
	for i := range out {
		out[i].LegacyOnly = true
	}
	return out
}

func only(m []string, codes ...string) []Option {
	out := opts(codes...)
	for i := range out {
		out[i].Modes = m
	}
	return out
}

func join(lists ...[]Option) []Option {
	var out []Option
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

var odorOptions = join(opts("normal", "changed"), legacy("slightly_unusual", "strong_unpleasant"))

// registry is the taxonomy, in the default category order (nbl_Log_Customize). B-N3-02 orders per user.
var registry = []Category{
	{Code: "bleeding", Group: "body", Modes: modes(cycleish, preg, post), Params: []Param{
		{Code: "flow", Type: Single, Options: opts("light", "medium", "heavy", "very_heavy"), Modes: modes(cycleish, preg), Alert: true},
		{Code: "spotting", Type: Bool, Modes: modes(cycleish, preg)},
		{Code: "color", Type: Single, Detail: true, Modes: cycleish,
			Options: join(opts("pink", "bright_red"), legacy("red"), opts("dark_red", "brown", "black"))},
		{Code: "clots", Type: Bool, Detail: true, Modes: modes(cycleish, post)},
		{Code: "clot_size", Type: Single, Detail: true, Modes: modes(cycleish, post),
			Options: join(opts("small", "large"), legacy("none", "medium"))},
		{Code: "odor", Type: Single, Detail: true, Options: odorOptions, Modes: modes(cycleish, post)},
		{Code: "lochia_amount", Type: Single, Options: opts("light", "medium", "heavy"), Modes: post},
		{Code: "lochia_color", Type: Single, Options: opts("red", "pink_brown", "yellow_white"), Modes: post},
	}},
	{Code: "pain", Group: "body", Modes: all, Params: []Param{
		{Code: "none", Type: Bool},
		{Code: "location", Type: Items, Levels: painLevels, Score: &Range{1, 10}, Options: join(
			opts("abdomen"),
			only(modes(cycleish, preg), "pelvis"),
			only(cycleish, "ovary"),
			opts("back", "head"),
			only(modes(cycleish, post), "breast"),
			only(preg, "leg"),
			only(post, "stitches"),
			only([]string{ModeMenopause}, "joints"),
		)},
		{Code: "relief", Type: Multi, Options: opts("heat", "painkiller", "rest", "light_exercise", "none")},
	}},
	{Code: "mood", Group: "mind", Modes: all, Params: []Param{
		{Code: "moods", Type: Multi, Custom: true, Options: opts(
			"calm", "happy", "energetic", "sensitive", "irritable", "sad", "anxious", "angry", "bored", "frustrated")},
		{Code: "weekly_checkin", Type: Link, Modes: post, Source: "postpartum_checkin"},
	}},
	{Code: "symptoms", Group: "body", Modes: all, Params: []Param{
		{Code: "digestive", Type: Items, Levels: symptomLevels, Options: join(
			opts("nausea", "bloating", "constipation", "diarrhea", "heartburn"))},
		{Code: "general", Type: Items, Levels: symptomLevels, Custom: true, Options: join(
			opts("fatigue", "dizziness", "breast_tenderness", "swelling", "hot_flashes", "chills", "insomnia"),
			only(preg, "leg_cramps"),
			only([]string{ModeMenopause}, "night_sweats", "brain_fog", "palpitations"),
		)},
	}},
	{Code: "discharge", Group: "body", Modes: modes(cycleish, preg), Params: []Param{
		{Code: "consistency", Type: Single, Options: opts("none", "sticky", "creamy", "watery", "egg_white")},
		{Code: "amount", Type: Single, Options: opts("light", "medium", "heavy")},
		{Code: "color", Type: Single, Detail: true, Options: opts("clear", "white", "yellow", "green", "gray", "pink_bloody")},
		{Code: "odor", Type: Single, Detail: true, Options: odorOptions},
		{Code: "symptoms", Type: Items, Levels: yesNo, Detail: true, Options: opts("itching", "burning")},
	}},
	{Code: "sleep", Group: "lifestyle", Modes: all, Params: []Param{
		{Code: "duration", Type: Single, Options: opts("0_3", "3_6", "6_9", "9_plus")},
		{Code: "quality", Type: Single, Options: opts("great", "good", "fair", "poor")},
	}},
	{Code: "appetite_energy", Group: "mind", Modes: all, Params: []Param{
		{Code: "energy", Type: Single, Options: join(opts("low", "medium", "high"), legacy("very_low", "very_high"))},
		{Code: "appetite", Type: Single, Options: opts("decreased", "normal", "increased")},
		{Code: "cravings", Type: Items, Levels: yesNo, Custom: true, Options: opts("sweet", "salty", "fatty", "carbs", "sour", "any")},
	}},
	{Code: "activity", Group: "lifestyle", Modes: all, Conditions: map[string]string{ModePostpartum: "after_6_weeks"}, Params: []Param{
		{Code: "types", Type: Multi, Custom: true, Options: opts(
			"walking", "running", "cycling", "gym", "yoga", "pilates", "swimming", "dance", "team_sport", "stretching",
			"pelvic_floor", "other")},
		{Code: "duration", Type: Integer, Range: &Range{1, 600}, Unit: "min"},
		{Code: "intensity", Type: Single, Options: join(opts("low", "medium"),
			only([]string{ModeCycle, ModeTTC, ModePostpartum, ModeMenopause, ModeTeen}, "high"))},
	}},
	{Code: "urogenital", Group: "body", Modes: all, Params: []Param{
		{Code: "symptoms", Type: Items, Levels: symptomLevels, Custom: true, Options: opts(
			"frequent_urination", "urination_burning", "urgency", "leakage", "vaginal_dryness", "vaginal_itching",
			"vaginal_burning", "odor_change")},
		{Code: "urination", Type: Single, Options: opts("increased", "decreased", "normal")},
	}},
	{Code: "sex", Group: "lifestyle", Modes: []string{ModeCycle, ModeTTC, ModePregnancy, ModePostpartum, ModeMenopause},
		Conditions: map[string]string{ModePregnancy: "optional", ModePostpartum: "after_6_weeks"}, Params: []Param{
			{Code: "desire", Type: Single, Options: opts("lower", "normal", "higher")},
			{Code: "intercourse", Type: Single, Options: opts("protected", "unprotected")},
			{Code: "symptoms", Type: Multi, Options: join(
				opts("dryness", "burning", "pain_during_intercourse", "bleeding_after_intercourse", "lubricant_use"),
				legacy("high_desire", "no_desire", "protected_intercourse", "unprotected_intercourse"))},
		}},
	{Code: "measurements", Group: "measure", Modes: all, Params: []Param{
		{Code: "weight", Type: Number, Range: &Range{20, 300}, Scale: 2, Unit: "kg"},
		{Code: "bbt", Type: Number, Range: &Range{35, 42}, Scale: 2, Unit: "celsius", Modes: fertile},
		{Code: "lh_test", Type: Single, Options: opts("negative", "faint", "positive"), Modes: fertile},
		{Code: "pregnancy_test", Type: Single, Options: opts("negative", "positive"), Modes: fertile},
		{Code: "heart_rate", Type: Integer, Range: &Range{30, 250}, Unit: "bpm", Modes: modes(preg, post, []string{ModeMenopause})},
		{Code: "bp_systolic", Type: Integer, Range: &Range{50, 300}, Unit: "mmhg", Modes: modes(preg, post, []string{ModeMenopause})},
		{Code: "bp_diastolic", Type: Integer, Range: &Range{30, 200}, Unit: "mmhg", Modes: modes(preg, post, []string{ModeMenopause})},
		{Code: "blood_sugar", Type: Number, Range: &Range{20, 600}, Scale: 1, Unit: "mg_dl", Modes: modes(preg, post)},
	}},
	{Code: "meds", Group: "other", Modes: all, Params: []Param{
		{Code: "taken", Type: Items, Levels: yesNo, Dynamic: true}, // care reminder ids (B-N3-03)
		{Code: "other", Type: TextItems, MaxLen: 255, Dynamic: true,
			Options: opts("painkillers", "hormonal_pills", "antibiotics", "supplements")},
	}},
	{Code: "skin_hair", Group: "body", Modes: all, Params: []Param{
		{Code: "symptoms", Type: Items, Levels: symptomLevels, Custom: true, Options: opts("acne", "oily_skin", "dry_skin", "hair_loss")},
	}},
	{Code: "breasts", Group: "body", Modes: post, Params: []Param{
		{Code: "symptoms", Type: Items, Levels: symptomLevels, Options: opts("engorgement", "nipple_pain", "redness")},
	}},
	{Code: "pregnancy", Group: "mode", Modes: preg, Params: []Param{
		{Code: "kicks", Type: Link, Source: "kick_counter"},
		{Code: "contractions", Type: Link, Source: "contraction_timer"},
	}},
	{Code: "baby", Group: "mode", Modes: post, Params: []Param{
		{Code: "feeding", Type: Link, Source: "feeding"},
		{Code: "pumping", Type: Link, Source: "feeding"},
		{Code: "diapers", Type: Link, Source: "diapers"},
		{Code: "baby_sleep", Type: Link, Source: "baby_sleep"},
	}},
	{Code: "note", Group: "other", Modes: all, Params: []Param{
		{Code: "text", Type: Text, MaxLen: 2000},
	}},
	{Code: "custom", Group: "other", Modes: all, Params: []Param{
		{Code: "items", Type: Items, Levels: yesNo, Dynamic: true, Custom: true}, // user's custom items (B-N3-02)
	}},
}

// Categories returns the registry (do not modify).
func Categories() []Category { return registry }

// CategoryByCode finds a category.
func CategoryByCode(code string) (*Category, bool) {
	for i := range registry {
		if registry[i].Code == code {
			return &registry[i], true
		}
	}
	return nil, false
}

// Param finds a param of the category.
func (c *Category) Param(code string) (*Param, bool) {
	for i := range c.Params {
		if c.Params[i].Code == code {
			return &c.Params[i], true
		}
	}
	return nil, false
}

// ParamModes are the modes the param shows in.
func (c *Category) ParamModes(p *Param) []string {
	if p.Modes != nil {
		return p.Modes
	}
	return c.Modes
}

// Available reports whether the param shows in mode.
func (c *Category) Available(p *Param, mode string) bool {
	return slices.Contains(c.Modes, mode) && slices.Contains(c.ParamModes(p), mode)
}

// Option finds an option / item of the param.
func (p *Param) Option(code string) (*Option, bool) {
	for i := range p.Options {
		if p.Options[i].Code == code {
			return &p.Options[i], true
		}
	}
	return nil, false
}

// OptionAvailable reports whether the option may be newly entered in mode (legacy-only values never are).
func (o *Option) OptionAvailable(mode string) bool {
	return !o.LegacyOnly && (o.Modes == nil || slices.Contains(o.Modes, mode))
}

// Storable reports whether values of the param live in health_log_entries.
func (p *Param) Storable() bool { return p.Type != Link }

// HasItems reports whether the param stores one row per item.
func (p *Param) HasItems() bool {
	return p.Type == Multi || p.Type == Items || p.Type == TextItems
}

// CustomParam is the param of the category that hosts the user's custom items (nil: none).
func (c *Category) CustomParam() *Param {
	for i := range c.Params {
		if c.Params[i].Custom {
			return &c.Params[i]
		}
	}
	return nil
}

// IsMode reports whether m is a known life-stage mode.
func IsMode(m string) bool { return slices.Contains(AllModes, m) }
