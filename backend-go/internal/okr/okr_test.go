package okr_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/okr"
)

func day(s string) time.Time { return okr.ParseDay(s) }

func TestProgress(t *testing.T) {
	assert.InDelta(t, 0.5, okr.Progress(29, 35, 32), 1e-9)
	assert.InDelta(t, 0.0, okr.Progress(29, 35, 27), 0, "below baseline clamps to 0")
	assert.InDelta(t, 1.0, okr.Progress(29, 35, 40), 0, "beyond target clamps to 1")
	assert.InDelta(t, 0.5, okr.Progress(20, 10, 15), 1e-9, "decreasing target (lower is better)")
	assert.InDelta(t, 1.0, okr.Progress(10, 10, 10), 0)
}

func TestStatusOf(t *testing.T) {
	assert.Equal(t, okr.StatusDone, okr.StatusOf(1, 0.5))
	assert.Equal(t, okr.StatusOnTrack, okr.StatusOf(0.45, 0.5), "above 80% of pace")
	assert.Equal(t, okr.StatusAtRisk, okr.StatusOf(0.3, 0.5))
	assert.Equal(t, okr.StatusOnTrack, okr.StatusOf(0, 0), "period not started")
}

func TestElapsed(t *testing.T) {
	assert.InDelta(t, 0, okr.Elapsed(day("2026-10-01"), day("2026-10-10"), day("2026-09-30")), 0)
	assert.InDelta(t, 0.5, okr.Elapsed(day("2026-10-01"), day("2026-10-10"), day("2026-10-05")), 1e-9)
	assert.InDelta(t, 1, okr.Elapsed(day("2026-10-01"), day("2026-10-10"), day("2026-11-01")), 0)
}

func TestEvaluateWith(t *testing.T) {
	obj := okr.Objective{ID: 1, StartsOn: day("2026-10-01"), EndsOn: day("2026-10-10"), KeyResults: []okr.KeyResult{
		{ID: 1, MetricKey: sql.NullString{String: "dau", Valid: true}, Baseline: 100, Target: 200},
		{ID: 2, Baseline: 0, Target: 10, ManualValue: sql.NullFloat64{Float64: 10, Valid: true}},
		{ID: 3, MetricKey: sql.NullString{String: "mau", Valid: true}, Baseline: 0, Target: 10},
	}}
	ev := okr.EvaluateWith([]okr.Objective{obj}, okr.Values{"dau": 120}, day("2026-10-05"))[0]
	assert.Equal(t, okr.StatusAtRisk, ev.KRs[0].Status, "20% done at 50% of the period")
	assert.Equal(t, okr.StatusDone, ev.KRs[1].Status)
	assert.Equal(t, okr.StatusNoData, ev.KRs[2].Status, "no live value")
	assert.Nil(t, ev.KRs[2].Progress)
	assert.InDelta(t, 0.6, *ev.Progress, 1e-9, "mean over the key results that have data")
	assert.Equal(t, okr.StatusOnTrack, ev.Status)
	s := okr.Summarize([]okr.EvaluatedObjective{ev}, day("2026-10-05"))
	assert.Equal(t, okr.Summary{Objectives: 1, KeyResults: 3, AtRisk: 1, Done: 1, NoData: 1}, *s)
	assert.Nil(t, okr.Summarize([]okr.EvaluatedObjective{ev}, day("2026-12-01")), "no active objective")
}
