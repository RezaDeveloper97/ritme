package labor_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
)

// T-M9-05: 5-1-1 reads the clinical config (pregnancy.contractions_511), not the message_contents params.
func TestParamsOn_RegistryDefaultsAndChanges(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	day := civildate.MustParse("2026-10-21")
	assert.Equal(t, labor.DefaultParams(), labor.ParamsOn(day), "version 1 equals the old constants")
	assert.Equal(t, labor.DefaultParams(), labor.ParamsOn(civildate.Date{}))

	// 13 contractions every 5 min lasting 60 s over 60 min → met; 6-min spacing → not met.
	assert.True(t, labor.FiveOneOne(series(t0, 13, 5*time.Minute, time.Minute), labor.ParamsOn(day)).Met)
	assert.False(t, labor.FiveOneOne(series(t0, 13, 6*time.Minute, time.Minute), labor.ParamsOn(day)).Met)

	// An approved duration_s of 70 s is honoured: 60-s contractions no longer meet the rule.
	clinicalconfig.SetVersions([]clinicalconfig.Version{{
		Key: clinicalconfig.KeyPregnancyContractions511, Version: 2,
		Value:         clinicalconfig.Value{"interval_min": 5, "duration_s": 70, "period_min": 60},
		EffectiveFrom: day,
	}})
	p := labor.ParamsOn(day)
	assert.Equal(t, 70*time.Second, p.DurationMin)
	assert.False(t, labor.FiveOneOne(series(t0, 13, 5*time.Minute, time.Minute), p).Met)
	assert.True(t, labor.FiveOneOne(series(t0, 13, 5*time.Minute, 75*time.Second), p).Met)
	assert.Equal(t, labor.DefaultParams(), labor.ParamsOn(day.AddDays(-1)), "the day before keeps version 1")
}
