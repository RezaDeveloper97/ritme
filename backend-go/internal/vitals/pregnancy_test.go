package vitals

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func TestPregnancyClassification(t *testing.T) {
	day := civildate.MustParse("2026-10-21")
	bp := func(s, d float64, pregnant bool) (string, bool) {
		r := Reading{Type: TypeBP, Date: day, Systolic: s, Diastolic: d, Pregnant: pregnant}
		return r.Class(), r.Urgent()
	}
	for _, tc := range []struct {
		s, d   float64
		class  string
		urgent bool
	}{
		{139, 89, ClassNormal, false}, {140, 80, ClassStage2, false}, {120, 90, ClassStage2, false},
		{159, 109, ClassStage2, false}, {160, 80, ClassCrisis, true}, {120, 110, ClassCrisis, true},
	} {
		c, u := bp(tc.s, tc.d, true)
		assert.Equal(t, tc.class, c, "%v/%v", tc.s, tc.d)
		assert.Equal(t, tc.urgent, u, "%v/%v", tc.s, tc.d)
	}
	c, u := bp(160, 100, false)
	assert.Equal(t, ClassStage2, c, "not pregnant: ACC/AHA unchanged")
	assert.False(t, u)
	assert.Empty(t, AlertRule(Reading{Type: TypeBP, Date: day, Systolic: 165, Diastolic: 112, Pregnant: true}),
		"the severe range is the pregnancy engine's, not the vitals copy")

	g := func(v float64, ctx string, pregnant bool) string {
		return Reading{Type: TypeGlucose, Date: day, MgDl: v, Context: ctx, Pregnant: pregnant}.Class()
	}
	assert.Equal(t, ClassInRange, g(94, ContextFasting, true))
	assert.Equal(t, ClassHigh, g(95, ContextFasting, true))
	assert.Equal(t, ClassInRange, g(99, ContextFasting, false))
	assert.Equal(t, ClassInRange, g(119, ContextAfterMeal, true))
	assert.Equal(t, ClassHigh, g(120, ContextAfterMeal, true))
	assert.Equal(t, ClassVeryHigh, g(200, ContextAfterMeal, true))
	assert.Equal(t, ClassInRange, g(130, ContextRandom, true), "random keeps the ADA band")
	assert.Equal(t, ClassInRange, g(120, ContextAfterMeal, false))

	_, ok := PregnancyReading(Reading{Type: TypeHR, Pulse: 80})
	assert.False(t, ok)
	in, ok := PregnancyReading(Reading{Type: TypeGlucose, MgDl: 120, Context: ContextAfterMeal})
	assert.True(t, ok)
	assert.Equal(t, "two_hour", in.Slot)
}

func TestPregnancyModalJSON(t *testing.T) {
	cp := AlertCopy{}
	a := jsonx.Obj("rule_key", "bp_high", "level", "urgent", "title", "T", "what_we_saw", "W", "advice", "A",
		"contact", "C", "actions", []any{jsonx.Obj("key", "call", "label", "Call"), jsonx.Obj("key", "add_to_visit_note", "label", "x"),
			jsonx.Obj("key", "ack", "label", "Ok")})
	m := cp.PregnancyModalJSON(a, "en", "fa").(*jsonx.OrderedMap)
	rule, _ := m.Get("rule")
	assert.Equal(t, "pregnancy_bp_high", rule)
	acts, _ := m.Get("actions")
	assert.Len(t, acts, 2)
	phone, _ := acts.([]any)[0].(*jsonx.OrderedMap).Get("phone")
	assert.Equal(t, "115", phone, "the vitals urgent copy's number")
	a.Set("level", "follow_up")
	assert.Nil(t, cp.PregnancyModalJSON(a, "en", "fa"))
}
