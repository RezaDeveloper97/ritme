package tools

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

var t0 = time.Date(2026, 10, 3, 21, 0, 0, 0, civildate.Tehran)

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func TestSummarizeDay(t *testing.T) {
	sessions := []store.PregnancyKickSession{
		// 10 kicks, the 10th after 24 min → normal.
		{StartedAt: t0, Kicks: 12, TenthKickAt: nt(t0.Add(24 * time.Minute)), LastKickAt: nt(t0.Add(30 * time.Minute))},
		{StartedAt: t0.Add(-5 * time.Hour), Kicks: 0},
		{StartedAt: t0.Add(-3 * time.Hour), Kicks: 4, LastKickAt: nt(t0.Add(-2 * time.Hour))},
	}
	d := SummarizeDay(sessions)
	assert.Equal(t, 16, d.Kicks)
	assert.Equal(t, t0.Add(-3*time.Hour), d.First, "the empty session does not count")
	assert.Equal(t, t0.Add(30*time.Minute), d.Last)
	assert.True(t, d.Normal)

	// 10 reached only after the 2-hour window → not normal.
	d = SummarizeDay([]store.PregnancyKickSession{{StartedAt: t0, Kicks: 10, TenthKickAt: nt(t0.Add(121 * time.Minute))}})
	assert.False(t, d.Normal)
	assert.Equal(t, t0, d.Last, "no last kick time → the start")
	assert.Equal(t, FetalDay{}, SummarizeDay(nil))
}

func TestKickJSON(t *testing.T) {
	k := store.PregnancyKickSession{ID: 3, StartedAt: t0, Kicks: 8, ActiveLock: sql.NullInt16{Int16: 1, Valid: true},
		PregnancyWeek: sql.NullInt16{Int16: 32, Valid: true}, LastKickAt: nt(t0.Add(20 * time.Minute))}
	raw, err := json.Marshal(KickJSON(k, t0.Add(24*time.Minute+10*time.Second)))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":3,"started_at":"2026-10-03T21:00:00+03:30","ended_at":null,"is_active":true,"kicks":8,
		"target":10,"window_minutes":120,"elapsed_seconds":1450,"time_to_target_seconds":null,"reached_target":false,
		"low_count":false,"last_kick_at":"2026-10-03T21:20:00+03:30","pregnancy_week":32}`, string(raw))

	// Past the window under the target: low_count (the call guidance).
	k.ActiveLock, k.EndedAt = sql.NullInt16{}, nt(t0.Add(2*time.Hour))
	m := KickJSON(k, t0.Add(3*time.Hour))
	v, _ := m.Get("low_count")
	assert.Equal(t, true, v)
	v, _ = m.Get("elapsed_seconds")
	assert.Equal(t, int64(7200), v)
}

func TestTimingJSON(t *testing.T) {
	var cs []store.PregnancyContraction
	for i := range 13 {
		s := t0.Add(time.Duration(i) * 5 * time.Minute)
		cs = append(cs, store.PregnancyContraction{ID: uint64(i + 1), StartedAt: s, EndedAt: nt(s.Add(time.Minute))})
	}
	cs = append(cs, store.PregnancyContraction{ID: 14, StartedAt: t0.Add(65 * time.Minute)})
	tm := Timing{Session: store.PregnancyContractionSession{ID: 9, StartedAt: t0, ActiveLock: sql.NullInt16{Int16: 1, Valid: true}}, Contractions: cs}
	raw, err := json.Marshal(TimingJSON(tm, labor.DefaultParams(), t0.Add(65*time.Minute+48*time.Second), true))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.InDelta(t, 13, m["count"], 0)
	assert.InDelta(t, 48, m["running"].(map[string]any)["elapsed_seconds"], 0)
	assert.InDelta(t, 60, m["avg_duration_seconds"], 0)
	assert.InDelta(t, 300, m["avg_interval_seconds"], 0)
	f := m["five_one_one"].(map[string]any)
	assert.Equal(t, true, f["met"])
	assert.InDelta(t, 61, f["span_minutes"], 0)
	list := m["contractions"].([]any)
	require.Len(t, list, 14)
	first := list[0].(map[string]any)
	assert.InDelta(t, 14, first["id"], 0, "newest first")
	assert.Nil(t, first["duration_seconds"])
	assert.InDelta(t, 300, first["interval_seconds"], 0)
	assert.Nil(t, list[13].(map[string]any)["interval_seconds"], "the first contraction has no interval")
}

func TestLangFilesHaveTheSameKeys(t *testing.T) {
	keys := func(code string) []string {
		raw, err := fs.ReadFile(langFS, "lang/"+code+"/pregnancy_tools.json")
		require.NoError(t, err)
		var m map[string]map[string]string
		require.NoError(t, json.Unmarshal(raw, &m))
		var out []string
		for g, kv := range m {
			for k := range kv {
				out = append(out, g+"."+k)
			}
		}
		return out
	}
	dirs, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, d := range dirs {
		assert.ElementsMatch(t, keys("en"), keys(d.Name()), d.Name())
	}
	assert.Equal(t, "Saved", T("messages.saved", "en"))
}
