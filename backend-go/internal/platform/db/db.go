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

// Pool limits. The Laravel stack keeps one connection per PHP-FPM worker; Go shares one pool.
const (
	maxOpenConns    = 25
	maxIdleConns    = 10
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
	pingTimeout     = 5 * time.Second
)

// DSN builds the go-sql-driver DSN for cfg. loc is the location DATETIME values are
// interpreted in; callers pass the APP_TIMEZONE location (Asia/Tehran).
func DSN(cfg config.DB, loc *time.Location) string {
	c := mysql.NewConfig()
	c.User = cfg.Username
	c.Passwd = cfg.Password
	c.Net = "tcp"
	c.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	c.DBName = cfg.Database
	c.ParseTime = true
	c.Loc = loc
	c.Collation = "utf8mb4_unicode_ci"
	c.Params = map[string]string{"charset": "utf8mb4"}
	return c.FormatDSN()
}

// Open opens and pings the pool.
func Open(ctx context.Context, cfg config.DB, loc *time.Location) (*sql.DB, error) {
	return OpenDSN(ctx, DSN(cfg, loc))
}

// OpenDSN opens and pings a pool from a ready DSN (integration tests use TEST_DB_DSN).
// The DSN must not contain a time_zone parameter.
func OpenDSN(ctx context.Context, dsn string) (*sql.DB, error) {
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	if _, ok := parsed.Params["time_zone"]; ok {
		return nil, fmt.Errorf("db: the DSN must not set time_zone (timestamps are Tehran wall-clock)")
	}

	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	pool.SetMaxOpenConns(maxOpenConns)
	pool.SetMaxIdleConns(maxIdleConns)
	pool.SetConnMaxLifetime(connMaxLifetime)
	pool.SetConnMaxIdleTime(connMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("db: ping %s@%s/%s: %w", parsed.User, parsed.Addr, parsed.DBName, err)
	}
	return pool, nil
}
