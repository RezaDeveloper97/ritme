package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The analysis card's pregnancy constants are version 1 of the clinical config the alert engines read (T-M9-05).
func TestPregnancyThresholds_MatchClinicalConfig(t *testing.T) {
	n := func(key, field string) float64 { return clinicalconfig.Num(key, field, civildate.Date{}) }
	assert.InDelta(t, BPHighSystolic, n(clinicalconfig.KeyPregnancyBPHigh, "systolic"), 0)
	assert.InDelta(t, BPHighDiastolic, n(clinicalconfig.KeyPregnancyBPHigh, "diastolic"), 0)
	assert.InDelta(t, BPSevereSystolic, n(clinicalconfig.KeyPregnancyBPSevere, "systolic"), 0)
	assert.InDelta(t, BPSevereDiastolic, n(clinicalconfig.KeyPregnancyBPSevere, "diastolic"), 0)
	assert.InDelta(t, GDMFastingMax, n(clinicalconfig.KeyPregnancyGDMTargets, "fasting"), 0)
	assert.InDelta(t, GDMOneHourMax, n(clinicalconfig.KeyPregnancyGDMTargets, "one_hour"), 0)
	assert.InDelta(t, GDMTwoHourMax, n(clinicalconfig.KeyPregnancyGDMTargets, "two_hour"), 0)
	assert.InDelta(t, KicksTarget, n(clinicalconfig.KeyPregnancyKicks, "target"), 0)
	assert.InDelta(t, KicksWindowMinutes, n(clinicalconfig.KeyPregnancyKicks, "window_min"), 0)
	assert.InDelta(t, KicksFromWeek, n(clinicalconfig.KeyPregnancyKicks, "from_week"), 0)
}
