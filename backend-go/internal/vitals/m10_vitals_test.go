package vitals

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M10-19 (SPEC v6 §3.2, §4.1, §6.1; D-170).

var specDay = civildate.MustParse("2026-11-01")

func specOn(t *testing.T) {
	t.Helper()
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyGlucoseContexts, Version: 2,
		Value: clinicalconfig.Value{"spec": 1}, EffectiveFrom: specDay}})
}

func class(mgdl float64, ctx, method string, d civildate.Date) string {
	return Reading{Type: TypeGlucose, Date: d, MgDl: mgdl, Context: ctx, Method: method}.Class()
}

func TestSPEC61_BloodPressure(t *testing.T) {
	clinicalconfig.Reset()
	for _, c := range []struct {
		sys, dia float64
		want     string
	}{
		{119, 79, ClassNormal}, {120, 79, ClassElevated}, {129, 79, ClassElevated}, {130, 79, ClassStage1},
		{129, 80, ClassStage1}, {139, 89, ClassStage1}, {140, 89, ClassStage2}, {180, 120, ClassStage2},
		{181, 100, ClassCrisis}, {150, 121, ClassCrisis},
	} {
		assert.Equal(t, c.want, Reading{Type: TypeBP, Systolic: c.sys, Diastolic: c.dia}.Class(), "%v/%v", c.sys, c.dia)
	}
}

func TestSPEC61_GlucoseFastingAndHypo(t *testing.T) {
	specOn(t)
	for _, d := range []civildate.Date{{}, specDay} { // unchanged by the spec flag
		for _, c := range []struct {
			mgdl float64
			want string
		}{
			{99, ClassInRange}, {100, ClassHigh}, {125, ClassHigh}, {126, ClassVeryHigh},
			{70, ClassInRange}, {69, ClassLow}, {54, ClassLow}, {53, ClassUrgentLow},
		} {
			assert.Equal(t, c.want, class(c.mgdl, ContextFasting, MethodGlucometer, d), "%v on %s", c.mgdl, d)
		}
	}
}

func TestGlucoseContexts_Spec0IsToday(t *testing.T) {
	clinicalconfig.Reset()
	d := civildate.MustParse("2026-10-22")
	for _, c := range []struct {
		mgdl float64
		want string
	}{{139, ClassInRange}, {140, ClassHigh}, {199, ClassHigh}, {200, ClassVeryHigh}} {
		assert.Equal(t, c.want, class(c.mgdl, ContextAfterMeal, MethodGlucometer, d), "post-meal %v", c.mgdl)
		assert.Equal(t, c.want, class(c.mgdl, ContextRandom, "", d), "random %v", c.mgdl)
		assert.Equal(t, c.want, class(c.mgdl, ContextBedtime, "", d), "bedtime %v", c.mgdl)
	}
	assert.False(t, ThresholdsOn(d).GlucoseContextsSpec)
}

func TestGlucoseContexts_Spec1(t *testing.T) {
	specOn(t)
	assert.True(t, ThresholdsOn(specDay).GlucoseContextsSpec)
	assert.Equal(t, ClassHigh, class(140, ContextAfterMeal, "", specDay.AddDays(-1)), "the day before keeps spec 0")

	assert.Equal(t, ClassInRange, class(139, ContextAfterMeal, MethodGlucometer, specDay))
	assert.Equal(t, ClassDiscuss, class(140, ContextAfterMeal, MethodGlucometer, specDay))
	assert.Equal(t, ClassDiscuss, class(250, ContextAfterMeal, MethodGlucometer, specDay), "no 200 class")
	assert.Equal(t, ClassDiscuss, class(250, ContextAfterMeal, "", specDay))
	assert.Equal(t, ClassVeryHigh, class(250, ContextAfterMeal, MethodLab, specDay), "a lab 2-h value keeps 200")
	for _, ctx := range []string{ContextRandom, ContextBedtime} {
		assert.Empty(t, class(199, ctx, MethodGlucometer, specDay), ctx)
		assert.Empty(t, class(120, ctx, MethodGlucometer, specDay), ctx)
		assert.Equal(t, ClassVeryHigh, class(200, ctx, MethodGlucometer, specDay), ctx)
		assert.Equal(t, ClassLow, class(60, ctx, "", specDay), "hypo wins: "+ctx)
		assert.Equal(t, ClassUrgentLow, class(50, ctx, "", specDay), "hypo wins: "+ctx)
	}
	assert.Equal(t, ClassUrgentLow, class(50, ContextAfterMeal, "", specDay))
	assert.Equal(t, ToneWatch, Tone(TypeGlucose, ClassDiscuss))

	// Pregnancy (T-M9-05) is unchanged: the GDM table still decides after a meal.
	preg := Reading{Type: TypeGlucose, Date: specDay, MgDl: 125, Context: ContextAfterMeal, Pregnant: true}
	assert.Equal(t, ClassHigh, preg.Class())
}

