// Package testdb gives integration tests a real MariaDB with the goose baseline applied.
//
// Isolation model (many packages, and many agents, run integration tests at the same time):
//   - every test *package* (= test binary = process) gets its own freshly created database,
//     named gt_<unix-time>_<package>_<random>, created on first use and migrated once;
//   - New(t) truncates every table (except goose's) and restores the rows the migrations seed
//     (the fa/en `languages` rows), so each test starts from the same state;
//   - tests in one package share that database, so they must not use t.Parallel().
//
// Environment:
//
//	TEST_DB_DSN        DSN of the test server (make test-int sets it). Unset → tests t.Skip().
//	TEST_DB_ADMIN_DSN  optional DSN of a user allowed to CREATE/DROP DATABASE (make test-int
//	                   passes root); defaults to TEST_DB_DSN.
//
// Databases are dropped by Main (use it from TestMain) and, as a safety net, any gt_* database
// older than staleAfter is dropped when a new one is created. `make test-db-down` wipes them all.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
)

const (
	dbPrefix     = "gt_"
	staleAfter   = 2 * time.Hour
	setupTimeout = 2 * time.Minute
)

var (
	once     sync.Once
	shared   *state
	setupErr error
)

type state struct {
	name     string
	dsn      string // DSN of the per-package database (same user/params as TEST_DB_DSN)
	adminDSN string // DSN without a database, for CREATE/DROP DATABASE
	pool     *sql.DB
	tables   []string
	seed     map[string]seedRows
}

type seedRows struct {
	columns []string
	rows    [][]any
}

// Enabled reports whether integration tests can run (TEST_DB_DSN is set).
func Enabled() bool { return os.Getenv("TEST_DB_DSN") != "" }

// New returns the package's database with every table reset to the post-migration state.
// It skips the test when TEST_DB_DSN is unset. The pool is shared: don't Close it.
func New(t testing.TB) *sql.DB {
	t.Helper()
	s := get(t)
	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()
	if err := s.reset(ctx); err != nil {
		t.Fatalf("testdb: reset %s: %v", s.name, err)
	}
	return s.pool
}

// DSN returns the DSN of the package's database (e.g. for code that opens its own pool).
// It skips the test when TEST_DB_DSN is unset. Call New first if the test needs a clean state.
func DSN(t testing.TB) string {
	t.Helper()
	return get(t).dsn
}

// LoadSQL executes a SQL file (e.g. a mariadb-dump of the contract fixtures, T-M2-05) against the
// package's database. The file may contain many statements; comments and
// `/*!… */` version hints are passed through to the server, which understands them.
func LoadSQL(t testing.TB, path string) {
	t.Helper()
	s := get(t)
	body, err := os.ReadFile(path) //nolint:gosec // test fixture path chosen by the test
	if err != nil {
		t.Fatalf("testdb: read %s: %v", path, err)
	}
	cfg, err := mysql.ParseDSN(s.dsn)
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	cfg.MultiStatements = true
	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("testdb: open: %v", err)
	}
	defer func() { _ = pool.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()
	if _, err := pool.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("testdb: load %s: %v", path, err)
	}
}

// Main runs the package's tests and drops the package database afterwards:
//
//	func TestMain(m *testing.M) { testdb.Main(m) }
//
// Optional: without it the database is left behind until the stale sweep or `make test-db-down`.
func Main(m *testing.M) {
	code := m.Run()
	if shared != nil {
		ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
		_ = shared.drop(ctx)
		cancel()
	}
	os.Exit(code)
}

func get(t testing.TB) *state {
	t.Helper()
	if !Enabled() {
		t.Skip("testdb: TEST_DB_DSN is not set (run `make test-int`)")
	}
	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
		defer cancel()
		shared, setupErr = setup(ctx)
	})
	if setupErr != nil {
		t.Fatalf("testdb: %v", setupErr)
	}
	return shared
}

