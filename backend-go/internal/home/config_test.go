package home

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/home/store"
)

// fakeConfigQ serves the whole-table config reads; every other Querier method panics (nil embed).
type fakeConfigQ struct {
	store.Querier
	tasks    []store.ListActiveTaskTemplatesRow
	affs     []store.ListActiveAffirmationsRow
	articles []store.ListPublishedHomeArticlesRow
}

func (f *fakeConfigQ) ListActiveTaskTemplates(context.Context) ([]store.ListActiveTaskTemplatesRow, error) {
	return f.tasks, nil
}

func (f *fakeConfigQ) ListActiveAffirmations(context.Context) ([]store.ListActiveAffirmationsRow, error) {
	return f.affs, nil
}

func (f *fakeConfigQ) ListPublishedHomeArticles(context.Context) ([]store.ListPublishedHomeArticlesRow, error) {
	return f.articles, nil
}

func phaseOf(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func TestForPhase_MatchesTheSQLScope(t *testing.T) {
	q := &fakeConfigQ{tasks: []store.ListActiveTaskTemplatesRow{
		{ID: 1}, {ID: 2, CyclePhase: phaseOf("luteal")}, {ID: 3, CyclePhase: phaseOf("Luteal ")},
		{ID: 4, CyclePhase: phaseOf("menstrual")},
	}}
	ids := func(phase string) []uint64 {
		rows, err := activeTaskTemplates(context.Background(), q, phase)
		require.NoError(t, err)
		out := []uint64{}
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out
	}
	assert.Equal(t, []uint64{1, 2, 3}, ids("luteal"), "NULL phase + equal under utf8mb4_unicode_ci, in snapshot order")
	assert.Equal(t, []uint64{1, 4}, ids("menstrual"))
	assert.Equal(t, []uint64{1}, ids(""), "no phase: only untagged rows")
	assert.Equal(t, []uint64{1}, ids("0"), `"0" is falsy in PHP`)

	q.affs = []store.ListActiveAffirmationsRow{{ID: 7, CyclePhase: phaseOf("follicular")}, {ID: 8}}
	rows, err := activeAffirmations(context.Background(), q, "follicular")
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	rows, err = activeAffirmations(context.Background(), q, "luteal")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, uint64(8), rows[0].ID)
}

func TestJSONOverlaps(t *testing.T) {
	c := []string{"late_luteal", "luteal"}
	assert.True(t, jsonOverlaps(json.RawMessage(`["menstrual","luteal"]`), c))
	assert.False(t, jsonOverlaps(json.RawMessage(`["menstrual"]`), c))
	assert.False(t, jsonOverlaps(json.RawMessage(`[]`), c))
	assert.True(t, jsonOverlaps(json.RawMessage(`"luteal"`), c), "a scalar overlaps an array that contains it")
	assert.False(t, jsonOverlaps(json.RawMessage(`null`), c), "JSON null is not SQL NULL and overlaps nothing")
	assert.False(t, jsonOverlaps(json.RawMessage(`{"luteal":1}`), c))
	assert.False(t, jsonOverlaps(json.RawMessage(`[1,"Luteal"]`), c), "JSON strings compare byte for byte")
}

func TestHomeArticles_TaggedFirstThenUntaggedLimitSix(t *testing.T) {
	tag := func(s string) db.NullRawJSON { return db.NullRawJSON{V: json.RawMessage(s), Valid: true} }
	q := &fakeConfigQ{} // snapshot order = sort_order, id
	for i, p := range []string{"", `["luteal"]`, `["menstrual"]`, "", `["luteal","pms"]`, "", "", "", "", `["luteal"]`} {
		a := store.ListPublishedHomeArticlesRow{ID: uint64(i + 1)} //nolint:gosec // small test index
		if p != "" {
			a.CyclePhases = tag(p)
		}
		q.articles = append(q.articles, a)
	}
	ids := func(c []string) []uint64 {
		rows, err := homeArticles(context.Background(), q, c)
		require.NoError(t, err)
		out := []uint64{}
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out
	}
	assert.Equal(t, []uint64{2, 5, 10, 1, 4, 6}, ids([]string{"luteal"}), "(cycle_phases IS NULL), sort_order, id; limit 6")
	assert.Equal(t, []uint64{1, 4, 6, 7, 8, 9}, ids(nil), "no candidates: untagged only")
	assert.Equal(t, []uint64{3, 1, 4, 6, 7, 8}, ids([]string{"menstrual"}))
}