func diab(mgdl float64, ctx string, targets Targets) Reading {
	return Reading{Type: TypeGlucose, Date: civildate.MustParse("2026-10-22"), MgDl: mgdl, Context: ctx,
		Diabetic: true, Targets: targets}
}

func TestDiabeticTargets(t *testing.T) {
	clinicalconfig.Reset()
	for _, ctx := range []string{ContextFasting, ContextBeforeMeal} {
		for _, c := range []struct {
			mgdl float64
			want string
		}{{79, ClassBelowTarget}, {80, ClassInTarget}, {130, ClassInTarget}, {131, ClassAboveTarget},
			{69, ClassLow}, {53, ClassUrgentLow}} {
			assert.Equal(t, c.want, diab(c.mgdl, ctx, nil).Class(), "%s %v", ctx, c.mgdl)
		}
	}
	assert.Equal(t, ClassInTarget, diab(179, ContextAfterMeal, nil).Class())
	assert.Equal(t, ClassAboveTarget, diab(180, ContextAfterMeal, nil).Class())
	assert.Equal(t, ClassInTarget, diab(75, ContextAfterMeal, nil).Class(), "no post-meal floor above the low cut-off")
	assert.Equal(t, ClassInRange, diab(120, ContextRandom, nil).Class(), "random never has a target")
	assert.Equal(t, ClassInRange, diab(120, ContextBedtime, nil).Class(), "no bedtime target without a row")

	// The doctor's override (90–140) replaces the registry band.
	over := Targets{TargetFasting: {Context: TargetFasting, Low: 90, High: 140, SetBy: SetByDoctor},
		TargetBedtime:  {Context: TargetBedtime, Low: 100, High: 150, SetBy: SetByUser},
		TargetPostMeal: {Context: TargetPostMeal, High: 160, SetBy: SetByDoctor}}
	for _, c := range []struct {
		mgdl float64
		want string
	}{{89, ClassBelowTarget}, {90, ClassInTarget}, {140, ClassInTarget}, {141, ClassAboveTarget}} {
		assert.Equal(t, c.want, diab(c.mgdl, ContextFasting, over).Class(), "override %v", c.mgdl)
	}
	assert.Equal(t, ClassAboveTarget, diab(131, ContextBeforeMeal, over).Class(), "pre-meal keeps the registry band")
	assert.Equal(t, ClassAboveTarget, diab(160, ContextAfterMeal, over).Class())
	assert.Equal(t, ClassInTarget, diab(159, ContextAfterMeal, over).Class())
	assert.Equal(t, ClassBelowTarget, diab(99, ContextBedtime, over).Class())
	assert.Equal(t, ClassInTarget, diab(150, ContextBedtime, over).Class())
	assert.Equal(t, ClassLow, diab(65, ContextFasting, over).Class(), "hypo wins over the override")

	// A user without diabetes who has a targets row is classified as non-diabetic.
	nd := diab(135, ContextFasting, over)
	nd.Diabetic = false
	assert.Equal(t, ClassVeryHigh, nd.Class())
	// Pregnancy wins over the targets.
	pr := diab(100, ContextFasting, nil)
	pr.Pregnant = true
	assert.Equal(t, ClassHigh, pr.Class(), "the GDM fasting ceiling (95)")

	assert.True(t, HasDiabetes([]byte(`["migraine","diabetes"]`)))
	assert.False(t, HasDiabetes([]byte(`["migraine"]`)))
	assert.False(t, HasDiabetes([]byte(`null`)))
}

