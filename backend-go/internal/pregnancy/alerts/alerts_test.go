package alerts

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// attrs builds a log's cast attribute values.
func attrs(kv ...any) *jsonx.OrderedMap { return jsonx.Obj(kv...) }

var (
	yes       = sql.NullBool{Bool: true, Valid: true}
	highRisk  = &store.PregnancyProfile{HasMiscarriageHistory: yes, RhFactor: sql.NullString{String: "negative", Valid: true}, PreExistingConditions: db.NullRawJSON{V: json.RawMessage(`["hypertension"]`), Valid: true}}
	lowRisk   = &store.PregnancyProfile{}
	summarize = func(ds []Draft) [][2]string {
		out := [][2]string{}
		for _, d := range ds {
			out = append(out, [2]string{string(d.Level), d.Title})
		}
		return out
	}
)

// Every branch of PregnancyAlertService::getSymptomAlertData + the 3-severe rule.
func TestSymptomRules(t *testing.T) {
	cases := []struct {
		name    string
		ctx     Context
		log     *jsonx.OrderedMap
		want    [][2]string
		trigger string
	}{
		{"bleeding severe → emergency", Context{Locale: "en", Week: 20, Profile: lowRisk}, attrs("has_bleeding", true, "bleeding_severity", "severe"),
			[][2]string{{"emergency", "Emergency: Heavy Bleeding"}}, `{"bleeding":"severe"}`},
		{"bleeding mild high-risk → emergency", Context{Locale: "en", Week: 20, Profile: highRisk}, attrs("has_bleeding", true, "bleeding_severity", "mild"),
			[][2]string{{"emergency", "Emergency: Heavy Bleeding"}}, `{"bleeding":"mild"}`},
		{"bleeding mild low-risk → warning", Context{Locale: "en", Week: 20, Profile: lowRisk}, attrs("has_bleeding", true, "bleeding_severity", nil),
			[][2]string{{"warning", "Warning: Bleeding"}}, `{"bleeding":null}`},
		{"spotting first trimester → warning", Context{Locale: "en", Week: 12, Profile: lowRisk}, attrs("has_spotting", true, "spotting_severity", "mild"),
			[][2]string{{"warning", "Warning: Spotting"}}, `{"spotting":"mild"}`},
		{"spotting severe later → warning", Context{Locale: "en", Week: 20, Profile: lowRisk}, attrs("has_spotting", true, "spotting_severity", "severe"),
			[][2]string{{"warning", "Warning: Spotting"}}, `{"spotting":"severe"}`},
		{"spotting mild later → info", Context{Locale: "fa", Week: 13, Profile: lowRisk}, attrs("has_spotting", true, "spotting_severity", "mild"),
			[][2]string{{"info", "توجه: لکه‌بینی خفیف"}}, `{"spotting":"mild"}`}, //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
		{"fluid leakage → emergency", Context{Locale: "en", Week: 30, Profile: lowRisk}, attrs("has_fluid_leakage", true, "fluid_leakage_severity", "mild"),
			[][2]string{{"emergency", "Emergency: Fluid Leakage"}}, `{"fluid_leakage":"mild"}`},
		{"severe sudden pain → emergency", Context{Locale: "fa", Week: 30, Profile: lowRisk}, attrs("has_severe_sudden_pain", true, "severe_sudden_pain_severity", "moderate"),
			[][2]string{{"emergency", "اورژانس: درد شدید"}}, `{"severe_sudden_pain":"moderate"}`},
		{"flag false → nothing", Context{Locale: "en", Week: 20, Profile: lowRisk}, attrs("has_bleeding", false, "bleeding_severity", "severe"),
			[][2]string{}, ""},
		{"three severe → multiple", Context{Locale: "en", Week: 20, Profile: lowRisk},
			attrs("headache_severity", "severe", "dizziness_severity", "severe", "back_pain_severity", "severe", "nausea_severity", "moderate"),
			[][2]string{{"warning", "Warning: Multiple Severe Symptoms"}}, `{"multiple_severe":true}`},
		{"two severe → nothing", Context{Locale: "en", Week: 20, Profile: lowRisk},
			attrs("headache_severity", "severe", "dizziness_severity", "severe"), [][2]string{}, ""},
		{"order: spotting, bleeding, fluid, pain, multiple", Context{Locale: "en", Week: 20, Profile: lowRisk},
			attrs("has_severe_sudden_pain", true, "has_fluid_leakage", true, "has_bleeding", true, "bleeding_severity", "severe",
				"has_spotting", true, "cramping_severity", "severe", "pelvic_pressure_severity", "severe", "fatigue_severity", "severe"),
			[][2]string{
				{"info", "Note: Light Spotting"}, {"emergency", "Emergency: Heavy Bleeding"},
				{"emergency", "Emergency: Fluid Leakage"}, {"emergency", "Emergency: Severe Pain"},
				{"warning", "Warning: Multiple Severe Symptoms"},
			}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ForSymptomLog(tc.ctx, tc.log)
			assert.Equal(t, tc.want, summarize(got))
			for _, d := range got {
				assert.Equal(t, "symptom_based", d.Type)
				assert.Equal(t, tc.ctx.Week, d.Week)
				assert.NotNil(t, d.MedicalFlags)
			}
			if tc.trigger != "" {
				b, err := json.Marshal(got[0].Trigger)
				require.NoError(t, err)
				assert.JSONEq(t, tc.trigger, string(b))
			}
		})
	}
}

