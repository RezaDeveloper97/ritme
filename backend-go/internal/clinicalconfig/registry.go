// Package clinicalconfig is the versioned table of clinical thresholds (bloom B-N9-11, D-88): every key has a value
// (named numbers), a unit, a source, a version and the day it applies from. Version 1 of every key is the code-held
// default below — byte-identical with the constants the engines used before — and is never stored. Later versions
// are proposed in the admin API (internal/admin/clinicalcfg), need two distinct medical advisors' approvals (never
// the proposer) and apply from their effective date; the engines read the version valid at their target date, so
// historic outputs are reproducible with the thresholds of their day.
//
// This package is a leaf (no DB): the admin package installs a loader at startup; when none is installed, the store
// is unreachable or a key / field is missing, every getter answers the code default.
package clinicalconfig

// Groups (the artboard filter chips).
const (
	GroupEngine    = "engine"    // موتور چرخه
	GroupVitals    = "vitals"    // علائم حیاتی
	GroupPregnancy = "pregnancy" // بارداری / پس از زایمان
	GroupLab       = "lab"       // آزمایش
)

// Groups in display order.
var Groups = []string{GroupEngine, GroupVitals, GroupPregnancy, GroupLab}

// Field is one named number of a key's value with its code default (version 1).
type Field struct {
	Name    string
	Default float64
	Min     float64 // accepted range of a proposal
	Max     float64
}

// Def is a key of the table.
type Def struct {
	Key    string
	Group  string
	Unit   string
	Source string // version 1 source
	Fields []Field
	// Order lists field pairs that must stay strictly increasing ({"min","max"}).
	Order [][2]string
}

// Keys.
const (
	KeyBPElevated       = "bp.elevated"
	KeyBPStage1         = "bp.stage1"
	KeyBPStage2         = "bp.stage2"
	KeyBPCrisis         = "bp.crisis"
	KeyGlucoseUrgentLow = "glucose.urgent_low"
	KeyGlucoseLow       = "glucose.low"
	KeyGlucoseFasting   = "glucose.fasting"
	KeyGlucosePostMeal  = "glucose.post_meal"
	KeyHRResting        = "hr.resting"

	KeyEPDSLikely        = "epds.cutoff.likely"
	KeyEPDSPossible      = "epds.cutoff.possible"
	KeyEPDSShortElevated = "epds.cutoff.short_elevated"

	KeyFIGOFrequency  = "cycle.figo.frequency"
	KeyFIGODuration   = "cycle.figo.duration"
	KeyFIGORegularity = "cycle.figo.regularity"
	KeyPeriodHardCap  = "engine.period_hard_cap"
	KeyWarningCap     = "engine.warning_cap"
	KeyMaxLateDays    = "engine.max_late_days"
	KeyFertileWindow  = "engine.fertile_window"

	// Engine v6 (T-M9-15, DECISIONS.md X-11). engine.fertile_window.after stays for v1.1 and is ignored by v6
	// (retired in T-M9-51).
	KeyLuteal        = "engine.luteal"
	KeyBBT           = "engine.bbt"
	KeyLHBBTConflict = "engine.lh_bbt_conflict"
	KeyPregnancyTest = "engine.pregnancy_test"
	KeyStaleAnchor   = "engine.stale_anchor"
	KeyTeenFrequency = "cycle.teen.frequency"

	KeyFerritinLow = "lab.ferritin.low"

	// Pregnancy emergency paths (T-M9-05, D-134, DECISIONS.md H-02…H-06).
	KeyPregnancyBPHigh          = "pregnancy.bp.high"
	KeyPregnancyBPSevere        = "pregnancy.bp.severe"
	KeyPregnancyGDMTargets      = "pregnancy.gdm.targets"
	KeyPregnancyKicks           = "pregnancy.kicks"
	KeyPregnancyManualFetalWeek = "pregnancy.fetal_movement.manual_from_week"
	KeyPregnancyContractions511 = "pregnancy.contractions_511"
)