func setup(ctx context.Context) (*state, error) {
	base, err := mysql.ParseDSN(os.Getenv("TEST_DB_DSN"))
	if err != nil {
		return nil, fmt.Errorf("parse TEST_DB_DSN: %w", err)
	}
	admin := base.Clone()
	if v := os.Getenv("TEST_DB_ADMIN_DSN"); v != "" {
		if admin, err = mysql.ParseDSN(v); err != nil {
			return nil, fmt.Errorf("parse TEST_DB_ADMIN_DSN: %w", err)
		}
	}
	admin.DBName = ""

	name, err := newName(time.Now())
	if err != nil {
		return nil, err
	}
	s := &state{name: name, adminDSN: admin.FormatDSN()}

	if err := s.adminExec(ctx, func(ctx context.Context, db *sql.DB) error {
		sweepStale(ctx, db, time.Now())
		if _, err := db.ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
			return fmt.Errorf("create database %s (TEST_DB_ADMIN_DSN needs CREATE privilege): %w", name, err)
		}
		if admin.User != base.User {
			// The test user only owns its own database in docker-compose.test.yml.
			grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", name, strings.ReplaceAll(base.User, "'", "''"))
			if _, err := db.ExecContext(ctx, grant); err != nil { //nolint:gosec // G701: user comes from the test DSN (env), quotes escaped
				return fmt.Errorf("grant on %s to %s: %w", name, base.User, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	cfg := base.Clone()
	cfg.DBName = name
	s.dsn = cfg.FormatDSN()
	if s.pool, err = platformdb.OpenDSN(ctx, s.dsn); err != nil {
		_ = s.drop(ctx)
		return nil, err
	}
	if err := platformdb.Migrate(ctx, s.pool); err != nil {
		_ = s.drop(ctx)
		return nil, err
	}
	if err := s.snapshot(ctx); err != nil {
		_ = s.drop(ctx)
		return nil, err
	}
	return s, nil
}

// snapshot records the table list and the rows the migrations seeded, so reset can restore them.
func (s *state) snapshot(ctx context.Context) error {
	tables, err := queryStrings(ctx, s.pool,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' AND table_name <> ?
		 ORDER BY table_name`, platformdb.VersionTable)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	s.tables = tables

	s.seed = map[string]seedRows{}
	for _, table := range s.tables {
		seed, err := readAll(ctx, s.pool, table)
		if err != nil {
			return err
		}
		if len(seed.rows) > 0 {
			s.seed[table] = seed
		}
	}
	return nil
}

// queryStrings runs a one-column query and returns its values.
func queryStrings(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Table and database names below come from information_schema or newName, never from input,
// so building the identifiers into SQL is safe (gosec G202 suppressed where it fires).

func readAll(ctx context.Context, db *sql.DB, table string) (seedRows, error) {
	rows, err := db.QueryContext(ctx, "SELECT * FROM `"+table+"`") //nolint:gosec // G202: see above
	if err != nil {
		return seedRows{}, fmt.Errorf("read %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return seedRows{}, err
	}
	out := seedRows{columns: cols}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return seedRows{}, err
		}
		out.rows = append(out.rows, vals)
	}
	return out, rows.Err()
}

// reset truncates every table (resetting AUTO_INCREMENT) and re-inserts the migration seed rows.
func (s *state) reset(ctx context.Context) error {
	conn, err := s.pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Session-scoped and on one pinned connection; restored before it returns to the pool.
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(context.WithoutCancel(ctx), "SET FOREIGN_KEY_CHECKS = 1") }()

	for _, table := range s.tables {
		if _, err := conn.ExecContext(ctx, "TRUNCATE TABLE `"+table+"`"); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	for table, seed := range s.seed {
		cols := "`" + strings.Join(seed.columns, "`, `") + "`"
		marks := strings.TrimSuffix(strings.Repeat("?, ", len(seed.columns)), ", ")
		q := "INSERT INTO `" + table + "` (" + cols + ") VALUES (" + marks + ")" //nolint:gosec // G202: see above
		for _, row := range seed.rows {
			if _, err := conn.ExecContext(ctx, q, row...); err != nil {
				return fmt.Errorf("restore %s: %w", table, err)
			}
		}
	}
	return nil
}

func (s *state) drop(ctx context.Context) error {
	if s.pool != nil {
		_ = s.pool.Close()
	}
	return s.adminExec(ctx, func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+s.name+"`")
		return err
	})
}

func (s *state) adminExec(ctx context.Context, fn func(context.Context, *sql.DB) error) error {
	db, err := sql.Open("mysql", s.adminDSN)
	if err != nil {
		return fmt.Errorf("open admin connection: %w", err)
	}
	defer func() { _ = db.Close() }()
	return fn(ctx, db)
}

var unsafeChars = regexp.MustCompile(`[^a-z0-9]+`)

// newName builds gt_<unix>_<package>_<random>; the unix time lets sweepStale spot leftovers.
func newName(now time.Time) (string, error) {
	pkg := strings.TrimSuffix(filepath.Base(os.Args[0]), ".test")
	pkg = strings.Trim(unsafeChars.ReplaceAllString(strings.ToLower(pkg), "_"), "_")
	if len(pkg) > 24 {
		pkg = pkg[:24]
	}
	var r [4]byte
	if _, err := rand.Read(r[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d_%s_%s", dbPrefix, now.Unix(), pkg, hex.EncodeToString(r[:])), nil
}

// sweepStale drops gt_* databases older than staleAfter (left by crashed or TestMain-less runs).
// Best effort: errors are ignored, a failed sweep must not fail the test run.
func sweepStale(ctx context.Context, db *sql.DB, now time.Time) {
	names, err := queryStrings(ctx, db, "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'gt\\_%'")
	if err != nil {
		return
	}
	for _, name := range names {
		parts := strings.SplitN(strings.TrimPrefix(name, dbPrefix), "_", 2)
		ts, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || now.Sub(time.Unix(ts, 0)) <= staleAfter {
			continue
		}
		_, _ = db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+name+"`")
	}
}
