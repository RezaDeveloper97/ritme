package enums

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-24: bleeding_log is a Go-only DataSource (D-138) that round-trips as its string and counts as actual data
// wherever user_logged does.
func TestDataSourceBleedingLog_RoundTrip(t *testing.T) {
	assert.Equal(t, "bleeding_log", string(DataSourceBleedingLog))

	raw, err := json.Marshal(DataSourceBleedingLog)
	require.NoError(t, err)
	assert.JSONEq(t, `"bleeding_log"`, string(raw))
	var back DataSource
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, DataSourceBleedingLog, back)

	got, ok := ParseDataSource("bleeding_log")
	assert.True(t, ok)
	assert.Equal(t, DataSourceBleedingLog, got)

	// PHP parity is untouched: the generated list and tryFrom still mirror DataSource.php.
	assert.NotContains(t, DataSourceValues(), "bleeding_log")
	_, ok = DataSourceFrom("bleeding_log")
	assert.False(t, ok)
}

// Every DataSource the Go code knows, with the hand-written predicates over it. A new case must be added here (the
// length check fails otherwise), so no predicate silently misses it.
func TestDataSource_EveryCaseHandled(t *testing.T) {
	want := map[DataSource]struct{ actual bool }{
		DataSourceUserLogged:           {actual: true},
		DataSourceUserProfileConfirmed: {actual: true},
		DataSourceOnboardingEstimate:   {actual: false},
		DataSourceOnboardingInitial:    {actual: false},
		DataSourceSystemDefault:        {actual: false},
		DataSourceEnginePrediction:     {actual: false},
		DataSourceBleedingLog:          {actual: true},
	}
	all := append(DataSourceCases(), DataSourceBleedingLog)
	require.Len(t, want, len(all))
	for _, e := range all {
		w, ok := want[e]
		require.True(t, ok, "%s missing from the table", e)
		assert.True(t, e.IsKnown(), "%s", e)
		assert.Equal(t, w.actual, e.IsActual(), "%s", e)
		assert.Equal(t, DataSourceUserLogged.IsActual(), DataSourceBleedingLog.IsActual())
	}

	for _, v := range []string{"", "unknown_source", "BLEEDING_LOG"} {
		e, ok := ParseDataSource(v)
		assert.False(t, ok, "%q", v)
		assert.Equal(t, v, string(e), "an unknown value is kept as is, never an error")
		assert.False(t, e.IsActual(), "%q", v)
	}
}
