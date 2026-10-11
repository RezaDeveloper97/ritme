package safety

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// TestUrgentScenarios is the nightly urgent-path run (.github/workflows/safety-scenarios-nightly.yml): every
// scenario must give its verdict with the clinical config in force today.
func TestUrgentScenarios(t *testing.T) {
	now := time.Now()
	r := RunScenarios(civildate.InTehran(now), now)
	assert.Empty(t, r.Failed)
	assert.Equal(t, len(Scenarios), r.Passed)
	assert.GreaterOrEqual(t, r.Total, 15)
}

func TestRunScenarios_ReportsFailuresAndPanics(t *testing.T) {
	saved := Scenarios
	t.Cleanup(func() { Scenarios = saved })
	Scenarios = []Scenario{
		{"ok", func(civildate.Date) bool { return true }},
		{"wrong", func(civildate.Date) bool { return false }},
		{"boom", func(civildate.Date) bool { panic("x") }},
	}
	r := RunAndRecord(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now())
	assert.Equal(t, 1, r.Passed)
	assert.Equal(t, []string{"wrong", "boom"}, r.Failed)
	require.NotNil(t, Last())
	assert.Equal(t, 3, Last().Total)
}

func TestNextNightly(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 11, h, m, 0, 0, civildate.Tehran) }
	assert.Equal(t, at(3, 0), nextNightly(at(1, 0)))
	assert.Equal(t, at(3, 0).AddDate(0, 0, 1), nextNightly(at(3, 0)))
	assert.Equal(t, at(3, 0).AddDate(0, 0, 1), nextNightly(at(22, 30)))
}

func TestReducedKicks(t *testing.T) {
	assert.True(t, ReducedKicks(6, 2*time.Hour))
	assert.False(t, ReducedKicks(6, time.Hour))
	assert.False(t, ReducedKicks(10, 3*time.Hour))
}

func TestWindows(t *testing.T) {
	cur, prev := windows(time.Date(2026, 10, 11, 15, 0, 0, 0, civildate.Tehran))
	assert.Equal(t, "2026-10-05", cur.fromDay.String())
	assert.Equal(t, "2026-10-12", cur.toDayEx.String())
	assert.Equal(t, "2026-09-28", prev.fromDay.String())
	assert.Equal(t, cur.from, prev.to)
}
