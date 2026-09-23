# Schema & migrations during M2 (goose + sqlc)

## Ownership rule

| Period | Owner of the schema | Who runs migrations on stage/prod |
|---|---|---|
| M2 (now → T-M2-27) | **Laravel** (`backend/database/migrations`) | Laravel entrypoint (`RUN_MIGRATIONS`) |
| After cutover (T-M2-27) | **goose** (`backend-go/db/migrations`) | Go entrypoint |

- The schema is **frozen** during M2. If a change is unavoidable, the same commit must contain
  **both** a Laravel migration **and** the matching goose migration (`db/migrations/000NN_<name>.sql`),
  and `make schema-diff` must stay green.
- The Go API never runs goose against stage/prod before T-M2-27. `db.Migrate` is called only by
  `internal/platform/db/testdb` and when bootstrapping a brand-new, empty environment.
- Never edit `00001_baseline.sql` to change the schema — add a new migration. The baseline is
  regenerated only if it drifts from what the Laravel migrations produce (see below).

## Files

| Path | What |
|---|---|
| `backend-go/db/migrations/00001_baseline.sql` | The whole schema (38 tables) + the fa/en `languages` rows (`INSERT IGNORE`). `Down` is refused on purpose. |
| `backend-go/db/embed.go`, `db/sqltypes.go` | Embed the migrations (`db.Migrations`); `db.NullRawJSON` (sqlc type for NULL-able JSON). |
| `backend-go/internal/platform/db/migrate.go` | `Migrate`, `NewMigrator` (goose provider), `StampBaseline`. |
| `backend-go/internal/platform/db/testdb` | Per-package throw-away database for integration tests. |
| `backend-go/sqlc.yaml` | One sqlc package per domain, declared up front. |
| `backend-go/db/queries/<domain>/*.sql` | sqlc queries → `internal/<domain>/store` (package `store`, generated). |
| `backend-go/scripts/schema-diff.sh` | `make schema-diff`. |

### How the baseline was made

`mariadb-dump --no-data --skip-comments --compact` of a MariaDB **11.4** database migrated by
`php artisan migrate` with `DB_CONNECTION=mysql` (the grammar stage and prod use), then:

- Laravel's `migrations` table dropped (goose tracks itself in `goose_db_version`);
- JSON columns written back as `json` (MariaDB expands `json` to
  `longtext COLLATE utf8mb4_bin CHECK (json_valid(col))`, identical to the dump, and sqlc needs `json`
  to apply the `json.RawMessage` override);
- `users` moved first for the foreign keys; `AUTO_INCREMENT=N` counters removed.

The task asked for a dump of **staging**. Reading the staging server needs the user's approval and was not
available when the baseline was written, so it came from a fresh Laravel-migrated MariaDB 11.4 instead.
Before cutover, check staging and prod against it (read-only, no data):

```bash
ssh root@89.251.8.115 'docker exec <stage-db-container> mariadb-dump -uroot -p… --no-data --skip-comments ritme' > /tmp/stage.sql
cd backend-go && scripts/schema-diff.sh --against /tmp/stage.sql
```

## `make schema-diff`

Builds two throw-away databases on the test MariaDB (`make test-db-up`), one with the Laravel
migrations (local `php` + `backend/vendor`, env vars only; `backend/.env` is not touched) and one with
`goose up`. Then it diffs:

- `mariadb-dump --no-data` of both, ignoring `migrations` / `goose_db_version` and `AUTO_INCREMENT` counters;
- seed rows: `languages` (minus timestamps) and every other table's row count.

Exit 0 = identical. `--against dump.sql` compares a schema-only dump (e.g. staging) with goose instead.
Run it in any commit that touches a migration on either side.

## sqlc

- Config: `backend-go/sqlc.yaml`. Packages: `auth, content, profile, healthlog, reminder, cycle,
  pregnancy, messages, home, admin, i18n` → `internal/<domain>/store`. A domain task only adds
  `.sql` files under `db/queries/<domain>/`, deletes the placeholder `sample.sql` there, and runs
  `make sqlc` (never edits `sqlc.yaml`). Commit the generated code.
- Type mapping (overrides):

  | Column | Go (NOT NULL) | Go (NULL-able) |
  |---|---|---|
  | `json` | `json.RawMessage` | `db.NullRawJSON` (= `sql.Null[json.RawMessage]`) |
  | `decimal` | `string` (Laravel `decimal:N` → `"62.50"`) | `sql.NullString` |
  | `date` | `civildate.Date` | `civildate.NullDate` |
  | `time` | `string` (`"HH:MM:SS"`) | `sql.NullString` |
  | `tinyint(1)` | `bool` | `sql.NullBool` |
  | `timestamp` / `datetime` | `time.Time` (Tehran wall-clock, `loc=Asia/Tehran`) | `sql.NullTime` |
  | `bigint unsigned` ids | `uint64` | – |

- `make sqlc` uses `sqlc` from `PATH`, then `$(go env GOPATH)/bin/sqlc`, else `go run …sqlc@v1.31.1`.

## Integration tests (`testdb`)

```go
func TestMain(m *testing.M) { testdb.Main(m) }   // optional: drops the package database at the end

func TestSomething(t *testing.T) {
    db := testdb.New(t)            // skips without TEST_DB_DSN; truncates all tables, restores fa/en
    q := store.New(db)
    …
}
```

- One fresh database **per test package** (`gt_<unix>_<pkg>_<rand>`), migrated once, so packages and
  parallel agents never share state. Within a package tests share it: no `t.Parallel()`.
- `New(t)` truncates every table (AUTO_INCREMENT back to 1) and re-inserts the rows the migrations
  seeded (`languages`).
- `testdb.LoadSQL(t, path)` loads a SQL dump (the contract fixtures of T-M2-05) into the package database.
- `testdb.DSN(t)` for code that opens its own pool.
- `make test-int` passes `TEST_DB_DSN` (the `ritme` user) and `TEST_DB_ADMIN_DSN` (root, for
  `CREATE DATABASE`). Leftover `gt_*` databases older than 2 h are swept; `make test-db-down` wipes all.

## Cutover (T-M2-27): version-stamping an existing database

Stage/prod already have every baseline table (built by Laravel), so the baseline must be **recorded, not
run**:

1. Stop Laravel migrations (`RUN_MIGRATIONS=false`) and confirm `scripts/schema-diff.sh --against <dump>`
   is green for that database.
2. Call `db.StampBaseline(ctx, pool)` once (idempotent): it creates `goose_db_version` and records
   versions 0 and 1 without executing anything. Equivalent SQL, if done by hand:
   `CREATE TABLE goose_db_version (id bigint(20) unsigned NOT NULL AUTO_INCREMENT, version_id bigint NOT NULL, is_applied boolean NOT NULL, tstamp timestamp NULL default now(), PRIMARY KEY(id));`
   `INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1), (1, 1);`
3. From then on the Go entrypoint may run `db.Migrate`, which applies only migrations after 00001.
4. Laravel's `migrations` table stays (harmless) until Laravel is removed.
