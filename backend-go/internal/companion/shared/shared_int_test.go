package shared_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func TestReader_SectionsAreFilteredSubsets(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120006001', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	owner := uint64(id) //nolint:gosec // test id
	other, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Nina', '09120006002', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	otherID, err := other.LastInsertId()
	require.NoError(t, err)

	entry := `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NULL, ?, 'manual', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`
	for _, row := range [][]any{
		{owner, "2026-09-22", "mood", "moods", "calm", "1", nil},
		{owner, "2026-09-22", "sex", "intercourse", "", "unprotected", nil},
		{owner, "2026-09-22", "note", "text", "", nil, "private note"},
		{owner, "2026-09-21", "discharge", "consistency", "", "egg_white", nil},
		{owner, "2026-09-10", "mood", "moods", "sad", "1", nil}, // outside the 7-day window
		{otherID, "2026-09-22", "mood", "moods", "angry", "1", nil},
	} {
		_, err := db.Exec(entry, row...)
		require.NoError(t, err)
	}

	r := shared.NewReader(db)
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

	v, err := r.Read(ctx, owner, companion.SectionSymptoms, "en", now)
	require.NoError(t, err)
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"calm"`)
	for _, hidden := range []string{"intercourse", "unprotected", "private note", "egg_white", "sad", "angry", `"sex"`, `"note"`} {
		assert.NotContains(t, s, hidden)
	}
	assert.Contains(t, s, `"from":"2026-09-17"`)

	v, err = r.Read(ctx, owner, companion.SectionPregnancy, "en", now)
	require.NoError(t, err)
	raw, _ = json.Marshal(v)
	assert.JSONEq(t, `{"is_active":false}`, string(raw))

	v, err = r.Read(ctx, owner, companion.SectionCycle, "en", now)
	require.NoError(t, err)
	raw, _ = json.Marshal(v)
	assert.Contains(t, string(raw), `"has_data":false`)

	for _, sec := range []companion.Section{companion.SectionMeds, companion.SectionAppointments} {
		v, err = r.Read(ctx, owner, sec, "en", now)
		require.NoError(t, err)
		raw, _ = json.Marshal(v)
		assert.Equal(t, "[]", string(raw))
	}

	_, err = r.Read(ctx, owner, companion.Section("bank"), "en", now)
	require.ErrorIs(t, err, companion.ErrInvalidSection)
}
