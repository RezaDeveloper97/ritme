package clinicalconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The T-M9-15 engine v6 keys (DECISIONS.md X-11): version 1 is the spec value, bounds and order validate like the
// existing keys.
func TestEngineV6Keys_DefaultsBoundsAndOrder(t *testing.T) {
	Reset()
	d := civildate.MustParse("2026-10-21")
	want := map[string]Value{
		KeyLuteal:        {"default": 14, "min": 10, "max": 16},
		KeyBBT:           {"strict": 1, "min_readings": 10, "rise_centi": 20},
		KeyLHBBTConflict: {"days": 2},
		KeyPregnancyTest: {"cycle_days_late": 8, "ttc_days_late": 1},
		KeyStaleAnchor:   {"extra_days": 14},
		KeyTeenFrequency: {"min": 21, "max": 45},
	}
	for key, v := range want {
		def, ok := Lookup(key)
		if !assert.True(t, ok, key) {
			continue
		}
		assert.Equal(t, GroupEngine, def.Group, key)
		assert.Equal(t, srcEngineV6, def.Source, key)
		assert.Equal(t, v, At(key, d).Value, key)
		assert.Equal(t, 1, At(key, d).Version, key)
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

	luteal, _ := Lookup(KeyLuteal)
	assert.Equal(t, "order:min", luteal.Validate(Value{"default": 12, "min": 12, "max": 16}))
	assert.Equal(t, "order:default", luteal.Validate(Value{"default": 16, "min": 10, "max": 16}))
	assert.Empty(t, luteal.Validate(Value{"default": 13, "min": 11, "max": 15}))

	teen, _ := Lookup(KeyTeenFrequency)
	assert.Equal(t, [][2]string{{"min", "max"}}, teen.Order, "the ranges never overlap today; the pair guards later bounds")

	bbt, _ := Lookup(KeyBBT)
	assert.Empty(t, bbt.Validate(Value{"strict": 0, "min_readings": 10, "rise_centi": 20}), "standard mode (E-16)")

	// The v1.1 source string stays on the v1.1 keys.
	hard, _ := Lookup(KeyPeriodHardCap)
	assert.Equal(t, srcEngine, hard.Source)
}
