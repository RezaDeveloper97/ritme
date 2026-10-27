package bbt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	enginebbt "github.com/ritme/backend-go/internal/cycle/engine/bbt"
	"github.com/ritme/backend-go/internal/fertility/bbt"
)

// The v6 chart rule is the engine's (engine/bbt, E-16, D-128): strict third reading ≥ baseline + 0.20 °C and at
// least MinReadings readings in the cycle; the legacy rule stays for v1.
func TestAnalyzeRule_Engine(t *testing.T) {
	rule := &enginebbt.Params{Strict: true, MinReadings: 10, RiseCenti: 20}
	cycle := bbt.CycleInput{Start: start, End: start.AddDays(27)}

	t.Run("clean biphasic confirms with the engine baseline", func(t *testing.T) {
		rs := series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3640, 3630, 3625, 3635,
			3665, 3670, 3675, 3680)
		c := bbt.AnalyzeRule(cycle, rs, start.AddDays(15), rule)
		assert.Equal(t, ptr(3640), c.Coverline)
		assert.Equal(t, ptr(13), c.ShiftDay)
		assert.Equal(t, ptr(12), c.OvulationDay())
		assert.Equal(t, bbt.PhasePostShift, c.Phase)
	})

	t.Run("strict: third high reading below baseline + 0.20 is rejected", func(t *testing.T) {
		// Days 7-9 are above the 3640 baseline, the third only by 0.10 (3650 < 3660). The legacy rule (first
		// reading ≥ +0.20) confirms it; the engine's strict rule does not.
		rs := series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3665, 3660, 3650, 3630, 3625, 3620)
		legacy := bbt.Analyze(cycle, rs, start.AddDays(11))
		assert.Equal(t, ptr(7), legacy.ShiftDay, "legacy rule confirms")
		c := bbt.AnalyzeRule(cycle, rs, start.AddDays(11), rule)
		assert.Nil(t, c.ShiftDay)
		assert.Equal(t, bbt.PhasePreShift, c.Phase)
		assert.Equal(t, ptr(3640), c.Coverline, "provisional coverline: the baseline of the unconfirmed rise")
	})

	t.Run("nine readings never confirm", func(t *testing.T) {
		rs := series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3665, 3670, 3675)
		assert.Equal(t, ptr(7), bbt.Analyze(cycle, rs, start.AddDays(8)).ShiftDay, "legacy rule confirms")
		c := bbt.AnalyzeRule(cycle, rs, start.AddDays(8), rule)
		assert.Nil(t, c.ShiftDay)
		assert.Equal(t, ptr(3640), c.Coverline)
		// The tenth reading confirms it.
		rs = append(rs, bbt.Reading{Date: start.AddDays(9), Value: 3680})
		assert.Equal(t, ptr(7), bbt.AnalyzeRule(cycle, rs, start.AddDays(9), rule).ShiftDay)
	})

	t.Run("flat readings: coverline is the latest six", func(t *testing.T) {
		rs := series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3645, 3630)
		c := bbt.AnalyzeRule(cycle, rs, start.AddDays(9), rule)
		assert.Nil(t, c.ShiftDay)
		assert.Equal(t, ptr(3645), c.Coverline)
	})

	t.Run("fewer than six readings: no coverline", func(t *testing.T) {
		c := bbt.AnalyzeRule(cycle, series(start, 3620, 3630, 3610), start.AddDays(2), rule)
		assert.Nil(t, c.Coverline)
	})

	t.Run("BuildRule past shift days use the rule", func(t *testing.T) {
		prev := bbt.CycleInput{Start: start.AddDays(-28), End: start.AddDays(-1)}
		rs := series(prev.Start, 3620, 3630, 3610, 3640, 3625, 3635, 3665, 3660, 3650, 3630, 3625, 3620)
		assert.Equal(t, []int{7}, bbt.Build([]bbt.CycleInput{cycle, prev}, rs, start, 1).PastShiftDays)
		assert.Equal(t, []int{}, bbt.BuildRule([]bbt.CycleInput{cycle, prev}, rs, start, 1, rule).PastShiftDays)
	})
}
