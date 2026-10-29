package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// B-N3-14b (N3 stage smoke B-5): a menopause user is never asked to log more cycles; without a non-cycle minimum to
// name (no Plus) the nudge is the plain keep-logging line.
func TestTopFinding_NoCycleNudge(t *testing.T) {
	in := ranged(emptyFixture(), Range6M)
	f := BuildTopFinding(in, CycleReport{}, SymptomsReport{DaysLogged: 12}, nil)
	assert.Equal(t, FindingNotEnoughData, f.Kind)
	assert.Equal(t, "findings.not_enough_days", f.Parts[0].Key)

	in.NoCycleNudge = true
	m := BuildTopFinding(in, CycleReport{}, SymptomsReport{DaysLogged: 12}, nil)
	assert.Equal(t, FindingKeepLogging, m.Kind)
	assert.Equal(t, []Phrase{{Key: "findings.keep_logging"}}, m.Parts)
}