func TestDiabeticTargets_ReportListsTargetClasses(t *testing.T) {
	clinicalconfig.Reset()
	today := civildate.MustParse("2026-10-04")
	mk := func(id uint64, when string, mgdl float64, ctx string) Reading {
		r := glu(id, when, mgdl, ctx)
		r.Diabetic = true
		return r
	}
	rs := []Reading{mk(1, "2026-10-04 07:45", 120, ContextFasting), mk(2, "2026-10-03 07:30", 140, ContextFasting),
		mk(3, "2026-10-03 15:00", 170, ContextAfterMeal)}
	m := decode(t, Report(TypeGlucose, rs, NewRange("14d", today), GlucoseFilterAll))
	var codes []string
	for _, d := range m["distribution"].([]any) {
		codes = append(codes, d.(map[string]any)["code"].(string))
	}
	assert.Equal(t, []string{ClassUrgentLow, ClassLow, ClassInRange, ClassInTarget, ClassHigh, ClassAboveTarget,
		ClassVeryHigh}, codes)
	tir := m["time_in_range"].(map[string]any)
	assert.Equal(t, float64(2), tir["in_range"])
	assert.Equal(t, float64(1), tir["above"])
	byCtx := m["by_context"].([]any)
	assert.Equal(t, map[string]any{"min": float64(80), "max": float64(130)}, byCtx[0].(map[string]any)["target"])
	assert.Equal(t, map[string]any{"min": float64(70), "max": float64(180)}, byCtx[1].(map[string]any)["target"])
}

func TestReport_Minimums(t *testing.T) {
	clinicalconfig.Reset()
	today := civildate.MustParse("2026-10-04")
	oneDay := []Reading{bp(1, "2026-10-04 08:00", 118, 76, 70), bp(2, "2026-10-04 21:00", 124, 80, 72)}
	m := decode(t, Report(TypeBP, oneDay, NewRange("7d", today), ""))
	assert.Equal(t, ReportStatusNotEnough, m["status"])
	assert.Equal(t, float64(1), m["readings_needed"])
	assert.Equal(t, float64(2), m["readings"])
	for _, k := range []string{"average", "pulse", "min", "max", "distribution", "morning_vs_night", "time_in_range"} {
		assert.Nil(t, m[k], k)
	}
	assert.Len(t, m["series"], 1, "the daily series stays")

	twoDays := []Reading{bp(1, "2026-10-04 08:00", 118, 76, 70), bp(2, "2026-10-03 21:00", 124, 80, 72)}
	m = decode(t, Report(TypeBP, twoDays, NewRange("7d", today), ""))
	assert.Equal(t, ReportStatusOK, m["status"])
	assert.Equal(t, float64(0), m["readings_needed"])
	assert.Equal(t, float64(121), m["average"].(map[string]any)["systolic"])
	assert.NotNil(t, m["distribution"])
	assert.NotNil(t, m["min"])

	m = decode(t, Report(TypeGlucose, nil, NewRange("14d", today), GlucoseFilterAll))
	assert.Equal(t, ReportStatusNotEnough, m["status"])
	assert.Equal(t, float64(2), m["readings_needed"])

	g := []Reading{glu(1, "2026-10-04 07:00", 90, ContextFasting), glu(2, "2026-10-03 07:00", 95, ContextFasting),
		glu(3, "2026-10-03 15:00", 150, ContextAfterMeal)}
	m = decode(t, Report(TypeGlucose, g, NewRange("14d", today), GlucoseFilterAll))
	assert.Equal(t, ReportStatusOK, m["status"])
	byCtx := m["by_context"].([]any)
	assert.NotNil(t, byCtx[0].(map[string]any)["average"])
	m = decode(t, Report(TypeGlucose, g, NewRange("14d", today), ContextAfterMeal))
	assert.Equal(t, ReportStatusNotEnough, m["status"])
	assert.Nil(t, m["by_context"].([]any)[0].(map[string]any)["average"])

	assert.Equal(t, 2, ReadingsNeeded(nil))
	assert.Equal(t, 1, ReadingsNeeded(oneDay[:1]))
	assert.Equal(t, 0, ReadingsNeeded(twoDays))
}

// thresholdNumbers are the safety and classification numbers of version 1 that a vitals text must not spell out
// (they come from the registry through placeholders).
func thresholdNumbers() []string {
	t := ThresholdsOn(civildate.Date{})
	g := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	out := []string{g(t.BPCrisisSystolic), g(t.BPCrisisDiastolic), g(t.GlucoseUrgentBelow), g(t.GlucoseLowBelow),
		g(t.BPStage2Systolic), g(t.BPStage1Systolic), g(t.BPElevatedSystolic), g(t.Diabetic.PostMealMax),
		g(t.Diabetic.FastingHi), g(t.GlucoseBands[ContextFasting].VeryHighFrom),
		g(t.GlucoseBands[ContextRandom].VeryHighFrom)}
	return append(out, "3.0") // urgent low in mmol/L
}

