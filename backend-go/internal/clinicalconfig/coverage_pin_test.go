package clinicalconfig_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/children/growth"
	cc "github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
)

func v1(t *testing.T, key string) cc.Version {
	t.Helper()
	cc.Reset()
	v := cc.At(key, civildate.MustParse("2026-10-21"))
	require.Equal(t, 1, v.Version, key)
	return v
}

func readRepo(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile("../../" + path)
	require.NoError(t, err, path)
	return string(b)
}

// TestCoverageKeys_PinnedToTheirOrigin: every T-M10-16 version 1 equals the constant or seed it replaces (imported
// where exported, else the file:line in the comment, checked by reading the file), or the SPEC v6 table for keys
// nothing read before.
func TestCoverageKeys_PinnedToTheirOrigin(t *testing.T) {
	t.Cleanup(cc.Reset)

	numbers := []struct {
		key    string
		want   cc.Value
		origin string
	}{
		// Laravel's `LmpDate.Date.AddDays(280)` (PregnancyCalculationService); calc reads the key since T-M10-18 —
		// pinned below through calc.EDDFromStart / calc.EDDFromLMP.
		{cc.KeyPregnancyEDD, cc.Value{"naegele_days": 280}, "Laravel AddDays(280)"},
		// No cycle-length correction anywhere today (SPEC §3.1: optional, off by default).
		{cc.KeyPregnancyEDDCycleCorrect, cc.Value{"enabled": 0}, "SPEC §3.1"},
		// SPEC §3.1 (ACOG CO 700): T2 from 14+0, T3 from 28+0. Today calc.go:132-139 starts T2 at 13 completed weeks
		// (day 91); T-M10-18 switches it.
		{cc.KeyPregnancyTrimester, cc.Value{"t2_from_days": 14 * 7, "t3_from_days": 28 * 7}, "SPEC §3.1"},
		{cc.KeyPregnancyIOMFirstTri, cc.Value{"lo": analysis.IOMFirstTrimesterGainMin, "hi": analysis.IOMFirstTrimesterGainMax},
			"analysis/pregnancy_thresholds.go:28-29"},
		// internal/messages/pregnancyalerts/rules.go:289 `c.intParam("spotting_until_week", 12)`; seed
		// db/migrations/00005_pregnancy_v2.sql:236-237.
		{cc.KeyPregnancySpotting, cc.Value{"until_week": 12}, "rules.go:289"},
		{cc.KeyEPDSSchedule, cc.Value{"full_days": postpartum.FullIntervalDays, "short_days": postpartum.ShortIntervalDays},
			"postpartum/epds.go:38-39"},
		// SPEC §3.3 table; nonpregnant_low = the hemoglobin catalog typical.low 12 (db/migrations/00034_labs.sql:126).
		{cc.KeyLabHb, cc.Value{"nonpregnant_low": 12, "pregnant_low": 11, "pregnant_t2_low": 10.5}, "SPEC §3.3"},
		{cc.KeyLabFerritinInflame, cc.Value{"low": 70}, "SPEC §3.3"},
		{cc.KeyLabVitD, cc.Value{"deficient_below": 12, "sufficient_from": 20}, "SPEC §3.3"},
		{cc.KeyLabTSHPregnancy, cc.Value{"upper": 4.0}, "SPEC §3.3"},
		{cc.KeyLabHbA1c, cc.Value{"prediabetes_from": 5.7, "diabetes_from": 6.5}, "SPEC §3.2"},
		// SPEC §3.3 / H-08 (0.9, user-accepted). Today labs/labs.go:89 LowConfidence = 0.7; T-M10-20 switches it.
		{cc.KeyLabOCR, cc.Value{"min_confidence": 0.9}, "SPEC §3.3"},
		{cc.KeyLabBorderline, cc.Value{"enabled": 1, "margin": labs.BorderlineMargin}, "labs/status.go:28"},
		// H-09: 0 = today's post-meal bands (vitals glucose.post_meal 140/200) for every non-fasting context.
		{cc.KeyGlucoseContexts, cc.Value{"spec": 0}, "DECISIONS.md H-09"},
		{cc.KeyGlucoseDiabetic, cc.Value{"fasting_lo": 80, "fasting_hi": 130, "post_meal_max": 180}, "SPEC §3.2"},
		// growth/growth.go:24-25 BandLowPercentile / BandHighPercentile; cross_lines SPEC §3.5 (nothing reads it today).
		{cc.KeyChildGrowth, cc.Value{"low_pct": growth.BandLowPercentile, "high_pct": growth.BandHighPercentile, "cross_lines": 2},
			"growth.go:24-25"},
	}
	for _, n := range numbers {
		assert.Equal(t, n.want, v1(t, n.key).Value, "%s (%s)", n.key, n.origin)
	}

	// bmi.who = enums/health_logic.go:27-31 BmiCategoryFromBmi (= profile/bmi.go:84-89): a boundary lands higher.
	b := v1(t, cc.KeyBMIWHO).Value
	assert.Equal(t, cc.Value{"underweight_below": 18.5, "overweight_from": 25, "obese_from": 30}, b)
	for _, x := range []struct {
		bmi  float64
		want enums.BmiCategory
	}{
		{b["underweight_below"] - 0.01, enums.BmiCategoryUnderweight}, {b["underweight_below"], enums.BmiCategoryNormal},
		{b["overweight_from"] - 0.01, enums.BmiCategoryNormal}, {b["overweight_from"], enums.BmiCategoryOverweight},
		{b["obese_from"] - 0.01, enums.BmiCategoryOverweight}, {b["obese_from"], enums.BmiCategoryObese},
	} {
		assert.Equal(t, x.want, enums.BmiCategoryFromBmi(x.bmi), x.bmi)
	}

	// pregnancy.edd v1: calc reads the key, and its due date is the former LMP + 280 days (no cycle correction while
	// pregnancy.edd.cycle_correction v1 is off, whatever the cycle length).
	cc.Reset()
	day := civildate.MustParse("2026-10-21")
	assert.Equal(t, 280, calc.NaegeleDays(day))
	lmp := civildate.MustParse("2026-06-25")
	assert.Equal(t, lmp.AddDays(280), calc.EDDFromStart(lmp))
	assert.Equal(t, lmp.AddDays(280), calc.EDDFromLMP(lmp, 35))

	// The literal without an exported constant: read it where it lives.
	assert.Contains(t, readRepo(t, "internal/messages/pregnancyalerts/rules.go"), `c.intParam("spotting_until_week", 12)`)

	// pregnancy.iom totals and BMI bands = analysis/pregnancy_thresholds.go:19-24 IOMBands (BMIMax 0 = open);
	// weekly rates = IOM 2009 Table S-1 (2nd/3rd trimester kg/week; not in the code or SPEC before).
	iom := v1(t, cc.KeyPregnancyIOM).Rows
	require.Len(t, iom, len(analysis.IOMBands))
	weekly := [][2]float64{{0.44, 0.58}, {0.35, 0.50}, {0.23, 0.33}, {0.17, 0.27}}
	for i, band := range analysis.IOMBands {
		var to any = band.BMIMax
		if band.BMIMax == 0 {
			to = nil
		}
		assert.Equal(t, cc.Row{"bmi_from": band.BMIMin, "bmi_to": to, "total_lo": band.GainMin, "total_hi": band.GainMax,
			"weekly_lo": weekly[i][0], "weekly_hi": weekly[i][1]}, iom[i], band.Category)
	}

	// pregnancy.redating = SPEC §3.1 table (ACOG CO 700 Table 1): until 8+6 → 5, 13+6 → 7, 15+6 → 7, 21+6 → 10,
	// 27+6 → 14, from 28+0 → 21 days.
	ga := func(w, d int) any { return float64(w*7 + d) }
	assert.Equal(t, []cc.Row{
		{"ga_until_days": ga(8, 6), "max_diff_days": 5.0},
		{"ga_until_days": ga(13, 6), "max_diff_days": 7.0},
		{"ga_until_days": ga(15, 6), "max_diff_days": 7.0},
		{"ga_until_days": ga(21, 6), "max_diff_days": 10.0},
		{"ga_until_days": ga(27, 6), "max_diff_days": 14.0},
		{"ga_until_days": nil, "max_diff_days": 21.0},
	}, v1(t, cc.KeyPregnancyRedating).Rows)
	assert.Contains(t, v1(t, cc.KeyPregnancyRedating).Source, "GA by LMP on the scan day")

	// screening.uspstf = SPEC §3.4 table; type keys are the checkup_types keys (00003_checkups.sql:86-91, the two
	// screening types arrive in T-M10-21), test kinds the checkup_records.test_kind values (00099).
	row := func(typ string, lo, hi any, months float64, kind, bmi any) cc.Row {
		return cc.Row{"type_key": typ, "age_min": lo, "age_max": hi, "interval_months": months, "last_test_kind": kind, "min_bmi": bmi}
	}
	assert.Equal(t, []cc.Row{
		row("pap_smear", 21.0, 29.0, 36, "cytology", nil),
		row("pap_smear", 30.0, 65.0, 36, "cytology", nil),
		row("pap_smear", 30.0, 65.0, 60, "hpv", nil),
		row("pap_smear", 30.0, 65.0, 60, "cotest", nil),
		row("mammography", 40.0, 74.0, 24, nil, nil),
		row("bp_screening", 18.0, 39.0, 36, nil, nil),
		row("bp_screening", 40.0, nil, 12, nil, nil),
		row("glucose_screening", 35.0, 70.0, 36, nil, 25.0),
		row("dentist", nil, nil, 6, nil, nil),
	}, v1(t, cc.KeyScreeningUSPSTF).Rows)
	checkups := readRepo(t, "db/migrations/00003_checkups.sql")
	for _, k := range []string{"pap_smear", "mammography", "dentist"} {
		assert.Contains(t, checkups, "('"+k+"',", "checkup type %s", k)
	}
	assert.Contains(t, readRepo(t, "db/migrations/00099_health_targets_confirmation_test_kind.sql"), "cytology | hpv | cotest")

	// child.vaccine_schedule = the child_vaccines seed of db/migrations/00031_children.sql:100-204 (code, age_months),
	// in seed order; dose = ordinal of the code within its vaccine (prefix before "_").
	seed := readRepo(t, "db/migrations/00031_children.sql")
	re := regexp.MustCompile(`\('child_vaccines', '([a-z0-9_]+)', \d+, [\s\S]*?"age_months":(\d+)\}`)
	matches := re.FindAllStringSubmatch(seed, -1)
	require.Len(t, matches, 21)
	vac := v1(t, cc.KeyChildVaccineSchedule).Rows
	require.Len(t, vac, len(matches))
	ordinal := map[string]float64{}
	for i, m := range matches {
		age, _ := strconv.Atoi(m[2])
		family := strings.SplitN(m[1], "_", 2)[0]
		ordinal[family]++
		assert.Equal(t, cc.Row{"vaccine_code": m[1], "dose": ordinal[family], "age_months": float64(age)}, vac[i], m[1])
	}
}
