package postpartum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// B-N9-11: a check keeps the band of the cut-offs valid on the day it was taken.
func TestEPDSUsesTheCutoffsOfItsDay(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyEPDSLikely, Version: 2,
		Value: clinicalconfig.Value{"score": 12}, EffectiveFrom: civildate.MustParse("2026-10-01")}})
	answers := map[string]int{"q1": 2, "q2": 2, "q3": 2, "q4": 2, "q5": 2, "q6": 2}
	old := ScoreOn(KindFull, answers, civildate.MustParse("2026-09-30"))
	assert.Equal(t, 12, old.Total)
	assert.Equal(t, BandPossible, old.Band)
	assert.False(t, old.Urgent)
	cur := ScoreOn(KindFull, answers, civildate.MustParse("2026-10-01"))
	assert.Equal(t, BandLikely, cur.Band)
	assert.False(t, cur.Urgent, "urgency is its own key since T-M9-06 (epds.urgent_total, 13)")
	assert.Equal(t, BandPossible, Check{Kind: KindFull, Total: 12, TakenOn: civildate.MustParse("2026-09-30")}.Band())
	assert.Equal(t, BandLikely, Check{Kind: KindFull, Total: 12, TakenOn: civildate.MustParse("2026-10-01")}.Band())
}

// T-M9-06 / H-01: the urgent total is config; 13 is today's behaviour, 31 switches the score path off. Item 10 > 0 is
// always urgent.
func TestEPDSUrgentTotalIsConfig(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	day := civildate.MustParse("2026-10-21")
	q10Low := ScoreOn(KindFull, map[string]int{"q1": 1, "q2": 1, "q3": 1, "q10": 1}, day)
	assert.Equal(t, 4, q10Low.Total)
	assert.Equal(t, BandLow, q10Low.Band)
	assert.True(t, q10Low.Urgent)
	assert.Equal(t, []string{ReasonSelfHarm}, q10Low.Reasons)

	at := func(total int) Result {
		answers := map[string]int{"q10": 0}
		for i, it := range Items[:9] {
			answers[it.Code] = min(3, max(0, total-3*i))
		}
		r := ScoreOn(KindFull, answers, day)
		require.Equal(t, total, r.Total)
		return r
	}
	assert.Equal(t, BandLow, at(9).Band)
	assert.False(t, at(9).Urgent)
	assert.Equal(t, BandPossible, at(10).Band)
	assert.False(t, at(10).Urgent)
	r13 := at(13)
	assert.Equal(t, BandLikely, r13.Band)
	assert.True(t, r13.Urgent, "default 13 = today's behaviour")
	assert.Equal(t, []string{ReasonScore}, r13.Reasons)

	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyEPDSUrgentTotal, Version: 2,
		Value: clinicalconfig.Value{"score": 31}, EffectiveFrom: civildate.MustParse("2026-10-01")}})
	r13 = at(13)
	assert.Equal(t, BandLikely, r13.Band)
	assert.False(t, r13.Urgent)
	assert.False(t, at(27).Urgent, "the top total without item 10 (9 × 3) is not urgent when off")
	assert.True(t, ScoreOn(KindFull, map[string]int{"q10": 3}, day).Urgent, "item 10 stays urgent")
}