const (
	srcBP       = "ACC/AHA 2017 (Whelton PK et al., Hypertension 2018;71:e13–e115)"
	srcGlucose  = "ADA Standards of Care 2025 §2 / §6"
	srcHR       = "AHA — All About Heart Rate (resting 60–100 bpm)"
	srcEPDS     = "Cox JL, Holden JM, Sagovsky R. Br J Psychiatry 1987 (EPDS-3: Kabir 2008)"
	srcFIGO     = "FIGO 2018 (Munro MG et al., Int J Gynaecol Obstet 2018;143:393–408)"
	srcEngine   = "Ritme cycle engine v1.1 (task.md §2.12)"
	srcFertile  = "Wilcox AJ et al., NEJM 1995;333:1517–21"
	srcLab      = "WHO 2020 guideline on serum ferritin concentrations"
	srcEngineV6 = "Ritme cycle engine v6 (docs/engine-v6/SPEC.md Part 2)"

	srcPregBP     = "ACOG Practice Bulletin No. 222, Gestational Hypertension and Preeclampsia (Obstet Gynecol 2020;135:e237–60): ≥ 140/90 hypertension, ≥ 160/110 severe range"
	srcPregGDM    = "ADA Standards of Care 2024 §15 Management of Diabetes in Pregnancy (fasting < 95, 1 h < 140, 2 h < 120 mg/dL)"
	srcPregKicks  = "ACOG, Special Tests for Monitoring Fetal Well-Being (count to 10 within 2 hours, usually from week 28)"
	srcPregManual = "Ritme pregnancy alerts v1 (reduced / no fetal movement from week 24, DECISIONS.md H-04)"
	srcPreg511    = "ACOG, How to Tell When Labor Begins (5-1-1: every 5 min, about 1 min each, for 1 hour); duration 45 s, DECISIONS.md H-06"
)

func one(name string, def, lo, hi float64) []Field { return []Field{{name, def, lo, hi}} }

