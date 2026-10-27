package search

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// fakeStates serves one prepared engine state for me.
type fakeStates struct {
	t  *testing.T
	st *engine.State
}

func (f fakeStates) ForUser(_ context.Context, uid uint64, _ civildate.Date) (*engine.State, error) {
	assert.Equal(f.t, me, uid)
	return f.st, nil
}

func statesWith(t *testing.T, starts ...string) fakeStates {
	var in engine.Input
	for _, s := range starts {
		d := engine.DayOf(civildate.MustParse(s))
		in.Periods = append(in.Periods, engine.Period{Start: d, End: d + 4})
	}
	return fakeStates{t: t, st: engine.Prepare(in, engineparams.For(today))}
}

// T-M9-37 (D-136): under the v6 engine the insight window is the engine's current cycle; 15 days late the cycle is
// unresolved, so there is no "current cycle" window (v1's 60-day rule would still give one) — the last 30 days.
func TestSearch_EngineWindow(t *testing.T) {
	for _, c := range []struct {
		name   string
		starts []string
		window string
	}{
		{"current cycle", []string{"2026-08-16", "2026-09-14"}, `"window":{"kind":"cycle","from":"2026-09-14","to":"2026-09-23"}`},
		{"15 days late", []string{"2026-07-14", "2026-08-11"}, `"window":{"kind":"days","from":"2026-08-25","to":"2026-09-23"}`},
		{"no data", nil, `"window":{"kind":"days","from":"2026-08-25","to":"2026-09-23"}`},
	} {
		logs := painLogs(t, taxonomy.ModeCycle)
		// The v1 period source is never read under v6.
		svc := NewService(logs, fakePeriods{t: t, start: "2026-08-11"}, fakeArticles{}, fakeCheckups{t: t}, fakeReminders{t: t}).
			WithEngine(statesWith(t, c.starts...))
		res, err := svc.Search(context.Background(), query("درد", ScopeMine, "fa"))
		require.NoError(t, err, c.name)
		pain := find(res.Groups[0], "log_insight", "pain")
		require.NotNil(t, pain, c.name)
		b, _ := json.Marshal(pain.Meta)
		assert.Contains(t, string(b), c.window, c.name)
	}

	// v1 keeps its 60-day rule: the same 43-day-old start is still a cycle window.
	logs := painLogs(t, taxonomy.ModeCycle)
	svc := NewService(logs, fakePeriods{t: t, start: "2026-08-11"}, fakeArticles{}, fakeCheckups{t: t}, fakeReminders{t: t})
	res, err := svc.Search(context.Background(), query("درد", ScopeMine, "fa"))
	require.NoError(t, err)
	b, _ := json.Marshal(find(res.Groups[0], "log_insight", "pain").Meta)
	assert.Contains(t, string(b), `"window":{"kind":"cycle","from":"2026-08-11"`)
}