var number = regexp.MustCompile(`\d+(?:\.\d+)?`)

// asciiDigits writes a locale's digits (lang `digits` / `decimal_separator`) as ASCII.
func asciiDigits(s, locale string) string {
	d := []rune(T("digits", locale))
	if len(d) == 10 {
		for i, r := range d {
			s = strings.ReplaceAll(s, string(r), string(rune('0'+i)))
		}
	}
	if sep := T("decimal_separator", locale); sep != "." {
		s = strings.ReplaceAll(s, sep, ".")
	}
	return s
}

func collectStrings(v any, out *[]string) {
	switch x := v.(type) {
	case string:
		*out = append(*out, x)
	case map[string]any:
		for _, e := range x {
			collectStrings(e, out)
		}
	case []any:
		for _, e := range x {
			collectStrings(e, out)
		}
	}
}

func hardCoded(t *testing.T, text, locale string) []string {
	t.Helper()
	var hits []string
	for _, n := range number.FindAllString(asciiDigits(text, locale), -1) {
		for _, th := range thresholdNumbers() {
			if n == th {
				hits = append(hits, n)
			}
		}
	}
	return hits
}

// The urgent copy (embedded fallback of every language) and the vitals screen copy (translation seed) never spell a
// threshold: they take {systolic}, {diastolic}, {mgdl}, {mmol_l}.
func TestCopy_NoHardCodedThresholds(t *testing.T) {
	clinicalconfig.Reset()
	locales, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, l := range locales {
		var f struct {
			Alerts map[string]any `json:"alerts"`
		}
		b, err := langFS.ReadFile("lang/" + l.Name() + "/vitals.json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &f))
		var texts []string
		collectStrings(f.Alerts, &texts)
		for _, s := range texts {
			assert.Empty(t, hardCoded(t, s, l.Name()), "lang/%s alerts: %q", l.Name(), s)
		}
	}
	seeds, err := filepath.Glob("../../resources/translations/*/vitals.json")
	require.NoError(t, err)
	require.NotEmpty(t, seeds)
	for _, p := range seeds {
		locale := filepath.Base(filepath.Dir(p))
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		for _, section := range []string{"alert", "report"} {
			var texts []string
			collectStrings(m[section], &texts)
			for _, s := range texts {
				assert.Empty(t, hardCoded(t, s, locale), "%s %s: %q", p, section, s)
			}
		}
	}
}

func TestCopy_PlaceholdersFilledFromTheReadingsVersion(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{
		{Key: clinicalconfig.KeyBPCrisis, Version: 2, Value: clinicalconfig.Value{"systolic": 170, "diastolic": 110},
			EffectiveFrom: civildate.MustParse("2026-10-01")},
		{Key: clinicalconfig.KeyGlucoseUrgentLow, Version: 2, Value: clinicalconfig.Value{"mgdl": 60},
			EffectiveFrom: civildate.MustParse("2026-10-01")},
	})
	render := func(r Reading, locale string) map[string]any {
		return decode(t, fillAlert(AlertCopy{}.AlertJSON(r, locale, locale), r, locale))
	}
	oldBP := Reading{Type: TypeBP, Date: civildate.MustParse("2026-09-30"), Systolic: 190, Diastolic: 100}
	a := render(oldBP, "en")
	assert.Contains(t, a["what_we_saw"], "above 180/120")
	newBP := oldBP
	newBP.Date = civildate.MustParse("2026-10-02")
	a = render(newBP, "en")
	assert.Contains(t, a["what_we_saw"], "above 170/110")
	assert.Contains(t, a["advice"], "above 170/110")
	assert.NotContains(t, a["what_we_saw"], "{")
	a = render(newBP, "fa")
	assert.Contains(t, a["what_we_saw"], "۱۷۰/۱۱۰")

	low := Reading{Type: TypeGlucose, Date: civildate.MustParse("2026-10-02"), MgDl: 58, Unit: UnitMgDl,
		Context: ContextFasting}
	a = render(low, "en")
	assert.Contains(t, a["what_we_saw"], "below 60 mg/dL (3.3 mmol/L)")
	a = render(low, "fa")
	assert.Contains(t, a["what_we_saw"], "کمتر از ۶۰ mg/dL (۳٫۳ mmol/L)")
	assert.Nil(t, fillAlert(nil, low, "en"))
}
