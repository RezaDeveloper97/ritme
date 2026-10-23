package migrations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const childMilestonesCDCVersion int64 = 102 // 00102_child_milestones_cdc2022.sql

// childBandCount is how many active items of group sit in the age band months.
func childBandCount(t *testing.T, conn *sql.DB, group string, months int) int {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE `group` = ? AND is_active = 1 "+
		"AND JSON_VALUE(meta, '$.age_months') = ?", group, months).Scan(&n))
	return n
}

func childMeta(t *testing.T, conn *sql.DB, group, code string) map[string]any {
	t.Helper()
	var raw string
	require.NoError(t, conn.QueryRow("SELECT meta FROM catalog_items WHERE `group` = ? AND code = ?", group, code).Scan(&raw))
	m := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	return m
}

// T-M10-22 (D-173, H-15): the CDC 2022 15- and 30-month bands with play ideas and doctor notes in every seeded
// language, needs_review = 1; the 3-month band marked as Ritme's own. Down removes the bands (and checks on them),
// unmarks the 3-month band; Up runs again.
func TestChildMilestonesCDC2022_UpDown(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	codes := languageCodes(t, conn)
	assertUp := func() {
		for _, m := range []int{15, 30} {
			assert.Equal(t, 5, childBandCount(t, conn, "child_milestones", m), "milestones %d", m)
			assert.Equal(t, 1, childBandCount(t, conn, "child_milestone_activities", m), "activities %d", m)
			assert.Equal(t, 1, childBandCount(t, conn, "child_milestone_notes", m), "note %d", m)
		}
		rows, err := conn.Query("SELECT title, IFNULL(body, '{}'), `group`, needs_review, meta FROM catalog_items " +
			"WHERE `group` LIKE 'child_milestone%' AND JSON_VALUE(meta, '$.age_months') IN (15, 30)")
		require.NoError(t, err)
		defer rows.Close()
		n := 0
		for rows.Next() {
			var title, body, group, meta string
			var review bool
			require.NoError(t, rows.Scan(&title, &body, &group, &review, &meta))
			n++
			assert.True(t, review, "needs_review")
			assert.Contains(t, meta, `"band_source":"cdc_2022"`)
			for _, code := range codes {
				assert.NotEmpty(t, translations(t, title)[code], "title %s", code)
				if group != "child_milestones" {
					assert.NotEmpty(t, translations(t, body)[code], "body %s", code)
				}
			}
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, 14, n)
		assert.Equal(t, "ritme", childMeta(t, conn, "child_milestones", "m3_social_smile")["band_source"])
		assert.Equal(t, "ritme", childMeta(t, conn, "child_milestone_activities", "m3_tummy_time")["band_source"])
		assert.Equal(t, "ritme", childMeta(t, conn, "child_milestone_notes", "m3")["band_source"])
		assert.NotContains(t, childMeta(t, conn, "child_milestones", "m36_converses"), "band_source", "m36 is CDC")
		assert.Equal(t, 7, childBandCount(t, conn, "child_milestones", 3), "the 3-month band stays")
	}
	assertUp()

	// A check on a new milestone goes with it on Down.
	uid := newV6User(t, conn, "09129990222")
	res, err := conn.Exec("INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'آوا', '2025-07-01', NOW(), NOW())", uid)
	require.NoError(t, err)
	cid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = conn.Exec("INSERT INTO child_milestone_checks (child_id, milestone_code, checked_on, created_at, updated_at) "+
		"VALUES (?, 'm15_takes_steps', '2026-10-01', NOW(), NOW()), (?, 'm12_cruises', '2026-07-01', NOW(), NOW())", cid, cid)
	require.NoError(t, err)

	_, err = p.DownTo(ctx, childMilestonesCDCVersion-1)
	require.NoError(t, err)
	for _, m := range []int{15, 30} {
		assert.Zero(t, childBandCount(t, conn, "child_milestones", m))
		assert.Zero(t, childBandCount(t, conn, "child_milestone_activities", m))
		assert.Zero(t, childBandCount(t, conn, "child_milestone_notes", m))
	}
	assert.NotContains(t, childMeta(t, conn, "child_milestones", "m3_social_smile"), "band_source")
	assert.NotContains(t, childMeta(t, conn, "child_milestone_notes", "m3"), "band_source")
	var checks []string
	cr, err := conn.Query("SELECT milestone_code FROM child_milestone_checks WHERE child_id = ? ORDER BY milestone_code", cid)
	require.NoError(t, err)
	defer cr.Close()
	for cr.Next() {
		var c string
		require.NoError(t, cr.Scan(&c))
		checks = append(checks, c)
	}
	require.NoError(t, cr.Err())
	assert.Equal(t, []string{"m12_cruises"}, checks, "only the new milestone's check is removed")

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assertUp()
}
