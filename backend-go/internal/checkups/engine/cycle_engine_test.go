package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cycleengine "github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func cday(s string) cycleengine.Day { return cycleengine.DayOf(civildate.MustParse(s)) }

func cperiod(s string) cycleengine.Period {
	return cycleengine.Period{Start: cday(s), End: cday(s) + 4}
}

func engineCycle(in cycleengine.Input, today string) Cycle {
	return CycleFromEngine(cycleengine.Prepare(in, cycleengine.DefaultParams()), civildate.MustParse(today))
}

var hist29 = []cycleengine.Period{cperiod("2026-06-01"), cperiod("2026-06-30"), cperiod("2026-07-29"), cperiod("2026-08-27")}

func TestCycleFromEngine_ManualLengthsAndChain(t *testing.T) {
	c := engineCycle(cycleengine.Input{Periods: hist29}, "2026-09-05")
	assert.Equal(t, civildate.MustParse("2026-08-27"), c.Start)
	assert.Equal(t, 29, c.Length)
	assert.Equal(t, civildate.MustParse("2026-09-25"), c.Future[0], "FutureCycles k=1")

	m := engineCycle(cycleengine.Input{
		Periods:  hist29,
		Declared: cycleengine.Declared{CycleLength: cycleengine.Some(33), Manual: true},
	}, "2026-09-05")
	assert.Equal(t, 33, m.Length, "manual lengths are honoured (the v1 profile built here had none)")
	assert.Equal(t, civildate.MustParse("2026-09-29"), m.Future[0])
}

func TestCycleFromEngine_NoAnchorIsZero(t *testing.T) {
	assert.Equal(t, Cycle{}, engineCycle(cycleengine.Input{}, "2026-09-05"))
}

func TestCycle_FutureChainDrivesTheProjections(t *testing.T) {
	c := engineCycle(cycleengine.Input{Periods: hist29}, "2026-09-05")
	assert.Equal(t, civildate.MustParse("2026-08-27"), c.startOnOrBefore(civildate.MustParse("2026-09-24")))
	assert.Equal(t, civildate.MustParse("2026-09-25"), c.startOnOrBefore(civildate.MustParse("2026-09-25")))
	assert.Equal(t, civildate.MustParse("2026-09-25"), c.startAfter(civildate.MustParse("2026-08-27")))
	// Beyond the chain the Start + k·Length rule continues from its last element.
	last := c.Future[len(c.Future)-1]
	assert.Equal(t, last.AddDays(29), c.startOnOrBefore(last.AddDays(30)))

	// A late cycle (today 2026-10-02, predicted 09-25): the next start is the chain's, not the past one.
	late := engineCycle(cycleengine.Input{Periods: hist29}, "2026-10-02")
	assert.True(t, late.startAfter(late.Start).After(civildate.MustParse("2026-10-02")))
}
