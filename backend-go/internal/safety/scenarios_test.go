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
	"github.com/ritme/backend-go/internal/platform/queue"
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

func TestNightlyCron_0300Tehran(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 11, h, m, 0, 0, civildate.Tehran) }
	next := func(after time.Time) time.Time {
		n, err := queue.Next(NightlyCron, after)
		require.NoError(t, err)
		return n
	}
	assert.True(t, at(3, 0).Equal(next(at(1, 0))))
	assert.True(t, at(3, 0).AddDate(0, 0, 1).Equal(next(at(3, 0))))
	assert.True(t, at(3, 0).AddDate(0, 0, 1).Equal(next(at(22, 30))))
	assert.True(t, at(3, 0).Equal(next(at(1, 0).UTC())), "the schedule is Tehran wall clock whatever the input zone")
}

// The nightly handler run twice records an equal run (the scenarios are pure for a given day).
func TestNightlyJob_Idempotent(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	fixed := time.Date(2026, 10, 21, 3, 0, 0, 0, civildate.Tehran)
	h := NightlyJob(quiet, func() time.Time { return fixed })
	require.NoError(t, h(context.Background(), []byte("{}")))
	first := *Last()
	require.NoError(t, h(context.Background(), []byte("{}")))
	assert.Equal(t, first, *Last())
	assert.Equal(t, len(Scenarios), first.Total)
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
