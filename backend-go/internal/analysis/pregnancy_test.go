package analysis

// Pregnancy analysis (B-N3-12): unit tests of the thresholds and golden tests on seeded pregnancies
// (fixed request day 2026-09-23). Re-record with UPDATE_GOLDEN=1 go test ./internal/analysis/ -run Pregnancy.

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/resources/translations"
)

func pregLabeler(t *testing.T, locale string) func(string) string {
	t.Helper()
	store := i18n.NewTranslationStore(translations.FS, "")
	own := store.NamespaceMessages(locale, PregnancyNamespace, "fa")
	require.NotNil(t, own)
	return PregnancyLabeler(own, copyFor(t, locale))
}

func emptyPregnancy(start civildate.Date) *PregnancyInput {
	return &PregnancyInput{
		Today: fixtureToday, Start: start, Due: start.AddDays(280),
		Weights: map[civildate.Date]float64{}, BP: map[civildate.Date]BPReading{},
		Symptoms: map[civildate.Date]map[string]bool{}, Kicks: map[civildate.Date]KickSession{},
	}
}

func mins(n int) *int { return &n }

// fullPregnancy: week 31 (30w5d), 60 kg logged 3 weeks before the start, 165 cm (BMI 22.0, normal), a
// weekly weight from week 6 on track, weekly blood pressure all under 140/90, fasting and 1-hour sugar,
// six days of kick counts in the last week (one over 2 hours), symptoms of each trimester, six past
// visits (one with a result note) and two booked ones. Plus (trial).
func fullPregnancy() *PregnancyInput {
	start := civildate.MustParse("2026-02-20")
	in := emptyPregnancy(start)
	in.HeightCM, in.DeepAnalysis = 165, true
	in.Weights[start.AddDays(-21)] = 60
	for w := 6; w <= 30; w++ {
		d := start.AddDays(w*7 + 2)
		gain := 0.1 * float64(w)
		if w > 13 {
			gain = 1.3 + 0.42*float64(w-13)
		}
		in.Weights[d] = 60 + gain
		in.BP[d] = BPReading{Systolic: float64(108 + w%5*3), Diastolic: float64(68 + w%4*2)}
		if w >= 24 && w%2 == 0 {
			in.Glucose = append(in.Glucose,
				GlucoseReading{Date: d, Slot: GlucoseFasting, Value: float64(84 + w%3*3)},
				GlucoseReading{Date: d, Slot: GlucoseOneHour, Value: float64(126 + w%4*3)})
		}
	}
	for i, m := range []int{38, 52, 41, 0, 35, 128, 30} {
		if m == 0 {
			continue // no count that day
		}
		in.Kicks[fixtureToday.AddDays(-6+i)] = KickSession{Count: 10, Minutes: mins(m)}
	}
	add := func(fromDay, toDay, every int, keys ...string) {
		for g := fromDay; g <= toDay; g += every {
			d := start.AddDays(g)
			if in.Symptoms[d] == nil {
				in.Symptoms[d] = map[string]bool{}
			}
			for _, k := range keys {
				in.Symptoms[d][k] = true
			}
		}
	}
	add(42, 90, 2, "symptoms.digestive.nausea", "symptoms.general.fatigue")
	add(44, 88, 6, "pregnancy.symptoms.vomiting")
	add(100, 190, 3, "pain.location.back", "symptoms.digestive.heartburn")
	add(200, 214, 2, "symptoms.general.swelling", "symptoms.general.insomnia")
	add(201, 213, 4, "pain.location.abdomen")
	for i, w := range []int{8, 12, 16, 20, 24, 28} {
		v := PregnancyVisit{Date: start.AddDays(w * 7), Title: "ویزیت", Done: i%2 == 0}
		if w == 24 {
			v.Title, v.ResultNote = "قند ۲ ساعته", "طبیعی"
		}
		in.Visits = append(in.Visits, v)
	}
	in.Visits = append(in.Visits,
		PregnancyVisit{Date: fixtureToday.AddDays(10), Title: "سونوگرافی"},
		PregnancyVisit{Date: fixtureToday.AddDays(30), Title: "ویزیت"})
	return in
}

// sparsePregnancy: week 21, free user, a profile weight but no height, three weights, one reading at
// 145/92, a kick day with too few movements, no visits.
func sparsePregnancy() *PregnancyInput {
	start := civildate.MustParse("2026-04-26")
	in := emptyPregnancy(start)
	pw := 72.0
	in.ProfileWeight = &pw
	in.Weights[start.AddDays(120)] = 75.5
	in.Weights[start.AddDays(135)] = 76.2
	in.Weights[start.AddDays(148)] = 77.0
	in.BP[start.AddDays(120)] = BPReading{Systolic: 122, Diastolic: 80}
	in.BP[start.AddDays(148)] = BPReading{Systolic: 145, Diastolic: 92}
	in.Glucose = []GlucoseReading{{Date: start.AddDays(140), Slot: GlucoseFasting, Value: 99}}
	in.Kicks[fixtureToday.AddDays(-1)] = KickSession{Count: 6}
	return in
}

// earlyPregnancy: week 6, nothing logged at all (no baseline, no height).
func earlyPregnancy() *PregnancyInput {
	in := emptyPregnancy(civildate.MustParse("2026-08-14"))
	in.DeepAnalysis = true
	return in
}

var pregnancyFixtures = map[string]func() *PregnancyInput{
	"preg_full": fullPregnancy, "preg_sparse": sparsePregnancy, "preg_early": earlyPregnancy,
}

