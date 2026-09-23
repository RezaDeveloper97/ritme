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
// Used ONLY by tests (testdb) and brand-new environments. The stage/prod entrypoint must not
// call it before the cutover (T-M2-27): Laravel owns the schema until then, and an existing
// database is version-stamped with StampBaseline instead of migrated.
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