// Defs is the registry in display order. The defaults are the pre-B-N9-11 constants (vitals/thresholds.go,
// postpartum/epds.go, analysis FIGO, cycle resolver / view caps).
var Defs = []Def{
	{Key: KeyPeriodHardCap, Group: GroupEngine, Unit: "day", Source: srcEngine, Fields: one("days", 12, 7, 20)},
	{Key: KeyWarningCap, Group: GroupEngine, Unit: "day", Source: srcEngine,
		Fields: []Field{{"extra_days", 3, 0, 10}, {"min_days", 8, 3, 15}}},
	{Key: KeyMaxLateDays, Group: GroupEngine, Unit: "day", Source: srcEngine, Fields: one("days", 14, 3, 60)},
	{Key: KeyFertileWindow, Group: GroupEngine, Unit: "day", Source: srcFertile,
		Fields: []Field{{"before", 5, 1, 10}, {"after", 1, 0, 3}}},
	// Engine v6 (T-M9-15). luteal: §2.16 / E-16 BBT-confirmed luteal lengths in min..max count, else default.
	{Key: KeyLuteal, Group: GroupEngine, Unit: "day", Source: srcEngineV6,
		Fields: []Field{{"default", 14, 10, 16}, {"min", 10, 7, 14}, {"max", 16, 14, 20}},
		Order:  [][2]string{{"min", "default"}, {"default", "max"}}},
	// bbt: §2.17 / E-16 "3 over 6"; strict 1 = the third high reading must rise ≥ rise_centi (0.01 °C) over the max
	// of the previous 6; min_readings per cycle.
	{Key: KeyBBT, Group: GroupEngine, Unit: "flag / count / 0.01 °C", Source: srcEngineV6,
		Fields: []Field{{"strict", 1, 0, 1}, {"min_readings", 10, 6, 20}, {"rise_centi", 20, 5, 50}}},
	{Key: KeyLHBBTConflict, Group: GroupEngine, Unit: "day", Source: srcEngineV6, Fields: one("days", 2, 1, 5)},
	// pregnancy_test: §2.22 suggest_pregnancy_test from days_late ≥ cycle_days_late (cycle) / ttc_days_late (ttc).
	{Key: KeyPregnancyTest, Group: GroupEngine, Unit: "day", Source: srcEngineV6,
		Fields: []Field{{"cycle_days_late", 8, 1, 21}, {"ttc_days_late", 1, 0, 14}}},
	// stale_anchor: §2.41 the onboarding anchor is stale after effective_cycle_length + extra_days.
	{Key: KeyStaleAnchor, Group: GroupEngine, Unit: "day", Source: srcEngineV6, Fields: one("extra_days", 14, 0, 60)},
	{Key: KeyFIGOFrequency, Group: GroupEngine, Unit: "day", Source: srcFIGO,
		Fields: []Field{{"min", 24, 14, 40}, {"max", 38, 20, 60}}, Order: [][2]string{{"min", "max"}}},
	{Key: KeyFIGODuration, Group: GroupEngine, Unit: "day", Source: srcFIGO, Fields: one("max", 8, 3, 15)},
	{Key: KeyFIGORegularity, Group: GroupEngine, Unit: "day", Source: srcFIGO,
		Fields: []Field{{"young", 9, 2, 30}, {"adult", 7, 2, 30}, {"late", 9, 2, 30}}},
	// Teen frequency band (< 18, E-08): ACOG Committee Opinion 651 (21–45 days); frequent < min, infrequent > max.
	{Key: KeyTeenFrequency, Group: GroupEngine, Unit: "day", Source: srcEngineV6,
		Fields: []Field{{"min", 21, 14, 30}, {"max", 45, 35, 60}}, Order: [][2]string{{"min", "max"}}},

	{Key: KeyBPElevated, Group: GroupVitals, Unit: "mmHg", Source: srcBP, Fields: one("systolic", 120, 80, 200)},
	{Key: KeyBPStage1, Group: GroupVitals, Unit: "mmHg", Source: srcBP,
		Fields: []Field{{"systolic", 130, 80, 220}, {"diastolic", 80, 40, 140}}},
	{Key: KeyBPStage2, Group: GroupVitals, Unit: "mmHg", Source: srcBP,
		Fields: []Field{{"systolic", 140, 80, 240}, {"diastolic", 90, 40, 150}}},
	{Key: KeyBPCrisis, Group: GroupVitals, Unit: "mmHg", Source: srcBP,
		Fields: []Field{{"systolic", 180, 100, 300}, {"diastolic", 120, 60, 200}}},
	{Key: KeyGlucoseUrgentLow, Group: GroupVitals, Unit: "mg/dL", Source: srcGlucose, Fields: one("mgdl", 54, 20, 100)},
	{Key: KeyGlucoseLow, Group: GroupVitals, Unit: "mg/dL", Source: srcGlucose, Fields: one("mgdl", 70, 30, 120)},
	{Key: KeyGlucoseFasting, Group: GroupVitals, Unit: "mg/dL", Source: srcGlucose,
		Fields: []Field{{"target_max", 100, 60, 300}, {"very_high_from", 126, 60, 400}},
		Order:  [][2]string{{"target_max", "very_high_from"}}},
	{Key: KeyGlucosePostMeal, Group: GroupVitals, Unit: "mg/dL", Source: srcGlucose,
		Fields: []Field{{"target_max", 140, 60, 300}, {"very_high_from", 200, 60, 400}},
		Order:  [][2]string{{"target_max", "very_high_from"}}},
	{Key: KeyHRResting, Group: GroupVitals, Unit: "bpm", Source: srcHR,
		Fields: []Field{{"min", 60, 30, 120}, {"max", 100, 50, 200}}, Order: [][2]string{{"min", "max"}}},

	{Key: KeyEPDSLikely, Group: GroupPregnancy, Unit: "score", Source: srcEPDS, Fields: one("score", 13, 1, 30)},
	{Key: KeyEPDSPossible, Group: GroupPregnancy, Unit: "score", Source: srcEPDS, Fields: one("score", 10, 1, 30)},
	{Key: KeyEPDSShortElevated, Group: GroupPregnancy, Unit: "score", Source: srcEPDS, Fields: one("score", 6, 1, 9)},

	// Reference only: lab statuses compare with the range printed on the sheet or the admin catalog (B-N6-06), so no
	// engine reads this key yet; it is versioned here so the medical team can approve the catalog fallback value.
	// Pregnancy emergency paths (T-M9-05). pregnancy.bp.high.urgent: 1 = the urgent level the v2 rule has today, 0 =
	// follow_up (advisor decision, H-02); the severe tier is always urgent.
	{Key: KeyPregnancyBPHigh, Group: GroupPregnancy, Unit: "mmHg", Source: srcPregBP,
		Fields: []Field{{"systolic", 140, 120, 160}, {"diastolic", 90, 80, 110}, {"urgent", 1, 0, 1}}},
	{Key: KeyPregnancyBPSevere, Group: GroupPregnancy, Unit: "mmHg", Source: srcPregBP,
		Fields: []Field{{"systolic", 160, 140, 200}, {"diastolic", 110, 90, 140}}},
	{Key: KeyPregnancyGDMTargets, Group: GroupPregnancy, Unit: "mg/dL", Source: srcPregGDM,
		Fields: []Field{{"fasting", 95, 80, 126}, {"one_hour", 140, 120, 200}, {"two_hour", 120, 100, 200}}},
	{Key: KeyPregnancyKicks, Group: GroupPregnancy, Unit: "count / min / week", Source: srcPregKicks,
		Fields: []Field{{"target", 10, 4, 20}, {"window_min", 120, 30, 240}, {"from_week", 28, 20, 34}}},
	{Key: KeyPregnancyManualFetalWeek, Group: GroupPregnancy, Unit: "week", Source: srcPregManual,
		Fields: one("week", 24, 16, 32)},
	{Key: KeyPregnancyContractions511, Group: GroupPregnancy, Unit: "min / s / min", Source: srcPreg511,
		Fields: []Field{{"interval_min", 5, 2, 15}, {"duration_s", 45, 20, 120}, {"period_min", 60, 20, 180}}},

	{Key: KeyFerritinLow, Group: GroupLab, Unit: "ng/mL", Source: srcLab, Fields: one("ngml", 15, 1, 100)},
}

