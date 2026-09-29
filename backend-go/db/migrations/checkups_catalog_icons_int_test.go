package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const checkupIconsVersion int64 = 7 // 00007_checkups_catalog_icons.sql

func catalogLook(t *testing.T, db *sql.DB) map[string][2]string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		"SELECT `key`, COALESCE(`icon`, ''), `tone` FROM checkup_types WHERE user_id IS NULL")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	out := map[string][2]string{}
	for rows.Next() {
		var key, icon, tone string
		require.NoError(t, rows.Scan(&key, &icon, &tone))
		out[key] = [2]string{icon, tone}
	}
	require.NoError(t, rows.Err())
	return out
}

func TestCheckupsCatalogIconsMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, checkupIconsVersion-1)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	seeded := catalogLook(t, db)
	require.Equal(t, [2]string{"breast", "violet"}, seeded["clinical_breast_exam"], "00003 seed")

	// An admin already re-styled the blood test: it must be kept.
	_, err = db.ExecContext(ctx, "UPDATE checkup_types SET icon = 'drop' WHERE `key` = 'blood_test'")
	require.NoError(t, err)

	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string][2]string{
		"breast_self_exam":     {"ribbon", "rose"},
		"clinical_breast_exam": {"stetho", "amber"},
		"pap_smear":            {"shield", "violet"},
		"blood_test":           {"drop", "amber"},
		"dentist":              {"tooth", "green"},
		"mammography":          {"ribbon", "neutral"},
	}, catalogLook(t, db))

	_, err = p.DownTo(ctx, checkupIconsVersion-1)
	require.NoError(t, err)
	down := catalogLook(t, db)
	seeded["blood_test"] = [2]string{"drop", "amber"}
	assert.Equal(t, seeded, down, "Down restores the seed values")
}
