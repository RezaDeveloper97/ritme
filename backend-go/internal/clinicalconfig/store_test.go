package clinicalconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestDefaultsAreVersionOne(t *testing.T) {
	Reset()
	d := civildate.MustParse("2026-10-11")
	for _, def := range Defs {
		v := At(def.Key, d)
		assert.Equal(t, 1, v.Version, def.Key)
		assert.Equal(t, def.DefaultValue(), v.Value, def.Key)
		assert.Empty(t, def.Validate(def.DefaultValue()), "default of %s must pass its own validation", def.Key)
	}
	assert.Equal(t, 180.0, Num(KeyBPCrisis, "systolic", d))
	assert.Equal(t, 12, Int(KeyPeriodHardCap, "days", d))
	assert.Empty(t, Fingerprint(d, KeyBPCrisis))
	assert.Equal(t, 0, At("nope", d).Version)
}

func TestEffectiveFromIsRespected(t *testing.T) {
	t.Cleanup(Reset)
	SetVersions([]Version{
		{Key: KeyBPCrisis, Version: 3, Value: Value{"systolic": 160, "diastolic": 100}, EffectiveFrom: civildate.MustParse("2026-12-01")},
		{Key: KeyBPCrisis, Version: 2, Value: Value{"systolic": 170}, EffectiveFrom: civildate.MustParse("2026-10-01")},
		{Key: "unknown", Version: 2, Value: Value{"x": 1}, EffectiveFrom: civildate.MustParse("2026-01-01")},
	})
	old := civildate.MustParse("2026-09-30")
	assert.Equal(t, 1, At(KeyBPCrisis, old).Version, "before the first effective date → version 1")
	assert.Equal(t, 180.0, Num(KeyBPCrisis, "systolic", old))
	on := civildate.MustParse("2026-10-01")
	assert.Equal(t, 2, At(KeyBPCrisis, on).Version)
	assert.Equal(t, 170.0, Num(KeyBPCrisis, "systolic", on))
	assert.Equal(t, 120.0, Num(KeyBPCrisis, "diastolic", on), "a missing field falls back to the code default")
	assert.Equal(t, 160.0, Num(KeyBPCrisis, "systolic", civildate.MustParse("2027-01-01")))
	assert.Equal(t, 180.0, Num(KeyBPCrisis, "systolic", civildate.Date{}), "a zero date reads version 1")
	assert.Equal(t, "bp.crisis@2", Fingerprint(on, KeyBPElevated, KeyBPCrisis))
}

func TestValidate(t *testing.T) {
	def, _ := Lookup(KeyHRResting)
	assert.Empty(t, def.Validate(Value{"min": 55, "max": 100}))
	assert.Equal(t, "fields", def.Validate(Value{"min": 55}))
	assert.Equal(t, "fields", def.Validate(Value{"min": 55, "x": 1}))
	assert.Equal(t, "range:min", def.Validate(Value{"min": 5, "max": 100}))
	assert.Equal(t, "order:min", def.Validate(Value{"min": 100, "max": 100}))
}
