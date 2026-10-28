package accountexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newUser(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id)
}

// seedPostpartum gives uid a birth, two recovery logs and an EPDS check with item 10 answered.
func seedPostpartum(t *testing.T, db *sql.DB, uid uint64, temp string) {
	t.Helper()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO postpartum_profiles (user_id, birth_date, delivery_type, baby_count, source, created_at, updated_at) VALUES (?, '2026-09-10', 'cesarean', 1, 'direct', NOW(), NOW())`, []any{uid}},
		{`INSERT INTO postpartum_recovery_logs (user_id, log_date, temperature_c, pad_soaked_within_hour, severe_pain, breathless, postpartum_visit, created_at, updated_at) VALUES (?, '2026-09-15', ?, 1, 0, NULL, NULL, NOW(), NOW())`, []any{uid, temp}},
		{`INSERT INTO postpartum_recovery_logs (user_id, log_date, temperature_c, postpartum_visit, created_at, updated_at) VALUES (?, '2026-09-20', NULL, 1, NOW(), NOW())`, []any{uid}},
		{`INSERT INTO epds_checks (user_id, kind, taken_on, answers, total, self_harm, urgent, created_at, updated_at) VALUES (?, 'full', '2026-09-24', '{"10":2}', 17, 2, 1, NOW(), NOW())`, []any{uid}},
	} {
		_, err := db.Exec(q.sql, q.args...)
		require.NoError(t, err)
	}
}

func postpartumSection(t *testing.T, db *sql.DB, uid uint64) (map[string]any, string) {
	t.Helper()
	var load loader
	for _, s := range Sections(db, nil) {
		if s.Key == "postpartum" {
			load = s.Load
		}
	}
	require.NotNil(t, load, "postpartum section is registered")
	v, err := load(context.Background(), uid)
	require.NoError(t, err)
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out, string(raw)
}

// T-M9-06c: the export carries her postpartum profile and recovery logs, never an EPDS row (D-54), and never another
// user's postpartum rows.
func TestPostpartumExport_OwnerRowsOnlyNoEPDS(t *testing.T) {
	db := testdb.New(t)
	sara := newUser(t, db, "09120009401")
	nina := newUser(t, db, "09120009402")
	seedPostpartum(t, db, sara, "38.4")
	seedPostpartum(t, db, nina, "39.7")
	// Nina has an extra day Sara does not have.
	_, err := db.Exec(`INSERT INTO postpartum_recovery_logs (user_id, log_date, severe_pain, created_at, updated_at) VALUES (?, '2026-09-30', 1, NOW(), NOW())`, nina)
	require.NoError(t, err)

	out, raw := postpartumSection(t, db, sara)
	assert.ElementsMatch(t, []string{"profile", "recovery_logs"}, keys(out))

	prof := out["profile"].(map[string]any)
	assert.Equal(t, float64(sara), prof["user_id"])
	assert.Equal(t, "cesarean", prof["delivery_type"])
	assert.Contains(t, prof["birth_date"], "2026-09-10")

	logs := out["recovery_logs"].([]any)
	require.Len(t, logs, 2)
	first, second := logs[0].(map[string]any), logs[1].(map[string]any)
	assert.Equal(t, float64(sara), first["user_id"])
	assert.Equal(t, "38.4", first["temperature_c"])
	assert.Equal(t, true, first["pad_soaked_within_hour"])
	assert.Equal(t, false, first["severe_pain"])
	assert.Nil(t, first["breathless"], "not answered stays null")
	assert.Equal(t, float64(sara), second["user_id"])
	assert.Equal(t, true, second["postpartum_visit"])
	assert.Nil(t, second["temperature_c"])

	// No EPDS anywhere (no answers, totals or flags).
	for _, k := range []string{"epds", "epds_checks", "answers", "total", "self_harm", "urgent"} {
		assert.NotContains(t, raw, `"`+k+`"`)
	}
	// Nothing of Nina's: her temperature, her extra day, her user id.
	assert.NotContains(t, raw, "39.7")
	assert.NotContains(t, raw, "2026-09-30")
	for _, l := range logs {
		assert.NotEqual(t, float64(nina), l.(map[string]any)["user_id"])
	}

	// A user without a birth: null profile, empty list.
	none := newUser(t, db, "09120009403")
	out, _ = postpartumSection(t, db, none)
	assert.Nil(t, out["profile"])
	assert.Equal(t, []any{}, out["recovery_logs"])
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