func TestPregnancy_Golden(t *testing.T) {
	names := make([]string, 0, len(pregnancyFixtures))
	for n := range pregnancyFixtures {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		in := pregnancyFixtures[name]()
		in.Label = pregLabeler(t, "fa")
		golden(t, name+"_pregnancy", render(t, BuildPregnancy(in)))
	}
	en := fullPregnancy()
	en.Label = pregLabeler(t, "en")
	golden(t, "preg_full_pregnancy_en", render(t, BuildPregnancy(en)))
}

func sec(t *testing.T, out *jsonx.OrderedMap, key string) map[string]any {
	t.Helper()
	raw := render(t, out)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	s, _ := m["sections"].(map[string]any)
	v, _ := s[key].(map[string]any)
	return v
}

func TestPregnancy_IOMBands(t *testing.T) {
	assert.Equal(t, "underweight", IOMBandFor(18.4).Category)
	assert.Equal(t, "normal", IOMBandFor(18.5).Category)
	assert.Equal(t, "normal", IOMBandFor(24.9).Category)
	assert.Equal(t, "overweight", IOMBandFor(25).Category)
	assert.Equal(t, "obese", IOMBandFor(30).Category)
	assert.Equal(t, "obese", IOMBandFor(45).Category)

	normal := IOMBandFor(22)
	lo, hi := RecommendedGain(normal, 0)
	assert.Zero(t, lo+hi)
	lo, hi = RecommendedGain(normal, 13)
	assert.InDelta(t, 0.5, lo, 1e-9)
	assert.InDelta(t, 2, hi, 1e-9)
	lo, hi = RecommendedGain(normal, 40)
	assert.InDelta(t, 11.5, lo, 1e-9)
	assert.InDelta(t, 16, hi, 1e-9)
	lo, hi = RecommendedGain(normal, 42)
	assert.InDelta(t, 11.5, lo, 1e-9, "flat after term")
	assert.InDelta(t, 16, hi, 1e-9)
	lo, _ = RecommendedGain(IOMBandFor(32), 40)
	assert.InDelta(t, 5, lo, 1e-9)
}

func TestPregnancy_Statuses(t *testing.T) {
	full := BuildPregnancy(fullPregnancy())
	w := sec(t, full, "weight_gain")
	assert.Equal(t, true, w["ready"])
	data := w["data"].(map[string]any)
	assert.Equal(t, "within", data["status"])
	assert.Equal(t, "normal", data["bmi_category"])
	assert.Equal(t, "before_pregnancy", data["baseline"].(map[string]any)["source"])
	assert.NotNil(t, data["recent_4w"])
	assert.Equal(t, "below_threshold", sec(t, full, "blood_pressure")["data"].(map[string]any)["status"])
	kicks := sec(t, full, "kicks")["data"].(map[string]any)
	assert.Equal(t, false, kicks["all_within_window"], "one day took 128 minutes")
	assert.InDelta(t, 6, kicks["sessions"], 0)

	sparse := BuildPregnancy(sparsePregnancy())
	ws := sec(t, sparse, "weight_gain")
	assert.Equal(t, false, ws["ready"])
	assert.Equal(t, "height", ws["data"].(map[string]any)["missing"])
	bp := sec(t, sparse, "blood_pressure")["data"].(map[string]any)
	assert.Equal(t, "high", bp["status"])
	assert.InDelta(t, 1, bp["high_count"], 0)
	g := sec(t, sparse, "glucose")
	assert.Equal(t, true, g["locked"])
	assert.Nil(t, g["data"], "a locked section computes and sends nothing")
	assert.Equal(t, true, sec(t, sparse, "symptoms")["locked"])

	// A severe reading wins over a high one.
	in := sparsePregnancy()
	in.BP[fixtureToday] = BPReading{Systolic: 150, Diastolic: 112}
	assert.Equal(t, "severe", sec(t, BuildPregnancy(in), "blood_pressure")["data"].(map[string]any)["status"])

	// Above the band.
	in = fullPregnancy()
	in.Weights[fixtureToday] = 80
	assert.Equal(t, "above", sec(t, BuildPregnancy(in), "weight_gain")["data"].(map[string]any)["status"])

	early := BuildPregnancy(earlyPregnancy())
	assert.Equal(t, "baseline", sec(t, early, "weight_gain")["data"].(map[string]any)["missing"])
	assert.Equal(t, false, sec(t, early, "visits")["ready"])
}

func TestPregnancy_BaselineOrder(t *testing.T) {
	start := civildate.MustParse("2026-02-20")
	in := emptyPregnancy(start)
	pw := 70.0
	in.ProfileWeight = &pw
	in.Weights[start.AddDays(-200)] = 50 // too old to be the pre-pregnancy weight
	in.Weights[start.AddDays(40)] = 61
	in.Weights[start.AddDays(60)] = 62
	w, src, d, ok := in.baseline()
	require.True(t, ok)
	assert.InDelta(t, 61, w, 0)
	assert.Equal(t, BaselineFirstTrimester, src)
	assert.Equal(t, start.AddDays(40), *d)

	in.Weights[start.AddDays(-10)] = 59
	_, src, _, _ = in.baseline()
	assert.Equal(t, BaselineBeforePregnancy, src)

	in = emptyPregnancy(start)
	in.ProfileWeight = &pw
	w, src, d, ok = in.baseline()
	require.True(t, ok)
	assert.InDelta(t, 70, w, 0)
	assert.Equal(t, BaselineProfile, src)
	assert.Nil(t, d)
}