var byKey = func() map[string]*Def {
	m := make(map[string]*Def, len(Defs))
	for i := range Defs {
		m[Defs[i].Key] = &Defs[i]
	}
	return m
}()

// defaults caches version 1 of every key (shared: callers never mutate a Version's Value).
var defaults = func() map[string]Value {
	m := make(map[string]Value, len(Defs))
	for i := range Defs {
		m[Defs[i].Key] = Defs[i].DefaultValue()
	}
	return m
}()

// Lookup returns a key's definition.
func Lookup(key string) (*Def, bool) {
	d, ok := byKey[key]
	return d, ok
}

// DefaultValue is version 1 of a key (nil for an unknown key).
func (d *Def) DefaultValue() Value {
	v := make(Value, len(d.Fields))
	for _, f := range d.Fields {
		v[f.Name] = f.Default
	}
	return v
}

// Validate reports the first problem of a proposed value ("" = valid): every field present, in range, and ordered
// pairs strictly increasing. Unknown fields are rejected.
func (d *Def) Validate(v Value) string {
	if len(v) != len(d.Fields) {
		return "fields"
	}
	for _, f := range d.Fields {
		x, ok := v[f.Name]
		if !ok {
			return "fields"
		}
		if x < f.Min || x > f.Max {
			return "range:" + f.Name
		}
	}
	for _, p := range d.Order {
		if v[p[0]] >= v[p[1]] {
			return "order:" + p[0]
		}
	}
	return ""
}
