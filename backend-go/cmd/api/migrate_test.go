package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// Integration tests (make test-int PKG=./cmd/api/...): the three database states stage can be in
// when backend-go starts with RUN_MIGRATIONS=true (T-M2-28). Each test builds its state from the
// package's testdb database and leaves the full baseline schema behind for the next one.

func TestMain(m *testing.M) { testdb.Main(m) }

func tables(t *testing.T, pool *sql.DB) []string {
	t.Helper()
	rows, err := pool.QueryContext(context.Background(),
		`SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_name`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		out = append(out, name)
	}
	require.NoError(t, rows.Err())
	return out
}

func dropAll(t *testing.T, pool *sql.DB) {
	t.Helper()
	ctx := context.Background()
	conn, err := pool.Conn(ctx)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	_, err = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0")
	require.NoError(t, err)
	for _, name := range tables(t, pool) {
		_, err := conn.ExecContext(ctx, "DROP TABLE `"+name+"`")
		require.NoError(t, err)
	}
	_, err = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1")
	require.NoError(t, err)
}

func dbVersion(t *testing.T, pool *sql.DB) int64 {
	t.Helper()
	p, err := db.NewMigrator(pool)
	require.NoError(t, err)
	v, err := p.GetDBVersion(context.Background())
	require.NoError(t, err)
	return v
}

// latestVersion is the newest embedded goose migration (the baseline plus every later one).
func latestVersion(t *testing.T, pool *sql.DB) int64 {
	t.Helper()
	p, err := db.NewMigrator(pool)
	require.NoError(t, err)
	src := p.ListSources()
	require.NotEmpty(t, src)
	return src[len(src)-1].Version
}

func quietLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }

// Fresh, empty database (a new stage volume): the baseline is applied.
func TestMigrateOnStart_EmptyDatabaseAppliesBaseline(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	want := tables(t, pool) // the fully migrated schema, goose_db_version included
	dropAll(t, pool)
	require.Empty(t, tables(t, pool))

	res, err := db.MigrateOnStart(ctx, pool)
	require.NoError(t, err)
	assert.Equal(t, db.StartBaselineApplied, res.Action)
	assert.Contains(t, res.Applied, db.BaselineVersion)
	assert.Equal(t, latestVersion(t, pool), res.Version)
	assert.Equal(t, want, tables(t, pool), "baseline + later migrations + goose_db_version")

	var languages int
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&languages))
	assert.Equal(t, 2, languages, "the baseline seeds fa/en")
}

// Stage today: schema and rows built by Laravel (its `migrations` table), no goose table. The
// baseline must be STAMPED, never run — running it would fail on (or re-create) existing tables.
func TestMigrateOnStart_LaravelSchemaIsStampedNotRecreated(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	_, err := pool.ExecContext(ctx, "DROP TABLE "+db.VersionTable)
	require.NoError(t, err)
	_, err = pool.ExecContext(ctx, `CREATE TABLE migrations (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, migration VARCHAR(255) NOT NULL, batch INT NOT NULL)`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.ExecContext(context.Background(), "DROP TABLE IF EXISTS migrations") })
	// Live data that must survive.
	_, err = pool.ExecContext(ctx, "INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000099', NOW(), NOW())")
	require.NoError(t, err)

	res, err := db.MigrateOnStart(ctx, pool)
	require.NoError(t, err)
	assert.Equal(t, db.StartStamped, res.Action)
	assert.NotContains(t, res.Applied, db.BaselineVersion, "baseline stamped, not applied")
	for _, v := range res.Applied {
		assert.Greater(t, v, db.BaselineVersion, "only migrations added after the baseline run")
	}
	assert.Equal(t, latestVersion(t, pool), res.Version)

	var users int
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users))
	assert.Equal(t, 1, users, "existing rows untouched")

	// A second start is the ordinary goose-managed path.
	res, err = db.MigrateOnStart(ctx, pool)
	require.NoError(t, err)
	assert.Equal(t, db.StartUpToDate, res.Action)
	assert.Empty(t, res.Applied)
}

// Already under goose: only pending migrations run (none — testdb is fully migrated).
func TestMigrateOnStart_GooseManagedAppliesPendingOnly(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	require.Equal(t, latestVersion(t, pool), dbVersion(t, pool))
	want := tables(t, pool)

	res, err := db.MigrateOnStart(ctx, pool)
	require.NoError(t, err)
	assert.Equal(t, db.StartUpToDate, res.Action)
	assert.Empty(t, res.Applied)
	assert.Equal(t, latestVersion(t, pool), res.Version)
	assert.Equal(t, want, tables(t, pool))
}

// Tables but no bookkeeping at all: not goose's schema — refuse rather than guess.
func TestMigrateOnStart_UnknownSchemaIsRefused(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	_, err := pool.ExecContext(ctx, "DROP TABLE "+db.VersionTable)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.StampBaseline(context.Background(), pool)) })

	_, err = db.MigrateOnStart(ctx, pool)
	require.ErrorContains(t, err, "unknown schema")
}

// The cmd/api wrapper logs and propagates.
func TestMigrateOnStartWrapper(t *testing.T) {
	pool := testdb.New(t)
	require.NoError(t, migrateOnStart(context.Background(), quietLogger(), pool))
}
