package clinicalconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The T-M9-06 keys: version 1 is today's behaviour (EPDS urgent at 13) and the SPEC v6 §3.1 values; each key
// validates its bounds and the visit days stay in order.
func TestPostpartumKeys_DefaultsAndBounds(t *testing.T) {
	Reset()
	d := civildate.MustParse("2026-10-21")
	want := map[string]Value{
		KeyPostpartumRedFlags: {"fever_c": 38.0},
		KeyPostpartumVisits:   {"contact_by_day": 21, "reminder_day": 42, "comprehensive_by_day": 84},
		KeyEPDSUrgentTotal:    {"score": 13},
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
	urgent, _ := Lookup(KeyEPDSUrgentTotal)
	assert.Empty(t, urgent.Validate(Value{"score": 31}), "the advisor may switch the score path off (H-01)")
	visits, _ := Lookup(KeyPostpartumVisits)
	assert.Equal(t, "order:reminder_day", visits.Validate(Value{"contact_by_day": 21, "reminder_day": 84, "comprehensive_by_day": 84}))
}
