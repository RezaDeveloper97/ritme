package taxonomy

import (
	"slices"
	"strconv"
	"strings"
)

// Log preferences (B-N3-02, nbl_Log_Customize): per user and mode the category order, the hidden
// categories and up to MaxPinned quick tiles. Without saved preferences the order is the registry order
// and the tiles follow the mode and, in the cycle modes, today's cycle phase («ثبت سریع حداکثر ۸ کاشی، بر
// اساس حالت و فاز مرتب می‌شود», Log_Taxonomy rules).
//
// A tile key is a category code ("pain": the tile opens the category's section) or "category.param"
// ("measurements.weight", "pregnancy.kicks": the tile opens that param — the artboards' «وزن», «فشار
// خون», «حرکات جنین» tiles).

// MaxPinned is the most quick tiles a user can pin (nbl_Log_Customize: «حداکثر ۸»).
const MaxPinned = 8

// Cycle phases the tile defaults know (enums.MainPhase values).
const (
	PhaseMenstrual      = "menstrual"
	PhaseFollicular     = "follicular"
	PhaseFertile        = "fertile"
	PhaseLuteal         = "luteal"
	PhasePeriodExpected = "period_expected"
)

// PhaseModes are the modes whose default tiles follow the cycle phase.
var PhaseModes = []string{ModeCycle, ModeTTC, ModeTeen}

// defaultTiles: per mode, per phase ("" = no phase / unknown) the default quick tiles in order. The
// list is filtered by what is available in the mode and what the user hid, then topped up from fillTiles.
var defaultTiles = map[string]map[string][]string{
	ModeCycle: {
		// nbl_Log_Sheet_Cycle (day 25, luteal): پریود · درد · حال · علائم · ترشحات · خواب · وزن · یادداشت
		"":                  {"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"},
		PhaseLuteal:         {"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"},
		PhasePeriodExpected: {"bleeding", "pain", "mood", "symptoms", "sleep", "discharge", "measurements.weight", "note"},
		PhaseMenstrual:      {"bleeding", "pain", "mood", "symptoms", "sleep", "appetite_energy", "measurements.weight", "note"},
		PhaseFollicular:     {"mood", "appetite_energy", "activity", "discharge", "symptoms", "sleep", "measurements.weight", "note"},
		PhaseFertile:        {"discharge", "sex", "mood", "symptoms", "pain", "sleep", "measurements.weight", "note"},
	},
	ModeTTC: {
		"":                  {"measurements.bbt", "measurements.lh_test", "discharge", "sex", "bleeding", "mood", "symptoms", "note"},
		PhaseMenstrual:      {"bleeding", "pain", "measurements.bbt", "mood", "symptoms", "sleep", "discharge", "note"},
		PhaseFollicular:     {"measurements.bbt", "discharge", "measurements.lh_test", "sex", "mood", "symptoms", "sleep", "note"},
		PhaseFertile:        {"measurements.lh_test", "measurements.bbt", "discharge", "sex", "mood", "symptoms", "pain", "note"},
		PhaseLuteal:         {"measurements.bbt", "measurements.pregnancy_test", "symptoms", "mood", "bleeding", "sex", "discharge", "note"},
		PhasePeriodExpected: {"measurements.pregnancy_test", "measurements.bbt", "bleeding", "symptoms", "mood", "pain", "discharge", "note"},
	},
	// nbl_Log_Sheet_Preg: حرکات جنین · انقباض · علائم · حال · وزن · فشار خون · خواب · یادداشت
	ModePregnancy: {
		"": {"pregnancy.kicks", "pregnancy.contractions", "symptoms", "mood", "measurements.weight", "measurements.bp_systolic", "sleep", "note"},
	},
	// nbl_Log_Sheet_Post: شیردهی · خون‌ریزی · درد · حال روحی · خواب · وزن · دارو · یادداشت
	ModePostpartum: {
		"": {"baby.feeding", "bleeding", "pain", "mood", "sleep", "measurements.weight", "meds", "note"},
	},
	ModeMenopause: {
		"": {"symptoms", "sleep", "mood", "bleeding", "pain", "urogenital", "measurements.weight", "note"},
	},
}

// fillTiles tops up a default list that lost tiles to the mode filter (teen has no `sex`) or to hidden
// categories.
var fillTiles = []string{"bleeding", "pain", "mood", "symptoms", "sleep", "measurements.weight", "note",
	"appetite_energy", "skin_hair", "discharge", "activity", "urogenital", "meds"}

func tileModeTable(mode string) map[string][]string {
	if mode == ModeTeen {
		return defaultTiles[ModeCycle] // teen follows cycle (minus what teen does not show)
	}
	return defaultTiles[mode]
}

// TileCategory is the category code of a tile key.
func TileCategory(key string) string {
	cat, _, _ := strings.Cut(key, ".")
	return cat
}

