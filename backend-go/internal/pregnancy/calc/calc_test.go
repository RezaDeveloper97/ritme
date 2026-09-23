package calc

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

var today = civildate.MustParse("2026-09-23")

func nd(s string) civildate.NullDate {
	return civildate.NullDate{Date: civildate.MustParse(s), Valid: true}
}
func ni(n int32) sql.NullInt32   { return sql.NullInt32{Int32: n, Valid: true} }
func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

// Contract personas (docs/go-migration/contract.md): LMP 2026-08-02 → week 8, ultrasound
// 23w2d on 2026-09-09 → week 26, manual 12w0d entered 2026-09-16 → week 14.
func TestGestationalAgeBySource(t *testing.T) {
	cases := []struct {
		name                string
		p                   store.PregnancyProfile
		weeks, days, total  int
		trimester, week     int
		conf                string
		uncertainty         int
		edd                 string
		daysLeft, weeksLeft int
	}{
		{"lmp", store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: nd("2026-08-02")},
			7, 3, 52, 1, 8, "medium", 3, "2027-05-09", 228, 32},
		{"ultrasound", store.PregnancyProfile{AgeSource: ns("ultrasound"), UltrasoundDate: nd("2026-09-09"), UltrasoundWeeks: ni(23), UltrasoundDays: ni(2)},
			25, 2, 177, 2, 26, "high", 1, "2027-01-04", 103, 14},
		{"manual", store.PregnancyProfile{AgeSource: ns("manual"), ManualEntryDate: nd("2026-09-16"), ManualWeeks: ni(12), ManualDays: ni(0)},
			13, 0, 91, 2, 14, "low", 5, "2027-03-31", 189, 27},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(&tc.p, "en", today)
			ga := c.GestationalAge()
			require.True(t, ga.Valid)
			assert.Equal(t, tc.weeks, ga.Weeks)
			assert.Equal(t, tc.days, ga.Days)
			assert.Equal(t, tc.total, ga.TotalDays)
			assert.Equal(t, tc.trimester, ga.Trimester)
			assert.Equal(t, tc.conf, string(ga.Confidence))
			assert.Equal(t, tc.uncertainty, ga.Uncertainty)
			assert.Equal(t, tc.week, c.CurrentWeek())
			edd := c.EDD()
			require.NotNil(t, edd)
			assert.Equal(t, tc.edd, edd.Date.String())
			assert.Equal(t, tc.daysLeft, edd.DaysRemaining)
			assert.Equal(t, tc.weeksLeft, edd.WeeksRemaining)
		})
	}
}

func TestEmptyAndClamp(t *testing.T) {
	// No profile / no age source: empty GA, week 1 (null + 1), no EDD.
	assert.False(t, New(nil, "fa", today).GestationalAge().Valid)
	assert.Equal(t, 1, New(nil, "fa", today).CurrentWeek())
	activated := &store.PregnancyProfile{PregnancyMode: true}
	assert.Equal(t, 1, New(activated, "fa", today).CurrentWeek())
	assert.Nil(t, New(activated, "fa", today).EDD())
	// Missing ultrasound days → empty.
	assert.False(t, New(&store.PregnancyProfile{AgeSource: ns("ultrasound"), UltrasoundDate: nd("2026-09-01"), UltrasoundWeeks: ni(3)}, "en", today).GestationalAge().Valid)
	// Clamp at 40; past the due date days_remaining is 0.
	late := &store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: nd("2025-11-01")}
	assert.Equal(t, 40, New(late, "en", today).CurrentWeek())
	assert.Equal(t, 0, New(late, "en", today).EDD().DaysRemaining)
	assert.Equal(t, 0, New(late, "en", today).EDD().WeeksRemaining)
}

func TestTrimesterBoundaries(t *testing.T) {
	for weeks, want := range map[int]int{0: 1, 12: 1, 13: 2, 27: 2, 28: 3, 45: 3} {
		assert.Equal(t, want, Trimester(weeks), "weeks %d", weeks)
	}
}

func TestFetalMovementThresholds(t *testing.T) {
	for _, tc := range []struct {
		lmp              string
		active, required bool
	}{
		{"2026-05-31", false, false}, // 16w3d → week 17
		{"2026-05-24", true, false},  // 17w3d → week 18
		{"2026-04-12", false, true},  // 23w3d → week 24
	} {
		c := New(&store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: nd(tc.lmp)}, "en", today)
		assert.Equal(t, tc.active || tc.required, c.FetalMovementTrackingActive(), tc.lmp)
		assert.Equal(t, tc.required, c.FetalMovementRequired(), tc.lmp)
	}
}

