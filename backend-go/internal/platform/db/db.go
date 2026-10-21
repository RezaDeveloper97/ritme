// Package db opens the MariaDB connection pool shared with the Laravel backend.
//
// Timestamps in the Ritme schema are stored as Asia/Tehran wall-clock values (Laravel
// writes them with APP_TIMEZONE=Asia/Tehran and the session time_zone untouched). The
// DSN therefore uses parseTime=true&loc=Asia/Tehran so DATETIME/TIMESTAMP columns scan
// into time.Time in Tehran, and this package must NEVER set the session time_zone:
// doing so would make MariaDB convert TIMESTAMP columns and shift every value.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/platform/config"
)

// pingTimeout bounds the start-up ping. Pool sizing comes from config.DBPool (DB_MAX_OPEN, …).
const pingTimeout = 5 * time.Second

// Collation is the connection collation (utf8mb4, the schema's charset). It also makes client-side interpolation
// safe: go-sql-driver refuses interpolateParams with a multi-byte collation that could swallow the escaping
// backslash (big5, gbk, sjis, …), and utf8mb4 is not one of them.
const Collation = "utf8mb4_unicode_ci"

// DSN builds the go-sql-driver DSN for cfg. loc is the location DATETIME values are
// interpreted in; callers pass the APP_TIMEZONE location (Asia/Tehran).
//
// interpolateParams=true (T-M9-09): the driver inlines the escaped arguments into the SQL text, so a
// parameterised query is one round-trip (COM_QUERY) instead of prepare + execute + close. Time arguments are
// still formatted in loc, exactly as the binary protocol did.
func DSN(cfg config.DB, loc *time.Location) string {
	c := mysql.NewConfig()
	c.User = cfg.Username
	c.Passwd = cfg.Password
	c.Net = "tcp"
	c.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	c.DBName = cfg.Database
	c.ParseTime = true
	c.Loc = loc
	c.Collation = Collation
	c.Params = map[string]string{"charset": "utf8mb4"}
	c.InterpolateParams = true
	return c.FormatDSN()
}

// Open opens and pings the pool, sized by cfg.Pool.
func Open(ctx context.Context, cfg config.DB, loc *time.Location) (*sql.DB, error) {
	return OpenDSNWithPool(ctx, DSN(cfg, loc), cfg.Pool)
}

// OpenDSN opens and pings an instrumented pool from a ready DSN with the default pool sizing (integration tests
// use TEST_DB_DSN). The DSN must not contain a time_zone parameter.
func OpenDSN(ctx context.Context, dsn string) (*sql.DB, error) {
	return OpenDSNWithPool(ctx, dsn, config.DBPool{})
}

// OpenDSNWithPool is OpenDSN with explicit pool sizing (zero fields → defaults). interpolateParams is always on,
// whatever the DSN says, so tests run the same wire protocol as production.
func OpenDSNWithPool(ctx context.Context, dsn string, poolCfg config.DBPool) (*sql.DB, error) {
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	if _, ok := parsed.Params["time_zone"]; ok {
		return nil, fmt.Errorf("db: the DSN must not set time_zone (timestamps are Tehran wall-clock)")
	}
	parsed.InterpolateParams = true

	connector, err := mysql.NewConnector(parsed)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	// Every query is timed and counted (instrument.go, T-M9-07).
	pool := sql.OpenDB(Instrument(connector))
	pc := poolCfg.WithDefaults()
	pool.SetMaxOpenConns(pc.MaxOpen)
	pool.SetMaxIdleConns(pc.MaxIdle)
	pool.SetConnMaxLifetime(pc.ConnMaxLifetime)
	pool.SetConnMaxIdleTime(pc.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("db: ping %s@%s/%s: %w", parsed.User, parsed.Addr, parsed.DBName, err)
	}
	return pool, nil
}
