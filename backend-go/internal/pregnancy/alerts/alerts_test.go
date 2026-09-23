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
		{"bleeding severe → emergency", Context{"en", 20, lowRisk}, attrs("has_bleeding", true, "bleeding_severity", "severe"),
			[][2]string{{"emergency", "Emergency: Heavy Bleeding"}}, `{"bleeding":"severe"}`},
		{"bleeding mild high-risk → emergency", Context{"en", 20, highRisk}, attrs("has_bleeding", true, "bleeding_severity", "mild"),
			[][2]string{{"emergency", "Emergency: Heavy Bleeding"}}, `{"bleeding":"mild"}`},
		{"bleeding mild low-risk → warning", Context{"en", 20, lowRisk}, attrs("has_bleeding", true, "bleeding_severity", nil),
			[][2]string{{"warning", "Warning: Bleeding"}}, `{"bleeding":null}`},
		{"spotting first trimester → warning", Context{"en", 12, lowRisk}, attrs("has_spotting", true, "spotting_severity", "mild"),
			[][2]string{{"warning", "Warning: Spotting"}}, `{"spotting":"mild"}`},
		{"spotting severe later → warning", Context{"en", 20, lowRisk}, attrs("has_spotting", true, "spotting_severity", "severe"),
			[][2]string{{"warning", "Warning: Spotting"}}, `{"spotting":"severe"}`},
		{"spotting mild later → info", Context{"fa", 13, lowRisk}, attrs("has_spotting", true, "spotting_severity", "mild"),
			[][2]string{{"info", "توجه: لکه‌بینی خفیف"}}, `{"spotting":"mild"}`}, //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
		{"fluid leakage → emergency", Context{"en", 30, lowRisk}, attrs("has_fluid_leakage", true, "fluid_leakage_severity", "mild"),
			[][2]string{{"emergency", "Emergency: Fluid Leakage"}}, `{"fluid_leakage":"mild"}`},
		{"severe sudden pain → emergency", Context{"fa", 30, lowRisk}, attrs("has_severe_sudden_pain", true, "severe_sudden_pain_severity", "moderate"),
			[][2]string{{"emergency", "اورژانس: درد شدید"}}, `{"severe_sudden_pain":"moderate"}`},
		{"flag false → nothing", Context{"en", 20, lowRisk}, attrs("has_bleeding", false, "bleeding_severity", "severe"),
			[][2]string{}, ""},
		{"three severe → multiple", Context{"en", 20, lowRisk},
			attrs("headache_severity", "severe", "dizziness_severity", "severe", "back_pain_severity", "severe", "nausea_severity", "moderate"),
			[][2]string{{"warning", "Warning: Multiple Severe Symptoms"}}, `{"multiple_severe":true}`},
		{"two severe → nothing", Context{"en", 20, lowRisk},
			attrs("headache_severity", "severe", "dizziness_severity", "severe"), [][2]string{}, ""},
		{"order: spotting, bleeding, fluid, pain, multiple", Context{"en", 20, lowRisk},
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
		{"fasting 95 is fine", attrs("fasting_blood_sugar", jsonx.MustDecimal("95", 2)), [][2]string{}},
		{"fasting 95.01", attrs("fasting_blood_sugar", jsonx.MustDecimal("95.01", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"post-meal 140.5", attrs("post_meal_blood_sugar", jsonx.MustDecimal("140.5", 2)), [][2]string{{"warning", "Warning: High Blood Sugar"}}},
		{"severe depression", attrs("has_depression_feelings", true, "depression_severity", "severe"), [][2]string{{"warning", "Note: Mental Health"}}},
		{"depression without flag", attrs("has_depression_feelings", false, "depression_severity", "severe"), [][2]string{}},
		{"all three in order", attrs("systolic_pressure", int64(165), "diastolic_pressure", int64(112),
			"fasting_blood_sugar", jsonx.MustDecimal("130.5", 2), "has_depression_feelings", true, "depression_severity", "severe"),
			[][2]string{{"warning", "Warning: High Blood Pressure"}, {"warning", "Warning: High Blood Sugar"}, {"warning", "Note: Mental Health"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ForWeeklyLog(Context{"en", 27, highRisk}, tc.log)
			assert.Equal(t, tc.want, summarize(got))
			for _, d := range got {
				assert.Nil(t, d.MedicalFlags, "weekly alerts never set medical_history_flags")
			}
		})
	}

	got := ForWeeklyLog(Context{"fa", 27, nil}, attrs("systolic_pressure", int64(142), "diastolic_pressure", int64(92),
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
		got := ForFetalMovement(Context{"en", tc.week, highRisk}, attrs("movement_status", tc.status))
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
	got := ForFetalMovement(Context{"ar", 26, nil}, attrs("movement_status", "reduced"))
	assert.Equal(t, "Warning: Reduced Fetal Movement", got[0].Title)
	d := MissingData(Context{"fa", 10, nil}, "weekly", 5)
	assert.Equal(t, enums.AlertLevelInfo, d.Level)
	assert.Equal(t, "missing_data", d.Type)
	assert.Equal(t, "یادآوری: ثبت اطلاعات هفتگی", d.Title)
	assert.Contains(t, d.Message, "5 روز")
}
