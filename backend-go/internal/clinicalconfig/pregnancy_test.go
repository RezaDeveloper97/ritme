package clinicalconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The T-M9-05 keys: version 1 equals the constants the engines used before (plus the new escalations), and each key
// validates its bounds.
func TestPregnancyKeys_DefaultsAndBounds(t *testing.T) {
	Reset()
	d := civildate.MustParse("2026-10-21")
	want := map[string]Value{
		KeyPregnancyBPHigh:          {"systolic": 140, "diastolic": 90, "urgent": 1},
		KeyPregnancyBPSevere:        {"systolic": 160, "diastolic": 110},
		KeyPregnancyGDMTargets:      {"fasting": 95, "one_hour": 140, "two_hour": 120},
		KeyPregnancyKicks:           {"target": 10, "window_min": 120, "from_week": 28},
		KeyPregnancyManualFetalWeek: {"week": 24},
		KeyPregnancyContractions511: {"interval_min": 5, "duration_s": 45, "period_min": 60},
	}
	for key, v := range want {
		def, ok := Lookup(key)
		if !assert.True(t, ok, key) {
			continue
		}
		assert.Equal(t, GroupPregnancy, def.Group, key)
		assert.NotEmpty(t, def.Source, key)
		assert.Equal(t, v, At(key, d).Value, key)
		assert.Empty(t, def.Validate(v), key)
		for _, f := range def.Fields {
			low := def.DefaultValue()
			low[f.Name] = f.Min - 1
			assert.Equal(t, "range:"+f.Name, def.Validate(low), key)
			high := def.DefaultValue()
			high[f.Name] = f.Max + 1
			assert.Equal(t, "range:"+f.Name, def.Validate(high), key)
		}
	}
	urgent, _ := Lookup(KeyPregnancyBPHigh)
	assert.Empty(t, urgent.Validate(Value{"systolic": 140, "diastolic": 90, "urgent": 0}), "the advisor may set follow_up (H-02)")
}