func TestMedicalHistoryFlags(t *testing.T) {
	enc := func(p *store.PregnancyProfile) string {
		b, err := jsonx.Marshal(MedicalHistoryFlags(p), 0)
		require.NoError(t, err)
		return string(b)
	}
	assert.Equal(t, `[]`, enc(nil))
	assert.Equal(t, `[]`, enc(lowRisk))
	assert.Equal(t, `[]`, enc(&store.PregnancyProfile{PreExistingConditions: db.NullRawJSON{V: json.RawMessage(`[]`), Valid: true}}))
	assert.JSONEq(t, `{"miscarriage_history":true,"rh_negative":true,"pre_existing_conditions":["hypertension"]}`, enc(highRisk))
	assert.JSONEq(t, `{"high_risk_history":true}`, enc(&store.PregnancyProfile{HasHighRiskHistory: yes}))
}

func TestWeeklyRules(t *testing.T) {
	cases := []struct {
		name string
		log  *jsonx.OrderedMap
		want [][2]string
	}{
		{"normal", attrs("systolic_pressure", int64(115), "diastolic_pressure", int64(75), "fasting_blood_sugar", jsonx.MustDecimal("90", 2)), [][2]string{}},
		{"systolic 140", attrs("systolic_pressure", int64(140), "diastolic_pressure", int64(80)), [][2]string{{"warning", "Warning: High Blood Pressure"}}},
		{"diastolic 90", attrs("systolic_pressure", int64(120), "diastolic_pressure", int64(90)), [][2]string{{"warning", "Warning: High Blood Pressure"}}},
		{"missing diastolic → no BP alert", attrs("systolic_pressure", int64(180), "diastolic_pressure", nil), [][2]string{}},
		{"systolic 139 / 89 fine", attrs("systolic_pressure", int64(139), "diastolic_pressure", int64(89)), [][2]string{}},
		{"159/109 still the warning", attrs("systolic_pressure", int64(159), "diastolic_pressure", int64(109)), [][2]string{{"warning", "Warning: High Blood Pressure"}}},
		{"systolic 160 → emergency", attrs("systolic_pressure", int64(160), "diastolic_pressure", int64(80)), [][2]string{{"emergency", "Emergency: Severe Blood Pressure"}}},
		{"diastolic 110 → emergency", attrs("systolic_pressure", int64(120), "diastolic_pressure", int64(110)), [][2]string{{"emergency", "Emergency: Severe Blood Pressure"}}},
		{"fasting 94.99 is fine", attrs("fasting_blood_sugar", jsonx.MustDecimal("94.99", 2)), [][2]string{}},
		{"fasting 95 (≥, T-M9-05)", attrs("fasting_blood_sugar", jsonx.MustDecimal("95", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"post-meal 139.99 is fine", attrs("post_meal_blood_sugar", jsonx.MustDecimal("139.99", 2)), [][2]string{}},
		{"post-meal 140", attrs("post_meal_blood_sugar", jsonx.MustDecimal("140", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"fasting 95.01", attrs("fasting_blood_sugar", jsonx.MustDecimal("95.01", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"post-meal 140.5", attrs("post_meal_blood_sugar", jsonx.MustDecimal("140.5", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"severe depression", attrs("has_depression_feelings", true, "depression_severity", "severe"), [][2]string{{"warning", "Note: Mental Health"}}},
		{"depression without flag", attrs("has_depression_feelings", false, "depression_severity", "severe"), [][2]string{}},
		{"all three in order", attrs("systolic_pressure", int64(165), "diastolic_pressure", int64(112),
			"fasting_blood_sugar", jsonx.MustDecimal("130.5", 2), "has_depression_feelings", true, "depression_severity", "severe"),
			[][2]string{{"emergency", "Emergency: Severe Blood Pressure"}, {"warning", "Warning: High Blood Sugar"}, {"warning", "Note: Mental Health"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ForWeeklyLog(Context{Locale: "en", Week: 27, Profile: highRisk}, tc.log)
			assert.Equal(t, tc.want, summarize(got))
			for _, d := range got {
				assert.Nil(t, d.MedicalFlags, "weekly alerts never set medical_history_flags")
			}
		})
	}

	got := ForWeeklyLog(Context{Locale: "fa", Week: 27, Profile: nil}, attrs("systolic_pressure", int64(142), "diastolic_pressure", int64(92),
		"fasting_blood_sugar", jsonx.MustDecimal("97.5", 2), "post_meal_blood_sugar", jsonx.NullDecimal))
	require.Len(t, got, 2)
	assert.Equal(t, "فشار خون شما (142/92) بالاتر از حد طبیعی است. لطفاً با پزشک مشورت کنید.", got[0].Message)
	b, err := json.Marshal(got[1].Trigger)
	require.NoError(t, err)
	assert.JSONEq(t, `{"fasting":"97.50","post_meal":null}`, string(b), "sugar values are decimal strings")
}

func TestFetalMovementRules(t *testing.T) {
	cases := []struct {
		week   int
		status string
		want   [][2]string
	}{
		{24, "none", [][2]string{{"emergency", "Emergency: No Fetal Movement"}}},
		{30, "reduced", [][2]string{{"warning", "Warning: Reduced Fetal Movement"}}},
		{23, "none", [][2]string{}},
		{30, "normal", [][2]string{}},
		{30, "felt", [][2]string{}},
	}
	for _, tc := range cases {
		got := ForFetalMovement(Context{Locale: "en", Week: tc.week, Profile: highRisk}, attrs("movement_status", tc.status))
		assert.Equal(t, tc.want, summarize(got), "%d %s", tc.week, tc.status)
		for _, d := range got {
			assert.Nil(t, d.MedicalFlags)
			b, _ := json.Marshal(d.Trigger)
			assert.JSONEq(t, `{"fetal_movement":"`+tc.status+`"}`, string(b))
		}
	}
}

func TestLocaleAndMissingData(t *testing.T) {
	// Any locale other than fa is English (e.g. ar).
	got := ForFetalMovement(Context{Locale: "ar", Week: 26, Profile: nil}, attrs("movement_status", "reduced"))
	assert.Equal(t, "Warning: Reduced Fetal Movement", got[0].Title)
	d := MissingData(Context{Locale: "fa", Week: 10, Profile: nil}, "weekly", 5)
	assert.Equal(t, enums.AlertLevelInfo, d.Level)
	assert.Equal(t, "missing_data", d.Type)
	assert.Equal(t, "یادآوری: ثبت اطلاعات هفتگی", d.Title)
	assert.Contains(t, d.Message, "5 روز")
}

// T-M9-05: the severe tier is urgent with 115 in both languages; the manual fetal path follows the registry week.
func TestSevereBPCopyAndRegistry(t *testing.T) {
	log := attrs("systolic_pressure", int64(165), "diastolic_pressure", int64(112))
	fa := ForWeeklyLog(Context{Locale: "fa", Week: 30}, log)
	require.Len(t, fa, 1)
	assert.Equal(t, enums.AlertLevelEmergency, fa[0].Level)
	assert.Contains(t, fa[0].Message, "۱۱۵")
	assert.Contains(t, fa[0].Message, "165/112")
	en := ForWeeklyLog(Context{Locale: "en", Week: 30}, log)
	assert.Contains(t, en[0].Message, "115")
	assert.Equal(t, []string{"Contact your doctor or midwife now", "Call emergency services (115)"}, en[0].Actions)
	assert.Len(t, ForFetalMovement(Context{Locale: "en", Week: 24}, attrs("movement_status", "reduced")), 1)
}
