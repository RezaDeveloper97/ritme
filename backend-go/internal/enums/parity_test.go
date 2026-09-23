package enums

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Parity with the PHP runtime: testdata/php_enums.json is produced by testdata/dump_enums.php (plain
// `php`, no framework boot) and compared case by case with the generated and hand-written Go.

// enumAccessor is filled per enum by zz_generated_registry_test.go.
type enumAccessor struct {
	values      func() []string
	label       func(v, locale string) string
	description func(v, locale string) string
	icon        func(v string) string
	options     func(locale string) []Option
}

type phpDump struct {
	Enums map[string]struct {
		Values  []string                       `json:"values"`
		Methods map[string]map[string][]string `json:"methods"`
		PerCase map[string][]any               `json:"per_case"`
		Options map[string][]Option            `json:"options"`
	} `json:"enums"`
	Static map[string]json.RawMessage `json:"static"`
}

func loadDump(t *testing.T) phpDump {
	t.Helper()
	raw, err := os.ReadFile("testdata/php_enums.json")
	require.NoError(t, err)
	var d phpDump
	require.NoError(t, json.Unmarshal(raw, &d))
	return d
}

// perCase maps "<Enum>.<phpMethod>" to the Go port, returning a JSON-comparable value.
var perCase = map[string]func(v string) any{
	"CycleSubphase.fertilityLevelV11":        func(v string) any { return string(CycleSubphase(v).FertilityLevelV11()) },
	"CycleSubphase.fertilityLevel":           func(v string) any { return string(CycleSubphase(v).FertilityLevel()) },
	"CycleSubphase.canonical":                func(v string) any { return string(CycleSubphase(v).Canonical()) },
	"CyclePhase.subphases":                   func(v string) any { return anySlice(valuesOf(CyclePhase(v).Subphases())) },
	"MainPhase.legacyPhase":                  func(v string) any { p, ok := MainPhase(v).LegacyPhase(); return nullable(string(p), ok) },
	"CycleVariability.uncertaintyRange":      func(v string) any { return float64(CycleVariability(v).UncertaintyRange()) },
	"CycleWarning.requiresUserInput":         func(v string) any { return CycleWarning(v).RequiresUserInput() },
	"EnergyLevel.score":                      func(v string) any { return float64(EnergyLevel(v).Score()) },
	"ResolutionSource.isPredicted":           func(v string) any { return ResolutionSource(v).IsPredicted() },
	"DataStatus.priority":                    func(v string) any { return float64(DataStatus(v).Priority()) },
	"DataQualityFlag.excludesFromPrediction": func(v string) any { return DataQualityFlag(v).ExcludesFromPrediction() },
	"ClotsAmount.isPresent":                  func(v string) any { return ClotsAmount(v).IsPresent() },
	"DataSource.isActual":                    func(v string) any { return DataSource(v).IsActual() },
	"ReminderType.icon":                      func(v string) any { return ReminderType(v).Icon() },
	"RecommendationType.icon":                func(v string) any { return RecommendationType(v).Icon() },
	"TaskCategory.icon":                      func(v string) any { return TaskCategory(v).Icon() },
}

// handOptions covers enums whose PHP options() is not a plain map over cases().
var handOptions = map[string]func(string) []Option{
	"CycleSubphase": CycleSubphaseOptions,
}

func TestEnumsMatchPHP(t *testing.T) {
	d := loadDump(t)

	var php, goNames []string
	for name := range d.Enums {
		php = append(php, name)
	}
	for name := range generatedEnums {
		goNames = append(goNames, name)
	}
	sort.Strings(php)
	sort.Strings(goNames)
	require.Equal(t, php, goNames, "every PHP enum is generated, and nothing else")

	for name, want := range d.Enums {
		acc := generatedEnums[name]
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, want.Values, acc.values(), "values in PHP case order")

			for method, byLocale := range want.Methods {
				fn := acc.label
				if method == "description" {
					fn = acc.description
				}
				require.NotNil(t, fn, "%s() not generated", method)
				for locale, texts := range byLocale {
					for i, v := range want.Values {
						assert.Equal(t, texts[i], fn(v, locale), "%s(%q) for %s", method, locale, v)
					}
				}
			}
			assert.Equal(t, want.Methods["label"] != nil, acc.label != nil, "label() presence")
			assert.Equal(t, want.Methods["description"] != nil, acc.description != nil, "description() presence")

			for method, results := range want.PerCase {
				fn, ok := perCase[name+"."+method]
				require.True(t, ok, "PHP %s::%s() has no Go port registered in perCase", name, method)
				for i, v := range want.Values {
					assert.Equal(t, results[i], fn(v), "%s(%s)", method, v)
				}
			}
			assert.Equal(t, want.PerCase["icon"] != nil, acc.icon != nil, "icon() presence")

			if want.Options != nil {
				opts := acc.options
				if h, ok := handOptions[name]; ok {
					opts = h
				}
				require.NotNil(t, opts, "options() not ported")
				for locale, rows := range want.Options {
					assert.Equal(t, rows, opts(locale), "options(%q)", locale)
				}
			}
		})
	}
}

