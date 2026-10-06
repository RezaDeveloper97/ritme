package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

func TestSummaryVitals_MergedReadings(t *testing.T) {
	d := civildate.MustParse
	in := &Input{VitalsLoaded: true, VitalReadings: VitalReadingsOf([]vitals.Reading{
		{Type: vitals.TypeBP, Date: d("2026-10-05"), Systolic: 120, Diastolic: 80, MeasuredAt: d("2026-10-05").TehranMidnight()},
		{Type: vitals.TypeBP, Date: d("2026-10-04"), Systolic: 130, Diastolic: 84, Source: vitals.SourceLog},
		{Type: vitals.TypeGlucose, Date: d("2026-10-05"), MgDl: 94, Context: vitals.ContextFasting, MeasuredAt: d("2026-10-05").TehranMidnight()},
		{Type: vitals.TypeGlucose, Date: d("2026-10-05"), MgDl: 160, Context: vitals.ContextAfterMeal, MeasuredAt: d("2026-10-05").TehranMidnight()},
		{Type: vitals.TypeGlucose, Date: d("2026-10-03"), MgDl: 100, Context: vitals.ContextRandom, Source: vitals.SourceLog},
		{Type: vitals.TypeHR, Date: d("2026-10-05"), Pulse: 70},
	})}
	v := in.vitals(d("2026-09-01"), d("2026-10-06"))
	assert.Equal(t, 2, v.BPReadings)
	assert.InDelta(t, 125, *v.Systolic, 0.001)
	assert.InDelta(t, 82, *v.Diastolic, 0.001)
	assert.Equal(t, 2, v.GlucoseReadings, "fasting timed + log value; after-meal left out")
	assert.InDelta(t, 97, *v.Glucose, 0.001)
}

func TestMergeVitals_Pregnancy(t *testing.T) {
	d := civildate.MustParse
	at := d("2026-10-05").TehranMidnight()
	in := &PregnancyInput{Start: d("2026-06-01"), Today: d("2026-10-06"), BP: map[civildate.Date]BPReading{
		d("2026-10-05"): {Systolic: 110, Diastolic: 70},
	}, Glucose: []GlucoseReading{
		{Date: d("2026-10-05"), Slot: GlucoseFasting, Value: 99},
		{Date: d("2026-10-04"), Slot: GlucoseFasting, Value: 90},
	}}
	mergeVitals(in, VitalReadingsOf([]vitals.Reading{
		{Type: vitals.TypeBP, Date: d("2026-10-05"), Systolic: 128, Diastolic: 82, MeasuredAt: at},
		{Type: vitals.TypeBP, Date: d("2026-10-05"), Systolic: 142, Diastolic: 88, MeasuredAt: at.Add(1)},
		{Type: vitals.TypeGlucose, Date: d("2026-10-05"), MgDl: 92, Context: vitals.ContextFasting, MeasuredAt: at},
		{Type: vitals.TypeGlucose, Date: d("2026-10-05"), MgDl: 118, Context: vitals.ContextAfterMeal, MeasuredAt: at},
		{Type: vitals.TypeGlucose, Date: d("2026-10-03"), MgDl: 150, Context: vitals.ContextRandom, Source: vitals.SourceLog},
	}))
	assert.Equal(t, BPReading{Systolic: 142, Diastolic: 88}, in.BP[d("2026-10-05")], "highest reading of the day")
	require.Len(t, in.Glucose, 3)
	assert.Equal(t, GlucoseReading{Date: d("2026-10-04"), Slot: GlucoseFasting, Value: 90}, in.Glucose[0])
	assert.Contains(t, in.Glucose, GlucoseReading{Date: d("2026-10-05"), Slot: GlucoseFasting, Value: 92})
	assert.Contains(t, in.Glucose, GlucoseReading{Date: d("2026-10-05"), Slot: GlucoseTwoHour, Value: 118})
}
