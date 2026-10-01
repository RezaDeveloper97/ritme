package taxonomy

// Menopause daily log (CB-MENO-01, nbl_Meno_Log): the board's 13 symptoms in five groups, the bleeding
// choice and the triggers. They are ordinary taxonomy slots — general items the other modes share
// (hot_flashes, insomnia, fatigue, vaginal_dryness) plus menopause-only items in the same categories —
// so health_log_entries stays the one day log and cross-mode analysis keeps one code per symptom.
// The groups are presentation only (labels: log-taxonomy `presets.menopause.<group>`); see
// docs/canvas-build/menopause.md.

// Ref points at one loggable slot: an item of an items param, or a whole param when Item is "".
type Ref struct {
	Category, Param, Item string
}

// PresetGroup is one titled block of a mode preset.
type PresetGroup struct {
	Code string
	Refs []Ref
}

func sym(item string) Ref { return Ref{"symptoms", "general", item} }
func uro(item string) Ref { return Ref{"urogenital", "symptoms", item} }

// menopausePreset is the board order. Symptom rows take the levels no · mild · moderate · severe
// (joints: pain levels, «ندارم» = no entry).
var menopausePreset = []PresetGroup{
	{Code: "vasomotor", Refs: []Ref{sym("hot_flashes"), sym("night_sweats"), sym("palpitations")}},
	{Code: "sleep", Refs: []Ref{sym("insomnia")}},
	{Code: "mind", Refs: []Ref{sym("anxiety"), sym("irritability"), sym("low_mood"), sym("brain_fog")}},
	{Code: "body", Refs: []Ref{{"pain", "location", "joints"}, sym("fatigue")}},
	{Code: "urogenital", Refs: []Ref{uro("vaginal_dryness"), uro("bladder_symptoms"), uro("low_libido")}},
	{Code: "bleeding", Refs: []Ref{{"bleeding", "presence", ""}}},
	{Code: "triggers", Refs: []Ref{{"menopause", "triggers", ""}}},
}

// MenopausePreset returns the menopause log groups (do not modify).
func MenopausePreset() []PresetGroup { return menopausePreset }

// MenopauseSymptoms are the 13 symptom slots of the preset (the severity rows), in board order.
func MenopauseSymptoms() []Ref {
	var out []Ref
	for _, g := range menopausePreset {
		for _, r := range g.Refs {
			if r.Item != "" {
				out = append(out, r)
			}
		}
	}
	return out
}
