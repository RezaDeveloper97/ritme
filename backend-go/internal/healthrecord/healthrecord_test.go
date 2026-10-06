package healthrecord

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Saved", T("messages.saved", "en"))
	assert.Equal(t, "ذخیره شد", T("messages.saved", "fa"))
	assert.Equal(t, "You can add at most 20 pregnancies.", Tp("validation.too_many_pregnancies", map[string]string{"max": "20"}, "en"))
	assert.NotEmpty(t, attributes("fa"))
}

func TestCleanAllergies(t *testing.T) {
	assert.Equal(t, []string{"Penicillin", "Pollen dust"}, cleanAllergies([]any{" Penicillin ", "PENICILLIN", "Pollen   dust"}))
	assert.Equal(t, []string{}, cleanAllergies([]any{}))
	assert.Nil(t, cleanAllergies([]any{"a", "  "}))
}

func d(s string) civildate.Date { return civildate.MustParse(s) }

func TestTrackedPregnancies_EntriesOrderAndLossOnlyEnded(t *testing.T) {
	tr := TrackedPregnancies{
		Active: true, Due: d("2027-03-01"),
		Birth:     &store.GetRecordBirthRow{BirthDate: d("2024-06-10"), BabyCount: 2},
		LossCount: 1,
		ManualRows: []store.HealthRecordPregnancy{
			{ID: 7, Outcome: OutcomeVaginal, EndedOn: civildate.NullDate{Date: d("2019-01-01"), Valid: true},
				BabyCount: sql.NullInt16{Int16: 1, Valid: true}},
		},
	}
	es := tr.Entries()
	require.Len(t, es, 4)
	assert.Equal(t, OutcomeOngoing, es[0].Outcome)
	assert.Equal(t, OutcomeBirth, es[1].Outcome, "delivery type not told")
	assert.Equal(t, 2, es[1].BabyCount)
	assert.Equal(t, OutcomeVaginal, es[2].Outcome)
	assert.Equal(t, OutcomeEnded, es[3].Outcome)
	assert.True(t, es[3].Date.IsZero(), "a loss carries no date")

	sec := pregnanciesSection(es, AudienceOwner)
	b, err := json.Marshal(sec)
	require.NoError(t, err)
	assert.JSONEq(t, `{"pregnancies_count":4,"births_count":2,"items":[
		{"id":null,"source":"tracked","outcome":"ongoing","date":"2027-03-01","baby_count":null,"editable":false},
		{"id":null,"source":"tracked","outcome":"birth","date":"2024-06-10","baby_count":2,"editable":false},
		{"id":7,"source":"manual","outcome":"vaginal","date":"2019-01-01","baby_count":1,"editable":true},
		{"id":null,"source":"tracked","outcome":"ended","date":null,"baby_count":null,"editable":false}]}`, string(b))

	share, err := json.Marshal(pregnanciesSection(es, AudienceShare))
	require.NoError(t, err)
	assert.NotContains(t, string(share), `"id":7`)
	assert.NotContains(t, string(share), `"editable":true`)
}

func TestVitalsSummary(t *testing.T) {
	from, to := d("2026-09-07"), d("2026-10-06")
	rs := []vitals.Reading{
		{Type: vitals.TypeBP, Date: d("2026-10-05"), Systolic: 116, Diastolic: 74},
		{Type: vitals.TypeBP, Date: d("2026-10-04"), Systolic: 132, Diastolic: 86},
		{Type: vitals.TypeBP, Date: d("2026-08-01"), Systolic: 200, Diastolic: 130}, // outside
		{Type: vitals.TypeHR, Date: d("2026-10-05"), Pulse: 64, Context: vitals.HRResting},
		{Type: vitals.TypeHR, Date: d("2026-10-05"), Pulse: 140, Context: vitals.HRAfterExercise}, // not resting
		{Type: vitals.TypeGlucose, Date: d("2026-10-05"), MgDl: 89, Context: vitals.ContextFasting},
		{Type: vitals.TypeGlucose, Date: d("2026-10-04"), MgDl: 104, Context: vitals.ContextFasting},
		{Type: vitals.TypeGlucose, Date: d("2026-10-04"), MgDl: 148, Context: vitals.ContextAfterMeal},
		{Type: vitals.TypeGlucose, Date: d("2026-10-03"), MgDl: 101, Context: vitals.ContextRandom},
	}
	m, has := VitalsSummary(rs, from, to)
	require.True(t, has)
	b, err := json.Marshal(m)
	require.NoError(t, err)
	assert.JSONEq(t, `{"from":"2026-09-07","to":"2026-10-06","days":30,
		"blood_pressure":{"systolic":124,"diastolic":80,"min":{"systolic":116,"diastolic":74},"max":{"systolic":132,"diastolic":86},
			"readings":2,"classification":"stage1","tone":"high","unit":"mmhg"},
		"heart_rate":{"avg":64,"min":64,"max":64,"readings":1,"unit":"bpm"},
		"glucose_fasting":{"avg":97,"min":89,"max":104,"readings":2,"in_target":1,"in_target_percent":50,"unit":"mg_dl"},
		"glucose_after_meal":{"avg":148,"min":148,"max":148,"readings":1,"in_target":0,"in_target_percent":0,"unit":"mg_dl"},
		"glucose_other":{"avg":101,"min":101,"max":101,"readings":1,"in_target":1,"in_target_percent":100,"unit":"mg_dl"}}`, string(b))

	_, has = VitalsSummary(nil, from, to)
	assert.False(t, has)
}
