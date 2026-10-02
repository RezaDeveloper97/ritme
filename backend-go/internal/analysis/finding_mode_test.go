package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// B-N3-14b (N3 stage smoke B-5): a menopause user is never asked to «log 3 more cycles».
func TestTopFinding_NoCycleNudge(t *testing.T) {
	f := BuildTopFinding(CycleReport{}, SymptomsReport{DaysLogged: 12})
	assert.Equal(t, FindingNotEnoughData, f.Kind)

	m := f.withoutCycleNudge()
	assert.Equal(t, FindingKeepLogging, m.Kind)
	assert.Equal(t, []Phrase{{Key: "findings.keep_logging"}}, m.Parts)

	none := BuildTopFinding(CycleReport{}, SymptomsReport{})
	assert.Equal(t, FindingNoData, none.withoutCycleNudge().Kind, "other kinds are kept")
}
