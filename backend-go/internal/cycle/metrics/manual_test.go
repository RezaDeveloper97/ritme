package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
)

// B-N1-09: «خودکار از داده‌ها» off → the profile lengths win over the medians; on → unchanged.
func TestCalculate_LengthsManual(t *testing.T) {
	history := []model.History{pe("2026-01-01", "2026-01-05"), pe("2026-01-30", "2026-02-03"), pe("2026-02-28", "2026-03-04")}

	auto := metricsFor(history, profile(model.Int(32), model.Int(7)))
	assert.Equal(t, 29, auto.EffectiveCycleLength)
	assert.Equal(t, 5, auto.EffectivePeriodDuration)
	assert.Equal(t, enums.EffectiveSourceRecentValidCycles, auto.CycleLengthSource)

	pr := profile(model.Int(32), model.Int(7))
	pr.LengthsManual = true
	manual := metricsFor(history, pr)
	assert.Equal(t, 32, manual.EffectiveCycleLength)
	assert.Equal(t, 7, manual.EffectivePeriodDuration)
	assert.Equal(t, enums.EffectiveSourceProfile, manual.CycleLengthSource)
	assert.Equal(t, enums.EffectiveSourceProfile, manual.PeriodDurationSource)
	assert.Equal(t, auto.CalculatedCycleLength, manual.CalculatedCycleLength, "the medians are still reported")

	// Manual without profile values falls back to the medians, then the defaults.
	empty := profile(nil, nil)
	empty.LengthsManual = true
	m := metricsFor(history, empty)
	assert.Equal(t, 29, m.EffectiveCycleLength)
	assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.CycleLengthSource)
	m = metricsFor(nil, empty)
	assert.Equal(t, DefaultCycleLength, m.EffectiveCycleLength)
	assert.Equal(t, enums.EffectiveSourceDefault, m.CycleLengthSource)
}