func TestStaticHelpersMatchPHP(t *testing.T) {
	d := loadDump(t)
	rows := func(key string) [][]any {
		var out [][]any
		require.NoError(t, json.Unmarshal(d.Static[key], &out), key)
		require.NotEmpty(t, out, key)
		return out
	}
	str := func(v any) string { s, _ := v.(string); return s } // PHP null → ""

	for _, r := range rows("CycleVariability::fromStdDev") {
		assert.Equal(t, r[1], string(CycleVariabilityFromStdDev(r[0].(float64))), "fromStdDev(%v)", r[0])
	}
	for _, r := range rows("BmiCategory::fromBmi") {
		assert.Equal(t, r[1], string(BmiCategoryFromBmi(r[0].(float64))), "fromBmi(%v)", r[0])
	}
	for _, r := range rows("RegularityStatus::fromCycleLengths") {
		lengths := []int{}
		for _, n := range r[0].([]any) {
			lengths = append(lengths, int(n.(float64)))
		}
		assert.Equal(t, r[1], string(RegularityStatusFromCycleLengths(lengths)), "fromCycleLengths(%v)", lengths)
	}
	for _, r := range rows("CyclePhase::subphaseValuesFor") {
		assert.Equal(t, r[1], anySlice(CyclePhaseSubphaseValuesFor(str(r[0]))), "subphaseValuesFor(%v)", r[0])
	}

	nullableLabel := map[string]func(v, locale string) (string, bool){
		"CyclePhase::labelFor":            CyclePhaseLabelFor,
		"CycleSubphase::labelFor":         CycleSubphaseLabelFor,
		"RecommendationTrigger::labelFor": RecommendationTriggerLabelFor,
	}
	for key, fn := range nullableLabel {
		for _, r := range rows(key) {
			got, ok := fn(str(r[0]), r[1].(string))
			assert.Equal(t, r[2], nullable(got, ok), "%s(%v, %v)", key, r[0], r[1])
		}
	}
	for _, r := range rows("RecommendationType::labelFor") {
		assert.Equal(t, r[2], RecommendationTypeLabelFor(str(r[0]), r[1].(string)), "labelFor(%v, %v)", r[0], r[1])
	}
	for _, r := range rows("RecommendationType::iconFor") {
		assert.Equal(t, r[1], RecommendationTypeIconFor(str(r[0])), "iconFor(%v)", r[0])
	}

	var list []string
	require.NoError(t, json.Unmarshal(d.Static["CycleSubphase::contentBacked"], &list))
	assert.Equal(t, list, valuesOf(CycleSubphaseContentBacked()))
	require.NoError(t, json.Unmarshal(d.Static["CyclePhase::allSubphases"], &list))
	assert.Equal(t, list, valuesOf(CyclePhaseAllSubphases()))
	require.NoError(t, json.Unmarshal(d.Static["RecommendationTrigger::activeFor(null)"], &list))
	assert.Equal(t, list, RecommendationTriggerActiveFor(nil))
}

func anySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func nullable(s string, ok bool) any {
	if !ok {
		return nil
	}
	return s
}
