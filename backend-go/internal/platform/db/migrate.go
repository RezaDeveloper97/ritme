package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"

	dbfiles "github.com/ritme/backend-go/db"
)

// VersionTable is goose's bookkeeping table (Laravel's is `migrations`; both coexist until T-M2-27).
const VersionTable = "goose_db_version"

// BaselineVersion is the version of db/migrations/00001_baseline.sql: the whole schema as the
// Laravel migrations left it at the start of M2.
const BaselineVersion int64 = 1

// MigrationsFS returns the embedded goose migrations (db/migrations/*.sql).
func MigrationsFS() fs.FS {
	sub, err := fs.Sub(dbfiles.Migrations, "migrations")
	if err != nil { // only possible if the embed directive is broken
		panic(fmt.Sprintf("db: migrations fs: %v", err))
	}
	return sub
}

// NewMigrator returns a goose provider over the embedded migrations.
func NewMigrator(pool *sql.DB) (*goose.Provider, error) {
	// goose's default version table is VersionTable; the global Go-migration registry is unused.
	p, err := goose.NewProvider(goose.DialectMySQL, pool, MigrationsFS(),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("db: goose provider: %w", err)
	}
	return p, nil
}

// Migrate applies every pending goose migration.
//
// Used by tests (testdb) and, through MigrateOnStart, by environments where goose owns the schema
// (stage since T-M2-28). Production must not run it before the cutover (T-M2-27): Laravel owns that
// schema until then, and an existing database is version-stamped with StampBaseline, not migrated.
func Migrate(ctx context.Context, pool *sql.DB) error {
	p, err := NewMigrator(pool)
	if err != nil {
		return err
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// StampBaseline records the baseline as applied WITHOUT running it. This is the cutover step for
// a database whose schema was built by the Laravel migrations (stage, prod): afterwards goose
// sees version 1 and only runs migrations added after the baseline. Idempotent.
//
// Precondition (checked by `make schema-diff` and the cutover runbook): the database schema is
// identical to 00001_baseline.sql.
func StampBaseline(ctx context.Context, pool *sql.DB) error {
	store := mustStore()
	conn, err := pool.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: stamp: %w", err)
	}
	defer func() { _ = conn.Close() }()

	exists, err := store.TableExists(ctx, conn)
	if err != nil {
		return fmt.Errorf("db: stamp: version table: %w", err)
	}
	if !exists {
		if err := store.CreateVersionTable(ctx, conn); err != nil {
			return fmt.Errorf("db: stamp: create version table: %w", err)
		}
	}
	// goose writes version 0 when it creates its table itself; mirror that.
	for _, v := range []int64{0, BaselineVersion} {
		if _, err := store.GetMigration(ctx, conn, v); err == nil {
			continue
		} else if !errors.Is(err, database.ErrVersionNotFound) {
			return fmt.Errorf("db: stamp: read version %d: %w", v, err)
		}
		if err := store.Insert(ctx, conn, database.InsertRequest{Version: v}); err != nil {
			return fmt.Errorf("db: stamp: insert version %d: %w", v, err)
		}
	}
	return nil
}

// LaravelVersionTable is Laravel's migration bookkeeping table.
const LaravelVersionTable = "migrations"

// StartAction says what MigrateOnStart found and did.
type StartAction string

const (
	// StartBaselineApplied: the database was empty; every goose migration (baseline included) ran.
	StartBaselineApplied StartAction = "baseline_applied"
	// StartStamped: the schema was built by Laravel (its `migrations` table, no goose table); the
	// baseline was recorded as applied WITHOUT running it, then later migrations (if any) ran.
	StartStamped StartAction = "stamped_laravel_schema"
	// StartUpToDate: goose already owned the database; only pending migrations ran (maybe none).
	StartUpToDate StartAction = "goose_managed"
)

// StartResult reports MigrateOnStart's decision, for the startup log.
type StartResult struct {
	Action  StartAction
	Applied []int64 // versions applied by this call (empty when nothing was pending)
	Version int64   // goose version after the call
}

// MigrateOnStart brings a database under goose at process start (RUN_MIGRATIONS=true). It never
// re-creates tables that already exist:
//
//   - goose_db_version present            → apply pending migrations only;
//   - Laravel `migrations` table, no goose → StampBaseline (the baseline equals the Laravel-built
//     schema — scripts/schema-diff.sh), then apply migrations added after the baseline;
//   - no tables at all                    → apply everything, baseline included;
//   - tables but neither bookkeeping table → refuse: an unknown schema is not goose's to touch.
//
// One process at a time: goose's MySQL dialect has no session lock, so run a single migrating
// replica (stage has exactly one backend-go).
func MigrateOnStart(ctx context.Context, pool *sql.DB) (StartResult, error) {
	gooseTable, err := tableExists(ctx, pool, VersionTable)
	if err != nil {
		return StartResult{}, err
	}
	res := StartResult{Action: StartUpToDate}
	if !gooseTable {
		laravelTable, err := tableExists(ctx, pool, LaravelVersionTable)
		if err != nil {
			return StartResult{}, err
		}
		switch {
		case laravelTable:
			if err := StampBaseline(ctx, pool); err != nil {
				return StartResult{}, err
			}
			res.Action = StartStamped
		default:
			n, err := tableCount(ctx, pool)
			if err != nil {
				return StartResult{}, err
			}
			if n > 0 {
				return StartResult{}, fmt.Errorf("db: migrate on start: %d table(s) but neither %s nor %s — "+
					"refusing to apply the baseline over an unknown schema", n, VersionTable, LaravelVersionTable)
			}
			res.Action = StartBaselineApplied
		}
	}

	p, err := NewMigrator(pool)
	if err != nil {
		return StartResult{}, err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return StartResult{}, fmt.Errorf("db: migrate on start (%s): %w", res.Action, err)
	}
	for _, r := range results {
		res.Applied = append(res.Applied, r.Source.Version)
	}
	if res.Version, err = p.GetDBVersion(ctx); err != nil {
		return StartResult{}, fmt.Errorf("db: migrate on start: version: %w", err)
	}
	return res, nil
}

func tableExists(ctx context.Context, pool *sql.DB, name string) (bool, error) {
	var n int
	err := pool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`,
		name).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("db: table %s exists: %w", name, err)
	}
	return n > 0, nil
}

func tableCount(ctx context.Context, pool *sql.DB) (int, error) {
	var n int
	err := pool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("db: count tables: %w", err)
	}
	return n, nil
}

func mustStore() database.StoreExtender {
	s, err := database.NewStore(database.DialectMySQL, VersionTable)
	if err != nil { // only possible with an unknown dialect or empty table name
		panic(fmt.Sprintf("db: goose store: %v", err))
	}
	ext, ok := s.(database.StoreExtender)
	if !ok {
		panic("db: goose mysql store cannot check table existence")
	}
	return ext
}
