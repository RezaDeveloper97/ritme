package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T-M9-28 (I-02), T-M10-08b: a `flow` day-log value in pregnancy is the day's bleeding symptom — heavy flow severe,
// medium moderate, low mild; no flow, nothing.
func TestFlowSeverity(t *testing.T) {
	cases := []struct {
		intensity, severity string
		rank                int
	}{
		{"low", "mild", 1},
		{"medium", "moderate", 1},
		{"high", "severe", 2},
		{"very_high", "severe", 2},
		{"", "", 0},
		{"unknown", "", 0},
	}
	for _, c := range cases {
		assert.Equal(t, c.severity, FlowSeverity(c.intensity), c.intensity)
		assert.Equal(t, c.rank, FlowRank(c.intensity), c.intensity)
	}
	assert.Less(t, SeverityRank("mild"), SeverityRank("moderate"))
	assert.Less(t, SeverityRank("moderate"), SeverityRank("severe"))
	assert.Zero(t, SeverityRank(""))
}
