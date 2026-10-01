package migrations_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const menopauseVersion int64 = 22 // 00022_menopause.sql (00021 reserved by bloom)

// CB-MENO-01: the seeds land once, reference real taxonomy slots and checkup keys, and Down/Up round-trips the tables and the checkup_types.audiences column.
func TestMenopauseMigration_SeedsAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q, args...).Scan(&n))
		return n
	}
	groupCount := func(g string) int {
		return count("SELECT COUNT(*) FROM catalog_items WHERE `group` = ? AND needs_review = 1 AND JSON_CONTAINS(audiences, '\"menopause\"')", g)
	}
	want := map[string]int{"meno_score_items": 11, "meno_score_bands": 4, "meno_alerts": 7, "meno_tips": 14, "meno_checkup_groups": 4}
	for g, n := range want {
		assert.Equal(t, n, groupCount(g), g)
	}
	assert.Equal(t, 9, count("SELECT COUNT(*) FROM checkup_types WHERE `key` LIKE 'meno\\_%' AND is_active = 0 AND JSON_CONTAINS(audiences, '\"menopause\"')"))
	assert.Zero(t, count("SELECT COUNT(*) FROM checkup_types WHERE `key` NOT LIKE 'meno\\_%' AND audiences IS NOT NULL"),
		"existing M4 rows stay for everyone")

	// Score items: 16/16/12 domain maxima and every `log` ref is a menopause-loggable taxonomy slot.
	rows, err := db.QueryContext(ctx, "SELECT meta FROM catalog_items WHERE `group` = 'meno_score_items'")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	domainMax := map[string]int{}
	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))
		var meta struct {
			Domain string   `json:"domain"`
			Max    int      `json:"max"`
			Log    []string `json:"log"`
		}
		require.NoError(t, json.Unmarshal(raw, &meta))
		domainMax[meta.Domain] += meta.Max
		for _, ref := range meta.Log {
			parts := strings.Split(ref, ".")
			require.Len(t, parts, 3, ref)
			c, ok := taxonomy.CategoryByCode(parts[0])
			require.True(t, ok, ref)
			prm, ok := c.Param(parts[1])
			require.True(t, ok, ref)
			o, ok := prm.Option(parts[2])
			require.True(t, ok, ref)
			assert.True(t, c.Available(prm, taxonomy.ModeMenopause) && o.OptionAvailable(taxonomy.ModeMenopause), ref)
		}
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]int{"somatic": 16, "psychological": 16, "urogenital": 12}, domainMax)

	// Checkup groups only name checkup_types keys that exist.
	var groups []byte
	require.NoError(t, db.QueryRowContext(ctx, "SELECT JSON_ARRAYAGG(meta) FROM catalog_items WHERE `group` = 'meno_checkup_groups'").Scan(&groups))
	var metas []struct {
		Checkups []string `json:"checkups"`
	}
	require.NoError(t, json.Unmarshal(groups, &metas))
	for _, m := range metas {
		for _, key := range m.Checkups {
			assert.Equal(t, 1, count("SELECT COUNT(*) FROM checkup_types WHERE `key` = ? AND user_id IS NULL", key), key)
		}
	}

	_, err = p.DownTo(ctx, menopauseVersion-2)
	require.NoError(t, err)
	assert.Zero(t, count("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('hot_flashes','menopause_scores','treatment_items','treatment_intakes','side_effect_logs')"))
	assert.Zero(t, count("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'checkup_types' AND column_name = 'audiences'"))
	assert.Zero(t, count("SELECT COUNT(*) FROM catalog_items WHERE `group` LIKE 'meno\\_%'"))

	_, err = p.Up(ctx)
	require.NoError(t, err)
	for g, n := range want {
		assert.Equal(t, n, groupCount(g), g)
	}
}
