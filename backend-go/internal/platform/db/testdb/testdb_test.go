package testdb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// 38 tables = the Laravel schema at the start of M2 minus its `migrations` table (domain-inventory §1).
// Migrations after the baseline add more.
const baselineTables = 38

func latestVersion(t *testing.T, p interface{ ListSources() []*goose.Source }) int64 {
	t.Helper()
	src := p.ListSources()
	require.NotEmpty(t, src)
	return src[len(src)-1].Version
}

func TestBaselineAppliedToFreshMariaDB(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	var version string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version))
	assert.True(t, strings.HasPrefix(version, "11.4."), "want MariaDB 11.4, got %s", version)

	var tables int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name <> ?`,
		platformdb.VersionTable).Scan(&tables))
	assert.GreaterOrEqual(t, tables, baselineTables)

	var laravel int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'migrations'`).Scan(&laravel))
	assert.Zero(t, laravel, "Laravel's migrations table is not part of the goose baseline")

	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	v, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, latestVersion(t, p), v)

	rows, err := db.QueryContext(ctx, "SELECT code, direction, is_default FROM languages ORDER BY sort_order")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var code, dir string
		var isDefault bool
		require.NoError(t, rows.Scan(&code, &dir, &isDefault))
		got = append(got, code+"/"+dir+"/"+map[bool]string{true: "default", false: "-"}[isDefault])
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"fa/rtl/default", "en/ltr/-"}, got)
}

func TestNewResetsTablesAndKeepsSeedRows(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	_, err := db.ExecContext(ctx, "INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000000', NOW(), NOW())")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO user_profiles (user_id, created_at, updated_at) VALUES (LAST_INSERT_ID(), NOW(), NOW())")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "DELETE FROM languages WHERE code = 'en'")
	require.NoError(t, err)

	db = testdb.New(t)
	var users, languages int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&languages))
	assert.Zero(t, users)
	assert.Equal(t, 2, languages)

	res, err := db.ExecContext(ctx, "INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000001', NOW(), NOW())")
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	assert.EqualValues(t, 1, id, "TRUNCATE resets AUTO_INCREMENT")

	var fk int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT @@foreign_key_checks").Scan(&fk))
	assert.Equal(t, 1, fk)
}

func TestStampBaselineMarksExistingSchemaAsMigrated(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)

	// Simulate stage/prod at cutover: schema built by Laravel, no goose bookkeeping yet.
	_, err := db.ExecContext(ctx, "DROP TABLE "+platformdb.VersionTable)
	require.NoError(t, err)

	require.NoError(t, platformdb.StampBaseline(ctx, db))
	require.NoError(t, platformdb.StampBaseline(ctx, db), "idempotent")

	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	v, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, platformdb.BaselineVersion, v)
	// Later migrations still run on a stamped database (they are written to be no-ops where the
	// Laravel twin already created the object); the baseline never does.
	results, err := p.Up(ctx)
	require.NoError(t, err)
	for _, r := range results {
		assert.Greater(t, r.Source.Version, platformdb.BaselineVersion, "the baseline must not run again on a stamped database")
	}
	pending, err := p.HasPending(ctx)
	require.NoError(t, err)
	assert.False(t, pending)
}

func TestBaselineDownIsRefused(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	// Roll back to the baseline first, and restore the full schema for the next tests.
	_, err = p.DownTo(ctx, platformdb.BaselineVersion)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	_, err = p.Down(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	var users int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users), "schema intact")
	v, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, platformdb.BaselineVersion, v)
}

func TestLoadSQL(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	dump := filepath.Join(t.TempDir(), "fixture.sql")
	require.NoError(t, os.WriteFile(dump, []byte(`/*!40101 SET NAMES utf8mb4 */;
-- a comment
INSERT INTO users (id, mobile, created_at, updated_at) VALUES (7, '09120000007', '2026-01-01 10:00:00', '2026-01-01 10:00:00');
INSERT INTO user_profiles (user_id, created_at, updated_at) VALUES (7, '2026-01-01 10:00:00', '2026-01-01 10:00:00');
`), 0o600))

	testdb.LoadSQL(t, dump)

	var n int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_profiles WHERE user_id = 7").Scan(&n))
	assert.Equal(t, 1, n)
}