// TileAvailable reports whether the tile key exists and shows in mode.
func TileAvailable(key, mode string) bool {
	catCode, paramCode, hasParam := strings.Cut(key, ".")
	cat, ok := CategoryByCode(catCode)
	if !ok || !slices.Contains(cat.Modes, mode) {
		return false
	}
	if !hasParam {
		return true
	}
	p, ok := cat.Param(paramCode)
	return ok && cat.Available(p, mode)
}

// TileKeys are every tile key available in mode: each category, then each of its params, registry order.
func TileKeys(mode string) []string {
	var out []string
	for i := range registry {
		c := &registry[i]
		if !slices.Contains(c.Modes, mode) {
			continue
		}
		out = append(out, c.Code)
		for j := range c.Params {
			if c.Available(&c.Params[j], mode) {
				out = append(out, c.Code+"."+c.Params[j].Code)
			}
		}
	}
	return out
}

// ModeCategories are the category codes that show in mode, registry order (the default order).
func ModeCategories(mode string) []string {
	var out []string
	for i := range registry {
		if slices.Contains(registry[i].Modes, mode) {
			out = append(out, registry[i].Code)
		}
	}
	return out
}

// DefaultTiles are the default quick tiles for mode and today's phase (ignored outside PhaseModes),
// without the hidden categories, at most MaxPinned.
func DefaultTiles(mode, phase string, hidden []string) []string {
	table := tileModeTable(mode)
	if !slices.Contains(PhaseModes, mode) {
		phase = ""
	}
	list, ok := table[phase]
	if !ok {
		list = table[""]
	}
	out := make([]string, 0, MaxPinned)
	add := func(keys []string) {
		for _, k := range keys {
			if len(out) == MaxPinned {
				return
			}
			if TileAvailable(k, mode) && !slices.Contains(hidden, TileCategory(k)) && !slices.Contains(out, k) {
				out = append(out, k)
			}
		}
	}
	add(list)
	add(fillTiles)
	return out
}

// Category is the label of a category in the request language (the code when missing).
func (l Labels) Category(code string) string { return l.get("categories."+code+".title", code) }

// CustomItemPrefix starts the item code of a custom item in health_log_entries.
const CustomItemPrefix = "custom_"

// CustomItemCode is the item code of custom item id ("custom_12").
func CustomItemCode(id uint64) string { return CustomItemPrefix + strconv.FormatUint(id, 10) }

// Stored is what a user saved for a mode; a nil list means "not customised" (the default applies).
type Stored struct {
	Order, Hidden, Pinned []string
}

// Effective is the preferences in force for a mode.
type Effective struct {
	Mode    string
	Phase   string   // today's phase used for the default tiles ("" outside PhaseModes or unknown)
	Order   []string // every category of the mode, in the user's order
	Hidden  []string // hidden categories, in Order order
	Pinned  []string // quick tiles, in order (≤ MaxPinned)
	Custom  struct{ Order, Hidden, Pinned bool }
	Default bool // nothing customised
}

// Resolve merges the stored preferences (nil: none) with the defaults: stored codes the mode no longer
// shows are dropped, categories the user never ordered (new ones) follow hers in registry order, and
// tiles of hidden categories are not shown.
func Resolve(mode, phase string, s *Stored) Effective {
	if !slices.Contains(PhaseModes, mode) {
		phase = ""
	}
	e := Effective{Mode: mode, Phase: phase}
	if s == nil {
		s = &Stored{}
	}
	all := ModeCategories(mode)
	e.Custom.Order, e.Custom.Hidden, e.Custom.Pinned = s.Order != nil, s.Hidden != nil, s.Pinned != nil
	e.Default = !e.Custom.Order && !e.Custom.Hidden && !e.Custom.Pinned

	e.Order = make([]string, 0, len(all))
	for _, c := range s.Order {
		if slices.Contains(all, c) && !slices.Contains(e.Order, c) {
			e.Order = append(e.Order, c)
		}
	}
	for _, c := range all {
		if !slices.Contains(e.Order, c) {
			e.Order = append(e.Order, c)
		}
	}
	e.Hidden = []string{}
	for _, c := range e.Order {
		if slices.Contains(s.Hidden, c) {
			e.Hidden = append(e.Hidden, c)
		}
	}
	if s.Pinned == nil {
		e.Pinned = DefaultTiles(mode, phase, e.Hidden)
		return e
	}
	e.Pinned = make([]string, 0, MaxPinned)
	for _, k := range s.Pinned {
		if len(e.Pinned) < MaxPinned && TileAvailable(k, mode) && !slices.Contains(e.Hidden, TileCategory(k)) &&
			!slices.Contains(e.Pinned, k) {
			e.Pinned = append(e.Pinned, k)
		}
	}
	return e
}
