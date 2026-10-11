package postpartum

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
	assert.True(t, cur.Urgent)
	assert.Equal(t, BandPossible, Check{Kind: KindFull, Total: 12, TakenOn: civildate.MustParse("2026-09-30")}.Band())
	assert.Equal(t, BandLikely, Check{Kind: KindFull, Total: 12, TakenOn: civildate.MustParse("2026-10-01")}.Band())
}