func TestHighRisk(t *testing.T) {
	yes := sql.NullBool{Bool: true, Valid: true}
	assert.False(t, IsHighRisk(nil))
	assert.False(t, IsHighRisk(&store.PregnancyProfile{}))
	assert.True(t, IsHighRisk(&store.PregnancyProfile{HasMiscarriageHistory: yes}))
	assert.True(t, IsHighRisk(&store.PregnancyProfile{HasHighRiskHistory: yes}))
	assert.True(t, IsHighRisk(&store.PregnancyProfile{RhFactor: ns("negative")}))
	assert.False(t, IsHighRisk(&store.PregnancyProfile{RhFactor: ns("positive")}))
	cond := func(s string) db.NullRawJSON { return db.NullRawJSON{V: json.RawMessage(s), Valid: true} }
	assert.True(t, IsHighRisk(&store.PregnancyProfile{PreExistingConditions: cond(`["hypothyroidism","diabetes"]`)}))
	assert.True(t, IsHighRisk(&store.PregnancyProfile{PreExistingConditions: cond(`{"a":"chronic_hypertension"}`)}))
	assert.False(t, IsHighRisk(&store.PregnancyProfile{PreExistingConditions: cond(`["hypertension"]`)}))
}

func TestStatusJSON(t *testing.T) {
	p := &store.PregnancyProfile{
		PregnancyMode: true, AgeSource: ns("ultrasound"), UltrasoundDate: nd("2026-09-09"),
		UltrasoundWeeks: ni(23), UltrasoundDays: ni(2), RhFactor: ns("negative"),
	}
	b, err := json.Marshal(New(p, "fa", today).Status())
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, true, got["is_active"])
	assert.EqualValues(t, 26, got["current_week"])
	assert.Equal(t, "ultrasound", got["age_source"])
	assert.Equal(t, true, got["fetal_movement_required"])
	flags := got["flags"].(map[string]any)
	assert.Contains(t, flags, "daily_tracking")
	assert.Contains(t, flags, "rh_warning")
	assert.Contains(t, flags, "high_risk")
	assert.NotContains(t, flags, "fetal_movement")
	assert.Equal(t, "هفته 26 بارداری.", flags["week_info"].(map[string]any)["fa"])
	ga := got["gestational_age"].(map[string]any)
	assert.Equal(t, "بالا", ga["confidence_label"])
	assert.Equal(t, map[string]any{"en": "25 weeks, 2 days", "fa": "25 هفته و 2 روز"}, ga["formatted"])

	b, err = json.Marshal(New(nil, "en", today).Status())
	require.NoError(t, err)
	assert.JSONEq(t, `{"is_active":false,"gestational_age":{"weeks":null,"days":null,"total_days":null,"trimester":null,
		"confidence_level":null,"confidence_label":null,"uncertainty_days":null,"formatted":null},"estimated_due_date":null,
		"current_week":null,"trimester":null,"age_source":null,"confidence_level":null,"is_high_risk":false,
		"fetal_movement_tracking_active":false,"fetal_movement_required":false,
		"flags":{"no_profile":{"en":"Please complete pregnancy onboarding to get started.","fa":"لطفاً آنبوردینگ بارداری را تکمیل کنید."}}}`, string(b))
}

func TestDateFormatsAndConception(t *testing.T) {
	d := civildate.MustParse("2027-04-07")
	assert.Equal(t, "April 7, 2027", FormatEnglishDate(d))
	assert.Equal(t, "7 آوریل 2027", FormatPersianDate(d))
	assert.Equal(t, "31 دسامبر 2026", FormatPersianDate(civildate.MustParse("2026-12-31")))
	c := New(&store.PregnancyProfile{AgeSource: ns("lmp"), LmpDate: nd("2026-08-02")}, "en", today)
	b, err := json.Marshal(c.ConceptionDate())
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-08-16","formatted":{"en":"August 16, 2026","fa":"16 اوت 2026"}}`, string(b))
	assert.Nil(t, New(nil, "en", today).ConceptionDate())
}
