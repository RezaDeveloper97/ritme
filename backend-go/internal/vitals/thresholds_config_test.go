package vitals

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// B-N9-11: a reading is classified with the thresholds valid on its own day, so historic outputs stay reproducible
// after a threshold changes.
func TestReadingsUseTheVersionOfTheirDay(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{
		{Key: clinicalconfig.KeyBPCrisis, Version: 2, Value: clinicalconfig.Value{"systolic": 170, "diastolic": 110},
			EffectiveFrom: civildate.MustParse("2026-10-01")},
		{Key: clinicalconfig.KeyGlucoseUrgentLow, Version: 2, Value: clinicalconfig.Value{"mgdl": 60},
			EffectiveFrom: civildate.MustParse("2026-10-01")},
	})
	old := Reading{Type: TypeBP, Date: civildate.MustParse("2026-09-30"), Systolic: 175, Diastolic: 95}
	assert.Equal(t, ClassStage2, old.Class(), "the old reading keeps its version-1 class")
	assert.False(t, old.Urgent())
	assert.Empty(t, AlertRule(old))

	cur := old
	cur.Date = civildate.MustParse("2026-10-01")
	assert.Equal(t, ClassCrisis, cur.Class())
	assert.Equal(t, RuleBPCrisis, AlertRule(cur))

	g := Reading{Type: TypeGlucose, Date: civildate.MustParse("2026-09-30"), MgDl: 58, Context: ContextFasting}
	assert.Equal(t, ClassLow, g.Class())
	g.Date = civildate.MustParse("2026-10-02")
	assert.Equal(t, ClassUrgentLow, g.Class())

	assert.Equal(t, ThresholdsVersion, ThresholdsOn(civildate.MustParse("2026-09-30")).Version())
	assert.Equal(t, ThresholdsVersion+"+bp.crisis@2,glucose.urgent_low@2",
		ThresholdsOn(civildate.MustParse("2026-10-01")).Version())
	raw, err := marshal(ThresholdsJSON(ThresholdsOn(civildate.MustParse("2026-10-01"))))
	assert.NoError(t, err)
	assert.Contains(t, string(raw), "systolic > 170 and/or diastolic > 110")
}

func TestThresholdsJSON_VersionOneRules(t *testing.T) {
	clinicalconfig.Reset()
	raw, err := marshal(ThresholdsJSON(ThresholdsOn(civildate.MustParse("2026-10-11"))))
	assert.NoError(t, err)
	for _, rule := range []string{"systolic < 120 and diastolic < 80", "systolic 120–129 and diastolic < 80",
		"systolic 130–139 or diastolic 80–89", "systolic ≥ 140 or diastolic ≥ 90", "systolic > 180 and/or diastolic > 120",
		`"version":"` + ThresholdsVersion + `"`} {
		assert.Contains(t, string(raw), rule)
	}
}

func marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	return b.Bytes(), err
}
